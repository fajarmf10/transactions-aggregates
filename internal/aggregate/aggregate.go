package aggregate

import (
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/money"
	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

const MaxWindow = 90 * day

const day = 24 * time.Hour

type Window struct {
	Count int         `json:"count"`
	Sum   money.Money `json:"sum"`
	Avg   money.Money `json:"avg"`
}

type Aggregations struct {
	Last15m Window `json:"15m"`
	Last1h  Window `json:"1h"`
	Last24h Window `json:"24h"`
	Last30d Window `json:"30d"`
	Last90d Window `json:"90d"`
}

type accumulator struct {
	count int
	sum   money.Money
}

func (a *accumulator) add(amount money.Money) {
	a.count++
	a.sum = a.sum.Add(amount)
}

func (a *accumulator) window() Window {
	return Window{
		Count: a.count,
		Sum:   a.sum,
		Avg:   a.sum.DivInt(a.count),
	}
}

func Aggregate(txns []transaction.Transaction, now time.Time) Aggregations {
	var w15m, w1h, w24h, w30d, w90d accumulator

	for _, tx := range txns {
		age := now.Sub(tx.OccurredAt)
		if age > MaxWindow {
			continue
		}
		w90d.add(tx.Amount)
		if age <= 30*day {
			w30d.add(tx.Amount)
		}
		if age <= day {
			w24h.add(tx.Amount)
		}
		if age <= time.Hour {
			w1h.add(tx.Amount)
		}
		if age <= 15*time.Minute {
			w15m.add(tx.Amount)
		}
	}

	return Aggregations{
		Last15m: w15m.window(),
		Last1h:  w1h.window(),
		Last24h: w24h.window(),
		Last30d: w30d.window(),
		Last90d: w90d.window(),
	}
}