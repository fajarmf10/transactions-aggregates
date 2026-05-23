//go:build integration

package store

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/database"
	"github.com/fajarmf10/transactions-aggregates/internal/money"
	"github.com/fajarmf10/transactions-aggregates/internal/testutil"
	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

var integrationDB *database.Database

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, cleanup, err := testutil.StartPostgres(ctx)
	if err != nil {
		_, _ = os.Stderr.WriteString("integration setup failed: " + err.Error() + "\n")
		os.Exit(1)
	}
	integrationDB = db
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestIntegrationInsertAndQuery(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	amount := mustMoney(t, "250.50")
	occurredAt := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	tx := transaction.Transaction{
		ID:         "txn_store_1",
		UserID:     "user_store",
		Amount:     amount,
		OccurredAt: occurredAt,
	}

	if err := s.Insert(ctx, tx); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := s.Insert(ctx, tx); err != ErrDuplicate {
		t.Fatalf("duplicate insert: got %v, want ErrDuplicate", err)
	}

	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := s.TransactionsSince(ctx, "user_store", cutoff)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != tx.ID || got[0].UserID != tx.UserID {
		t.Errorf("got %+v, want ids from inserted tx", got[0])
	}
	if !got[0].Amount.Equal(amount) {
		t.Errorf("amount = %s, want %s", got[0].Amount, amount)
	}
	if !got[0].OccurredAt.Equal(occurredAt) {
		t.Errorf("occurred_at = %v, want %v", got[0].OccurredAt, occurredAt)
	}

	// Cutoff after the transaction excludes it.
	laterCutoff := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)
	got, err = s.TransactionsSince(ctx, "user_store", laterCutoff)
	if err != nil {
		t.Fatalf("query with later cutoff: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0 when cutoff is after occurred_at", len(got))
	}
}

func TestIntegrationTransactionsSinceFiltersByUserAndCutoff(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	cutoff := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
	mustInsert(t, s, ctx, "txn_a", "user_a", "10.00", time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC))
	mustInsert(t, s, ctx, "txn_b", "user_b", "20.00", time.Date(2026, 1, 14, 13, 0, 0, 0, time.UTC))
	mustInsert(t, s, ctx, "txn_old", "user_a", "30.00", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	gotA, err := s.TransactionsSince(ctx, "user_a", cutoff)
	if err != nil {
		t.Fatalf("query user_a: %v", err)
	}
	assertTxnIDs(t, gotA, "txn_a")

	gotB, err := s.TransactionsSince(ctx, "user_b", cutoff)
	if err != nil {
		t.Fatalf("query user_b: %v", err)
	}
	assertTxnIDs(t, gotB, "txn_b")
}

func TestIntegrationTransactionsSinceCutoffIsInclusive(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	cutoff := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
	mustInsert(t, s, ctx, "txn_at_cutoff", "user_boundary", "5.00", cutoff)
	mustInsert(t, s, ctx, "txn_before", "user_boundary", "1.00", cutoff.Add(-time.Hour))

	got, err := s.TransactionsSince(ctx, "user_boundary", cutoff)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	assertTxnIDs(t, got, "txn_at_cutoff")
}

func newStore(t *testing.T) *Store {
	t.Helper()
	if err := testutil.TruncateTransactions(context.Background(), integrationDB); err != nil {
		t.Fatalf("truncate transactions: %v", err)
	}
	return New(integrationDB.Pool)
}

func assertTxnIDs(t *testing.T, got []transaction.Transaction, wantIDs ...string) {
	t.Helper()
	if len(got) != len(wantIDs) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(wantIDs), wantIDs)
	}
	ids := make([]string, len(got))
	for i, tx := range got {
		ids[i] = tx.ID
	}
	slices.Sort(ids)
	want := append([]string(nil), wantIDs...)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("transaction ids = %v, want %v", ids, want)
	}
}

func mustMoney(t *testing.T, s string) money.Money {
	t.Helper()
	m, err := money.Parse(s)
	if err != nil {
		t.Fatalf("money.Parse(%q): %v", s, err)
	}
	return m
}

func mustInsert(t *testing.T, s *Store, ctx context.Context, id, userID, amount string, occurredAt time.Time) {
	t.Helper()
	tx := transaction.Transaction{
		ID:         id,
		UserID:     userID,
		Amount:     mustMoney(t, amount),
		OccurredAt: occurredAt.UTC(),
	}
	if err := s.Insert(ctx, tx); err != nil {
		t.Fatalf("insert %s: %v", id, err)
	}
}
