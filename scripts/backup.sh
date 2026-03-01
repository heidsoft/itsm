#!/bin/bash

# ITSM Database Backup Script
# Usage: ./backup.sh [backup_dir]
# Cron example: 0 2 * * * /path/to/itsm/scripts/backup.sh

set -e

# Configuration
BACKUP_DIR="${1:-/var/backups/itsm}"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="${DB_NAME:-itsm_cmdb}"
DB_USER="${DB_USER:-postgres}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# S3 backup configuration (optional)
S3_BUCKET="${S3_BUCKET:-}"
S3_PREFIX="${S3_PREFIX:-itsm-backup}"

# Create backup directory
mkdir -p "$BACKUP_DIR"

echo "=== ITSM Backup Started at $(date) ==="

# PostgreSQL backup
backup_postgres() {
    local backup_file="$BACKUP_DIR/postgres_${DATE}.sql.gz"

    echo "Backing up PostgreSQL database: $DB_NAME"

    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" | gzip > "$backup_file"

    if [ -f "$backup_file" ]; then
        local size=$(du -h "$backup_file" | cut -f1)
        echo "PostgreSQL backup completed: $backup_file ($size)"

        # Upload to S3 if configured
        if [ -n "$S3_BUCKET" ]; then
            echo "Uploading to S3: s3://$S3_BUCKET/$S3_PREFIX/postgres_${DATE}.sql.gz"
            aws s3 cp "$backup_file" "s3://$S3_BUCKET/$S3_PREFIX/postgres_${DATE}.sql.gz"
        fi
    else
        echo "ERROR: PostgreSQL backup failed"
        return 1
    fi
}

# Redis backup
backup_redis() {
    local backup_file="$BACKUP_DIR/redis_${DATE}.rdb"

    echo "Backing up Redis data"

    # Redis RDB backup
    redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" SAVE

    # Copy RDB file
    local redis_rdb_path="${REDIS_DATA_DIR:-/var/lib/redis}/dump.rdb"
    if [ -f "$redis_rdb_path" ]; then
        cp "$redis_rdb_path" "$backup_file"
        gzip "$backup_file"

        if [ -f "${backup_file}.gz" ]; then
            local size=$(du -h "${backup_file}.gz" | cut -f1)
            echo "Redis backup completed: ${backup_file}.gz ($size)"

            if [ -n "$S3_BUCKET" ]; then
                aws s3 cp "${backup_file}.gz" "s3://$S3_BUCKET/$S3_PREFIX/redis_${DATE}.rdb.gz"
            fi
        fi
    else
        echo "WARNING: Redis RDB file not found, skipping Redis backup"
    fi
}

# Configuration files backup
backup_config() {
    local backup_file="$BACKUP_DIR/config_${DATE}.tar.gz"

    echo "Backing up configuration files"

    local config_files=(
        "config.yaml"
        ".env"
        "nginx/"
    )

    tar -czf "$backup_file" "${config_files[@]}" 2>/dev/null || true

    if [ -f "$backup_file" ]; then
        local size=$(du -h "$backup_file" | cut -f1)
        echo "Configuration backup completed: $backup_file ($size)"

        if [ -n "$S3_BUCKET" ]; then
            aws s3 cp "$backup_file" "s3://$S3_BUCKET/$S3_PREFIX/config_${DATE}.tar.gz"
        fi
    fi
}

# Cleanup old backups
cleanup_old_backups() {
    echo "Cleaning up backups older than $RETENTION_DAYS days"

    # Local cleanup
    find "$BACKUP_DIR" -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "*.rdb.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true

    # S3 cleanup (if configured)
    if [ -n "$S3_BUCKET" ]; then
        echo "Cleaning up S3 backups older than $RETENTION_DAYS days"
        aws s3 ls "s3://$S3_BUCKET/$S3_PREFIX/" | while read -r line; do
            local file_date=$(echo "$line" | awk '{print $1}')
            local file_age=$(($(date +%s) - $(date -d "$file_date" +%s) / 86400))
            if [ "$file_age" -gt "$RETENTION_DAYS" ]; then
                local filename=$(echo "$line" | awk '{print $4}')
                aws s3 rm "s3://$S3_BUCKET/$S3_PREFIX/$filename"
            fi
        done
    fi
}

# Verify backup integrity
verify_backup() {
    local backup_file="$1"
    local ext="${backup_file##*.}"

    echo "Verifying backup: $backup_file"

    case "$ext" in
        gz)
            gzip -t "$backup_file" && echo "Backup verified successfully" || return 1
            ;;
        sql)
            # Check SQL file header
            head -c 100 "$backup_file" | grep -q "PostgreSQL" && echo "Backup verified successfully" || return 1
            ;;
        *)
            echo "Unknown backup format"
            ;;
    esac
}

# Main execution
main() {
    # Check prerequisites
    if ! command -v pg_dump &> /dev/null; then
        echo "ERROR: pg_dump not found. Please install PostgreSQL client."
        exit 1
    fi

    # Run backups
    backup_postgres
    backup_redis || echo "WARNING: Redis backup failed, continuing..."
    backup_config || echo "WARNING: Config backup failed, continuing..."

    # Cleanup
    cleanup_old_backups

    echo "=== ITSM Backup Completed at $(date) ==="

    # Send notification (optional - configure webhook)
    if [ -n "$BACKUP_WEBHOOK_URL" ]; then
        curl -X POST "$BACKUP_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "{\"status\": \"success\", \"date\": \"$DATE\", \"backup_dir\": \"$BACKUP_DIR\"}" \
            2>/dev/null || true
    fi
}

# Run main function
main
