# CMDB discovery governance migration runbook

PR3 uses expand/backfill/contract. Do not run the contract file in the same
deployment as the expand file.

## 1. Preflight and expand

Back up the database, record table row counts, then run:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -f migrations/20260829_cmdb_discovery_governance_expand.sql
```

The expand script is idempotent. Records whose source cannot be proven are
marked `legacy`; they are not attached to a discovery source automatically.

Validate the backfill:

```sql
SELECT count(*) AS resources,
       count(identity_hash) AS identified,
       count(*) FILTER (WHERE source_id = 'legacy') AS legacy
FROM cloud_resources;

SELECT tenant_id, identity_hash, count(*), jsonb_agg(id ORDER BY id)
FROM cloud_resources
WHERE identity_hash IS NOT NULL AND identity_hash <> ''
GROUP BY tenant_id, identity_hash
HAVING count(*) > 1;
```

Compare the recorded row counts and application-level aggregate hashes before
and after expand. Exercise an application rollback while the expand columns are
present. Keep the old read path for at least one compatibility release.

## 2. Contract dry-run and conflict handling

Run the contract file in staging first. It writes current conflicts to
`cmdb_identity_migration_conflicts` before deliberately aborting when any exist:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -f migrations/20260829_cmdb_discovery_governance_contract.sql
```

Resolve every listed record through an approved data-governance decision. Never
delete, merge, or rebind a CI automatically. Rerun expand after corrections,
then rerun contract. A successful contract creates only partial unique indexes
for non-empty canonical identities and idempotency keys.

## 3. Recovery

If the contract indexes must be removed, run:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -f migrations/20260829_cmdb_discovery_governance_contract_down.sql
```

Do not drop expand columns during application rollback. Stop discovery workers
before restoring a database snapshot so stale fencing-token holders cannot
write after recovery.
