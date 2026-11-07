package store

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"gofence/internal/apperr"
	"gofence/internal/geo"
	"gofence/internal/id"
	"gofence/internal/model"
	"gofence/internal/rules"
)

type fenceInfo struct {
	ID       string
	Name     string
	Category string
}

type transition struct {
	fence     fenceInfo
	eventType string
}

// UpdateLocation stores a fix, compares it with the vehicle's previous
// geofence membership, records entry and exit events, and returns alerts for
// rules that match. The caller broadcasts those alerts after this returns.
func (s *Store) UpdateLocation(ctx context.Context, vehicleID string, lat, lng float64, recordedAt time.Time) (model.LocationUpdate, error) {
	if err := geo.ValidateLatLng(lat, lng); err != nil {
		return model.LocationUpdate{}, apperr.New(400, err.Error())
	}
	recordedAt = recordedAt.UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.LocationUpdate{}, err
	}
	defer tx.Rollback(ctx)

	var number, driver string
	err = tx.QueryRow(ctx, `
		SELECT vehicle_number, driver_name FROM vehicles WHERE id = $1 FOR UPDATE
	`, vehicleID).Scan(&number, &driver)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.LocationUpdate{}, apperr.New(404, "vehicle not found")
	}
	if err != nil {
		return model.LocationUpdate{}, err
	}

	var last *time.Time
	if err := tx.QueryRow(ctx, `SELECT MAX(recorded_at) FROM locations WHERE vehicle_id = $1`, vehicleID).Scan(&last); err != nil {
		return model.LocationUpdate{}, err
	}
	if last != nil && !recordedAt.After(*last) {
		return model.LocationUpdate{}, apperr.New(409, "timestamp must be later than the vehicle's last recorded location")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO locations (vehicle_id, latitude, longitude, geom, recorded_at)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6)
	`, vehicleID, lat, lng, lng, lat, recordedAt); err != nil {
		return model.LocationUpdate{}, err
	}

	current, err := loadContaining(ctx, tx, lng, lat)
	if err != nil {
		return model.LocationUpdate{}, err
	}
	previous, err := loadMembership(ctx, tx, vehicleID)
	if err != nil {
		return model.LocationUpdate{}, err
	}
	changes := membershipChanges(previous, current)

	var configured []rules.Rule
	if len(changes) > 0 {
		ids := make([]string, len(changes))
		for i, change := range changes {
			ids[i] = change.fence.ID
		}
		configured, err = loadRules(ctx, tx, ids)
		if err != nil {
			return model.LocationUpdate{}, err
		}
	}

	alerts := []model.AlertMessage{}
	for _, change := range changes {
		violationID, err := id.New("viol_")
		if err != nil {
			return model.LocationUpdate{}, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO violations (id, vehicle_id, geofence_id, event_type, latitude, longitude, recorded_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, violationID, vehicleID, change.fence.ID, change.eventType, lat, lng, recordedAt); err != nil {
			return model.LocationUpdate{}, err
		}

		var matched *rules.Rule
		for _, rule := range configured {
			if rule.GeofenceID != change.fence.ID {
				continue
			}
			if rules.Matches(rule, vehicleID, change.eventType) {
				copyRule := rule
				matched = &copyRule
				break
			}
		}
		if matched == nil {
			continue
		}
		eventID, err := id.New("evt_")
		if err != nil {
			return model.LocationUpdate{}, err
		}
		message := model.AlertMessage{
			EventID:   eventID,
			EventType: change.eventType,
			Timestamp: fmtTime(recordedAt),
			Vehicle: model.AlertVehicle{
				VehicleID:     vehicleID,
				VehicleNumber: number,
				DriverName:    driver,
			},
			Geofence: model.AlertGeofence{
				GeofenceID:   change.fence.ID,
				GeofenceName: change.fence.Name,
				Category:     change.fence.Category,
			},
			Location: model.AlertLocation{Latitude: lat, Longitude: lng},
		}
		payload, err := json.Marshal(message)
		if err != nil {
			return model.LocationUpdate{}, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO alert_events (id, violation_id, rule_id, vehicle_id, geofence_id, event_type, payload)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		`, eventID, violationID, matched.ID, vehicleID, change.fence.ID, change.eventType, string(payload)); err != nil {
			return model.LocationUpdate{}, err
		}
		alerts = append(alerts, message)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM vehicle_geofence_state WHERE vehicle_id = $1`, vehicleID); err != nil {
		return model.LocationUpdate{}, err
	}
	for _, fence := range current {
		if _, err := tx.Exec(ctx, `
			INSERT INTO vehicle_geofence_state (vehicle_id, geofence_id, updated_at)
			VALUES ($1, $2, $3)
		`, vehicleID, fence.ID, recordedAt); err != nil {
			return model.LocationUpdate{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return model.LocationUpdate{}, err
	}
	if len(alerts) > 0 {
		log.Printf("queued %d alert(s) for vehicle %s", len(alerts), vehicleID)
	}

	inside := make([]model.InsideGeofence, 0, len(current))
	for _, fence := range current {
		inside = append(inside, model.InsideGeofence{
			GeofenceID:   fence.ID,
			GeofenceName: fence.Name,
			Status:       "inside",
		})
	}
	return model.LocationUpdate{
		VehicleID:        vehicleID,
		LocationUpdated:  true,
		CurrentGeofences: inside,
		Alerts:           alerts,
	}, nil
}

func (s *Store) GetVehicleLocation(ctx context.Context, vehicleID string) (model.VehicleLocation, error) {
	var res model.VehicleLocation
	err := s.pool.QueryRow(ctx, `SELECT id, vehicle_number FROM vehicles WHERE id = $1`, vehicleID).
		Scan(&res.VehicleID, &res.VehicleNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.VehicleLocation{}, apperr.New(404, "vehicle not found")
	}
	if err != nil {
		return model.VehicleLocation{}, err
	}
	res.CurrentGeofences = []model.OccupiedGeofence{}

	var lat, lng float64
	var recorded time.Time
	err = s.pool.QueryRow(ctx, `
		SELECT latitude, longitude, recorded_at
		FROM locations
		WHERE vehicle_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`, vehicleID).Scan(&lat, &lng, &recorded)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, nil
	}
	if err != nil {
		return model.VehicleLocation{}, err
	}
	res.CurrentLocation = &model.Fix{Latitude: lat, Longitude: lng, Timestamp: fmtTime(recorded)}

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, category
		FROM geofences
		WHERE ST_Covers(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY name, id
	`, lng, lat)
	if err != nil {
		return model.VehicleLocation{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var fence model.OccupiedGeofence
		if err := rows.Scan(&fence.GeofenceID, &fence.GeofenceName, &fence.Category); err != nil {
			return model.VehicleLocation{}, err
		}
		res.CurrentGeofences = append(res.CurrentGeofences, fence)
	}
	return res, rows.Err()
}

