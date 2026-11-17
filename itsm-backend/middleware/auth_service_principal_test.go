package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"itsm-backend/internal/authz"
)

const serviceTokenTestSecret = "service-principal-test-secret"

// signRawClaims 用测试密钥签发任意 Claims，用于构造「伪造主体标记」的负例。
func signRawClaims(t *testing.T, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(serviceTokenTestSecret))
	require.NoError(t, err)
	return s
}

func runAuthMiddleware(t *testing.T, token string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/ai/chat", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	AuthMiddleware(serviceTokenTestSecret)(c)
	return c, w
}

// 正向：签发工具产出的 service token 必须通过认证，并把 PrincipalService
// 同时写进 gin key 与 request context（Gate 2.5 从后者读）。
func TestAuthMiddleware_ServicePrincipalInjected(t *testing.T) {
	token, err := GenerateServiceToken("svc-ai-sidecar", ServicePrincipalRole, 7, serviceTokenTestSecret, time.Hour)
	require.NoError(t, err)

	c, w := runAuthMiddleware(t, token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, authz.PrincipalService, c.GetString("principal"))
	assert.Equal(t, ServicePrincipalRole, c.GetString("role"))
	assert.Equal(t, 7, c.GetInt("tenant_id"))
	assert.Equal(t, 0, c.GetInt("user_id"), "service 不是人类用户，user_id 必须为 0")
	assert.Equal(t, authz.PrincipalService,
		authz.PrincipalFromContext(c.Request.Context()),
		"Gate 2.5 必须能从 request context 读到 service 主体")
}

// 存量 human token 不受影响：principal 缺省按 human 注入，行为与 Phase A 一致。
func TestAuthMiddleware_HumanTokenUnaffected(t *testing.T) {
	token, err := GenerateAccessToken(42, "alice", "manager", 3, serviceTokenTestSecret, time.Hour)
	require.NoError(t, err)

	c, w := runAuthMiddleware(t, token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, authz.PrincipalHuman, c.GetString("principal"))
	assert.Equal(t, authz.PrincipalHuman, authz.PrincipalFromContext(c.Request.Context()))
	assert.Equal(t, 42, c.GetInt("user_id"))
}

// 负向：principal 必须是闭集内的值。未知标记（即使签名正确）必须拒绝，
// 否则签发端拼错字符串会静默降级成绕过零信任的特权主体。
func TestAuthMiddleware_UnknownPrincipalRejected(t *testing.T) {
	token := signRawClaims(t, Claims{
		UserID: 1, Username: "x", Role: "admin", TenantID: 3, TokenType: "access",
		Principal: "super-service",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	_, w := runAuthMiddleware(t, token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "无效的主体标记")
}

// 负向：service 主体不得携带高权限角色/非零 user_id/缺租户。
// 这一条锁定「签发端误配不提权」——只放行只读载体角色。
func TestAuthMiddleware_ServiceMustBeReadOnly(t *testing.T) {
	cases := []struct {
		name   string
		role   string
		userID int
		tenant int
	}{
		{"高权限角色", "sysadmin", 0, 3},
		{"非零用户ID", ServicePrincipalRole, 9, 3},
		{"缺租户", ServicePrincipalRole, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := signRawClaims(t, Claims{
				UserID: tc.userID, Username: "svc", Role: tc.role, TenantID: tc.tenant,
				TokenType: "access", Principal: authz.PrincipalService,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
			})
			_, w := runAuthMiddleware(t, token)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "服务主体凭证不合法")
		})
	}
}

// 签发函数自身的约束：非法参数直接报错，不能产出「看起来合法」的凭证。
func TestGenerateServiceToken_RejectsInvalidArgs(t *testing.T) {
	_, err := GenerateServiceToken("svc", ServicePrincipalRole, 0, serviceTokenTestSecret, time.Hour)
	require.Error(t, err, "缺租户必须拒绝")

	_, err = GenerateServiceToken("svc", ServicePrincipalRole, 1, serviceTokenTestSecret, 0)
	require.Error(t, err, "非正 TTL 必须拒绝")

	_, err = GenerateServiceToken("svc", "sysadmin", 1, serviceTokenTestSecret, time.Hour)
	require.Error(t, err, "非只读载体角色必须拒绝")
}
