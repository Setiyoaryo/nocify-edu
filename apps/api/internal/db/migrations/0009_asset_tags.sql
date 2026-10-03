-- M10: tag assets with the concepts they depict, so the content pipeline can
-- match a question to a licensed illustration. The tag vocabulary is open and
-- matching is read-only (`tags && $1`), so a plain text array is enough — no
-- join table, no integrity to enforce. status/ai_generated are never tags; they
-- stay query filters.
ALTER TABLE assets ADD COLUMN IF NOT EXISTS tags text[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS assets_tags_idx ON assets USING gin (tags);
