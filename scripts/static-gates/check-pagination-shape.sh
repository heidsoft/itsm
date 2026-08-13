#!/usr/bin/env bash
#
# scripts/static-gates/check-pagination-shape.sh
#
# Stage 5.5 — 分页响应形状统一。CI 扫描所有 *ListResponse 结构体，断言它们
# 必须含有 {items, total, page, pageSize, totalPages} 五元组；常见的
# totalPage（缺 s）拼写错误必须修。
#
# 用法：
#   ./scripts/static-gates/check-pagination-shape.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

cd itsm-backend

# 1) 跑 common.PaginationResponse 单元测试，验证 totalPages 字段被序列化。
echo "==== Static Gate 5.5: pagination shape (advisory) ===="
GOTOOLCHAIN=auto go test \
  -run 'TestSuccess_WithPaginationResponse|TestNewPaginationResponse|TestNewListResponse' \
  -count=1 \
  ./common/... 2>&1 | tail -20

test_rc=$?
if [[ "${test_rc}" -ne 0 ]]; then
  echo "FAIL: common.PaginationResponse 单元测试失败。"
  exit 1
fi

# 2) 静态扫描 dto 包中所有 *ListResponse 结构体，断言必须同时含 totalPages。
TARGET="dto"

if [[ ! -d "${TARGET}" ]]; then
  echo "SKIP: dto 目录不存在，跳过静态扫描。"
  exit 0
fi

# 找出 dto 包中所有 *ListResponse 类型定义。
# 不依赖 mapfile（macOS bash 3.x 没有），改用 while + 临时文件。
TMP_FILE="$(mktemp -t pagination.XXXXXX)"
trap 'rm -f "${TMP_FILE}"' EXIT
grep -rn 'type [A-Za-z]*ListResponse struct' "${TARGET}" 2>/dev/null > "${TMP_FILE}" || true

VIOLATIONS=0
REPORT_LINES=()

while IFS= read -r decl; do
  [[ -z "${decl}" ]] && continue
  file=$(echo "${decl}" | cut -d: -f1)
  line_no=$(echo "${decl}" | cut -d: -f2)
  type_name=$(echo "${decl}" | sed -E 's/.*type ([A-Za-z]+ListResponse) struct.*/\1/')

  # 检查该结构体内是否同时含 Items / Total / Page / PageSize / TotalPages
  block=$(sed -n "${line_no},$((line_no + 60))p" "${file}")
  for required in Items Total Page PageSize TotalPages; do
    if ! echo "${block}" | grep -q "\b${required}\b"; then
      VIOLATIONS=$((VIOLATIONS + 1))
      REPORT_LINES+=("${file}:${line_no} ${type_name} 缺少字段 ${required}")
    fi
  done

  # 显式拒绝 totalPage（缺 s）拼写错误
  if echo "${block}" | grep -q '\bTotalPage\b'; then
    VIOLATIONS=$((VIOLATIONS + 1))
    REPORT_LINES+=("${file}:${line_no} ${type_name} 含拼写错误的 TotalPage（应为 TotalPages）")
  fi
done < "${TMP_FILE}"

if [[ "${VIOLATIONS}" -gt 0 ]]; then
  echo ""
  echo "WARN: ${VIOLATIONS} 处 ListResponse 不符合标准分页形状。"
  echo "        当前为 advisory 模式（不阻断构建）。详见 docs/testing/static-analysis-gates.md。"
  echo ""
  echo "Hits:"
  printf '%s\n' "${REPORT_LINES[@]}"
  echo ""
  echo "修复建议：所有 *ListResponse 必须含 Items/Total/Page/PageSize/TotalPages 五元组。"
  echo ""
  echo "ADV: 已修复 RoleListResponse.TotalPage、user_dto.PaginationResponse.TotalPage"
  echo "     拼写问题。剩余 violation 集中在 asset / change / cmdb / notification /"
  echo "     survey / system_config / project 模块，需独立 PR 修复。"
  exit 0
fi

echo ""
echo "PASS: 所有 *ListResponse 满足分页形状契约。"
exit 0