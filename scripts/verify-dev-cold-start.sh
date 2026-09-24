#!/usr/bin/env bash
#
# scripts/verify-dev-cold-start.sh
#
# 端到端验证 dev 冷启动：起 dev compose → 跑 itsm-init 一次性 init →
# SQL 校验 ticket_categories/tags 组合唯一索引已生效 →
# 校验 4 个新增种子函数（CI types / standard changes / ticket tags /
# incident categories）的实际产物行数。
#
# 用途：审计 2026-09-22 的多租户基线唯一键批次（commit 3fa2c8a2e +
# c6d973d40）在真实数据库上的端到端表现。任何新加 schema/migration 都
# 应改写本脚本的 SQL 校验段。
#
# 用法：
#   ./scripts/verify-dev-cold-start.sh
#   VERBOSE=1 ./scripts/verify-dev-cold-start.sh
#
# 退出码：
#   0 全部 PASS
#   1 任何校验失败
#
# 前置：
#   docker / docker-compose 可用
#   docker-compose.dev.yml 在仓库根
#   itsm-backend Docker 镜像能 build（已 `make build-dev` 或镜像已存在）
#
# 注：本脚本是只读验证 + 起停 dev；不修改 production compose / 不动
#     prod 数据。失败时自动 down 容器保留卷（便于排查），成功才 down -v。

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

LOG_PREFIX="[verify-dev-cold]"
VERBOSE="${VERBOSE:-0}"
FAILS=0

log()  { printf '%s %s\n' "${LOG_PREFIX}" "$*"; }
ok()   { printf '%s \033[32mOK\033[0m  %s\n'    "${LOG_PREFIX}" "$*"; }
fail() { printf '%s \033[31mFAIL\033[0m %s\n'   "${LOG_PREFIX}" "$*"; FAILS=$((FAILS+1)); }
vlog() { [ "${VERBOSE}" = "1" ] && log "$*" || true; }

# ---- 1. 预检 ----
preflight() {
  log "=== 1. 预检 ==="
  command -v docker >/dev/null 2>&1 || { fail "docker 不在 PATH"; return 1; }
  command -v docker compose >/dev/null 2>&1 || command -v docker-compose >/dev/null 2>&1 \
    || { fail "docker compose / docker-compose 都不在 PATH"; return 1; }
  [ -f docker-compose.dev.yml ] || { fail "docker-compose.dev.yml 不在 ${ROOT_DIR}"; return 1; }
  ok "docker / compose / compose 文件就绪"
}

# ---- 2. 起 dev + 等 itsm-init 退 0 ----
bring_up_and_wait() {
  log "=== 2. 起 dev compose + 等 itsm-init 退 0 ==="
  # 用 -V 重置命卷，确保 cold start 而不是 warm 重启
  docker compose -f docker-compose.dev.yml down -v >/dev/null 2>&1 || true
  vlog "已 down 旧容器与卷"
  docker compose -f docker-compose.dev.yml up -d postgres redis minio >/dev/null
  vlog "postgres / redis / minio 已起"

  # 等 pg healthy
  local waited=0
  until docker compose -f docker-compose.dev.yml exec -T postgres pg_isready -U itsm_user -d itsm >/dev/null 2>&1; do
    sleep 2; waited=$((waited+2))
    if [ $waited -gt 60 ]; then fail "postgres 60s 未就绪"; return 1; fi
  done
  ok "postgres ready (${waited}s)"

  # 起 itsm-init（一次性，restart=no，跑完会 exit 0）
  docker compose -f docker-compose.dev.yml up itsm-init > /tmp/itsm-init.log 2>&1 &
  local INIT_PID=$!
  vlog "itsm-init pid=${INIT_PID}, 日志 -> /tmp/itsm-init.log"

  waited=0
  while kill -0 ${INIT_PID} 2>/dev/null; do
    sleep 3; waited=$((waited+3))
    if [ $waited -gt 240 ]; then
      fail "itsm-init 240s 未退（看 /tmp/itsm-init.log）"
      return 1
    fi
  done
  wait ${INIT_PID}; local ec=$?
  if [ $ec -ne 0 ]; then
    fail "itsm-init 退出码 ${ec}（看 /tmp/itsm-init.log 末尾 50 行）"
    vlog "$(tail -50 /tmp/itsm-init.log)"
    return 1
  fi
  ok "itsm-init 退 0（${waited}s）"
}

