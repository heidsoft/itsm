#!/usr/bin/env bash
#
# scripts/docs-gate/check-product-drift.sh
#
# Gate C.6 — 产品口径漂移守卫（口径不一致 = 构建失败）
#
# 为什么有这条门禁
# ----------------
# 2026-09-22 审计（output/product-drift-overdesign-audit-2026-09-22.md）的核心规律是：
#   **有机器守卫的平面不漂移，只靠文档约定的平面必然漂移。**
# 授权平面有 5 道守卫，于是零漂移；而成熟度口径、领域清单、NON-GOALS、表面规模、
# 覆盖率口径这些"只写在文档里"的平面，5 项全部漂移。
# 所以治漂移的办法不是"下次注意"，而是把口径不一致变成构建失败 —— 就是本脚本。
#
# 检查项
# ------
#   C.6.1 成熟度口径一致性：README 成熟度表 ↔ 能力契约成熟度表（双向闭合）
#         可用 ↔ GA 候选/GA 核心；预览 ↔ Pilot/Disabled。缺映射、缺对应行也算漂移。
#   C.6.2 AGENTS.md 领域清单：若硬编码列举领域，则每个都必须真实存在（防指向幽灵域）
#   C.6.3 handler 域包接线：handlers/<domain>/ 要么被生产代码引用、要么删除。
#         这是 8 月评审 R2（"新层零路由"）的同类守卫 —— 当时只修了 incident 实例，
#         没有加守卫，于是 department/root_cause/dashboard 以同样形态复发。
#   C.6.4 产品表面棘轮：页面数/域包数/service 文件数/装配行数只降不升（基线可显式上调）
#   C.6.5 覆盖率口径披露：jest 只统计 src/lib 时，ROADMAP 必须显式披露该 scope
#
# 豁免（waiver）
# -------------
# 存量漂移想放行，必须在 product-drift-waivers.txt 登记 owner + 到期日 + 理由。
# 豁免到期即反向 FAIL —— 防止豁免变成永久沉默。豁免是债务登记，不是解决方案。
#
# 模式
# ----
# 默认 hard：存在 FAIL 即退出码 1。接受 `--strict`（向后兼容，等同默认）
# 与 `--advisory`（仅报告不阻断）两个参数。
#
# 用法：
#   ./scripts/docs-gate/check-product-drift.sh              # hard
#   ./scripts/docs-gate/check-product-drift.sh --advisory   # 仅报告
#
# 可覆写环境变量（供测试使用）：
#   DOCS_GATE_ROOT          被检查的仓库根（默认脚本上两级）
#   PRODUCT_DRIFT_BASELINE  基线文件路径
#   PRODUCT_DRIFT_WAIVERS   豁免文件路径
#   PRODUCT_DRIFT_TODAY     当今日期（YYYY-MM-DD），用于豁免到期判定
#

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="${DOCS_GATE_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"
BASELINE_FILE="${PRODUCT_DRIFT_BASELINE:-${SCRIPT_DIR}/product-surface-baseline.txt}"
WAIVER_FILE="${PRODUCT_DRIFT_WAIVERS:-${SCRIPT_DIR}/product-drift-waivers.txt}"
TODAY="${PRODUCT_DRIFT_TODAY:-$(date +%Y-%m-%d)}"

MODE="hard"
case "${1:-}" in
  --advisory) MODE="advisory" ;;
  --strict | "") MODE="hard" ;;
  *) MODE="hard" ;;
esac

BE="${ROOT_DIR}/itsm-backend"
FE="${ROOT_DIR}/itsm-frontend"

FAILS=0
WARNS=0
WAIVED=0

