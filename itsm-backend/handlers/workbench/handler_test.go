package workbench

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/middleware"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workbenchEnvelope 对应 common.Response 的 {code,message,data} 包裹结构。
type workbenchEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// workbenchFixture 显式区分两个租户、两个 actor 和一个不存在的租户，
// 避免测试用位置参数传 5 个 int 造成串号。
type workbenchFixture struct {
	handler       *Handler
	tenantA       int
	tenantB       int
	actorA        int
	actorB        int
	missingTenant int
}

// setupWorkbenchTest 在内存 SQLite + 全量 ent schema 上组装真实 repository/service/handler，
// 并显式区分 tenantA / tenantB 与两个 actor。返回的 handler 挂在真实 gin 路由上。
//
// 种子刻意覆盖四个领域各一条活跃记录，外加 tenantA 一条已软删工单：
//   - changes 没有 deleted_at 列，任何给 changes 加软删谓词的回归都会在这里变红
//   - 每个领域只放一条，phase/分页断言才能给出确定数值
func setupWorkbenchTest(t *testing.T) *workbenchFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	const actorName = "workbench-actor-%s"
	mkUser := func(suffix string, tenantID int) int {
		u, err := client.User.Create().
			SetUsername(fmt.Sprintf(actorName, suffix)).
			SetEmail(fmt.Sprintf(actorName+"@example.com", suffix)).
			SetName(fmt.Sprintf(actorName, suffix)).
			SetPasswordHash("not-a-real-hash").
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return u.ID
	}
	mkTenant := func(code string) int {
		tn, err := client.Tenant.Create().
			SetName("Workbench " + code).
			SetCode(code).
			Save(ctx)
		require.NoError(t, err)
		return tn.ID
	}

	tenantA, tenantB := mkTenant("wb-a"), mkTenant("wb-b")
	actorA, actorB := mkUser("a", tenantA), mkUser("b", tenantB)
	const missingTenant = 9999

	_, err = client.Ticket.Create().
		SetTitle("A-工单-待处理").SetTicketNumber("TK-A-1").
		SetRequesterID(actorA).SetAssigneeID(actorA).SetTenantID(tenantA).
		SetStatus("open").SetPriority("high").Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTitle("A-工单-已软删").SetTicketNumber("TK-A-2").
		SetRequesterID(actorA).SetTenantID(tenantA).
		SetStatus("closed").SetPriority("low").
		SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	_, err = client.Incident.Create().
		SetTitle("A-事件-新建").SetIncidentNumber("IN-A-1").
		SetReporterID(actorA).SetAssigneeID(actorA).SetTenantID(tenantA).
		SetStatus("new").SetPriority("critical").Save(ctx)
	require.NoError(t, err)

	_, err = client.Problem.Create().
		SetTitle("A-问题-处理中").SetCreatedBy(actorA).SetAssigneeID(actorA).SetTenantID(tenantA).
		SetStatus("in_progress").SetPriority("high").Save(ctx)
	require.NoError(t, err)

	_, err = client.Change.Create().
		SetTitle("A-变更-草稿").SetCreatedBy(actorA).SetAssigneeID(actorA).SetTenantID(tenantA).
		SetStatus("draft").SetPriority("medium").Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTitle("B-工单-待处理").SetTicketNumber("TK-B-1").
		SetRequesterID(actorB).SetAssigneeID(actorB).SetTenantID(tenantB).
		SetStatus("open").SetPriority("high").Save(ctx)
	require.NoError(t, err)

	return &workbenchFixture{
		handler:       NewHandler(NewService(NewRepository(db))),
		tenantA:       tenantA,
		tenantB:       tenantB,
		actorA:        actorA,
		actorB:        actorB,
		missingTenant: missingTenant,
	}
}

// doWorkbench 走真实 GET /api/v1/workbench 入口。tenantID<=0 时不注入租户上下文，
// 用于验证 fail-closed 分支。
func doWorkbench(t *testing.T, h *Handler, tenantID int, query string) (int, workbenchEnvelope) {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	router.GET("/api/v1/workbench", h.Query)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workbench"+query, nil))

	var env workbenchEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	return rec.Code, env
}

