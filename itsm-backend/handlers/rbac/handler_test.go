package rbac

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entrole "itsm-backend/ent/role"
	"itsm-backend/middleware"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// rbacEnvelope 对应 common.Response 的 {code,message,data} 包裹结构。
type rbacEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// setupRBACTest 在内存 SQLite + 全量 ent schema 上组装 RBAC handler
// （不同测试用独立 DSN 隔离，避免互相污染）。
func setupRBACTest(t *testing.T) (*Handler, *ent.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(
		service.NewRoleService(client, logger),
		service.NewPermissionService(client, logger),
		service.NewMenuService(client, logger),
		logger,
	)
	return h, client
}

// doRBACRequest 走真实 gin 路由发起请求。tenantID<=0 时不注入租户上下文，
// 用于验证未授权分支。
func doRBACRequest(t *testing.T, h *Handler, tenantID int, path string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	router.GET("/roles", h.ListRoles)
	router.GET("/roles/:id", h.GetRole)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// doRBACPost 发起带 JSON body 的写请求（用于 AssignPermissions 等）。
func doRBACPost(t *testing.T, h *Handler, tenantID int, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	router.POST("/roles/:id/permissions", h.AssignPermissions)

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

// doRBACWrite 走真实 gin 路由发起角色写请求（POST /roles、PUT /roles/:id）。
func doRBACWrite(t *testing.T, h *Handler, tenantID int, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	router.POST("/roles", h.CreateRole)
	router.PUT("/roles/:id", h.UpdateRole)

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func mkRBACTenant(t *testing.T, client *ent.Client, tag string) int {
	t.Helper()
	slug := strings.ReplaceAll(t.Name(), "/", "-") + "-" + tag
	tn, err := client.Tenant.Create().
		SetName("Tenant-" + slug).
		SetCode(strings.ToUpper(tag) + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain(slug + ".test").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tn.ID
}

func mkRBACPermission(t *testing.T, client *ent.Client, tenantID int, code string) int {
	t.Helper()
	p, err := client.Permission.Create().
		SetCode(code).
		SetName(code).
		SetResource(strings.SplitN(code, ":", 2)[0]).
		SetAction(code).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return p.ID
}

// mkRBACRole 创建角色并可选挂载权限。
//
// 注意：角色↔权限关联的系统真源是 RolePermission 实体（带 tenant_id，服务层
// AssignPermissions/getRolePermissionsByIDs 均读写它），**不是** ent 的 M2M 边
// role.Permissions。用 AddPermissionIDs 写边会静默落到无人读取的联表，
// 导致授权看起来成功但查询为空——夹具必须直接写 RolePermission。
func mkRBACRole(t *testing.T, client *ent.Client, tenantID int, name, code string, scope entrole.DataScope, active bool, permIDs ...int) *ent.Role {
	t.Helper()
	r, err := client.Role.Create().
		SetName(name).
		SetCode(code).
		SetDescription(name + " description").
		SetDataScope(scope).
		SetIsActive(active).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)

	for _, pid := range permIDs {
		_, err := client.RolePermission.Create().
			SetRoleID(r.ID).
			SetPermissionID(pid).
			SetTenantID(tenantID).
			Save(context.Background())
		require.NoError(t, err)
	}
	return r
}

// parseRoleList 解析列表响应为 RoleListResponse。
func parseRoleList(t *testing.T, w *httptest.ResponseRecorder) dto.RoleListResponse {
	t.Helper()
	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "响应体: %s", w.Body.String())
	require.Equal(t, 0, env.Code, "业务码应为 0，响应体: %s", w.Body.String())
	var list dto.RoleListResponse
	require.NoError(t, json.Unmarshal(env.Data, &list))
	return list
}

func TestListRoles_TenantIsolation(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	tenantB := mkRBACTenant(t, client, "b")

	mkRBACRole(t, client, tenantA, "A-管理员", "role-a-admin", entrole.DataScopeAll, true)
	mkRBACRole(t, client, tenantA, "A-运维", "role-a-ops", entrole.DataScopeDepartment, true)
	mkRBACRole(t, client, tenantB, "B-管理员", "role-b-admin", entrole.DataScopeAll, true)

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, w.Code)
	list := parseRoleList(t, w)

	require.Equal(t, 2, list.Total, "租户 A 只应看到自己的 2 个角色")
	for _, r := range list.Roles {
		assert.Equal(t, tenantA, r.TenantID, "角色 %s 不属于租户 A，发生跨租户泄漏", r.Code)
		assert.NotEqual(t, tenantB, r.TenantID)
	}
}

func TestListRoles_RequiresTenantContext(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	mkRBACRole(t, client, tenantA, "A-管理员", "role-a-admin", entrole.DataScopeAll, true)

	// tenantID=0 → 不注入租户上下文，模拟中间件未生效的请求。
	w := doRBACRequest(t, h, 0, "/roles?page=1&pageSize=20")
	require.Equal(t, http.StatusUnauthorized, w.Code, "缺少租户上下文必须 401，不能放行")

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEqual(t, 0, env.Code, "失败响应业务码不应为 0")
}

func TestListRoles_SearchFiltersWithinTenant(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	tenantB := mkRBACTenant(t, client, "b")

	mkRBACRole(t, client, tenantA, "运维工程师", "role-a-ops", entrole.DataScopeAll, true)
	mkRBACRole(t, client, tenantA, "开发工程师", "role-a-dev", entrole.DataScopeAll, true)
	// 同名关键词但属于其他租户，不得被搜索到。
	mkRBACRole(t, client, tenantB, "运维工程师", "role-b-ops", entrole.DataScopeAll, true)

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20&search="+"%E8%BF%90%E7%BB%B4")
	require.Equal(t, http.StatusOK, w.Code)
	list := parseRoleList(t, w)

	require.Equal(t, 1, list.Total, "搜索结果必须限定在本租户内")
	require.Len(t, list.Roles, 1)
	assert.Equal(t, "role-a-ops", list.Roles[0].Code)
	assert.Equal(t, tenantA, list.Roles[0].TenantID)
}

func TestListRoles_PaginationMath(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	for i := 0; i < 5; i++ {
		mkRBACRole(t, client, tenantA,
			fmt.Sprintf("角色%d", i), fmt.Sprintf("role-a-%d", i), entrole.DataScopeAll, true)
	}

	w := doRBACRequest(t, h, tenantA, "/roles?page=2&pageSize=2")
	require.Equal(t, http.StatusOK, w.Code)
	list := parseRoleList(t, w)

	assert.Equal(t, 5, list.Total, "total 应是全量计数，不受分页影响")
	assert.Equal(t, 2, list.Page)
	assert.Equal(t, 2, list.PageSize)
	assert.Equal(t, 3, list.TotalPages, "5 条 / 每页 2 条 = 3 页（向上取整）")
	assert.Len(t, list.Roles, 2)
}

func TestListRoles_MapsStatusScopeAndPermissions(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	p1 := mkRBACPermission(t, client, tenantA, "ticket:read")
	p2 := mkRBACPermission(t, client, tenantA, "ticket:write")

	mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeOwner, true, p1, p2)
	mkRBACRole(t, client, tenantA, "停用角色", "role-a-off", entrole.DataScopeAll, false)

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, w.Code)
	list := parseRoleList(t, w)
	require.Len(t, list.Roles, 2)

	byCode := make(map[string]dto.RoleDTO, len(list.Roles))
	for _, r := range list.Roles {
		byCode[r.Code] = r
	}

	ops := byCode["role-a-ops"]
	assert.Equal(t, "active", ops.Status, "is_active=true 应映射为 active")
	assert.Equal(t, "owner", ops.DataScope)
	assert.ElementsMatch(t, []string{"ticket:read", "ticket:write"}, ops.Permissions)

	off := byCode["role-a-off"]
	assert.Equal(t, "inactive", off.Status, "is_active=false 应映射为 inactive")
	assert.Empty(t, off.Permissions, "无权限时应为空数组而非 null 缺失")
}

