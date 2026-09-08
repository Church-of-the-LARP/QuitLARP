-- +goose Up
CREATE TABLE users (
    id             BIGSERIAL PRIMARY KEY,
    username       TEXT NOT NULL,
    email          TEXT NOT NULL,
    password_hash  TEXT,                -- bcrypt; NULL for Google-only accounts
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    role           TEXT NOT NULL DEFAULT 'user'
                   CHECK (role IN ('user', 'admin', 'superadmin')),
    google_sub     TEXT,                -- Google account identifier (linked)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Case-insensitive uniqueness for the login identifiers.
CREATE UNIQUE INDEX users_username_lower_idx ON users (LOWER(username));
CREATE UNIQUE INDEX users_email_lower_idx ON users (LOWER(email));
CREATE UNIQUE INDEX users_google_sub_idx ON users (google_sub) WHERE google_sub IS NOT NULL;

-- Keep updated_at honest on row changes.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER users_set_updated_at ON users;
DROP FUNCTION set_updated_at();
DROP TABLE users;
