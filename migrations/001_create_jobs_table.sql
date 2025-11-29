-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
    id VARCHAR(36) PRIMARY KEY,
    type VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    result TEXT,
    error TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    retry_after TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for efficient querying of pending jobs
CREATE INDEX idx_jobs_status_created_at ON jobs (status, created_at);

-- Index for retry queries
CREATE INDEX idx_jobs_retry_after ON jobs (retry_after) WHERE retry_after IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE jobs;
-- +goose StatementEnd