// =============================================================================
// 角色启用状态契约回归（P0：启用/禁用开关曾是假控件）
//
// 线上实测：PUT /api/v1/roles/19 {"status":"inactive"} 返回 HTTP 200 + code 0
// "success"，但响应体只有 isActive:true、数据库 is_active 仍为 t。根因是 service
// 只读 req.IsActive，而前端与 DTO 契约只发 status，字段被静默丢弃。
// 以下用例在修复前全部失败，修复后通过。
// =============================================================================

// TestUpdateRole_StatusTogglesIsActive 启用/禁用开关必须真实落库。
func TestUpdateRole_StatusTogglesIsActive(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	r := mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true)

	w := doRBACWrite(t, h, tenantA, http.MethodPut, fmt.Sprintf("/roles/%d", r.ID),
		map[string]any{"status": "inactive"})
	require.Equal(t, http.StatusOK, w.Code, "响应体: %s", w.Body.String())

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code, "禁用角色应成功，响应体: %s", w.Body.String())

	var resp dto.RoleResponse
	require.NoError(t, json.Unmarshal(env.Data, &resp))
	assert.Equal(t, "inactive", resp.Status, "响应必须回显已生效的 status")

	// 单一契约：不得再输出 isActive 同义字段，避免前端读错字段导致「看起来没变」
	assert.NotContains(t, string(env.Data), `"isActive"`, "启用状态只允许 status 一个线上字段")

	persisted, err := client.Role.Get(context.Background(), r.ID)
	require.NoError(t, err)
	assert.False(t, persisted.IsActive, "数据库中 is_active 必须真的翻转为 false")

	// 再翻回 active，确认双向可逆而非单向写死
	w = doRBACWrite(t, h, tenantA, http.MethodPut, fmt.Sprintf("/roles/%d", r.ID),
		map[string]any{"status": "active"})
	require.Equal(t, http.StatusOK, w.Code)
	persisted, err = client.Role.Get(context.Background(), r.ID)
	require.NoError(t, err)
	assert.True(t, persisted.IsActive, "数据库中 is_active 必须能翻回 true")
}

