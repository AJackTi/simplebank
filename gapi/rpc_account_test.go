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

func TestCreateAccount(t *testing.T) {
	user := util.RandomOwner()
	account := randomAccount(user)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
			Owner:    user,
			Balance:  0,
			Currency: account.Currency,
		})).
		Return(account, nil)

	rsp, err := server.CreateAccount(newAuthContext(t, server, user, token.AccessTokenType), &pb.CreateAccountRequest{
		Currency: account.Currency,
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(convertAccount(&account), rsp.Account))
}

func TestGetAccount(t *testing.T) {
	user := util.RandomOwner()
	account := randomAccount(user)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(account.ID)).
		Return(account, nil)

	rsp, err := server.GetAccount(newAuthContext(t, server, user, token.AccessTokenType), &pb.GetAccountRequest{
		Id: account.ID,
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(convertAccount(&account), rsp.Account))
}

func TestGetAccountRejectsOtherUser(t *testing.T) {
	owner := util.RandomOwner()
	account := randomAccount(owner)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(account.ID)).
		Return(account, nil)

	rsp, err := server.GetAccount(newAuthContext(t, server, util.RandomOwner(), token.AccessTokenType), &pb.GetAccountRequest{
		Id: account.ID,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}

func TestListAccounts(t *testing.T) {
	user := util.RandomOwner()
	accounts := randomAccounts(user, 3)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		ListAccounts(gomock.Any(), gomock.Eq(db.ListAccountsParams{
			Owner:  user,
			Limit:  5,
			Offset: 0,
		})).
		Return(accounts, nil)

	rsp, err := server.ListAccounts(newAuthContext(t, server, user, token.AccessTokenType), &pb.ListAccountsRequest{
		PageId:   1,
		PageSize: 5,
	})
	require.NoError(t, err)
	require.Len(t, rsp.Accounts, len(accounts))
	for i := range accounts {
		require.True(t, proto.Equal(convertAccount(&accounts[i]), rsp.Accounts[i]))
	}
}

func randomAccount(owner string) db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}

func randomAccounts(owner string, count int) []db.Account {
	accounts := make([]db.Account, count)
	for i := range accounts {
		accounts[i] = randomAccount(owner)
	}
	return accounts
}
