#!/usr/bin/env bash
#
# scripts/static-gates/check-http-status-mapping.sh
#
# Stage 5.2 — common.Fail 必须映射 2002/2004/2005 到 HTTP 401/403/404。
# 实现为运行 common 包的特定测试 (TestFail_Unauthorized2002 /
# TestFail_ToolPermissionDenied2004 / TestFail_UnknownTool2005)。这些测试在
# common/response_test.go 中固化映射契约。
#
# 用法：
#   ./scripts/static-gates/check-http-status-mapping.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}"

cd itsm-backend

# GOTOOLCHAIN=auto 让 go test 拉取 go.mod 要求的版本（避免本机 go 版本不够）。
GOTOOLCHAIN=auto go test \
  -run 'TestFail_Unauthorized2002|TestFail_ToolPermissionDenied2004|TestFail_UnknownTool2005|TestFailWithData_Unauthorized2002|TestFailWithData_ToolPermissionDenied2004|TestFailWithData_UnknownTool2005' \
  -count=1 \
  ./common/...

rc=$?

echo "==== Static Gate 5.2: HTTP status mapping check ===="
if [[ "${rc}" -ne 0 ]]; then
  echo "FAIL: 2002/2004/2005 → HTTP 401/403/404 映射契约被破坏。"
  echo ""
  echo "修复建议：在 common/response.go 的 Fail 与 FailWithData 的 switch 中"
  echo "        添加 UnauthorizedCode → 401、ToolPermissionDeniedCode → 403、"
  echo "        UnknownToolCode → 404 三个 case。"
  exit 1
fi

echo "PASS: 2002/2004/2005 已正确映射到 401/403/404。"
exit 0