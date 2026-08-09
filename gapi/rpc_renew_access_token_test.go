package gapi

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func TestRenewAccessToken(t *testing.T) {
	user := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user, token.RefreshTokenType, time.Minute)
	require.NoError(t, err)

	session := db.Session{
		ID:           refreshPayload.ID,
		Username:     user,
		RefreshToken: refreshToken,
		UserAgent:    "test",
		ClientIp:     "127.0.0.1",
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
		CreatedAt:    time.Now(),
	}

	store.EXPECT().
		GetSession(gomock.Any(), gomock.Eq(refreshPayload.ID)).
		Return(session, nil)

	rsp, err := server.RenewAccessToken(context.Background(), &pb.RenewAccessTokenRequest{
		RefreshToken: refreshToken,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rsp.AccessToken)
	require.WithinDuration(t, time.Now().Add(server.config.AccessTokenDuration), rsp.AccessTokenExpiresAt.AsTime(), time.Second)
}

func TestRenewAccessTokenRejectsMissingSession(t *testing.T) {
	user := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user, token.RefreshTokenType, time.Minute)
	require.NoError(t, err)

	store.EXPECT().
		GetSession(gomock.Any(), gomock.Eq(refreshPayload.ID)).
		Return(db.Session{}, sql.ErrNoRows)

	rsp, err := server.RenewAccessToken(context.Background(), &pb.RenewAccessTokenRequest{
		RefreshToken: refreshToken,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}

func TestRenewAccessTokenRejectsBlockedSession(t *testing.T) {
	user := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user, token.RefreshTokenType, time.Minute)
	require.NoError(t, err)

	store.EXPECT().
		GetSession(gomock.Any(), gomock.Eq(refreshPayload.ID)).
		Return(db.Session{
			ID:           uuid.New(),
			Username:     user,
			RefreshToken: refreshToken,
			IsBlocked:    true,
			ExpiresAt:    time.Now().Add(time.Minute),
		}, nil)

	rsp, err := server.RenewAccessToken(context.Background(), &pb.RenewAccessTokenRequest{
		RefreshToken: refreshToken,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}
