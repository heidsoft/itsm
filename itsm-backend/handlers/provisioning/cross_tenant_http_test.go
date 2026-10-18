// Package provisioning 的跨租户 HTTP 回归测试。
//
// 交付任务（provisioning task）的启动/执行是「创建云资源」级别的高风险副作用，
// 隔离保证必须从真实生产入口（router.SetupRoutes 布线 + JWT 认证 + RBAC
// RequirePermission + handler + service + ent + durable command）发起验证，
// 而不是直接打 service。参照 router/ticket_attachment_route_test.go 的搭台方式。
//
// 已钉住的契约（按当前生产行为如实断言）：
//   - 同租户 owner 启动交付是正控制：任务 pending + SR 置 provisioning + 入箱一条命令；
//   - 跨租户启动交付 400/4000（handler 把所有失败统一映射为 BadRequest），
//     任务行、SR 状态、operational_commands 三处都必须零副作用；
//   - 跨租户与「资源不存在」的错误形态不可区分（均 400/4000）；
//   - 跨租户执行 500/5001（handler 把所有失败统一映射为 InternalError），
//     任务保持 pending、SR 不被推进；与不存在的任务同形态。
package provisioning_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	domainServiceRequest "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	provisioningHandler "itsm-backend/handlers/provisioning"
	serviceRequestHandler "itsm-backend/handlers/service_request"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User  // tenant A super_admin：能穿过 RBAC 正常操作本租户
	stranger *ent.User // tenant B super_admin：能穿过 RBAC，但资源全在租户 A
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:prov_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Prov A").SetCode("prov-a").SetDomain("prov-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Prov B").SetCode("prov-b").SetDomain("prov-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("prov-a-owner").SetEmail("prov-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("prov-b-stranger").SetEmail("prov-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	// SR handler 需要真实构造（provisioning 子路由挂在 service-requests 组内）。
	srHandler := serviceRequestHandler.NewHandler(serviceRequestHandler.NewService(
		serviceRequestHandler.NewEntRepository(client), nil, nil, client, logger, nil))
	provHandler := provisioningHandler.NewHandler(service.NewProvisioningService(client, logger), logger)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "provisioning-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:             secret,
		Logger:                logger,
		Client:                client,
		ServiceRequestHandler: srHandler,
		ProvisioningHandler:   provHandler,
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
	}
}

// newApprovedSR 建一条可直接启动交付的最小服务请求（security_approved 是
// CreateTaskFromServiceRequest 的状态门槛）。
func (fx *fixture) newApprovedSR(t *testing.T, title string) *ent.ServiceRequest {
	t.Helper()
	ctx := context.Background()
	catalog := fx.client.ServiceCatalog.Create().
		SetName("catalog for " + title).
		SetStatus("active").
		SetTenantID(fx.tenantA.ID).
		SaveX(ctx)
	return fx.client.ServiceRequest.Create().
		SetTitle(title).
		SetStatus(domainServiceRequest.StatusSecurityApproved).
		SetCatalogID(catalog.ID).
		SetRequesterID(fx.userA.ID).
		SetTenantID(fx.tenantA.ID).
		SaveX(ctx)
}

func (fx *fixture) newPendingTask(t *testing.T, sr *ent.ServiceRequest) *ent.ProvisioningTask {
	t.Helper()
	return fx.client.ProvisioningTask.Create().
		SetTenantID(fx.tenantA.ID).
		SetServiceRequestID(sr.ID).
		SetProvider("alicloud").
		SetResourceType("ecs").
		SetPayload(map[string]any{"resource_type": "ecs"}).
		SetStatus("pending").
		SaveX(context.Background())
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

func (fx *fixture) doNoAuth(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

// envelope 解出 {code, message, data} 信封。
func envelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), "body=%s", w.Body.String())
	return out
}

// envelopeItems 从 data 里取数组（列表负例断言零泄漏用）。
func envelopeItems(t *testing.T, w *httptest.ResponseRecorder) []any {
	t.Helper()
	data, _ := envelope(t, w)["data"].([]any)
	return data
}

func nTasks(t *testing.T, fx *fixture) int {
	t.Helper()
	rows, err := fx.client.ProvisioningTask.Query().All(context.Background())
	require.NoError(t, err)
	return len(rows)
}

func nCommands(t *testing.T, fx *fixture) int {
	t.Helper()
	rows, err := fx.client.OperationalCommand.Query().All(context.Background())
	require.NoError(t, err)
	return len(rows)
}

