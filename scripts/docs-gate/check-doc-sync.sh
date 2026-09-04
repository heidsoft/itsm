#!/usr/bin/env bash
#
# scripts/docs-gate/check-doc-sync.sh
#
# Docs Gate C.5：代码 <-> 文档同步新鲜度检查。
#
# 检查项（可自动化验证的一致性信号）：
#   1. README / README.en 引用的 make 目标必须在 Makefile 中存在
#      （防止文档引导用户执行不存在的命令）。
#   2. ROADMAP.md 的 "Last synced" 日期落后最近一次提交超过 14 天
#      （防止路线图与代码迭代脱节）。
#   3. CHANGELOG.md 必须保留 [Unreleased] 段
#      （发布时才允许把它折叠成版本号）。
#   4. docs/documentation-governance.md 的 "最后审查" 日期超过 90 天
#      （防止治理规则本身过期）。
#
# 用法：
#   ./scripts/docs-gate/check-doc-sync.sh           # advisory 模式
#   ./scripts/docs-gate/check-doc-sync.sh --strict  # hard 模式
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

STRICT="${1:-}"
FAILS=()

fail() {
  FAILS+=("$1")
  echo "FAIL: $1"
}

pass() {
  echo "PASS: $1"
}

# ---------- 1. README 引用的 make 目标必须存在 ----------
check_make_targets() {
  local file="$1"
  [ -f "$file" ] || return 0
  # 提取 `make <target>`（排除 make install/依赖安装类的通配），逐一核对 Makefile
  local targets
  targets=$(grep -oE 'make [a-z][a-z0-9_-]+' "$file" | awk '{print $2}' | sort -u)
  local available
  available=$(grep -E '^[a-zA-Z_-]+:' Makefile | sed 's/:.*//' | sort -u)
  for t in ${targets}; do
    if grep -qx "$t" <<< "$available"; then
      pass "$file: make ${t} 存在于 Makefile"
    else
      fail "$file 引用了 Makefile 中不存在的目标: make ${t}"
    fi
  done
}

check_make_targets README.md
check_make_targets README.en.md

# ---------- 2. ROADMAP Last synced 新鲜度 ----------
if [ -f ROADMAP.md ]; then
  synced=$(grep -m1 -oE 'Last synced: [0-9]{4}-[0-9]{2}-[0-9]{2}' ROADMAP.md | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}')
  if [ -z "$synced" ]; then
    fail "ROADMAP.md 缺少 'Last synced: YYYY-MM-DD' 声明"
  else
    last_commit_date=$(git log -1 --format=%cd --date=format:%Y-%m-%d 2>/dev/null)
    if [ -n "$last_commit_date" ]; then
      synced_ts=$(date -j -f '%Y-%m-%d' "$synced" +%s 2>/dev/null)
      commit_ts=$(date -j -f '%Y-%m-%d' "$last_commit_date" +%s 2>/dev/null)
      if [ -n "$synced_ts" ] && [ -n "$commit_ts" ]; then
        age_days=$(( (commit_ts - synced_ts) / 86400 ))
        if [ "$age_days" -gt 14 ]; then
          fail "ROADMAP Last synced (${synced}) 落后最近提交 (${last_commit_date}) ${age_days} 天，超过 14 天阈值"
        else
          pass "ROADMAP Last synced (${synced}) 与最近提交 (${last_commit_date}) 同步（落后 ${age_days} 天）"
        fi
      fi
    fi
  fi
else
  fail "ROADMAP.md 不存在"
fi

# ---------- 3. CHANGELOG [Unreleased] 段 ----------
if [ -f CHANGELOG.md ]; then
  if grep -q '^## \[Unreleased\]' CHANGELOG.md; then
    pass "CHANGELOG.md 保留 [Unreleased] 段"
  else
    fail "CHANGELOG.md 缺少 [Unreleased] 段（发布后也必须保留空段）"
  fi
else
  fail "CHANGELOG.md 不存在"
fi

# ---------- 4. 治理文档审查新鲜度 ----------
GOV="docs/documentation-governance.md"
if [ -f "$GOV" ]; then
  reviewed=$(grep -m1 -oE '最后审查：[0-9]{4}-[0-9]{2}-[0-9]{2}' "$GOV" | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}')
  if [ -z "$reviewed" ]; then
    fail "$GOV 缺少 '最后审查：YYYY-MM-DD' 声明"
  else
    now_ts=$(date +%s)
    reviewed_ts=$(date -j -f '%Y-%m-%d' "$reviewed" +%s 2>/dev/null)
    if [ -n "$reviewed_ts" ]; then
      age_days=$(( (now_ts - reviewed_ts) / 86400 ))
      if [ "$age_days" -gt 90 ]; then
        fail "$GOV 最后审查（${reviewed}）已 ${age_days} 天未复审，超过 90 天阈值"
      else
        pass "$GOV 最后审查（${reviewed}）新鲜度正常（${age_days} 天）"
      fi
    fi
  fi
else
  fail "$GOV 不存在"
fi

# ---------- 结果 ----------
echo ""
if [ "${#FAILS[@]}" -gt 0 ]; then
  echo "check-doc-sync: ${#FAILS[@]} 项不一致"
  if [ "$STRICT" = "--strict" ]; then
    exit 1
  fi
  echo "[advisory] 仅报告，不阻断。"
fi
exit 0
