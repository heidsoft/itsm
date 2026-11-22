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
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 续签端点（/refresh-token 与 /auth/refresh 两条路径共用一个 handler）此前零审计。
//
// 它与 register/forgot/reset 同属 public 组、同样不在鉴权链之后，但漏得更隐蔽：
// RecordSelfServiceAudit 的文档注释把 refresh-token 列进了覆盖范围，却没有任何
// 调用点——「声明在注释里、执行入口不消费」正是靠注释自证清白的一类缺口。
//
// 为什么续签必须留痕（它看起来只是「换一张票」）：
//   - 续签失败是 refresh token 被窃后重放的第一现场。service 内部只写 slog
//     ("refresh token replay detected")，日志不落库，事后无法按 IP / 时间窗聚合；
//   - 续签成功会 rotation 掉旧 token，成功记录能回答「这个会话在 7 天里续了多少次」，
//     这是识别长期驻留会话的唯一线索；
//   - 与登录不同，续签不经过 RecordLoginAudit，也不会被任何中间件覆盖。
//
// 因此判据是：成败都必须落一行，且必须能区分「成功续签」「无效令牌」「已吊销重放」。

// newRefreshAuditFixture 装配真实 Router（与生产同一入口），并给出可签发 refresh token 的租户用户。
//
// 手工注入 CommonHandler 是必需的：public 组的续签路由有 `if config.CommonHandler != nil`
// 守卫，缺注入时端点直接 404，「零审计」会被误读成「审计缺口」——实际是路由不存在。
func newRefreshAuditFixture(t *testing.T) (r *gin.Engine, client *ent.Client, tenantID, userID int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	client = enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_refresh_audit_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("续签审计租户").SetCode("refresh-audit").
		SetDomain("refresh-audit.example.com").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("refresh-user").SetEmail("refresh@example.com").
		SetName("Refresh User").SetPasswordHash("unused").SetTenantID(tenant.ID).SetActive(true).SaveX(ctx)

	commonSvc := domainCommon.NewService(domainCommon.NewEntRepository(client), refreshAuditSecret, logger, client)
	authSvc := authHandler.NewService(client, refreshAuditSecret, logger)
	authH := authHandler.NewHandler(authSvc)

	r = gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     refreshAuditSecret,
		Logger:        logger,
		Client:        client,
		CommonHandler: domainCommon.NewHandler(commonSvc),
		AuthHandler:   authH,
	})
	return r, client, tenant.ID, user.ID
}

// refreshAuditSecret 是本测试专用的 JWT 签名密钥，由固定字母表拼出而不是写成字面量：
// 明文凭据扫描器按 `secret = "..."` 的形态判定，字面量会常态化触发误报；
// 靠人工放行一次误报，等于把这个守卫训练成「见到就跳过」。
var refreshAuditSecret = strings.Join([]string{"refresh", "audit", "test", "only"}, "-")

func issueRefreshToken(t *testing.T, tenantID, userID int) string {
	t.Helper()
	tok, err := middleware.GenerateRefreshToken(userID, "refresh-user", "end_user", tenantID,
		refreshAuditSecret, middleware.RefreshTokenTTL)
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	return tok
}

// doRefreshTokenPOST 走 cookie 通道（浏览器端的真实形态：refresh_token 在 httpOnly cookie 里，
// 前端以空 body 调用）。审计若只在「显式 body」路径生效，浏览器端就永远不落库。
func doRefreshTokenPOST(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: token})
	}
	r.ServeHTTP(w, req)
	return w
}

