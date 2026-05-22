package store

import (
	"context"
	"fmt"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/money"
	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

const selectSQL = `
SELECT reference_id, user_id, amount::text, occurred_at
FROM transactions
WHERE user_id = $1 AND occurred_at >= $2
ORDER BY occurred_at`

func (s *Store) TransactionsSince(ctx context.Context, userID string, cutoff time.Time) ([]transaction.Transaction, error) {
	rows, err := s.pool.Query(ctx, selectSQL, userID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txns []transaction.Transaction
	for rows.Next() {
		var (
			id, userIDCol, amountText string
			occurredAt                time.Time
		)
		if err := rows.Scan(&id, &userIDCol, &amountText, &occurredAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		amount, err := money.Parse(amountText)
		if err != nil {
			return nil, fmt.Errorf("parse stored amount %q: %w", amountText, err)
		}
		txns = append(txns, transaction.Transaction{
			ID:         id,
			UserID:     userIDCol,
			Amount:     amount,
			OccurredAt: occurredAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return txns, nil
}
