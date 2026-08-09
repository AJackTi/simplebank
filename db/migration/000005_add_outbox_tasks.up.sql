CREATE TABLE outbox_tasks (
    id UUID PRIMARY KEY,
    task_type TEXT NOT NULL,
    queue TEXT NOT NULL,
    payload TEXT NOT NULL,
    max_retry INT NOT NULL DEFAULT 10,
    attempts INT NOT NULL DEFAULT 0,
    process_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT NOT NULL DEFAULT '',
    dispatched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT outbox_tasks_status_check CHECK (status IN ('pending', 'dispatched', 'failed')),
    CONSTRAINT outbox_tasks_max_retry_check CHECK (max_retry > 0),
    CONSTRAINT outbox_tasks_attempts_check CHECK (attempts >= 0)
);

CREATE INDEX outbox_tasks_pending_idx
    ON outbox_tasks (status, process_at, created_at);
