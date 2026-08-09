package db

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidTransferAmount = errors.New("transfer amount must be positive")
	ErrSameAccountTransfer   = errors.New("source and destination accounts must differ")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrCurrencyMismatch      = errors.New("account currencies do not match")
)

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx performs a money transfer from one account to the other
// It creates a transfer record, add account entries, and update accounts' balance within a single database transaction
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (*TransferTxResult, error) {
	if arg.Amount <= 0 {
		return nil, ErrInvalidTransferAmount
	}
	if arg.FromAccountID == arg.ToAccountID {
		return nil, ErrSameAccountTransfer
	}

	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		fromAccount, toAccount, err := lockTransferAccounts(ctx, q, arg.FromAccountID, arg.ToAccountID)
		if err != nil {
			return err
		}
		if fromAccount.Currency != toAccount.Currency {
			return fmt.Errorf("%w: source uses %s and destination uses %s", ErrCurrencyMismatch, fromAccount.Currency, toAccount.Currency)
		}
		if fromAccount.Balance < arg.Amount {
			return fmt.Errorf("%w: account %d has balance %d, needs %d", ErrInsufficientFunds, fromAccount.ID, fromAccount.Balance, arg.Amount)
		}

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(arg))
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// get account -> update its balance
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}
		if err != nil {
			return fmt.Errorf("update account balances: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func lockTransferAccounts(
	ctx context.Context,
	q *Queries,
	fromAccountID int64,
	toAccountID int64,
) (fromAccount Account, toAccount Account, err error) {
	firstAccountID := fromAccountID
	secondAccountID := toAccountID
	if firstAccountID > secondAccountID {
		firstAccountID, secondAccountID = secondAccountID, firstAccountID
	}

	firstAccount, err := q.GetAccountForUpdate(ctx, firstAccountID)
	if err != nil {
		return Account{}, Account{}, fmt.Errorf("lock account %d: %w", firstAccountID, err)
	}
	secondAccount, err := q.GetAccountForUpdate(ctx, secondAccountID)
	if err != nil {
		return Account{}, Account{}, fmt.Errorf("lock account %d: %w", secondAccountID, err)
	}

	if fromAccountID == firstAccountID {
		return firstAccount, secondAccount, nil
	}

	return secondAccount, firstAccount, nil
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	if err != nil {
		return
	}

	return
}
