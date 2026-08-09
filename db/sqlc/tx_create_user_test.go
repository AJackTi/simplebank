package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AJackTi/simplebank/util"
)

func TestCreateUserTxCreatesOutboxTask(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)

	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	taskID := uuid.New()
	username := util.RandomOwner()
	result, err := store.CreateUserTx(context.Background(), CreateUserTxParams{
		CreateUserParams: CreateUserParams{
			Username:       username,
			HashedPassword: hashedPassword,
			FullName:       util.RandomOwner(),
			Email:          util.RandomEmail(),
		},
		OutboxTasks: []CreateOutboxTaskParams{
			{
				ID:        taskID,
				TaskType:  "task:send_verify_email",
				Queue:     "critical",
				Payload:   `{"username":"` + username + `"}`,
				MaxRetry:  10,
				ProcessAt: time.Now().Add(10 * time.Second),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, username, result.User.Username)

	task, err := store.GetOutboxTask(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, taskID, task.ID)
	require.Equal(t, "task:send_verify_email", task.TaskType)
	require.Equal(t, "critical", task.Queue)
	require.Equal(t, OutboxTaskStatusPending, task.Status)
}

func TestCreateUserTxRollsBackWhenOutboxTaskInsertFails(t *testing.T) {
	requireTestDatabase(t)
	store := NewStore(testDB)

	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	taskID := uuid.New()
	username := util.RandomOwner()
	_, err = store.CreateUserTx(context.Background(), CreateUserTxParams{
		CreateUserParams: CreateUserParams{
			Username:       username,
			HashedPassword: hashedPassword,
			FullName:       util.RandomOwner(),
			Email:          util.RandomEmail(),
		},
		OutboxTasks: []CreateOutboxTaskParams{
			{
				ID:        taskID,
				TaskType:  "task:send_verify_email",
				Queue:     "critical",
				Payload:   `{"username":"` + username + `"}`,
				MaxRetry:  0,
				ProcessAt: time.Now().Add(10 * time.Second),
			},
		},
	})
	require.Error(t, err)

	_, err = store.GetUser(context.Background(), username)
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = store.GetOutboxTask(context.Background(), taskID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
