#!/usr/bin/env bash
#
# scripts/static-gates/check-context-bg.sh
#
# Stage 5.4 — 防止 context.Background() 蔓延到 service 层后台 goroutine。
# service 层的 go func(){...}() 必须派生自有超时的 context，否则上游请求
# 已结束但后台 goroutine 仍在跑（造成 RLS 上下文丢失、连接泄漏）。
#
# 实现：扫描 service 层下同时包含 "go func" 与 "context.Background()"
# 的文件，输出报告。该门禁当前为 advisory（不阻断构建），因为仓库已
# 积累 16 处历史用法（详见 docs/testing/static-analysis-gates.md）。当所有
# 命中都迁移到 common.WithTimeout(ctx, ...) 之后，将恢复为硬门禁。
#
# 用法：
#   ./scripts/static-gates/check-context-bg.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

TARGETS=(
  "itsm-backend/service"
)

VIOLATIONS=0
REPORT_LINES=()

for target in "${TARGETS[@]}"; do
  if [[ ! -d "${target}" ]]; then
    continue
  fi
  while IFS= read -r src; do
    [[ -z "${src}" ]] && continue
    if ! grep -qE '\bgo func' "${src}"; then
      continue
    fi
    bg_lines=$(grep -nE 'context\.Background\(\)' "${src}" 2>/dev/null || true)
    [[ -z "${bg_lines}" ]] && continue
    while IFS= read -r line; do
      [[ -z "${line}" ]] && continue
      VIOLATIONS=$((VIOLATIONS + 1))
      REPORT_LINES+=("${src}:${line}")
    done <<< "${bg_lines}"
  done < <(find "${target}" -type f -name "*.go" -not -name "*_test.go" -not -name "*_mock.go" 2>/dev/null)
done

echo "==== Static Gate 5.4: context.Background in service goroutines (advisory) ===="
if [[ "${VIOLATIONS}" -gt 0 ]]; then
  echo "WARN: ${VIOLATIONS} 处 service 层文件同时包含 'go func' 与 'context.Background()'。"
  echo "        当前为 advisory 模式（不阻断构建）。请人工确认 context.Background()"
  echo "        不在 goroutine 内被使用。"
  echo ""
  echo "Hits:"
  printf '%s\n' "${REPORT_LINES[@]}"
  echo ""
  echo "修复建议：service 层 go func 必须派生 context.WithTimeout / context.WithCancel，"
  echo "        或者使用 common.WithTimeout(ctx, 30*time.Second) 工具函数。"
  echo ""
  echo "ADV: 当所有命中都迁移完成后，本门禁将恢复为硬门禁（exit 1）。"
  exit 0
fi

echo "PASS: 没有 service 层在后台 goroutine 内裸用 context.Background() 的可疑用法。"
exit 0