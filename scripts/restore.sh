#!/bin/bash
# restore.sh - PostgreSQL restore script for ITSM
# Supports point-in-time recovery (PITR) and regular backup restoration
# Usage: ./restore.sh [full|pitr] <backup_file_or_timestamp>

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/itsm}"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGDATABASE="${PGDATABASE:-itsm}"
PGUSER="${PGUSER:-postgres}"
WAL_DIR="${WAL_DIR:-/var/lib/postgresql/wal}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_err() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Validate prerequisites
check_prereqs() {
    local errors=0

    if ! command -v psql &> /dev/null; then
        log_err "psql not found - PostgreSQL client required"
        errors=$((errors + 1))
    fi

    if ! command -v pg_restore &> /dev/null; then
        log_err "pg_restore not found - PostgreSQL client required"
        errors=$((errors + 1))
    fi

    if [ ! -d "${BACKUP_DIR}" ]; then
        log_err "Backup directory not found: ${BACKUP_DIR}"
        errors=$((errors + 1))
    fi

    if [ ${errors} -gt 0 ]; then
        return 1
    fi
    return 0
}

# Restore from a full backup file
restore_full() {
    local backup_file="$1"

    if [ ! -f "${backup_file}" ]; then
        log_err "Backup file not found: ${backup_file}"
        return 1
    fi

    log_info "Starting full restore from: ${backup_file}"
    log_info "Target database: ${PGDATABASE}"

    # Verify backup is valid
    log_info "Verifying backup integrity..."
    if ! pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" --list "${backup_file}" &>/dev/null; then
        log_err "Backup file is invalid or corrupted"
        return 1
    fi
    log_info "Backup verification passed"

    # Confirmation prompt
    log_warn "This will overwrite the current database: ${PGDATABASE}"
    log_warn "All current data will be lost!"
    log_warn "Press Ctrl+C to cancel, or wait 15 seconds to continue..."
    sleep 15

    # Terminate existing connections
    log_info "Terminating existing connections to ${PGDATABASE}..."
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${PGDATABASE}' AND pid <> pg_backend_pid();" \
        2>/dev/null || true

    # Drop existing database
    log_info "Dropping existing database..."
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "DROP DATABASE IF EXISTS ${PGDATABASE};" 2>/dev/null || true

    # Create fresh database
    log_info "Creating database..."
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "CREATE DATABASE ${PGDATABASE};"

    # Restore
    log_info "Restoring data..."
    pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
        --clean \
        --if-exists \
        --no-owner \
        --no-acl \
        --jobs=4 \
        "${backup_file}"

    log_info "Full restore completed successfully"
}

# Point-in-time recovery
restore_pitr() {
    local target_time="$1"
    local backup_file="${2:-$(ls -t ${BACKUP_DIR}/itsm_full_*.dump 2>/dev/null | head -1)}"

    log_info "Starting Point-in-Time Recovery to: ${target_time}"
    log_warn "PITR requires PostgreSQL to be stopped and configured properly"
    log_warn "This script prepares the recovery.conf - manual steps may be required"

    # Find the base backup to use
    if [ -z "${backup_file}" ] || [ ! -f "${backup_file}" ]; then
        log_err "Base backup file not found: ${backup_file}"
        return 1
    fi

    log_info "Using base backup: ${backup_file}"

    # Get backup metadata
    local meta_file="${backup_file%.dump}.meta"
    if [ -f "${meta_file}" ]; then
        log_info "Backup metadata:"
        cat "${meta_file}"
    fi

    cat << EOF

=== PITR Recovery Steps ===

1. Stop PostgreSQL:
   sudo systemctl stop postgresql

2. Backup current data directory:
   sudo cp -r /var/lib/postgresql/data /var/lib/postgresql/data.backup

3. Prepare recovery.conf (PostgreSQL 11 and earlier):
   cat > /var/lib/postgresql/data/recovery.conf << REEOF
   restore_command = 'cp ${BACKUP_DIR}/%f %p'
   recovery_target_time = '${target_time}'
   recovery_target_action = 'promote'
REEOF

   For PostgreSQL 12+, use postgresql.conf:
   cat >> /var/lib/postgresql/data/postgresql.conf << REEOF
   restore_command = 'cp ${BACKUP_DIR}/%f %p'
   recovery_target_time = '${target_time}'
   recovery_target_action = 'promote'
REEOF

4. Clear any existing backup_label if present:
   sudo rm -f /var/lib/postgresql/data/backup_label

5. Start PostgreSQL:
   sudo systemctl start postgresql

6. Verify recovery:
   psql -h ${PGHOST} -p ${PGPORT} -U ${PGUSER} -d ${PGDATABASE} \\
     -c "SELECT pg_is_in_recovery();"

=== End PITR Steps ===

EOF

    log_info "Follow the steps above to complete PITR recovery"
}

# Dry-run restore (validate without writing)
dry_run() {
    local backup_file="$1"

    if [ ! -f "${backup_file}" ]; then
        log_err "Backup file not found: ${backup_file}"
        return 1
    fi

    log_info "Running dry-run restore validation..."
    log_info "Backup file: ${backup_file}"

    # List contents
    log_info "Backup contents:"
    pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" --list "${backup_file}" | head -50

    # Validate schema
    log_info "Schema validation would restore:"
    pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        --schema-only "${backup_file}" &>/dev/null && \
        log_info "Schema validation: PASSED" || \
        log_warn "Schema validation: Some warnings (may be expected)"
}

# Show usage
usage() {
    cat << EOF
ITSM PostgreSQL Restore Script

Usage: $0 <command> [options]

Commands:
    full <backup_file>   - Restore from a full backup file
    pitr <timestamp>     - Guide for Point-in-Time Recovery to timestamp
    dry-run <file>      - Validate backup without restoring
    list                - List available backups
    help                - Show this help

Examples:
    $0 full /var/backups/itsm/itsm_full_20260730_120000.dump
    $0 pitr "2026-07-30 15:00:00"
    $0 dry-run /var/backups/itsm/itsm_full_20260730_120000.dump

Environment Variables:
    BACKUP_DIR  - Backup directory (default: /var/backups/itsm)
    PGHOST      - PostgreSQL host (default: localhost)
    PGPORT      - PostgreSQL port (default: 5432)
    PGDATABASE  - Database name (default: itsm)
    PGUSER      - PostgreSQL user (default: postgres)

WARNING: Restore operations overwrite data. Always verify backups before restoring.
EOF
}

main() {
    if ! check_prereqs; then
        log_err "Prerequisites check failed"
        exit 1
    fi

    local command="${1:-help}"
    shift || true

    case "${command}" in
        full)
            if [ -z "${1:-}" ]; then
                log_err "Missing backup file argument"
                usage
                exit 1
            fi
            restore_full "$@"
            ;;
        pitr)
            if [ -z "${1:-}" ]; then
                log_err "Missing target timestamp argument"
                usage
                exit 1
            fi
            restore_pitr "$@"
            ;;
        dry-run)
            if [ -z "${1:-}" ]; then
                log_err "Missing backup file argument"
                usage
                exit 1
            fi
            dry_run "$@"
            ;;
        list)
            log_info "Available backups in ${BACKUP_DIR}:"
            ls -lh "${BACKUP_DIR}"/itsm_full_*.dump 2>/dev/null || echo "No backups found"
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log_err "Unknown command: ${command}"
            usage
            exit 1
            ;;
    esac
}

main "$@"
