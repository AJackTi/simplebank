package gapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
)

func TestLoginUser(t *testing.T) {
	user, password := randomCreateUser(t)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		GetUser(gomock.Any(), gomock.Eq(user.Username)).
		Return(user, nil)
	store.EXPECT().
		CreateSession(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
			require.Equal(t, user.Username, arg.Username)
			require.NotEmpty(t, arg.RefreshToken)
			require.False(t, arg.IsBlocked)
			require.NotZero(t, arg.ExpiresAt)
			return db.Session{
				ID:           arg.ID,
				Username:     arg.Username,
				RefreshToken: arg.RefreshToken,
				UserAgent:    arg.UserAgent,
				ClientIp:     arg.ClientIp,
				IsBlocked:    arg.IsBlocked,
				ExpiresAt:    arg.ExpiresAt,
			}, nil
		})

	rsp, err := server.LoginUser(context.Background(), &pb.LoginUserRequest{
		Username: user.Username,
		Password: password,
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(convertUser(&user), rsp.User))
	require.NotEmpty(t, rsp.SessionId)
	require.NotEmpty(t, rsp.AccessToken)
	require.NotEmpty(t, rsp.RefreshToken)
	require.NotNil(t, rsp.AccessTokenExpiresAt)
	require.NotNil(t, rsp.RefreshTokenExpiresAt)
}
