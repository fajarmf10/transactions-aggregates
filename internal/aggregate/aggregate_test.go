package aggregate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/money"
	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

var now = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func tx(t *testing.T, amount string, occurredAt time.Time) transaction.Transaction {
	t.Helper()
	m, err := money.Parse(amount)
	if err != nil {
		t.Fatalf("money.Parse(%q): %v", amount, err)
	}
	return transaction.Transaction{ID: "t", UserID: "u", Amount: m, OccurredAt: occurredAt}
}

func TestAggregateEmpty(t *testing.T) {
	got := Aggregate(nil, now)
	for _, w := range []Window{got.Last15m, got.Last1h, got.Last24h, got.Last30d, got.Last90d} {
		if w.Count != 0 {
			t.Errorf("empty window count = %d, want 0", w.Count)
		}
		if !w.Sum.Equal(money.Zero) || !w.Avg.Equal(money.Zero) {
			t.Errorf("empty window sum/avg = %s/%s, want 0/0", w.Sum, w.Avg)
		}
	}
}

func TestAggregateWindowPlacement(t *testing.T) {
	txns := []transaction.Transaction{
		tx(t, "10", now.Add(-5*time.Minute)),
		tx(t, "10", now.Add(-30*time.Minute)),
		tx(t, "10", now.Add(-5*time.Hour)),
		tx(t, "10", now.Add(-10*24*time.Hour)),
		tx(t, "10", now.Add(-60*24*time.Hour)),
		tx(t, "10", now.Add(-100*24*time.Hour)),
	}
	got := Aggregate(txns, now)
	for _, c := range []struct {
		name string
		want int
		got  int
	}{
		{"15m", 1, got.Last15m.Count},
		{"1h", 2, got.Last1h.Count},
		{"24h", 3, got.Last24h.Count},
		{"30d", 4, got.Last30d.Count},
		{"90d", 5, got.Last90d.Count},
	} {
		if c.got != c.want {
			t.Errorf("%s count = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestAggregateBoundaryIsInclusive(t *testing.T) {
	txns := []transaction.Transaction{tx(t, "10", now.Add(-15*time.Minute))}
	if got := Aggregate(txns, now); got.Last15m.Count != 1 {
		t.Errorf("transaction exactly 15m old: 15m count = %d, want 1", got.Last15m.Count)
	}
}

func TestAggregateFutureTimestampCounts(t *testing.T) {
	txns := []transaction.Transaction{tx(t, "10", now.Add(30*time.Second))}
	got := Aggregate(txns, now)
	if got.Last15m.Count != 1 || got.Last90d.Count != 1 {
		t.Errorf("future-dated transaction not counted: 15m=%d 90d=%d",
			got.Last15m.Count, got.Last90d.Count)
	}
}

func TestAggregateSumAndAverage(t *testing.T) {
	txns := []transaction.Transaction{
		tx(t, "100000.00", now.Add(-1*time.Minute)),
		tx(t, "200000.00", now.Add(-2*time.Minute)),
		tx(t, "150000.00", now.Add(-3*time.Minute)),
	}
	w := Aggregate(txns, now).Last15m
	if w.Count != 3 {
		t.Fatalf("count = %d, want 3", w.Count)
	}
	wantSum, _ := money.Parse("450000")
	wantAvg, _ := money.Parse("150000")
	if !w.Sum.Equal(wantSum) {
		t.Errorf("sum = %s, want 450000", w.Sum)
	}
	if !w.Avg.Equal(wantAvg) {
		t.Errorf("avg = %s, want 150000", w.Avg)
	}
}

func TestAggregateAverageRounds(t *testing.T) {
	txns := []transaction.Transaction{
		tx(t, "100.00", now.Add(-1*time.Minute)),
		tx(t, "100.01", now.Add(-2*time.Minute)),
	}
	wantAvg, _ := money.Parse("100.01")
	if w := Aggregate(txns, now).Last15m; !w.Avg.Equal(wantAvg) {
		t.Errorf("avg = %s, want 100.01 (rounded from 100.005)", w.Avg)
	}
}

func TestAggregationsJSONShape(t *testing.T) {
	txns := []transaction.Transaction{tx(t, "150000.00", now.Add(-1*time.Minute))}
	b, err := json.Marshal(Aggregate(txns, now))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	win := `{"count":1,"sum":150000.00,"avg":150000.00}`
	want := `{"15m":` + win + `,"1h":` + win + `,"24h":` + win + `,"30d":` + win + `,"90d":` + win + `}`
	if string(b) != want {
		t.Errorf("JSON shape mismatch\n got: %s\nwant: %s", b, want)
	}
}