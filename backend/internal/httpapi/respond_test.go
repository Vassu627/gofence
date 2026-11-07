package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gofence/internal/apperr"
)

func TestWriteJSONIncludesTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/geofences", nil)
	req = req.WithContext(context.WithValue(req.Context(), startKey, time.Now().Add(-2*time.Millisecond)))
	rr := httptest.NewRecorder()
	writeJSON(rr, req, http.StatusCreated, map[string]any{"id": "geo_1"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ns, ok := body["time_ns"].(string)
	if !ok {
		t.Fatalf("time_ns %#v", body["time_ns"])
	}
	if _, err := strconv.ParseInt(ns, 10, 64); err != nil || ns == "0" {
		t.Fatal(ns, err)
	}
	if body["id"] != "geo_1" {
		t.Fatal(body)
	}
}

func TestDecodeRequiresBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	var dst map[string]any
	err := decode(req, &dst)
	var ue *apperr.Error
	if !errors.As(err, &ue) || ue.Status != http.StatusBadRequest {
		t.Fatal(err)
	}
}
