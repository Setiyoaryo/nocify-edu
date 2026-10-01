-- Content pipeline: question candidates (M4, §22, §40).
--
-- A question candidate is a chunk that has been mapped to a skill/CP and turned
-- into a reviewable question by a deterministic template (§23). The generated
-- payload must pass the shared question schema (packages/question-schema) before
-- it is stored; a reviewer is still the final authority (§24). Canonical ids are
-- copied, not FKs. Owner: the Python content-pipeline service.

CREATE TABLE IF NOT EXISTS content_staging.question_candidates (
    id                    bigserial PRIMARY KEY,
    document_id           bigint NOT NULL REFERENCES content_staging.source_documents (id) ON DELETE CASCADE,
    chunk_id              bigint NOT NULL REFERENCES content_staging.document_chunks (id) ON DELETE CASCADE,
    source_id             int NOT NULL,
    -- canonical identifiers, copied at generation time (no cross-schema FK)
    curriculum_version_id int,
    skill_id              int,
    cp_id                 int,
    phase_code            text NOT NULL DEFAULT '',
    grade_code            text NOT NULL DEFAULT '',
    subject_code          text NOT NULL DEFAULT '',
    template              text NOT NULL DEFAULT 'cloze',
    type                  text NOT NULL,
    difficulty            smallint CHECK (difficulty BETWEEN 1 AND 5),
    learning_objective    text NOT NULL DEFAULT '',
    prompt                text NOT NULL,
    -- full question envelope: {code, skill, type, difficulty, learning_objective,
    -- content, answer, explanation}; validated against the shared schema
    payload               jsonb NOT NULL,
    content_hash          text NOT NULL,
    duplicate_of          bigint REFERENCES content_staging.question_candidates (id) ON DELETE SET NULL,
    confidence            numeric(4,3) NOT NULL DEFAULT 0,
    -- automatic quality checks (§40): [{code, message, blocking}]
    qa_issues             jsonb NOT NULL DEFAULT '[]',
    status                text NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'needs_review', 'approved', 'rejected', 'published', 'retired')),
    created_by            text NOT NULL DEFAULT 'pipeline',
    reviewed_by           text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    -- one candidate per template per chunk, so re-runs replace in place
    UNIQUE (chunk_id, template)
);
CREATE INDEX IF NOT EXISTS question_candidates_source_idx
    ON content_staging.question_candidates (source_id, status);
CREATE INDEX IF NOT EXISTS question_candidates_hash_idx
    ON content_staging.question_candidates (content_hash);
CREATE INDEX IF NOT EXISTS question_candidates_skill_idx
    ON content_staging.question_candidates (skill_id);
