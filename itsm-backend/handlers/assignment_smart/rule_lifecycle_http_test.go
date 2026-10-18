// Package assignment_smart 的规则协议写面 HTTP 回归（B9.1）。
//
// 读面信封与推荐/分配推荐契约已由 router/ticket_rule_attachment_list_route_test.go
// 覆盖；本文件聚焦**写面与错误语义**：
//   - 规则 CRUD 经真实 ticket 路由（/api/v1/tickets/assignment-rules*），
//     权限码 assignment_rule:{read,write}（§1.41）与 ticket:assign（动作面）。
//   - [Z1 已修复] DELETE /tickets/assignment-rules/{id} 条件删除验 affected
//     rows，不存在/跨租户/重复删除 → 404/4004。
//   - [Z2 已修复] POST /tickets/assignment-rules/test 的 rule 跨租户/不存在 →
//     404/4004（sentinel ErrAssignmentRuleNotFound）。
//   - [Z3 已修复] 同接口的 ticketId 补租户谓词：他租工单不再被读取匹配；
//     跨租户工单 → 404/4004（sentinel ErrTestTicketNotFound）。
//   - UpdateAssignmentRule 用 UpdateOneID().Where(tenant) 条件更新，
//     0 行命中返回 *ent.NotFoundError → classifyError 404/4004。
package assignment_smart_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketassignmentrule"
	assignmentSmartHandler "itsm-backend/handlers/assignment_smart"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	endUser *ent.User
	ruleA   *ent.TicketAssignmentRule
	ruleB   *ent.TicketAssignmentRule
	ticketA *ent.Ticket
	ticketB *ent.Ticket
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:assignment_rule_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Rule A").SetCode("rule-a").SetDomain("rule-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Rule B").SetCode("rule-b").SetDomain("rule-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("rule-a-owner").SetEmail("rule-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("rule-a-enduser").SetEmail("rule-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	mkRule := func(name string, tenant *ent.Tenant) *ent.TicketAssignmentRule {
		return client.TicketAssignmentRule.Create().
			SetName(name).
			SetPriority(10).
			SetConditions(nil).
			SetActions(map[string]interface{}{"type": "user", "value": float64(userA.ID)}).
			SetIsActive(true).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}
	ruleA := mkRule("租户A规则", tenantA)
	ruleB := mkRule("租户B规则", tenantB)

	mkTicket := func(number, title string, tenant *ent.Tenant, priority string) *ent.Ticket {
		return client.Ticket.Create().
			SetTicketNumber(number).
			SetTitle(title).
			SetStatus("open").
			SetPriority(priority).
			SetRequesterID(userA.ID).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}
	ticketA := mkTicket("TK-A-001", "租户A工单", tenantA, "high")
	ticketB := mkTicket("TK-B-001", "租户B工单", tenantB, "high")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "assignment-rule-secret"
	ruleSvc := service.NewTicketAssignmentRuleService(client, logger)
	smartSvc := service.NewTicketAssignmentSmartService(client, logger, service.NewTicketAssignmentService(client, logger), ruleSvc)
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:                    secret,
		Logger:                       logger,
		Client:                       client,
		TicketAssignmentSmartHandler: assignmentSmartHandler.NewHandler(smartSvc, ruleSvc, logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, endUser: endUser,
		ruleA: ruleA, ruleB: ruleB,
		ticketA: ticketA, ticketB: ticketB,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, payload ...string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if len(payload) > 0 {
		reader = strings.NewReader(payload[0])
	}
	req := httptest.NewRequest(method, path, reader)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

func TestAssignmentRuleLifecycle(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/tickets/assignment-rules"
	ctx := context.Background()

	t.Run("同租创建规则正控制camelCase", func(t *testing.T) {
		payload := `{"name":"新规则","priority":5,"conditions":[{"field":"priority","operator":"eq","value":"high"}],"actions":{"type":"user","value":` + fmt.Sprint(fx.userA.ID) + `},"isActive":true}`
		w := fx.do(t, http.MethodPost, base, fx.userA, payload)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		require.Contains(t, w.Body.String(), `"isActive"`, "字段必须是 camelCase")
		require.Contains(t, w.Body.String(), `"executionCount"`)
		require.NotContains(t, w.Body.String(), `"is_active"`, "不得出现 snake_case 字段")
	})

	t.Run("本租Update正控制且跨租Update被条件谓词拦截", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.ruleA.ID), fx.userA, `{"name":"改名A"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"name":"改名A"`)

		// 跨租户更新：UpdateOneID().Where(tenant) 命中 0 行 → *ent.NotFoundError
		// → classifyError 404/4004；对端规则名不得改变。
		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.ruleB.ID), fx.userA, `{"name":"越权改名"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		b, err := fx.client.TicketAssignmentRule.Get(ctx, fx.ruleB.ID)
		require.NoError(t, err)
		require.Equal(t, "租户B规则", b.Name, "跨租户更新不得触碰对端")
	})

	t.Run("跨租户Delete是404掩码且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, fx.ruleB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		n, err := fx.client.TicketAssignmentRule.Query().Where(ticketassignmentrule.TenantIDEQ(fx.tenantB.ID)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, n, "跨租户删除不得注销对端行")
	})

	t.Run("跨租户Test规则是404掩码", func(t *testing.T) {
		payload := `{"ruleId":` + fmt.Sprint(fx.ruleB.ID) + `,"ticketId":` + fmt.Sprint(fx.ticketA.ID) + `}`
		w := fx.do(t, http.MethodPost, base+"/test", fx.userA, payload)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("跨租户Test工单是404掩码零读取", func(t *testing.T) {
		payload := `{"ruleId":` + fmt.Sprint(fx.ruleA.ID) + `,"ticketId":` + fmt.Sprint(fx.ticketB.ID) + `}`
		w := fx.do(t, http.MethodPost, base+"/test", fx.userA, payload)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("auto-assign与recommendations跨租户工单404", func(t *testing.T) {
		// 预先钉好的语义（service 已用 NewBusinessError 404），锁定防回归。
		for _, probe := range []struct {
			method, path string
		}{
			{http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/auto-assign", fx.ticketB.ID)},
			{http.MethodGet, fmt.Sprintf("/api/v1/tickets/assign-recommendations/%d", fx.ticketB.ID)},
		} {
			w := fx.do(t, probe.method, probe.path, fx.userA)
			require.Equal(t, http.StatusNotFound, w.Code, "path=%s body=%s", probe.path, w.Body.String())
			require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		}
	})

	t.Run("同租重复删除第二次是404不再假装成功", func(t *testing.T) {
		rule := fx.client.TicketAssignmentRule.Create().SetName("待删规则").SetIsActive(true).SetTenantID(fx.tenantA.ID).SaveX(ctx)
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, rule.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, rule.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("规则面权限码403禁拒", func(t *testing.T) {
		// §1.41：规则读取需 assignment_rule:read（end_user 不持有）。
		w := fx.do(t, http.MethodGet, base, fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)

		// 指派动作面 ticket:assign：end_user 也不持有。
		w = fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/auto-assign", fx.ticketA.ID), fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
