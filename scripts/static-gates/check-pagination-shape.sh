#!/usr/bin/env bash
#
# scripts/static-gates/check-pagination-shape.sh
#
# Stage 5.5 — 列表信封的分页形状契约。
#
# 本脚本**只做委托**，不再自带扫描器。原因（2026-10-03 实测）：同一件债务此前有两个
# 互相矛盾的判定：
#
#   1. 本脚本旧版：`grep 'type [A-Za-z]*ListResponse struct'` + 60 行文本窗口，断言每个
#      列表结构体都含 {items,total,page,pageSize,totalPages} 五元组，且永远 exit 0
#      （advisory）。
#   2. itsm-backend/tests/contract/list_envelope_ratchet_test.go：AST 扫描 dto/*.go，
#      三条棘轮基线（领域名集合键 / 分页键不完整 / 分页别名残留），随 `go test ./...`
#      在 backend-ci 里**硬失败**。
#
# 覆盖关系实测为「棘轮是旧扫描器的超集」：当前 28 个 `*ListResponse` 结构体全部含
# total 字段（棘轮的识别条件是 名称含 List 且带 total），而棘轮还额外覆盖 11 个不以
# ListResponse 结尾的信封（ListTicketsResponse/ListCIsResponse/ListProblemsResponse/
# ListAuditLogsResponse 等）——旧扫描器对它们完全失明。旧文本规则在本批删除 8 个死 DTO
# 前实测报 67 处字段级违规、删除后仍有 47 处，且永远 exit 0。
#
# 判定冲突更要紧：旧扫描器要求「所有列表都带 page/pageSize/totalPages」，与
# docs/api-reference.md 已承认并评审过的「不分页的列表」契约（`{items,total}`）直接矛盾。
# 照字面把它升成硬门禁，等于锁死一条永远无法满足的规则，只会逼人放宽阈值。
# 因此收敛方向是**删掉重复扫描器、把真相交给棘轮**，而不是给旧规则加硬。
#
# 用法：
#   ./scripts/static-gates/check-pagination-shape.sh
#

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${ROOT_DIR}/itsm-backend"

echo "==== Static Gate 5.5: list envelope / pagination shape (hard) ===="

# 1) common 的分页序列化单元测试：totalPages 必须由 NewListResponse 真实算出并序列化。
GOTOOLCHAIN=auto go test \
  -run 'TestSuccess_WithPaginationResponse|TestNewPaginationResponse|TestNewListResponse' \
  -count=1 \
  ./common/... > "${TMPDIR:-/tmp}/pagination-shape-common.log" 2>&1
common_rc=$?
tail -20 "${TMPDIR:-/tmp}/pagination-shape-common.log"
if [[ "${common_rc}" -ne 0 ]]; then
  echo "FAIL: common 分页序列化单元测试失败。"
  exit 1
fi

# 2) 列表信封棘轮：新增违规与基线过期都会失败（双向比对见测试内 diffRatchet）。
GOTOOLCHAIN=auto go test \
  -run 'TestListEnvelope' \
  -count=1 \
  ./tests/contract/... > "${TMPDIR:-/tmp}/pagination-shape-ratchet.log" 2>&1
ratchet_rc=$?
tail -30 "${TMPDIR:-/tmp}/pagination-shape-ratchet.log"
if [[ "${ratchet_rc}" -ne 0 ]]; then
  echo "FAIL: 列表信封违反 data.items 单一契约或分页键双轨（见 tests/contract/list_envelope_ratchet_test.go）。"
  exit 1
fi

# 3) 债务趋势可观察：基线条目数只减不增，收敛一条就从基线删除一条。
BASELINE_COUNTS="$(grep -hoE '^\t"[^"]*\|[^"]*"' tests/contract/list_envelope_ratchet_test.go | wc -l | tr -d ' ')"
echo ""
echo "PASS: 列表信封棘轮为绿。存量基线条目 ${BASELINE_COUNTS}（领域名集合键 + 分页键不完整 + 分页别名）。"
echo "      收敛一条 = 改 json tag 为 items 并补齐分页键，同步 Mapper/真实路由测试/前端 src/lib/api 类型与调用点，"
echo "      再从对应基线删除该条。"
exit 0
