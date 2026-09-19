package approval_chain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件覆盖 GET /api/v1/approval-chains 的列表契约，走真实 handler + production
// service + Ent 仓储，而不是孤立 helper。
//
// 修复前列表响应使用 `size` json tag 且不返回 totalPages，前端因此无从判断分页；
// 同时前端发送 name/多值 status/dateRange 全部被静默丢弃（多值 status 甚至因为
// 精确匹配而返回 0 行），name 检索是页面上的可见能力但后端完全没有实现。

type listContractFixture struct {
	client  *ent.Client
	handler *Handler
	tenantA int
	tenantB int
	userA   int
}

func newListContractFixture(t *testing.T) *listContractFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:approval_chain_list_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().
		SetName("ChainA").SetCode("chain-a").SetDomain("chain-a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().
		SetName("ChainB").SetCode("chain-b").SetDomain("chain-b.test").SaveX(ctx)

	userA := client.User.Create().
		SetUsername("chain-owner-a").SetName("Chain Owner A").SetEmail("chain-a@example.com").
		SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)

	return &listContractFixture{
		client:  client,
		handler: NewHandler(service.NewApprovalChainService(client, zap.NewNop().Sugar()), zap.NewNop().Sugar()),
		tenantA: tenantA.ID,
		tenantB: tenantB.ID,
		userA:   userA.ID,
	}
}

type chainSeed struct {
	name        string
	entityType  string
	status      string
	updatedAt   time.Time
	advancedSet bool
}

func (f *listContractFixture) seed(t *testing.T, tenantID int, in chainSeed) *ent.ApprovalChain {
	t.Helper()
	ctx := context.Background()

	step := schema.ApprovalChainStep{
		Level:      1,
		ApproverID: f.userA,
		Name:       "一级审批",
		IsRequired: true,
	}
	if in.advancedSet {
		step.ApprovalType = "parallel"
		step.Threshold = 2
		step.FallbackAction = "escalate"
		step.FallbackApproverID = f.userA
		step.FallbackRole = "role:ops_manager"
		step.ConditionPriorities = []string{"high", "urgent"}
		step.ConditionAmountMin = 1000
		step.ConditionAmountMax = 5000
	}

	create := f.client.ApprovalChain.Create().
		SetName(in.name).
		SetEntityType(in.entityType).
		SetStatus(in.status).
		SetChain([]schema.ApprovalChainStep{step}).
		SetTenantID(tenantID)
	if !in.updatedAt.IsZero() {
		create.SetUpdatedAt(in.updatedAt)
	}
	return create.SaveX(ctx)
}

// doList 以 admin 角色发起列表查询，返回 HTTP status 与已解析的响应体。
func (f *listContractFixture) doList(t *testing.T, query url.Values, tenantID int) (int, map[string]interface{}) {
	t.Helper()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", f.userA)
		c.Set("role", "admin")
	})
	r.GET("/api/v1/approval-chains", f.handler.ListChains)

	target := "/api/v1/approval-chains"
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body
}

// q 构造查询参数，保证空格等字符按 URL 规范编码。
func q(kv ...string) url.Values {
	if len(kv)%2 != 0 {
		panic("q 需要成对的 key/value")
	}
	values := url.Values{}
	for i := 0; i < len(kv); i += 2 {
		values.Set(kv[i], kv[i+1])
	}
	return values
}

func (f *listContractFixture) listData(t *testing.T, query url.Values) map[string]interface{} {
	t.Helper()
	status, body := f.doList(t, query, f.tenantA)
	require.Equal(t, http.StatusOK, status, body["message"])
	require.Equal(t, float64(0), body["code"], body["message"])
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是列表信封，实际 %v", body)
	return data
}

func (f *listContractFixture) listNames(t *testing.T, query url.Values) []string {
	t.Helper()
	items, ok := f.listData(t, query)["items"].([]interface{})
	require.True(t, ok, "data.items 缺失")

	names := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		require.True(t, ok)
		name, _ := item["name"].(string)
		names = append(names, name)
	}
	return names
}

