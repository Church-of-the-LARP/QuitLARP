-- +goose Up
-- One row per event a user should be told about. read_at is null while
-- unread; it never stores who else can see it, notifications are private.
CREATE TABLE notifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    detail     TEXT NOT NULL,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notifications_user_id_created_at_idx ON notifications (user_id, created_at DESC);

-- +goose Down
DROP TABLE notifications;
