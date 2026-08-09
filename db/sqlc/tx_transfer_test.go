package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createTransferTestAccount(t *testing.T, balance int64, currency string) Account {
	t.Helper()

	user := createRandomUser(t)
	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Balance:  balance,
		Currency: currency,
	})
	require.NoError(t, err)

	return account
}

func requireAccountBalance(t *testing.T, store Store, accountID, balance int64) {
	t.Helper()

	account, err := store.GetAccount(context.Background(), accountID)
	require.NoError(t, err)
	require.Equal(t, balance, account.Balance)
}

func skipBalanceUpdatesForAccount(t *testing.T, accountID int64) {
	t.Helper()

	functionName := fmt.Sprintf("test_skip_balance_update_%d", accountID)
	triggerName := fmt.Sprintf("test_skip_balance_update_trigger_%d", accountID)
	_, err := testDB.ExecContext(context.Background(), fmt.Sprintf(`
CREATE FUNCTION %s() RETURNS trigger AS $$
BEGIN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER %s
BEFORE UPDATE OF balance ON accounts
FOR EACH ROW
WHEN (OLD.id = %d)
EXECUTE FUNCTION %s();`, functionName, triggerName, accountID, functionName))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, cleanupErr := testDB.ExecContext(context.Background(), fmt.Sprintf(`
DROP TRIGGER IF EXISTS %s ON accounts;
DROP FUNCTION IF EXISTS %s();`, triggerName, functionName))
		require.NoError(t, cleanupErr)
	})
}

func TestTransferTxRejectsNonPositiveAmount(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)

	for _, amount := range []int64{0, -1} {
		t.Run(fmt.Sprintf("amount=%d", amount), func(t *testing.T) {
			fromAccount := createTransferTestAccount(t, 100, "USD")
			toAccount := createTransferTestAccount(t, 50, "USD")

			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccount.ID,
				ToAccountID:   toAccount.ID,
				Amount:        amount,
			})

			require.ErrorIs(t, err, ErrInvalidTransferAmount)
			require.Nil(t, result)
			requireAccountBalance(t, store, fromAccount.ID, 100)
			requireAccountBalance(t, store, toAccount.ID, 50)
		})
	}
}

func TestTransferTxRejectsSelfTransfer(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	account := createTransferTestAccount(t, 100, "USD")

	result, err := store.TransferTx(context.Background(), TransferTxParams{
		FromAccountID: account.ID,
		ToAccountID:   account.ID,
		Amount:        10,
	})

	require.ErrorIs(t, err, ErrSameAccountTransfer)
	require.Nil(t, result)
	requireAccountBalance(t, store, account.ID, 100)
}

func TestTransferTxRejectsInsufficientFundsWithoutChangingBalances(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	fromAccount := createTransferTestAccount(t, 10, "USD")
	toAccount := createTransferTestAccount(t, 50, "USD")

	result, err := store.TransferTx(context.Background(), TransferTxParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        11,
	})

	require.ErrorIs(t, err, ErrInsufficientFunds)
	require.Nil(t, result)
	requireAccountBalance(t, store, fromAccount.ID, 10)
	requireAccountBalance(t, store, toAccount.ID, 50)
}

func TestTransferTxRejectsCurrencyMismatch(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	fromAccount := createTransferTestAccount(t, 100, "USD")
	toAccount := createTransferTestAccount(t, 50, "EUR")

	result, err := store.TransferTx(context.Background(), TransferTxParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        10,
	})

	require.ErrorIs(t, err, ErrCurrencyMismatch)
	require.Nil(t, result)
	requireAccountBalance(t, store, fromAccount.ID, 100)
	requireAccountBalance(t, store, toAccount.ID, 50)
}

