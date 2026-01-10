-- Manual migration: align discovery_sources/jobs with existing schema.

ALTER TABLE discovery_sources
  ADD COLUMN IF NOT EXISTS provider VARCHAR,
  ADD COLUMN IF NOT EXISTS tenant_id BIGINT;

CREATE INDEX IF NOT EXISTS discovery_sources_tenant_id
  ON discovery_sources (tenant_id);

ALTER TABLE discovery_jobs
  ALTER COLUMN source_id TYPE VARCHAR USING source_id::text;

DROP INDEX IF EXISTS discovery_jobs_source_id;
CREATE INDEX IF NOT EXISTS discovery_jobs_source_id
  ON discovery_jobs (source_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'discovery_jobs_source_id_fkey'
  ) THEN
    ALTER TABLE discovery_jobs
      ADD CONSTRAINT discovery_jobs_source_id_fkey
      FOREIGN KEY (source_id) REFERENCES discovery_sources(id) ON DELETE CASCADE;
  END IF;
END $$;
