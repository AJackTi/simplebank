package db

import (
	"context"
)

// CreateUserTxParams contains the input parameters of the create user transaction
type CreateUserTxParams struct {
	CreateUserParams
	AfterCreate func(user User) error
}

// CreateUserTxResult is the result of the create user transaction
type CreateUserTxResult struct {
	User User
}

// CreateUserTx performs create user transaction
func (store *SQLStore) CreateUserTx(ctx context.Context, arg CreateUserTxParams) (*CreateUserTxResult, error) {
	var result CreateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}

		return arg.AfterCreate(result.User)
	})

	return &result, err
}
