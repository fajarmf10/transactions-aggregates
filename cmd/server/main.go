package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/api"
	"github.com/fajarmf10/transactions-aggregates/internal/store"
)

func main() {
	cfg := loadConfig()
	if cfg.databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	db, err := connectWithRetry(ctx, cfg.databaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("apply schema: %v", err)
	}

	handler := api.NewHandler(store.New(db.Pool))
	server := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", cfg.listenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
