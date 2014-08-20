-- +goose Up
CREATE TABLE clients (
    client_id TEXT PRIMARY KEY,
    client_secret TEXT NOT NULL,
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE clients;