func TestProvisioningCrossTenant_HTTP(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()

	// ---------- 启动交付（POST /service-requests/:id/provision） ----------

	var srA *ent.ServiceRequest
	t.Run("同租户启动交付是正控制：任务+状态回写+入箱命令", func(t *testing.T) {
		req := fx.newApprovedSR(t, "SR 启动交付正控制")
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/provision", req.ID), fx.userA, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		tasks, err := fx.client.ProvisioningTask.Query().All(ctx)
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		assert.Equal(t, "pending", tasks[0].Status)
		assert.Equal(t, fx.tenantA.ID, tasks[0].TenantID)

		sr, err := fx.client.ServiceRequest.Get(ctx, req.ID)
		require.NoError(t, err)
		assert.Equal(t, "provisioning", sr.Status, "启动交付必须把 SR 推进到 provisioning")

		require.Equal(t, 1, nCommands(t, fx), "执行副作用必须经 durable command 入箱")
		srA = req
	})

	t.Run("跨租户启动交付 fail closed 且零副作用", func(t *testing.T) {
		require.NotNil(t, srA, "正控制必须先落库")
		beforeCmds := nCommands(t, fx)

		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/provision", srA.ID), fx.stranger, nil)
		// 已知语义（如实钉住）：handler 把全部失败统一映射 BadRequest(4000)，
		// 未区分 not-found；fail closed 成立但没有 404 语义。修复时应改为 404 并同步本断言。
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4000`)
		assert.NotContains(t, w.Body.String(), `"task"`, "失败响应不得内嵌任何任务数据")

		require.Equal(t, beforeCmds, nCommands(t, fx), "跨租户启动不得入箱任何命令")
		require.Equal(t, 1, nTasks(t, fx), "跨租户启动不得创建任何任务行")

		updated, err := fx.client.ServiceRequest.Get(ctx, srA.ID)
		require.NoError(t, err)
		assert.Equal(t, "provisioning", updated.Status, "SR 状态不得被跨租户请求改变")
	})

	t.Run("跨租户与不存在资源启动交付的错误形态不可区分", func(t *testing.T) {
		w1 := fx.do(t, http.MethodPost, "/api/v1/service-requests/999999/provision", fx.stranger, nil)
		require.Equal(t, http.StatusBadRequest, w1.Code)
		require.Contains(t, w1.Body.String(), `"code":4000`)
		w2 := fx.do(t, http.MethodPost, "/api/v1/service-requests/999998/provision", fx.stranger, nil)
		assert.Equal(t, w1.Code, w2.Code)
		assert.Equal(t, w1.Body.String(), w2.Body.String(),
			"租户外资源与任意不存在资源在观测上应完全同构，避免靠响应差异枚举资源")
	})

	// ---------- 执行任务（POST /provisioning-tasks/:id/execute） ----------

	t.Run("跨租户执行 fail closed 且状态不被推进；同租户执行到 succeeded", func(t *testing.T) {
		srE := fx.newApprovedSR(t, "SR 交付执行隔离")
		taskE := fx.newPendingTask(t, srE)
		beforeCmds := nCommands(t, fx)

		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/provisioning-tasks/%d/execute", taskE.ID), fx.stranger, nil)
		// 已知缺陷（如实钉住）：handler 把全部失败统一映射 InternalErrorCode(5001)，
		// not-found 也一样；跨租户探测表现为 500。修复时应改为 404/4004 并同步本断言。
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":5001`)
		assert.NotContains(t, w.Body.String(), "ecs", "失败响应不得内嵌任务 payload")

		task, err := fx.client.ProvisioningTask.Get(ctx, taskE.ID)
		require.NoError(t, err)
		assert.Equal(t, "pending", task.Status, "跨租户执行不得把任务推进到 running/succeeded")
		sr, err := fx.client.ServiceRequest.Get(ctx, srE.ID)
		require.NoError(t, err)
		assert.Equal(t, domainServiceRequest.StatusSecurityApproved, sr.Status, "SR 状态不得被跨租户执行推进")

		// 与不存在的任务同形态
		w2 := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/999999/execute", fx.stranger, nil)
		assert.Equal(t, w.Code, w2.Code)
		assert.Equal(t, w.Body.String(), w2.Body.String())
		require.Equal(t, beforeCmds, nCommands(t, fx), "跨租户执行不得产生任何命令行")

		// 正控制：owner 执行自己的任务，真实推进 task + SR
		wOwn := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/provisioning-tasks/%d/execute", taskE.ID), fx.userA, nil)
		require.Equal(t, http.StatusOK, wOwn.Code, "body=%s", wOwn.Body.String())
		require.Contains(t, wOwn.Body.String(), `"code":0`)

		task, err = fx.client.ProvisioningTask.Get(ctx, taskE.ID)
		require.NoError(t, err)
		assert.Equal(t, "succeeded", task.Status)
		sr, err = fx.client.ServiceRequest.Get(ctx, srE.ID)
		require.NoError(t, err)
		assert.Equal(t, "delivered", sr.Status)
	})

	// ---------- 任务列表（GET /service-requests/:id/provisioning-tasks） ----------

	t.Run("跨租户任务列表零泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/provisioning-tasks", srA.ID), fx.stranger, nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"code":0`)
		assert.Empty(t, envelopeItems(t, w), "租户外的任务是查询黑洞，不是列表可见项")

		wOwn := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/provisioning-tasks", srA.ID), fx.userA, nil)
		require.Len(t, envelopeItems(t, wOwn), 1, "正控制：owner 必须能看到自己的任务")
	})

	t.Run("未认证请求 401", func(t *testing.T) {
		w := fx.doNoAuth(t, http.MethodPost, "/api/v1/service-requests/1/provision")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), fmt.Sprintf(`"code":%d`, common.AuthFailedCode))
	})
}
