-- +goose Up
-- Attempts are the actual submissions of an event: one solo user or one team
-- per event task. Each attempt owns a workspace repository and a deployment,
-- so a competition with five teams produces five independent deployments per
-- task instead of one shared event workspace.
CREATE TABLE event_attempts (
    id            BIGSERIAL PRIMARY KEY,
    event_id      BIGINT NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    event_task_id BIGINT NOT NULL REFERENCES event_tasks (id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    kind          TEXT NOT NULL CHECK (kind IN ('solo', 'team')),
    owner_id      BIGINT REFERENCES users (id) ON DELETE SET NULL,
    join_code     TEXT NOT NULL,
    repo_path     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX event_attempts_join_code_idx ON event_attempts (join_code);
CREATE UNIQUE INDEX event_attempts_repo_path_idx ON event_attempts (repo_path);
CREATE INDEX event_attempts_event_task_id_idx ON event_attempts (event_task_id);
CREATE INDEX event_attempts_owner_id_idx ON event_attempts (owner_id);

CREATE TRIGGER event_attempts_set_updated_at
    BEFORE UPDATE ON event_attempts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Membership of one attempt. A solo attempt has its owner as the only member;
-- a team attempt has an owner plus invited members.
CREATE TABLE event_attempt_members (
    attempt_id BIGINT NOT NULL REFERENCES event_attempts (id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member'
               CHECK (role IN ('owner', 'member')),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (attempt_id, user_id)
);

CREATE INDEX event_attempt_members_user_id_idx ON event_attempt_members (user_id);

-- One deployment per attempt. The public id is the unguessable path segment of
-- /deploy/<public_id>/ and survives redeploys of the same attempt.
CREATE TABLE attempt_deployments (
    id         BIGSERIAL PRIMARY KEY,
    attempt_id BIGINT NOT NULL UNIQUE REFERENCES event_attempts (id) ON DELETE CASCADE,
    public_id  TEXT NOT NULL,
    commit_sha TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'empty'
               CHECK (status IN ('empty', 'deploying', 'live', 'failed')),
    error      TEXT NOT NULL DEFAULT '',
    published  BOOLEAN NOT NULL DEFAULT FALSE,
    deployed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX attempt_deployments_public_id_idx ON attempt_deployments (public_id);

CREATE TRIGGER attempt_deployments_set_updated_at
    BEFORE UPDATE ON attempt_deployments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Preserve the workspaces of the previous event-task model by turning each
-- existing event task into a solo attempt owned by the event host. The old
-- repository and deployment are carried over unchanged.
INSERT INTO event_attempts
    (event_id, event_task_id, name, kind, owner_id, join_code, repo_path,
     created_at, updated_at)
SELECT t.event_id, t.id, COALESCE(a.title, 'Attempt'), 'solo', e.host_id,
       SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 10),
       'events/event-' || t.event_id || '/task-' || t.id,
       t.created_at, t.created_at
FROM event_tasks t
JOIN events e ON e.id = t.event_id
JOIN assessments a ON a.id = t.assessment_id
WHERE e.host_id IS NOT NULL;

INSERT INTO event_attempt_members (attempt_id, user_id, role, joined_at)
SELECT id, owner_id, 'owner', created_at
FROM event_attempts
WHERE owner_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO attempt_deployments
    (attempt_id, public_id, commit_sha, status, error, published, deployed_at,
     created_at, updated_at)
SELECT att.id, d.public_id, d.commit_sha, d.status, d.error, d.published,
       d.deployed_at, d.created_at, d.updated_at
FROM event_deployments d
JOIN event_attempts att ON att.event_task_id = d.event_task_id;

INSERT INTO attempt_deployments (attempt_id, public_id)
SELECT att.id, SUBSTRING(MD5(RANDOM()::TEXT) FROM 1 FOR 20)
FROM event_attempts att
WHERE NOT EXISTS (
    SELECT 1 FROM attempt_deployments d WHERE d.attempt_id = att.id
);

DROP TABLE event_deployments;

-- +goose Down
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

INSERT INTO event_deployments
    (event_task_id, public_id, commit_sha, status, error, published, deployed_at,
     created_at, updated_at)
SELECT DISTINCT ON (att.event_task_id)
       att.event_task_id, d.public_id, d.commit_sha, d.status, d.error,
       d.published, d.deployed_at, d.created_at, d.updated_at
FROM attempt_deployments d
JOIN event_attempts att ON att.id = d.attempt_id
ORDER BY att.event_task_id, d.id;

DROP TABLE attempt_deployments;
DROP TABLE event_attempt_members;
DROP TABLE event_attempts;
