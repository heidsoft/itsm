// Package escalation_matrix 的 HTTP 契约回归（B9.1）。
//
// service 层语义（深拷贝/先落库后缓存/降级）已由 service/escalation_matrix_test.go
// 与 escalation_matrix_persistence_test.go 覆盖；本文件经真实路由锁定 HTTP 面：
//   - 权限码 sla:{read,write}（升级矩阵属 SLA 配置面）。
//   - GET /escalation-matrices 未配置租户返回默认矩阵（非 nil）。
//   - PUT → GET 往返：写入的自定义级别原样读回（PUT 是覆盖写语义）。
//   - 多租户隔离：tenantA 自定义配置对 tenantB 不可见（读走租户谓词 store）。
//   - POST /escalation-matrices/invalidate-cache 清缓存后 GET 仍命中已落库
//     配置（store 路径生效，配置不丢）。
//   - PUT 非法 JSON → 400 参数错误。
package escalation_matrix_test

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
	approvalChainHandler "itsm-backend/handlers/approval_chain"
	escalationMatrixHandler "itsm-backend/handlers/escalation_matrix"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine    *gin.Engine
	client    *ent.Client
	secret    string
	tenantA   *ent.Tenant
	tenantB   *ent.Tenant
	userA     *ent.User
	userB     *ent.User
	endUser   *ent.User
	agentUser *ent.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:escalation_matrix_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Matrix A").SetCode("matrix-a").SetDomain("matrix-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Matrix B").SetCode("matrix-b").SetDomain("matrix-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("matrix-a-owner").SetEmail("matrix-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("matrix-b-owner").SetEmail("matrix-b@example.com").SetName("Owner B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("matrix-a-enduser").SetEmail("matrix-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	// 「有 sla:read、无 sla:write」的读写分权样本（end_user 的 sla:read 已退役）。
	agentUser := client.User.Create().SetUsername("matrix-a-agent").SetEmail("matrix-a-agent@example.com").SetName("Agent A").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "escalation-matrix-secret"
	matrixSvc := service.NewEscalationMatrixServiceWithStore(logger, service.NewSystemConfigEscalationMatrixStore(client, logger))
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		// 升级矩阵路由挂在 SetupApprovalChainRoutes 内，外层 gate 要求
		// ApprovalChainHandler != nil（与生产装配一致）。
		ApprovalChainHandler:    approvalChainHandler.NewHandler(service.NewApprovalChainService(client, logger), logger),
		EscalationMatrixHandler: escalationMatrixHandler.NewHandler(logger, matrixSvc),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, userB: userB, endUser: endUser, agentUser: agentUser,
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

func TestEscalationMatrixHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/escalation-matrices"

	t.Run("未配置租户返回默认矩阵非nil", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		require.NotEqual(t, "null", string(env.Data), "默认矩阵不得是 null")
		require.Contains(t, w.Body.String(), `"afterMinutes"`, "字段必须是 camelCase")
		require.Contains(t, w.Body.String(), `"notifyUserIDs"`)
		require.NotContains(t, w.Body.String(), `"after_minutes"`, "不得出现 snake_case 字段")
	})

	t.Run("PUT定制矩阵并GET往返", func(t *testing.T) {
		payload := `{"critical":[{"level":7,"afterMinutes":9,"notifyRoles":["manager"],"notifyUserIDs":[],"description":"自制矩阵"}],"high":[],"medium":[],"low":[]}`
		w := fx.do(t, http.MethodPut, base, fx.userA, payload)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)

		w = fx.do(t, http.MethodGet, base, fx.userA)
		env := decodeEnvelope(t, w)
		var matrix map[string][]struct {
			Level        int    `json:"level"`
			AfterMinutes int    `json:"afterMinutes"`
			Description  string `json:"description"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &matrix), "data=%s", string(env.Data))
		require.Len(t, matrix["critical"], 1, "自定义矩阵的 critical 链应原样读回")
		require.Equal(t, 7, matrix["critical"][0].Level)
		require.Equal(t, 9, matrix["critical"][0].AfterMinutes)
		require.Equal(t, "自制矩阵", matrix["critical"][0].Description)
	})

	t.Run("多租户隔离定制配置不外泄", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base, fx.userB)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		require.NotContains(t, w.Body.String(), "自制矩阵", "tenantB 不得读到 tenantA 的定制矩阵")
		require.Contains(t, w.Body.String(), `"afterMinutes":15`, "tenantB 仍是默认矩阵（15 分钟首级）")
	})

	t.Run("invalidate-cache后配置不丢store路径生效", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base+"/invalidate-cache", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		w = fx.do(t, http.MethodGet, base, fx.userA)
		require.Contains(t, w.Body.String(), "自制矩阵", "清缓存后 GET 仍命中 system_configs 落库配置")
	})

	t.Run("PUT非法JSON是400参数错误", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, base, fx.userA, `{bad json`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.NotEqual(t, 0, decodeEnvelope(t, w).Code)
	})

	t.Run("矩阵面权限码读写分权", func(t *testing.T) {
		// sla:{read,write}。end_user 的 sla:read 已随 2026-10-26 自助门户收敛
		// 退役（管理面读码整体撤出，RetiredRolePermissionCodes 有登记），
		// 不能再当「有读无写」样本。改用 agent：authz 默认集持 sla:read
		// （坐席需看 SLA 时限）不持 sla:write → GET 200、PUT 403。
		w := fx.do(t, http.MethodGet, base, fx.agentUser)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		w = fx.do(t, http.MethodPut, base, fx.agentUser, `{}`)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
