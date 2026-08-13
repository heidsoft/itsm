#!/usr/bin/env bash
#
# scripts/static-gates/check-bare-json.sh
#
# Stage 5.1 — 禁止 c.JSON(...) 绕过 common.Success / common.Fail。
# 允许例外：handler_test.go / *_test.go 文件本身。
#
# 用法：
#   ./scripts/static-gates/check-bare-json.sh
# 退出码：
#   0  通过
#   非0 发现违规
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

TARGETS=(
  "itsm-backend/handlers"
  "itsm-backend/service"
  "itsm-backend/controller"
)

# 命中模式：
#   c.JSON(<digit>, <anything>)      裸 c.JSON 调用
#   c.JSON(http.Status..., <data>)    实际更可能写成 c.JSON(200, ...)，但常见
# 模式：grep -nE 'c\.JSON\([0-9]'
# 排除 *_test.go / mock* / *_mock.go
PATTERN='c\.JSON\([0-9]'
EXCLUDE_PATTERN='_test\.go$|_mock\.go$|mocks/'

VIOLATIONS=0
REPORT_LINES=()

for target in "${TARGETS[@]}"; do
  if [[ ! -d "${target}" ]]; then
    continue
  fi
  while IFS= read -r line; do
    [[ -z "${line}" ]] && continue
    # 排除测试与 mock
    if echo "${line}" | grep -qE "${EXCLUDE_PATTERN}"; then
      continue
    fi
    # 排除 // 注释行
    if echo "${line}" | grep -qE '^\s*//'; then
      continue
    fi
    VIOLATIONS=$((VIOLATIONS + 1))
    REPORT_LINES+=("${line}")
  done < <(grep -rnE "${PATTERN}" "${target}" 2>/dev/null || true)
done

echo "==== Static Gate 5.1: bare c.JSON() check ===="
if [[ "${VIOLATIONS}" -gt 0 ]]; then
  echo "FAIL: ${VIOLATIONS} bare c.JSON() call(s) found."
  echo ""
  echo "Violations:"
  printf '%s\n' "${REPORT_LINES[@]}"
  echo ""
  echo "修复建议：改用 common.Success(c, data) / common.Fail(c, code, msg) / common.SuccessWithList。"
  exit 1
fi

echo "PASS: 没有裸 c.JSON 调用，所有响应均通过 common.Success / Fail。"
exit 0