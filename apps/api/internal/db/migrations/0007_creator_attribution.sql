-- M8: creator / revenue attribution readiness (§39).
--
-- Records *who* created content and *how it performs*, so revenue share can be
-- built later without another migration. M8 computes no money: it stores the
-- inputs (creator, share %, usage/completion/quality) only.
--
-- Creator identity is deliberately decoupled from `users`. `questions.created_by`
-- conflates the author with whoever pressed publish (the pipeline publisher sets
-- it to the approving reviewer), and external bank-soal authors have no login.
-- A `creators` row may optionally link to a user account.
--
-- All changes are additive: nullable columns and new tables only, no rewrite of
-- a populated table, so this is safe on a running product.

CREATE TABLE creators (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code         text NOT NULL UNIQUE,
    display_name text NOT NULL,
    -- 'system' = the pipeline itself; 'staff' = an internal author (often linked
    -- to a user); 'external' = a third-party bank-soal author.
    kind         text NOT NULL DEFAULT 'staff'
                 CHECK (kind IN ('system', 'staff', 'external')),
    -- Optional link to a login. SET NULL keeps the attribution if the account goes.
    user_id      uuid REFERENCES users (id) ON DELETE SET NULL,
    -- Revenue-share readiness (§39): the agreed share, in percent. No payout math.
    share_pct    numeric(5,2) NOT NULL DEFAULT 0 CHECK (share_pct BETWEEN 0 AND 100),
    status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
-- One creator per account; creators without an account are unaffected.
CREATE UNIQUE INDEX creators_user_uniq ON creators (user_id) WHERE user_id IS NOT NULL;

-- Attribution target for a question. NULL = unattributed (seeded/CMS content).
ALTER TABLE questions ADD COLUMN creator_id uuid REFERENCES creators (id) ON DELETE SET NULL;
CREATE INDEX questions_creator_idx ON questions (creator_id);

-- Derived rollup, one row per question, rebuilt from raw assessment evidence the
-- same way `skill_progress` is rebuilt from `question_attempts` (§39). Kept
-- separate from `questions` so the running product table stays untouched and the
-- rollup can be recomputed at will.
CREATE TABLE question_attribution (
    question_id      uuid PRIMARY KEY REFERENCES questions (id) ON DELETE CASCADE,
    creator_id       uuid REFERENCES creators (id) ON DELETE SET NULL,
    usage_count      int NOT NULL DEFAULT 0 CHECK (usage_count >= 0),      -- times served
    completion_count int NOT NULL DEFAULT 0 CHECK (completion_count >= 0), -- answered to a final state
    correct_count    int NOT NULL DEFAULT 0 CHECK (correct_count >= 0),
    first_try_count  int NOT NULL DEFAULT 0 CHECK (first_try_count >= 0),
    hint_count       int NOT NULL DEFAULT 0 CHECK (hint_count >= 0),
    quality_score    numeric(5,2) NOT NULL DEFAULT 0 CHECK (quality_score BETWEEN 0 AND 100),
    published_at     timestamptz,
    computed_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX question_attribution_creator_idx ON question_attribution (creator_id);

-- Adaptation lineage (§39). `question_provenance` is already 1:1 with questions
-- and holds `source_document_id` by value, so the adaptation fields belong here.
-- These are the only mutable columns on an otherwise insert-once row.
ALTER TABLE question_provenance
    ADD COLUMN adaptation_type text NOT NULL DEFAULT 'original'
        CHECK (adaptation_type IN ('original', 'adapted', 'translated', 'remixed', 'excerpt', 'ai_assisted')),
    ADD COLUMN adapted_by  uuid REFERENCES creators (id) ON DELETE SET NULL,
    ADD COLUMN reviewed_by uuid REFERENCES users (id) ON DELETE SET NULL;

-- System creator for pipeline-published content. Fixed id so Go can reference it
-- without a lookup; `code` is the human handle.
INSERT INTO creators (id, code, display_name, kind, share_pct)
VALUES ('00000000-0000-0000-0000-00000000c0de', 'system', 'EduPlay (sistem)', 'system', 0)
ON CONFLICT (code) DO NOTHING;

-- Backfill: pipeline-published questions are the ones with a provenance row.
-- Never use `created_by` — for published candidates that is the approving
-- reviewer, not the author.
UPDATE questions q
SET creator_id = '00000000-0000-0000-0000-00000000c0de'
FROM question_provenance p
WHERE p.question_id = q.id AND q.creator_id IS NULL;
