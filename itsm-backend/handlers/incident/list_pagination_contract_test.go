package incident

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件锁死 GET /api/v1/incidents 的分页入参语义，打在真实 handler +
// production service + Ent 仓储上（复用 list_contract_test.go 的夹具）。
//
// 修复前 handler 用 c.DefaultQuery("page","1") / c.DefaultQuery("pageSize","10")
// 自建一份默认值，把未经校验的解析结果直接下传给仓储的
// Offset((page-1)*size).Limit(size)，而响应信封又由 common.NewPaginationResponse
// 单独归一化，于是「实际读了多少行」和「声称每页多少条」是两套数字：
//   - 缺省页长 10，而平台契约（common.DefaultPageSize、docs/api-reference.md
//     通用分页说明、其余走 GetPaginationFromQuery 的端点）是 20；
//   - pageSize=5000 既不拒绝也不夹紧，Limit(5000) 把整个租户读穿，
//     响应却写着 pageSize:100，客户端据此算出的页数与实际收到的载荷不符；
//   - pageSize=0 / 非数字 → Limit(0)，读到空列表但信封按缺省页长回显，
//     界面显示「每页 20 条、共 N 条」却是 0 行；
//   - page=0/-3 → 负 offset（生产 Postgres 对此直接报错）。
//
// 收敛后分页只有一个所有者 common.GetPaginationFromQuery：缺省 20、只采纳
// 1-100、越界与非数字回落缺省、页码下界 1，且信封回显的就是实际采纳的读取参数。

// seedBulk 追加 n 条租户 A 事件，用于证明越界/非法页长确实会把整表读穿。
// 只有行数超过信封声称页长时，「Limit 生效」与「Limit 被跳过」才可区分。
func (f *listContractFixture) seedBulk(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		f.seed(t, f.tenantA, incidentSeed{number: fmt.Sprintf("INC-BULK-%03d", i)})
	}
}

// oversizedFixture 建 105 条租户 A 事件：任何一次整表读取都会返回 105 行，
// 而收敛后的读取每页最多 20 行。
func oversizedFixture(t *testing.T) *listContractFixture {
	t.Helper()

	f := newListContractFixture(t)
	f.baseSeed(t)
	f.seedBulk(t, 100)
	return f
}

func (f *listContractFixture) assertEnvelope(t *testing.T, query string, wantPage, wantPageSize, wantTotalPages float64, wantItems int) {
	t.Helper()

	data := f.listData(t, query, f.tenantA, f.agentA)

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "data.items 缺失: %v", data)

	assert.Equal(t, wantPage, data["page"], "query=%s", query)
	assert.Equal(t, wantPageSize, data["pageSize"], "query=%s", query)
	assert.Equal(t, wantTotalPages, data["totalPages"], "query=%s", query)
	assert.Len(t, items, wantItems, "query=%s：实际返回行数必须与信封声称的页长一致", query)
}

func TestList_MissingPaginationUsesPlatformDefault(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 5 条数据、缺省每页 20 → 一页装完。
	f.assertEnvelope(t, "", 1, 20, 1, 5)
}

func TestList_OversizedPageSizeCannotReadWholeTable(t *testing.T) {
	f := oversizedFixture(t)

	// 修复前：Limit(5000) 一次读穿 105 条，信封却声称 pageSize=100、totalPages=2。
	f.assertEnvelope(t, "page=1&pageSize=5000", 1, 20, 6, 20)
}

func TestList_NonNumericPageSizeFallsBackToDefault(t *testing.T) {
	f := oversizedFixture(t)

	// 修复前：strconv.Atoi 失败得到 size=0，Ent 的 Limit(0) 等于不加 LIMIT，
	// 于是「写错分页参数」比「写一个大的分页参数」更危险——整表直接被读穿。
	f.assertEnvelope(t, "pageSize=abc", 1, 20, 6, 20)
}

func TestList_ZeroAndNegativePageSizeFallBackToDefault(t *testing.T) {
	f := oversizedFixture(t)

	f.assertEnvelope(t, "pageSize=0", 1, 20, 6, 20)
	f.assertEnvelope(t, "pageSize=-1", 1, 20, 6, 20)
}

func TestList_NonPositivePageKeepsOffsetNonNegative(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 修复前：Offset((0-1)*2) 是负偏移，生产 Postgres 对此直接报错；页码下界必须是 1。
	f.assertEnvelope(t, "page=0&pageSize=2", 1, 2, 3, 2)
	f.assertEnvelope(t, "page=-3&pageSize=2", 1, 2, 3, 2)
}

func TestList_MaxPageSizeBoundaryIsAccepted(t *testing.T) {
	f := oversizedFixture(t)

	// 100 是合法上界：原样采纳，105 条数据分 2 页。
	f.assertEnvelope(t, "page=1&pageSize=100", 1, 100, 2, 100)
}

func TestList_PaginationEchoMatchesAdoptedValues(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 第 3 页每页 2 条：offset=(3-1)*2=4，5 条数据只剩最后 1 条。
	f.assertEnvelope(t, "page=3&pageSize=2", 3, 2, 3, 1)
}
