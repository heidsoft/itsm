#!/usr/bin/env bash
#
# fresh-install-gate.sh — 证明「私有化全新部署」这条链真的能跑起来。
#
# 为什么要单独一个门禁：日期化磁盘迁移在既有安装里走「收养不执行」（migration/legacy_record.go
# 的 adoptionCutoffUTC 之前只登记、不执行，账本 checksum 为空），所以引用不存在对象、或去读/写
# seed 才拥有的数据的脚本，只在空卷首装时暴露。2026-10-01 清卷全新部署实测就是靠这条链抓出 9 个
# 脚本缺陷；而唯一会跑到它的 ga-gate「Start core stack」自 2026-09-22 起连续红、且不在 main 的
# 必需检查里。本脚本把同一个证明做成独立、廉价、可读的断言。
#
# 断言（全部来自空卷实跑，不是单测 mock）：
#   1. itsm-init（ITSM_BOOTSTRAP_ONLY + AUTO_MIGRATE + AUTO_SEED）退出码必须为 0；
#   2. schema_migrations 入账数达到下限，且每条磁盘迁移都有非空 checksum（空 checksum
#      意味着该脚本被登记却从未真正执行过，正是「只在新装炸」的来源；历史收养条目 001-006 除外）；
#   3. process_definitions 有已部署的活跃模板（workflow-core 的校验对象）；
#   4. 活跃 process_bindings 的悬空 key 必须为 0（与 seeder 的 verifyWorkflowTemplates 同一谓词）；
#   5. 基线租户确实播种到了流程绑定（>0），防止整段 skip 伪装成功。
#
# 用法：
#   scripts/fresh-install-gate.sh              # 构建镜像 + 空卷跑完并拆除
#   GATE_BUILD=0 scripts/fresh-install-gate.sh # 复用已构建镜像（本地调试提速）
#   GATE_KEEP=1  scripts/fresh-install-gate.sh # 失败后保留容器与卷供排查
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

GATE_RUN_ID="${GATE_RUN_ID:-$$}"
export GATE_RUN_ID          # docker-compose.fresh-gate.yml 需要读到它来唯一化容器名
PROJECT="${GATE_PROJECT:-itsm-gate-${GATE_RUN_ID}}"
COMPOSE=(docker compose -p "$PROJECT" -f docker-compose.dev.yml -f docker-compose.fresh-gate.yml)
# GATE_EXTRA_OVERRIDE 只用于注入「故意破坏首装」的夹具，证明本门禁真的会红（见 scripts/
# fixtures/fresh-gate-negative.override.yml）。正常运行不需要它。
if [ -n "${GATE_EXTRA_OVERRIDE:-}" ]; then COMPOSE+=(-f "$GATE_EXTRA_OVERRIDE"); fi
MIN_MIGRATIONS="${MIN_MIGRATIONS:-50}"
GATE_BUILD="${GATE_BUILD:-1}"
GATE_KEEP="${GATE_KEEP:-0}"
ARTIFACT_DIR="${GATE_ARTIFACT_DIR:-reports/fresh-install-gate/$(date -u +%Y-%m-%d)-run-${GATE_RUN_ID}}"

log() { printf '[fresh-install-gate] %s\n' "$*"; }
fail() {
  log "FAIL: $*"
  mkdir -p "$ARTIFACT_DIR"
  "${COMPOSE[@]}" logs --no-color >"$ARTIFACT_DIR/compose.log" 2>&1 || true
  log "容器日志已存到 $ARTIFACT_DIR/compose.log"
  exit 1
}

