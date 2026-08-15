#!/usr/bin/env bash
#
# scripts/docs-gate/check-duplicate-roadmap.sh
#
# Gate C.2 — Roadmap 重复内容检测。
#
# 规则（见 docs/documentation-governance.md §维护规则 #1）：
#   - 路线图唯一事实源是仓库根 ROADMAP.md。
#   - docs/roadmap.md 仅保留跳转说明（已在 v1.0 整改）。
#   - 任何 markdown 文档不得包含完整的版本时间表、版本主题、Status 表。
#   - 允许指向 ROADMAP.md 的链接，不允许复制路线图内容。
#
# 检测方式：
#   1. 扫描所有 .md（除 ROADMAP.md 与文档治理说明），匹配 `## Release Timeline` /
#      `## 📅 Release Timeline` / `## Version Plan` / `## 🛣️ Roadmap` 等小节。
#   2. 检查每个匹配小节内是否包含多个 `v1.* / v2.* / v3.*` 版本号；若 ≥3 个
#      视为复制路线图，违规。
#
# 当前阶段（v1.5）advisory。
# v2.0 起升级为 hard。
#
# 用法：
#   ./scripts/docs-gate/check-duplicate-roadmap.sh [--strict]
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

STRICT="${1:-}"
VIOLATIONS=0

echo "########################################"
echo "# Gate C.2 — Roadmap 重复检测"
echo "########################################"

# 豁免：ROADMAP.md 本身、文档治理说明、release README（描述如何写发布报告而非路线）
EXEMPT_REGEX='(^\./ROADMAP\.md$|^./docs/documentation-governance\.md$|^./docs/roadmap\.md$|^./docs/release/README\.md$|^./docs/release/REPORT_TEMPLATE\.md$|^./AGENTS\.md$|^./CHANGELOG\.md$)'

# 收集目标文件
TARGETS="$(git ls-files '*.md' | grep -vE "${EXEMPT_REGEX}" || true)"

scan_roadmap_table() {
  local file="$1"
  # 抽取所有匹配 ## Release Timeline / 🛣️ Roadmap / Version Plan / 📅 Release Timeline 的小节内容
  awk '
    BEGIN { in_target=0; buf=""; }
    /^##[[:space:]]+(.*[Rr]oadmap|.*[Rr]elease[[:space:]]+[Tt]imeline|.*[Vv]ersion[[:space:]]+[Pp]lan|📅|🛣️)/ {
      in_target=1; buf=$0; next;
    }
    in_target && /^##[[:space:]]/ {
      in_target=0;
      # 检查是否含 ≥3 个版本号（v1.x / v2.x / v3.x）
      vcount = gsub(/v[0-9]+\.[0-9]+/, "&", buf);
      if (vcount >= 3) {
        printf("  - %s :: roadmap-like section with %d version refs\n", FILENAME, vcount);
      }
      buf="";
      next;
    }
    in_target {
      buf = buf "\n" $0;
    }
    END {
      if (in_target) {
        vcount = gsub(/v[0-9]+\.[0-9]+/, "&", buf);
        if (vcount >= 3) {
          printf("  - %s :: roadmap-like section with %d version refs\n", FILENAME, vcount);
        }
      }
    }
  ' "${file}"
}

for f in ${TARGETS}; do
  HITS="$(scan_roadmap_table "${f}" || true)"
  if [ -n "${HITS}" ]; then
    echo "${HITS}"
    VIOLATIONS=$((VIOLATIONS + $(echo "${HITS}" | wc -l | tr -d ' ')))
  fi
done

echo ""
echo "########################################"
echo "# Gate C.2 Summary: ${VIOLATIONS} violation(s)"
echo "########################################"

if [ "${STRICT}" = "--strict" ]; then
  if [ "${VIOLATIONS}" -gt 0 ]; then
    echo "::error::Duplicate roadmap-like content detected. Move to ROADMAP.md and link from docs."
    exit 1
  fi
fi

if [ "${VIOLATIONS}" -gt 0 ]; then
  echo "[advisory] Roadmap content duplicated in ${VIOLATIONS} file(s). Consider linking to ROADMAP.md."
fi
exit 0
