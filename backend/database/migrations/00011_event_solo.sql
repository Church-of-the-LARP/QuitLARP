-- +goose Up
-- Solo workspaces are auto-created events that back one user's private
-- practice run on an unbound task. They are hidden from the events list.
ALTER TABLE events ADD COLUMN solo BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE events DROP COLUMN solo;