cleanup() {
  if [ "$GATE_KEEP" = "1" ]; then
    log "GATE_KEEP=1：保留 project=$PROJECT 的容器与卷"
    return
  fi
  "${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

# 前置检查：容器名/端口唯一化后本不应冲突，但固定 image 拉取失败与 docker 未运行是常见的静默
# 假绿来源，先显式确认。
command -v docker >/dev/null 2>&1 || fail "docker 不可用"
docker info >/dev/null 2>&1 || fail "docker daemon 未运行"

log "project=$PROJECT run=${GATE_RUN_ID}（全新命名卷，不复用任何既有数据）"
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

# 用 ${BUILD_FLAG:+--build} 而不是空数组：macOS 自带 bash 3.2 在 set -u 下展开空数组会直接
# 报 unbound variable 中断脚本（本地实测踩过），而 CI 的 bash 5 不会——两边行为必须一致。
BUILD_FLAG=""
if [ "$GATE_BUILD" = "1" ]; then BUILD_FLAG=--build; fi

log "启动 postgres + redis 并等待健康"
# shellcheck disable=SC2086
"${COMPOSE[@]}" up -d --wait ${BUILD_FLAG:+--build} postgres redis \
  || fail "postgres/redis 未能在健康时间内就绪（镜像拉取或 compose 配置问题）"

log "执行首装（migrate + seed），退出码必须为 0"
mkdir -p "$ARTIFACT_DIR"
INIT_LOG="$(mktemp)"
if ! "${COMPOSE[@]}" up --abort-on-container-exit --exit-code-from itsm-init itsm-init \
  >"$INIT_LOG" 2>&1; then
  cp "$INIT_LOG" "$ARTIFACT_DIR/itsm-init.log"
  tail -40 "$ARTIFACT_DIR/itsm-init.log" || true
  fail "itsm-init 在空卷上首装失败（详见 $ARTIFACT_DIR/itsm-init.log）"
fi
cp "$INIT_LOG" "$ARTIFACT_DIR/itsm-init.log"
rm -f "$INIT_LOG"
log "itsm-init 退出 0"

psql_scalar() {
  local sql="$1"
  "${COMPOSE[@]}" exec -T postgres psql -U itsm_user -d itsm -tAc "$sql" | tr -d '[:space:]'
}

MIGRATIONS="$(psql_scalar 'SELECT count(*) FROM schema_migrations')"
[ -n "$MIGRATIONS" ] || fail "读取 schema_migrations 失败"
[ "$MIGRATIONS" -ge "$MIN_MIGRATIONS" ] || fail "迁移入账 $MIGRATIONS < 下限 $MIN_MIGRATIONS"

# 全新安装会执行每一个磁盘迁移，因此除「无条件收养的历史条目」外不该有空 checksum。
# 001-006 是 Go 注册表之前由 ent 基线/手工生效的历史版本，migration/legacy_record.go
# 有意只登记不执行、也不带 checksum；其余（2026* 日期化与 add_missing_indexes* 等未日期化
# 的磁盘脚本）在首装里必须真实执行并记账。空 checksum 正是「脚本被登记却从未验证」的形态。
NEVER_RUN="$(psql_scalar "SELECT count(*) FROM schema_migrations WHERE checksum = '' AND version !~ '^00[1-6]_'")"
[ "$NEVER_RUN" = "0" ] || fail "账本里有 $NEVER_RUN 条磁盘迁移 checksum 为空；首装应当全部真实执行过（历史收养条目 001-006 除外）"

DEFS="$(psql_scalar 'SELECT count(*) FROM process_definitions WHERE is_active = true')"
[ "${DEFS:-0}" -gt 0 ] || fail "process_definitions 没有活跃模板；BPMN 模板未部署，workflow-core 必然校验失败"

BINDINGS="$(psql_scalar 'SELECT count(*) FROM process_bindings WHERE is_active = true')"
[ "${BINDINGS:-0}" -gt 0 ] || fail "活跃 process_bindings 为 0；流程绑定整段未播种（skip 伪装成功）"

DANGLING="$(psql_scalar 'SELECT count(*) FROM process_bindings b LEFT JOIN process_definitions d ON d.key = b.process_definition_key AND d.tenant_id = b.tenant_id AND d.is_active = true WHERE b.is_active = true AND d.id IS NULL')"
[ "$DANGLING" = "0" ] || fail "有 $DANGLING 条活跃流程绑定指向未部署的流程定义（首装会在这条不变量上回滚）"

# 运营性索引/序列：不属于 ent schema，但承担正确性约束（事件编号唯一、SLA 违规防重）。
# 这组 DDL 曾挂在数据库连接路径上——空卷首装时表还不存在，四条语句全部失败并被
# "non-fatal" 日志吞掉且不再重试，于是全新安装永久缺这三个索引（2026-10-27 空库首装
# 与生产库对账实测）。创建点已移到 InitializeStorage 末尾，这里断言它们真的存在，
# 而不是「日志说建了」。
for obj in sla_violation_unique_open_idx sla_violation_ticket_type_idx incidents_tenant_number_idx; do
  PRESENT="$(psql_scalar "SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname = '$obj'")"
  [ "$PRESENT" = "1" ] || fail "运营性索引 $obj 缺失；首装环境的防重约束不成立"
done
SEQ="$(psql_scalar "SELECT count(*) FROM pg_class WHERE relkind = 'S' AND relname = 'incident_number_seq'")"
[ "$SEQ" = "1" ] || fail "incident_number_seq 缺失；事件编号会退回 COUNT+1 并发复用"

# 索引撞车：同一张表上不允许存在两份「列、谓词、排序方向完全相同」的索引。
# 判据不能只看列名——列相同但排序方向或谓词不同的两条索引服务不同查询，合并会退化。
# 唯一性刻意不参与判据：唯一 btree 能服务非唯一 btree 的全部查找，所以「唯一 + 非唯一」
# 同列同序也是纯冗余（008_add_initialization_ledger 的 UNIQUE(run_id,component,attempt)
# 配一条同列 CREATE INDEX 就是这种情况）。
# 这类重复的成本是双写两棵 B 树 + 规划器候选集翻倍，且不会有任何功能报错，只能靠对账
# 发现（2026-10-27 生产库实测 32 对完全重复 + 1 对唯一覆盖）。收敛动作在
# database.EnsureIndexConvergence（InitializeStorage 末尾）执行——不能写成日期化迁移，
# 因为无日期的 add_missing_indexes* 按字符串排序落在其后，会把删掉的索引又建回来。
# 此处守住收敛后的结果。
DUP_INDEXES="$(psql_scalar "
WITH idx AS (
  SELECT i.indrelid::regclass::text AS tbl, c.relname AS idx,
         regexp_replace(pg_get_indexdef(i.indexrelid),
                        '^CREATE (UNIQUE )?INDEX (\S+) ON (\S+) ', 'CREATE INDEX ON \3 ') AS sem,
         i.indoption::text AS opt, i.indisunique AS uniq
  FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
  WHERE i.indrelid IN (SELECT oid FROM pg_class
                       WHERE relkind = 'r' AND relnamespace = 'public'::regnamespace)
    AND NOT i.indisprimary
)
SELECT count(*) FROM idx a JOIN idx b
  ON a.tbl = b.tbl AND a.idx < b.idx
WHERE a.sem = b.sem AND a.opt = b.opt")"
if [ "${DUP_INDEXES:-0}" != "0" ]; then
  psql_scalar "
WITH idx AS (
  SELECT i.indrelid::regclass::text AS tbl, c.relname AS idx,
         regexp_replace(pg_get_indexdef(i.indexrelid),
                        '^CREATE (UNIQUE )?INDEX (\S+) ON (\S+) ', 'CREATE INDEX ON \3 ') AS sem,
         i.indoption::text AS opt, i.indisunique AS uniq
  FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
  WHERE i.indrelid IN (SELECT oid FROM pg_class
                       WHERE relkind = 'r' AND relnamespace = 'public'::regnamespace)
    AND NOT i.indisprimary
)
SELECT a.tbl || ': ' || a.idx || ' (unique=' || a.uniq || ') == '
       || b.idx || ' (unique=' || b.uniq || ')' FROM idx a JOIN idx b
  ON a.tbl = b.tbl AND a.idx < b.idx
WHERE a.sem = b.sem AND a.opt = b.opt
ORDER BY a.tbl" >&2 || true
  fail "首装库存在 $DUP_INDEXES 对重复索引（见上方清单）；重复索引只增加写放大，不提供任何查询能力"
fi

log "PASS: 迁移 $MIGRATIONS 条（空 checksum 0）、活跃流程定义 ${DEFS}、活跃流程绑定 ${BINDINGS}（悬空 ${DANGLING}）、运营性索引/序列 4/4、重复索引 0 对"
log "空卷首装链路与关键不变量成立"
