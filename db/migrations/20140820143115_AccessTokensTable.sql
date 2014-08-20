-- +goose Up
CREATE TABLE access_tokens (
    access_token TEXT PRIMARY KEY,
    client_id TEXT REFERENCES clients ON DELETE CASCADE,
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    token_type TEXT NOT NULL,
    expires_in INTEGER NOT NULL CONSTRAINT positive_expire CHECK (expires_in > 0),
    expires_on TIMESTAMP NOT NULL,
    refresh_token TEXT
);

-- +goose Down
DROP TABLE access_tokens;
