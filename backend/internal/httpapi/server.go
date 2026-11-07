package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"gofence/internal/config"
	"gofence/internal/store"
	"gofence/internal/ws"
)

type Server struct {
	cfg     config.Config
	handler *Handler
	hub     *ws.Hub
}

func NewServer(cfg config.Config, db *store.Store) *Server {
	hub := ws.NewHub(cfg.APIKey)
	h := &Handler{
		db:         db,
		hub:        hub,
		limiter:    NewLimiter(cfg.RatePerMin, time.Minute),
		trustProxy: cfg.TrustProxy,
	}
	return &Server{cfg: cfg, handler: h, hub: hub}
}

func (s *Server) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(timing)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(apiKey(s.cfg.APIKey))
	router.Use(accessLog)
	router.Use(recoverer)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, r, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, r, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	})

	router.Get("/health", s.handler.health)
	router.Post("/geofences", s.handler.createGeofence)
	router.Get("/geofences", s.handler.listGeofences)
	router.Post("/vehicles", s.handler.createVehicle)
	router.Get("/vehicles", s.handler.listVehicles)
	router.With(s.handler.limit).Post("/vehicles/location", s.handler.updateLocation)
	router.Get("/vehicles/location/{vehicle_id}", s.handler.getLocation)
	router.Post("/alerts/configure", s.handler.configureAlert)
	router.Get("/alerts", s.handler.listAlerts)
	router.Get("/violations/history", s.handler.history)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws/alerts" {
			s.hub.ServeWS(w, r)
			return
		}
		router.ServeHTTP(w, r)
	})
}

func (s *Server) Run() error {
	httpSrv := &http.Server{
		Addr:              ":" + s.cfg.Port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on :%s", s.cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sig:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(ctx)
}