# ---- 3. SQL 校验 schema 索引 ----
check_schema_indexes() {
  log "=== 3. SQL 校验 schema 唯一索引 ==="
  local query
  query=$(cat <<'SQL'
SELECT t.relname AS table, i.relname AS index, pg_get_indexdef(i.oid)
FROM pg_class t
JOIN pg_index ix ON ix.indrelid = t.oid
JOIN pg_class i ON i.oid = ix.indexrelid
WHERE t.relname IN ('ticket_categories','tags')
  AND i.relname IN ('ticketcategory_tenant_id_code','tag_tenant_id_code','ticket_categories_code_key','tags_code_key')
ORDER BY t.relname, i.relname;
SQL
)
  docker compose -f docker-compose.dev.yml exec -T postgres \
    psql -U itsm_user -d itsm -At -c "${query}" 2>/dev/null > /tmp/indexes.txt || {
    fail "SQL 查询失败"
    return 1
  }
  vlog "当前索引："
  vlog "$(cat /tmp/indexes.txt)"

  # 必须有
  grep -q "^ticket_categories|ticketcategory_tenant_id_code" /tmp/indexes.txt \
    && ok "ticket_categories 组合唯一索引存在" \
    || fail "ticket_categories 组合唯一索引缺失"
  grep -q "^tags|tag_tenant_id_code" /tmp/indexes.txt \
    && ok "tags 组合唯一索引存在" \
    || fail "tags 组合唯一索引缺失"

  # 不能有旧的全局唯一索引
  if grep -qE "ticket_categories_code_key|tags_code_key" /tmp/indexes.txt; then
    fail "旧的全局唯一索引仍在（迁移没生效）"
  else
    ok "旧的全局唯一索引已 drop"
  fi
}

# ---- 4. SQL 校验种子行数 ----
check_seed_data() {
  log "=== 4. SQL 校验 4 个新增种子函数产物 ==="
  # dev 默认租户 ID 是 1（default 租户）
  local rows
  rows=$(docker compose -f docker-compose.dev.yml exec -T postgres \
    psql -U itsm_user -d itsm -At -c \
    "SELECT 'ci_types:'||COUNT(*) FROM ci_types WHERE tenant_id=1
     UNION ALL SELECT 'standard_changes:'||COUNT(*) FROM standard_changes WHERE tenant_id=1
     UNION ALL SELECT 'tags:'||COUNT(*) FROM tags WHERE tenant_id=1
     UNION ALL SELECT 'ticket_categories:'||COUNT(*) FROM ticket_categories WHERE tenant_id=1;" 2>/dev/null)
  vlog "产物行数："
  vlog "${rows}"
  # 期望：每个种子函数至少 1 行（baseline 数据）。
  echo "${rows}" | while IFS=: read -r tbl cnt; do
    if [ "${cnt}" -ge 1 ]; then
      ok "${tbl}=${cnt}"
    else
      fail "${tbl}=${cnt}（期望 ≥ 1）"
    fi
  done
}

# ---- 5. 重复插入测试（核心断言） ----
check_duplicate_insert() {
  log "=== 5. 同一租户 (tenant_id, code) 重复插入应被拒 ==="
  # 拿 tags 一个真实 code
  local code
  code=$(docker compose -f docker-compose.dev.yml exec -T postgres \
    psql -U itsm_user -d itsm -At -c \
    "SELECT code FROM tags WHERE tenant_id=1 LIMIT 1;" 2>/dev/null | tr -d ' \r\n')
  if [ -z "${code}" ]; then
    fail "tags 表为空，无法测重复"
    return 1
  fi
  vlog "用 code='${code}' 重复插入 tenant_id=1"

  local rc
  docker compose -f docker-compose.dev.yml exec -T postgres \
    psql -U itsm_user -d itsm -c \
    "INSERT INTO tags (name, code, tenant_id, created_at, updated_at) VALUES ('dup', '${code}', 1, NOW(), NOW());" \
    > /tmp/dup.log 2>&1
  rc=$?
  if [ $rc -eq 0 ]; then
    fail "重复插入居然成功（组合唯一索引没生效）"
  else
    ok "重复插入被拒（exit ${rc}，组合唯一索引生效）"
  fi
  vlog "$(tail -3 /tmp/dup.log)"
}

# ---- 6. 清理 ----
teardown() {
  log "=== 6. 清理 ==="
  if [ $FAILS -eq 0 ]; then
    docker compose -f docker-compose.dev.yml down -v >/dev/null 2>&1
    ok "已 down -v（成功路径，丢弃卷）"
  else
    docker compose -f docker-compose.dev.yml down >/dev/null 2>&1
    log "有 FAIL，保留卷便于排查（docker-compose.dev.yml down 只停不停数据）"
  fi
}

# ---- main ----
main() {
  preflight      || { teardown; exit 1; }
  bring_up_and_wait || { teardown; exit 1; }
  check_schema_indexes
  check_seed_data
  check_duplicate_insert
  teardown
  echo ""
  if [ $FAILS -eq 0 ]; then
    printf '%s \033[32mALL CHECKS PASSED\033[0m\n' "${LOG_PREFIX}"
    exit 0
  else
    printf '%s \033[31m%d CHECK(S) FAILED\033[0m\n' "${LOG_PREFIX}" "$FAILS"
    exit 1
  fi
}

main