func (f *listContractFixture) baseSeed(t *testing.T) {
	t.Helper()
	f.seed(t, f.tenantA, chainSeed{name: "Network Change Approval", entityType: "change", status: "active"})
	f.seed(t, f.tenantA, chainSeed{name: "Database Access Request", entityType: "ticket", status: "active"})
	f.seed(t, f.tenantA, chainSeed{name: "Legacy Incident Chain", entityType: "incident", status: "inactive"})
	// 同名审批链存在于另一个租户，用于验证名称筛选不会跨租户命中。
	f.seed(t, f.tenantB, chainSeed{name: "Network Change Approval", entityType: "change", status: "active"})
}

func TestListChains_UsesStandardListEnvelope(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	data := f.listData(t, q())

	// 修复前响应带的是 size，且没有 totalPages；前端只能靠 items.length 猜。
	for _, key := range []string{"items", "total", "page", "pageSize", "totalPages"} {
		_, ok := data[key]
		assert.True(t, ok, "data.%s 缺失，实际 keys=%v", key, data)
	}
	assert.NotContains(t, data, "size")
	assert.NotContains(t, data, "totalCount")
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(20), data["pageSize"])
	assert.Equal(t, float64(1), data["totalPages"])
}

func TestListChains_HonorsNameFilter(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 页面上的「搜索审批链名称」在修复前被静默丢弃，这里必须真正下推到查询。
	assert.ElementsMatch(t, []string{"Network Change Approval"}, f.listNames(t, q("name", "network change")))
	assert.ElementsMatch(t, []string{"Database Access Request"}, f.listNames(t, q("name", "Database")))
	assert.Empty(t, f.listNames(t, q("name", "no-such-chain")))
	// 空白 name 视为未传，不得把列表清空。
	assert.Len(t, f.listNames(t, q("name", "")), 3)
}

func TestListChains_HonorsEntityTypeAndStatusFilters(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.ElementsMatch(t, []string{"Network Change Approval"}, f.listNames(t, q("entityType", "change")))
	assert.ElementsMatch(t, []string{"Legacy Incident Chain"}, f.listNames(t, q("status", "inactive")))
	assert.ElementsMatch(t, []string{"Database Access Request"},
		f.listNames(t, q("entityType", "ticket", "status", "active")))
	// 条件之间是 AND：entityType=change 与 entityType=incident 不得互相串味。
	assert.Empty(t, f.listNames(t, q("entityType", "change", "status", "inactive")))

	// 两个条件叠加时总数与 items 必须一致，不能只过滤 items。
	data := f.listData(t, q("entityType", "change"))
	items := data["items"].([]interface{})
	assert.Equal(t, data["total"], float64(len(items)))
}

func TestListChains_CombinesNameWithOtherFilters(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.ElementsMatch(t, []string{"Network Change Approval"},
		f.listNames(t, q("name", "change", "entityType", "change")))
	assert.Empty(t, f.listNames(t, q("name", "change", "entityType", "incident")))
}

func TestListChains_PaginatesWithTotalPages(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	data := f.listData(t, q("page", "1", "pageSize", "2"))
	assert.Equal(t, float64(3), data["total"], "total 必须是过滤后的全量，而不是当前页数量")
	assert.Equal(t, float64(2), data["totalPages"])
	assert.Len(t, data["items"].([]interface{}), 2)

	data = f.listData(t, q("page", "2", "pageSize", "2"))
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(2), data["totalPages"])
	assert.Len(t, data["items"].([]interface{}), 1)
}

func TestListChains_FiltersStayWithinTenant(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 租户 A 与租户 B 各有一条同名审批链，名称筛选不得跨租户命中。
	assert.ElementsMatch(t, []string{"Network Change Approval"}, f.listNames(t, q("name", "network")))

	_, body := f.doList(t, q("name", "network"), f.tenantB)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"], "租户 B 只能看到自己的同名审批链")
}

