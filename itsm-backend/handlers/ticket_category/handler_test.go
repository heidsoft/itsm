package ticket_category

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
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
	group := r.Group("/api/v1")
	group.Group("/ticket-categories").GET("", h.ListCategories)
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
