#!/usr/bin/env bash
#
# scripts/docs-gate/check-layering-convergence.sh
#
# Gate C.8 — 分层收敛守卫（service/ 架构债务只降不升）
#
# 为什么有这条门禁
# ----------------
# AGENTS.md 规定 handlers/<domain>/ 是唯一路由目标，service/ 只保留真正跨域共享
# 的逻辑。但 2026-11-07 盘点显示 service/ 仍有 174 个生产文件，42 个域没有自己的
# service.go，70+ 个 handler 文件直接 import service/ —— 迁移策略长期是「机会主义」
# 的（touch 时顺手搬），既没有时间线也没有守卫，于是边搬边漏。
# 2026-11-07 批次 1 前三批共 13 个域落位后，L.1 已由 174 收紧到 160，L.3 由
# 50 域/71 文件收紧到 38 域/59 文件；每批下降数字见 CHANGELOG [Unreleased]。
#
# 本守卫把三个平面变成构建失败，配合 plans/service-layer-migration-plan.md 的
# 四批次收口计划使用。设计原则与 C.6 一致：**有机器守卫的平面不漂移**。
#
# 检查项
# ------
#   L.1 internal/services/ 顶层生产 .go 文件**文件名棘轮**
#         2026-11-12 收口批把 service/ 顶层包（125 生产 + 207 测试）整体归位
#         internal/services/，包名仍是 service。归位目录被改名/删空时，空清单会
#         让棘轮静默 PASS —— 因此 L.1 同时做「扫描 >=100 个 .go」有效性检查，
#         L.1b 单独守「service/ 顶层零 .go」。与 C.6.4 总数棘轮互补：
#         总数棘轮挡不住「删一个加一个」，文件名棘轮能。新增生产文件一律 FAIL；
#         新增 *_test.go 不阻断；减少则 PASS 并提示用 --update-baseline 收紧。
#   L.1b service/ 顶层零 .go（归位回归锁，不看 baseline）
#   L.2 handlers/ 根目录零游离 .go 文件
#         C.6.3 只遍历 handlers/*/ 目录，看不见根目录的松散文件 ——
#         dashboard_handler.go 正是这样长期游离在领域切片之外（2026-11-07 收口）。
#         本项防止同类复发：任何 handlers/*.go 都是 FAIL。
#   L.3 handlers/<domain>/ 生产文件 import "itsm-backend/service" 按域计数棘轮
#         存量逐域清零（只允许下降）。baseline 中不存在的域若出现计数，视为
#         新增违规 FAIL —— 新域必须自带 service.go，不得回退到共享层。
#   L.4 进度报告（advisory）：域包总数 / 已有 service.go 的域 / 零 service/ 依赖的域
#   L.5 已退役遗留代码零复活：批次 1 退出时删除的 8 个零消费方实现，其文件名与
#         独有包级哨兵符号都不得在 service/ 顶层重新出现（L.1 只锁文件名，换名
#         即可绕过；L.5 锁符号）。清单见 retired-services.txt，只增不减。
#         扫描文件数 <100 或清单条目 <8 时直接 FAIL —— 零命中必须可信，不能是
#         扫描器失效造成的假绿。
#
# 用法：
#   ./scripts/docs-gate/check-layering-convergence.sh                 # hard
#   ./scripts/docs-gate/check-layering-convergence.sh --advisory     # 只报告
#   ./scripts/docs-gate/check-layering-convergence.sh --update-baseline
#         重新采集当前工作树并覆写两份 baseline。**只在存量确实下降后执行**，
#         且在 CHANGELOG [Unreleased] 记录下降数字；禁止为了「让 CI 绿」上调。
#
# 可覆写环境变量（供测试使用）：
#   DOCS_GATE_ROOT              被检查的仓库根（默认脚本上两级）
#   LAYERING_SERVICE_BASELINE   L.1 baseline 路径
#   LAYERING_IMPORT_BASELINE    L.3 baseline 路径
#
# 🔴 locale 铁律：与 check-product-drift.sh 同源，macOS BSD awk 在 UTF-8 locale 下
# 对 CJK 做 collation 比较会产生假阳性。全脚本强制 LC_ALL=C，勿改回。
#
export LC_ALL=C

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="${DOCS_GATE_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"
BE="${ROOT_DIR}/itsm-backend"
SERVICE_BASELINE="${LAYERING_SERVICE_BASELINE:-${SCRIPT_DIR}/service-prod-baseline.txt}"
IMPORT_BASELINE="${LAYERING_IMPORT_BASELINE:-${SCRIPT_DIR}/handler-service-imports-baseline.txt}"
RETIRED_LIST="${LAYERING_RETIRED_LIST:-${SCRIPT_DIR}/retired-services.txt}"

