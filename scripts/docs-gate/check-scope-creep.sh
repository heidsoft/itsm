#!/usr/bin/env bash
#
# scripts/docs-gate/check-scope-creep.sh
#
# Gate C.7 — 产品需求收敛守卫（继续扩散 = 构建失败）
#
# 为什么有这条门禁
# ----------------
# Gate C.6 治的是「口径漂移」（文档说 A、代码做 B）。但 2026-09-28 盘点暴露了
# 另一种更贵的失败：**口径没漂移，但产品面一直在变大**。
#
# 实证（2026-09-28）：
#   - C.6.4 表面棘轮 5 项全红：handlers 65>64、service 333>329、app.go 1811>1799、
#     ent schema 133>132、前端页面 169>168 —— 守卫在报警，但基线被静默突破后无人收口；
#     （当日 git 归因：五项增长全部指向契约主链路，属合法基线变更，见 §基线文件）
#   - README 成熟度 14 个能力域里 **9 个仍是「预览」**，而 ROADMAP v2.0/v3.0 已经在
#     规划服务拆分、多区域、MSP 计费、插件市场 v2、Agent 市场、移动 PWA 等 14 项新能力面。
#
# 结论：这不是"功能不够"，是"功能面铺得比能守住的宽"。C.6 管不住这种事，
# 因为它只校验「说的和做的是否一致」，不校验「该不该做」。C.7 补的就是这一层。
#
# ⚠️ 首版误判教训（务必读完再改规则）
# -----------------------------------
# 首版 C.7.1 曾检出「10 个空壳模块 / 15 个页面」，并建议"接后端或从菜单下线"。
# 逐文件读代码后确认**全部是误判**，三类原因：
#   1. 组件路径没补 `.tsx` 扩展名 —— `@/components/license/LicenseList` 解析失败，
#      于是 licenses(4 页) 被误判，实际它 import 了 `AssetApi` 是真功能；
#   2. API 信号只认 `lib/api`，漏了 `lib/services/*-service.ts` 这一层封装 ——
#      applications(`/api/v1/applications`)、tags(`/api/v1/tags`) 被误判；
#   3. 没排除**兼容重定向页** —— enterprise/settings/sla-dashboard/system/teams/
#      templates/workflows 共 7 个模块的页面只有 `redirect('/admin/...')`，
#      是刻意保留旧链接的兼容路由，删掉会让历史链接 404，不是缺陷。
# 教训：**静态检测结论在下判断前必须逐文件复核**，否则守卫会逼人删掉正确的东西。
# 修正后真实空壳 = 0；规则保留（防未来新增真空壳），但白名单已清空。
#
# 检查项
# ------
#   C.7.1 空壳前端模块零容忍：`src/app/(main)/<module>` 有 page.tsx，但模块自身与其
#         引用的 `@/components/*` 组件都找不到任何后端调用 → 空壳。存量登记在白名单
#         （owner + 到期日），**新增空壳一律 FAIL**，白名单到期未清理也 FAIL。
#   C.7.2 预览域棘轮：README 成熟度表中「预览」能力域数量只减不增。
#         新增能力域前，必须先把一个既有的「预览」转成「可用」。
#   C.7.3 规划能力面冻结：ROADMAP v2.0 + v3.0 未勾选项总数只减不增。
#         想开新能力面，先关掉一个老的。
#   C.7.4 文件级孤儿检测：`src/components/**` 与 `src/lib/api/**` 中不被任何
#         `src/app/**` 页面（含其传递引用）到达的文件。通过 BFS 从所有 app router
#         入口点（page.tsx/layout.tsx/loading.tsx/error.tsx + 根 layout + middleware）
#         出发，解析静态/动态 import 构建可达集合，报告不可达文件。首版为 advisory
#         （仅 WARN 不 FAIL），等 R5 人工复核清理后再转 blocking。
#
# 豁免（waiver）
# -------------
# 存量空壳想放行，必须在 scope-shell-whitelist.txt 登记 owner + 到期日 + 理由。
# 与 C.6 同一哲学：**豁免是债务登记，不是解决方案**；到期日次日立即反向 FAIL。
#
# 模式
# ----
# 默认 hard：存在 FAIL 即退出码 1。接受 `--strict`（等同默认）与 `--advisory`（仅报告）。
#
# 用法：
#   ./scripts/docs-gate/check-scope-creep.sh              # hard
#   ./scripts/docs-gate/check-scope-creep.sh --advisory   # 仅报告
#
# 可覆写环境变量（供测试使用）：
#   ROOT_DIR / FRONTEND_DIR / BASELINE_FILE / WHITELIST_FILE
#
set -uo pipefail

