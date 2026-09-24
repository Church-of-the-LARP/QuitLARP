-- +goose Up
-- Assessment kind: the challenge family the assessment belongs to. LeetCode
-- is the only kind defined today; new assessments default to it.
ALTER TABLE assessments ADD COLUMN kind TEXT NOT NULL DEFAULT 'leetcode';

-- The single candidate-edited file a chapter declares, stored as a
-- chapter-relative slash path; empty when the chapter predates the field.
ALTER TABLE chapters ADD COLUMN task_file TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE chapters DROP COLUMN task_file;
ALTER TABLE assessments DROP COLUMN kind;
