package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"gofence/internal/apperr"
	"gofence/internal/geo"
	"gofence/internal/model"
	"gofence/internal/store"
	"gofence/internal/ws"
)

type Handler struct {
	db         *store.Store
	hub        *ws.Hub
	limiter    *Limiter
	trustProxy bool
}

func (h *Handler) writeErr(w http.ResponseWriter, r *http.Request, err error) {
	var ue *apperr.Error
	if errors.As(err, &ue) {
		writeJSON(w, r, ue.Status, map[string]string{"error": ue.Message})
		return
	}
	log.Printf("request error: %v", err)
	writeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

func (h *Handler) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.limiter.Allow(clientIP(r, h.trustProxy)) {
			writeJSON(w, r, http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded, try again shortly",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		writeJSON(w, r, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createGeofence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Coordinates [][]float64 `json:"coordinates"`
		Category    string      `json:"category"`
	}
	if err := decode(r, &req); err != nil {
		h.writeErr(w, r, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 120 {
		h.writeErr(w, r, apperr.New(400, "name is required and must be at most 120 characters"))
		return
	}
	if utf8.RuneCountInString(req.Description) > 2000 {
		h.writeErr(w, r, apperr.New(400, "description must be at most 2000 characters"))
		return
	}
	if !model.ValidCategory(req.Category) {
		h.writeErr(w, r, apperr.New(400, "category must be one of delivery_zone, restricted_zone, toll_zone, customer_area"))
		return
	}
	ring, err := geo.ValidateRing(req.Coordinates)
	if err != nil {
		h.writeErr(w, r, apperr.New(400, err.Error()))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	g, err := h.db.CreateGeofence(ctx, req.Name, req.Description, req.Category, ring)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{
		"id":     g.ID,
		"name":   g.Name,
		"status": "active",
	})
}

func (h *Handler) listGeofences(w http.ResponseWriter, r *http.Request) {
	category := queryPtr(r, "category")
	if category != nil && !model.ValidCategory(*category) {
		h.writeErr(w, r, apperr.New(400, "category must be one of delivery_zone, restricted_zone, toll_zone, customer_area"))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	items, err := h.db.ListGeofences(ctx, category)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, struct {
		Geofences []model.Geofence `json:"geofences"`
	}{Geofences: items})
}

func (h *Handler) createVehicle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VehicleNumber string `json:"vehicle_number"`
		DriverName    string `json:"driver_name"`
		VehicleType   string `json:"vehicle_type"`
		Phone         string `json:"phone"`
	}
	if err := decode(r, &req); err != nil {
		h.writeErr(w, r, err)
		return
	}
	req.VehicleNumber = strings.TrimSpace(req.VehicleNumber)
	req.DriverName = strings.TrimSpace(req.DriverName)
	req.VehicleType = strings.TrimSpace(req.VehicleType)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.VehicleNumber == "" || utf8.RuneCountInString(req.VehicleNumber) > 32 {
		h.writeErr(w, r, apperr.New(400, "vehicle_number is required and must be at most 32 characters"))
		return
	}
	if req.DriverName == "" || utf8.RuneCountInString(req.DriverName) > 80 {
		h.writeErr(w, r, apperr.New(400, "driver_name is required and must be at most 80 characters"))
		return
	}
	if req.VehicleType == "" || utf8.RuneCountInString(req.VehicleType) > 40 {
		h.writeErr(w, r, apperr.New(400, "vehicle_type is required and must be at most 40 characters"))
		return
	}
	if req.Phone == "" || utf8.RuneCountInString(req.Phone) > 32 {
		h.writeErr(w, r, apperr.New(400, "phone is required and must be at most 32 characters"))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	v, err := h.db.CreateVehicle(ctx, req.VehicleNumber, req.DriverName, req.VehicleType, req.Phone)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{
		"id":             v.ID,
		"vehicle_number": v.VehicleNumber,
		"status":         "active",
	})
}

func (h *Handler) listVehicles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := dbCtx(r)
	defer cancel()
	items, err := h.db.ListVehicles(ctx)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, struct {
		Vehicles []model.Vehicle `json:"vehicles"`
	}{Vehicles: items})
}

