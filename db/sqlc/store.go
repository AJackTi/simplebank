package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	Querier
	TransferTx(context.Context, TransferTxParams) (*TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (*CreateUserTxResult, error)
}

// SQLStore provides all functions to execute db queries and transactions
type SQLStore struct {
	*Queries
	db *sql.DB
}

// NewStore creates a new store
func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rErr)
		}

		return err
	}

	return tx.Commit()
}
