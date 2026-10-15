package msp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/domain/role"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	enttenant "itsm-backend/ent/tenant"
	entuser "itsm-backend/ent/user"
	"itsm-backend/internal/authz"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const mspStatusTestSecret = "test-jwt-secret-for-msp-status"

// newMSPIdentityRouter 复刻生产身份链：真实 AuthMiddleware 签发并解析 token，
// 再由真实 MSPMiddleware 从数据库推导 MSPContext。断言的是「认证身份到底能推出
// 什么 MSP 权限」，而不是测试自己伪造的 gin 键。
func newMSPIdentityRouter(t *testing.T, client *ent.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(nil, service.NewTicketServiceForTest(client, logger), logger)

	r := gin.New()
	r.Use(gin.Recovery())
	// 与 router/msp_routes.go 相同的挂载方式：路径 + AuthMiddleware + MSPMiddleware。
	group := r.Group("/api/v1/msp")
	group.Use(middleware.AuthMiddleware(mspStatusTestSecret))
	group.Use(middleware.MSPMiddleware(client))
	group.GET("/status", h.GetMSPStatus)
	return r
}

func seedMSPIdentity(t *testing.T, client *ent.Client, username, tenantType, mspRole, platformRole string) int {
	t.Helper()
	ctx := t.Context()

	tenantRow, err := client.Tenant.Create().
		SetName("tenant-" + username).
		SetCode("code-" + username).
		SetType(enttenant.Type(tenantType)).
		Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole(entuser.Role(platformRole)).
		SetTenantID(tenantRow.ID).
		Save(ctx)
	require.NoError(t, err)

	if mspRole != "" {
		u, err = client.User.UpdateOne(u).SetMspRole(entuser.MspRole(mspRole)).Save(ctx)
		require.NoError(t, err)
	}
	return u.ID
}

func getToken(t *testing.T, userID int, platformRole string) string {
	t.Helper()
	token, err := middleware.GenerateAccessToken(userID, "msp-status-caller", platformRole, 1, mspStatusTestSecret, time.Minute)
	require.NoError(t, err)
	return token
}

func fetchMSPStatus(t *testing.T, r *gin.Engine, userID int, platformRole string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/msp/status", nil)
	req.Header.Set("Authorization", "Bearer "+getToken(t, userID, platformRole))
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body
}

func statusData(t *testing.T, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是状态对象，实际响应 %v", body)
	return data
}

// TestGetMSPStatus_IdentityDerivedFromAuthenticatedUser 锁定 MSP 身份口径：
// MSP 能力只来自「服务商租户 + users.msp_role」，平台角色名不参与判定。
func TestGetMSPStatus_IdentityDerivedFromAuthenticatedUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_status_identity?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	providerAdminID := seedMSPIdentity(t, client, "provider_admin", "msp_provider", "provider_admin", "agent")
	providerAgentID := seedMSPIdentity(t, client, "provider_agent", "msp_provider", "provider_agent", "agent")
	customerUserID := seedMSPIdentity(t, client, "customer_end_user", "msp_provider", "customer_user", "end_user")
	platformAdminID := seedMSPIdentity(t, client, "plain_admin", "standard", "", "admin")
	platformSuperID := seedMSPIdentity(t, client, "plain_super", "internal", "", "super_admin")

	r := newMSPIdentityRouter(t, client)

	tests := []struct {
		name          string
		userID        int
		platformRole  string
		wantIsMsp     bool
		wantIsAdmin   bool
		wantMSPRole   string
		wantMessage   string
		forbidMessage string
	}{
		{
			name:         "服务商管理员同时获得员工与管理员展示位",
			userID:       providerAdminID,
			platformRole: "agent",
			wantIsMsp:    true,
			wantIsAdmin:  true,
			wantMSPRole:  "provider_admin",
		},
		{
			name:          "服务商客服只是员工",
			userID:        providerAgentID,
			platformRole:  "agent",
			wantIsMsp:     true,
			wantIsAdmin:   false,
			wantMSPRole:   "provider_agent",
			forbidMessage: "管理员模式：可配置MSP功能",
		},
		{
			name:         "客户侧成员不是服务商管理员",
			userID:       customerUserID,
			platformRole: "end_user",
			wantIsMsp:    true,
			wantIsAdmin:  false,
			wantMSPRole:  "customer_user",
		},
		{
			// 决策回归锁：把 gin 键名从 user_role 「修」成 role 会让平台管理员
			// 拿到一个所有 MSP 数据端点都 403 的页面，这正是 347d44511 移除的旁路。
			name:        "平台 admin 不构成 MSP 访问权",
			userID:      platformAdminID,
			wantIsMsp:   false,
			wantIsAdmin: false,
			wantMessage: "非MSP用户",
		},
		{
			name:        "平台 super_admin 不构成 MSP 访问权",
			userID:      platformSuperID,
			wantIsMsp:   false,
			wantIsAdmin: false,
			wantMessage: "非MSP用户",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpStatus, body := fetchMSPStatus(t, r, tc.userID, tc.platformRole)
			require.Equal(t, http.StatusOK, httpStatus, body["message"])
			assert.Equal(t, float64(0), body["code"], "业务成功必须返回 code=0")

			data := statusData(t, body)
			assert.Equal(t, tc.wantIsMsp, data["isMsp"], "isMsp 必须来自认证身份推导")
			assert.Equal(t, tc.wantIsAdmin, data["isAdmin"], "isAdmin 只表示 provider_admin")

			if tc.wantMSPRole != "" {
				assert.Equal(t, tc.wantMSPRole, data["role"])
			}
			if tc.wantMessage != "" {
				assert.Equal(t, tc.wantMessage, data["message"])
			}
			if tc.forbidMessage != "" {
				assert.NotEqual(t, tc.forbidMessage, data["message"])
			}

			// 契约：camelCase，且不得泄漏 snake_case 或内部标识。
			raw, err := json.Marshal(body)
			require.NoError(t, err)
			assert.NotContains(t, string(raw), `"is_msp"`)
			assert.NotContains(t, string(raw), `"is_admin"`)
			assert.NotContains(t, string(raw), `"msp_user_id"`)
		})
	}
}

// TestMSPStatusAdminFlagMatchesManagerLevelGrant 把 isAdmin 的口径钉在真实授权词表上：
// 被打标为管理员的 msp_role，映射后的 RBAC 角色必须确实持有 MSP 管理写权限；
// 未打标的角色必须没有，否则 UI 入口与 RequireMSPPermission 的放行会分叉。
func TestMSPStatusAdminFlagMatchesManagerLevelGrant(t *testing.T) {
	defaults := authz.RolePermissionDefaults()
	const managementCode = "msp_allocation:write"

	managerRBAC := middleware.GetMSPRBACRole(middleware.MSPRoleManager)
	require.Equal(t, role.MspManager, managerRBAC, "provider_admin 的映射目标已变化，需同步 isAdmin 口径")
	assert.Contains(t, defaults[managerRBAC], managementCode,
		"provider_admin 必须映射到持有 %s 的角色，否则管理入口是假承诺", managementCode)

	assert.True(t, middleware.IsMSPManagerRole(middleware.MSPRoleManager))
	for _, mspRole := range []string{"provider_agent", "customer_user", ""} {
		assert.False(t, middleware.IsMSPManagerRole(mspRole), "%q 不是服务商管理员", mspRole)
		mapped := middleware.GetMSPRBACRole(mspRole)
		assert.NotContains(t, defaults[mapped], managementCode,
			"%s 映射到 %s 却持有 %s，isAdmin 词表需要扩展", mspRole, mapped, managementCode)
	}
}
