CREATE TABLE transactions (
    reference_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    amount NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user_occurred_at
    ON transactions (user_id, occurred_at);
