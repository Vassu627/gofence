package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gofence/internal/apperr"
	"gofence/internal/geo"
	"gofence/internal/id"
	"gofence/internal/model"
)

type Store struct {
	pool *pgxpool.Pool
}

type HistoryFilter struct {
	VehicleID  *string
	GeofenceID *string
	Start      *time.Time
	End        *time.Time
	Limit      int
	Offset     int
}

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if strings.TrimSpace(url) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	var last error
	for attempt := 1; attempt <= 30; attempt++ {
		cfg, err := pgxpool.ParseConfig(url)
		if err != nil {
			return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
		}
		cfg.MaxConns = 10
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("connect database: %w", last)
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) CreateGeofence(ctx context.Context, name, description, category string, ring [][]float64) (model.Geofence, error) {
	gid, err := id.New("geo_")
	if err != nil {
		return model.Geofence{}, err
	}
	raw, err := json.Marshal(ring)
	if err != nil {
		return model.Geofence{}, err
	}
	wkt := geo.WKT(ring)
	var valid bool
	if err := s.pool.QueryRow(ctx, `SELECT ST_IsValid(ST_SetSRID(ST_GeomFromText($1), 4326))`, wkt).Scan(&valid); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return model.Geofence{}, apperr.New(400, "invalid polygon geometry")
		}
		return model.Geofence{}, err
	}
	if !valid {
		return model.Geofence{}, apperr.New(400, "polygon is self-intersecting or otherwise invalid")
	}
	var created time.Time
	err = s.pool.QueryRow(ctx, `
		INSERT INTO geofences (id, name, description, category, coordinates, geom, status)
		VALUES ($1, $2, $3, $4, $5::jsonb, ST_SetSRID(ST_GeomFromText($6), 4326), 'active')
		RETURNING created_at
	`, gid, name, description, category, string(raw), wkt).Scan(&created)
	if err != nil {
		return model.Geofence{}, err
	}
	return model.Geofence{
		ID:          gid,
		Name:        name,
		Description: description,
		Coordinates: ring,
		Category:    category,
		CreatedAt:   fmtTime(created),
	}, nil
}

