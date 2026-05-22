package store

import (
	"context"
	"fmt"

	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

const insertSQL = `
INSERT INTO transactions (reference_id, user_id, amount, occurred_at)
VALUES ($1, $2, $3::numeric, $4)
ON CONFLICT (reference_id) DO NOTHING`

func (s *Store) Insert(ctx context.Context, tx transaction.Transaction) error {
	tag, err := s.pool.Exec(ctx, insertSQL,
		tx.ID, tx.UserID, tx.Amount.String(), tx.OccurredAt)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDuplicate
	}
	return nil
}
