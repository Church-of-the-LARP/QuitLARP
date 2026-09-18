-- +goose Up
-- Live-taking runtime for assessments: a host's scheduled group invite,
-- one private attempt per person per run, and the per-attempt chapter
-- cursor. See docs/assessment-runtime/TASK.md for the full design.

-- A host's scheduled sitting for a group. Always scheduled — there is no
-- unscheduled/open-ended invite in this design. It has no roster and no
-- state machine of its own beyond "can someone still accept it": just a
-- shareable code, a mandatory start time, and an optional headcount.
CREATE TABLE assessment_invites (
    id                 BIGSERIAL PRIMARY KEY,
    assessment_id      BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    host_id            BIGINT REFERENCES users (id) ON DELETE SET NULL,
    invite_code        TEXT NOT NULL UNIQUE,
    scheduled_start_at TIMESTAMPTZ NOT NULL,
    time_limit_minutes INTEGER CHECK (time_limit_minutes > 0),
    max_uses           INTEGER,
    uses_count         INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX assessment_invites_assessment_id_idx ON assessment_invites (assessment_id);

-- One row per person's one run through one assessment. invite_id NULL
-- means a solo, self-started attempt; non-NULL means it was created by
-- accepting that invite. time_limit_minutes is copied from the
-- assessment (or the invite's override) at creation time, so a later
-- edit to the assessment doesn't retroactively change an in-flight
-- attempt's deadline.
CREATE TABLE assessment_attempts (
    id                 BIGSERIAL PRIMARY KEY,
    assessment_id      BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    invite_id          BIGINT REFERENCES assessment_invites (id) ON DELETE SET NULL,
    user_id            BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status             TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'running', 'ended')),
    time_limit_minutes INTEGER NOT NULL CHECK (time_limit_minutes > 0),
    current_chapter_id BIGINT REFERENCES chapters (id) ON DELETE SET NULL,
    started_at         TIMESTAMPTZ,
    ended_at           TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX assessment_attempts_assessment_id_idx ON assessment_attempts (assessment_id);

-- Cheap "what's currently live" queries.
CREATE INDEX assessment_attempts_status_idx
    ON assessment_attempts (status) WHERE status != 'ended';

-- Retakes are allowed (a user may accumulate several ENDED attempts at
-- the same assessment), but at most one ACTIVE (non-ended) attempt at a
-- time — otherwise two browser tabs, or a solo start racing an invite
-- accept, could each spin up their own live Runner over the same
-- person's progress. A plain UNIQUE constraint can't express "unique
-- only while active" (and wouldn't even stop duplicate solo attempts,
-- since Postgres treats NULLs as distinct), hence the partial index.
CREATE UNIQUE INDEX assessment_attempts_one_active_idx
    ON assessment_attempts (assessment_id, user_id) WHERE status != 'ended';

CREATE TRIGGER assessment_attempts_set_updated_at
    BEFORE UPDATE ON assessment_attempts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The per-attempt chapter cursor. At most one progress row per chapter
-- per attempt. The "reject skip-ahead" check reads this table: before
-- advancing to chapter N+1, confirm chapter N has a row here with
-- completed_at set.
CREATE TABLE chapter_progress (
    id           BIGSERIAL PRIMARY KEY,
    attempt_id   BIGINT NOT NULL REFERENCES assessment_attempts (id) ON DELETE CASCADE,
    chapter_id   BIGINT NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE (attempt_id, chapter_id)
);

CREATE INDEX chapter_progress_attempt_id_idx ON chapter_progress (attempt_id);

-- +goose Down
DROP TABLE chapter_progress;
DROP TABLE assessment_attempts;
DROP TABLE assessment_invites;
