#!/usr/bin/env bash
#
# scripts/docs-gate/run-all.sh
#
# 一键运行 docs-gate 的 6 条规则：
#   C.1 硬编码生产密码（hardcoded passwords）
#   C.2 Roadmap 重复
#   C.3 内部 markdown 链接失效（advisory）
#   C.4 发布报告无 revision 断言
#   C.5 代码 <-> 文档同步新鲜度（make 目标存在性 / ROADMAP 与 CHANGELOG 新鲜度）
#   C.6 产品口径漂移（成熟度口径一致性 / 领域清单 / 零路由域包 / 表面棘轮 / 覆盖率口径）
#
# 阻断强度（重要，勿凭注释判断，以脚本实际退出码为准）：
#   - C.6 **默认 hard**：存在 FAIL 即退出码 1，无需 --strict。
#   - C.1–C.5 **仅在传 --strict 时 hard**；不带参数时为 advisory（只报告不阻断）。
#     ✅ 2026-09-23 存量已清零：C.1 0 / C.2 0 / C.3 0（修 51 断链+加模板白名单）/
#        C.4 0（加模板白名单）/ C.5 0；CI workflow 已传 --strict 升级全 hard。
#
# 用法：
#   ./scripts/docs-gate/run-all.sh          # C.6 hard + C.1–C.5 advisory
#   ./scripts/docs-gate/run-all.sh --strict # 全部 hard（C.1–C.5 存量清零后才可用）
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
run_gate "C.6 product drift"        "${ROOT_DIR}/scripts/docs-gate/check-product-drift.sh"

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
