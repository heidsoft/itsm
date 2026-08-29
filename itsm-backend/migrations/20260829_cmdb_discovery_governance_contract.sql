-- PR3 contract migration. Run only after one compatibility release and after
-- expand/backfill validation. It aborts with a conflict report; it never merges.

CREATE TABLE IF NOT EXISTS cmdb_identity_migration_conflicts (
    conflict_type varchar NOT NULL,
    tenant_id bigint NOT NULL,
    conflict_key varchar NOT NULL,
    record_ids jsonb NOT NULL,
    detected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (conflict_type, tenant_id, conflict_key)
);

DELETE FROM cmdb_identity_migration_conflicts
WHERE conflict_type IN (
    'cloud_resource_identity',
    'discovery_job_idempotency',
    'discovery_result_identity'
);

INSERT INTO cmdb_identity_migration_conflicts
    (conflict_type, tenant_id, conflict_key, record_ids)
SELECT 'cloud_resource_identity', tenant_id, identity_hash,
       jsonb_agg(id ORDER BY id)
FROM cloud_resources
WHERE identity_hash IS NOT NULL AND identity_hash <> ''
GROUP BY tenant_id, identity_hash
HAVING count(*) > 1
ON CONFLICT (conflict_type, tenant_id, conflict_key)
DO UPDATE SET record_ids = excluded.record_ids, detected_at = now();

INSERT INTO cmdb_identity_migration_conflicts
    (conflict_type, tenant_id, conflict_key, record_ids)
SELECT 'discovery_job_idempotency', tenant_id,
       concat_ws('|', operation, source_id, idempotency_key),
       jsonb_agg(id ORDER BY id)
FROM discovery_jobs
WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''
GROUP BY tenant_id, operation, source_id, idempotency_key
HAVING count(*) > 1
ON CONFLICT (conflict_type, tenant_id, conflict_key)
DO UPDATE SET record_ids = excluded.record_ids, detected_at = now();

INSERT INTO cmdb_identity_migration_conflicts
    (conflict_type, tenant_id, conflict_key, record_ids)
SELECT 'discovery_result_identity', tenant_id,
       concat_ws('|', job_id::text, resource_identity),
       jsonb_agg(id ORDER BY id)
FROM discovery_results
WHERE resource_identity IS NOT NULL AND resource_identity <> ''
GROUP BY tenant_id, job_id, resource_identity
HAVING count(*) > 1
ON CONFLICT (conflict_type, tenant_id, conflict_key)
DO UPDATE SET record_ids = excluded.record_ids, detected_at = now();

-- The report above commits under normal psql autocommit before this gate starts.
-- If the gate aborts, operators can still inspect the persisted conflict rows.
BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM cmdb_identity_migration_conflicts) THEN
        RAISE EXCEPTION
            'CMDB identity conflicts detected; inspect cmdb_identity_migration_conflicts and resolve explicitly';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS cloud_resources_tenant_identity_key
    ON cloud_resources (tenant_id, identity_hash)
    WHERE identity_hash IS NOT NULL AND identity_hash <> '';
CREATE UNIQUE INDEX IF NOT EXISTS discovery_jobs_tenant_idempotency_key
    ON discovery_jobs (tenant_id, operation, source_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
CREATE UNIQUE INDEX IF NOT EXISTS discovery_results_job_identity_key
    ON discovery_results (tenant_id, job_id, resource_identity)
    WHERE resource_identity IS NOT NULL AND resource_identity <> '';

-- Drop the superseded row-ID based uniqueness only after canonical uniqueness
-- exists. Keep a non-unique lookup index for compatibility reads.
DROP INDEX IF EXISTS cloudresource_cloud_account_id_resource_id;
CREATE INDEX IF NOT EXISTS cloud_resources_account_resource_lookup_idx
    ON cloud_resources (cloud_account_id, resource_id);

COMMIT;
