package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func main() {
	url := flag.String("url", "http://localhost:8080/transactions", "target URL")
	tps := flag.Int("tps", 100, "target throughput in transactions per second")
	duration := flag.Duration("duration", 30*time.Second, "how long to sustain -tps")
	users := flag.Int("users", 50, "distinct user_id values (rotated per request)")
	amount := flag.Float64("amount", 100.0, "transaction amount in JSON body")
	flag.Parse()

	if *tps <= 0 {
		fmt.Fprintln(os.Stderr, "-tps must be positive")
		os.Exit(1)
	}
	if *users <= 0 {
		fmt.Fprintln(os.Stderr, "users must be positive")
		os.Exit(1)
	}

	var seq atomic.Uint64
	targeter := func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}
		n := seq.Add(1)
		userID := fmt.Sprintf("load_user_%d", n%uint64(*users))
		txnID := fmt.Sprintf("load_%d_%d", time.Now().UnixNano(), n)
		ts := time.Now().UTC().Format(time.RFC3339)

		body, err := json.Marshal(map[string]any{
			"transaction_id": txnID,
			"user_id":        userID,
			"amount":         *amount,
			"timestamp":      ts,
		})
		if err != nil {
			return err
		}

		tgt.Method = http.MethodPost
		tgt.URL = *url
		tgt.Body = body
		tgt.Header = http.Header{"Content-Type": []string{"application/json"}}
		return nil
	}

	fmt.Printf("load test: %s\n", *url)
	fmt.Printf("tps: %d  duration: %s  users: %d\n\n", *tps, *duration, *users)

	attacker := vegeta.NewAttacker(
		vegeta.Timeout(10*time.Second),
		vegeta.MaxWorkers(uint64(min(*tps, 200))),
	)
	var metrics vegeta.Metrics
	pacer := vegeta.ConstantPacer{Freq: *tps, Per: time.Second}
	for res := range attacker.Attack(targeter, pacer, *duration, "transactions") {
		metrics.Add(res)
	}
	metrics.Close()

	if metrics.Requests == 0 {
		fmt.Fprintln(os.Stderr, "no requests completed")
		os.Exit(1)
	}

	if err := vegeta.NewTextReporter(&metrics)(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		os.Exit(1)
	}

	if metrics.Success < 0.99 {
		os.Exit(1)
	}
}