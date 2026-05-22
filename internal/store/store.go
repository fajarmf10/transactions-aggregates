package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicate = errors.New("transaction already exists")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