trim() { printf '%s' "$1" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'; }

fail() {
  FAILS=$((FAILS + 1))
  echo "  FAIL: $1"
}
pass() { echo "  PASS: $1"; }
warn() {
  WARNS=$((WARNS + 1))
  echo "  WARN: $1"
}
skip() { echo "  SKIP: $1"; }

# waived <rule> <subject> —— 命中且未过期返回 0；命中但已过期返回 1 并记一次 FAIL
waived() {
  local rule="$1" subject="$2"
  [ -f "${WAIVER_FILE}" ] || return 1
  local r s owner exp reason
  while IFS='|' read -r r s owner exp reason; do
    r="$(trim "${r:-}")"
    [ -z "${r}" ] && continue
    case "${r}" in \#*) continue ;; esac
    [ "${r}" = "${rule}" ] || continue
    s="$(trim "${s:-}")"
    [ "${s}" = "${subject}" ] || continue
    exp="$(trim "${exp:-}")"
    if [ -n "${exp}" ] && [[ "${exp}" < "${TODAY}" ]]; then
      fail "豁免已过期（${exp}）: ${rule}/${subject} —— 债务未在期限内解决，请修掉或重新登记"
      return 1
    fi
    echo "  WAIVED: ${rule}/${subject} (owner=$(trim "${owner:-}"), expires=${exp:-N/A})"
    WAIVED=$((WAIVED + 1))
    return 0
  done < "${WAIVER_FILE}"
  return 1
}

count_handlers_dirs() {
  [ -d "${BE}/handlers" ] || {
    echo 0
    return 0
  }
  ls -d "${BE}/handlers"/*/ 2>/dev/null | wc -l | tr -d ' '
}

# ---------------------------------------------------------------------------
# C.6.1 成熟度口径一致性
# ---------------------------------------------------------------------------

readme_maturity_rows() {
  awk '
    /^## 能力与成熟度/ { inb = 1; next }
    inb && /^## / { exit }
    inb && /^\|/ {
      n = split($0, a, "|")
      if (n < 4) next
      name = a[2]; state = a[3]
      gsub(/^[ \t]+|[ \t]+$/, "", name)
      gsub(/^[ \t]+|[ \t]+$/, "", state)
      if (name == "" || state == "") next
      if (name ~ /^:?-+:?$/) next
      if (name == "能力域") next
      print name "\t" state
    }
  ' "${ROOT_DIR}/README.md"
}

contract_maturity_rows() {
  awk '
    /^\| 能力域 \| 当前成熟度/ { inb = 1; next }
    inb && /^## / { exit }
    inb && /^\|/ {
      n = split($0, a, "|")
      if (n < 4) next
      name = a[2]; state = a[3]
      gsub(/^[ \t]+|[ \t]+$/, "", name)
      gsub(/^[ \t]+|[ \t]+$/, "", state)
      if (name == "" || state == "") next
      if (name ~ /^:?-+:?$/) next
      print name "\t" state
    }
    inb && !/^\|/ && !/^[ \t]*$/ { exit }
  ' "${ROOT_DIR}/docs/product/itsm-commercial-capability-contract.md"
}

# 显式映射：README 能力域 -> 能力契约能力域
#   __SUBDOMAIN__ = README 比契约更细，契约中并入父域评定，不单独比对
#   __UNKNOWN__   = 未登记（新增能力域必须登记，否则口径无法比对）
contract_for() {
  case "$1" in
    "工单与事件") echo "工单/事件" ;;
    "工单类型与动态表单") echo "__SUBDOMAIN__" ;;
    "变更管理") echo "变更" ;;
    "问题与 Known Error") echo "问题/已知错误" ;;
    "服务目录与请求") echo "服务目录/服务请求" ;;
    "CMDB") echo "CMDB" ;;
    "CMDB 云发现") echo "__SUBDOMAIN__" ;;
    "BPMN 与审批") echo "BPMN/审批" ;;
    "SLA") echo "SLA" ;;
    "知识与 RAG") echo "知识/RAG" ;;
    "AI 辅助") echo "AI" ;;
    "通知与连接器") echo "连接器/通知" ;;
    "RBAC/多租户") echo "RBAC/多租户/审计" ;;
    "报表与运营") echo "报表/运营" ;;
    *) echo "__UNKNOWN__" ;;
  esac
}

# 状态兼容：可用 -> 契约含 GA；预览 -> 契约含 Pilot 或 Disabled
state_ok() {
  case "$1" in
    可用) case "$2" in *GA*) return 0 ;; *) return 1 ;; esac ;;
    预览) case "$2" in *Pilot* | *Disabled*) return 0 ;; *) return 1 ;; esac ;;
    *) return 1 ;;
  esac
}

check_maturity_parity() {
  echo "-- C.6.1 成熟度口径一致性（README ↔ 能力契约）"
  local rfile="${ROOT_DIR}/README.md"
  local cfile="${ROOT_DIR}/docs/product/itsm-commercial-capability-contract.md"
  if [ ! -f "${rfile}" ] || [ ! -f "${cfile}" ]; then
    skip "README.md 或能力契约不存在，跳过"
    return 0
  fi

  local tr tc
  tr="$(mktemp)"
  tc="$(mktemp)"
  readme_maturity_rows >"${tr}"
  contract_maturity_rows >"${tc}"

  if [ ! -s "${tr}" ] || [ ! -s "${tc}" ]; then
    fail "成熟度表解析失败（README 行=$(wc -l <"${tr}" | tr -d ' ')，契约行=$(wc -l <"${tc}" | tr -d ' ')）—— 表头或标题可能已变更，请同步本脚本"
    rm -f "${tr}" "${tc}"
    return 0
  fi

  local claimed="" pairs=0
  local name state cname cstate
  while IFS=$'\t' read -r name state; do
    [ -z "${name}" ] && continue
    cname="$(contract_for "${name}")"
    case "${cname}" in
      __UNKNOWN__)
        waived "C.6.1-map" "${name}" ||
          fail "README 能力域「${name}」未登记到本脚本映射表，口径无法比对（新增能力域必须登记）"
        continue
        ;;
      __SUBDOMAIN__)
        continue
        ;;
    esac
    cstate="$(awk -F'\t' -v k="${cname}" '$1 == k { print $2 }' "${tc}")"
    if [ -z "${cstate}" ]; then
      waived "C.6.1-missing" "${name}" ||
        fail "README 能力域「${name}」在能力契约中找不到对应行「${cname}」"
      continue
    fi
    claimed="${claimed}${cname}
"
    pairs=$((pairs + 1))
    if state_ok "${state}" "${cstate}"; then
      pass "「${name}」README=${state} / 契约=${cstate}"
    else
      waived "C.6.1" "${name}" ||
        fail "口径冲突：「${name}」README 标「${state}」，能力契约标「${cstate}」—— 二者不可能同时为真"
    fi
  done <"${tr}"

  # 反向闭合：契约里的能力域必须在 README 出现，否则对外承诺漏报
  while IFS=$'\t' read -r cname cstate; do
    [ -z "${cname}" ] && continue
    if ! printf '%s' "${claimed}" | grep -qxF "${cname}"; then
      waived "C.6.1-reverse" "${cname}" ||
        fail "能力契约能力域「${cname}」在 README 成熟度表中无对应行（对外承诺漏报）"
    fi
  done <"${tc}"

  if [ "${pairs}" -gt 0 ]; then
    echo "  ---- 已比对 ${pairs} 对能力域 ----"
  fi

  rm -f "${tr}" "${tc}"
}

# ---------------------------------------------------------------------------
# C.6.2 AGENTS.md 领域清单
# ---------------------------------------------------------------------------

check_agents_domain_list() {
  echo "-- C.6.2 AGENTS.md 领域清单"
  local f="${ROOT_DIR}/AGENTS.md"
  if [ ! -f "${f}" ]; then
    skip "AGENTS.md 不存在，跳过"
    return 0
  fi
  local line
  line="$(grep -m1 -E 'Existing domains:' "${f}" || true)"
  if [ -z "${line}" ]; then
    pass "AGENTS.md 未硬编码领域清单（清单以 handlers/ 目录为准，天然不会漂移）"
    return 0
  fi

  local list
  list="$(printf '%s' "${line}" | sed -n 's/.*Existing domains:[[:space:]]*//p' | sed 's/\..*$//')"
  local declared=0 missing=0 ghost=""
  local IFS=','
  local d
  for d in ${list}; do
    d="$(trim "${d}")"
    [ -z "${d}" ] && continue
    declared=$((declared + 1))
    if [ ! -d "${BE}/handlers/${d}" ]; then
      missing=$((missing + 1))
      ghost="${ghost} ${d}"
      waived "C.6.2" "${d}" ||
        fail "AGENTS.md 列出的领域「${d}」不存在于 itsm-backend/handlers/（清单已过期）"
    fi
  done
  unset IFS

  if [ "${missing}" -eq 0 ]; then
    pass "AGENTS.md 声明的 ${declared} 个领域全部存在"
  else
    echo "  ---- 幽灵领域:${ghost} ----"
  fi

  local actual
  actual="$(count_handlers_dirs)"
  if [ "${declared}" -ne "${actual}" ]; then
    warn "AGENTS.md 声明 ${declared} 个域，handlers/ 实际 ${actual} 个 —— 硬编码清单无法随代码演进，建议改为引用目录"
  fi
}

# ---------------------------------------------------------------------------
# C.6.3 handler 域包接线
# ---------------------------------------------------------------------------

check_handlers_wiring() {
  echo "-- C.6.3 handler 域包接线（零路由守卫）"
  if [ ! -d "${BE}/handlers" ]; then
    skip "itsm-backend/handlers 不存在，跳过"
    return 0
  fi
  local dir n prod refs wired=0
  for dir in "${BE}/handlers"/*/; do
    [ -d "${dir}" ] || continue
    n="$(basename "${dir}")"
    case "${n}" in common | shared) continue ;; esac

    prod="$(find "${dir}" -maxdepth 1 -name '*.go' -not -name '*_test.go' 2>/dev/null | wc -l | tr -d ' ')"
    if [ "${prod}" -eq 0 ]; then
      waived "C.6.3-empty" "${n}" ||
        fail "空域包 handlers/${n}/（无生产 .go 文件）—— 空目录不应入库"
      continue
    fi

    refs="$(grep -rl "\"itsm-backend/handlers/${n}\"" "${BE}" --include='*.go' 2>/dev/null \
      | grep -v "/handlers/${n}/" | wc -l | tr -d ' ')"
    if [ "${refs}" -eq 0 ]; then
      waived "C.6.3" "${n}" ||
        fail "零路由域包 handlers/${n}/（${prod} 个生产文件，全仓无生产代码 import）—— 要么接线，要么删除"
    else
      wired=$((wired + 1))
    fi
  done
  pass "已接线域包 ${wired} 个"
}

# ---------------------------------------------------------------------------
# C.6.4 产品表面棘轮
# ---------------------------------------------------------------------------

measure() {
  case "$1" in
    handlers_dirs) count_handlers_dirs ;;
    service_go_files) ls "${BE}/service"/*.go 2>/dev/null | wc -l | tr -d ' ' ;;
    bootstrap_app_lines)
      if [ -f "${BE}/internal/bootstrap/app.go" ]; then
        wc -l <"${BE}/internal/bootstrap/app.go" | tr -d ' '
      else
        echo ""
      fi
      ;;
    ent_schema_files) ls "${BE}/ent/schema"/*.go 2>/dev/null | wc -l | tr -d ' ' ;;
    frontend_pages) find "${FE}/src/app" -name 'page.tsx' 2>/dev/null | wc -l | tr -d ' ' ;;
    *) echo "" ;;
  esac
}

check_surface_ratchet() {
  echo "-- C.6.4 产品表面棘轮（只降不升）"
  if [ ! -f "${BASELINE_FILE}" ]; then
    skip "基线文件 ${BASELINE_FILE} 不存在，跳过"
    return 0
  fi
  local line key want got
  while IFS= read -r line; do
    line="${line%%#*}"
    line="$(trim "${line}")"
    [ -z "${line}" ] && continue
    case "${line}" in *=*) ;; *) continue ;; esac
    key="$(trim "${line%%=*}")"
    want="$(trim "${line#*=}")"
    [ -z "${key}" ] && continue
    got="$(measure "${key}")"
    if [ -z "${got}" ]; then
      warn "基线键 ${key} 无对应测量方式（脚本与基线文件不同步）"
      continue
    fi
    if [ "${got}" -gt "${want}" ]; then
      waived "C.6.4" "${key}" ||
        fail "表面增长：${key} = ${got} > 基线 ${want}（增长需在同一 commit 上调基线并说明理由）"
    elif [ "${got}" -lt "${want}" ]; then
      pass "${key} = ${got} < 基线 ${want}（已下降，建议同步下调基线）"
    else
      pass "${key} = ${got}（等于基线）"
    fi
  done <"${BASELINE_FILE}"
}

# ---------------------------------------------------------------------------
# C.6.5 覆盖率口径披露
# ---------------------------------------------------------------------------

check_coverage_scope() {
  echo "-- C.6.5 覆盖率口径披露"
  local j="${FE}/jest.config.js"
  if [ ! -f "${j}" ]; then
    skip "jest.config.js 不存在，跳过"
    return 0
  fi

  local block base covers_ui=0
  block="$(awk '/collectCoverageFrom/,/\]/' "${j}")"
  if printf '%s' "${block}" | grep -qE 'src/(app|components)'; then
    covers_ui=1
  fi
  base="$(printf '%s\n' "${block}" | grep -oE "'[^']+'" | tr -d "'" | grep -v '^!' | head -1)"
  base="${base%%/\*\**}"
  if [ -z "${base}" ]; then
    skip "无法解析 collectCoverageFrom 范围"
    return 0
  fi
  if [ "${covers_ui}" -eq 1 ]; then
    pass "jest 覆盖率范围包含 app/components，无需额外口径披露"
    return 0
  fi

  local rm="${ROOT_DIR}/ROADMAP.md"
  if [ ! -f "${rm}" ]; then
    skip "ROADMAP.md 不存在，跳过"
    return 0
  fi
  local sect
  sect="$(awk '/^## .*Key Metrics/ { f = 1; next } f && /^## / { exit } f' "${rm}")"
  if printf '%s' "${sect}" | grep -qF "${base}"; then
    pass "ROADMAP 指标章节已披露前端覆盖率口径（scope=${base}）"
  else
    waived "C.6.5" "frontend_coverage_scope" ||
      fail "jest 覆盖率仅统计 ${base}/**，但 ROADMAP 指标章节未披露该口径（80% 门槛会被误读为产品覆盖率）"
  fi
}

# ---------------------------------------------------------------------------

echo "########################################"
echo "# Gate C.6 — 产品口径漂移守卫"
echo "# root=${ROOT_DIR}"
echo "# today=${TODAY} mode=${MODE}"
echo "########################################"

check_maturity_parity
check_agents_domain_list
check_handlers_wiring
check_surface_ratchet
check_coverage_scope

echo ""
echo "########################################"
echo "# Gate C.6 Summary: ${FAILS} FAIL, ${WARNS} WARN, ${WAIVED} WAIVED"
echo "########################################"

if [ "${FAILS}" -gt 0 ]; then
  echo "::error::产品口径漂移 ${FAILS} 项。修掉，或在 scripts/docs-gate/product-drift-waivers.txt 登记 owner + 到期日。"
  if [ "${MODE}" = "advisory" ]; then
    echo "[advisory] 仅报告，不阻断。"
    exit 0
  fi
  exit 1
fi

echo "Gate C.6 passed."
exit 0