func TestTransferTxReturnsAccountsByTransferRoleRegardlessOfIDOrder(t *testing.T) {
	requireTestDatabase(t)
	for _, sourceIsLowerID := range []bool{true, false} {
		testName := "source-higher-id"
		if sourceIsLowerID {
			testName = "source-lower-id"
		}

		t.Run(testName, func(t *testing.T) {
			store := NewStore(testDB)
			lowerIDAccount := createTransferTestAccount(t, 100, "USD")
			higherIDAccount := createTransferTestAccount(t, 100, "USD")

			fromAccount := lowerIDAccount
			toAccount := higherIDAccount
			if !sourceIsLowerID {
				fromAccount, toAccount = higherIDAccount, lowerIDAccount
			}

			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccount.ID,
				ToAccountID:   toAccount.ID,
				Amount:        10,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, fromAccount.ID, result.FromAccount.ID)
			require.Equal(t, int64(90), result.FromAccount.Balance)
			require.Equal(t, toAccount.ID, result.ToAccount.ID)
			require.Equal(t, int64(110), result.ToAccount.Balance)
			requireAccountBalance(t, store, fromAccount.ID, 90)
			requireAccountBalance(t, store, toAccount.ID, 110)
		})
	}
}

func TestTransferTxPropagatesBalanceUpdateErrorAndRollsBack(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	fromAccount := createTransferTestAccount(t, 10, "USD")
	toAccount := createTransferTestAccount(t, 20, "USD")
	skipBalanceUpdatesForAccount(t, fromAccount.ID)

	result, err := store.TransferTx(context.Background(), TransferTxParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        1,
	})

	require.Error(t, err)
	require.Nil(t, result)
	requireAccountBalance(t, store, fromAccount.ID, 10)
	requireAccountBalance(t, store, toAccount.ID, 20)

	transfers, listTransfersErr := store.ListTransfers(context.Background(), ListTransfersParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   fromAccount.ID,
		Limit:         1,
	})
	require.NoError(t, listTransfersErr)
	require.Empty(t, transfers)
}

func TestTransferTxHandlesConcurrentTransfers(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	fromAccount := createTransferTestAccount(t, 1_000, "USD")
	toAccount := createTransferTestAccount(t, 500, "USD")

	const (
		transferCount = 5
		amount        = int64(10)
	)
	type outcome struct {
		result *TransferTxResult
		err    error
	}
	outcomes := make(chan outcome, transferCount)

	for range transferCount {
		go func() {
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccount.ID,
				ToAccountID:   toAccount.ID,
				Amount:        amount,
			})
			outcomes <- outcome{result: result, err: err}
		}()
	}

	seenDebits := make(map[int64]bool, transferCount)
	for range transferCount {
		outcome := <-outcomes
		require.NoError(t, outcome.err)
		require.NotNil(t, outcome.result)

		result := outcome.result
		require.Equal(t, fromAccount.ID, result.Transfer.FromAccountID)
		require.Equal(t, toAccount.ID, result.Transfer.ToAccountID)
		require.Equal(t, amount, result.Transfer.Amount)
		require.Equal(t, fromAccount.ID, result.FromEntry.AccountID)
		require.Equal(t, -amount, result.FromEntry.Amount)
		require.Equal(t, toAccount.ID, result.ToEntry.AccountID)
		require.Equal(t, amount, result.ToEntry.Amount)

		debit := fromAccount.Balance - result.FromAccount.Balance
		require.Equal(t, debit, result.ToAccount.Balance-toAccount.Balance)
		require.GreaterOrEqual(t, debit, amount)
		require.LessOrEqual(t, debit, int64(transferCount)*amount)
		require.Zero(t, debit%amount)
		require.False(t, seenDebits[debit])
		seenDebits[debit] = true
	}

	requireAccountBalance(t, store, fromAccount.ID, 950)
	requireAccountBalance(t, store, toAccount.ID, 550)
}

func TestTransferTxAvoidsDeadlockForOppositeDirections(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)
	account1 := createTransferTestAccount(t, 100, "USD")
	account2 := createTransferTestAccount(t, 100, "USD")

	const (
		transferCount = 10
		amount        = int64(10)
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errs := make(chan error, transferCount)

	for i := range transferCount {
		fromAccountID := account1.ID
		toAccountID := account2.ID
		if i%2 == 1 {
			fromAccountID, toAccountID = toAccountID, fromAccountID
		}

		go func() {
			_, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})
			errs <- err
		}()
	}

	for range transferCount {
		require.NoError(t, <-errs)
	}
	require.NoError(t, ctx.Err())
	requireAccountBalance(t, store, account1.ID, 100)
	requireAccountBalance(t, store, account2.ID, 100)
}
