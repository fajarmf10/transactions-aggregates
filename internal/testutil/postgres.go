//go:build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/fajarmf10/transactions-aggregates/internal/database"
)

func StartPostgres(ctx context.Context) (*database.Database, func(), error) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		db, err := database.Connect(ctx, url)
		if err != nil {
			return nil, nil, err
		}
		if err := db.Migrate(ctx); err != nil {
			db.Close()
			return nil, nil, err
		}
		return db, db.Close, nil
	}

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("transactions"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, nil, fmt.Errorf("postgres connection string: %w", err)
	}

	db, err := database.Connect(ctx, url)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, nil, err
	}
	if err := db.Migrate(ctx); err != nil {
		db.Close()
		_ = testcontainers.TerminateContainer(container)
		return nil, nil, err
	}

	cleanup := func() {
		db.Close()
		_ = testcontainers.TerminateContainer(container)
	}
	return db, cleanup, nil
}

func TruncateTransactions(ctx context.Context, db *database.Database) error {
	_, err := db.Pool.Exec(ctx, "TRUNCATE TABLE transactions")
	return err
}
