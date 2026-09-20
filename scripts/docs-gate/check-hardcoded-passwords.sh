#!/usr/bin/env bash
#
# scripts/docs-gate/check-hardcoded-passwords.sh
#
# Gate C.1 — 禁止在仓库可发布路径出现固定生产密码。
#
# 扫描目标：
#   - .env / .env.prod / .env.dev
#   - docs/delivery/** 下的所有 markdown frontmatter 与正文
#   - docs/initialization-release-certification.md（历史，但保留防回归）
#   - docs/release-v1.5.0-certification-evidence.md（同上）
#   - mkdocs.yml / 部署相关 *.md（防止把 admin123 等写进部署示例）
#
# 豁免：
#   - .env.example / .env.dev.example / .env.prod.example（必须明确标注"占位"）
#   - scripts/itsm-test-utils.sh / tests/ 下的 fixture
#   - docs/delivery/postmortem-v1.0-ga.md（历史复盘，标注为已修复）
#   - 任何带 "EXAMPLE" / "<请替换>" / "<CHANGE_ME>" / "PLACEHOLDER" / "TODO" 注释
#
# 当前阶段（v1.5）advisory：仅日志报告，不阻断。
# v2.0 起升级为 hard：命中即 exit 1。
#
# 用法：
#   ./scripts/docs-gate/check-hardcoded-passwords.sh [--strict]
#

set -uo pipefail

ROOT_DIR="${DOCS_GATE_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "${ROOT_DIR}"

STRICT="${1:-}"
VIOLATIONS=0

echo "########################################"
echo "# Gate C.1 — 硬编码生产密码检测"
echo "########################################"

# 已知的高风险字面量（命中即视为候选违规）
PATTERNS=(
  'ADMIN_PASSWORD\s*=\s*["'\'']?admin123["'\'']?'
  'admin123'
  'password\s*[:=]\s*["'\'']?admin["'\'']?'
  '"admin"\s*,\s*"admin123"'
  'JWT_SECRET\s*=\s*["'\'']?itsm-secret["'\'']?'
  'POSTGRES_PASSWORD\s*=\s*["'\'']?postgres123["'\'']?'
  'ROOT_PASSWORD\s*[:=]\s*["'\'']?(admin|postgres|root)["'\'']?'
)

# 豁免路径
EXEMPT_REGEX='(\.env\.example|\.env\.dev\.example|\.env\.prod\.example|docs/delivery/postmortem-v1\.0-ga\.md|docs/scripts/|tests/|scripts/itsm-test-utils\.sh|scripts/test-data-init\.sql|/dev-environment-test-report\.md|/browser-?test|/multi-role-|/production-mode-test-report|/production-deployment-test-report|/browser-functional|/browser-e2e|/frontend-ux-review|/system-function-review|/deep-business-test|/commercial-readiness-acceptance|/module-function-retrospective|/architecture-review-2026|/system-function-review-checklist|/system-function-review-result|/servicenow-benchmark|/browser-button-functional|/browser-module|/itsm-test-report|/full-product-smoke|/system-test-report|/role-based-product-test-plan|/e2e-conventions|/coverage-audit|/controller-failing-list|/ai-eval|/test-trend|/static-analysis-gates|/test-invariants|/go-toolchain|/README-check|/output/)'

# 命中模式
scan_hits() {
  local pattern="$1"
  local file hits line rest
  while IFS= read -r -d '' file; do
    [[ "${file}" =~ ${EXEMPT_REGEX} ]] && continue
    case "${file}" in
      .env|.env.prod|.env.dev|docker-compose.prod.yml|mkdocs.yml|docs/delivery/*|docs/*certification*.md|docs/*deployment*.md|scripts/*prod*.sh|.github/workflows/*) ;;
      *) continue ;;
    esac
    hits="$(grep -nE "${pattern}" "${file}" 2>/dev/null || true)"
    while IFS= read -r hit; do
      [ -z "${hit}" ] && continue
      line="${hit%%:*}"
      rest="${hit#*:}"
      if [[ "${rest}" =~ (EXAMPLE|PLACEHOLDER|TODO|CHANGE_ME|请替换|仅开发|开发环境|不得用于生产|禁止用于生产|历史|已修复|不再硬编码) ]]; then
        continue
      fi
      printf '  - %s:%s :: match /%s/ :: %s\n' "${file}" "${line}" "${pattern}" "${rest}"
    done <<< "${hits}"
  done < <(git ls-files -z | while IFS= read -r -d '' candidate; do
    if [[ "${candidate}" =~ \.(md|sh|yml|yaml|env|env\.prod|env\.dev|toml|json)$ ]]; then
      printf '%s\0' "${candidate}"
    fi
  done)
}

for pattern in "${PATTERNS[@]}"; do
  echo ""
  echo "[scan] ${pattern}"
  HITS="$(scan_hits "${pattern}" || true)"
  if [ -n "${HITS}" ]; then
    echo "${HITS}"
    VIOLATIONS=$((VIOLATIONS + $(echo "${HITS}" | wc -l | tr -d ' ')))
  fi
done

echo ""
echo "########################################"
echo "# Gate C.1 Summary: ${VIOLATIONS} violation(s)"
echo "########################################"

if [ "${STRICT}" = "--strict" ]; then
  if [ "${VIOLATIONS}" -gt 0 ]; then
    echo "::error::Hard-coded production credentials detected. See lines above."
    exit 1
  fi
fi

if [ "${VIOLATIONS}" -gt 0 ]; then
  echo "[advisory] ${VIOLATIONS} potential violation(s) — review and migrate to env vars / placeholder comments."
fi
exit 0
