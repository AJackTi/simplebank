package worker

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
)

func TestProcessTaskSendVerifyEmailSendsEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sender := &recordingEmailSender{}
	processor := &RedisTaskProcessor{
		store:       store,
		emailSender: sender,
	}

	task := asynq.NewTask(TaskSendVerifyEmail, []byte(`{"username":"alice"}`))
	user := db.User{
		Username: "alice",
		FullName: "Alice Example",
		Email:    "alice@example.com",
	}
	store.EXPECT().
		GetUser(gomock.Any(), "alice").
		Return(user, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), task)
	require.NoError(t, err)
	require.Len(t, sender.calls, 1)
	require.Equal(t, []string{"alice@example.com"}, sender.calls[0].to)
	require.Equal(t, verifyEmailSubject, sender.calls[0].subject)
	require.Contains(t, sender.calls[0].content, "Alice Example")
	require.Contains(t, sender.calls[0].content, `"alice"`)
}

func TestProcessTaskSendVerifyEmailSkipsMissingUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sender := &recordingEmailSender{}
	processor := &RedisTaskProcessor{
		store:       store,
		emailSender: sender,
	}

	task := asynq.NewTask(TaskSendVerifyEmail, []byte(`{"username":"missing"}`))
	store.EXPECT().
		GetUser(gomock.Any(), "missing").
		Return(db.User{}, sql.ErrNoRows)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), task)
	require.Error(t, err)
	require.ErrorIs(t, err, asynq.SkipRetry)
	require.Empty(t, sender.calls)
}

func TestProcessTaskSendVerifyEmailReturnsSenderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sender := &recordingEmailSender{err: errors.New("smtp down")}
	processor := &RedisTaskProcessor{
		store:       store,
		emailSender: sender,
	}

	task := asynq.NewTask(TaskSendVerifyEmail, []byte(`{"username":"alice"}`))
	store.EXPECT().
		GetUser(gomock.Any(), "alice").
		Return(db.User{
			Username: "alice",
			FullName: "Alice Example",
			Email:    "alice@example.com",
		}, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), task)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send verify email")
	require.Len(t, sender.calls, 1)
}

type recordingEmailSender struct {
	err   error
	calls []recordedEmail
}

type recordedEmail struct {
	to      []string
	subject string
	content string
}

func (sender *recordingEmailSender) SendEmail(ctx context.Context, to []string, subject string, content string) error {
	sender.calls = append(sender.calls, recordedEmail{
		to:      append([]string(nil), to...),
		subject: subject,
		content: content,
	})
	return sender.err
}