func loadContaining(ctx context.Context, tx pgx.Tx, lng, lat float64) ([]fenceInfo, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, name, category
		FROM geofences
		WHERE ST_Covers(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY name, id
	`, lng, lat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFences(rows)
}

func loadMembership(ctx context.Context, tx pgx.Tx, vehicleID string) ([]fenceInfo, error) {
	rows, err := tx.Query(ctx, `
		SELECT g.id, g.name, g.category
		FROM vehicle_geofence_state s
		JOIN geofences g ON g.id = s.geofence_id
		WHERE s.vehicle_id = $1
		ORDER BY g.name, g.id
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFences(rows)
}

func scanFences(rows pgx.Rows) ([]fenceInfo, error) {
	out := []fenceInfo{}
	for rows.Next() {
		var fence fenceInfo
		if err := rows.Scan(&fence.ID, &fence.Name, &fence.Category); err != nil {
			return nil, err
		}
		out = append(out, fence)
	}
	return out, rows.Err()
}

func membershipChanges(previous, current []fenceInfo) []transition {
	prevByID := make(map[string]fenceInfo, len(previous))
	prevIDs := make([]string, len(previous))
	for i, fence := range previous {
		prevByID[fence.ID] = fence
		prevIDs[i] = fence.ID
	}
	currByID := make(map[string]fenceInfo, len(current))
	currIDs := make([]string, len(current))
	for i, fence := range current {
		currByID[fence.ID] = fence
		currIDs[i] = fence.ID
	}
	entered, exited := rules.Diff(prevIDs, currIDs)
	changes := make([]transition, 0, len(entered)+len(exited))
	for _, id := range entered {
		changes = append(changes, transition{fence: currByID[id], eventType: "entry"})
	}
	for _, id := range exited {
		changes = append(changes, transition{fence: prevByID[id], eventType: "exit"})
	}
	return changes
}

func loadRules(ctx context.Context, tx pgx.Tx, geofenceIDs []string) ([]rules.Rule, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, geofence_id, vehicle_id, event_type
		FROM alert_rules
		WHERE status = 'active' AND geofence_id = ANY($1::text[])
	`, geofenceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rules.Rule{}
	for rows.Next() {
		var rule rules.Rule
		if err := rows.Scan(&rule.ID, &rule.GeofenceID, &rule.VehicleID, &rule.EventType); err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}
