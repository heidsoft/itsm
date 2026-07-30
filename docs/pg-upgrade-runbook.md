# PostgreSQL Upgrade Runbook

## Overview

This document records the PostgreSQL upgrade procedure from v16 to v17, including pre-upgrade checks, migration steps, and validation.

**Current Production Version:** PostgreSQL 17.10 (Debian 17.10-1.pgdg12+1)
**Previous Version:** PostgreSQL 16.x
**Upgrade Date:** 2026-07 (documented post-upgrade)

---

## Upgrade Overview

PostgreSQL major version upgrades require careful planning:

1. **Pre-upgrade validation** - Verify current state and create backup
2. **Schema compatibility check** - Detect incompatibilities
3. **Binary upgrade** - Replace PostgreSQL binaries (pg_upgrade)
4. **Post-upgrade validation** - Verify data integrity and performance

---

## Pre-Upgrade Checklist

### 1. Backup Database

```bash
# Full backup before upgrade
./scripts/backup.sh full

# Verify backup
./scripts/backup.sh verify /var/backups/itsm/itsm_full_YYYYMMDD_HHMMSS.dump
```

### 2. Check Current Version

```sql
-- Login to PostgreSQL
psql -U postgres -d itsm -c "SELECT version();"
-- Expected: PostgreSQL 16.x

-- Check active connections
psql -U postgres -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'itsm';"
```

### 3. Verify No Replication Lag

```sql
-- Check replication status (if applicable)
SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn
FROM pg_stat_replication;
```

### 4. Install PostgreSQL 17

```bash
# Add PostgreSQL GPG key
curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg

# Add repository (Debian 12)
echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] https://apt.postgresql.org/pub/repos/apt bookworm-pgdg main" \
    | sudo tee /etc/apt/sources.list.d/pgdg.list

# Update and install PostgreSQL 17
sudo apt-get update
sudo apt-get install -y postgresql-17
```

---

## Upgrade Procedure (pg_upgrade)

### 1. Stop PostgreSQL

```bash
sudo systemctl stop postgresql
sudo systemctl stop postgresql@16-main  # If using custom cluster name
```

### 2. Run pg_upgrade

```bash
# Create upgrade directory
sudo -u postgres mkdir -p /var/lib/postgresql/17_upgrade

# Run pg_upgrade (binary upgrade mode)
sudo -u postgres pg_upgrade \
    --old-datadir=/var/lib/postgresql/16/main \
    --new-datadir=/var/lib/postgresql/17/main \
    --old-bindir=/usr/lib/postgresql/16/bin \
    --new-bindir=/usr/lib/postgresql/17/bin \
    --old-options="-c config_file=/etc/postgresql/16/main/postgresql.conf" \
    --new-options="-c config_file=/etc/postgresql/17/main/postgresql.conf" \
    --link \
    --jobs=4
```

Options explained:
- `--link`: Creates hard links instead of copying files (faster, uses less space)
- `--jobs=4`: Uses 4 parallel jobs for faster upgrade

### 3. Start PostgreSQL 17

```bash
sudo systemctl start postgresql@17-main
# Or on systems with single cluster:
sudo systemctl start postgresql
```

### 4. Run Vacuum/Analyze

```bash
# As postgres user, run vacuumdb to update statistics
sudo -u postgres vacuumdb --all --analyze-in-stages

# Check for bloated tables
sudo -u postgres vacuumdb --all --analyze 2>&1 | tail -20
```

---

## Post-Upgrade Validation

### 1. Verify Version

```sql
psql -U postgres -d itsm -c "SELECT version();"
-- Should show: PostgreSQL 17.10
```

### 2. Check Data Integrity

```sql
-- Check for corrupt tables
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC LIMIT 20;

-- Verify row counts match expected values
-- (Compare with pre-upgrade counts if available)
```

### 3. Test Application Connectivity

```bash
# Test backend connectivity
curl -s http://localhost:8090/api/v1/health | jq .

# Test database queries
psql -U postgres -d itsm -c "SELECT COUNT(*) FROM users;"
psql -U postgres -d itsm -c "SELECT COUNT(*) FROM tenant;"
```

### 4. Check for Deprecation Warnings

```bash
# Check PostgreSQL logs for warnings
sudo journalctl -u postgresql@17-main --since "1 hour ago" | grep -i warning
sudo tail -100 /var/log/postgresql/postgresql-17-main.log | grep -i warning
```

---

## Rollback Procedure

If the upgrade fails or critical issues are found:

### 1. Stop PostgreSQL 17

```bash
sudo systemctl stop postgresql@17-main
```

### 2. Restore from Backup

```bash
# Use the restore script
./scripts/restore.sh full /var/backups/itsm/itsm_full_YYYYMMDD_HHMMSS.dump
```

### 3. Reinstall PostgreSQL 16

```bash
sudo apt-get install -y postgresql-16
sudo systemctl start postgresql@16-main
```

---

## Known Issues / Notes

### 1. RLS Behavior Changes
PostgreSQL 17 maintains backward compatibility with RLS policies. No action required.

### 2. Monitoring Adjustments
Update any monitoring queries that reference deprecated system views if applicable.

### 3. Connection Pooling
If using PgBouncer or pgpool, verify compatibility with PostgreSQL 17.

---

## CI/CD Integration

This upgrade runbook is validated by the `pg-disaster-recovery.yml` workflow which:
- Runs weekly backup/restore drills
- Validates RLS enforcement after upgrades
- Tests fencing token fault injection

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| DBA Lead | | | |
| Platform Lead | | | |
| Security | | | |
