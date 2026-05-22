package transaction

import (
	"errors"
	"testing"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/money"
)

const validBody = `{"transaction_id":"txn_abc123","user_id":"user_42","amount":150000.00,"timestamp":"2025-03-28T14:30:00Z"}`

func TestParseValid(t *testing.T) {
	tx, err := Parse([]byte(validBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.ID != "txn_abc123" {
		t.Errorf("ID = %q, want txn_abc123", tx.ID)
	}
	if tx.UserID != "user_42" {
		t.Errorf("UserID = %q, want user_42", tx.UserID)
	}
	want, _ := money.Parse("150000")
	if !tx.Amount.Equal(want) {
		t.Errorf("Amount = %s, want 150000", tx.Amount)
	}
	if !tx.OccurredAt.Equal(time.Date(2025, 3, 28, 14, 30, 0, 0, time.UTC)) {
		t.Errorf("OccurredAt = %v, want 2025-03-28T14:30:00Z", tx.OccurredAt)
	}
}

func assertRejected(t *testing.T, name, body, wantMsg string) {
	t.Helper()
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatalf("%s: expected a validation error", name)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("%s: error %v is not a *ValidationError", name, err)
	}
	if wantMsg != "" && ve.Message != wantMsg {
		t.Errorf("%s: message = %q, want %q", name, ve.Message, wantMsg)
	}
}

func TestParseRejectsMissingFields(t *testing.T) {
	cases := []struct{ name, body, msg string }{
		{"missing transaction_id", `{"user_id":"u","amount":1,"timestamp":"2025-03-28T14:30:00Z"}`, "transaction_id is required"},
		{"blank transaction_id", `{"transaction_id":"  ","user_id":"u","amount":1,"timestamp":"2025-03-28T14:30:00Z"}`, "transaction_id is required"},
		{"missing user_id", `{"transaction_id":"t","amount":1,"timestamp":"2025-03-28T14:30:00Z"}`, "user_id is required"},
		{"missing amount", `{"transaction_id":"t","user_id":"u","timestamp":"2025-03-28T14:30:00Z"}`, "amount is required"},
		{"missing timestamp", `{"transaction_id":"t","user_id":"u","amount":1}`, "timestamp is required"},
	}
	for _, c := range cases {
		assertRejected(t, c.name, c.body, c.msg)
	}
}

func TestParseRejectsBadAmount(t *testing.T) {
	const ts = `"timestamp":"2025-03-28T14:30:00Z"`
	assertRejected(t, "zero amount",
		`{"transaction_id":"t","user_id":"u","amount":0,`+ts+`}`,
		"amount must be a positive number")
	assertRejected(t, "negative amount",
		`{"transaction_id":"t","user_id":"u","amount":-5,`+ts+`}`,
		"amount must be a positive number")
	assertRejected(t, "too many decimal places",
		`{"transaction_id":"t","user_id":"u","amount":1.234,`+ts+`}`, "")
	assertRejected(t, "amount sent as string",
		`{"transaction_id":"t","user_id":"u","amount":"100",`+ts+`}`, "")
}

func TestParseRejectsBadTimestamp(t *testing.T) {
	assertRejected(t, "unparseable timestamp",
		`{"transaction_id":"t","user_id":"u","amount":1,"timestamp":"not-a-date"}`,
		"timestamp must be an RFC 3339 datetime")
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	assertRejected(t, "truncated body", `{"transaction_id":`, "request body is not valid JSON")
}