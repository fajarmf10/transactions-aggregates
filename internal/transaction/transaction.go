package transaction

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/fajarmf10/transactions-aggregates/internal/money"
)

type Transaction struct {
	ID         string
	UserID     string
	Amount     money.Money
	OccurredAt time.Time
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type request struct {
	TransactionID *string      `json:"transaction_id"`
	UserID        *string      `json:"user_id"`
	Amount        *money.Money `json:"amount"`
	Timestamp     *string      `json:"timestamp"`
}

func Parse(body []byte) (Transaction, error) {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return Transaction{}, &ValidationError{Message: decodeMessage(err)}
	}
	return req.toTransaction()
}

func (req request) toTransaction() (Transaction, error) {
	if req.TransactionID == nil || strings.TrimSpace(*req.TransactionID) == "" {
		return Transaction{}, invalid("transaction_id is required")
	}
	if req.UserID == nil || strings.TrimSpace(*req.UserID) == "" {
		return Transaction{}, invalid("user_id is required")
	}
	if req.Amount == nil {
		return Transaction{}, invalid("amount is required")
	}
	if !req.Amount.IsPositive() {
		return Transaction{}, invalid("amount must be a positive number")
	}
	if req.Timestamp == nil || *req.Timestamp == "" {
		return Transaction{}, invalid("timestamp is required")
	}

	occurredAt, err := time.Parse(time.RFC3339, *req.Timestamp)
	if err != nil {
		return Transaction{}, invalid("timestamp must be an RFC 3339 datetime")
	}

	return Transaction{
		ID:         *req.TransactionID,
		UserID:     *req.UserID,
		Amount:     *req.Amount,
		OccurredAt: occurredAt.UTC(),
	}, nil
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}

func decodeMessage(err error) string {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "request body is not valid JSON"
	}
	return err.Error()
}