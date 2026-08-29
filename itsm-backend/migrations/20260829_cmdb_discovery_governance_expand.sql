-- PR3 expand/backfill migration. Safe to rerun.
-- This migration deliberately does not add canonical identity uniqueness.

BEGIN;

ALTER TABLE discovery_sources
    ADD COLUMN IF NOT EXISTS cloud_account_id bigint,
    ADD COLUMN IF NOT EXISTS service_codes jsonb,
    ADD COLUMN IF NOT EXISTS regions jsonb,
    ADD COLUMN IF NOT EXISTS schedule varchar,
    ADD COLUMN IF NOT EXISTS reconcile_policy varchar DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS stale_threshold bigint DEFAULT 3,
    ADD COLUMN IF NOT EXISTS last_success_at timestamptz;

ALTER TABLE discovery_jobs
    ADD COLUMN IF NOT EXISTS operation varchar DEFAULT 'full_discovery',
    ADD COLUMN IF NOT EXISTS idempotency_key varchar,
    ADD COLUMN IF NOT EXISTS request_fingerprint varchar,
    ADD COLUMN IF NOT EXISTS source_snapshot jsonb,
    ADD COLUMN IF NOT EXISTS scope_snapshot jsonb,
    ADD COLUMN IF NOT EXISTS completed_scopes jsonb,
    ADD COLUMN IF NOT EXISTS failed_scopes jsonb,
    ADD COLUMN IF NOT EXISTS snapshot_generation varchar,
    ADD COLUMN IF NOT EXISTS requested_by bigint,
    ADD COLUMN IF NOT EXISTS queued_at timestamptz,
    ADD COLUMN IF NOT EXISTS heartbeat_at timestamptz,
    ADD COLUMN IF NOT EXISTS lease_owner varchar,
    ADD COLUMN IF NOT EXISTS lease_expires_at timestamptz,
    ADD COLUMN IF NOT EXISTS fencing_token bigint DEFAULT 0,
    ADD COLUMN IF NOT EXISTS attempt bigint DEFAULT 0,
    ADD COLUMN IF NOT EXISTS parent_job_id bigint,
    ADD COLUMN IF NOT EXISTS max_attempts bigint DEFAULT 3,
    ADD COLUMN IF NOT EXISTS progress bigint DEFAULT 0,
    ADD COLUMN IF NOT EXISTS error_code varchar,
    ADD COLUMN IF NOT EXISTS error_message varchar,
    ADD COLUMN IF NOT EXISTS cancel_requested_at timestamptz;

ALTER TABLE discovery_results
    ADD COLUMN IF NOT EXISTS resource_identity varchar,
    ADD COLUMN IF NOT EXISTS identity_version bigint DEFAULT 1,
    ADD COLUMN IF NOT EXISTS resource_snapshot jsonb,
    ADD COLUMN IF NOT EXISTS before_hash varchar,
    ADD COLUMN IF NOT EXISTS after_hash varchar,
    ADD COLUMN IF NOT EXISTS error_code varchar,
    ADD COLUMN IF NOT EXISTS error_message varchar;

ALTER TABLE cloud_resources
    ADD COLUMN IF NOT EXISTS identity_version bigint DEFAULT 1,
    ADD COLUMN IF NOT EXISTS provider varchar,
    ADD COLUMN IF NOT EXISTS partition varchar DEFAULT 'public',
    ADD COLUMN IF NOT EXISTS canonical_account_id varchar,
    ADD COLUMN IF NOT EXISTS resource_scope varchar DEFAULT 'regional',
    ADD COLUMN IF NOT EXISTS service_code varchar,
    ADD COLUMN IF NOT EXISTS resource_type varchar,
    ADD COLUMN IF NOT EXISTS identity_hash varchar,
    ADD COLUMN IF NOT EXISTS source_id varchar,
    ADD COLUMN IF NOT EXISTS source_fingerprint varchar,
    ADD COLUMN IF NOT EXISTS missing_count bigint DEFAULT 0;

