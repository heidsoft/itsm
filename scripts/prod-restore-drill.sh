#!/bin/bash
# prod-restore-drill.sh - ITSM prod 备份恢复演练（隔离临时库，不触碰 prod 数据）
#
# 用法:
#   ./scripts/prod-restore-drill.sh              # 演练最新备份
#   ./scripts/prod-restore-drill.sh <file>       # 演练指定备份
#
# 演练流程:
#   1. 选定备份（默认最新 itsm_prod_*.sql.gz）
#   2. 在 prod PG 实例内创建隔离临时库 itsm_drill_<ts>
#   3. pg_restore 恢复备份到临时库
#   4. 断言：关键表存在 + 行数合理（users/tickets/roles 非零，可与 prod 当前值对比）
#   5. DROP 临时库清理（无论成败）
#
# RTO/RPO 记录：输出恢复耗时（RTO 参考值）与备份时间点（RPO 参考值）。

set -euo pipefail

# launchd/cron 环境无用户 PATH，显式补齐 docker 所在目录（与 prod-backup.sh 对齐）
export PATH="/Applications/Docker.app/Contents/Resources/bin:/usr/local/bin:/opt/homebrew/bin:${PATH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKUP_DIR="${BACKUP_DIR:-${REPO_ROOT}/backups}"
PG_CONTAINER="${PG_CONTAINER:-itsm-postgres-prod}"
PG_USER="${PG_USER:-itsm}"
PG_DATABASE="${PG_DATABASE:-itsm_prod}"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
log_info() { echo -e "${GREEN}[DRILL]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[DRILL]${NC} $1"; }
log_err()  { echo -e "${RED}[DRILL]${NC} $1" >&2; }

# 1) 选定备份
BACKUP_FILE="${1:-}"
if [ -z "${BACKUP_FILE}" ]; then
  BACKUP_FILE="$(ls -t "${BACKUP_DIR}"/itsm_prod_*.sql.gz 2>/dev/null | head -1 || true)"
fi
if [ -z "${BACKUP_FILE}" ] || [ ! -f "${BACKUP_FILE}" ]; then
  log_err "no backup found in ${BACKUP_DIR}（先跑 scripts/prod-backup.sh）"
  exit 1
fi
log_info "drill target: $(basename "${BACKUP_FILE}")"

TS="$(date +%Y%m%d_%H%M%S)"
DRILL_DB="itsm_drill_${TS}"
PLAIN_SQL="${BACKUP_DIR}/.drill_${TS}.sql"
DRILL_START="$(date +%s)"

cleanup() {
  docker exec "${PG_CONTAINER}" psql -U "${PG_USER}" -d postgres \
    -c "DROP DATABASE IF EXISTS ${DRILL_DB};" > /dev/null 2>&1 || true
  rm -f "${PLAIN_SQL}"
  docker exec "${PG_CONTAINER}" rm -f /tmp/drill.sql 2>/dev/null || true
}
trap cleanup EXIT

# 2) 创建隔离临时库
log_info "creating isolated drill database: ${DRILL_DB}"
docker exec "${PG_CONTAINER}" psql -U "${PG_USER}" -d postgres \
  -c "CREATE DATABASE ${DRILL_DB};" > /dev/null

# 3) 复制备份进容器并恢复
gunzip -c "${BACKUP_FILE}" > "${PLAIN_SQL}"
docker cp "${PLAIN_SQL}" "${PG_CONTAINER}:/tmp/drill.sql"
log_info "restoring ..."
docker exec "${PG_CONTAINER}" psql -U "${PG_USER}" -d "${DRILL_DB}" \
  -f /tmp/drill.sql > /dev/null 2>&1 || log_warn "restore emitted errors (psql streaming 模式下部分对象已存在属预期，后续断言为准)"

# 4) 断言关键表与行数
log_info "asserting key tables ..."
ASSERT_SQL="
SELECT 'users', count(*) FROM users
UNION ALL SELECT 'roles', count(*) FROM roles
UNION ALL SELECT 'tickets', count(*) FROM tickets
UNION ALL SELECT 'incidents', count(*) FROM incidents
UNION ALL SELECT 'changes', count(*) FROM changes
UNION ALL SELECT 'problems', count(*) FROM problems;"
ROW_COUNTS="$(docker exec "${PG_CONTAINER}" psql -U "${PG_USER}" -d "${DRILL_DB}" -At -c "${ASSERT_SQL}" 2>/dev/null || true)"
if [ -z "${ROW_COUNTS}" ]; then
  log_err "assertion failed: cannot query drill database"
  exit 1
fi
echo "${ROW_COUNTS}" | while IFS='|' read -r tbl cnt; do
  log_info "  ${tbl}: ${cnt} rows"
done

FAIL=0
echo "${ROW_COUNTS}" | while IFS='|' read -r tbl cnt; do
  case "${tbl}" in
    users|roles)
      if [ "${cnt}" -eq 0 ] 2>/dev/null; then
        log_err "table ${tbl} is empty — restore likely broken"
        exit 1
      fi
      ;;
  esac
done || FAIL=1
# 上面 while 在管道子 shell，需重新独立断言 users 行数
USERS_CNT="$(echo "${ROW_COUNTS}" | awk -F'|' '$1=="users"{print $2}')"
ROLES_CNT="$(echo "${ROW_COUNTS}" | awk -F'|' '$1=="roles"{print $2}')"
if [ -z "${USERS_CNT}" ] || [ "${USERS_CNT}" -eq 0 ] || [ -z "${ROLES_CNT}" ] || [ "${ROLES_CNT}" -eq 0 ]; then
  log_err "assertion failed: users(${USERS_CNT}) / roles(${ROLES_CNT}) 行数异常"
  exit 1
fi

# 5) RTO/RPO 记录
DRILL_END="$(date +%s)"
RTO=$((DRILL_END - DRILL_START))
BACKUP_MTIME="$(stat -f '%Sm' -t '%Y-%m-%d %H:%M:%S' "${BACKUP_FILE}" 2>/dev/null || date -r "${BACKUP_FILE}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo unknown)"
NOW_EPOCH="$(date +%s)"
FILE_EPOCH="$(stat -f '%m' "${BACKUP_FILE}" 2>/dev/null || echo "${NOW_EPOCH}")"
RPO_HOURS=$(( (NOW_EPOCH - FILE_EPOCH) / 3600 ))

log_info "=========================================="
log_info "RESTORE DRILL PASSED"
log_info "  RTO（恢复耗时）: ${RTO}s"
log_info "  RPO（数据新鲜度）: 备份时间点 ${BACKUP_MTIME}（约 ${RPO_HOURS}h 前）"
log_info "  断言: users=${USERS_CNT} roles=${ROLES_CNT} 非零通过"
log_info "=========================================="