// TestUpdateRole_InvalidStatusRejected 非法枚举必须拒绝，禁止静默回退默认值。
func TestUpdateRole_InvalidStatusRejected(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	r := mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true)

	w := doRBACWrite(t, h, tenantA, http.MethodPut, fmt.Sprintf("/roles/%d", r.ID),
		map[string]any{"status": "enabled"})
	require.Equal(t, http.StatusBadRequest, w.Code, "非法 status 必须 400，响应体: %s", w.Body.String())

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, common.ParamErrorCode, env.Code, "业务码应为参数错误 1001")

	persisted, err := client.Role.Get(context.Background(), r.ID)
	require.NoError(t, err)
	assert.True(t, persisted.IsActive, "被拒绝的请求不得改动数据库")
}

// TestUpdateRole_BlankStatusDoesNotDisable 空串等同「未提交」，不得被当成 inactive。
func TestUpdateRole_BlankStatusDoesNotDisable(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	r := mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true)

	name := "改名后的运维"
	w := doRBACWrite(t, h, tenantA, http.MethodPut, fmt.Sprintf("/roles/%d", r.ID),
		map[string]any{"name": name, "status": ""})
	require.Equal(t, http.StatusOK, w.Code, "响应体: %s", w.Body.String())

	persisted, err := client.Role.Get(context.Background(), r.ID)
	require.NoError(t, err)
	assert.Equal(t, name, persisted.Name, "同请求中的其他字段仍应生效")
	assert.True(t, persisted.IsActive, "空 status 不得把角色静默禁用")
}

// TestCreateRole_HonorsStatus 创建时显式 inactive 必须落库为禁用，其余默认启用。
func TestCreateRole_HonorsStatus(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")

	w := doRBACWrite(t, h, tenantA, http.MethodPost, "/roles",
		dto.CreateRoleRequest{Name: "停用角色", Code: "role-a-off", Status: "inactive"})
	require.Equal(t, http.StatusOK, w.Code, "响应体: %s", w.Body.String())

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code, "响应体: %s", w.Body.String())
	var created dto.RoleResponse
	require.NoError(t, json.Unmarshal(env.Data, &created))
	assert.Equal(t, "inactive", created.Status)

	persisted, err := client.Role.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.False(t, persisted.IsActive, "显式 inactive 创建的角色不得被强制启用")

	// 未提交 status 时保持既有默认：启用
	w = doRBACWrite(t, h, tenantA, http.MethodPost, "/roles",
		dto.CreateRoleRequest{Name: "默认角色", Code: "role-a-default"})
	require.Equal(t, http.StatusOK, w.Code, "响应体: %s", w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code, "响应体: %s", w.Body.String())
	created = dto.RoleResponse{}
	require.NoError(t, json.Unmarshal(env.Data, &created))
	assert.Equal(t, "active", created.Status, "未提交 status 时默认启用")
}

