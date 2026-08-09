package gapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func TestCreateTransfer(t *testing.T) {
	user := util.RandomOwner()
	fromAccount := randomAccount(user)
	toAccount := randomAccount(util.RandomOwner())
	fromAccount.Currency = util.USD
	toAccount.Currency = util.USD
	amount := util.RandomMoney()
	fromAccount.Balance = amount * 2
	toAccount.Balance = amount
	result := randomTransferResult(fromAccount, toAccount, amount)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
		Return(fromAccount, nil)
	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
		Return(toAccount, nil)
	store.EXPECT().
		TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Amount:        amount,
		})).
		Return(result, nil)

	rsp, err := server.CreateTransfer(newAuthContext(t, server, user, token.AccessTokenType), &pb.CreateTransferRequest{
		FromAccountId: fromAccount.ID,
		ToAccountId:   toAccount.ID,
		Amount:        amount,
		Currency:      util.USD,
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(convertTransferTxResult(result), rsp.Result))
}

func TestCreateTransferRejectsOtherUser(t *testing.T) {
	owner := util.RandomOwner()
	fromAccount := randomAccount(owner)
	toAccount := randomAccount(util.RandomOwner())
	fromAccount.Currency = util.USD
	toAccount.Currency = util.USD

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
		Return(fromAccount, nil)
	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
		Times(0)
	store.EXPECT().
		TransferTx(gomock.Any(), gomock.Any()).
		Times(0)

	rsp, err := server.CreateTransfer(newAuthContext(t, server, util.RandomOwner(), token.AccessTokenType), &pb.CreateTransferRequest{
		FromAccountId: fromAccount.ID,
		ToAccountId:   toAccount.ID,
		Amount:        util.RandomMoney(),
		Currency:      util.USD,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}

func TestCreateTransferHandlesInsufficientFunds(t *testing.T) {
	user := util.RandomOwner()
	fromAccount := randomAccount(user)
	toAccount := randomAccount(util.RandomOwner())
	fromAccount.Currency = util.USD
	toAccount.Currency = util.USD

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
		Return(fromAccount, nil)
	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
		Return(toAccount, nil)
	store.EXPECT().
		TransferTx(gomock.Any(), gomock.Any()).
		Return(nil, db.ErrInsufficientFunds)

	rsp, err := server.CreateTransfer(newAuthContext(t, server, user, token.AccessTokenType), &pb.CreateTransferRequest{
		FromAccountId: fromAccount.ID,
		ToAccountId:   toAccount.ID,
		Amount:        util.RandomMoney(),
		Currency:      util.USD,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}

func randomTransferResult(fromAccount, toAccount db.Account, amount int64) *db.TransferTxResult {
	now := time.Now()
	return &db.TransferTxResult{
		Transfer: db.Transfer{
			ID:            util.RandomInt(1, 1000),
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Amount:        amount,
			CreatedAt:     now,
		},
		FromAccount: db.Account{
			ID:        fromAccount.ID,
			Owner:     fromAccount.Owner,
			Balance:   fromAccount.Balance - amount,
			Currency:  fromAccount.Currency,
			CreatedAt: fromAccount.CreatedAt,
		},
		ToAccount: db.Account{
			ID:        toAccount.ID,
			Owner:     toAccount.Owner,
			Balance:   toAccount.Balance + amount,
			Currency:  toAccount.Currency,
			CreatedAt: toAccount.CreatedAt,
		},
		FromEntry: db.Entry{
			ID:        util.RandomInt(1, 1000),
			AccountID: fromAccount.ID,
			Amount:    -amount,
			CreatedAt: now,
		},
		ToEntry: db.Entry{
			ID:        util.RandomInt(1, 1000),
			AccountID: toAccount.ID,
			Amount:    amount,
			CreatedAt: now,
		},
	}
}
