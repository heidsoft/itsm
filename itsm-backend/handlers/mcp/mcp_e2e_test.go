package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/config"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/mcp"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/bootstrap"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"
	"itsm-backend/pkg/seeder"
	"itsm-backend/router"
	"itsm-backend/service/marketplace"
)

// 这组端到端测试验证 MCP 通道的**真实请求链路**，而不是 service 层的替身行为。
//
// 为什么必须端到端：契约测试（mcp_contract_test.go）只覆盖 handlers/mcp.Service，
// 它绕过了真实链路上的所有中间件与装配。一旦真实链路出问题，契约测试全绿也发现不了：
//   - 路由是否真的注册（router 装配遗漏 → 端点 404）
//   - RBAC 是否真的拦截（mcp:read 未授权 → 应 403）
//   - principal 是否真的注入 request context（Gate 2.5 的读取点）
//   - CSRF 是否阻断 Bearer 客户端（Claude Desktop 等无法处理 CSRF token 流程）
//   - service principal 是否被 AuthMiddleware 的二次校验挡住
//   - 工具执行是否真的落到既有执行链并落审计
//
// 这些都是「骨架交付后最该验证但最容易漏」的部分。

const e2eJWTSecret = "mcp-e2e-jwt-secret"

// e2eFixture 是端到端测试的装配夹具。
type e2eFixture struct {
	router     *gin.Engine
	client     *ent.Client
	tenant     int
	sysadminID int
	endUserID  int
}

// setupMCPEndToEnd 装配真实 router（真实中间件链 + 真实 authz + 真实 seeder）。
func setupMCPEndToEnd(t *testing.T) *e2eFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// seeder 需要这两个环境变量（播种管理员与初始用户）。
	t.Setenv("ADMIN_PASSWORD", "mcp-e2e-admin-password")
	t.Setenv("SEED_USER1_PASSWORD", "mcp-e2e-user-password")

	logger := zaptest.NewLogger(t).Sugar()
	client := enttest.Open(t, "sqlite3", "file:mcp_e2e?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	// 播种：租户、角色、权限码、菜单。RBAC 中间件从 DB 读角色权限，
	// 没有播种数据则一切请求都会被判「用户不存在」或「无权限」。
	sd := seeder.NewSeeder(client, logger, &config.Config{
		Deployment: config.DeploymentConfig{Mode: "standalone"},
	})
	sd.SeedAll(context.Background())

	// 运行期权限目录（Gate 2.5 读它）必须装配，否则非 human 主体判定拿不到策略。
	authz.ConfigureShared(client)

	// 真实工具面 + 真实执行链：ToolRegistry 提供工具定义，ai.Service 承载
	// Gate 2 / Gate 2.5 / 审批 / 审计。缺依赖的工具在本次断言里不涉及。
	toolRegistry := service.NewToolRegistry(nil, nil, nil, client)
	aiService := ai.NewService(
		ai.NewEntRepository(client),
		logger,
		nil,
		toolRegistry,
		nil, nil, nil, nil, nil, nil,
	)
	marketplaceSvc := marketplace.NewService(client, logger)

	// 装配入口与生产同一条路径（bootstrap.BuildMCPHandler），
	// 测试自己拼一套接线的话，测的就不是生产真正跑的那条路径。
	mcpHandler, err := bootstrap.BuildMCPHandler(bootstrap.Dependencies{
		VisibilityChecker: visibilityFunc(func(context.Context, string, string, string, int) bool { return true }),
		ToolRegistry:      toolRegistry,
		AIService:         aiService,
		Marketplace:       marketplaceSvc,
		Logger:            logger,
	})
	require.NoError(t, err, "MCP handler 装配失败")

	f := &e2eFixture{tenant: 1, client: client}
	f.router = gin.New()
	router.SetupRoutes(f.router, &router.RouterConfig{
		JWTSecret: e2eJWTSecret,
		Logger:    logger,
		Client:    client,
		// CSRF 必须显式开启：RouterConfig.CSRFEnabled 默认 false，
		// 不开则 CSRF 中间件根本不挂载，「cookie 会话被拒」这条断言会假绿。
		// 生产由 config 装配决定（config.CSRFEnabled）。
		CSRFEnabled: true,
		MCPHandler:  mcpHandler,
	})
	f.createRoleUsers(t)
	return f
}

