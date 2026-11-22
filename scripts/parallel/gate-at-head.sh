#!/usr/bin/env bash
# 在纯 HEAD 隔离树复跑 docs-gate，并与当前工作树对比，用于门禁红的归因。
# 用法：
#   bash scripts/parallel/gate-at-head.sh              # 只跑 C.6 产品漂移守卫（最快）
#   bash scripts/parallel/gate-at-head.sh --all        # 跑完整 docs-gate
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODE="${1:---drift}"
TMP="$(mktemp -d /tmp/itsm-head-gate-XXXXXX)"
trap 'rm -rf "$TMP"' EXIT

case "$MODE" in
  --drift) GATE="scripts/docs-gate/check-product-drift.sh" ;;
  --all)   GATE="scripts/docs-gate/run-all.sh" ;;
  *) echo "用法: $0 [--drift|--all]" >&2; exit 2 ;;
esac

HEAD_SHA="$(git -C "$ROOT" rev-parse HEAD)"
echo "== HEAD 隔离树门禁复测 =="
echo "  HEAD=$HEAD_SHA"
echo "  gate=$GATE"
echo "  隔离树=$TMP"

git -C "$ROOT" archive HEAD | tar -x -C "$TMP"
bash "$TMP/$GATE" >"$TMP/head.out" 2>&1 || true
bash "$ROOT/$GATE" >"$TMP/current.out" 2>&1 || true

# 只保留判定行，避免路径/时间戳噪音影响对比。
python3 - "$TMP/head.out" "$TMP/current.out" <<'PY'
import re, sys

def summarize(path):
    keep = []
    for line in open(path, encoding="utf-8", errors="replace"):
        if re.search(r"(PASS:|FAIL:|WARN:|Summary:|failures,|Gate C\.[0-9]+:)", line):
            keep.append(line.rstrip())
    return keep

head, cur = summarize(sys.argv[1]), summarize(sys.argv[2])
print("\n== HEAD 树判定行 ==")
print("\n".join(head) if head else "(无判定行)")
print("\n== 当前工作树判定行 ==")
print("\n".join(cur) if cur else "(无判定行)")

hs, cs = set(head), set(cur)
only_head = hs - cs
only_cur = cs - hs
if not only_head and not only_cur:
    print("\n== 结论：判定行一致，当前门禁红均可在 HEAD 基线复现（既有红） ==")
else:
    print("\n== 差异（仅当前工作树有 = 本次引入的门禁红；仅 HEAD 有 = 已被修复） ==")
    for line in sorted(only_cur):
        print("  NOW  " + line)
    for line in sorted(only_head):
        print("  HEAD " + line)
PY