MODE="hard"
ACTION="check"
for arg in "$@"; do
  case "${arg}" in
    --advisory) MODE="advisory" ;;
    --strict | "") ;;
    --update-baseline) ACTION="update" ;;
    *) echo "unknown arg: ${arg}" >&2; exit 2 ;;
  esac
done

FAILS=0
WARNS=0

fail() {
  FAILS=$((FAILS + 1))
  echo "  FAIL: $1"
}
pass() { echo "  PASS: $1"; }
warn() {
  WARNS=$((WARNS + 1))
  echo "  WARN: $1"
}

# 归位后 internal/services/ 顶层生产文件名清单（不含 *_test.go，排序，去重）。
collect_service_prod() {
  find "${BE}/internal/services" -maxdepth 1 -name '*.go' -not -name '*_test.go' -print 2>/dev/null |
    awk -F/ '{ print $NF }' | sort -u
}

# 旧 service/ 顶层：收口批之后必须恒为空。子包（service/cloud 等）不在此列。
collect_legacy_service_go() {
  find "${BE}/service" -maxdepth 1 -name '*.go' -print 2>/dev/null | sort
}

# 每个 handlers/<domain>/ 里 import service/ 的生产文件数；根目录松散文件归入 "<root>"。
collect_import_counts() {
  # 新旧两条共享实现包路径都算依赖：只扫旧路径会让「改 import 路径」
  # 成为绕过 L.3 的通道。handlers 生产代码对该包的依赖必须恒为 0。
  grep -rl -e '"itsm-backend/service"' -e '"itsm-backend/internal/services"' "${BE}/handlers" --include='*.go' 2>/dev/null |
    grep -v '_test.go' |
    awk -F/ '
      {
        n = split($0, parts, "/")
        for (i = 1; i <= n; i++) {
          if (parts[i] == "handlers") {
            # 根层松散文件由 L.2 独占负责，这里只统计域包，避免重复报错。
            if (i + 1 < n) print parts[i + 1]
            break
          }
        }
      }
    ' | sort | uniq -c | awk '{ print $2 "=" $1 }' | sort
}

# ---------------------------------------------------------------------------
# --update-baseline：原子覆写两份 baseline
# ---------------------------------------------------------------------------
if [ "${ACTION}" = "update" ]; then
  tmp_s="$(mktemp "${TMPDIR:-/tmp}/layering-svc.XXXXXX")"
  tmp_i="$(mktemp "${TMPDIR:-/tmp}/layering-imp.XXXXXX")"

  {
    echo "# Gate C.8 / L.1 — internal/services/ 顶层生产 .go 文件名棘轮"
    echo "# 2026-11-12 收口批：service/ 顶层包整体归位 internal/services/（包名不变）。"
    echo "# 清单仍是同一批实现文件；service/ 顶层零 .go 由 L.1b 单独守。"
    echo "# 每行一个文件名（不含 *_test.go）。新增生产文件即 FAIL；存量下降后"
    echo "# 用 ./check-layering-convergence.sh --update-baseline 收紧本文件，"
    echo "# 并在 CHANGELOG [Unreleased] 记录下降数字。禁止为放行新代码上调。"
    collect_service_prod
  } >"${tmp_s}"

  {
    echo "# Gate C.8 / L.3 — handlers/<domain>/ 生产文件 import service/ 计数棘轮"
    echo "# 格式 domain=count。只允许下降；新域出现计数一律 FAIL。"
    echo "# 收紧方式同上；逐域清零进度见 plans/service-layer-migration-plan.md。"
    collect_import_counts
  } >"${tmp_i}"

  mv "${tmp_s}" "${SERVICE_BASELINE}"
  mv "${tmp_i}" "${IMPORT_BASELINE}"
  echo "baseline updated: $(grep -vc '^#' "${SERVICE_BASELINE}") service files, $(grep -vc '^#' "${IMPORT_BASELINE}") domains"
  exit 0
