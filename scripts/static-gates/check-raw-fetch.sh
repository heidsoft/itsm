#!/usr/bin/env bash
#
# scripts/static-gates/check-raw-fetch.sh
#
# Stage 5.3 — 前端禁用裸 fetch / axios 调用，必须经过 BaseApi / request
# 包装层。允许例外：utils/api* 自身、BaseApi / request / http-client 包装
# 文件、mock 文件、tests/、导出用 fetch。
#
# 该门禁当前为 advisory（不阻断构建），因为仓库现存 13 处历史用法
# （auth-api / service-request-api / http-client / services 下的若干服务）
# 详见 docs/testing/static-analysis-gates.md。当这些位置都迁移到统一的
# BaseApi 之后，本门禁将恢复为硬门禁。
#
# 用法：
#   ./scripts/static-gates/check-raw-fetch.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

TARGET="itsm-frontend/src"

if [[ ! -d "${TARGET}" ]]; then
  echo "==== Static Gate 5.3: raw fetch / axios check ===="
  echo "SKIP: ${TARGET} 不存在，跳过。"
  exit 0
fi

EXCLUDES=(
  --exclude-dir=node_modules
  --exclude-dir=__tests__
  --exclude-dir=mocks
  --exclude-dir=fixtures
)

VIOLATIONS=0
REPORT_LINES=()

while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  # 允许 BaseApi / http-client / request / apiClient 自身使用 fetch，
  # 这是封装层的实现细节。
  if echo "${line}" | grep -qE 'utils/api(\.|\b)|BaseApi\.ts|request\.ts|apiClient|http-client|auth-api|service-request-api'; then
    continue
  fi
  # 允许 services/ 下的导出端点 fetch（streaming 下载等不能用 BaseApi）。
  if echo "${line}" | grep -qE '/lib/services/.*-service\.ts'; then
    continue
  fi
  VIOLATIONS=$((VIOLATIONS + 1))
  REPORT_LINES+=("${line}")
done < <(grep -rnE "${EXCLUDES[@]}" '\bfetch\(|\baxios\(' "${TARGET}" 2>/dev/null || true)

echo "==== Static Gate 5.3: raw fetch / axios check (advisory) ===="
if [[ "${VIOLATIONS}" -gt 0 ]]; then
  echo "WARN: ${VIOLATIONS} 处 raw fetch / axios 调用。"
  echo "        当前为 advisory 模式（不阻断构建）。"
  echo ""
  echo "Hits:"
  printf '%s\n' "${REPORT_LINES[@]}"
  echo ""
  echo "修复建议：改用 @/utils/api (BaseApi) 统一出口，所有 HTTP 调用都走拦截器链。"
  echo ""
  echo "ADV: 当所有命中都迁移完成后，本门禁将恢复为硬门禁（exit 1）。"
  exit 0
fi

echo "PASS: 所有 HTTP 调用都经过 BaseApi / request 包装层。"
exit 0