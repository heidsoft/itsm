#!/usr/bin/env bash
#
# scripts/docs-gate/check-release-claims.sh
#
# Gate C.4 — 发布报告无 revision 锚点断言检测。
#
# 规则（见 docs/release/README.md §禁止出现的表述 与 docs/documentation-governance.md §维护规则 #6）：
#   任何 release / certification / readiness 文档若出现"全部通过""立即上线"
#   "零阻断""完美""彻底解决""100% 覆盖"等无 revision 锚点的总结性断言，
#   必须保证同一段落（或紧邻上下文）出现 commit SHA + 日期 + 镜像 digest 之一。
#   缺失即视为不可签字证据。
#
# 检测方式：
#   1. 扫描 docs/release/** 与 docs/*certification* 与 docs/*release-*.md。
#   2. 匹配禁用表述列表。
#   3. 抽取同一段（前后 5 行窗口）是否含 commit SHA / 日期 / 镜像 digest。
#
# 当前阶段（v1.5）advisory：仅警告，不阻断。
# v2.0 起升级为 hard。
#
# 用法：
#   ./scripts/docs-gate/check-release-claims.sh [--strict]
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

STRICT="${1:-}"
VIOLATIONS=0

echo "########################################"
echo "# Gate C.4 — 发布报告无 revision 断言检测"
echo "########################################"

# 禁用表述（中文 + 英文）
BANNED_PHRASES=(
  '全部通过'
  '全部 OK'
  '全部完成'
  '立即上线'
  '可以发布'
  '零阻断'
  '零问题'
  '无 P0'
  '完美'
  '彻底解决'
  '100% 覆盖'
  '覆盖率达成'
  'no blocker'
  'all green'
  'ready to ship'
  'production ready'
)

# 目标文件（发布 / 认证 / readiness）
TARGETS="$(
  git ls-files '*.md' 2>/dev/null \
    | grep -E '(docs/release/|docs/.*certification|docs/.*readiness|docs/.*release-|docs/.*GA-)'
)"

# 锚点正则：commit SHA（短 7 位 / 完整）/ 日期 / 镜像 digest
ANCHOR_REGEX='(commit[[:space:]]+[0-9a-f]{7,}|[0-9a-f]{7,40}|sha256:[0-9a-f]+|20[0-9]{2}-[0-9]{2}-[0-9]{2})'

scan_claims() {
  local file="$1"
  local phrase="$2"
  awk -v phrase="${phrase}" -v anchor="${ANCHOR_REGEX}" '
    {
      line = $0
      if (line ~ phrase) {
        # 上下文窗口：前后 5 行
        win_start = (NR > 5) ? NR - 5 : 1
        win = ""
        for (i = win_start; i <= NR + 5 && i <= total_lines; i++) {
          win = win " " lines[i]
        }
        # 检查窗口内是否有 anchor
        if (win !~ anchor) {
          printf("  - %s:%d :: \"%s\" without anchor (commit/date/digest)\n", FILENAME, NR, phrase)
        }
      }
      lines[NR] = line
      total_lines = NR
    }
  ' "${file}"
}

for f in ${TARGETS}; do
  for phrase in "${BANNED_PHRASES[@]}"; do
    HITS="$(scan_claims "${f}" "${phrase}" 2>/dev/null || true)"
    if [ -n "${HITS}" ]; then
      echo "${HITS}"
      VIOLATIONS=$((VIOLATIONS + $(echo "${HITS}" | wc -l | tr -d ' ')))
    fi
  done
done

echo ""
echo "########################################"
echo "# Gate C.4 Summary: ${VIOLATIONS} claim(s) without anchor"
echo "########################################"

if [ "${STRICT}" = "--strict" ]; then
  if [ "${VIOLATIONS}" -gt 0 ]; then
    echo "::error::Release claims without commit/date/digest anchor detected. Add evidence or revise language."
    exit 1
  fi
fi

if [ "${VIOLATIONS}" -gt 0 ]; then
  echo "[advisory] ${VIOLATIONS} claim(s) lack revision anchor. See RELEASE_TEMPLATE for compliant format."
fi
exit 0
