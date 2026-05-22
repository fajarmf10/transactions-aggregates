package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fajarmf10/transactions-aggregates/internal/aggregate"
	"github.com/fajarmf10/transactions-aggregates/internal/store"
	"github.com/fajarmf10/transactions-aggregates/internal/transaction"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func (h *Handler) registerTransactionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /transactions", h.ingest)
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	now := h.now().UTC()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read request body")
		return
	}

	tx, err := transaction.Parse(body)
	if err != nil {
		if ve, ok := errors.AsType[*transaction.ValidationError](err); ok {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	ctx := r.Context()
	if err := h.store.Insert(ctx, tx); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			writeError(w, http.StatusConflict, fmt.Sprintf("Transaction %s already exists", tx.ID))
			return
		}
		writeError(w, http.StatusInternalServerError, "could not store transaction")
		return
	}

	txns, err := h.store.TransactionsSince(ctx, tx.UserID, now.Add(-aggregate.MaxWindow))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load aggregations")
		return
	}

	writeJSON(w, http.StatusCreated, ingestResponse{
		TransactionID: tx.ID,
		UserID:        tx.UserID,
		Aggregations:  aggregate.Aggregate(txns, now),
	})
}

type ingestResponse struct {
	TransactionID string                 `json:"transaction_id"`
	UserID        string                 `json:"user_id"`
	Aggregations  aggregate.Aggregations `json:"aggregations"`
}
