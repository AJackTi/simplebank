package gapi

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func TestUpdateUser(t *testing.T) {
	user, _ := randomCreateUser(t)
	newFullName := util.RandomOwner()
	newEmail := util.RandomEmail()
	newPassword := util.RandomString(6)
	updatedUser := user
	updatedUser.FullName = newFullName
	updatedUser.Email = newEmail
	startedAt := time.Now()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
			require.Equal(t, user.Username, arg.Username)
			require.True(t, arg.FullName.Valid)
			require.Equal(t, newFullName, arg.FullName.String)
			require.True(t, arg.Email.Valid)
			require.Equal(t, newEmail, arg.Email.String)
			require.True(t, arg.HashedPassword.Valid)
			require.NoError(t, util.CheckPassword(newPassword, arg.HashedPassword.String))
			require.True(t, arg.PasswordChangedAt.Valid)
			require.WithinDuration(t, startedAt, arg.PasswordChangedAt.Time, 2*time.Second)

			updatedUser.HashedPassword = arg.HashedPassword.String
			updatedUser.PasswordChangedAt = arg.PasswordChangedAt.Time
			return updatedUser, nil
		})

	rsp, err := server.UpdateUser(newAuthContext(t, server, user.Username, token.AccessTokenType), &pb.UpdateUserRequest{
		Username: user.Username,
		FullName: &newFullName,
		Email:    &newEmail,
		Password: &newPassword,
	})

	require.NoError(t, err)
	require.True(t, proto.Equal(convertUser(&updatedUser), rsp.User))
}

func TestUpdateUserRejectsOtherUser(t *testing.T) {
	user, _ := randomCreateUser(t)
	newFullName := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Times(0)

	rsp, err := server.UpdateUser(newAuthContext(t, server, util.RandomOwner(), token.AccessTokenType), &pb.UpdateUserRequest{
		Username: user.Username,
		FullName: &newFullName,
	})

	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Nil(t, rsp)
}

func TestUpdateUserRejectsInvalidEmail(t *testing.T) {
	user, _ := randomCreateUser(t)
	invalidEmail := "not-an-email"

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Times(0)

	rsp, err := server.UpdateUser(newAuthContext(t, server, user.Username, token.AccessTokenType), &pb.UpdateUserRequest{
		Username: user.Username,
		Email:    &invalidEmail,
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Nil(t, rsp)
}

func TestUpdateUserReturnsNotFound(t *testing.T) {
	user, _ := randomCreateUser(t)
	newFullName := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Return(db.User{}, sql.ErrNoRows)

	rsp, err := server.UpdateUser(newAuthContext(t, server, user.Username, token.AccessTokenType), &pb.UpdateUserRequest{
		Username: user.Username,
		FullName: &newFullName,
	})

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Nil(t, rsp)
}
