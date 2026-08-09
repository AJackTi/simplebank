package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	db "github.com/AJackTi/simplebank/db/sqlc"
)

const TaskSendVerifyEmail = "task:send_verify_email"
const verifyEmailSubject = "Verify your SimpleBank account"

type PayloadSendVerifyEmail struct {
	Username string `json:"username"`
}

func NewSendVerifyEmailOutboxTask(username string) (db.CreateOutboxTaskParams, error) {
	payload, err := json.Marshal(PayloadSendVerifyEmail{Username: username})
	if err != nil {
		return db.CreateOutboxTaskParams{}, fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	return db.CreateOutboxTaskParams{
		ID:        uuid.New(),
		TaskType:  TaskSendVerifyEmail,
		Queue:     QueueCritical,
		Payload:   string(payload),
		MaxRetry:  10,
		ProcessAt: time.Now().Add(10 * time.Second),
	}, nil
}

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info().
		Str("type", task.Type()).
		Bytes("payload", task.Payload()).
		Str("queue", info.Queue).
		Int("max_retry", info.MaxRetry).
		Msg("enqueue task")
	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user doesn't exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	if processor.emailSender == nil {
		return fmt.Errorf("email sender is not configured")
	}

	content := buildVerifyEmailContent(user)
	if err := processor.emailSender.SendEmail(ctx, []string{user.Email}, verifyEmailSubject, content); err != nil {
		return fmt.Errorf("failed to send verify email: %w", err)
	}

	log.Info().
		Str("type", task.Type()).
		Bytes("payload", task.Payload()).
		Str("mail", user.Email).
		Msg("processed task")
	return nil
}

func buildVerifyEmailContent(user db.User) string {
	name := strings.TrimSpace(user.FullName)
	if name == "" {
		name = user.Username
	}

	return fmt.Sprintf(
		"Hi %s,\n\nYour SimpleBank account for username %q is ready.\nIf you did not request this account, you can ignore this email.\n\n-- SimpleBank\n",
		name,
		user.Username,
	)
}
