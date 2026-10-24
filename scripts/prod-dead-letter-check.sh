#!/bin/bash
# prod-dead-letter-check.sh - 死信积压阈值检查（P0-2 告警侧）
#
# 背景：2026-10-23~24 实测 prod 死信从 128 涨到 246（notification.deliver 占 71%），
# 无任何告警、静默累积。本脚本提供可定时执行的阈值检查；超阈值 exit 1。
#
# 用法:
#   ./scripts/prod-dead-letter-check.sh                  # 手动检查
#   ./scripts/prod-dead-letter-check.sh --json           # 输出 JSON（供通知渠道消费）
#
# 环境变量:
#   DEAD_LETTER_THRESHOLD  死信总量阈值（默认 50）
#   PG_CONTAINER           postgres 容器名（默认 itsm-postgres-prod）
#   ENV_FILE               env 文件（默认 <repo>/.env.prod，取 DB_USER/DB_NAME）
#   NOTIFY_WEBHOOK         可选：超阈值时 POST 告警 payload 的 webhook（如飞书机器人）
#
# launchd/cron 建议（参照 com.itsm.prod-backup.plist 的做法，每 30 分钟一次）:
#   */30 * * * * /path/to/repo/scripts/prod-dead-letter-check.sh >> /tmp/itsm-dead-letter.log 2>&1

set -euo pipefail

export PATH="/Applications/Docker.app/Contents/Resources/bin:/usr/local/bin:/opt/homebrew/bin:${PATH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

THRESHOLD="${DEAD_LETTER_THRESHOLD:-50}"
PG_CONTAINER="${PG_CONTAINER:-itsm-postgres-prod}"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env.prod}"
NOTIFY_WEBHOOK="${NOTIFY_WEBHOOK:-}"
OUTPUT_JSON=false
[ "${1:-}" = "--json" ] && OUTPUT_JSON=true

if [ ! -f "$ENV_FILE" ]; then
  echo "{\"error\":\"env file not found: $ENV_FILE\"}" >&2
  exit 2
fi
DB_USER="$(grep -E '^DB_USER=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]')"
DB_NAME="$(grep -E '^DB_NAME=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]')"
[ -z "$DB_USER" ] || [ -z "$DB_NAME" ] && { echo '{"error":"DB_USER/DB_NAME not set"}' >&2; exit 2; }

psql_exec() {
  docker exec "$PG_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -A -c "$1"
}

TOTAL=$(psql_exec "SELECT count(*) FROM operational_commands WHERE status='dead_letter';")
[ -z "$TOTAL" ] && TOTAL=0

OLDEST=$(psql_exec "SELECT coalesce(min(created_at)::date::text,'-') FROM operational_commands WHERE status='dead_letter';")

BREAKDOWN=$(psql_exec "SELECT command_type || ':' || count(*) FROM operational_commands WHERE status='dead_letter' GROUP BY command_type ORDER BY count(*) DESC;")
BREAKDOWN_LINE=$(echo "$BREAKDOWN" | awk 'BEGIN{ORS=""} {print (NR>1?", ":"") $0}')

if $OUTPUT_JSON; then
  JSON_TYPES=$(echo "$BREAKDOWN" | awk -F: '{printf "%s\"%s\":%s", sep, $1, $2; sep=","}')
  echo "{\"total\":$TOTAL,\"threshold\":$THRESHOLD,\"oldest\":\"$OLDEST\",\"by_type\":{${JSON_TYPES:-}},\"over_threshold\":$([ "$TOTAL" -gt "$THRESHOLD" ] && echo true || echo false)}"
else
  echo "[DEAD-LETTER] 总量: $TOTAL (阈值 $THRESHOLD)"
  echo "[DEAD-LETTER] 最老: $OLDEST"
  echo "[DEAD-LETTER] 分布: ${BREAKDOWN_LINE:--}"
fi

if [ "$TOTAL" -gt "$THRESHOLD" ]; then
  if [ -n "$NOTIFY_WEBHOOK" ]; then
    curl -s -m 5 -X POST "$NOTIFY_WEBHOOK" -H 'Content-Type: application/json' \
      -d "{\"msg_type\":\"text\",\"content\":{\"text\":\"ITSM 死信告警：积压 $TOTAL 条（阈值 $THRESHOLD），最老 $OLDEST。分布: $BREAKDOWN_LINE\"}}" >/dev/null 2>&1 || true
  fi
  [ "$OUTPUT_JSON" = true ] || echo "[DEAD-LETTER] ❌ 超过阈值，请到 /admin/operations 死信面归因处置（重放或 cancel）"
  exit 1
fi

[ "$OUTPUT_JSON" = true ] || echo "[DEAD-LETTER] ✅ 低于阈值"
exit 0
