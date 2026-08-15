#!/usr/bin/env bash
#
# scripts/docs-gate/check-broken-links.sh
#
# Gate C.3 — 内部 markdown 链接失效检测（advisory only）。
#
# 策略：
#   - 仅扫描 .md / .mdx 中的内部相对链接与路径锚点（以 ./ ../ / 开头的链接）。
#   - 不验证外部链接（避免外部站点波动误伤）。
#   - 使用 grep + awk 抽取 [text](path)，逐条验证 path 是否存在。
#
# 已知限制（advisory）：
#   - 跨 repo 链接（GitHub issue / PR）跳过。
#   - 文档锚点 (#section) 仅做存在性检查，不解析 heading。
#
# 当前阶段（v1.5）advisory：仅警告，不阻断。
# v2.0 起升级为 hard。
#
# 用法：
#   ./scripts/docs-gate/check-broken-links.sh [--strict]
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

STRICT="${1:-}"
VIOLATIONS=0

echo "########################################"
echo "# Gate C.3 — 内部 markdown 链接失效检测（advisory）"
echo "########################################"

TARGETS="$(git ls-files '*.md' '*.mdx' 2>/dev/null)"

# 抽取内部链接，验证存在性
check_links_in_file() {
  local file="$1"
  local dir
  dir="$(dirname "${file}")"

  # 匹配 [text](path) 与 <path> 两种形式
  # 仅保留相对路径或 ./ ../ 开头的（内部）；http(s):// 跳过
  grep -oE '\[[^]]+\]\(([^)]+)\)' "${file}" \
    | sed -E 's/^\[[^]]+\]\(//; s/\)$//' \
    | grep -E '^((\./|\.\./|/)[^h][^t][^t][^p]|[A-Za-z0-9._/-]+\.(md|mdx|md#|mdx#))' \
    | while IFS= read -r link; do
        # 去掉 #anchor
        path="${link%%#*}"
        if [ -z "${path}" ]; then
          continue
        fi
        # 解析相对路径
        if [[ "${path}" == /* ]]; then
          full="${ROOT_DIR}${path}"
        else
          full="${ROOT_DIR}/${dir}/${path}"
        fi
        # 规范化（去掉 ./ 与 ../）
        full_norm="$(cd "${dir}" 2>/dev/null && cd "$(dirname "${path}")" 2>/dev/null && pwd 2>/dev/null)/$(basename "${path}")"
        if [ ! -e "${full_norm}" ]; then
          echo "  - ${file}: link '${link}' -> ${full_norm} (missing)"
        fi
      done
}

for f in ${TARGETS}; do
  HITS="$(check_links_in_file "${f}" 2>/dev/null || true)"
  if [ -n "${HITS}" ]; then
    echo "${HITS}"
    VIOLATIONS=$((VIOLATIONS + $(echo "${HITS}" | wc -l | tr -d ' ')))
  fi
done

echo ""
echo "########################################"
echo "# Gate C.3 Summary: ${VIOLATIONS} broken link(s)"
echo "########################################"

if [ "${STRICT}" = "--strict" ]; then
  if [ "${VIOLATIONS}" -gt 0 ]; then
    echo "::error::Broken internal links detected. Fix paths or convert to external URL."
    exit 1
  fi
fi

if [ "${VIOLATIONS}" -gt 0 ]; then
  echo "[advisory] ${VIOLATIONS} potentially broken internal link(s). Review and fix."
fi
exit 0
