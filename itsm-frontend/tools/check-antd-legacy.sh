#!/usr/bin/env bash
# Anti-pattern guard for legacy Ant Design v4 APIs.
# Run from anywhere; resolves its own repo root.
# Exits 1 if any forbidden pattern is found in src/ ts/tsx files.

set -euo pipefail

# Locate the directory that contains this script, then walk up to src/.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$REPO_ROOT/src"

if [ ! -d "$SRC_DIR" ]; then
  echo "✗ src/ not found at $SRC_DIR"
  exit 1
fi

PATTERNS=(
  'Space direction='
  '<Tabs\.TabPane'
  'Form\.(Input|TextArea|Select|DatePicker|Radio|Checkbox)\b'
  "import \{ message \} from 'antd'"
  'import \{ message \} from "antd"'
  "import \{ notification \} from 'antd'"
  'import \{ notification \} from "antd"'
  'Modal\.(confirm|info|success|error|warning)\('
)

# Collect matching ts/tsx files once (POSIX-compatible, bash 3.2-safe).
FILES=$(find "$SRC_DIR" -type f \( -name '*.ts' -o -name '*.tsx' \) 2>/dev/null)
if [ -z "$FILES" ]; then
  echo "✗ No source files found under $SRC_DIR"
  exit 1
fi

# Use the absolute grep path to bypass any shell function aliases (macOS).
GREP_BIN=/usr/bin/grep
if [ ! -x "$GREP_BIN" ]; then
  GREP_BIN=$(command -v grep)
fi

FAIL=0
for p in "${PATTERNS[@]}"; do
  matches=$(printf '%s\n' "$FILES" | xargs "$GREP_BIN" -nE -- "$p" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    echo ""
    echo "✗ Legacy antd API found: $p"
    echo "$matches"
    FAIL=1
  fi
done

if [ $FAIL -eq 0 ]; then
  echo "✓ No legacy antd APIs detected"
fi
exit $FAIL