// TestListRoles_StatusFilter status 过滤必须真实生效，否则筛选器是假控件。
func TestListRoles_StatusFilter(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true)
	mkRBACRole(t, client, tenantA, "开发", "role-a-dev", entrole.DataScopeAll, true)
	mkRBACRole(t, client, tenantA, "停用角色", "role-a-off", entrole.DataScopeAll, false)

	inactive := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20&status=inactive"))
	assert.Equal(t, 1, inactive.Total, "status=inactive 只应命中 1 个禁用角色")
	require.Len(t, inactive.Roles, 1)
	assert.Equal(t, "role-a-off", inactive.Roles[0].Code)

	active := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20&status=active"))
	assert.Equal(t, 2, active.Total, "status=active 只应命中 2 个启用角色")
	for _, r := range active.Roles {
		assert.Equal(t, "active", r.Status)
	}

	all := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20"))
	assert.Equal(t, 3, all.Total, "不带 status 时应返回全部角色")

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20&status=bogus")
	require.Equal(t, http.StatusBadRequest, w.Code, "非法 status 过滤值必须 400 而非静默忽略")
	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, common.ParamErrorCode, env.Code)
}

// TestListRoles_RejectsSnakeCasePageSize 分页只认 camelCase pageSize。
// 契约见 common.GetPaginationFromQuery：page_size 形态不再解析。
func TestListRoles_RejectsSnakeCasePageSize(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	for i := 0; i < 5; i++ {
		mkRBACRole(t, client, tenantA,
			fmt.Sprintf("角色%d", i), fmt.Sprintf("role-a-%d", i), entrole.DataScopeAll, true)
	}

	snake := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=2"))
	assert.Equal(t, 20, snake.PageSize, "snake_case page_size 必须被忽略并回落默认值")
	assert.Len(t, snake.Roles, 5, "被忽略的 page_size 不得截断结果")

	camel := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=2"))
	assert.Equal(t, 2, camel.PageSize, "camelCase pageSize 必须生效")
	assert.Len(t, camel.Roles, 2)
	assert.Equal(t, 5, camel.Total)
}

func TestGetRole_CrossTenantIsNotFound(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	tenantB := mkRBACTenant(t, client, "b")

	roleB := mkRBACRole(t, client, tenantB, "B-管理员", "role-b-admin", entrole.DataScopeAll, true)

	// 租户 A 持租户 B 的角色 ID 请求，属于越权（IDOR）尝试。
	w := doRBACRequest(t, h, tenantA, fmt.Sprintf("/roles/%d", roleB.ID))
	require.Equal(t, http.StatusNotFound, w.Code, "跨租户按 ID 读取必须拒绝，不能返回他租户资源")

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEqual(t, 0, env.Code)
}

func TestGetRole_SameTenantReturnsRole(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	r := mkRBACRole(t, client, tenantA, "A-管理员", "role-a-admin", entrole.DataScopeAll, true)

	w := doRBACRequest(t, h, tenantA, fmt.Sprintf("/roles/%d", r.ID))
	require.Equal(t, http.StatusOK, w.Code)

	var env rbacEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code)

	var resp dto.RoleResponse
	require.NoError(t, json.Unmarshal(env.Data, &resp))
	assert.Equal(t, r.ID, resp.ID)
	assert.Equal(t, "role-a-admin", resp.Code)
}

func TestGetRole_InvalidID(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")

	w := doRBACRequest(t, h, tenantA, "/roles/not-a-number")
	require.Equal(t, http.StatusBadRequest, w.Code, "非数字 ID 应返回参数错误而非 500")
}

// 确保权限实体按租户隔离，避免跨租户权限被枚举。
func TestListRoles_PermissionsAreTenantScoped(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	tenantB := mkRBACTenant(t, client, "b")

	permA := mkRBACPermission(t, client, tenantA, "ticket:read")
	permB := mkRBACPermission(t, client, tenantB, "secret:read")

	mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true, permA)
	mkRBACRole(t, client, tenantB, "B-运维", "role-b-ops", entrole.DataScopeAll, true, permB)

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, w.Code)
	list := parseRoleList(t, w)
	require.Len(t, list.Roles, 1)
	assert.Equal(t, []string{"ticket:read"}, list.Roles[0].Permissions)
	assert.NotContains(t, list.Roles[0].Permissions, "secret:read", "不得泄漏他租户权限码")
}

