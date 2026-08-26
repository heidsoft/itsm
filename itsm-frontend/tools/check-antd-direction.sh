#!/usr/bin/env bash
# Bug 7 回归保护：禁止前端代码继续使用 Ant Design v4/v5 的 `direction="vertical"` 写法。
# Space/Steps 在 v6 已迁移到 `orientation="vertical"`，grep 命中即视为回归。
# 此脚本与 tools/check-antd-legacy.sh 互补，专项盯 direction/orientation 切换。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$REPO_ROOT/src"

if [ ! -d "$SRC_DIR" ]; then
  echo "✗ src/ not found at $SRC_DIR"
  exit 1
fi

GREP_BIN=/usr/bin/grep
if [ ! -x "$GREP_BIN" ]; then
  GREP_BIN=$(command -v grep)
fi

FILES=$(find "$SRC_DIR" -type f \( -name '*.ts' -o -name '*.tsx' \) 2>/dev/null)
if [ -z "$FILES" ]; then
  echo "✗ No source files found under $SRC_DIR"
  exit 1
fi

# Bug 7 验收：direction="vertical" 与 direction='vertical' 必须 0 结果。
PATTERNS=(
  'direction="vertical"'
  "direction='vertical'"
)

FAIL=0
for p in "${PATTERNS[@]}"; do
  matches=$(printf '%s\n' "$FILES" | xargs "$GREP_BIN" -nF -- "$p" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    echo ""
    echo "✗ Bug 7 回归：仍存在 $p（应为 orientation=\"vertical\"）"
    echo "$matches"
    FAIL=1
  fi
done

# 反向保护：保证目标文件已经迁移到 orientation（避免有人把代码回退到无任何 vertical 设置）
ORIENTATION_FILES=(
  'src/app/(main)/admin/cab/page.tsx'
  'src/app/(main)/ai/audit/page.tsx'
  'src/app/(main)/email-intake/on-call/page.tsx'
  'src/app/(main)/admin/config-inheritance/page.tsx'
  'src/app/(main)/workflow/ticket-approval/page.tsx'
)
for f in "${ORIENTATION_FILES[@]}"; do
  full="$REPO_ROOT/$f"
  if [ -f "$full" ]; then
    if ! "$GREP_BIN" -qE "orientation=['\"]vertical['\"]" "$full"; then
      echo "✗ Bug 7 验收：$f 必须包含 orientation=\"vertical\""
      FAIL=1
    fi
  fi
done

if [ $FAIL -eq 0 ]; then
  echo "✓ Bug 7 验收：direction=\"vertical\" 已全部替换为 orientation=\"vertical\""
fi
exit $FAIL