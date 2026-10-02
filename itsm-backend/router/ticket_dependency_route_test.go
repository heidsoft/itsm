package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	ticketDependencyHandler "itsm-backend/handlers/ticket_dependency"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E1-5）：GET /api/v1/tickets/:id/dependencies 是
// router 真实注册的只读入口（router/ticket_routes.go:286，权限 ticket:read），
// handler 却用 ShouldBindJSON 绑一个 action 必填的结构体。GET 不带请求体，
// 于是这个入口**每次调用都固定 400「请求参数错误」**，从未可用过；
// 同时服务层「工单不存在/跨租户」被兜底成 500/5001，调用方分不清不存在与故障。
//
// 修复后的契约：影响分析的输入是枚举过滤语义，归属查询参数
// ?action=close|delete|change_status&newStatus=...；不存在与跨租户统一 404/4004。
// 必须从真实 Router 入口证明，并覆盖跨租户拒绝与「客户端自报 tenantId 不被信任」。

const dependencyImpactPathFmt = "/api/v1/tickets/%d/dependencies"

type dependencyRouteFixture struct {
	engine   *gin.Engine
	secret   string
	client   *ent.Client
	userA    *ent.User // tenant A super_admin
	stranger *ent.User // tenant B super_admin，用于跨租户探测
	parentA  *ent.Ticket
	childA1  *ent.Ticket
	childA2  *ent.Ticket
}

func setupTicketDependencyRouteTest(t *testing.T) dependencyRouteFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_ticket_dependency_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Dep A").SetCode("dep-a").SetDomain("dep-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Dep B").SetCode("dep-b").SetDomain("dep-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("dep-a-admin").SetEmail("dep-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("dep-b-admin").SetEmail("dep-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	newTicket := func(number, title, status string, tenantID int, parentID *int) *ent.Ticket {
		t.Helper()
		create := client.Ticket.Create().
			SetTicketNumber(number).
			SetTitle(title).
			SetDescription("依赖影响分析回归").
			SetType("incident").
			SetPriority("high").
			SetStatus(status).
			SetRequesterID(userA.ID).
			SetTenantID(tenantID)
		if parentID != nil {
			create = create.SetParentTicketID(*parentID)
		}
		return create.SaveX(ctx)
	}

	parentA := newTicket("TKT-DEP-A-001", "父工单", "open", tenantA.ID, nil)
	childA1 := newTicket("TKT-DEP-A-002", "子工单一", "in_progress", tenantA.ID, &parentA.ID)
	childA2 := newTicket("TKT-DEP-A-003", "子工单二", "open", tenantA.ID, &parentA.ID)
	// 租户 B 的工单把 parent_ticket_id 指向租户 A 的父工单：A 侧分析不得把它算进影响面。
	newTicket("TKT-DEP-B-900", "跨租户悬挂子工单", "open", tenantB.ID, &parentA.ID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "ticket-dependency-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketDependencyHandler: ticketDependencyHandler.NewHandler(service.NewTicketDependencyService(client, logger), logger),
	})

	return dependencyRouteFixture{
		engine:   r,
		secret:   secret,
		client:   client,
		userA:    userA,
		stranger: stranger,
		parentA:  parentA,
		childA1:  childA1,
		childA2:  childA2,
	}
}

type dependencyEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (f dependencyRouteFixture) do(t *testing.T, path string, user *ent.User) (int, dependencyEnvelope, string) {
	t.Helper()

	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, f.secret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)

	var env dependencyEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return w.Code, env, w.Body.String()
}

// impactDTO 解码分析结果，键必须是 camelCase。
func impactDTO(t *testing.T, env dependencyEnvelope) map[string]any {
	t.Helper()
	var data map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &data), "data=%s", string(env.Data))
	return data
}

