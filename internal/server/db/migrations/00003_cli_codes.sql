-- +goose Up
-- one-time codes for `sablier login` in the browser. only the code's hash is stored,
-- and the cli has to prove it started the login (pkce, s256)
CREATE TABLE cli_codes (
    code_hash  BYTEA       PRIMARY KEY,
    user_id    TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge  TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE cli_codes;
