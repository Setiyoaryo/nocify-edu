-- M8 follow-up: index assessment evidence by question.
--
-- The creator-attribution rollup (`question_attribution`) looks assessment
-- evidence up by question: the session-complete aggregate
-- (`WHERE i.question_id = ANY($1) GROUP BY i.question_id`), the erasure prune,
-- and the full rebuild's orphan delete all filter on `assessment_items.question_id`.
-- Neither the FK nor `UNIQUE (assessment_id, position)` indexes that column, so
-- those queries sequentially scan the largest evidence table (one row per served
-- item). This is the hot path of finishing a challenge.
--
-- Additive and idempotent. Note: `db.Migrate` runs each file inside a
-- transaction, so a plain (write-locking) CREATE INDEX is the only option here.
-- On an already-large table, pre-create it out-of-band with CONCURRENTLY using
-- the same name — IF NOT EXISTS then makes this a no-op (see docs/operations.md).

CREATE INDEX IF NOT EXISTS assessment_items_question_idx ON assessment_items (question_id);
