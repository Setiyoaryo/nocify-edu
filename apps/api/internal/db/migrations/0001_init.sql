-- EduPlay MVP schema. See docs/data-model.md for the rationale behind each table.

-- ---------------------------------------------------------------- accounts
CREATE TABLE users (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email           text NOT NULL,
    password_hash   text NOT NULL,
    display_name    text NOT NULL,
    role            text NOT NULL DEFAULT 'parent'
                    CHECK (role IN ('parent', 'content_editor', 'reviewer', 'admin')),
    consent_version text,
    consent_at      timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

-- Server-side sessions. id is sha256(token) so a DB leak does not leak live tokens.
CREATE TABLE sessions (
    id           text PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

-- ---------------------------------------------------------------- curriculum
CREATE TABLE curriculum_versions (
    id         serial PRIMARY KEY,
    code       text NOT NULL UNIQUE,
    name       text NOT NULL,
    is_active  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE education_levels (
    id   serial PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    sort int  NOT NULL DEFAULT 0
);

-- session_size and ux_band are data, not code: UX differs per band (§5.4, §8.2).
CREATE TABLE grades (
    id                 serial PRIMARY KEY,
    education_level_id int  NOT NULL REFERENCES education_levels (id),
    code               text NOT NULL UNIQUE,
    name               text NOT NULL,
    sort               int  NOT NULL DEFAULT 0,
    session_size       int  NOT NULL DEFAULT 8 CHECK (session_size BETWEEN 3 AND 30),
    ux_band            text NOT NULL DEFAULT 'early'
                       CHECK (ux_band IN ('early', 'lower', 'upper', 'secondary'))
);

CREATE TABLE subjects (
    id     serial PRIMARY KEY,
    code   text NOT NULL UNIQUE,
    name   text NOT NULL,
    motif  text NOT NULL DEFAULT '',
    sort   int  NOT NULL DEFAULT 0
);

CREATE TABLE chapters (
    id                    serial PRIMARY KEY,
    curriculum_version_id int  NOT NULL REFERENCES curriculum_versions (id),
    grade_id              int  NOT NULL REFERENCES grades (id),
    subject_id            int  NOT NULL REFERENCES subjects (id),
    code                  text NOT NULL,
    name                  text NOT NULL,
    sort                  int  NOT NULL DEFAULT 0,
    UNIQUE (curriculum_version_id, code)
);
CREATE INDEX chapters_grade_subject_idx ON chapters (grade_id, subject_id);

CREATE TABLE skills (
    id          serial PRIMARY KEY,
    chapter_id  int  NOT NULL REFERENCES chapters (id),
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    sort        int  NOT NULL DEFAULT 0
);

-- ---------------------------------------------------------------- children
CREATE TABLE children (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    nickname   text NOT NULL CHECK (char_length(nickname) BETWEEN 1 AND 30),
    grade_id   int  NOT NULL REFERENCES grades (id),
    avatar     text NOT NULL DEFAULT 'fox',
    pin_hash   text,
    xp         int  NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX children_parent_idx ON children (parent_id);

-- ---------------------------------------------------------------- assets
CREATE TABLE asset_licenses (
    id                     serial PRIMARY KEY,
    name                   text NOT NULL UNIQUE,
    url                    text NOT NULL,
    commercial_use         boolean NOT NULL,
    attribution_required   boolean NOT NULL,
    redistribution_allowed boolean NOT NULL DEFAULT false,
    notes                  text NOT NULL DEFAULT ''
);

CREATE TABLE assets (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code               text NOT NULL UNIQUE,            -- e.g. ILL-0001
    name               text NOT NULL,
    kind               text NOT NULL CHECK (kind IN ('logo','mascot','character','illustration','icon',
                                    'sticker','reward','background','shape','animation','sound','diagram','photo')),
    source             text NOT NULL,
    creator            text NOT NULL,
    license_id         int  REFERENCES asset_licenses (id),
    license_url        text NOT NULL DEFAULT '',
    downloaded_at      date,
    original_file      text NOT NULL DEFAULT '',
    modified           boolean NOT NULL DEFAULT false,
    modification_notes text NOT NULL DEFAULT '',
    ai_generated       boolean NOT NULL DEFAULT false,
    usage              text[] NOT NULL DEFAULT '{}',
    storage_key        text,
    mime_type          text,
    size_bytes         bigint,
    status             text NOT NULL DEFAULT 'imported'
                       CHECK (status IN ('imported','license_verified','design_approved','performance_checked','published','rejected')),
    created_by         uuid REFERENCES users (id),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- Anti AI slop policy (§27.1): AI imagery can never reach production.
    CONSTRAINT assets_no_ai_in_production CHECK (NOT (ai_generated AND status = 'published'))
);

-- ---------------------------------------------------------------- questions
CREATE TABLE questions (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code                 text NOT NULL UNIQUE,
    skill_id             int  NOT NULL REFERENCES skills (id),
    type                 text NOT NULL,
    difficulty           int  NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
    status               text NOT NULL DEFAULT 'draft'
                         CHECK (status IN ('draft','review','approved','published','retired')),
    learning_objective   text NOT NULL,
    source               text NOT NULL DEFAULT '',
    current_version_id   uuid,
    published_version_id uuid,
    created_by           uuid REFERENCES users (id),
    reviewer_id          uuid REFERENCES users (id),
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    published_at         timestamptz
);
CREATE INDEX questions_skill_status_idx ON questions (skill_id, status);

-- Versions are append-only: attempts point at the exact version the child saw.
CREATE TABLE question_versions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id uuid NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    version     int  NOT NULL,
    content     jsonb NOT NULL,
    answer      jsonb NOT NULL,
    explanation text NOT NULL,
    hint        text NOT NULL DEFAULT '',
    created_by  uuid REFERENCES users (id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (question_id, version)
);
ALTER TABLE questions ADD CONSTRAINT questions_current_version_fk
    FOREIGN KEY (current_version_id) REFERENCES question_versions (id) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE questions ADD CONSTRAINT questions_published_version_fk
    FOREIGN KEY (published_version_id) REFERENCES question_versions (id) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE question_media (
    question_version_id uuid NOT NULL REFERENCES question_versions (id) ON DELETE CASCADE,
    asset_id            uuid NOT NULL REFERENCES assets (id),
    role                text NOT NULL DEFAULT 'image',
    PRIMARY KEY (question_version_id, asset_id)
);

-- Content quality gate (§24): every approval records the checklist that was ticked.
CREATE TABLE question_reviews (
    id                  bigserial PRIMARY KEY,
    question_id         uuid NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    question_version_id uuid NOT NULL REFERENCES question_versions (id) ON DELETE CASCADE,
    reviewer_id         uuid NOT NULL REFERENCES users (id),
    from_status         text NOT NULL,
    to_status           text NOT NULL,
    checklist           jsonb NOT NULL DEFAULT '{}',
    notes               text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- assessments
-- An "assessment" row is one child's challenge run (the spec's AssessmentAttempt).
CREATE TABLE assessments (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id      uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    subject_id    int  NOT NULL REFERENCES subjects (id),
    grade_id      int  NOT NULL REFERENCES grades (id),
    status        text NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress','completed','abandoned')),
    planned_count int  NOT NULL,
    started_at    timestamptz NOT NULL DEFAULT now(),
    completed_at  timestamptz,
    score         int,
    accuracy      numeric(5,2),
    stars         int,
    xp_earned     int
);
CREATE INDEX assessments_child_idx ON assessments (child_id, started_at DESC);

CREATE TABLE assessment_items (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id       uuid NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    position            int  NOT NULL,
    question_id         uuid NOT NULL REFERENCES questions (id),
    question_version_id uuid NOT NULL REFERENCES question_versions (id),
    skill_id            int  NOT NULL REFERENCES skills (id),
    difficulty          int  NOT NULL,
    hint_shown          boolean NOT NULL DEFAULT false,
    attempts            int  NOT NULL DEFAULT 0,
    final_correct       boolean,
    first_try_correct   boolean,
    finalized_at        timestamptz,
    UNIQUE (assessment_id, position)
);

-- Raw learning evidence (§42–43). Never updated; deleted only with the child (erasure).
CREATE TABLE question_attempts (
    id                  bigserial PRIMARY KEY,
    assessment_item_id  uuid NOT NULL REFERENCES assessment_items (id) ON DELETE CASCADE,
    child_id            uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    question_id         uuid NOT NULL REFERENCES questions (id),
    question_version_id uuid NOT NULL REFERENCES question_versions (id),
    skill_id            int  NOT NULL REFERENCES skills (id),
    difficulty          int  NOT NULL,
    attempt_no          int  NOT NULL,
    response            jsonb NOT NULL,
    is_correct          boolean NOT NULL,
    response_time_ms    int  NOT NULL DEFAULT 0,
    hint_used           boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX question_attempts_child_skill_idx ON question_attempts (child_id, skill_id, created_at DESC);

CREATE FUNCTION forbid_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% rows are immutable', TG_TABLE_NAME;
END $$;
CREATE TRIGGER question_attempts_immutable BEFORE UPDATE ON question_attempts
    FOR EACH ROW EXECUTE FUNCTION forbid_update();

-- ---------------------------------------------------------------- progress (derived, recomputable)
CREATE TABLE skill_progress (
    child_id        uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    skill_id        int  NOT NULL REFERENCES skills (id),
    mastery         numeric(5,2) NOT NULL,
    evidence_count  int  NOT NULL,
    correct_count   int  NOT NULL,
    last_attempt_at timestamptz,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (child_id, skill_id)
);

CREATE TABLE daily_progress (
    child_id             uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    day                  date NOT NULL,
    challenges_completed int  NOT NULL DEFAULT 0,
    questions_answered   int  NOT NULL DEFAULT 0,
    correct_answers      int  NOT NULL DEFAULT 0,
    xp                   int  NOT NULL DEFAULT 0,
    PRIMARY KEY (child_id, day)
);

-- ---------------------------------------------------------------- rewards
CREATE TABLE badges (
    id          serial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL,
    icon        text NOT NULL
);

CREATE TABLE rewards (
    id          serial PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    kind        text NOT NULL CHECK (kind IN ('avatar','sticker','scene')),
    name        text NOT NULL,
    icon        text NOT NULL,
    xp_required int  NOT NULL DEFAULT 0
);

CREATE TABLE child_inventory (
    child_id    uuid NOT NULL REFERENCES children (id) ON DELETE CASCADE,
    item_kind   text NOT NULL CHECK (item_kind IN ('badge','reward')),
    item_code   text NOT NULL,
    acquired_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (child_id, item_kind, item_code)
);

-- ---------------------------------------------------------------- subscriptions & payments
CREATE TABLE subscriptions (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              uuid NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    plan                 text NOT NULL DEFAULT 'monthly',
    status               text NOT NULL CHECK (status IN ('trial','active','past_due','cancelled','expired')),
    current_period_start timestamptz NOT NULL,
    current_period_end   timestamptz NOT NULL,
    cancel_at_period_end boolean NOT NULL DEFAULT false,
    provider             text NOT NULL,
    provider_ref         text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE payment_checkouts (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    plan         text NOT NULL,
    amount       bigint NOT NULL,
    currency     text NOT NULL,
    provider     text NOT NULL,
    provider_ref text NOT NULL UNIQUE,
    status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','failed','expired')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    paid_at      timestamptz
);

-- Webhook inbox: UNIQUE(provider, event_id) makes processing idempotent.
CREATE TABLE payment_events (
    id           bigserial PRIMARY KEY,
    provider     text NOT NULL,
    event_id     text NOT NULL,
    event_type   text NOT NULL,
    payload      jsonb NOT NULL,
    processed_at timestamptz,
    error        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, event_id)
);

-- ---------------------------------------------------------------- audit & analytics
CREATE TABLE audit_logs (
    id          bigserial PRIMARY KEY,
    actor_id    uuid REFERENCES users (id) ON DELETE SET NULL,
    action      text NOT NULL,
    entity_type text NOT NULL,
    entity_id   text NOT NULL,
    data        jsonb NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id);

-- No child PII in analytics: child_id is an opaque uuid only.
CREATE TABLE analytics_events (
    id         bigserial PRIMARY KEY,
    name       text NOT NULL,
    user_id    uuid REFERENCES users (id) ON DELETE SET NULL,
    child_id   uuid REFERENCES children (id) ON DELETE SET NULL,
    props      jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX analytics_events_name_idx ON analytics_events (name, created_at);
