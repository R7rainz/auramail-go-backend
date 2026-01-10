-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS email_summaries (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    gmail_id TEXT UNIQUE NOT NULL,
    category TEXT NOT NULL,
    company TEXT,
    role TEXT,
    summary TEXT,
    deadline TEXT,
    apply_link TEXT,
    data JSONB NOT NULL, -- The full AIResult struct as JSON
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_summaries_user_id ON email_summaries(user_id);
CREATE INDEX idx_summaries_gmail_id ON email_summaries(gmail_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_summaries;
-- +goose StatementEnd
