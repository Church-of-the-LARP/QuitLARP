-- +goose Up
-- Events group unbound tasks into a hosted, collaborative session (interview
-- or competition). The host runs the event; members share its workspaces.
CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    host_id     BIGINT REFERENCES users (id) ON DELETE SET NULL,
    status      TEXT NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'live', 'ended')),
    join_code   TEXT NOT NULL,
    starts_at   TIMESTAMPTZ,
    ends_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX events_join_code_idx ON events (join_code);
CREATE INDEX events_host_id_idx ON events (host_id);

CREATE TRIGGER events_set_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Participants of an event. The host is also stored here with role 'host' so
-- membership checks have one place to look.
CREATE TABLE event_members (
    event_id  BIGINT NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id   BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member'
              CHECK (role IN ('host', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX event_members_user_id_idx ON event_members (user_id);

-- Unbound tasks attached to an event. Each row owns one workspace repository
-- and one deployment; the task definition itself lives in assessments.
CREATE TABLE event_tasks (
    id            BIGSERIAL PRIMARY KEY,
    event_id      BIGINT NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    assessment_id BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    position      INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, assessment_id)
);

CREATE INDEX event_tasks_event_id_idx ON event_tasks (event_id);

-- Static deployment of one event workspace. public_id is the unguessable
-- path segment of /deploy/<public_id>/; the host publishes it after the
-- event so it appears as a finished project.
CREATE TABLE event_deployments (
    id            BIGSERIAL PRIMARY KEY,
    event_task_id BIGINT NOT NULL UNIQUE REFERENCES event_tasks (id) ON DELETE CASCADE,
    public_id     TEXT NOT NULL,
    commit_sha    TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'empty'
                  CHECK (status IN ('empty', 'deploying', 'live', 'failed')),
    error         TEXT NOT NULL DEFAULT '',
    published     BOOLEAN NOT NULL DEFAULT FALSE,
    deployed_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX event_deployments_public_id_idx ON event_deployments (public_id);

CREATE TRIGGER event_deployments_set_updated_at
    BEFORE UPDATE ON event_deployments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE event_deployments;
DROP TABLE event_tasks;
DROP TABLE event_members;
DROP TABLE events;
