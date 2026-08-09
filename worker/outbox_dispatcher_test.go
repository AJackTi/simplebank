package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
)

func TestOutboxDispatcherDispatchesVerifyEmailTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	distributor := &recordingTaskDistributor{}

	task := newOutboxTaskForTest()
	store.EXPECT().
		ListPendingOutboxTasks(gomock.Any(), defaultOutboxBatchSize).
		Return([]db.OutboxTask{task}, nil)
	store.EXPECT().
		MarkOutboxTaskDispatched(gomock.Any(), task.ID).
		Return(dispatchedTask(task), nil)

	dispatcher := NewRedisOutboxDispatcher(store, distributor)
	err := dispatcher.DispatchPendingTasks(context.Background())

	require.NoError(t, err)
	require.Len(t, distributor.calls, 1)
	require.Equal(t, "alice", distributor.calls[0].payload.Username)
	requireOptionValue(t, distributor.calls[0].opts, asynq.TaskIDOpt, task.ID.String())
	requireOptionValue(t, distributor.calls[0].opts, asynq.QueueOpt, task.Queue)
	requireOptionValue(t, distributor.calls[0].opts, asynq.MaxRetryOpt, int(task.MaxRetry))
	requireTimeOptionValue(t, distributor.calls[0].opts, asynq.ProcessAtOpt, task.ProcessAt)
}

func TestOutboxDispatcherTreatsTaskIDConflictAsDispatched(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	distributor := &recordingTaskDistributor{err: asynq.ErrTaskIDConflict}

	task := newOutboxTaskForTest()
	store.EXPECT().
		ListPendingOutboxTasks(gomock.Any(), defaultOutboxBatchSize).
		Return([]db.OutboxTask{task}, nil)
	store.EXPECT().
		MarkOutboxTaskDispatched(gomock.Any(), task.ID).
		Return(dispatchedTask(task), nil)

	dispatcher := NewRedisOutboxDispatcher(store, distributor)
	err := dispatcher.DispatchPendingTasks(context.Background())

	require.NoError(t, err)
	require.Len(t, distributor.calls, 1)
}

func TestOutboxDispatcherMarksRetryOnRedisError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	distributor := &recordingTaskDistributor{err: errors.New("redis down")}

	task := newOutboxTaskForTest()
	startedAt := time.Now()
	store.EXPECT().
		ListPendingOutboxTasks(gomock.Any(), defaultOutboxBatchSize).
		Return([]db.OutboxTask{task}, nil)
	store.EXPECT().
		MarkOutboxTaskFailed(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.MarkOutboxTaskFailedParams) (db.OutboxTask, error) {
			require.Equal(t, task.ID, arg.ID)
			require.Equal(t, "redis down", arg.LastError)
			require.True(t, arg.ProcessAt.After(startedAt))
			return task, nil
		})

	dispatcher := NewRedisOutboxDispatcher(store, distributor)
	err := dispatcher.DispatchPendingTasks(context.Background())

	require.NoError(t, err)
	require.Len(t, distributor.calls, 1)
}

type recordingTaskDistributor struct {
	err   error
	calls []recordedTaskDistribution
}

type recordedTaskDistribution struct {
	payload PayloadSendVerifyEmail
	opts    []asynq.Option
}

func (distributor *recordingTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	distributor.calls = append(distributor.calls, recordedTaskDistribution{
		payload: *payload,
		opts:    append([]asynq.Option(nil), opts...),
	})
	return distributor.err
}

func newOutboxTaskForTest() db.OutboxTask {
	return db.OutboxTask{
		ID:        uuid.New(),
		TaskType:  TaskSendVerifyEmail,
		Queue:     QueueCritical,
		Payload:   `{"username":"alice"}`,
		MaxRetry:  10,
		ProcessAt: time.Now().Add(time.Minute),
		Status:    db.OutboxTaskStatusPending,
	}
}

func dispatchedTask(task db.OutboxTask) db.OutboxTask {
	task.Status = db.OutboxTaskStatusDispatched
	task.DispatchedAt.Valid = true
	task.DispatchedAt.Time = time.Now()
	return task
}

func requireOptionValue(t *testing.T, opts []asynq.Option, optionType asynq.OptionType, value any) {
	t.Helper()

	for _, opt := range opts {
		if opt.Type() == optionType {
			require.Equal(t, value, opt.Value())
			return
		}
	}

	require.Failf(t, "missing option", "option type %v was not set", optionType)
}

func requireTimeOptionValue(t *testing.T, opts []asynq.Option, optionType asynq.OptionType, value time.Time) {
	t.Helper()

	for _, opt := range opts {
		if opt.Type() == optionType {
			require.WithinDuration(t, value, opt.Value().(time.Time), time.Second)
			return
		}
	}

	require.Failf(t, "missing option", "option type %v was not set", optionType)
}