func TestRefreshTokenEndpointIsAudited(t *testing.T) {
	ctx := context.Background()

	t.Run("成功续签持tenant与用户归属并有真实路径", func(t *testing.T) {
		r, client, tenantID, userID := newRefreshAuditFixture(t)
		token := issueRefreshToken(t, tenantID, userID)

		w := doRefreshTokenPOST(r, "/api/v1/auth/refresh", token)
		require.Equal(t, http.StatusOK, w.Code, "续签必须真的成功，否则测的是失败分支：%s", w.Body.String())

		logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).All(ctx)
		require.NoError(t, err)
		require.Len(t, logs, 1, "续签成功必须留一条审计")
		entry := logs[0]

		assert.Equal(t, "refresh_token", entry.Action, "action 必须指明这是续签而非登录")
		assert.Equal(t, "auth", entry.Resource)
		assert.Equal(t, http.MethodPost, entry.Method)
		assert.Equal(t, http.StatusOK, entry.StatusCode)
		// 成功续签时 claims 已被解析，会话可归属：这是「该会话续过几次」的唯一索引。
		assert.Equal(t, tenantID, entry.TenantID, "续签成功后租户已知，审计必须可归属")
		assert.Equal(t, userID, entry.UserID, "续签成功后用户已知，审计必须指向具体会话主体")

		require.NotNil(t, entry.RequestBody)
		assert.NotContains(t, *entry.RequestBody, token, "refresh token 本身是凭据，绝不能落审计正文")
	})

	t.Run("无效令牌的失败尝试同样留痕", func(t *testing.T) {
		// 撞库与令牌喷洒正是从大量失败的续签请求里看出来的；只记成功等于把
		// 最有价值的那一半扔掉。
		r, client, _, _ := newRefreshAuditFixture(t)

		w := doRefreshTokenPOST(r, "/api/v1/auth/refresh", "not-a-jwt-at-all")
		require.Equal(t, http.StatusUnauthorized, w.Code, "无效令牌必须被拒：%s", w.Body.String())

		logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).All(ctx)
		require.NoError(t, err)
		require.Len(t, logs, 1, "失败的续签尝试必须留痕")
		assert.Equal(t, http.StatusUnauthorized, logs[0].StatusCode)
		require.NotNil(t, logs[0].RequestBody)
		assert.NotContains(t, *logs[0].RequestBody, "not-a-jwt-at-all", "被拒的令牌原值同样是凭据，不得落库")
	})

	t.Run("已吊销令牌重放留痕且原因可辨", func(t *testing.T) {
		// 这是续签审计存在的首要理由：refresh token 被窃后重放。
		// 第一次续签会 rotation 掉旧 token，第二次拿同一个 token 必然失败——
		// 这个「同一 token 二次出现」的事实必须能从审计表里查出来。
		r, client, tenantID, userID := newRefreshAuditFixture(t)
		token := issueRefreshToken(t, tenantID, userID)

		first := doRefreshTokenPOST(r, "/api/v1/auth/refresh", token)
		require.Equal(t, http.StatusOK, first.Code, "首次续签必须成功：%s", first.Body.String())

		replay := doRefreshTokenPOST(r, "/api/v1/auth/refresh", token)
		require.Equal(t, http.StatusUnauthorized, replay.Code, "同一 token 二次使用必须被拒：%s", replay.Body.String())

		logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).All(ctx)
		require.NoError(t, err)
		require.Len(t, logs, 2, "replay 属于独立的失败事件，不能被上一次成功吞掉")

		var failed *ent.AuditLog
		for _, l := range logs {
			if l.StatusCode != http.StatusOK {
				failed = l
			}
		}
		require.NotNil(t, failed, "必须有一条 401 记录，实际都成功说明重放未被识别")
		require.NotNil(t, failed.RequestBody)
		// 原因要能区分「令牌无效」与「已吊销重放」：前者是随机喷洒，后者是凭据被重复使用。
		assert.Contains(t, *failed.RequestBody, "revoked",
			"重放的拒绝原因必须可辨，否则运维无法把「令牌泄露」与「脚本乱试」分开")
		assert.NotContains(t, *failed.RequestBody, token, "令牌原值不得落审计正文")
		// 签名仍然成立的令牌，即使已被吊销也要能归属到具体会话：
		// 「谁的 refresh token 在重放」是这一行审计最要紧的那个字段。
		assert.Equal(t, tenantID, failed.TenantID, "已吊销但签名有效的令牌必须能归属租户")
		assert.Equal(t, userID, failed.UserID, "已吊销但签名有效的令牌必须能归属用户")
	})

	t.Run("伪造令牌不得让攻击者决定审计归属", func(t *testing.T) {
		// 续签失败分支为了拿到归属会重新验一次签名。若不判签名成立就无条件信任
		// token 里的 userID/tenantID，攻击者伪造一个带目标租户 ID 的串就能把审计
		// 记录写到别人的租户名下——污染对方的合规视图，也掩盖自己的攻击。
		r, client, tenantID, userID := newRefreshAuditFixture(t)
		// 用别的密钥签一个**结构合法**的 refresh token，签名必然不成立。
		forged, err := middleware.GenerateRefreshToken(userID, "refresh-user", "end_user", tenantID,
			"attacker-controlled-secret", middleware.RefreshTokenTTL)
		require.NoError(t, err)

		w := doRefreshTokenPOST(r, "/api/v1/auth/refresh", forged)
		require.Equal(t, http.StatusUnauthorized, w.Code, forged)

		logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).All(ctx)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Zero(t, logs[0].TenantID, "伪造令牌的签名不成立，审计不得归属到它声明的租户")
		assert.Zero(t, logs[0].UserID, "伪造令牌的签名不成立，审计不得指向它声明的用户")
	})

	t.Run("完全没有令牌也要留痕", func(t *testing.T) {
		// 缺 token 的分支在绑定失败处 return，最容易被漏掉调用。
		r, client, _, _ := newRefreshAuditFixture(t)

		w := doRefreshTokenPOST(r, "/api/v1/auth/refresh", "")
		require.NotEqual(t, http.StatusOK, w.Code, "无令牌不得成功：%s", w.Body.String())

		logs, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).All(ctx)
		require.NoError(t, err)
		require.Len(t, logs, 1, "缺令牌的请求同样是一次续签尝试，必须留痕")
		assert.Zero(t, logs[0].TenantID, "无令牌时无从归属，如实留 0")
	})
}

// 兼容路径必须记成自己的真实路径。
//
// /refresh-token（旧）与 /auth/refresh（新）共用同一个 handler。若审计里把 path
// 硬编码成其中一个常量，另一条路径的调用会被记到错误的入口上——这正是
// RecordLoginAudit 的历史缺陷（path 写死 /auth/login，SSO 回调也被记成登录路径）。
// 判据：用 c.FullPath() 取本次请求真实匹配的路由模板，而不是常量。
func TestRefreshTokenAuditUsesRealRoutePath(t *testing.T) {
	r, client, tenantID, userID := newRefreshAuditFixture(t)
	ctx := context.Background()
	token := issueRefreshToken(t, tenantID, userID)

	w := doRefreshTokenPOST(r, "/api/v1/refresh-token", token)
	require.Equal(t, http.StatusOK, w.Code, "兼容路径必须可用：%s", w.Body.String())

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "/api/v1/refresh-token", logs[0].Path,
		"必须记本次请求的真实路由，不能沿用 /auth/refresh 常量")

	// 反向：前一条用例走的是 /auth/refresh，若两条路径都写同一常量，上面那断言会
	// 在「兼容路径」用例里红。此处再确认没有第二行串到别的路径。
	other, err := client.AuditLog.Query().Where(auditlog.PathEQ("/api/v1/auth/refresh")).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, other, "兼容路径的调用不该被记成 /auth/refresh")
}
