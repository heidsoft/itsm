package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"
	ticketTypeHandler "itsm-backend/handlers/ticket_type"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTicketTypeRoutes_DBOnlyPathPrecheckForDeclaredRole 回归：挂在分组参数上的路由
// 声明（tenant.GET("/ticket-types", RequirePermission("ticket","read"), ...)）必须进入
// RBACMiddleware 的路径预检映射。
//
// 2026-10-04 P0：扫描器此前无法解析函数参数分组的前缀，这类声明被 /api/v1 过滤静默
// 丢弃，DBOnly 生产下非 super_admin 角色在预检处恒 403——工单类型、teams、tags、
// problem-relationships、approval-records、my-approvals 等 21 个端点全部锁死，
// 也是 ga-gate 租户隔离步骤自 2026-09-27 起持续红的根因。
// 本用例走真实 SetupRoutes 入口，同时锁住「fail-closed 仍然生效」：缺权限的角色必须 403。
func TestTicketTypeRoutes_DBOnlyPathPrecheckForDeclaredRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:router_ticket_type_precheck?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	prevMode := middleware.PermissionConfig.Mode
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeDBOnly
	t.Cleanup(func() {
		middleware.PermissionConfig.Mode = prevMode
		middleware.InvalidateAllPermissionCachesEx()
	})
	// 权限缓存是进程级全局（键为 role_tenant），先失效再断言，避免同角色名串味。
	middleware.InvalidateAllPermissionCachesEx()

	tenantA := client.Tenant.Create().SetName("Precheck A").SetCode("precheck-a").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Precheck B").SetCode("precheck-b").SetStatus("active").SaveX(ctx)

	// 两个租户各自安装同一套基线：agent 持有 ticket:read，technician 一条权限都没有。
	for _, tenant := range []*ent.Tenant{tenantA, tenantB} {
		readPerm := client.Permission.Create().
			SetCode("ticket:read").SetName("Read tickets").
			SetResource("ticket").SetAction("read").
			SetTenantID(tenant.ID).SaveX(ctx)
		agent := client.Role.Create().SetCode("agent").SetName("Agent").
			SetTenantID(tenant.ID).SetIsActive(true).SaveX(ctx)
		client.RolePermission.Create().SetRoleID(agent.ID).SetPermissionID(readPerm.ID).
			SetTenantID(tenant.ID).SaveX(ctx)
		client.Role.Create().SetCode("technician").SetName("Technician").
			SetTenantID(tenant.ID).SetIsActive(true).SaveX(ctx)
	}

	newUser := func(username string, role user.Role, tenantID int) *ent.User {
		return client.User.Create().
			SetUsername(username).SetEmail(username + "@example.com").SetName(username).
			SetPasswordHash("not-used-in-test").SetRole(role).SetActive(true).
			SetTenantID(tenantID).SaveX(ctx)
	}
	agentA := newUser("agent-a", user.RoleAgent, tenantA.ID)
	agentB := newUser("agent-b", user.RoleAgent, tenantB.ID)
	deniedA := newUser("denied-a", user.RoleTechnician, tenantA.ID)

	typeA := client.TicketType.Create().
		SetCode("precheck-service").SetName("Precheck Service").
		SetDescription("tenant A type").SetIcon("wrench").SetColor("#123456").
		SetCustomFields(map[string]interface{}{}).
		SetApprovalChain([]interface{}{}).SetAssignmentRules([]interface{}{}).
		SetNotificationConfig(map[string]interface{}{}).
		SetPermissionConfig(map[string]interface{}{}).
		SetTenantID(int64(tenantA.ID)).SetCreatedBy(int64(agentA.ID)).
		SetCreatedAt(time.Now()).SetUpdatedAt(time.Now()).SaveX(ctx)

	logger := zaptest.NewLogger(t).Sugar()
	const secret = "ticket-type-precheck-secret"
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:         secret,
		Logger:            logger,
		Client:            client,
		TicketTypeHandler: ticketTypeHandler.NewHandler(service.NewTicketTypeService(client, logger), logger),
	})

	get := func(actor *ent.User, role, path string) *httptest.ResponseRecorder {
		t.Helper()
		token, err := middleware.GenerateAccessToken(actor.ID, actor.Username, role, actor.TenantID, secret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 1) 声明权限的真实持有者必须通过预检（修复前此处是 403 权限不足）。
	listed := get(agentA, "agent", "/api/v1/ticket-types?page=1&pageSize=20")
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	var listBody struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID   int    `json:"id"`
				Code string `json:"code"`
			} `json:"items"`
			Total      int `json:"total"`
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			TotalPages int `json:"totalPages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &listBody))
	assert.Zero(t, listBody.Code)
	require.Len(t, listBody.Data.Items, 1)
	assert.Equal(t, typeA.Code, listBody.Data.Items[0].Code)
	assert.Equal(t, 1, listBody.Data.Total)
	assert.Equal(t, 1, listBody.Data.TotalPages)

	// 2) 单条读取同租户放行，跨租户 404 且不回显私有内容。
	one := get(agentA, "agent", "/api/v1/ticket-types/"+strconv.Itoa(typeA.ID))
	require.Equal(t, http.StatusOK, one.Code, one.Body.String())
	cross := get(agentB, "agent", "/api/v1/ticket-types/"+strconv.Itoa(typeA.ID))
	require.Equal(t, http.StatusNotFound, cross.Code)
	assert.NotContains(t, cross.Body.String(), "Precheck Service")

	// 3) fail-closed 不被放宽：角色行存在但没有任何授权时仍必须 403。
	denied := get(deniedA, "technician", "/api/v1/ticket-types?page=1&pageSize=20")
	require.Equal(t, http.StatusForbidden, denied.Code, denied.Body.String())
	var deniedBody map[string]any
	require.NoError(t, json.Unmarshal(denied.Body.Bytes(), &deniedBody))
	assert.Equal(t, float64(2003), deniedBody["code"])
}
