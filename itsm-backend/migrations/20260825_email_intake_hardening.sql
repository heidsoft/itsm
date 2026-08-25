-- Email intake concurrency hardening. Run after 20260824_email_intake_poc.sql.
-- Preflight fails with actionable diagnostics instead of failing halfway
-- through index/constraint creation. Install btree_gist as DBA before running.
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gist') THEN
    RAISE EXCEPTION 'btree_gist is required; install it as a database administrator before this migration';
  END IF;
  IF EXISTS (SELECT 1 FROM connector_configs WHERE name='email' AND enabled=TRUE GROUP BY tenant_id HAVING COUNT(*) > 1) THEN
    RAISE EXCEPTION 'multiple enabled email connectors exist for one or more tenants; disable duplicates before migration';
  END IF;
  IF EXISTS (SELECT 1 FROM on_call_shifts a JOIN on_call_shifts b ON a.id < b.id AND a.tenant_id=b.tenant_id AND a.schedule_id=b.schedule_id AND tstzrange(a.start_at,a.end_at,'[)') && tstzrange(b.start_at,b.end_at,'[)')) THEN
    RAISE EXCEPTION 'overlapping on-call shifts exist; resolve overlaps before migration';
  END IF;
END $$;

DO $$ BEGIN
  ALTER TABLE on_call_shifts ADD CONSTRAINT on_call_shifts_no_overlap
    EXCLUDE USING gist (
      tenant_id WITH =,
      schedule_id WITH =,
      tstzrange(start_at, end_at, '[)') WITH &&
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS connector_configs_one_enabled_email_per_tenant_uq
  ON connector_configs (tenant_id)
  WHERE name = 'email' AND enabled = TRUE;