MODE="hard"
for arg in "$@"; do
  case "$arg" in
    --advisory) MODE="advisory" ;;
    --strict)   MODE="hard" ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${ROOT_DIR:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
FRONTEND_DIR="${FRONTEND_DIR:-$ROOT_DIR/itsm-frontend}"
BASELINE_FILE="${BASELINE_FILE:-$SCRIPT_DIR/product-scope-baseline.txt}"
WHITELIST_FILE="${WHITELIST_FILE:-$SCRIPT_DIR/scope-shell-whitelist.txt}"

FAILS=0
WARNS=0
WAIVED=0
TODAY="$(date +%Y-%m-%d)"

say()  { printf '%s\n' "$*"; }
fail() { FAILS=$((FAILS+1)); printf '  FAIL: %s\n' "$*"; }
pass() { printf '  PASS: %s\n' "$*"; }
warn() { WARNS=$((WARNS+1)); printf '  WARN: %s\n' "$*"; }

# 读取基线（key=value，忽略注释）
baseline_get() {
  local key="$1" default="${2:-}"
  [ -f "$BASELINE_FILE" ] || { printf '%s' "$default"; return; }
  local v
  v="$(grep -E "^${key}=" "$BASELINE_FILE" 2>/dev/null | tail -1 | cut -d= -f2 | cut -d'#' -f1 | tr -d ' ')"
  printf '%s' "${v:-$default}"
}

# 白名单查找：返回 "owner|expires" 或空
whitelist_lookup() {
  local subject="$1"
  [ -f "$WHITELIST_FILE" ] || return
  grep -vE '^\s*(#|$)' "$WHITELIST_FILE" 2>/dev/null | awk -F'|' -v s="$subject" '
    $1==s { print $2 "|" $3; exit }'
}

say "########################################"
say "# Gate C.7 — 产品需求收敛守卫"
say "# root=$ROOT_DIR"
say "# today=$TODAY mode=$MODE"
say "########################################"

# ---------------------------------------------------------------- C.7.1
say "-- C.7.1 空壳前端模块零容忍（有页面、无后端调用）"

SHELL_MODULES=""
if [ -d "$FRONTEND_DIR/src/app/(main)" ]; then
  SHELL_MODULES="$(python3 - "$FRONTEND_DIR" <<'PYEOF'
import os, re, sys
fe = sys.argv[1]
main = os.path.join(fe, "src/app/(main)")
comp_root = os.path.join(fe, "src/components")
# 注意：除 api 层外还要认 `lib/services/*-service.ts` 这层封装，
# 以及组件里常见的 `xxxApi` 命名（首版漏掉这两类，导致大量误判）。
API = re.compile(r"/api/v1|lib/api|lib/services|apiClient|[a-zA-Z]+Api\b|useMutation|useQuery|axios|fetch\(|request\(")
# 旧链接兼容重定向：只有 redirect(...)/router.replace(...) 且本身无后端调用的页面，
# 是刻意保留的历史路由，不是空壳（首版没排除这一类型）。
REDIRECT = re.compile(r"\bredirect\(|router\.replace\(|useRouter\(\)[^;]*replace")

def read(p):
    try:
        with open(p, encoding="utf-8", errors="ignore") as f:
            return f.read()
    except Exception:
        return ""

def collect_files(path):
    """解析 @/components/X：可能是 X.tsx / X.ts / 目录 X/ / X/index.tsx。"""
    cands = [path, path + ".tsx", path + ".ts",
             os.path.join(path, "index.tsx"), os.path.join(path, "index.ts")]
    out = []
    for c in cands:
        if os.path.isfile(c):
            out.append(c)
        elif os.path.isdir(c):
            for r, _, fs in os.walk(c):
                for f in fs:
                    if f.endswith((".tsx", ".ts")):
                        out.append(os.path.join(r, f))
    return out

def has_api(path):
    return any(API.search(read(f)) for f in collect_files(path))

def components_of(mod):
    out = set()
    base = os.path.join(main, mod)
    for r, _, fs in os.walk(base):
        for f in fs:
            if f.endswith((".tsx", ".ts")):
                for m in re.finditer(r"from\s+['\"]@/components/([^'\"]+)['\"]", read(os.path.join(r, f))):
                    out.add(m.group(1))
    return out

def page_files(mod):
    base = os.path.join(main, mod)
    return [os.path.join(r, f) for r, _, fs in os.walk(base) for f in fs if f == "page.tsx"]

bad = []
if os.path.isdir(main):
    for mod in sorted(os.listdir(main)):
        p = os.path.join(main, mod)
        if not os.path.isdir(p):
            continue
        pages = page_files(mod)
        if not pages:
            continue
        # 全部页面都是"兼容重定向" → 合法的历史路由兼容层，不计为空壳
        if all((REDIRECT.search(read(f)) and not API.search(read(f))) for f in pages):
            continue
        if has_api(p):
            continue
        if any(has_api(os.path.join(comp_root, c)) for c in components_of(mod)):
            continue
        bad.append(mod)
print("\n".join(bad))
PYEOF
)"
fi

shell_count=0
if [ -n "$SHELL_MODULES" ]; then
  shell_count="$(printf '%s\n' "$SHELL_MODULES" | grep -c .)"
fi
say "  -- 检出空壳模块 ${shell_count} 个"

