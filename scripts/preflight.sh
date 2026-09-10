#!/bin/bash
#
# preflight.sh — ITSM 升级/部署前预检（Upgrade Guard C1 第一块）
#
# 2026-09-10 事故驱动（详见 docs/product/deployment-upgrade-safety-diagnosis-and-design.md）：
#   - prod 半瘫 502（dev/prod 共用 compose 项目名 + 8090 端口争用）
#   - itsm-init 因 workflow_templates SERIAL/IDENTITY 漂移硬失败
#   - add_missing_indexes.sql 写好 4 个月从未执行（脚本已写未执行 = 事故）
#
# 本脚本把上述排障动作固化为升级前必跑的只读检查，输出可评审报告：
#   check 1  Docker 环境与 compose 配置有效性
#   check 2  端口争用（BACKEND_DIAG_PORT / 8090 与本机监听）
#   check 3  镜像 freshness（VERSION vs prod 运行镜像 digest，latest 渗透告警）
#   check 4  数据库 schema 漂移（legacy 账本缺口 + SERIAL/IDENTITY 异常列）
#   check 5  索引缺口（仅 PK 表 / 有 tenant_id 无租户前缀索引）
#
# Usage:
#   ./scripts/preflight.sh [--db-only] [--json]
#
# Options:
#   --db-only   只跑数据库相关检查（4/5）
#   --json      机器可读输出（追加在人类可读输出之后）
#
# Exit code: 0 = 全部通过；1 = 存在 BLOCKER（升级前必须处理）；2 = 仅 WARN。

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

COMPOSE_PROD="${PROJECT_ROOT}/docker-compose.prod.yml"
ENV_FILE="${PROJECT_ROOT}/.env.prod"

DB_ONLY=false
JSON_OUTPUT=false
for arg in "$@"; do
  case "$arg" in
    --db-only) DB_ONLY=true ;;
    --json) JSON_OUTPUT=true ;;
    *) log_warn "未知参数: ${arg}（忽略）" ;;
  esac
done

BLOCKERS=0
WARNS=0
declare -a SUMMARY=()

note() { # note <status> <message>
  case "$1" in
    BLOCKER) log_error "$2"; BLOCKERS=$((BLOCKERS+1)); SUMMARY+=("BLOCKER: $2") ;;
    WARN)    log_warn  "$2"; WARNS=$((WARNS+1));    SUMMARY+=("WARN: $2") ;;
    *)       log_success "[OK]      $2" ;;
  esac
}

pg_exec() { # pg_exec <psql args...>  — 复用容器内 psql
  docker exec itsm-postgres-prod psql -U "${DB_USER:-itsm}" -d "${DB_NAME:-itsm_prod}" "$@"
}

# ============================================================
# check 1: Docker 环境与 compose 配置
# ============================================================
if [ "$DB_ONLY" = false ]; then
  log_phase "Check 1/5 · Docker 环境与 compose 配置"
  if ! docker info >/dev/null 2>&1; then
    note BLOCKER "Docker 守护进程不可用"
  else
    log_info "Docker 守护进程正常"
    if [ ! -f "$ENV_FILE" ]; then
      note BLOCKER ".env.prod 不存在（先运行 make prod-init）"
    fi
    if [ -f "$COMPOSE_PROD" ] && [ -f "$ENV_FILE" ]; then
      if dc -f "$COMPOSE_PROD" --env-file "$ENV_FILE" config -q 2>/dev/null; then
        log_info "docker-compose.prod.yml 语法与变量解析通过"
      else
        note BLOCKER "docker-compose.prod.yml config 校验失败"
      fi
      # 项目名固定校验（dev/prod 隔离修复 2026-09-10）
      if grep -qE '^name: itsm-prod' "$COMPOSE_PROD"; then
        log_info "compose 项目名已固定为 itsm-prod（dev/prod 隔离）"
      else
        note WARN "docker-compose.prod.yml 顶部未固定 name: itsm-prod，dev/prod 可能互相顶掉容器"
      fi
    fi
  fi
fi