// 走真实授权写入路径（AssignPermissions → RolePermission 实体），验证授权结果
// 能被 ListRoles 读回。该用例同时守住「M2M 边 vs RolePermission 实体」双表示陷阱：
// 若将来改成写 ent 边，此测试会立刻失败。
func TestAssignPermissions_ThenListRoles_ReflectsGrants(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	p1 := mkRBACPermission(t, client, tenantA, "ticket:read")
	p2 := mkRBACPermission(t, client, tenantA, "ticket:write")
	roleEnt := mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true)

	w := doRBACPost(t, h, tenantA, fmt.Sprintf("/roles/%d/permissions", roleEnt.ID),
		dto.AssignPermissionsRequest{PermissionIDs: []int{p1, p2}})
	require.Equal(t, http.StatusOK, w.Code, "授权应成功，响应体: %s", w.Body.String())

	list := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20"))
	require.Len(t, list.Roles, 1)
	assert.ElementsMatch(t, []string{"ticket:read", "ticket:write"}, list.Roles[0].Permissions,
		"写入 RolePermission 后必须能在列表读回")

	// 重新授权为单一权限，验证 Replace 语义（旧关联被清除，不是追加）。
	w = doRBACPost(t, h, tenantA, fmt.Sprintf("/roles/%d/permissions", roleEnt.ID),
		dto.AssignPermissionsRequest{PermissionIDs: []int{p1}})
	require.Equal(t, http.StatusOK, w.Code)

	list = parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20"))
	require.Len(t, list.Roles, 1)
	assert.Equal(t, []string{"ticket:read"}, list.Roles[0].Permissions,
		"重复授权应 Replace 而非 Append")
}

// 安全回归（R1）：传入他租户的 permission ID 必须拒绝，否则可跨租户提权。
func TestAssignPermissions_RejectsCrossTenantPermission(t *testing.T) {
	h, client := setupRBACTest(t)
	tenantA := mkRBACTenant(t, client, "a")
	tenantB := mkRBACTenant(t, client, "b")

	permA := mkRBACPermission(t, client, tenantA, "ticket:read")
	permB := mkRBACPermission(t, client, tenantB, "secret:read")
	roleEnt := mkRBACRole(t, client, tenantA, "运维", "role-a-ops", entrole.DataScopeAll, true, permA)

	// 混合本租户与他租户权限 ID —— 必须整体拒绝，不能部分写入。
	w := doRBACPost(t, h, tenantA, fmt.Sprintf("/roles/%d/permissions", roleEnt.ID),
		dto.AssignPermissionsRequest{PermissionIDs: []int{permA, permB}})
	require.Equal(t, http.StatusBadRequest, w.Code, "跨租户授权必须 400 拒绝，响应体: %s", w.Body.String())

	list := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&pageSize=20"))
	require.Len(t, list.Roles, 1)
	assert.NotContains(t, list.Roles[0].Permissions, "secret:read", "被拒绝的授权不得落库")
}

// =============================================================================
// 薄层 mock 测试：补 handler 分支覆盖（不替代上方 SQLite 集成测试）。
// 现有集成测试用真实 service + ent，此处用 mock 验证 handler 自身的
// 401/400/哨兵分流等分支逻辑。
// =============================================================================

type mockRoleService struct{ mock.Mock }