func (h *Handler) updateLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VehicleID string   `json:"vehicle_id"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Timestamp string   `json:"timestamp"`
	}
	if err := decode(r, &req); err != nil {
		h.writeErr(w, r, err)
		return
	}
	req.VehicleID = strings.TrimSpace(req.VehicleID)
	if req.VehicleID == "" {
		h.writeErr(w, r, apperr.New(400, "vehicle_id is required"))
		return
	}
	if req.Latitude == nil {
		h.writeErr(w, r, apperr.New(400, "latitude is required"))
		return
	}
	if req.Longitude == nil {
		h.writeErr(w, r, apperr.New(400, "longitude is required"))
		return
	}
	if err := geo.ValidateLatLng(*req.Latitude, *req.Longitude); err != nil {
		h.writeErr(w, r, apperr.New(400, err.Error()))
		return
	}
	recorded, err := time.Parse(time.RFC3339, strings.TrimSpace(req.Timestamp))
	if err != nil {
		h.writeErr(w, r, apperr.New(400, "timestamp must be ISO 8601, for example 2025-01-15T10:35:00Z"))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	res, err := h.db.UpdateLocation(ctx, req.VehicleID, *req.Latitude, *req.Longitude, recorded.UTC())
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if len(res.Alerts) > 0 {
		alerts := append([]model.AlertMessage(nil), res.Alerts...)
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("broadcast panic: %v", rec)
				}
			}()
			for _, alert := range alerts {
				h.hub.Broadcast(alert)
			}
		}()
	}
	writeJSON(w, r, http.StatusOK, res)
}

func (h *Handler) getLocation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "vehicle_id"))
	if id == "" || len(id) > 128 {
		h.writeErr(w, r, apperr.New(400, "vehicle_id is required"))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	res, err := h.db.GetVehicleLocation(ctx, id)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, res)
}

func (h *Handler) configureAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GeofenceID string  `json:"geofence_id"`
		VehicleID  *string `json:"vehicle_id"`
		EventType  string  `json:"event_type"`
	}
	if err := decode(r, &req); err != nil {
		h.writeErr(w, r, err)
		return
	}
	req.GeofenceID = strings.TrimSpace(req.GeofenceID)
	req.EventType = strings.TrimSpace(req.EventType)
	if req.GeofenceID == "" {
		h.writeErr(w, r, apperr.New(400, "geofence_id is required"))
		return
	}
	if !model.ValidEvent(req.EventType) {
		h.writeErr(w, r, apperr.New(400, "event_type must be one of entry, exit, both"))
		return
	}
	if req.VehicleID != nil {
		trimmed := strings.TrimSpace(*req.VehicleID)
		if trimmed == "" {
			req.VehicleID = nil
		} else {
			req.VehicleID = &trimmed
		}
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	rule, err := h.db.CreateAlert(ctx, req.GeofenceID, req.VehicleID, req.EventType)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{
		"alert_id":    rule.AlertID,
		"geofence_id": rule.GeofenceID,
		"vehicle_id":  rule.VehicleID,
		"event_type":  rule.EventType,
		"status":      rule.Status,
	})
}

func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := dbCtx(r)
	defer cancel()
	items, err := h.db.ListAlerts(ctx, queryPtr(r, "geofence_id"), queryPtr(r, "vehicle_id"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, struct {
		Alerts []model.AlertRule `json:"alerts"`
	}{Alerts: items})
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	offset, err := parseOffset(r.URL.Query().Get("offset"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	start, err := parseQueryTime(r.URL.Query().Get("start_date"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	end, err := parseQueryTime(r.URL.Query().Get("end_date"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if start != nil && end != nil && start.After(*end) {
		h.writeErr(w, r, apperr.New(400, "start_date must be before end_date"))
		return
	}
	ctx, cancel := dbCtx(r)
	defer cancel()
	page, err := h.db.ListViolations(ctx, store.HistoryFilter{
		VehicleID:  queryPtr(r, "vehicle_id"),
		GeofenceID: queryPtr(r, "geofence_id"),
		Start:      start,
		End:        end,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, page)
}

func dbCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 8*time.Second)
}

func queryPtr(r *http.Request, key string) *string {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return nil
	}
	return &v
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 50, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, apperr.New(400, "limit must be a positive integer")
	}
	if n > 500 {
		n = 500
	}
	return n, nil
}

func parseOffset(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 100000 {
		return 0, apperr.New(400, "offset must be between 0 and 100000")
	}
	return n, nil
}

func parseQueryTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, apperr.New(400, "dates must be ISO 8601, for example 2025-01-01T00:00:00Z")
	}
	u := t.UTC()
	return &u, nil
}
