package api

import (
	"context"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

type TransactionStore interface {
	Insert(ctx context.Context, tx transaction.Transaction) error
	TransactionsSince(ctx context.Context, userID string, cutoff time.Time) ([]transaction.Transaction, error)
}

type Handler struct {
	store TransactionStore
	now   func() time.Time
}

func NewHandler(s TransactionStore) *Handler {
	return &Handler{store: s, now: time.Now}
}
