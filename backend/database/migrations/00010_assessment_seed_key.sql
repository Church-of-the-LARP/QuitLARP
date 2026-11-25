-- +goose Up
-- Seeded assessments carry a stable key so startup can create them exactly
-- once and never duplicate or overwrite user-authored content.
ALTER TABLE assessments ADD COLUMN seed_key TEXT;

CREATE UNIQUE INDEX assessments_seed_key_idx
    ON assessments (seed_key) WHERE seed_key IS NOT NULL;

-- +goose Down
DROP INDEX assessments_seed_key_idx;
ALTER TABLE assessments DROP COLUMN seed_key;
