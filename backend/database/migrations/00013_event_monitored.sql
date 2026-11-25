-- +goose Up
-- Monitored sessions are created by HR (moderator/admin) and expose the host
-- results view. Collaborations created by regular users are not monitored.
ALTER TABLE events ADD COLUMN monitored BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE events e
SET monitored = TRUE
FROM users u
WHERE u.id = e.host_id AND u.role IN ('moderator', 'admin') AND NOT e.solo;

-- +goose Down
ALTER TABLE events DROP COLUMN monitored;
