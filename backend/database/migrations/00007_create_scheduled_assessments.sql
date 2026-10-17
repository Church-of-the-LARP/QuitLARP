-- +goose Up
-- A user's planned attempt at an assessment. Powers the dashboard's
-- "upcoming tests" list. One row per user and assessment; rescheduling
-- overwrites scheduled_at.
CREATE TABLE scheduled_assessments (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    assessment_id BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    scheduled_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, assessment_id)
);

CREATE INDEX scheduled_assessments_user_time_idx
    ON scheduled_assessments (user_id, scheduled_at);

CREATE TRIGGER scheduled_assessments_set_updated_at
    BEFORE UPDATE ON scheduled_assessments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE scheduled_assessments;
