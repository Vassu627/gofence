# Geofence Control

Full-stack geofencing and vehicle tracking app for the MapUp assessment. Operators draw zones, register vehicles, post location fixes, and receive live entry and exit alerts.

The Go API stores polygons in PostGIS and answers “is this vehicle inside?” with `ST_Covers`. A WebSocket at `/ws/alerts` pushes a message when a stored alert rule matches. The React app shows the same yard on a Leaflet map.

## Run it

Docker Desktop must be running.

```bash
docker compose up --build
```

- API: http://localhost:8080
- Web UI: http://localhost:4173

Open the UI, choose **Replay sample route**, and watch the live feed. That creates a downtown delivery zone, a restricted yard, a truck, and drives the truck across both boundaries.

Local setup, curl examples, and deployment steps are in [SETUP.md](SETUP.md).

## Layout

- `backend/` — Go API, PostGIS schema, WebSocket hub
- `frontend/` — React, Vite, Leaflet
- `scripts/smoke.mjs` — end-to-end check of the HTTP contract
- `docker-compose.yml` — Postgres/PostGIS, API, and web UI

## Submission checklist

These steps need your accounts. Do not make the repository public.

- [ ] Push this project to a **private** GitHub repository
- [ ] Add these collaborators: `vedantp@mapup.ai`, `ajayap@mapup.ai`, `jyotis@mapup.ai`, `tanishqs@mapup.ai`, `naveenn@mapup.ai`, `asijitp@mapup.ai`, `dewanshp@mapup.ai`, `karkuvelpandip@mapup.ai`, `parask@mapup.ai`
- [ ] Build and push the backend image to Docker Hub
- [ ] Deploy the API container and the frontend (Vercel, Netlify, or Cloudflare Pages)
- [ ] Record the 30–45 minute walkthrough and put the unlisted link below
- [ ] Submit the Google Form from the assessment email

Demo video: _add the unlisted link here before submitting_