# ============================================================
# check 2: 端口争用
# ============================================================
if [ "$DB_ONLY" = false ]; then
  log_phase "Check 2/5 · 端口争用"
  DIAG_PORT="$(grep -E '^BACKEND_DIAG_PORT=' "$ENV_FILE" 2>/dev/null | cut -d= -f2 || echo 8090)"
  DIAG_PORT="${DIAG_PORT:-8090}"
  for p in "$DIAG_PORT" 80 3000 5432 6379; do
    if port_in_use "$p"; then
      holder="$(lsof -iTCP:"$p" -sTCP:LISTEN -P 2>/dev/null | tail -1 | awk '{print $1}')"
      if [ "$p" = "5432" ] || [ "$p" = "6379" ]; then
        note WARN "本机 $holder 占用 ${p}（本机服务与容器端口易混淆，确认连接目标）"
      else
        log_info "$p 已监听（${holder}）— 确认属预期服务"
      fi
    else
      log_info "$p 空闲"
    fi
  done
  if [ "$DIAG_PORT" = "8090" ] && port_in_use 8090; then
    holder="$(lsof -iTCP:8090 -sTCP:LISTEN -P 2>/dev/null | tail -1 | awk '{print $1}')"
    note BLOCKER "BACKEND_DIAG_PORT=8090 且 8090 已被 $holder 占用 — dev 栈同机时必须错开（.env.prod 设 8091）"
  fi
fi

# ============================================================
# check 3: 镜像 freshness（VERSION vs 运行镜像）
# ============================================================
if [ "$DB_ONLY" = false ]; then
  log_phase "Check 3/5 · 镜像 freshness"
  VERSION_VAL="$(grep -E '^VERSION=' "$ENV_FILE" 2>/dev/null | cut -d= -f2 || echo '')"
  if [ -z "$VERSION_VAL" ]; then
    note WARN ".env.prod 未设置 VERSION"
  elif [ "$VERSION_VAL" = "latest" ]; then
    note WARN "VERSION=latest：版本不可追溯（无 digest 证据），企业交付禁止 latest"
  else
    log_info "VERSION=$VERSION_VAL"
    if docker image inspect "itsm-backend:${VERSION_VAL}" >/dev/null 2>&1; then
      log_info "本地存在 itsm-backend:${VERSION_VAL}"
    else
      note WARN "本地不存在 itsm-backend:${VERSION_VAL}，运行中的可能是旧镜像"
    fi
  fi
  # 运行容器与本地 latest 镜像一致性（改码未重建的经典场景）
  if container_running itsm-backend-prod; then
    run_id="$(docker inspect --format '{{.Image}}' itsm-backend-prod 2>/dev/null | cut -c8-19)"
    latest_id="$(docker images --format '{{.ID}}' itsm-backend:latest 2>/dev/null | head -1)"
    if [ -n "$run_id" ] && [ -n "$latest_id" ] && [ "$run_id" != "$latest_id" ]; then
      note WARN "itsm-backend-prod 运行镜像 ≠ 本地 latest（源码改了但没重建/没重启）"
    else
      log_info "运行镜像与本地 latest 一致"
    fi
  fi
fi

# ============================================================
# check 4: 数据库 schema 漂移
# ============================================================
log_phase "Check 4/5 · 数据库 schema 漂移"
if ! docker exec itsm-postgres-prod psql -U "${DB_USER:-itsm}" -d "${DB_NAME:-itsm_prod}" -c '\q' >/dev/null 2>&1; then
  note BLOCKER "itsm-postgres-prod 不可达（无法做 DB 侧检查）"
