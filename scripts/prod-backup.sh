#!/bin/bash
# prod-backup.sh - ITSM prod 数据库备份（容器内 pg_dump，零宿主机依赖）
#
# 用法:
#   ./scripts/prod-backup.sh              # 手动备份（保留 KEEP_BACKUPS 份）
#   ./scripts/prod-backup.sh --verify     # 备份后立即做 restore 校验
#
# 环境变量:
#   BACKUP_DIR     备份输出目录（默认 <repo>/backups）
#   KEEP_BACKUPS   保留份数（默认 7）
#   PG_CONTAINER   postgres 容器名（默认 itsm-postgres-prod）
#   PG_USER / PG_DATABASE（默认 itsm / itsm_prod）
#
# 设计要点:
#   - pg_dump 在容器内执行（容器自带 PostgreSQL 17 客户端，宿主机无需 psql）
#   - 原子落盘：先写 .part 临时文件，校验通过后 rename 到最终名
#   - 保留策略：超出 KEEP_BACKUPS 的最旧备份自动清理
#   - 校验：gzip -t 完整性 + pg_restore --list 可读性（需容器内 pg_restore）

set -euo pipefail

# launchd/cron 环境无用户 PATH（仅 /usr/bin:/bin:/usr/sbin:/sbin），显式补齐
# docker 与常用工具所在目录，否则定时任务会因 command not found 静默失败。
export PATH="/Applications/Docker.app/Contents/Resources/bin:/usr/local/bin:/opt/homebrew/bin:${PATH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKUP_DIR="${BACKUP_DIR:-${REPO_ROOT}/backups}"
KEEP_BACKUPS="${KEEP_BACKUPS:-7}"
PG_CONTAINER="${PG_CONTAINER:-itsm-postgres-prod}"
PG_USER="${PG_USER:-itsm}"
PG_DATABASE="${PG_DATABASE:-itsm_prod}"

VERIFY_AFTER_BACKUP=false
if [ "${1:-}" = "--verify" ]; then
  VERIFY_AFTER_BACKUP=true
fi

GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
log_info() { echo -e "${GREEN}[BACKUP]${NC} $1"; }
log_err()  { echo -e "${RED}[BACKUP]${NC} $1" >&2; }

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
FINAL_FILE="${BACKUP_DIR}/itsm_prod_${TIMESTAMP}.sql.gz"
PART_FILE="${FINAL_FILE}.part"

mkdir -p "${BACKUP_DIR}"

# 1) 容器内 pg_dump | 宿主机 gzip 流式落盘（.part 原子写）
log_info "pg_dump ${PG_DATABASE} from ${PG_CONTAINER} ..."
if ! docker exec "${PG_CONTAINER}" pg_dump -U "${PG_USER}" -d "${PG_DATABASE}" --no-owner --no-privileges \
    | gzip > "${PART_FILE}"; then
  rm -f "${PART_FILE}"
  log_err "pg_dump failed"
  exit 1
fi

# 2) 完整性校验：gzip -t
if ! gzip -t "${PART_FILE}"; then
  rm -f "${PART_FILE}"
  log_err "gzip integrity check failed"
  exit 1
fi

# 3) 内容校验：dump 必须包含关键头部（PG17 的 \restrict token 在前几行，放宽窗口）
#    注意：grep -q 匹配后立即退出会向上游 gunzip/head 发 SIGPIPE(141)，在 pipefail
#    下整条管道被判非 0 → 误报 sanity 失败。故在子 shell 中临时关闭 pipefail，
#    以 grep 的退出码为准（gzip 完整性已由步骤 2 的 gzip -t 保证，无中途损坏风险）。
if ! (set +o pipefail; gunzip -c "${PART_FILE}" | head -20 | grep -q "PostgreSQL database dump"); then
  rm -f "${PART_FILE}"
  log_err "dump content sanity check failed"
  exit 1
fi

mv "${PART_FILE}" "${FINAL_FILE}"
SIZE="$(du -h "${FINAL_FILE}" | cut -f1)"
log_info "backup written: ${FINAL_FILE} (${SIZE})"

# 4) 可选：备份后校验。pg_dump 默认输出 plain SQL 格式，pg_restore 不支持（仅
#    认 custom/tar 归档），故改用「完整标记」校验：`PostgreSQL database dump
#    complete` 仅在 pg_dump 顺利完成时写入文件末尾，缺失即视为不完整。
#    真正的可用性验证（实际恢复+行数断言）由 prod-restore-drill.sh 承担。
if [ "${VERIFY_AFTER_BACKUP}" = true ]; then
  log_info "verifying dump completeness ..."
  if (set +o pipefail; gunzip -c "${FINAL_FILE}" | tail -50 | grep -q "PostgreSQL database dump complete"); then
    log_info "dump completeness check passed"
  else
    log_err "dump completeness check FAILED（缺少 complete 标记，dump 可能不完整）"
    exit 1
  fi
fi

# 5) 保留策略：只保留最近 KEEP_BACKUPS 份
LIST_OUTPUT="$(ls -t "${BACKUP_DIR}"/itsm_prod_*.sql.gz 2>/dev/null || true)"
if [ -n "${LIST_OUTPUT}" ]; then
  echo "${LIST_OUTPUT}" | tail -n +$((KEEP_BACKUPS + 1)) | while read -r old; do
    [ -f "${old}" ] && rm -f "${old}" && log_info "pruned old backup: $(basename "${old}")"
  done
fi

# 6) 结果汇总
COUNT="$(ls -1 "${BACKUP_DIR}"/itsm_prod_*.sql.gz 2>/dev/null | wc -l | tr -d ' ')"
log_info "done. backups in ${BACKUP_DIR}: ${COUNT} (keep=${KEEP_BACKUPS})"
