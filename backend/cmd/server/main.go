package main

import (
	"context"
	"log"

	"gofence/internal/config"
	"gofence/internal/httpapi"
	"gofence/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lmsgprefix)
	log.SetPrefix("api ")

	cfg := config.Load()
	ctx := context.Background()
	pool, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := store.New(pool)
	if err := db.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("database ready")

	if err := httpapi.NewServer(cfg, db).Run(); err != nil {
		log.Fatal(err)
	}
}
