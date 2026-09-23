-- +goose Up
-- Main branch commit the repository-derived assessment content was last
-- synced from; NULL until the first accepted push.
ALTER TABLE assessments ADD COLUMN main_commit TEXT;

-- When the repository content was last synced, so a stale assessment is
-- visible without comparing git history.
ALTER TABLE assessments ADD COLUMN synced_at TIMESTAMPTZ;

-- How a chapter starts: from its own clean state, or continued from the
-- previous chapter's state.
ALTER TABLE chapters ADD COLUMN start_mode TEXT NOT NULL DEFAULT 'clean'
    CHECK (start_mode IN ('clean', 'continue'));

-- +goose Down
ALTER TABLE chapters DROP COLUMN start_mode;
ALTER TABLE assessments DROP COLUMN synced_at;
ALTER TABLE assessments DROP COLUMN main_commit;