if [ "$shell_count" -gt 0 ]; then
  while IFS= read -r mod; do
    [ -n "$mod" ] || continue
    entry="$(whitelist_lookup "$mod")"
    if [ -z "$entry" ]; then
      fail "新增空壳模块「${mod}」：菜单可见但无任何后端调用。二选一：接后端，或从菜单下线。"
    else
      owner="${entry%%|*}"
      expires="${entry##*|}"
      if [ "$TODAY" \> "$expires" ]; then
        fail "空壳模块「${mod}」白名单已于 ${expires} 到期（owner=${owner}）：必须接后端或从菜单下线，不再续期。"
      else
        WAIVED=$((WAIVED+1))
        printf '  WAIVED: 空壳模块「%s」（owner=%s, 到期=%s）已登记为存量债务\n' "${mod}" "${owner}" "${expires}"
      fi
    fi
  done <<< "$SHELL_MODULES"
else
  pass "无空壳前端模块"
fi

# ---------------------------------------------------------------- C.7.2
say "-- C.7.2 预览域棘轮（README 成熟度表「预览」数量只减不增）"

README="${ROOT_DIR}/README.md"
preview_now=0
if [ -f "$README" ]; then
  preview_now="$(awk '/^\| 能力域 \| 状态 \| 说明 \|/{flag=1;next} /^\|[: -]+\|/{next} flag&&/^\|/{n=split($0,a,"|"); st=a[3]; gsub(/^[ \t]+|[ \t]+$/,"",st); if (st=="预览") c++} flag&&!/^\|/{flag=0} END{print c+0}' "$README")"
fi
preview_base="$(baseline_get preview_domains 999999)"
say "  -- README「预览」能力域 = ${preview_now}（基线 ${preview_base}）"
if [ "$preview_now" -gt "$preview_base" ]; then
  fail "预览域增加：${preview_now} > 基线 ${preview_base}。新增能力域前，先把一个「预览」转成「可用」——只进不出就是扩散。"
else
  pass "预览域未增加（${preview_now} ≤ ${preview_base}）"
fi

# ---------------------------------------------------------------- C.7.3
say "-- C.7.3 规划能力面冻结（ROADMAP v2.0 + v3.0 未完条目只减不增）"

ROADMAP="${ROOT_DIR}/ROADMAP.md"
future_now=0
if [ -f "$ROADMAP" ]; then
  # v2.0 用 - [ ] 复选框，v3.0 是纯文本列表；两者都算「已规划但未交付的能力面」。
  # 只匹配行首无缩进的 `- `，避免把子项重复计入。
  future_now="$(awk '/^## .*(v2\.0|v3\.0)/{f=1} /^## .*(v1\.7|Always-On|📊|🤝|📜)/{f=0} f && /^- /{c++} END{print c+0}' "$ROADMAP")"
fi
future_base="$(baseline_get roadmap_future_items 999999)"
say "  -- v2.0/v3.0 未完条目 = ${future_now}（基线 ${future_base}）"
if [ "$future_now" -gt "$future_base" ]; then
  fail "未来规划条目增加：${future_now} > 基线 ${future_base}。想开新能力面，先划掉一个老的；v2.0/v3.0 在预览域清零前不得扩容。"
else
  pass "未来规划条目未增加（${future_now} ≤ ${future_base}）"
fi

# ---------------------------------------------------------------- C.7.4
say "-- C.7.4 文件级孤儿检测（src/components/** 与 src/lib/api/** 不可达文件）"

ORPHAN_HELPER="$SCRIPT_DIR/find-frontend-orphans.py"
ORPHAN_LIST=""
ORPHAN_COUNT=0
if [ -f "$ORPHAN_HELPER" ] && command -v python3 >/dev/null 2>&1; then
  ORPHAN_LIST="$(python3 "$ORPHAN_HELPER" "$FRONTEND_DIR" 2>/dev/null)" || ORPHAN_LIST=""
  if [ -n "$ORPHAN_LIST" ]; then
    ORPHAN_COUNT="$(printf '%s\n' "$ORPHAN_LIST" | grep -c .)"
  fi
fi

say "  -- 检出不可达文件 ${ORPHAN_COUNT} 个"

if [ "$ORPHAN_COUNT" -gt 0 ]; then
  # 首版设为 advisory：只报告不阻断，等 R5 人工复核清理后再转 blocking。
  # 理由：check-scope-creep.sh:22-33 的教训——静态检测假阳性会逼人删掉正确的东西。
  while IFS= read -r orphan; do
    [ -n "$orphan" ] || continue
    warn "不可达文件：${orphan}"
  done <<< "$ORPHAN_LIST"
  say "  -- C.7.4 当前为 advisory 模式（仅报告，不阻断）"
else
  pass "无文件级孤儿"
fi

# ---------------------------------------------------------------- 汇总
say "########################################"
say "# Gate C.7 Summary: ${FAILS} FAIL, ${WARNS} WARN, ${WAIVED} WAIVED"
say "########################################"

if [ "$FAILS" -gt 0 ]; then
  if [ "$MODE" = "advisory" ]; then
    say "::warning::产品需求扩散 ${FAILS} 项（advisory 模式不阻断）。"
    exit 0
  fi
  say "::error::产品需求扩散 ${FAILS} 项。收敛方案见 output/product-scope-convergence-2026-09-28.md。"
  exit 1
fi

exit 0