fi

# 读取 baseline（忽略注释与空行）。
read_baseline() {
  local f="$1"
  if [ ! -f "${f}" ]; then
    return 1
  fi
  grep -v '^#' "${f}" | grep -v '^[[:space:]]*$' || true
}

if [ ! -d "${BE}" ]; then
  echo "SKIP: ${BE} 不存在"
  exit 0
fi

echo "== Gate C.8 分层收敛守卫 =="

# ---------------------------------------------------------------------------
# L.1 service/ 生产文件名棘轮
# ---------------------------------------------------------------------------
echo "-- L.1 internal/services/ 生产文件棘轮"
if [ ! -f "${SERVICE_BASELINE}" ]; then
  fail "L.1 baseline 缺失：${SERVICE_BASELINE}（先跑 --update-baseline）"
else
  # 扫描器零命中保护：归位目录消失时棘轮会「零新增」静默 PASS。
  l1_files="$(find "${BE}/internal/services" -maxdepth 1 -name '*.go' -print 2>/dev/null | wc -l | tr -d ' ')"
  if [ "${l1_files}" -lt 100 ]; then
    fail "L.1 扫描器失效：internal/services/ 顶层只扫到 ${l1_files} 个 .go（预期 >=100），零命中不可信"
  fi
  cur_s="$(mktemp "${TMPDIR:-/tmp}/layering-cur-s.XXXXXX")"
  base_s="$(mktemp "${TMPDIR:-/tmp}/layering-base-s.XXXXXX")"
  collect_service_prod >"${cur_s}"
  read_baseline "${SERVICE_BASELINE}" | sort -u >"${base_s}"

  added="$(comm -13 "${base_s}" "${cur_s}")"
  removed="$(comm -23 "${base_s}" "${cur_s}")"
  if [ -n "${added}" ]; then
    added_count="$(printf '%s\n' "${added}" | wc -l | tr -d ' ')"
    echo "  FAIL: L.1 新增 ${added_count} 个 service/ 生产文件（业务逻辑须落 handlers/<domain>/，跨域基础设施落 internal/）："
    printf '%s\n' "${added}" | sed 's/^/         /'
    FAILS=$((FAILS + added_count))
  else
    pass "无新增生产文件（当前 $(wc -l <"${cur_s}" | tr -d ' ') 个）"
  fi
  if [ -n "${removed}" ]; then
    n_removed="$(printf '%s\n' "${removed}" | wc -l | tr -d ' ')"
    warn "L.1 存量下降 ${n_removed} 个文件 —— 请跑 --update-baseline 收紧基线并在 CHANGELOG 记录"
    printf '%s\n' "${removed}" | sed 's/^/         /'
  fi
  rm -f "${cur_s}" "${base_s}"
fi

# ---------------------------------------------------------------------------
# L.1b service/ 顶层零 .go —— 归位后任何新落盘的顶层 .go 都是把业务逻辑
# 写回共享上帝包；不看 baseline，避免「换个新文件名就绕过棘轮」。
# ---------------------------------------------------------------------------
echo "-- L.1b service/ 顶层零 .go（归位回归锁）"
legacy_go="$(collect_legacy_service_go)"
if [ -n "${legacy_go}" ]; then
  legacy_count="$(printf '%s\n' "${legacy_go}" | wc -l | tr -d ' ')"
  echo "  FAIL: L.1b service/ 顶层重新出现 ${legacy_count} 个 .go（共享实现只许在 internal/services/）："
  printf '%s\n' "${legacy_go}" | sed 's/^/         /'
  FAILS=$((FAILS + legacy_count))
