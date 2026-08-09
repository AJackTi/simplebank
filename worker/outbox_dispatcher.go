package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	db "github.com/AJackTi/simplebank/db/sqlc"
)

const defaultOutboxBatchSize int32 = 100

type OutboxDispatcher interface {
	Start(ctx context.Context)
	DispatchPendingTasks(ctx context.Context) error
}

type RedisOutboxDispatcher struct {
	store       db.Store
	distributor TaskDistributor
	batchSize   int32
	interval    time.Duration
}

func NewRedisOutboxDispatcher(store db.Store, distributor TaskDistributor) OutboxDispatcher {
	return &RedisOutboxDispatcher{
		store:       store,
		distributor: distributor,
		batchSize:   defaultOutboxBatchSize,
		interval:    time.Second,
	}
}

func (dispatcher *RedisOutboxDispatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(dispatcher.interval)
	defer ticker.Stop()

	for {
		if err := dispatcher.DispatchPendingTasks(ctx); err != nil {
			log.Error().Err(err).Msg("dispatch outbox tasks failed")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (dispatcher *RedisOutboxDispatcher) DispatchPendingTasks(ctx context.Context) error {
	tasks, err := dispatcher.store.ListPendingOutboxTasks(ctx, dispatcher.batchSize)
	if err != nil {
		return fmt.Errorf("list pending outbox tasks: %w", err)
	}

	for _, task := range tasks {
		if err := dispatcher.dispatchTask(ctx, task); err != nil {
			log.Error().
				Err(err).
				Str("task_id", task.ID.String()).
				Str("task_type", task.TaskType).
				Msg("failed to dispatch outbox task")
		}
	}

	return nil
}

func (dispatcher *RedisOutboxDispatcher) dispatchTask(ctx context.Context, task db.OutboxTask) error {
	switch task.TaskType {
	case TaskSendVerifyEmail:
		return dispatcher.dispatchSendVerifyEmail(ctx, task)
	default:
		return dispatcher.markTaskFailed(ctx, task, fmt.Errorf("unsupported outbox task type: %s", task.TaskType))
	}
}

func (dispatcher *RedisOutboxDispatcher) dispatchSendVerifyEmail(ctx context.Context, task db.OutboxTask) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return dispatcher.markTaskFailed(ctx, task, fmt.Errorf("unmarshal outbox payload: %w", err))
	}

	opts := []asynq.Option{
		asynq.TaskID(task.ID.String()),
		asynq.Queue(task.Queue),
		asynq.MaxRetry(int(task.MaxRetry)),
		asynq.ProcessAt(task.ProcessAt),
	}

	err := dispatcher.distributor.DistributeTaskSendVerifyEmail(ctx, &payload, opts...)
	if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
		return dispatcher.markTaskFailed(ctx, task, err)
	}

	if _, err := dispatcher.store.MarkOutboxTaskDispatched(ctx, task.ID); err != nil {
		return fmt.Errorf("mark task dispatched: %w", err)
	}

	return nil
}

func (dispatcher *RedisOutboxDispatcher) markTaskFailed(ctx context.Context, task db.OutboxTask, dispatchErr error) error {
	nextProcessAt := time.Now().Add(time.Second * time.Duration(task.Attempts+1))
	_, err := dispatcher.store.MarkOutboxTaskFailed(ctx, db.MarkOutboxTaskFailedParams{
		ID:        task.ID,
		LastError: dispatchErr.Error(),
		ProcessAt: nextProcessAt,
	})
	if err != nil {
		return fmt.Errorf("mark task failed: %w", err)
	}

	return nil
}
