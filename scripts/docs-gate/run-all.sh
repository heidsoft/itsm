#!/usr/bin/env bash
#
# scripts/docs-gate/run-all.sh
#
# 一键运行 docs-gate 的 5 条规则：
#   C.1 硬编码生产密码（hardcoded passwords）
#   C.2 Roadmap 重复
#   C.3 内部 markdown 链接失效（advisory）
#   C.4 发布报告无 revision 断言
#   C.5 代码 <-> 文档同步新鲜度（make 目标存在性 / ROADMAP 与 CHANGELOG 新鲜度）
#
# 当前阶段（v2.0）全部 hard：缺失任意关键字段阻断构建。
# 此前（v1.5）advisory 模式已废弃；--strict 保留向后兼容但不再需要。
#
# 用法：
#   ./scripts/docs-gate/run-all.sh          # hard 模式（默认阻断）
#   ./scripts/docs-gate/run-all.sh --strict # 向后兼容，等同于默认行为
#

set -uo pipefail

ROOT_DIR="${DOCS_GATE_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "${ROOT_DIR}"

STRICT="${1:-}"

TOTAL=0
FAILED=0
FAILED_NAMES=()

run_gate() {
  local name="$1"
  local script="$2"
  TOTAL=$((TOTAL + 1))
  echo ""
  echo "########################################"
  echo "# Docs Gate ${name}"
  echo "########################################"
  if [[ ! -x "${script}" ]]; then
    chmod +x "${script}" 2>/dev/null || true
  fi
  # Pass strict flag only when set; use +"${arr[@]}" idiom to keep set -u safe.
  local extra=()
  if [ "${STRICT}" = "--strict" ]; then
    extra+=("--strict")
  fi
  if ! bash "${script}" ${extra[@]+"${extra[@]}"}; then
    FAILED=$((FAILED + 1))
    FAILED_NAMES+=("${name}")
  fi
}

run_gate "C.1 hardcoded passwords"   "${ROOT_DIR}/scripts/docs-gate/check-hardcoded-passwords.sh"
run_gate "C.2 duplicate roadmap"    "${ROOT_DIR}/scripts/docs-gate/check-duplicate-roadmap.sh"
run_gate "C.3 broken internal links" "${ROOT_DIR}/scripts/docs-gate/check-broken-links.sh"
run_gate "C.4 release claims"       "${ROOT_DIR}/scripts/docs-gate/check-release-claims.sh"
run_gate "C.5 doc sync freshness"   "${ROOT_DIR}/scripts/docs-gate/check-doc-sync.sh"

echo ""
echo "########################################"
echo "# Docs Gates Summary: ${TOTAL} total, ${FAILED} failed"
echo "########################################"

if [ "${FAILED}" -gt 0 ]; then
  echo "Failed gates:"
  for n in "${FAILED_NAMES[@]}"; do
    echo "  - ${n}"
  done
  echo ""
  echo "ERROR: Documentation quality gates failed. Fix the above violations before merging."
  exit 1
fi
echo "All docs gates passed."
