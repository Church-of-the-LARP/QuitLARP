-- +goose Up
-- Let an HR host allow candidates to invite collaborators into their interview
-- attempt (for pair-interview formats). Off by default.
ALTER TABLE events ADD COLUMN allow_invites BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE events DROP COLUMN allow_invites;
