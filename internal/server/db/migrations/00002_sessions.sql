-- +goose Up
-- web sign-in. only the cookie token's hash is stored
CREATE TABLE sessions (
    token_hash BYTEA       PRIMARY KEY,
    user_id    TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE sessions;
