# Setup

## Prerequisites

- Docker Desktop with Compose v2
- For a local (non-Docker) backend: Go 1.23+ and Node.js 20+
- A browser with internet access so the map tiles can load

The database password in `docker-compose.yml` is for local use only. Change it before any shared deployment.

## Run with Docker Compose

From the repository root:

```bash
docker compose up --build
```

| Service | URL |
| --- | --- |
| API | http://localhost:8080 |
| Web UI | http://localhost:4173 |
| Postgres | localhost:5433 (`gofence` / `gofence`, database `gofence`). Inside Compose the API uses host `db` port `5432`. |

The web image is built with `VITE_API_URL=http://localhost:8080` because the browser, not the container, calls the API. If you publish the API somewhere else, rebuild the frontend with that public URL:

```bash
docker compose build web --build-arg VITE_API_URL=https://your-api.example.com
```

Stop and remove the local database volume:

```bash
docker compose down -v
```

## Run the processes yourself

Start only the database:

```bash
docker compose up -d db
```

Backend (PowerShell):

```powershell
cd backend
$env:DATABASE_URL = "postgres://gofence:gofence@localhost:5433/gofence?sslmode=disable"
go run ./cmd/server
```

Backend (bash):

```bash
cd backend
DATABASE_URL="postgres://gofence:gofence@localhost:5433/gofence?sslmode=disable" go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

The dev server is http://localhost:5173 and talks to http://localhost:8080. Copy `frontend/.env.example` to `frontend/.env` to override that.

Optional backend settings, from `backend/.env.example`:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | API port |
| `DATABASE_URL` | required | Postgres connection string |
| `API_KEY` | empty | When set, clients must send `X-API-Key`. WebSocket clients may pass `?api_key=` |
| `CORS_ORIGINS` | `*` | Comma-separated browser origins |
| `RATE_LIMIT_PER_MIN` | `120` | Limit for `POST /vehicles/location`, per client IP |
| `TRUST_PROXY` | `false` | Use the first `X-Forwarded-For` hop as the rate-limit key |

`GET /health` stays open so the container health check works when `API_KEY` is set.

## Tests

```bash
cd backend
go test ./...
```

Point-in-polygon, alert matching, validation, and the rate limiter run without a database. Set `DATABASE_URL` to a migrated PostGIS database to also run the location entry/exit test.

With the API already running:

```bash
node scripts/smoke.mjs
```

## API contract

Every JSON response includes `time_ns`, the handler duration in nanoseconds, as a string.

Identifiers look like `geo_`, `veh_`, `alert_`, `viol_`, and `evt_` plus 10 hex characters.

Coordinates are `[latitude, longitude]`. A polygon must be closed, have at least 4 points (3 corners plus the repeated first point), and stay inside the WGS84 ranges.

Categories: `delivery_zone`, `restricted_zone`, `toll_zone`, `customer_area`.

Alert events: `entry`, `exit`, `both`. Omit `vehicle_id` to watch every vehicle.

### Create a geofence

```bash
curl -s -X POST http://localhost:8080/geofences \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Downtown Delivery Zone\",\"description\":\"Main delivery area\",\"coordinates\":[[37.7749,-122.4194],[37.7849,-122.4194],[37.7849,-122.4094],[37.7749,-122.4094],[37.7749,-122.4194]],\"category\":\"delivery_zone\"}"
```

### List geofences

```bash
curl -s "http://localhost:8080/geofences?category=delivery_zone"
```

### Register a vehicle

```bash
curl -s -X POST http://localhost:8080/vehicles \
  -H "Content-Type: application/json" \
  -d "{\"vehicle_number\":\"KA-01-AB-1234\",\"driver_name\":\"John Doe\",\"vehicle_type\":\"truck\",\"phone\":\"+1234567890\"}"
```

### List vehicles

```bash
curl -s http://localhost:8080/vehicles
```

### Post a location

Use the ids returned above. The second call moves the vehicle inside the sample zone and, once an alert rule exists, produces an entry.

```bash
curl -s -X POST http://localhost:8080/vehicles/location \
  -H "Content-Type: application/json" \
  -d "{\"vehicle_id\":\"veh_123\",\"latitude\":37.77,\"longitude\":-122.4144,\"timestamp\":\"2026-11-07T12:00:00Z\"}"

curl -s -X POST http://localhost:8080/vehicles/location \
  -H "Content-Type: application/json" \
  -d "{\"vehicle_id\":\"veh_123\",\"latitude\":37.778,\"longitude\":-122.418,\"timestamp\":\"2026-11-07T12:00:02Z\"}"
```

A fix is rejected with HTTP 409 when its timestamp is not later than that vehicle’s previous fix. The same vehicle’s updates are serialized with a row lock so two overlapping posts cannot both miss an entry.

### Current location

```bash
curl -s http://localhost:8080/vehicles/location/veh_123
```

A vehicle that has never reported a fix returns `"current_location": null`.

### Configure an alert

Leave out `vehicle_id` to cover every vehicle.

```bash
curl -s -X POST http://localhost:8080/alerts/configure \
  -H "Content-Type: application/json" \
  -d "{\"geofence_id\":\"geo_123\",\"vehicle_id\":\"veh_123\",\"event_type\":\"entry\"}"
