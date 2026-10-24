#!/bin/bash
# prod-post-deploy-verify.sh - 部署后验收剧本（P1-1）
#
# 背景：此前部署验收靠临时拼命令，本脚本把「滚动更新后必须过的检查」固化为
# 一条命令，输出可作为发布证据留存（ROADMAP 发布门禁第 2 条）。
#
# 检查项:
#   1. 版本对齐（调用 check-deploy-version.sh，--skip-version 可跳过）
#   2. 容器健康（docker compose ps 全部 healthy）
#   3. readyz 200
#   4. 管理员登录（httpOnly cookie，用后即删）+ 18 域读探测全 200/code:0
#   5. 死信计数汇总（不判阈值，仅披露；阈值判断见 prod-dead-letter-check.sh）
#
# 用法:
#   ./scripts/prod-post-deploy-verify.sh               # 标准验收
#   ./scripts/prod-post-deploy-verify.sh --full        # 追加 itsm-cli doctor + flow run
#
# 环境变量:
#   BASE_URL       入口地址（默认 http://localhost，走 nginx）
#   ADMIN_USERNAME 登录用户（默认 admin）
#   ADMIN_PASSWORD 默认从 <repo>/.env.prod 读取；也可显式传入
#   ENV_FILE       env 文件路径（默认 <repo>/.env.prod）

set -euo pipefail

