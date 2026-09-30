-- Curriculum phases, Capaian Pembelajaran (CP) and source provenance (§2, §19, §27).
--
-- Additive only: existing tables gain nullable columns and new tables are added.
-- Nothing is renamed or dropped, so the running product is unaffected.
--
-- Design notes:
--   * `phases` + `grades.phase_id` model Fase Fondasi (PAUD) and Fase A/B/C (SD)
--     without hardcoding grade ranges in code.
--   * `curriculum_cp.subject_id` is nullable because Fase Fondasi is organised by
--     developmental element, not by subject (§3).
--   * Provenance lives in canonical tables (not a foreign key into the pipeline's
--     staging schema), so it survives when staging is pruned.

CREATE TABLE curriculum_source_documents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code          text NOT NULL UNIQUE,
    title         text NOT NULL,
    publisher     text NOT NULL DEFAULT '',
    source_url    text NOT NULL DEFAULT '',
    document_hash text NOT NULL DEFAULT '',
    effective_date date,
    retrieved_at  timestamptz,
    license       text NOT NULL DEFAULT '',
    notes         text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE curriculum_versions ADD COLUMN revision text NOT NULL DEFAULT '';
ALTER TABLE curriculum_versions ADD COLUMN effective_date date;
ALTER TABLE curriculum_versions ADD COLUMN source_document_id uuid
    REFERENCES curriculum_source_documents (id);

CREATE TABLE phases (
    id                 serial PRIMARY KEY,
    code               text NOT NULL UNIQUE,
    name               text NOT NULL,
    education_level_id int REFERENCES education_levels (id),
    sort               int NOT NULL DEFAULT 0
);

ALTER TABLE grades ADD COLUMN phase_id int REFERENCES phases (id);
CREATE INDEX grades_phase_idx ON grades (phase_id);

-- A Capaian Pembelajaran statement. grade_id is NULL when the CP spans a whole
-- phase; subject_id is NULL for Fase Fondasi.
CREATE TABLE curriculum_cp (
    id                    serial PRIMARY KEY,
    curriculum_version_id int  NOT NULL REFERENCES curriculum_versions (id),
    phase_id              int  REFERENCES phases (id),
    grade_id              int  REFERENCES grades (id),
    subject_id            int  REFERENCES subjects (id),
    element               text NOT NULL DEFAULT '',
    code                  text NOT NULL,
    statement             text NOT NULL,
    source_document_id    uuid REFERENCES curriculum_source_documents (id),
    effective_date        date,
    sort                  int  NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (curriculum_version_id, code)
);
CREATE INDEX curriculum_cp_phase_subject_idx ON curriculum_cp (phase_id, subject_id);
CREATE INDEX curriculum_cp_grade_idx ON curriculum_cp (grade_id);

-- A skill is derived from a CP; nullable so existing skills keep working.
ALTER TABLE skills ADD COLUMN cp_id int REFERENCES curriculum_cp (id);
CREATE INDEX skills_cp_idx ON skills (cp_id);