func (s *Store) ListGeofences(ctx context.Context, category *string) ([]model.Geofence, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, coordinates, category, created_at
		FROM geofences
		WHERE ($1::text IS NULL OR category = $1)
		ORDER BY created_at DESC, id DESC
	`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Geofence{}
	for rows.Next() {
		var g model.Geofence
		var raw string
		var created time.Time
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &raw, &g.Category, &created); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &g.Coordinates); err != nil {
			return nil, fmt.Errorf("decode coordinates for %s: %w", g.ID, err)
		}
		g.CreatedAt = fmtTime(created)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) CreateVehicle(ctx context.Context, number, driver, vehicleType, phone string) (model.Vehicle, error) {
	vid, err := id.New("veh_")
	if err != nil {
		return model.Vehicle{}, err
	}
	var created time.Time
	err = s.pool.QueryRow(ctx, `
		INSERT INTO vehicles (id, vehicle_number, driver_name, vehicle_type, phone, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		RETURNING created_at
	`, vid, number, driver, vehicleType, phone).Scan(&created)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Vehicle{}, apperr.New(409, "vehicle number already registered")
		}
		return model.Vehicle{}, err
	}
	return model.Vehicle{
		ID:            vid,
		VehicleNumber: number,
		DriverName:    driver,
		VehicleType:   vehicleType,
		Phone:         phone,
		Status:        "active",
		CreatedAt:     fmtTime(created),
	}, nil
}

func (s *Store) ListVehicles(ctx context.Context) ([]model.Vehicle, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, vehicle_number, driver_name, vehicle_type, phone, status, created_at
		FROM vehicles
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Vehicle{}
	for rows.Next() {
		var v model.Vehicle
		var created time.Time
		if err := rows.Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &created); err != nil {
			return nil, err
		}
		v.CreatedAt = fmtTime(created)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) CreateAlert(ctx context.Context, geofenceID string, vehicleID *string, eventType string) (model.AlertRule, error) {
	var geofenceName string
	err := s.pool.QueryRow(ctx, `SELECT name FROM geofences WHERE id = $1`, geofenceID).Scan(&geofenceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.AlertRule{}, apperr.New(404, "geofence not found")
	}
	if err != nil {
		return model.AlertRule{}, err
	}
	var vehicleNumber *string
	if vehicleID != nil {
		var number string
		err = s.pool.QueryRow(ctx, `SELECT vehicle_number FROM vehicles WHERE id = $1`, *vehicleID).Scan(&number)
		if errors.Is(err, pgx.ErrNoRows) {
			return model.AlertRule{}, apperr.New(404, "vehicle not found")
		}
		if err != nil {
			return model.AlertRule{}, err
		}
		vehicleNumber = &number
	}
	aid, err := id.New("alert_")
	if err != nil {
		return model.AlertRule{}, err
	}
	var created time.Time
	err = s.pool.QueryRow(ctx, `
		INSERT INTO alert_rules (id, geofence_id, vehicle_id, event_type, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING created_at
	`, aid, geofenceID, vehicleID, eventType).Scan(&created)
	if err != nil {
		return model.AlertRule{}, err
	}
	return model.AlertRule{
		AlertID:       aid,
		GeofenceID:    geofenceID,
		GeofenceName:  geofenceName,
		VehicleID:     vehicleID,
		VehicleNumber: vehicleNumber,
		EventType:     eventType,
		Status:        "active",
		CreatedAt:     fmtTime(created),
	}, nil
}

func (s *Store) ListAlerts(ctx context.Context, geofenceID, vehicleID *string) ([]model.AlertRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.geofence_id, g.name, a.vehicle_id, v.vehicle_number,
		       a.event_type, a.status, a.created_at
		FROM alert_rules a
		JOIN geofences g ON g.id = a.geofence_id
		LEFT JOIN vehicles v ON v.id = a.vehicle_id
		WHERE ($1::text IS NULL OR a.geofence_id = $1)
		  AND ($2::text IS NULL OR a.vehicle_id = $2)
		ORDER BY a.created_at DESC, a.id DESC
	`, geofenceID, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AlertRule{}
	for rows.Next() {
		var rule model.AlertRule
		var created time.Time
		if err := rows.Scan(
			&rule.AlertID, &rule.GeofenceID, &rule.GeofenceName,
			&rule.VehicleID, &rule.VehicleNumber, &rule.EventType, &rule.Status, &created,
		); err != nil {
			return nil, err
		}
		rule.CreatedAt = fmtTime(created)
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (s *Store) ListViolations(ctx context.Context, f HistoryFilter) (model.ViolationPage, error) {
	where := []string{"TRUE"}
	args := []any{}
	if f.VehicleID != nil {
		args = append(args, *f.VehicleID)
		where = append(where, fmt.Sprintf("v.vehicle_id = $%d", len(args)))
	}
	if f.GeofenceID != nil {
		args = append(args, *f.GeofenceID)
		where = append(where, fmt.Sprintf("v.geofence_id = $%d", len(args)))
	}
	if f.Start != nil {
		args = append(args, *f.Start)
		where = append(where, fmt.Sprintf("v.recorded_at >= $%d", len(args)))
	}
	if f.End != nil {
		args = append(args, *f.End)
		where = append(where, fmt.Sprintf("v.recorded_at <= $%d", len(args)))
	}
	clause := strings.Join(where, " AND ")
	var total int
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM violations v WHERE %s`, clause)
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return model.ViolationPage{}, err
	}
	listArgs := append(append([]any{}, args...), f.Limit, f.Offset)
	listSQL := fmt.Sprintf(`
		SELECT v.id, v.vehicle_id, ve.vehicle_number, v.geofence_id, g.name,
		       v.event_type, v.latitude, v.longitude, v.recorded_at
		FROM violations v
		JOIN vehicles ve ON ve.id = v.vehicle_id
		JOIN geofences g ON g.id = v.geofence_id
		WHERE %s
		ORDER BY v.recorded_at DESC, v.id DESC
		LIMIT $%d OFFSET $%d
	`, clause, len(listArgs)-1, len(listArgs))
	rows, err := s.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return model.ViolationPage{}, err
	}
	defer rows.Close()
	page := model.ViolationPage{Violations: []model.Violation{}, TotalCount: total}
	for rows.Next() {
		var item model.Violation
		var recorded time.Time
		if err := rows.Scan(
			&item.ID, &item.VehicleID, &item.VehicleNumber, &item.GeofenceID, &item.GeofenceName,
			&item.EventType, &item.Latitude, &item.Longitude, &recorded,
		); err != nil {
			return model.ViolationPage{}, err
		}
		item.Timestamp = fmtTime(recorded)
		page.Violations = append(page.Violations, item)
	}
	return page, rows.Err()
}

func fmtTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
