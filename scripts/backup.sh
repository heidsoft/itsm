#!/bin/bash
# backup.sh - PostgreSQL backup script for ITSM
# Supports full backups and incremental (WAL-based) backups
# Usage: ./backup.sh [full|wal|list|verify]

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/itsm}"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGDATABASE="${PGDATABASE:-itsm}"
PGUSER="${PGUSER:-postgres}"
# WAL 归档目录默认跟随 BACKUP_DIR，保证在 CI/非 postgres 用户环境下也可写
WAL_DIR="${WAL_DIR:-${BACKUP_DIR}/wal}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_err() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Ensure backup directory exists
ensure_backup_dir() {
    mkdir -p "${BACKUP_DIR}"
    mkdir -p "${WAL_DIR}"
}

# Perform full backup using pg_dump
full_backup() {
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/itsm_full_${timestamp}.dump"

    log_info "Starting full backup to ${backup_file}"

    # Create backup metadata
    local metadata_file="${BACKUP_DIR}/itsm_full_${timestamp}.meta"
    cat > "${metadata_file}" << EOF
{
  "type": "full",
  "timestamp": "${timestamp}",
  "database": "${PGDATABASE}",
  "pg_version": "$(psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" -t -c 'SELECT version()' 2>/dev/null | tr -d '\n')",
  "backup_file": "${backup_file}",
  "started_at": "$(date -Iseconds)"
}
EOF

    # Perform pg_dump with compression
    pg_dump -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
        -Fc \
        -c \
        --if-exists \
        -f "${backup_file}" 2>&1

    local exit_code=$?
    if [ ${exit_code} -eq 0 ]; then
        # Update metadata with completion info
        echo ", \"completed_at\": \"$(date -Iseconds)\", \"status\": \"success\" }" >> "${metadata_file}"
        log_info "Full backup completed successfully: ${backup_file}"
    else
        echo ", \"completed_at\": \"$(date -Iseconds)\", \"status\": \"failed\" }" >> "${metadata_file}"
        log_err "Full backup failed with exit code ${exit_code}"
        return ${exit_code}
    fi
}

# Perform WAL backup (incremental)
wal_backup() {
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local wal_file="${WAL_DIR}/itsm_wal_${timestamp}"

    log_info "Starting WAL backup"

    # Archive the current WAL
    if command -v pg_switch_wal &> /dev/null; then
        # PostgreSQL 13+
        pg_switch_wal -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" 2>/dev/null || true
    fi

    # Copy WAL files
    local wal_count=0
    for wal in "${WAL_DIR}"/*; do
        if [ -f "${wal}" ] && [[ ! "${wal}" =~ .*partial$ ]]; then
            cp "${wal}" "${BACKUP_DIR}/"
            wal_count=$((wal_count + 1))
        fi
    done

    log_info "WAL backup completed: ${wal_count} files"
}

# List available backups
list_backups() {
    log_info "Available backups in ${BACKUP_DIR}:"
    echo ""
    ls -lh "${BACKUP_DIR}"/itsm_*.dump "${BACKUP_DIR}"/itsm_*.meta 2>/dev/null | head -20 || echo "No backups found"
    echo ""
    log_info "WAL archives:"
    ls -lh "${BACKUP_DIR}"/itsm_wal_* 2>/dev/null | head -10 || echo "No WAL archives found"
}

# Verify backup integrity
verify_backup() {
    local backup_file="${1:-$(ls -t ${BACKUP_DIR}/itsm_full_*.dump 2>/dev/null | head -1)}"

    if [ -z "${backup_file}" ] || [ ! -f "${backup_file}" ]; then
        log_err "Backup file not found: ${backup_file}"
        return 1
    fi

    log_info "Verifying backup: ${backup_file}"

    # Test restore using pg_restore --list
    if pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
        --list "${backup_file}" &>/dev/null; then
        log_info "Backup verification passed"
    else
        log_err "Backup verification failed - backup file may be corrupted"
        return 1
    fi
}

# Restore from backup
restore_backup() {
    local backup_file="${1:-$(ls -t ${BACKUP_DIR}/itsm_full_*.dump 2>/dev/null | head -1)}"

    if [ -z "${backup_file}" ] || [ ! -f "${backup_file}" ]; then
        log_err "Backup file not found: ${backup_file}"
        echo "Usage: $0 restore <backup_file>"
        return 1
    fi

    log_warn "This will overwrite the current database: ${PGDATABASE}"
    log_warn "Press Ctrl+C to cancel, or wait 10 seconds to continue..."
    sleep 10

    log_info "Restoring from ${backup_file}"

    # Terminate existing connections
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${PGDATABASE}' AND pid <> pg_backend_pid();" 2>/dev/null || true

    # Drop and recreate database
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "DROP DATABASE IF EXISTS ${PGDATABASE};" 2>/dev/null || true
    psql -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d postgres \
        -c "CREATE DATABASE ${PGDATABASE};" 2>/dev/null || true

    # Restore
    pg_restore -h "${PGHOST}" -p "${PGPORT}" -U "${PGUSER}" -d "${PGDATABASE}" \
        --clean --if-exists \
        --no-owner \
        --no-acl \
        "${backup_file}"

    log_info "Restore completed successfully"
}

# Show usage
usage() {
    cat << EOF
ITSM PostgreSQL Backup Script

Usage: $0 <command> [options]

Commands:
    full    - Create a full pg_dump backup (compressed, Fc format)
    wal     - Archive WAL files for incremental backup
    list    - List available backups
    verify  - Verify backup integrity (optionally specify backup file)
    restore - Restore from a backup (requires backup file path)
    help    - Show this help message

Environment Variables:
    BACKUP_DIR  - Backup directory (default: /var/backups/itsm)
    WAL_DIR     - WAL archive directory (default: \$BACKUP_DIR/wal)
    PGHOST      - PostgreSQL host (default: localhost)
    PGPORT      - PostgreSQL port (default: 5432)
    PGDATABASE  - Database name (default: itsm)
    PGUSER      - PostgreSQL user (default: postgres)

Examples:
    $0 full              # Create full backup
    $0 wal               # Archive WAL files
    $0 list              # List all backups
    $0 verify            # Verify latest backup
    $0 restore backup.dump  # Restore from specific backup

Note: For PITR (Point-In-Time Recovery), configure PostgreSQL's
      archive_mode and archive_command in postgresql.conf
EOF
}

# Main
main() {
    ensure_backup_dir

    local command="${1:-help}"
    shift || true

    case "${command}" in
        full)
            full_backup "$@"
            ;;
        wal)
            wal_backup "$@"
            ;;
        list)
            list_backups "$@"
            ;;
        verify)
            verify_backup "$@"
            ;;
        restore)
            restore_backup "$@"
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
