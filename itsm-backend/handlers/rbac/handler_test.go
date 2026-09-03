package rbac

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	entrole "itsm-backend/ent/role"
	"itsm-backend/middleware"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
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

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20")
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
	w := doRBACRequest(t, h, 0, "/roles?page=1&page_size=20")
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

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20&search="+"%E8%BF%90%E7%BB%B4")
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

	w := doRBACRequest(t, h, tenantA, "/roles?page=2&page_size=2")
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

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20")
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

	w := doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20")
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

	list := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20"))
	require.Len(t, list.Roles, 1)
	assert.ElementsMatch(t, []string{"ticket:read", "ticket:write"}, list.Roles[0].Permissions,
		"写入 RolePermission 后必须能在列表读回")

	// 重新授权为单一权限，验证 Replace 语义（旧关联被清除，不是追加）。
	w = doRBACPost(t, h, tenantA, fmt.Sprintf("/roles/%d/permissions", roleEnt.ID),
		dto.AssignPermissionsRequest{PermissionIDs: []int{p1}})
	require.Equal(t, http.StatusOK, w.Code)

	list = parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20"))
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

	list := parseRoleList(t, doRBACRequest(t, h, tenantA, "/roles?page=1&page_size=20"))
	require.Len(t, list.Roles, 1)
	assert.NotContains(t, list.Roles[0].Permissions, "secret:read", "被拒绝的授权不得落库")
}
