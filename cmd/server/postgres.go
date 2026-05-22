package main

import (
	"context"
	"log"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/database"
)

func connectWithRetry(ctx context.Context, url string) (*database.Database, error) {
	const attempts = 10
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		db, err := database.Connect(ctx, url)
		if err == nil {
			return db, nil
		}
		lastErr = err
		log.Printf("postgres not ready (attempt %d/%d): %v", attempt, attempts, err)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
