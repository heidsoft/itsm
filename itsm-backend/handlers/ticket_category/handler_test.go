package ticket_category

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GET /api/v1/ticket-categories 契约回归：
//   - 过滤参数是 camelCase parentId / isActive（历史实现读取 snake_case
//     parent_id / active，筛选条件被静默丢弃）。
//   - page/pageSize 必须生效（历史实现硬编码 1/100，前端分页参数完全无效）。
//   - tenantId 只能来自认证上下文，禁止由查询参数注入。

func newCategoryTestRouter(t *testing.T, client *ent.Client, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(service.NewTicketCategoryService(client), logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		c.Next()
	})
	// 与 router.SetupTicketCategoryRoutes 相同的路径与方法（省略 RequirePermission，
	// 权限链路由 router 包测试覆盖），确保测试打到真实生产 handler 入口。
	group := r.Group("/api/v1").Group("/ticket-categories")
	group.GET("", h.ListCategories)
	group.POST("", h.CreateCategory)
	group.PUT("/:id", h.UpdateCategory)
	return r
}

type categoryListEnvelope struct {
	Code int `json:"code"`
	Data struct {
		Items      []map[string]interface{} `json:"items"`
		Total      int                      `json:"total"`
		Page       int                      `json:"page"`
		PageSize   int                      `json:"pageSize"`
		TotalPages int                      `json:"totalPages"`
	} `json:"data"`
}

func listCategories(t *testing.T, r *gin.Engine, query string) categoryListEnvelope {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ticket-categories?"+query, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var env categoryListEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return env
}

func seedCategory(t *testing.T, client *ent.Client, tenantID int, code string, parentID *int, active bool) int {
	t.Helper()
	create := client.TicketCategory.Create().
		SetName(code).
		SetCode(code).
		SetTenantID(tenantID).
		SetIsActive(active).
		SetLevel(1)
	if parentID != nil {
		create.SetParentID(*parentID)
	}
	category, err := create.Save(context.Background())
	require.NoError(t, err)
	return category.ID
}

func TestListCategories_HonorsCamelCaseParentFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_list_parent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	parent := seedCategory(t, client, 1, "ROOT", nil, true)
	child := seedCategory(t, client, 1, "CHILD-1", &parent, true)
	seedCategory(t, client, 1, "CHILD-2", &parent, true)
	seedCategory(t, client, 1, "OTHER", nil, true)

	r := newCategoryTestRouter(t, client, 1)

	// child 无下级：精确过滤必须返回空集。过滤器被丢弃时会返回全部 4 条。
	env := listCategories(t, r, "parentId="+strconv.Itoa(child))
	assert.Equal(t, 0, env.Code)
	assert.Empty(t, env.Data.Items)
	assert.Equal(t, 0, env.Data.Total)

	env = listCategories(t, r, "parentId="+strconv.Itoa(parent))
	assert.Len(t, env.Data.Items, 2)
	assert.Equal(t, 2, env.Data.Total)
}

func TestListCategories_HonorsCamelCaseIsActiveFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_list_active?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedCategory(t, client, 1, "ON", nil, true)
	seedCategory(t, client, 1, "OFF-1", nil, false)
	seedCategory(t, client, 1, "OFF-2", nil, false)

	r := newCategoryTestRouter(t, client, 1)
	env := listCategories(t, r, "isActive=false")

	require.Equal(t, 0, env.Code)
	assert.Equal(t, 2, env.Data.Total, "isActive=false 只应命中 2 条停用分类")
	require.Len(t, env.Data.Items, 2)
	for _, item := range env.Data.Items {
		assert.Equal(t, false, item["isActive"])
	}
}

func TestListCategories_HonorsPagination(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_list_page?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	for _, code := range []string{"A", "B", "C", "D", "E"} {
		seedCategory(t, client, 1, code, nil, true)
	}

	r := newCategoryTestRouter(t, client, 1)
	env := listCategories(t, r, "page=2&pageSize=2")

	require.Equal(t, 0, env.Code)
	assert.Equal(t, 5, env.Data.Total, "total 必须是全量计数，不受分页影响")
	assert.Len(t, env.Data.Items, 2)
	assert.Equal(t, 2, env.Data.Page)
	assert.Equal(t, 2, env.Data.PageSize)
	assert.Equal(t, 3, env.Data.TotalPages)
}

func TestListCategories_IgnoresInjectedTenantID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_list_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedCategory(t, client, 1, "OWN", nil, true)
	seedCategory(t, client, 2, "FOREIGN", nil, true)

	r := newCategoryTestRouter(t, client, 1)
	env := listCategories(t, r, "tenantId=2")

	require.Equal(t, 0, env.Code)
	assert.Equal(t, 1, env.Data.Total, "查询参数 tenantId 不得覆盖认证租户上下文")
	require.Len(t, env.Data.Items, 1)
	assert.Equal(t, "OWN", env.Data.Items[0]["code"])
}

// POST/PUT /api/v1/ticket-categories 契约：分类关联的工作流是建单选路的一个真实来源，
// 写入时必须能在本租户解析到激活的流程定义。未知 Key 返回 400 + code 4000，
// 而不是「保存成功」后工单仍走默认流程。
func TestCategoryWrite_RejectsUnknownWorkflowKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_write_wf?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	r := newCategoryTestRouter(t, client, 1)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ticket-categories", strings.NewReader(
		`{"name":"网络故障","code":"net-fault","isActive":true,"workflowDefinitionKey":"ghost_flow"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 4000, env.Code)
	assert.Equal(t, "工作流不存在、未激活或不属于当前租户", env.Message)

	count, err := client.TicketCategory.Query().Where(ticketcategory.CodeEQ("net-fault")).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "被拒绝的写入不得留下分类行")
}

func TestCategoryWrite_AcceptsDeployedWorkflowKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cat_write_wf_ok?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("dep-cat-net").SetDeploymentName("deploy cat net").SetTenantID(1).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("net_flow").SetName("网络故障流程").SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(dep.ID).SetTenantID(1).Save(ctx)
	require.NoError(t, err)

	r := newCategoryTestRouter(t, client, 1)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ticket-categories", strings.NewReader(
		`{"name":"网络故障","code":"net-fault","isActive":true,"workflowDefinitionKey":"net_flow"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var created struct {
		Code int `json:"code"`
		Data struct {
			ID                    int    `json:"id"`
			WorkflowDefinitionKey string `json:"workflowDefinitionKey"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, 0, created.Code)
	assert.Equal(t, "net_flow", created.Data.WorkflowDefinitionKey)

	// 显式传空串才是解除绑定；未传字段保持原值
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/ticket-categories/"+strconv.Itoa(created.Data.ID), strings.NewReader(`{"name":"网络故障"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	row, err := client.TicketCategory.Get(ctx, created.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, "net_flow", row.WorkflowDefinitionKey, "未传 workflowDefinitionKey 时不得清空既有绑定")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/ticket-categories/"+strconv.Itoa(created.Data.ID), strings.NewReader(`{"workflowDefinitionKey":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	row, err = client.TicketCategory.Get(ctx, created.Data.ID)
	require.NoError(t, err)
	assert.Empty(t, row.WorkflowDefinitionKey)
}
