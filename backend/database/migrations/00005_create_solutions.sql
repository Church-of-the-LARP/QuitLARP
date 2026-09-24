-- +goose Up
-- One row per user and assessment: the persisted solution repository. The
-- repository path is derived from (assessment_id, user_id), see
-- models.SolutionRepoID, so it is not stored here. Deleting the solver
-- removes the row with it.
CREATE TABLE solutions (
    id            BIGSERIAL PRIMARY KEY,
    assessment_id BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    user_id       BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    is_public     BOOLEAN NOT NULL DEFAULT FALSE,
    commit_sha    TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assessment_id, user_id)
);

CREATE TRIGGER solutions_set_updated_at
    BEFORE UPDATE ON solutions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE solutions;