// createRoleUsers 建两个持具体角色的用户，供 RBAC 正反用例使用。
//
// 为什么不用 seeder 播的 admin：它的角色是 super_admin，走 `*:*` 旁路
// （架构地图已记载），用它测 RBAC 拦截会假绿 —— 放行不是因为权限对，
// 而是因为旁路。必须建持真实角色的用户，拦截才有判别力。
//
// end_user 由 seeder guard 反向锁保证不持 mcp:read，因此它必须 403；
// sysadmin 经 allPermissionCodes() 全量派生，天然持 mcp:read。
func (f *e2eFixture) createRoleUsers(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	create := func(username string, role user.Role) int {
		u, err := f.client.User.Create().
			SetUsername(username).
			SetEmail(username + "@mcp-e2e.local").
			SetName(username).
			SetRole(role).
			SetPasswordHash("mcp-e2e-not-a-real-hash").
			SetActive(true).
			SetTenantID(f.tenant).
			Save(ctx)
		require.NoError(t, err, "创建测试用户失败: %s", username)
		return u.ID
	}
	f.sysadminID = create("mcp-e2e-sysadmin", domainrole.SysAdmin)
	f.endUserID = create("mcp-e2e-enduser", domainrole.EndUser)
}

// mcpCall 发一条 JSON-RPC 请求并返回解析后的响应。
func mcpCall(t *testing.T, f *e2eFixture, token string, method string, params interface{}, extraHeaders map[string]string) (int, map[string]interface{}) {
	t.Helper()
	body := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method}
	if params != nil {
		body["params"] = params
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		// MCP 客户端的标准认证方式：Authorization Bearer。
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	var parsed map[string]interface{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("响应不是合法 JSON: %v (body=%s)", err, rec.Body.String())
		}
	}
	return rec.Code, parsed
}

// resultOf 从 JSON-RPC 响应里取出 result 对象。
func resultOf(t *testing.T, parsed map[string]interface{}) map[string]interface{} {
	t.Helper()
	require.NotNil(t, parsed, "响应为空")
	if e, ok := parsed["error"]; ok {
		t.Fatalf("JSON-RPC 错误: %v", e)
	}
	res, ok := parsed["result"].(map[string]interface{})
	require.True(t, ok, "result 缺失或类型不对: %v", parsed)
	return res
}

// ---------------------------------------------------------------------------
// 端到端：生命周期与工具面
// ---------------------------------------------------------------------------

func TestMCPEndToEnd_InitializeOverHTTP(t *testing.T) {
	f := setupMCPEndToEnd(t)

	// 端点必须真的注册（装配遗漏会 404）。
	code, parsed := mcpCall(t, f, "", mcp.MethodInitialize, map[string]interface{}{
		"protocolVersion": "2025-06-18",
		"clientInfo":      map[string]interface{}{"name": "e2e-probe", "version": "1.0.0"},
	}, nil)

	// 无凭据 → 401（认证中间件拦截），证明路由已注册且受保护。
	require.Equal(t, http.StatusUnauthorized, code, "MCP 端点必须要求认证")
	require.NotNil(t, parsed)
}

func TestMCPEndToEnd_ToolsListReturnsRealCatalog(t *testing.T) {
	f := setupMCPEndToEnd(t)
	token := issueHumanToken(t, f.tenant, f.sysadminID, "sysadmin")

	code, parsed := mcpCall(t, f, token, mcp.MethodToolsList, nil, nil)
	require.Equal(t, http.StatusOK, code, "持 mcp:read 的角色应能列举工具面")

	res := resultOf(t, parsed)
	tools, ok := res["tools"].([]interface{})
	require.True(t, ok, "result.tools 必须是数组")
	require.NotEmpty(t, tools, "工具面不应为空（真实 ToolRegistry 有工具）")

	// 第一个工具的形状必须符合 MCP 规范。
	first, ok := tools[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, first["name"], "工具必须有 name")
	assert.NotEmpty(t, first["description"], "工具必须有 description")
	schema, ok := first["inputSchema"].(map[string]interface{})
	require.True(t, ok, "工具必须有 inputSchema")
	assert.Equal(t, "object", schema["type"], "inputSchema 必须是 object 类型")
}

// ---------------------------------------------------------------------------
// 端到端：权限拦截
// ---------------------------------------------------------------------------