else
  pass "service/ 顶层零 .go（子包不受影响）"
fi

# ---------------------------------------------------------------------------
# L.2 handlers/ 根目录零游离文件
# ---------------------------------------------------------------------------
echo "-- L.2 handlers/ 根目录零游离文件"
loose="$(find "${BE}/handlers" -maxdepth 1 -name '*.go' -print 2>/dev/null)"
if [ -n "${loose}" ]; then
  loose_count="$(printf '%s\n' "${loose}" | wc -l | tr -d ' ')"
  echo "  FAIL: L.2 handlers/ 根目录有 ${loose_count} 个游离 .go 文件（必须迁入 handlers/<domain>/；C.6.3 只扫子目录，看不见这里）："
  printf '%s\n' "${loose}" | sed 's/^/         /'
  FAILS=$((FAILS + loose_count))
else
  pass "handlers/ 根目录无 .go 文件"
fi

# ---------------------------------------------------------------------------
# L.3 按域 import service/ 计数棘轮
# ---------------------------------------------------------------------------
echo "-- L.3 handlers/ 对共享实现包的依赖棘轮（新旧路径双扫）"
if [ ! -f "${IMPORT_BASELINE}" ]; then
  fail "L.3 baseline 缺失：${IMPORT_BASELINE}（先跑 --update-baseline）"
else
  cur_i="$(mktemp "${TMPDIR:-/tmp}/layering-cur-i.XXXXXX")"
  base_i="$(mktemp "${TMPDIR:-/tmp}/layering-base-i.XXXXXX")"
  collect_import_counts >"${cur_i}"
  read_baseline "${IMPORT_BASELINE}" >"${base_i}"

  l3_fails=0
  l3_drops=0
  while IFS='=' read -r domain base_count; do
    [ -z "${domain}" ] && continue
    cur_count="$(awk -F= -v d="${domain}" '$1 == d { print $2 }' "${cur_i}")"
    cur_count="${cur_count:-0}"
    if [ "${cur_count}" -gt "${base_count}" ]; then
      fail "L.3 handlers/${domain}/ 对 service/ 的依赖 ${base_count} → ${cur_count}（只允许下降）"
      l3_fails=$((l3_fails + 1))
    elif [ "${cur_count}" -lt "${base_count}" ]; then
      l3_drops=$((l3_drops + 1))
    fi
  done <"${base_i}"

  # baseline 中没有记录的域：任何计数都是新增违规。
  while IFS='=' read -r domain cur_count; do
    [ -z "${domain}" ] && continue
    known="$(awk -F= -v d="${domain}" '$1 == d { print $2 }' "${base_i}")"
    if [ -z "${known}" ]; then
      fail "L.3 新域 handlers/${domain}/ 有 ${cur_count} 个文件 import service/ —— 新域必须自带 service.go，不得回退共享层"
      l3_fails=$((l3_fails + 1))
    fi
  done <"${cur_i}"

  if [ "${l3_fails}" -eq 0 ]; then
    total="$(awk -F= '{ s += $2 } END { print s + 0 }' "${cur_i}")"
    pass "无域依赖上升（当前合计 ${total} 个文件 / $(wc -l <"${cur_i}" | tr -d ' ') 个域）"
  fi
  if [ "${l3_drops}" -gt 0 ]; then
    warn "L.3 有 ${l3_drops} 个域依赖下降 —— 请跑 --update-baseline 收紧并在 CHANGELOG 记录"
  fi
  rm -f "${cur_i}" "${base_i}"
fi

