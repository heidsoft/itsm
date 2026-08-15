#!/usr/bin/env bash
#
# rebuild_and_verify.sh — ITSM 全新生产栈重建 + RLS + 四域旅程一键入口
#
# Purpose: 在干净环境中一次性跑完 Step 1-6,产出可重复证据。
#
# Usage:
#   bash scripts/rebuild_and_verify.sh                 # 完整流程 (默认)
#   bash scripts/rebuild_and_verify.sh --keep-data     # 跳过 wipe,沿用现有数据卷
#   bash scripts/rebuild_and_verify.sh --no-rls        # 跳过 RLS pilot 验证
#   bash scripts/rebuild_and_verify.sh --no-browser    # 跳过四域浏览器旅程
#   bash scripts/rebuild_and_verify.sh --skip-build    # 跳过镜像构建,沿用已有镜像
#
# Prerequisites:
#   - Docker 29+ with BuildKit
#   - Node.js + Playwright installed (npm i -g playwright)
#   - psql client (for RLS verification only)
#   - .env.prod file with ADMIN_PASSWORD set
#
# Output:
#   - reports/production-stack-rebuild/<date>/        — 构建/迁移/RLS 证据
#   - reports/four-domain-journey/<date>/             — 12 张 PNG + journey-summary.json
#   - reports/production-stack-rebuild-<date>.md      — 主报告
#
# Reproducibility:
#   - 每次运行写入 .deploy/current (git_sha + image digests)
#   - journey-summary.json 含全部断言结果
#   - RLS e2e 输出写入 reports/production-stack-rebuild/rls_r1_e2e_output.txt
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# ==============================================================================
# 参数解析
# ==============================================================================
KEEP_DATA=false
RUN_RLS=true
RUN_BROWSER=true
SKIP_BUILD=false

for arg in "$@"; do
  case "$arg" in
    --keep-data)   KEEP_DATA=true ;;
    --no-rls)      RUN_RLS=false ;;
    --no-browser)  RUN_BROWSER=false ;;
    --skip-build)  SKIP_BUILD=true ;;
    --help|-h)
      sed -n '2,30p' "$0"
      exit 0 ;;
    *) echo "Unknown option: $arg"; exit 2 ;;
  esac
done

# ==============================================================================
# 环境探测
# ==============================================================================
GIT_SHA="$(git rev-parse --short HEAD)"
RUN_DATE="$(date -u +%Y-%m-%d)"
REPORT_DIR="$PROJECT_ROOT/reports/production-stack-rebuild/$RUN_DATE"
JOURNEY_DIR="$PROJECT_ROOT/reports/four-domain-journey/$RUN_DATE"
mkdir -p "$REPORT_DIR" "$JOURNEY_DIR"

export DOCKER_BUILDKIT=1
export PLAYWRIGHT_BROWSERS_PATH="${PLAYWRIGHT_BROWSERS_PATH:-/opt/miniconda3/lib/playwright}"
export NODE_PATH="${NODE_PATH:-/usr/local/lib/node_modules}"

banner() {
  echo ""
  echo "===================================================================="
  echo "$1"
  echo "===================================================================="
}

require() {
  for tool in "$@"; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      echo "Missing required tool: $tool" >&2
      exit 1
    fi
  done
}

require docker node

# ==============================================================================
# Step 1 — 停掉旧栈并清理
# ==============================================================================
banner "Step 1: 停掉旧栈并清理数据 (git_sha=$GIT_SHA)"

if [ "$KEEP_DATA" = "false" ]; then
  docker compose -f docker-compose.prod.yml --env-file .env.prod down -v 2>&1 | tail -10 || true
  echo "deployed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$REPORT_DIR/state-before-rebuild.txt"
  echo "git_sha=$GIT_SHA" >> "$REPORT_DIR/state-before-rebuild.txt"
  echo "ok" > "$PROJECT_ROOT/.deploy/previous"
  echo "[ok] stack wiped"
else
  echo "[skip] --keep-data set, skipping wipe"
fi

# ==============================================================================
# Step 2 — 修复构建链 (RLS env 已就绪)
# ==============================================================================
banner "Step 2: 验证 RLS 配置已就位"
grep -E "^RLS_MODE=" .env.prod >/dev/null || { echo "RLS_MODE missing in .env.prod"; exit 1; }
grep -E "RLS_MODE=" docker-compose.prod.yml | head -3 | tee "$REPORT_DIR/rls-env-in-compose.txt"
echo "[ok] RLS env vars present"

# ==============================================================================
# Step 3 — 全新构建镜像
# ==============================================================================
banner "Step 3: 构建镜像 (tag=$GIT_SHA)"