```

### List alert rules

```bash
curl -s "http://localhost:8080/alerts?geofence_id=geo_123&vehicle_id=veh_123"
```

`vehicle_id` on a rule filters to that exact rule. A rule with no vehicle is stored as `null` and means all vehicles.

### Crossing history

```bash
curl -s "http://localhost:8080/violations/history?vehicle_id=veh_123&limit=100"
```

Optional filters: `geofence_id`, `start_date`, `end_date` (ISO 8601), `limit` (default 50, values above 500 are clamped), and `offset`.

### WebSocket

Connect to `ws://localhost:8080/ws/alerts`. When a location update enters or exits a zone and a rule matches, connected browsers receive:

```json
{
  "event_id": "evt_999",
  "event_type": "entry",
  "timestamp": "2026-11-07T12:00:02Z",
  "vehicle": {
    "vehicle_id": "veh_123",
    "vehicle_number": "KA-01-AB-1234",
    "driver_name": "John Doe"
  },
  "geofence": {
    "geofence_id": "geo_123",
    "geofence_name": "Downtown Delivery Zone",
    "category": "delivery_zone"
  },
  "location": { "latitude": 37.778, "longitude": -122.418 }
}
```

The HTTP handler stores the crossing and the alert, commits, then broadcasts on another goroutine. A missing rule still records the crossing in history; it does not push a live alert. Restricted zones do not alert by themselves. The UI offers a checkbox, on by default for restricted zones, that creates an all-vehicle entry and exit rule.

In the browser console:

```js
const ws = new WebSocket("ws://localhost:8080/ws/alerts")
ws.onmessage = (event) => console.log(JSON.parse(event.data))
```

## Using the web app

1. **Live map → Draw.** Click corners, close the shape, name it, and save. Restricted zones should keep the alert checkbox on.
2. **Vehicles.** Register a number, driver, type, and phone. Use **Place on map** or type coordinates and send a fix.
3. **Alert rules.** Pick a geofence, optionally one vehicle, and entry, exit, or both.
4. **Live map → Replay sample route.** This is the fastest way to see an alert arrive.
5. **Crossings.** Filter history by vehicle, geofence, and time, then page through the results.

Toasts stay up for about eight seconds. The feed on the map keeps the latest 50 messages for the browser session. The socket reconnects on its own.

## Architecture

```
Browser (React + Leaflet)
    |  REST JSON, every body includes time_ns
    |  WebSocket /ws/alerts
Go API (chi)
    |  SQL + ST_Covers on a GIST index
PostgreSQL + PostGIS
```

Location updates do this inside one transaction:

1. Lock the vehicle row.
2. Reject a timestamp that is not newer than the last fix.
3. Insert the point.
4. Ask PostGIS which polygons cover it (`ST_Covers`, so the boundary counts as inside).
5. Diff that set with the membership stored after the previous fix. New ids are entries. Missing ids are exits.
6. Insert every crossing into `violations`.
7. If an active rule matches the geofence, the vehicle (or all vehicles), and the event type, insert `alert_events` and return the WebSocket payload.
8. Replace `vehicle_geofence_state` for that vehicle.
9. Commit, then broadcast.

`internal/geo/polygon.go` has the even-odd point-in-polygon used by the unit tests. The request path uses PostGIS so the GIST index does the search. Both treat coordinates as `[latitude, longitude]`; the SQL converts to PostGIS’s longitude-first order.

## Deploy

Backend image:

```bash
docker build -t YOUR_DOCKERHUB_USER/geofence-api:1.0.0 ./backend
docker push YOUR_DOCKERHUB_USER/geofence-api:1.0.0
```

Run that image with `DATABASE_URL` pointing at a Postgres instance that has PostGIS. Render, Fly.io, Railway, or a VPS all work. Set `CORS_ORIGINS` to the frontend origin and `TRUST_PROXY=true` when the platform puts a proxy in front.

Frontend on Vercel, Netlify, or Cloudflare Pages:

- Root directory: `frontend`
- Build command: `npm run build`
- Output directory: `dist`
- Environment variable: `VITE_API_URL=https://your-api.example.com`

The API URL is baked in at build time. Rebuild after changing it. The API must be reachable from the public internet on HTTPS if the site itself is HTTPS, and the WebSocket URL is derived by switching `http` to `ws` (or `https` to `wss`).

## Limits worth mentioning in the walkthrough

- One API process keeps WebSocket clients in memory. More than one instance needs a shared bus (Redis, NATS) or sticky sessions plus a fan-out service.
- The map loads each vehicle location with its own request. A batch read would matter once there are hundreds of vehicles.
- `locations` and `violations` grow without a retention policy. Partition them by month before the tables get large.
- There is no delete or edit API for zones, vehicles, or rules.
- Out-of-order fixes are rejected instead of replayed into history.
- Polygons are a single outer ring. Holes and multipolygons are not accepted.
- Authentication is an optional shared `API_KEY`, not per-user accounts.
- The rate limit is in-memory, so it resets on restart and is not shared across instances.
