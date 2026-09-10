#!/bin/bash
#
# release-evidence.sh — 发布证据包生成（Upgrade Guard C4）
#
# 把「每次升级必须可追溯」落到一个命令：
#   Git SHA / 镜像 digest / DB 版本 / 迁移账本 / E2E 与预检结论
#   聚合为 output/release-evidence-<version>-<timestamp>.md（append-only 证据档案）
#
# Usage:
#   ./scripts/release-evidence.sh [--e2e-summary <file>]
#
# Options:
#   --e2e-summary <file>  把 E2E 测试结论文件嵌入证据包（可选）
#
# Exit code: 0 = 证据包已生成；1 = 关键证据缺失（镜像/DB 不可达）。

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

ENV_FILE="${PROJECT_ROOT}/.env.prod"
OUT_DIR="${PROJECT_ROOT}/output"
E2E_SUMMARY=""

while [ $# -gt 0 ]; do
  case "$1" in
    --e2e-summary) E2E_SUMMARY="$2"; shift 2 ;;
    *) log_warn "未知参数: $1（忽略）"; shift ;;
  esac
done

mkdir -p "$OUT_DIR"

# ---------- Git 证据 ----------
GIT_SHA="$(git -C "$PROJECT_ROOT" rev-parse HEAD 2>/dev/null || echo UNKNOWN)"
GIT_DIRTY="$(git -C "$PROJECT_ROOT" status --porcelain 2>/dev/null | head -5)"
GIT_BRANCH="$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo UNKNOWN)"
GIT_TAG="$(git -C "$PROJECT_ROOT" describe --tags --exact-match 2>/dev/null || echo 'no-tag')"

# ---------- 版本 ----------
VERSION_VAL="$(grep -E '^VERSION=' "$ENV_FILE" 2>/dev/null | cut -d= -f2 || echo 'unset')"
[ -z "$VERSION_VAL" ] && VERSION_VAL="unset"

TS="$(date -u +'%Y%m%d-%H%M%S')"
OUT_FILE="${OUT_DIR}/release-evidence-${VERSION_VAL}-${TS}.md"

# ---------- 镜像证据 ----------
backend_digest="$(docker images --digests --format '{{.Digest}}' itsm-backend:latest 2>/dev/null | head -1)"
frontend_digest="$(docker images --digests --format '{{.Digest}}' itsm-frontend:latest 2>/dev/null | head -1)"
backend_id="$(docker images --format '{{.ID}}' itsm-backend:latest 2>/dev/null | head -1)"
frontend_id="$(docker images --format '{{.ID}}' itsm-frontend:latest 2>/dev/null | head -1)"
backend_created="$(docker images --format '{{.CreatedAt}}' itsm-backend:latest 2>/dev/null | head -1)"

# ---------- 运行态证据 ----------
backend_running="$(docker inspect --format '{{.State.Health.Status}}' itsm-backend-prod 2>/dev/null || echo 'not-running')"

# ---------- DB 证据 ----------
db_version="$(docker exec itsm-postgres-prod psql -U itsm -d itsm_prod -t -A -c 'SHOW server_version;' 2>/dev/null || echo 'DB-UNREACHABLE')"
migration_count="$(docker exec itsm-postgres-prod psql -U itsm -d itsm_prod -t -A -c 'SELECT count(*) FROM schema_migrations;' 2>/dev/null || echo 'N/A')"
last_migrations="$(docker exec itsm-postgres-prod psql -U itsm -d itsm_prod -t -A -c "SELECT version || ' (' || release_version || ', ' || applied_at::date || ')' FROM schema_migrations ORDER BY version DESC LIMIT 8;" 2>/dev/null | sed 's/^/- /' || echo '- N/A')"
serial_tables="$(docker exec itsm-postgres-prod psql -U itsm -d itsm_prod -t -A -c "
  SELECT string_agg(DISTINCT c.relname, ',')
  FROM pg_index i JOIN pg_class c ON c.oid=i.indrelid
  JOIN pg_namespace n ON n.oid=c.relnamespace
  JOIN information_schema.columns ic ON ic.table_schema=n.nspname AND ic.table_name=c.relname
   AND ic.column_name='id' AND ic.is_identity='NO' AND ic.column_default LIKE 'nextval%'
  WHERE n.nspname='public' AND c.relkind='r';
