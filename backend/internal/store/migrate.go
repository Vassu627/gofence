package store

import (
	"context"
	"fmt"
)

const migrateSQL = `
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS geofences (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL CHECK (category IN ('delivery_zone', 'restricted_zone', 'toll_zone', 'customer_area')),
    coordinates JSONB NOT NULL,
    geom geometry(Polygon, 4326) NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vehicles (
    id TEXT PRIMARY KEY,
    vehicle_number TEXT NOT NULL UNIQUE,
    driver_name TEXT NOT NULL,
    vehicle_type TEXT NOT NULL,
    phone TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS locations (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    geom geometry(Point, 4326) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vehicle_geofence_state (
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    geofence_id TEXT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (vehicle_id, geofence_id)
);

CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    geofence_id TEXT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    vehicle_id TEXT REFERENCES vehicles(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('entry', 'exit', 'both')),
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS violations (
    id TEXT PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    geofence_id TEXT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('entry', 'exit')),
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS alert_events (
    id TEXT PRIMARY KEY,
    violation_id TEXT NOT NULL REFERENCES violations(id) ON DELETE CASCADE,
    rule_id TEXT REFERENCES alert_rules(id) ON DELETE SET NULL,
    vehicle_id TEXT NOT NULL,
    geofence_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('entry', 'exit')),
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS geofences_geom_gix ON geofences USING GIST (geom);
CREATE INDEX IF NOT EXISTS geofences_category_idx ON geofences (category);
CREATE INDEX IF NOT EXISTS locations_vehicle_recorded_idx ON locations (vehicle_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS locations_geom_gix ON locations USING GIST (geom);
CREATE INDEX IF NOT EXISTS violations_vehicle_idx ON violations (vehicle_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS violations_geofence_idx ON violations (geofence_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS violations_recorded_idx ON violations (recorded_at DESC);
CREATE INDEX IF NOT EXISTS alert_rules_geofence_idx ON alert_rules (geofence_id);
CREATE INDEX IF NOT EXISTS alert_rules_vehicle_idx ON alert_rules (vehicle_id);
`

func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	// Simple query protocol, so the script can contain more than one statement.
	if err := conn.Conn().PgConn().Exec(ctx, migrateSQL).Close(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
