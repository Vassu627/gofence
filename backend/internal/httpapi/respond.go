package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gofence/internal/apperr"
)

type ctxKey int

const startKey ctxKey = 1

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	body := map[string]any{}
	if err := json.Unmarshal(payload, &body); err != nil || body == nil {
		body = map[string]any{}
	}
	body["time_ns"] = elapsed(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body)
}

func elapsed(r *http.Request) string {
	start, ok := r.Context().Value(startKey).(time.Time)
	if !ok || start.IsZero() {
		return "0"
	}
	d := time.Since(start).Nanoseconds()
	if d < 0 {
		d = 0
	}
	return strconv.FormatInt(d, 10)
}

func decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return apperr.New(http.StatusBadRequest, "request body is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil {
		return apperr.New(http.StatusBadRequest, "invalid request body")
	}
	if len(body) > 1<<20 {
		return apperr.New(http.StatusBadRequest, "request body is too large")
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return apperr.New(http.StatusBadRequest, "request body is required")
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return apperr.New(http.StatusBadRequest, "invalid JSON body")
	}
	return nil
}