ALTER TABLE configuration_items
    ADD COLUMN IF NOT EXISTS source_id varchar,
    ADD COLUMN IF NOT EXISTS canonical_cloud_account_id varchar,
    ADD COLUMN IF NOT EXISTS source_last_seen_at timestamptz,
    ADD COLUMN IF NOT EXISTS source_missing_count bigint DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source_fingerprint varchar;

-- Deterministic legacy backfill. Internal cloud_account_id is not part of the
-- canonical identity; the provider-native account_id survives row recreation.
UPDATE cloud_resources cr
SET provider = lower(trim(ca.provider)),
    canonical_account_id = trim(ca.account_id),
    service_code = lower(trim(cs.service_code)),
    resource_type = lower(trim(cs.resource_type_code)),
    region = lower(trim(cr.region)),
    resource_scope = CASE WHEN trim(coalesce(cr.zone, '')) <> '' THEN 'zonal'
                          WHEN trim(coalesce(cr.region, '')) <> '' THEN 'regional'
                          ELSE 'global' END,
    source_id = coalesce(nullif(cr.source_id, ''), 'legacy'),
    identity_version = 1
FROM cloud_accounts ca, cloud_services cs
WHERE cr.cloud_account_id = ca.id
  AND cr.service_id = cs.id
  AND cr.tenant_id = ca.tenant_id
  AND cr.tenant_id = cs.tenant_id
  AND (cr.identity_hash IS NULL OR cr.identity_hash = '');

UPDATE cloud_resources
SET identity_hash = encode(sha256(convert_to(concat_ws('|',
        'v1', tenant_id::text, provider, partition, canonical_account_id,
        resource_scope, coalesce(region, ''), service_code, resource_type, resource_id), 'UTF8')), 'hex')
WHERE (identity_hash IS NULL OR identity_hash = '')
  AND provider IS NOT NULL AND provider <> ''
  AND canonical_account_id IS NOT NULL AND canonical_account_id <> ''
  AND service_code IS NOT NULL AND service_code <> ''
  AND resource_type IS NOT NULL AND resource_type <> ''
  AND resource_id <> '';

UPDATE configuration_items
SET source_id = 'legacy',
    canonical_cloud_account_id = nullif(trim(cloud_account_id), ''),
    source_last_seen_at = coalesce(cloud_sync_time, last_discovered),
    source_missing_count = 0
WHERE source_id IS NULL
  AND (cloud_resource_id <> '' OR cloud_resource_ref_id IS NOT NULL);

UPDATE discovery_jobs
SET status = CASE status
    WHEN 'pending' THEN 'queued'
    WHEN 'running' THEN 'discovering'
    WHEN 'success' THEN 'succeeded'
    ELSE status
END,
queued_at = coalesce(queued_at, created_at),
operation = coalesce(nullif(operation, ''), 'full_discovery'),
max_attempts = coalesce(max_attempts, 3),
fencing_token = coalesce(fencing_token, 0),
attempt = coalesce(attempt, 0),
progress = coalesce(progress, 0);

UPDATE discovery_results
SET resource_snapshot = coalesce(resource_snapshot, '{}'::jsonb) || '{"identityStatus":"legacy"}'::jsonb
WHERE resource_identity IS NULL;

CREATE INDEX IF NOT EXISTS discovery_sources_tenant_cloud_account_idx
    ON discovery_sources (tenant_id, cloud_account_id);
CREATE INDEX IF NOT EXISTS discovery_jobs_idempotency_expand_idx
    ON discovery_jobs (tenant_id, operation, source_id, idempotency_key);
CREATE INDEX IF NOT EXISTS discovery_jobs_claim_idx
    ON discovery_jobs (status, lease_expires_at);
CREATE INDEX IF NOT EXISTS discovery_results_identity_expand_idx
    ON discovery_results (tenant_id, job_id, resource_identity);
CREATE INDEX IF NOT EXISTS cloud_resources_identity_expand_idx
    ON cloud_resources (tenant_id, identity_hash);
CREATE INDEX IF NOT EXISTS configuration_items_source_expand_idx
    ON configuration_items (tenant_id, source_id);

COMMIT;