# ---------------------------------------------------------------------------
# L.5 已退役遗留代码零复活（批次 1 退出回归锁）
# ---------------------------------------------------------------------------
echo "-- L.5 已退役遗留代码零复活"
if [ ! -f "${RETIRED_LIST}" ]; then
  fail "L.5 退役清单缺失：${RETIRED_LIST}"
else
  l5_files="$(find "${BE}/internal/services" -maxdepth 1 -name '*.go' -print 2>/dev/null | sort)"
  l5_file_count="$(printf '%s\n' "${l5_files}" | grep -c '\.go$')"
  l5_entries=0
  l5_hits=0
  if [ "${l5_file_count}" -lt 100 ]; then
    fail "L.5 扫描器失效：internal/services/ 顶层只扫到 ${l5_file_count} 个 .go（预期 >=100），零命中不可信"
    l5_hits=$((l5_hits + 1))
  else
    while IFS=' ' read -r l5_orig l5_syms l5_rest; do
      case "${l5_orig}" in '' | \#*) continue ;; esac
      [ -z "${l5_syms}" ] && continue
      l5_entries=$((l5_entries + 1))
      if [ -f "${BE}/internal/services/${l5_orig}" ]; then
        fail "L.5 退役文件 ${l5_orig} 重新出现在 internal/services/ 顶层"
        l5_hits=$((l5_hits + 1))
      fi
      IFS=',' read -r -a l5_arr <<<"${l5_syms}"
      for l5_sym in "${l5_arr[@]}"; do
        [ -z "${l5_sym}" ] && continue
        # 逐符号比对，不取并集；扫描范围只限 service/ 顶层（含测试）。
        l5_hit="$(grep -lw "${l5_sym}" ${l5_files} 2>/dev/null | head -3)"
        if [ -n "${l5_hit}" ]; then
          fail "L.5 退役符号 ${l5_sym}（来自 ${l5_orig}）在 internal/services/ 顶层复活：$(printf '%s' "${l5_hit}" | tr '\n' ' ')"
          l5_hits=$((l5_hits + 1))
        fi
      done
    done <"${RETIRED_LIST}"
  fi
  if [ "${l5_entries}" -lt 8 ]; then
    fail "L.5 清单只解析到 ${l5_entries} 条（预期 >=8）：清单被裁剪或格式损坏，零命中不可信"
    l5_hits=$((l5_hits + 1))
  fi
  if [ "${l5_hits}" -eq 0 ]; then
    pass "${l5_entries} 项退役实现零复活（扫描 ${l5_file_count} 个 internal/services/ 顶层 .go）"
  fi
fi

# ---------------------------------------------------------------------------
# L.4 进度报告（只报告，不阻断）
# ---------------------------------------------------------------------------
echo "-- L.4 分层收敛进度（advisory）"
total_domains="$(find "${BE}/handlers" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')"
with_service="$(find "${BE}/handlers" -mindepth 2 -maxdepth 2 -name 'service.go' 2>/dev/null | wc -l | tr -d ' ')"
clean_domains="$(awk -F= '{ print $1 }' <(collect_import_counts) | sort -u)"
clean_count=0
for d in $(find "${BE}/handlers" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; 2>/dev/null | sort); do
  case "${d}" in common | shared) continue ;; esac
  if ! printf '%s\n' "${clean_domains}" | grep -qx "${d}"; then
    clean_count=$((clean_count + 1))
  fi
done
echo "  域包 ${total_domains} 个 / 已有 service.go ${with_service} 个 / 零 service/ 依赖 ${clean_count} 个"
echo "  internal/services/ 生产文件 $(collect_service_prod | wc -l | tr -d ' ') 个；service/ 顶层遗留 $(collect_legacy_service_go | wc -l | tr -d ' ') 个 .go（目标 0）"

echo ""
echo "Gate C.8: ${FAILS} failures, ${WARNS} warnings (mode=${MODE})"

if [ "${FAILS}" -gt 0 ] && [ "${MODE}" = "hard" ]; then
  exit 1
fi
exit 0
