-- Content pipeline: material candidates (M4, §21, §30, §40).
--
-- A material candidate is a chunk that has been mapped to a skill/CP and turned
-- into a reviewable teaching snippet, with child-readability metadata and the
-- results of the automatic quality checks. Canonical ids are copied, not FKs.
-- Owner: the Python content-pipeline service.

CREATE TABLE IF NOT EXISTS content_staging.material_candidates (
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
    title                 text NOT NULL,
    summary               text NOT NULL,
    learning_objective    text NOT NULL DEFAULT '',
    difficulty_hint       smallint CHECK (difficulty_hint BETWEEN 1 AND 5),
    language              text NOT NULL DEFAULT '',
    -- child readability (§30)
    age_band              text NOT NULL DEFAULT '',
    reading_level         text NOT NULL DEFAULT '',
    complexity            text NOT NULL DEFAULT '',
    word_count            int NOT NULL DEFAULT 0,
    sentence_count        int NOT NULL DEFAULT 0,
    avg_sentence_length   numeric(5,2) NOT NULL DEFAULT 0,
    visual_required       boolean NOT NULL DEFAULT false,
    audio_required        boolean NOT NULL DEFAULT false,
    content_hash          text NOT NULL,
    duplicate_of          bigint REFERENCES content_staging.material_candidates (id) ON DELETE SET NULL,
    confidence            numeric(4,3) NOT NULL DEFAULT 0,
    -- automatic quality checks (§40): [{code, message, blocking}]
    qa_issues             jsonb NOT NULL DEFAULT '[]',
    status                text NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'needs_review', 'approved', 'rejected', 'published', 'retired')),
    created_by            text NOT NULL DEFAULT 'pipeline',
    reviewed_by           text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chunk_id)
);
CREATE INDEX IF NOT EXISTS material_candidates_source_idx
    ON content_staging.material_candidates (source_id, status);
CREATE INDEX IF NOT EXISTS material_candidates_hash_idx
    ON content_staging.material_candidates (content_hash);
CREATE INDEX IF NOT EXISTS material_candidates_skill_idx
    ON content_staging.material_candidates (skill_id);