export PATH="/Applications/Docker.app/Contents/Resources/bin:/usr/local/bin:/opt/homebrew/bin:${PATH}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BASE_URL="${BASE_URL:-http://localhost}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env.prod}"
RUN_FULL=false
[ "${1:-}" = "--full" ] && RUN_FULL=true
SKIP_VERSION=false
for a in "$@"; do [ "$a" = "--skip-version" ] && SKIP_VERSION=true; done

GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
PASS=0; FAIL=0
ok()   { echo -e "${GREEN}[PASS]${NC} $1"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL=$((FAIL+1)); }

COOKIE_FILE="$(mktemp)"
trap 'rm -f "$COOKIE_FILE"' EXIT

echo "=== ITSM 部署后验收 $(date '+%F %T') base=$BASE_URL ==="

# ---- 1. 版本对齐 ----
if $SKIP_VERSION; then
  echo "[SKIP] 版本对齐检查"
else
  if bash "$SCRIPT_DIR/check-deploy-version.sh"; then ok "版本对齐（.env.prod VERSION == git tag）"; else fail "版本漂移（见上方 VERSION 输出）"; fi
fi

# ---- 2. 容器健康 ----
COMPOSE_PROJECT="${COMPOSE_PROJECT:-itsm-prod}"
UNHEALTHY=$(docker ps --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" --format '{{.Names}} {{.Status}}' | grep -cv healthy || true)
if [ "$UNHEALTHY" -eq 0 ]; then
  ok "compose 项目 ${COMPOSE_PROJECT} 全容器 healthy"
else
  docker ps --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" --format '{{.Names}} {{.Status}}' | grep -v healthy || true
  fail "${UNHEALTHY} 个容器非 healthy"
fi

# ---- 3. readyz ----
READY=$(curl -s -o /dev/null -w '%{http_code}' -m 5 "$BASE_URL/api/v1/readyz" 2>/dev/null || echo 000)
if [ "$READY" = "200" ]; then ok "readyz 200"; else fail "readyz 返回 $READY"; fi

# ---- 4. 登录 + 18 域读探测 ----
if [ -z "${ADMIN_PASSWORD:-}" ] && [ -f "$ENV_FILE" ]; then
  ADMIN_PASSWORD="$(grep -E '^ADMIN_PASSWORD=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]')"
fi
if [ -z "${ADMIN_PASSWORD:-}" ]; then
  fail "无法获得 ADMIN_PASSWORD（未显式传入且 $ENV_FILE 缺失该键）"
else
  LOGIN_HTTP=$(curl -s -o /tmp/itsm-verify-login.json -w '%{http_code}' -m 10 \
    -c "$COOKIE_FILE" -X POST "$BASE_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"${ADMIN_USERNAME}\",\"password\":\"${ADMIN_PASSWORD}\"}" || echo 000)
  LOGIN_CODE=$(sed -n 's/.*"code":\([0-9]*\).*/\1/p' /tmp/itsm-verify-login.json | head -1)
  rm -f /tmp/itsm-verify-login.json
  if [ "$LOGIN_HTTP" = "200" ] && [ "$LOGIN_CODE" = "0" ]; then
    ok "管理员登录 ${ADMIN_USERNAME}（cookie 用后即删）"
  else
    fail "登录失败 HTTP=$LOGIN_HTTP code=$LOGIN_CODE（后续探测跳过）"
  fi

  PROBE_PATHS=(
    tickets incidents problems changes service-requests
    cmdb/cis knowledge/articles bpmn/process-definitions bpmn/process-instances
    sla/policies notification-deliveries connectors releases
    marketplace/items admin/operations/commands users tenants reports/service-quality
  )
  FAILED_PROBES=""
  for P in "${PROBE_PATHS[@]}"; do
    BODY=$(curl -s -m 10 -b "$COOKIE_FILE" "$BASE_URL/api/v1/$P?page=1&pageSize=1")
    HTTP_OK=$(echo "$BODY" | grep -q '"code":0' && echo yes || echo no)
    if [ "$HTTP_OK" = "yes" ]; then
      ok "读探测 /$P"
    else
      fail "读探测 /$P 非 code:0: $(echo "$BODY" | head -c 120)"
      FAILED_PROBES="$FAILED_PROBES $P"
    fi
  done
fi

# ---- 5. 死信披露（不判阈值） ----
DB_USER="$(grep -E '^DB_USER=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]' 2>/dev/null || true)"
DB_NAME="$(grep -E '^DB_NAME=' "$ENV_FILE" | head -1 | cut -d= -f2 | tr -d '[:space:]' 2>/dev/null || true)"
if [ -n "$DB_USER" ] && [ -n "$DB_NAME" ]; then
  DL_COUNT=$(docker exec itsm-postgres-prod psql -U "$DB_USER" -d "$DB_NAME" -t -A -c \
    "SELECT count(*) FROM operational_commands WHERE status='dead_letter';" 2>/dev/null || echo "?")
  if [ "$DL_COUNT" != "?" ]; then
    echo "[INFO] dead-letter stock: $DL_COUNT (threshold/处置见 scripts/prod-dead-letter-check.sh + /admin/operations)"
    ok "dead-letter count readable"
  else
    fail "dead-letter count query failed"
  fi
fi

# ---- 6. 可选 --full：doctor + flow run ----
if $RUN_FULL; then
  if [ -x "$REPO_ROOT/itsm-cli/bin/itsm.js" ] || [ -f "$REPO_ROOT/itsm-cli/bin/itsm.js" ]; then
    if node "$REPO_ROOT/itsm-cli/bin/itsm.js" doctor >/dev/null 2>&1; then ok "itsm-cli doctor"; else fail "itsm-cli doctor"; fi
    if node "$REPO_ROOT/itsm-cli/bin/itsm.js" flow run --json >/dev/null 2>&1; then ok "itsm-cli flow run"; else fail "itsm-cli flow run（场景失败需人工判 failed/blocked）"; fi
  else
    fail "--full 需要 itsm-cli（$REPO_ROOT/itsm-cli/bin/itsm.js 不存在）"
  fi
fi

echo "=== 验收结果: $PASS passed / $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
  [ -n "${FAILED_PROBES:-}" ] && echo "失败探测:$FAILED_PROBES"
  exit 1
fi
echo "可作为发布证据留存（建议归档: output/release-evidence/<date>-post-deploy.txt）"
exit 0
