-- +goose Up
-- Assessment domain: authored coding challenges where the author ships a
-- base template file plus hidden test files that later run against the
-- candidate's library in a container.
CREATE TABLE assessments (
    id                    BIGSERIAL PRIMARY KEY,
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL,
    difficulty            TEXT NOT NULL DEFAULT 'easy'
                          CHECK (difficulty IN ('easy', 'medium', 'hard')),
    time_limit_minutes    INTEGER NOT NULL CHECK (time_limit_minutes > 0),
    template_file_name    TEXT NOT NULL,
    template_file_content TEXT NOT NULL DEFAULT '',
    author_id             BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER assessments_set_updated_at
    BEFORE UPDATE ON assessments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Tags are shared across assessments; names are unique case-insensitively.
CREATE TABLE tags (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX tags_name_lower_idx ON tags (LOWER(name));

-- Many-to-many link between assessments and tags.
CREATE TABLE assessment_tags (
    assessment_id BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    tag_id        BIGINT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (assessment_id, tag_id)
);

CREATE INDEX assessment_tags_tag_id_idx ON assessment_tags (tag_id);

-- Chapters order and structure the content of an assessment. Position is a
-- 1-based ordering hint; rows sort by (position, id).
CREATE TABLE chapters (
    id                 BIGSERIAL PRIMARY KEY,
    assessment_id      BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    position           INTEGER NOT NULL,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL,
    time_limit_minutes INTEGER NOT NULL CHECK (time_limit_minutes > 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX chapters_assessment_id_idx ON chapters (assessment_id);

CREATE TRIGGER chapters_set_updated_at
    BEFORE UPDATE ON chapters
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Hidden tests: the candidate may see name and description, but the testing
-- file itself is only readable by the author (or an admin).
CREATE TABLE tests (
    id                 BIGSERIAL PRIMARY KEY,
    assessment_id      BIGINT NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    description        TEXT NOT NULL,
    file_name          TEXT NOT NULL,
    file_content       TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tests_assessment_id_idx ON tests (assessment_id);

CREATE TRIGGER tests_set_updated_at
    BEFORE UPDATE ON tests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE tests;
DROP TABLE chapters;
DROP TABLE assessment_tags;
DROP TABLE tags;
DROP TABLE assessments;