" 2>/dev/null || echo 'N/A')"

# ---------- 预检证据 ----------
preflight_note="未运行（建议：升级前先 make preflight）"
if [ -n "${PREFLIGHT_RESULT:-}" ]; then
  preflight_note="$PREFLIGHT_RESULT"
fi

# ---------- 组装 ----------
cat > "$OUT_FILE" <<EOF
# 发布证据包：${VERSION_VAL}

> 本文件由 \`make release-evidence\` 生成，属不可变证据档案——**生成后不得手改**；
> 如需补充结论，追加新章节（勿改历史行）。

- 生成时间：$(date -u +'%Y-%m-%dT%H:%M:%SZ')
- 放行结论：⬜ 未评定（升级完成后由责任人手动填写：⬜ 放行 / ⬜ 拒绝 + 理由）

## 1. 版本与代码

| 项 | 值 |
|:---|:---|
| VERSION | ${VERSION_VAL} |
| Git branch | ${GIT_BRANCH} |
| Git SHA | ${GIT_SHA} |
| Git tag | ${GIT_TAG} |
| 工作区状态 | $( [ -z "$GIT_DIRTY" ] && echo 'clean（与 SHA 完全一致）' || echo "DIRTY — 未提交变更：${GIT_DIRTY//$'\n'/; }" ) |

## 2. 镜像

| 镜像 | ID | digest |
|:---|:---|:---|
| itsm-backend | ${backend_id:-N/A} | ${backend_digest:-N/A} |
| itsm-frontend | ${frontend_id:-N/A} | ${frontend_digest:-N/A} |

- backend 镜像构建时间：${backend_created:-N/A}
- itsm-backend-prod 健康状态：${backend_running}

## 3. 数据库与迁移

- PostgreSQL server_version：${db_version}
- 迁移账本总条数：${migration_count}
- 最近应用（账本倒序 8 条）：

${last_migrations}

- SERIAL 风格 id 表（漂移信号，应为 raw-SQL 管辖表）：${serial_tables:-无}

## 4. 门禁结论

| 门禁 | 结论 | 记录方式 |
|:---|:---|:---|
| preflight | ${preflight_note} | make preflight 输出 |
| backend go test | ⬜ 未填写 | CI/本机运行后回填 |
| E2E 业务旅程 | ⬜ 未填写 | make e2e / 手动运行后回填 |
| 备份与恢复演练 | ⬜ 未填写 | scripts/backup.sh verify 后回填 |

EOF

# ---------- E2E 摘要嵌入 ----------
if [ -n "$E2E_SUMMARY" ] && [ -f "$E2E_SUMMARY" ]; then
  cat >> "$OUT_FILE" <<EOF
## 5. E2E 结论（来源：$(basename "$E2E_SUMMARY")）

\`\`\`
$(head -50 "$E2E_SUMMARY")
\`\`\`

EOF
fi

log_info "证据包已生成: ${OUT_FILE#$PROJECT_ROOT/}"

# 关键证据缺失告警
MISSING=0
[ -z "$backend_digest" ] && log_warn "itsm-backend 镜像 digest 缺失" && MISSING=$((MISSING+1))
[ "$db_version" = "DB-UNREACHABLE" ] && log_warn "DB 不可达，数据库证据缺失" && MISSING=$((MISSING+1))
[ -n "$GIT_DIRTY" ] && log_warn "工作区不干净：证据 SHA 不能等同发布内容，建议先提交"
[ "$MISSING" -gt 0 ] && exit 1
exit 0
