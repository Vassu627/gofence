package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"gofence/internal/apperr"
)

func TestUpdateLocationDetectsEntryAndExit(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	st := New(pool)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	suffix := time.Now().UnixNano()
	ring := [][]float64{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	fence, err := st.CreateGeofence(ctx, fmt.Sprintf("test-fence-%d", suffix), "unit", "delivery_zone", ring)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM geofences WHERE id = $1`, fence.ID)
	})
	vehicle, err := st.CreateVehicle(ctx, fmt.Sprintf("T-%d", suffix), "Test Driver", "van", "+10000000000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM vehicles WHERE id = $1`, vehicle.ID)
	})

	base := time.Now().UTC().Add(-time.Minute).Truncate(time.Millisecond)
	outside, err := st.UpdateLocation(ctx, vehicle.ID, 2, 2, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(outside.CurrentGeofences) != 0 || len(outside.Alerts) != 0 {
		t.Fatalf("outside = %+v", outside)
	}

	inside, err := st.UpdateLocation(ctx, vehicle.ID, 0.5, 0.5, base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(inside.CurrentGeofences) != 1 || len(inside.Alerts) != 0 {
		t.Fatalf("first inside = %+v", inside)
	}
	_, err = st.UpdateLocation(ctx, vehicle.ID, 0.5, 0.5, base.Add(time.Second))
	var status *apperr.Error
	if !errors.As(err, &status) || status.Status != 409 {
		t.Fatalf("stale timestamp err = %v", err)
	}
	again, err := st.UpdateLocation(ctx, vehicle.ID, 0.55, 0.55, base.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Alerts) != 0 {
		t.Fatal("repeat inside should not alert or create a new crossing")
	}
	if _, err := st.UpdateLocation(ctx, vehicle.ID, 3, 3, base.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}

	if _, err := st.CreateAlert(ctx, fence.ID, nil, "entry"); err != nil {
		t.Fatal(err)
	}
	alerted, err := st.UpdateLocation(ctx, vehicle.ID, 0.4, 0.4, base.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerted.Alerts) != 1 || alerted.Alerts[0].EventType != "entry" {
		t.Fatalf("alerted = %+v", alerted.Alerts)
	}
	if alerted.Alerts[0].Vehicle.VehicleNumber != vehicle.VehicleNumber {
		t.Fatalf("vehicle payload = %+v", alerted.Alerts[0].Vehicle)
	}

	page, err := st.ListViolations(ctx, HistoryFilter{VehicleID: &vehicle.ID, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalCount != 3 {
		t.Fatalf("total = %d, rows = %+v", page.TotalCount, page.Violations)
	}
	if page.Violations[0].EventType != "entry" {
		t.Fatalf("latest = %+v", page.Violations[0])
	}
}
