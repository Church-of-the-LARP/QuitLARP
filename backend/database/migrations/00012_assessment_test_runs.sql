-- +goose Up
-- The latest checks a user ran for an assessment, so event hosts can review
-- candidate results instead of relying on an ephemeral response.
CREATE TABLE assessment_test_runs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    assessment_id BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    results       JSONB NOT NULL DEFAULT '[]',
    all_passed    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX assessment_test_runs_latest_idx
    ON assessment_test_runs (assessment_id, user_id, created_at DESC);

-- +goose Down
DROP TABLE assessment_test_runs;