func TestMCPEndToEnd_UnauthorizedRoleRejected(t *testing.T) {
	f := setupMCPEndToEnd(t)
	// end_user 不持 mcp:read（seeder guard 反向锁锁定）。
	token := issueHumanToken(t, f.tenant, f.endUserID, "end_user")

	code, _ := mcpCall(t, f, token, mcp.MethodToolsList, nil, nil)
	assert.Equal(t, http.StatusForbidden, code, "无 mcp:read 的角色必须被 RBAC 拦截")
}

func TestMCPEndToEnd_UnknownUserRejected(t *testing.T) {
	f := setupMCPEndToEnd(t)
	// JWT 合法但 DB 中无此用户 → RBAC 中间件应判「用户不存在」。
	token := issueHumanToken(t, f.tenant, 999999, "sysadmin")

	code, _ := mcpCall(t, f, token, mcp.MethodToolsList, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, code, "DB 中不存在的用户必须被拒绝")
}

// ---------------------------------------------------------------------------
// 端到端：CSRF 与 Bearer 客户端兼容性
// ---------------------------------------------------------------------------

func TestMCPEndToEnd_BearerClientNotBlockedByCSRF(t *testing.T) {
	f := setupMCPEndToEnd(t)
	token := issueHumanToken(t, f.tenant, f.sysadminID, "sysadmin")

	// MCP 客户端（Claude Desktop 等）只发 Authorization 头，无法处理 CSRF token 流程。
	// 若 CSRF 中间件对 Bearer 也拦，这个通道对外就是不可用的。
	code, parsed := mcpCall(t, f, token, mcp.MethodToolsList, nil, nil)
	require.Equal(t, http.StatusOK, code, "Bearer 认证不应被 CSRF 拦截（MCP 客户端无法处理 CSRF 流程）")
	require.NotNil(t, parsed["result"])
}

func TestMCPEndToEnd_CookieSessionWithoutCSRFTokenRejected(t *testing.T) {
	f := setupMCPEndToEnd(t)
	token := issueHumanToken(t, f.tenant, f.sysadminID, "sysadmin")

	// 反向：cookie 会话必须照常校验 CSRF（浏览器会自动附带 cookie，
	// 若按「有 cookie 就跳过」放行，CSRF 保护等于失效）。
	payload, err := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": mcp.MethodToolsList})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusOK, rec.Code, "cookie 会话缺 CSRF token 必须被拒绝")
}

// ---------------------------------------------------------------------------
// 端到端：service principal 路径
// ---------------------------------------------------------------------------

func TestMCPEndToEnd_ServicePrincipalTokenAccepted(t *testing.T) {
	f := setupMCPEndToEnd(t)
	// service JWT：principal=service、user_id=0、role=audit_admin（只读载体角色）。
	token, err := middleware.GenerateServiceToken("mcp-e2e-svc", middleware.ServicePrincipalRole, f.tenant, e2eJWTSecret, time.Hour)
	require.NoError(t, err)

	code, parsed := mcpCall(t, f, token, mcp.MethodToolsList, nil, nil)
	require.Equal(t, http.StatusOK, code, "service 主体持 audit_admin（持 mcp:read）应能发现工具面")
	require.NotNil(t, parsed["result"])
}

func TestMCPEndToEnd_ServicePrincipalCannotImpersonateHuman(t *testing.T) {
	f := setupMCPEndToEnd(t)
	// service 主体签发时 role 必须是只读载体角色，签发端自行拒绝误配。
	_, err := middleware.GenerateServiceToken("mcp-e2e-svc", "sysadmin", f.tenant, e2eJWTSecret, time.Hour)
	require.Error(t, err, "service token 不得配高权角色")

	// tenant 必填，禁自报。
	_, err = middleware.GenerateServiceToken("mcp-e2e-svc", middleware.ServicePrincipalRole, 0, e2eJWTSecret, time.Hour)
	require.Error(t, err, "service token 必须绑定固定租户")
}

// issueHumanToken 签发一个 human 主体 JWT。
func issueHumanToken(t *testing.T, tenantID, userID int, role string) string {
	t.Helper()
	token, err := middleware.GenerateAccessToken(userID, "e2e-user", role, tenantID, e2eJWTSecret, time.Hour)
	require.NoError(t, err)
	return token
}
