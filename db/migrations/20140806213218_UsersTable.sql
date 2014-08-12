-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL,
    password TEXT NOT NULL,
    salt TEXT NOT NULL
);


-- +goose Down
DROP TABLE users;

