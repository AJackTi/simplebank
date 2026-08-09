package gapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/util"
	"github.com/AJackTi/simplebank/worker"
)

func TestCreateUser(t *testing.T) {
	user, password := randomCreateUser(t)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		CreateUserTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreateUserTxParams) (*db.CreateUserTxResult, error) {
			require.Equal(t, user.Username, arg.Username)
			require.Equal(t, user.FullName, arg.FullName)
			require.Equal(t, user.Email, arg.Email)
			require.NotEmpty(t, arg.HashedPassword)
			require.Len(t, arg.OutboxTasks, 1)

			task := arg.OutboxTasks[0]
			require.Equal(t, worker.TaskSendVerifyEmail, task.TaskType)
			require.Equal(t, worker.QueueCritical, task.Queue)
			require.Equal(t, int32(10), task.MaxRetry)
			require.NotZero(t, task.ID)
			require.True(t, task.ProcessAt.After(time.Now().Add(5*time.Second)))

			var payload worker.PayloadSendVerifyEmail
			require.NoError(t, json.Unmarshal([]byte(task.Payload), &payload))
			require.Equal(t, user.Username, payload.Username)

			return &db.CreateUserTxResult{User: user}, nil
		})

	rsp, err := server.CreateUser(context.Background(), &pb.CreateUserRequest{
		Username: user.Username,
		Password: password,
		FullName: user.FullName,
		Email:    user.Email,
	})
	require.NoError(t, err)
	require.NotNil(t, rsp)
	require.True(t, proto.Equal(convertUser(&user), rsp.User))
}

func TestCreateUserReturnsInternalWhenTransactionFails(t *testing.T) {
	user, password := randomCreateUser(t)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		CreateUserTx(gomock.Any(), gomock.Any()).
		Return(nil, sql.ErrConnDone)

	rsp, err := server.CreateUser(context.Background(), &pb.CreateUserRequest{
		Username: user.Username,
		Password: password,
		FullName: user.FullName,
		Email:    user.Email,
	})
	require.Error(t, err)
	require.Nil(t, rsp)
}

func randomCreateUser(t *testing.T) (db.User, string) {
	t.Helper()

	password := util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	return user, password
}