func decodeList(t *testing.T, env workbenchEnvelope) struct {
	Items      []WorkbenchItem `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
} {
	t.Helper()
	var out struct {
		Items      []WorkbenchItem `json:"items"`
		Total      int             `json:"total"`
		Page       int             `json:"page"`
		PageSize   int             `json:"pageSize"`
		TotalPages int             `json:"totalPages"`
	}
	require.NoError(t, json.Unmarshal(env.Data, &out))
	return out
}

// TestQueryFailsWithoutTenantContext 锁定租户上下文缺失必须 401 fail-closed。
// 历史实现读的是 c.Get("tenantID")（全站规范键是 tenant_id），
// 任何已登录请求都会走到这里并返回 401，/workbench 页面恒显加载失败。
func TestQueryFailsWithoutTenantContext(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, env := doWorkbench(t, f.handler, 0, "")

	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, 2001, env.Code)
	assert.NotContains(t, string(env.Data), "TK-A-1")
}

// TestQueryReturnsOwnTenantOnly 锁定租户隔离：tenantA 拿不到 tenantB 的记录。
func TestQueryReturnsOwnTenantOnly(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, envA := doWorkbench(t, f.handler, f.tenantA, "?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, 0, envA.Code)
	listA := decodeList(t, envA)

	titlesA := make([]string, 0, len(listA.Items))
	for _, item := range listA.Items {
		titlesA = append(titlesA, item.Title)
		assert.Equal(t, f.tenantA, item.TenantID)
	}
	assert.Contains(t, titlesA, "A-工单-待处理")
	assert.NotContains(t, titlesA, "B-工单-待处理")
	assert.NotContains(t, titlesA, "A-工单-已软删")

	_, envB := doWorkbench(t, f.handler, f.tenantB, "?page=1&pageSize=20")
	listB := decodeList(t, envB)
	require.Len(t, listB.Items, 1)
	assert.Equal(t, "B-工单-待处理", listB.Items[0].Title)
	assert.Equal(t, f.tenantB, listB.Items[0].TenantID)
}

// TestQueryCoversAllFourDomains 锁定 changes 分支可用：
// changes 表没有 deleted_at，沿用统一软删谓词会让整条 UNION 报错。
func TestQueryCoversAllFourDomains(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, env := doWorkbench(t, f.handler, f.tenantA, "?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, status)
	list := decodeList(t, env)

	byType := map[string]string{}
	for _, item := range list.Items {
		byType[item.RecordType] = item.Title
	}
	assert.Equal(t, "A-工单-待处理", byType["ticket"])
	assert.Equal(t, "A-事件-新建", byType["incident"])
	assert.Equal(t, "A-问题-处理中", byType["problem"])
	assert.Equal(t, "A-变更-草稿", byType["change"], "changes 域必须能进统一工作台")
}

// TestQueryPhaseFilterBeforePaging 锁定 phase 过滤发生在 SQL 侧而不是取回一页之后再筛：
// 先 LIMIT 再按 phase 过滤会让 total 与翻页都失真，
// 且历史实现的 count SQL 外层括号不闭合，这一步在修复前必然 500。
func TestQueryPhaseFilterBeforePaging(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, env := doWorkbench(t, f.handler, f.tenantA, "?phase=active&page=1&pageSize=1")
	require.Equal(t, http.StatusOK, status, "phase 过滤后的 count 查询必须可执行")
	require.Equal(t, 0, env.Code)
	list := decodeList(t, env)

	require.Len(t, list.Items, 1)
	assert.Equal(t, "A-问题-处理中", list.Items[0].Title)
	assert.Equal(t, "active", list.Items[0].Phase)
	assert.Equal(t, 1, list.Total, "total 必须是过滤后的总数，而不是全量计数")
	assert.Equal(t, 1, list.TotalPages)
}

// TestQueryListContract 锁定响应契约：{items,total,page,pageSize,totalPages} + camelCase item，
// 且空结果必须是 []。
func TestQueryListContract(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, env := doWorkbench(t, f.handler, f.tenantA, "?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, status)

	var generic map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(env.Data, &generic))
	for _, key := range []string{"items", "total", "page", "pageSize", "totalPages"} {
		assert.Contains(t, generic, key)
	}
	assert.NotContains(t, generic, "total_count")
	assert.NotContains(t, generic, "records")

	var rawItems []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(generic["items"], &rawItems))
	require.NotEmpty(t, rawItems)
	for _, item := range rawItems {
		for _, key := range []string{"recordType", "title", "assigneeId", "createdAt", "updatedAt", "tenantId", "phase"} {
			assert.Contains(t, item, key)
		}
		for key := range item {
			assert.NotContains(t, key, "_", "item 字段必须 camelCase，实际键=%s", key)
		}
	}

	// 不存在的租户必须返回空数组而不是 null。
	_, emptyEnv := doWorkbench(t, f.handler, f.missingTenant, "?page=1&pageSize=20")
	var emptyData struct {
		Items *[]WorkbenchItem `json:"items"`
		Total int              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(emptyEnv.Data, &emptyData))
	require.NotNil(t, emptyData.Items, "空列表必须序列化为 []，前端不得用 null 兜底")
	assert.Empty(t, *emptyData.Items)
	assert.Zero(t, emptyData.Total)
}

// TestQueryAssigneeFilter 锁定按处理人过滤只在租户内生效，
// 传入他人 assigneeId 不能带出别的租户数据。
func TestQueryAssigneeFilter(t *testing.T) {
	f := setupWorkbenchTest(t)

	status, env := doWorkbench(t, f.handler, f.tenantA, fmt.Sprintf("?assigneeId=%d&page=1&pageSize=20", f.actorB))
	require.Equal(t, http.StatusOK, status)
	list := decodeList(t, env)
	assert.Zero(t, list.Total, "tenantB 的 actor 在 tenantA 上下文里不应命中任何记录")

	statusB, envB := doWorkbench(t, f.handler, f.tenantB, fmt.Sprintf("?assigneeId=%d&page=1&pageSize=20", f.actorB))
	require.Equal(t, http.StatusOK, statusB)
	listB := decodeList(t, envB)
	require.Equal(t, 1, listB.Total)
	assert.Equal(t, "B-工单-待处理", listB.Items[0].Title)
}
