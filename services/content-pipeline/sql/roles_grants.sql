-- Production hardening: least-privilege role for the content pipeline.
--
-- This is NOT applied automatically. Run it once per environment as a DB owner
-- (see docs/content-pipeline-roadmap.md, D5). Replace the password placeholder
-- before running; keep the real value in the environment, never in git.
--
-- After this, the pipeline connects with CONTENT_PIPELINE_DATABASE_URL and can
-- only write inside content_staging, and can only read the canonical taxonomy.

-- CREATE ROLE content_pipeline LOGIN PASSWORD 'REPLACE_ME';

GRANT USAGE ON SCHEMA content_staging TO content_pipeline;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA content_staging TO content_pipeline;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA content_staging TO content_pipeline;
ALTER DEFAULT PRIVILEGES IN SCHEMA content_staging
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO content_pipeline;
ALTER DEFAULT PRIVILEGES IN SCHEMA content_staging
    GRANT USAGE, SELECT ON SEQUENCES TO content_pipeline;

-- Read-only access to canonical curriculum taxonomy, so mapping can resolve
-- skills. No INSERT/UPDATE/DELETE on any canonical table.
GRANT USAGE ON SCHEMA public TO content_pipeline;
GRANT SELECT ON
    public.curriculum_versions,
    public.education_levels,
    public.grades,
    public.subjects,
    public.chapters,
    public.skills
TO content_pipeline;

-- The application role needs read access to staging for review and publish.
GRANT USAGE ON SCHEMA content_staging TO eduplay;
GRANT SELECT ON ALL TABLES IN SCHEMA content_staging TO eduplay;
ALTER DEFAULT PRIVILEGES IN SCHEMA content_staging
    GRANT SELECT ON TABLES TO eduplay;
