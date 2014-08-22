-- +goose Up
ALTER TABLE access_tokens
ADD COLUMN parent_token TEXT REFERENCES access_tokens (access_token);

-- +goose Down
ALTER TABLE access_tokens
DROP COLUMN parent_token;
