-- name: CreateOutboxTask :one
INSERT INTO outbox_tasks (
    id,
    task_type,
    queue,
    payload,
    max_retry,
    process_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, 'pending'
) RETURNING *;

-- name: GetOutboxTask :one
SELECT * FROM outbox_tasks
WHERE id = $1
LIMIT 1;

-- name: ListPendingOutboxTasks :many
SELECT * FROM outbox_tasks
WHERE status = 'pending'
  AND process_at <= now()
ORDER BY process_at, created_at
LIMIT $1;

-- name: MarkOutboxTaskDispatched :one
UPDATE outbox_tasks
SET
    status = 'dispatched',
    dispatched_at = now(),
    last_error = '',
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkOutboxTaskFailed :one
UPDATE outbox_tasks
SET
    attempts = attempts + 1,
    last_error = $2,
    process_at = $3,
    status = CASE
        WHEN attempts + 1 >= max_retry THEN 'failed'
        ELSE 'pending'
    END,
    updated_at = now()
WHERE id = $1
RETURNING *;