func TestListChains_FailsClosedWithoutTenantContext(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	r := gin.New()
	// 故意不注入 tenant_id：缺少租户上下文必须 fail closed，不得回退到默认租户。
	r.Use(func(c *gin.Context) { c.Set("role", "admin") })
	r.GET("/api/v1/approval-chains", f.handler.ListChains)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/approval-chains", nil))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, float64(2002), body["code"])
	assert.Nil(t, body["data"], "未授权响应不得泄露任何审批链数据")

	// 显式传入 0 同样视为无效租户。
	status, body := f.doList(t, q(), 0)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, float64(2002), body["code"])
}

// TestListChains_ReturnsAdvancedStepFields 断言 8 个高级步骤字段经 HTTP 边界
// 往返后仍是 camelCase，且不返回已废弃的 timeoutHours/conditions。
func TestListChains_ReturnsAdvancedStepFields(t *testing.T) {
	f := newListContractFixture(t)
	f.seed(t, f.tenantA, chainSeed{
		name: "Advanced Chain", entityType: "change", status: "active", advancedSet: true,
	})

	items := f.listData(t, q())["items"].([]interface{})
	require.Len(t, items, 1)

	item := items[0].(map[string]interface{})
	chain, ok := item["chain"].([]interface{})
	require.True(t, ok, "chain 必须是数组，实际 %v", item["chain"])
	require.Len(t, chain, 1)

	step := chain[0].(map[string]interface{})
	for _, key := range []string{
		"approvalType", "threshold", "fallbackAction", "fallbackApproverId",
		"fallbackRole", "conditionPriorities", "conditionAmountMin", "conditionAmountMax",
	} {
		require.Contains(t, step, key, "高级步骤字段缺失: %s", key)
	}
	assert.NotContains(t, step, "timeoutHours")
	assert.NotContains(t, step, "conditions")

	assert.Equal(t, "parallel", step["approvalType"])
	assert.Equal(t, float64(2), step["threshold"])
	assert.Equal(t, "escalate", step["fallbackAction"])
	assert.Equal(t, float64(f.userA), step["fallbackApproverId"])
	assert.Equal(t, "role:ops_manager", step["fallbackRole"])
	assert.Equal(t, float64(1000), step["conditionAmountMin"])
	assert.Equal(t, float64(5000), step["conditionAmountMax"])
	assert.ElementsMatch(t, []interface{}{"high", "urgent"}, step["conditionPriorities"])
}

func TestListChains_OrdersByUpdatedAtDesc(t *testing.T) {
	f := newListContractFixture(t)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	f.seed(t, f.tenantA, chainSeed{name: "Oldest", entityType: "ticket", status: "active", updatedAt: base})
	f.seed(t, f.tenantA, chainSeed{name: "Newest", entityType: "ticket", status: "active", updatedAt: base.Add(48 * time.Hour)})
	f.seed(t, f.tenantA, chainSeed{name: "Middle", entityType: "ticket", status: "active", updatedAt: base.Add(24 * time.Hour)})

	assert.Equal(t, []string{"Newest", "Middle", "Oldest"}, f.listNames(t, q()))
}

// TestListChains_StatusFilterIsExactSingleValue 固化 status 为精确单值筛选，
// 防止重新出现「多值逗号拼接」这类伪筛选。
func TestListChains_StatusFilterIsExactSingleValue(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.Len(t, f.listNames(t, q("status", "active")), 2)
	assert.Len(t, f.listNames(t, q("status", "inactive")), 1)
	assert.Len(t, f.listNames(t, q()), 3)

	// 未知/多值 status 不是合法筛选值，必须返回空集而不是被当作「不过滤」。
	assert.Empty(t, f.listNames(t, q("status", "active,inactive")))
	assert.Empty(t, f.listNames(t, q("status", "bogus")))
}