func TestTicketDependencyImpactRoute(t *testing.T) {
	fx := setupTicketDependencyRouteTest(t)
	basePath := fmt.Sprintf(dependencyImpactPathFmt, fx.parentA.ID)

	t.Run("带 action 查询参数的 GET 必须可用", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath+"?action=close", fx.userA)

		require.Equal(t, http.StatusOK, status, "body=%s", raw)
		assert.Equal(t, 0, env.Code)

		data := impactDTO(t, env)
		assert.Equal(t, float64(fx.parentA.ID), data["ticketId"])
		assert.Equal(t, "TKT-DEP-A-001", data["ticketNumber"])
		assert.Equal(t, "close", data["action"])
		// 只统计同租户子工单：租户 B 那条 parent_ticket_id 指向本工单，不得进入影响面。
		assert.Equal(t, float64(2), data["affectedCount"], "body=%s", raw)

		warnings, ok := data["warnings"].([]any)
		require.True(t, ok)
		assert.Len(t, warnings, 2, "两条未完成子工单都应给出告警")

		affected, ok := data["affectedTickets"].([]any)
		require.True(t, ok)
		first := affected[0].(map[string]any)
		assert.Equal(t, "blocked", first["impactType"])
		assert.Contains(t, first["description"], "父工单关闭")

		// 服务层的风险分级口径：0 条告警 low，≤2 条 medium，更多 high。
		assert.Equal(t, "medium", data["riskLevel"])
	})

	t.Run("省略 action 是 400 并说明允许的取值", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath, fx.userA)

		assert.Equal(t, http.StatusBadRequest, status, "body=%s", raw)
		assert.Equal(t, 1001, env.Code)
		assert.Contains(t, env.Message, "action")
		assert.Contains(t, env.Message, "change_status")
	})

	t.Run("非法 action 是 400 而不是 500", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath+"?action=archive", fx.userA)

		assert.Equal(t, http.StatusBadRequest, status, "body=%s", raw)
		assert.Equal(t, 1001, env.Code)
		assert.NotContains(t, strings.ToLower(raw), "internal")
	})

	t.Run("change_status 带 newStatus 走对应分析分支", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath+"?action=change_status&newStatus=closed", fx.userA)

		require.Equal(t, http.StatusOK, status, "body=%s", raw)
		assert.Equal(t, 0, env.Code)

		data := impactDTO(t, env)
		assert.Equal(t, "change_status", data["action"])
		assert.Equal(t, float64(2), data["affectedCount"])

		affected := data["affectedTickets"].([]any)
		first := affected[0].(map[string]any)
		assert.Equal(t, "status_change", first["impactType"])
		assert.Contains(t, first["description"], "closed")
	})

	t.Run("delete 把子工单标成孤立", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath+"?action=delete", fx.userA)

		require.Equal(t, http.StatusOK, status, "body=%s", raw)
		data := impactDTO(t, env)

		affected := data["affectedTickets"].([]any)
		require.Len(t, affected, 2)
		for _, item := range affected {
			assert.Equal(t, "orphaned", item.(map[string]any)["impactType"])
		}
	})

	t.Run("跨租户读父工单按不存在处理且不泄漏对方数据", func(t *testing.T) {
		status, env, raw := fx.do(t, basePath+"?action=close", fx.stranger)

		assert.Equal(t, http.StatusNotFound, status, "body=%s", raw)
		assert.Equal(t, 4004, env.Code)
		// 不得确认对方资源存在：响应里既没有工单号也没有标题，也没有底层 ent 错误文本。
		assert.NotContains(t, raw, "TKT-DEP-A-001")
		assert.NotContains(t, raw, "父工单")
		assert.NotContains(t, strings.ToLower(raw), "ticket not found")
	})

	t.Run("客户端自报 tenantId 不会被信任", func(t *testing.T) {
		selfReported := fmt.Sprintf("%s?action=close&tenantId=%d", basePath, fx.parentA.TenantID)
		status, env, raw := fx.do(t, selfReported, fx.stranger)

		assert.Equal(t, http.StatusNotFound, status, "body=%s", raw)
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("工单不存在是 404", func(t *testing.T) {
		status, env, raw := fx.do(t, fmt.Sprintf(dependencyImpactPathFmt, 987654)+"?action=close", fx.userA)

		assert.Equal(t, http.StatusNotFound, status, "body=%s", raw)
		assert.Equal(t, 4004, env.Code)
		assert.NotContains(t, strings.ToLower(raw), "sql")
	})

	t.Run("路径参数不是数字是 400", func(t *testing.T) {
		status, env, raw := fx.do(t, "/api/v1/tickets/not-a-number/dependencies?action=close", fx.userA)

		assert.Equal(t, http.StatusBadRequest, status, "body=%s", raw)
		assert.Equal(t, 1001, env.Code)
	})

	t.Run("未认证不能拿到影响分析", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, basePath+"?action=close", nil)
		w := httptest.NewRecorder()
		fx.engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
	})
}
