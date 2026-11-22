package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	authHandler "itsm-backend/handlers/auth"
	domainCommon "itsm-backend/handlers/common"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 公开组（public）的自助账号端点是企业合规的第一道记录面：
// 谁在什么时候从哪个 IP 注册了账号、申请了密码重置、真的重置了密码。
// 这些动作**都不在鉴权链之后**（无 JWT、无 tenant 上下文），因此 auth 组上
// 那层 AuditMiddleware 覆盖不到，此前零审计。
//
// 为什么**不能**直接给 public 组挂 AuditMiddleware：
//  1. /auth/login 已有专职 RecordLoginAudit（含失败原因、账户锁定等业务语义），
//     中间件会再写一条 → 每次登录两条记录，「登录失败次数」统计直接翻倍。
//  2. 中间件按 shouldAuditRequest 对**所有 POST** 生效，飞书/钉钉/企业微信
//     回调、CSRF token 获取都会被卷进审计表，噪音淹没真正的合规事件。
//  3. 中间件从 ctx 取 tenant_id/user_id，public 组恒为 0 → 无法归属租户。
// 故必须在 handler 内逐端点显式记录（allow-list），而不是整组粗暴挂载。

// newPublicAuditRouter 装配真实 Router（与生产同一入口），断言从真实入口出发。
//
// 必须手工注入 CommonHandler / AuthHandler：public 组的两段路由各自有
// `if config.XXXHandler != nil` 守卫，缺注入时端点直接 404，
// 测出来的「零审计」是「路由不存在」而非「审计缺口」——这是最容易自欺的一类假绿。
func newPublicAuditRouter(t *testing.T) (*gin.Engine, *ent.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_public_audit_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	commonRepo := domainCommon.NewEntRepository(client)
	commonSvc := domainCommon.NewService(commonRepo, "public-audit-secret", logger, client)
	authSvc := authHandler.NewService(client, "public-audit-secret", logger)

	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     "public-audit-secret",
		Logger:        logger,
		Client:        client,
		CommonHandler: domainCommon.NewHandler(commonSvc),
		AuthHandler:   authHandler.NewHandler(authSvc),
	})
	return r, client
}

func doPublicPOST(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestPublicSelfServiceEndpointsAreAudited(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name   string
		path   string
		body   string
		action string
		secret string
	}{
		{
			name:   "注册新账号",
			path:   "/api/v1/auth/register",
			body:   `{"username":"newbie","email":"newbie@example.com","password":"Str0ngPass2026!","passwordConfirm":"Str0ngPass2026!"}`,
			action: "register",
			secret: "Str0ngPass2026!",
		},
		{
			name:   "申请密码重置",
			path:   "/api/v1/auth/forgot-password",
			body:   `{"email":"who@example.com"}`,
			action: "forgot_password",
		},
		{
			name:   "执行密码重置",
			path:   "/api/v1/auth/reset-password",
			body:   `{"token":"tok-abc","email":"who@example.com","password":"N3wPass2026!","passwordConfirm":"N3wPass2026!"}`,
			action: "reset_password",
			secret: "N3wPass2026!",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := doPublicPOST(r, tc.path, tc.body)

			// 端点本身的业务结论不作断言（可能因租户未初始化/邮件未配置/令牌无效而失败）。
			// 这里锁的是「无论业务成败，这一笔动作都必须留痕」——爆破与撞库
			// 正是从大量失败请求里看出来的。
			logs, err := client.AuditLog.Query().Where(auditlog.PathEQ(tc.path)).All(ctx)
			require.NoError(t, err)
			require.Len(t, logs, 1, "公开自助端点必须留一条审计（0 条=合规缺口）：status=%d", w.Code)

			entry := logs[0]
			assert.Equal(t, tc.action, entry.Action, "action 必须能区分具体动作，不能一律记成 create")
			assert.Equal(t, "auth", entry.Resource)
			assert.Equal(t, http.MethodPost, entry.Method)
			assert.Equal(t, tc.path, entry.Path, "path 必须是真实路径，不能写成 /api/v1/auth/login")
			if tc.secret != "" {
				require.NotNil(t, entry.RequestBody)
				assert.NotContains(t, *entry.RequestBody, tc.secret, "密码明文不得落审计")
			}
			require.NotNil(t, entry.RequestBody)
			assert.NotContains(t, *entry.RequestBody, "tok-abc", "重置令牌不得落审计")
		})
	}
}

// 正控制：login 已有专职 RecordLoginAudit，若把中间件也挂到 public 组，
// 一次登录会落两条——「登录失败次数」这类安全指标会直接翻倍。
func TestLoginStillProducesExactlyOneAuditRow(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	w := doPublicPOST(r, "/api/v1/auth/login", `{"username":"ghost","password":"wrong-pass","tenantCode":"nope"}`)
	require.NotEqual(t, http.StatusOK, w.Code, "凭据必须被拒，否则测不到失败留痕路径")

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(logs), 1,
		"一次登录最多一条审计：%d 条说明 public 组被挂了 AuditMiddleware，与 RecordLoginAudit 重复", len(logs))
	assert.NotEmpty(t, logs, "失败的登录尝试必须留痕")
}

// 成功路径必须独立可证明。
//
// 上一条用例（TestPublicSelfServiceEndpointsAreAudited）在默认夹具下所有自助端点
// 都走**失败**分支：注册因「零活跃租户」返 503、重置因令牌无效返 401。
// 于是把 success 分支的审计调用摘掉后测试仍全绿——这正是「只测到失败分支
// 就以为整条路径都覆盖」的假绿。此处先建唯一活跃租户把注册推到成功，
// 让成功分支拥有自己的判别式。
func TestSuccessfulRegistrationIsAudited(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	// Register 要求「恰好一个活跃租户」，否则按 fail-closed 返 503。
	client.Tenant.Create().SetName("自助审计租户").SetCode("self-audit").
		SetDomain("self-audit.example.com").SetStatus("active").SaveX(ctx)

	w := doPublicPOST(r, "/api/v1/auth/register",
		`{"username":"freshuser","email":"fresh@example.com","password":"Str0ngPass2026!","passwordConfirm":"Str0ngPass2026!"}`)
	require.Equal(t, http.StatusOK, w.Code,
		"必须真的注册成功，否则测的仍是失败分支：body=%s", w.Body.String())

	logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/register")).All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1, "注册成功必须留一条审计")
	entry := logs[0]
	assert.Equal(t, "register", entry.Action)
	assert.Equal(t, http.StatusOK, entry.StatusCode, "成功分支的 status_code 必须是 200，不能沿用失败分支的默认值")
	// 用户已真的落库，审计必须能指向它——否则「谁注册了」无从追溯。
	assert.NotZero(t, entry.TenantID, "注册成功后租户已知，审计应带上租户归属")
	require.NotNil(t, entry.RequestBody)
	assert.NotContains(t, *entry.RequestBody, "Str0ngPass2026!", "注册密码不得落审计")
	assert.NotContains(t, *entry.RequestBody, "fresh@example.com", "邮箱属凭据面，审计只留用户名")
	assert.Contains(t, *entry.RequestBody, "freshuser")
}

// 负控制：public 组里的非账号类端点不得被卷进审计表。
func TestPublicNonAccountEndpointsAreNotAudited(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/csrf-token", nil)
	r.ServeHTTP(w, req)

	logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/csrf-token")).All(ctx)
	require.NoError(t, err)
	assert.Empty(t, logs, "CSRF token 获取不是合规事件，不该落审计")
}