else
  # 4a. legacy 账本缺口（001-006 未记账 → 发现机制会重放）
  missing_legacy="$(pg_exec -t -A -c "
    SELECT string_agg(v, ',') FROM (VALUES
      ('001_initial_schema'),('002_add_notification_preferences'),('003_add_audit_indexes'),
      ('004_add_sla_calendar'),('005_add_external_id_mapping'),('006_add_change_approvals')
    ) AS legacy(v)
    WHERE NOT EXISTS (SELECT 1 FROM schema_migrations sm WHERE sm.version = legacy.v);
  " 2>/dev/null || echo 'QUERY_FAIL')"
  if [ "$missing_legacy" = "QUERY_FAIL" ]; then
    note BLOCKER "schema_migrations 表查询失败（账本损坏？）"
  elif [ -n "$missing_legacy" ]; then
    note WARN "legacy 迁移未记账: ${missing_legacy}（修复后首次启动会自动补记，不影响升级）"
  else
    log_info "legacy 账本 001-006 完整"
  fi

  # 4b. SERIAL/IDENTITY 漂移：ent 管辖表（132 张）应全为 IDENTITY；
  #     已知例外=raw SQL 表，出现新例外即漂移信号
  serial_tables="$(pg_exec -t -A -c "
    SELECT string_agg(DISTINCT c.relname, ',' ORDER BY c.relname)
    FROM pg_index i
    JOIN pg_class c ON c.oid = i.indrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN information_schema.columns ic
      ON ic.table_schema = n.nspname AND ic.table_name = c.relname
     AND ic.column_name = 'id' AND ic.is_identity = 'NO'
     AND ic.column_default LIKE 'nextval%'
    WHERE n.nspname = 'public' AND c.relkind = 'r';
  " 2>/dev/null || echo 'QUERY_FAIL')"
  if [ "$serial_tables" = "QUERY_FAIL" ]; then
    note WARN "SERIAL 表扫描失败"
  elif [ -n "$serial_tables" ]; then
    note WARN "存在 SERIAL 风格 id 的表: ${serial_tables}（ent diff 可能硬失败，确认是否在 ent 管辖内）"
  else
    log_info "无 SERIAL 风格 id 表（IDENTITY 对齐）"
  fi
fi

# ============================================================
# check 5: 索引缺口
# ============================================================
log_phase "Check 5/5 · 索引缺口"
pk_only_count="$(pg_exec -t -A -c "
  SELECT count(*) FROM (
    SELECT i.indrelid
    FROM pg_index i
    JOIN pg_class c ON c.oid = i.indrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relkind = 'r'
      AND c.relname NOT LIKE 'schema_migrations'
    GROUP BY i.indrelid
    HAVING count(*) = 1
  ) t;
" 2>/dev/null || echo 'QUERY_FAIL')"
if [ "$pk_only_count" = "QUERY_FAIL" ]; then
  note WARN "索引扫描失败"
elif [ "$pk_only_count" -gt 50 ]; then
  note WARN "仅 PK 索引的表 ${pk_only_count} 张（历史基线残留；>50 属已知批次，增速异常才 BLOCKER）"
elif [ "$pk_only_count" -gt 0 ]; then
  note WARN "仅 PK 索引的表 ${pk_only_count} 张"
else
  log_info "索引覆盖完整"
fi
tenant_missing="$(pg_exec -t -A -c "
  SELECT count(*) FROM information_schema.columns t
  WHERE t.table_schema = 'public' AND t.column_name = 'tenant_id'
    AND NOT EXISTS (
      SELECT 1 FROM pg_indexes p
      WHERE p.schemaname = 'public' AND p.tablename = t.table_name
        AND p.indexdef ILIKE '%tenant_id%'
    );
" 2>/dev/null || echo 'QUERY_FAIL')"
if [ "$tenant_missing" != "QUERY_FAIL" ] && [ "$tenant_missing" -gt 0 ]; then
  note WARN "有 tenant_id 但缺租户前缀索引的表 ${tenant_missing} 张（多租户查询性能+隔离证据缺口）"
fi

# ============================================================
# 汇总
# ============================================================
log_phase "Preflight 汇总"
if [ ${#SUMMARY[@]} -eq 0 ]; then
  log_info "全部检查通过，可以升级/部署。"
fi
for item in "${SUMMARY[@]}"; do
  echo "  $item"
done
if [ "$JSON_OUTPUT" = true ]; then
  echo '{"blockers": '"$BLOCKERS"', "warnings": '"$WARNS"'}'
fi

if [ "$BLOCKERS" -gt 0 ]; then
  log_error "Preflight 未通过：${BLOCKERS} 个 BLOCKER，${WARNS} 个 WARN —— 先修复再升级"
  exit 1
elif [ "$WARNS" -gt 0 ]; then
  log_warn "Preflight 通过（带 ${WARNS} 个 WARN）—— 评审后可继续"
  exit 2
fi
log_info "Preflight 全部通过"
exit 0