func (m *mockRoleService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	args := m.Called(ctx, req, tenantID)
	if r, ok := args.Get(0).(*dto.RoleResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRoleService) GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error) {
	args := m.Called(ctx, id, tenantID)
	if r, ok := args.Get(0).(*dto.RoleResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRoleService) ListRoles(ctx context.Context, tenantID int, params *dto.GetRolesParams) ([]*dto.RoleResponse, int, error) {
	args := m.Called(ctx, tenantID, params)
	if l, ok := args.Get(0).([]*dto.RoleResponse); ok {
		return l, args.Int(1), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *mockRoleService) UpdateRole(ctx context.Context, id int, req *dto.UpdateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	args := m.Called(ctx, id, req, tenantID)
	if r, ok := args.Get(0).(*dto.RoleResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRoleService) DeleteRole(ctx context.Context, id int, tenantID int) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *mockRoleService) AssignPermissions(ctx context.Context, roleID int, permissionIDs []int, tenantID int) error {
	args := m.Called(ctx, roleID, permissionIDs, tenantID)
	return args.Error(0)
}

type mockPermissionService struct{ mock.Mock }

func (m *mockPermissionService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest, tenantID int) (*dto.PermissionResponse, error) {
	args := m.Called(ctx, req, tenantID)
	if r, ok := args.Get(0).(*dto.PermissionResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPermissionService) ListPermissions(ctx context.Context, tenantID int, resource string) ([]*dto.PermissionResponse, error) {
	args := m.Called(ctx, tenantID, resource)
	if l, ok := args.Get(0).([]*dto.PermissionResponse); ok {
		return l, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPermissionService) InitDefaultPermissions(ctx context.Context, tenantID int) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

type mockMenuService struct{ mock.Mock }

func (m *mockMenuService) CreateMenu(ctx context.Context, req *dto.CreateMenuRequest, tenantID int) (*dto.MenuDTO, error) {
	args := m.Called(ctx, req, tenantID)
	if r, ok := args.Get(0).(*dto.MenuDTO); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMenuService) GetMenu(ctx context.Context, id int, tenantID int) (*dto.MenuDTO, error) {
	args := m.Called(ctx, id, tenantID)
	if r, ok := args.Get(0).(*dto.MenuDTO); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMenuService) ListMenus(ctx context.Context, tenantID int) ([]*dto.MenuDTO, error) {
	args := m.Called(ctx, tenantID)
	if l, ok := args.Get(0).([]*dto.MenuDTO); ok {
		return l, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMenuService) UpdateMenu(ctx context.Context, id int, req *dto.UpdateMenuRequest, tenantID int) (*dto.MenuDTO, error) {
	args := m.Called(ctx, id, req, tenantID)
	if r, ok := args.Get(0).(*dto.MenuDTO); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMenuService) DeleteMenu(ctx context.Context, id int, tenantID int) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *mockMenuService) GetUserMenus(ctx context.Context, userID int, tenantID int) (*dto.MenuTreeResponse, error) {
	args := m.Called(ctx, userID, tenantID)
	if r, ok := args.Get(0).(*dto.MenuTreeResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

// newMockRbacHandler 用 mock service 构造 handler。
func newMockRbacHandler(t *testing.T, role *mockRoleService, perm *mockPermissionService, menu *mockMenuService) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return NewHandler(role, perm, menu, zaptest.NewLogger(t).Sugar())
}

func mockCtx(method, path, body string, tenantID int) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body != "" {
		c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	if tenantID > 0 {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
	}
	c.Set("user_id", 1)
	c.Set("role", "admin")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	return w, c
}

func TestCreateRole_Success_Mock(t *testing.T) {
	role := &mockRoleService{}
	h := newMockRbacHandler(t, role, &mockPermissionService{}, &mockMenuService{})
	role.On("CreateRole", mock.Anything, mock.Anything, 1).Return(&dto.RoleResponse{ID: 1, Code: "agent"}, nil)

	w, c := mockCtx(http.MethodPost, "/api/v1/roles", `{"name":"Agent"}`, 1)
	h.CreateRole(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	role.AssertExpectations(t)
}

func TestCreateRole_MissingTenant_Mock(t *testing.T) {
	role := &mockRoleService{}
	h := newMockRbacHandler(t, role, &mockPermissionService{}, &mockMenuService{})

	w, c := mockCtx(http.MethodPost, "/api/v1/roles", `{"name":"Agent"}`, 0)
	h.CreateRole(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	role.AssertNotCalled(t, "CreateRole")
}

func TestCreateRole_ValidationFailure_Mock(t *testing.T) {
	role := &mockRoleService{}
	h := newMockRbacHandler(t, role, &mockPermissionService{}, &mockMenuService{})

	// 缺 name（binding required）→ 400
	w, c := mockCtx(http.MethodPost, "/api/v1/roles", `{"code":"x"}`, 1)
	h.CreateRole(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	role.AssertNotCalled(t, "CreateRole")
}

// 核心哨兵分流：跨租户权限分配 → 400 而非 500（R1 契约，与集成测试互补）。
func TestAssignPermissions_CrossTenantSentinel_Mock(t *testing.T) {
	role := &mockRoleService{}
	h := newMockRbacHandler(t, role, &mockPermissionService{}, &mockMenuService{})
	role.On("AssignPermissions", mock.Anything, 1, []int{99}, 1).Return(service.ErrPermissionNotInTenant)

	w, c := mockCtx(http.MethodPost, "/api/v1/roles/1/permissions", `{"permissionIds":[99]}`, 1)
	h.AssignPermissions(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.ParamErrorCode, resp.Code)
	role.AssertExpectations(t)
}

func TestGetUserMenus_Unauthenticated_Mock(t *testing.T) {
	menu := &mockMenuService{}
	h := newMockRbacHandler(t, &mockRoleService{}, &mockPermissionService{}, menu)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users/me/menus", nil)
	// 不设置 user_id → 401

	h.GetUserMenus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	menu.AssertNotCalled(t, "GetUserMenus")
}

// 直接引用哨兵，防止重构时误删错误语义。
func TestSentinelExists(t *testing.T) {
	assert.Error(t, service.ErrPermissionNotInTenant)
	assert.True(t, errors.Is(service.ErrPermissionNotInTenant, service.ErrPermissionNotInTenant))
}