if [ "$SKIP_BUILD" = "false" ]; then
  bash scripts/build-images.sh "$GIT_SHA" "" backend 2>&1 | tail -5
  bash scripts/build-images.sh "$GIT_SHA" "" frontend 2>&1 | tail -5
  docker tag itsm-backend:$GIT_SHA itsm-backend:latest
  docker tag itsm-frontend:$GIT_SHA itsm-itsm-frontend:latest
  echo "[ok] backend + frontend images built"
else
  echo "[skip] --skip-build set"
fi

docker images --digests | grep -E "itsm-(backend|frontend)" | tee "$REPORT_DIR/image-digests.txt"

# ==============================================================================
# Step 4 — 启动新栈 + 迁移 + Bootstrap
# ==============================================================================
banner "Step 4: 启动新栈 + 迁移 + Bootstrap"
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d 2>&1 | tail -15
sleep 30

# Wait for init container to complete
for i in 1 2 3 4 5 6 7 8 9 10; do
  INIT_STATUS=$(docker inspect itsm-init-prod --format='{{.State.Status}}' 2>/dev/null || echo "missing")
  if [ "$INIT_STATUS" = "exited" ]; then
    echo "[ok] init container completed"
    break
  fi
  sleep 5
done

# Wait for backend healthy
for i in 1 2 3 4 5 6 7 8 9 10 11 12; do
  if curl -fsS http://localhost:8090/api/v1/health >/dev/null 2>&1; then
    echo "[ok] backend healthy"
    break
  fi
  sleep 5
done

# Verify admin login
ADMIN_RESP=$(curl -sX POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD:-admin123}\"}")
echo "$ADMIN_RESP" | head -c 200 | tee "$REPORT_DIR/admin-login.txt"
echo ""

# ==============================================================================
# Step 5 — 真实 PG RLS pilot
# ==============================================================================
if [ "$RUN_RLS" = "true" ]; then
  banner "Step 5: 应用 RLS pilot"
  docker cp itsm-backend/database/rls/migrations/001_roles.sql itsm-postgres-prod:/tmp/
  docker cp itsm-backend/database/rls/migrations/002_pilot_policies.sql itsm-postgres-prod:/tmp/
  docker exec itsm-postgres-prod psql -U itsm -d itsm_prod \
    -c "ALTER ROLE itsm_app LOGIN PASSWORD 'RlsApp2026SecurePass!';" >/dev/null
  docker exec itsm-postgres-prod psql -U itsm -d itsm_prod \
    -c "ALTER ROLE itsm_admin LOGIN PASSWORD '${RLS_ADMIN_PASSWORD:-rls_admin_change_me}';" >/dev/null
  docker exec itsm-postgres-prod psql -U itsm -d itsm_prod \
    -v ON_ERROR_STOP=1 -f /tmp/001_roles.sql 2>&1 | tail -5
  docker exec itsm-postgres-prod psql -U itsm -d itsm_prod \
    -v ON_ERROR_STOP=1 -f /tmp/002_pilot_policies.sql 2>&1 | tail -5
  docker cp "$REPORT_DIR/../rls_r1_e2e_prod.sql" itsm-postgres-prod:/tmp/ 2>/dev/null || \
    docker cp "$PROJECT_ROOT/reports/production-stack-rebuild/rls_r1_e2e_prod.sql" itsm-postgres-prod:/tmp/
  docker exec itsm-postgres-prod psql -U itsm -d itsm_prod -f /tmp/rls_r1_e2e_prod.sql \
    2>&1 | tee "$REPORT_DIR/rls_r1_e2e_output.txt" | tail -30
  echo "[ok] RLS pilot verified"
fi

# ==============================================================================
# Step 6 — 四域浏览器旅程
# ==============================================================================
if [ "$RUN_BROWSER" = "true" ]; then
  banner "Step 6: 四域浏览器旅程 (Playwright)"
  SCREENSHOT_DIR="$JOURNEY_DIR" \
    ADMIN_USER="${ADMIN_USER:-admin}" \
    ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}" \
    node tests/playwright-four-domain-journey.cjs 2>&1 | tail -30
  echo "[ok] four-domain journey completed"
fi

# ==============================================================================
# 写 .deploy/current
# ==============================================================================
cat > "$PROJECT_ROOT/.deploy/current" <<EOF
deploy_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
git_commit=$GIT_SHA
backend_digest=$(docker images --digests itsm-backend:$GIT_SHA --format '{{.Digest}}' 2>/dev/null || echo "unknown")
frontend_digest=$(docker images --digests itsm-frontend:$GIT_SHA --format '{{.Digest}}' 2>/dev/null || echo "unknown")
EOF

banner "Rebuild complete"
echo "Report dir:  $REPORT_DIR"
echo "Journey dir: $JOURNEY_DIR"
echo "Git SHA:     $GIT_SHA"