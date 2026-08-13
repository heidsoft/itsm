#!/usr/bin/env bash
#
# scripts/static-gates/run-all.sh
#
# Stage 5 — 一键运行 5 条静态门禁：
#   5.1 禁止裸 c.JSON
#   5.2 common.Fail 2002/2004/2005 → 401/403/404 映射契约
#   5.3 前端禁用 raw fetch / axios
#   5.4 service 层 go func + context.Background 蔓延检测
#   5.5 ListResponse 分页形状契约
#
# 用法：
#   ./scripts/static-gates/run-all.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

TOTAL=0
FAILED=0
FAILED_NAMES=()

run_gate() {
  local name="$1"
  local script="$2"
  TOTAL=$((TOTAL + 1))
  echo ""
  echo "########################################"
  echo "# Static Gate ${name}"
  echo "########################################"
  if [[ ! -x "${script}" ]]; then
    chmod +x "${script}" 2>/dev/null || true
  fi
  if ! bash "${script}"; then
    FAILED=$((FAILED + 1))
    FAILED_NAMES+=("${name}")
  fi
}

run_gate "5.1 bare c.JSON"          "${ROOT_DIR}/scripts/static-gates/check-bare-json.sh"
run_gate "5.2 HTTP status mapping"  "${ROOT_DIR}/scripts/static-gates/check-http-status-mapping.sh"
run_gate "5.3 raw fetch / axios"    "${ROOT_DIR}/scripts/static-gates/check-raw-fetch.sh"
run_gate "5.4 context.Background"   "${ROOT_DIR}/scripts/static-gates/check-context-bg.sh"
run_gate "5.5 pagination shape"     "${ROOT_DIR}/scripts/static-gates/check-pagination-shape.sh"

echo ""
echo "########################################"
echo "# Static Gates Summary: ${TOTAL} total, ${FAILED} failed"
echo "########################################"
if [[ "${FAILED}" -gt 0 ]]; then
  echo "Failed gates:"
  for n in "${FAILED_NAMES[@]}"; do
    echo "  - ${n}"
  done
  exit 1
fi
echo "All static gates passed."
exit 0