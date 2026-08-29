-- Roll back only PR3 contract constraints. Expand columns intentionally remain
-- available so the previous application version can continue reading old fields.

DROP INDEX IF EXISTS discovery_results_job_identity_key;
DROP INDEX IF EXISTS discovery_jobs_tenant_idempotency_key;
DROP INDEX IF EXISTS cloud_resources_tenant_identity_key;
DROP INDEX IF EXISTS cloud_resources_account_resource_lookup_idx;
-- Do not recreate the superseded unique index automatically: canonical data may
-- legitimately contain identities that collide under the old reduced key.
CREATE INDEX IF NOT EXISTS cloud_resources_account_resource_lookup_idx
    ON cloud_resources (cloud_account_id, resource_id);
