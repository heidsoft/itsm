package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	cmdbHandler "itsm-backend/handlers/cmdb"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestCMDBGovernanceReportRouteContract 打在真实 SetupRoutes 上，锁治理看板
// 路由的三态契约：无 CMDB 读权的角色 403、跨租户读不到别人的治理数据、
// 归属租户 super_admin 200 且响应键为 camelCase 治理结构。
func TestCMDBGovernanceReportRouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_governance_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:   cmdbListSecret,
		Logger:      logger,
		Client:      client,
		CMDBHandler: cmdbHandler.NewHandler(cmdbHandler.NewService(cmdbHandler.NewEntRepository(client), nil, logger)),
	})

	tenantA := seedGovernanceTenant(ctx, t, client, "gov-a")
	tenantB := seedGovernanceTenant(ctx, t, client, "gov-b")

	call := func(t *testing.T, userID int, role string, tenantID int) (*httptest.ResponseRecorder, governanceEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, "user", role, tenantID, cmdbListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/governance/report", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env governanceEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w, env
	}

	t.Run("归属租户 super_admin 200 且响应为 camelCase 治理结构", func(t *testing.T) {
		w, env := call(t, tenantA.adminID, "super_admin", tenantA.tenantID)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, env.Code, "body=%s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"completenessPct"`, "质量指标必须是 camelCase: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), `"retire_confirm"`, "差异分桶键必须是受控词表: %s", w.Body.String())
		// 空租户也是全 0 快照，不是 null data
		assert.NotEqual(t, "null", string(env.Data), "data=%s", string(env.Data))
	})

	t.Run("无 cmdb:read 的角色 403", func(t *testing.T) {
		// end_user 的默认集已无 cmdb:read（2026-10-26 自助门户收敛退役）
		w, env := call(t, tenantA.endUserID, "end_user", tenantA.tenantID)
		assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		assert.Equal(t, 2003, env.Code)
	})

	t.Run("跨租户读取只看到自己的空快照", func(t *testing.T) {
		// A 管理员声明自己是 B 租户：B 的治理快照与 A 的互不可见，
		// 且各租户的 data 都是各自的计数（此处双侧都空，键结构必须一致成立）。
		w, env := call(t, tenantA.adminID, "super_admin", tenantB.tenantID)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, env.Code)
		assert.Contains(t, w.Body.String(), `"cis":0`)
	})
}

// governanceEnvelope 复用全站 {code,message,data} 契约。
type governanceEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type governanceTenantFixture struct {
	tenantID  int
	adminID   int
	endUserID int
}

func seedGovernanceTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) governanceTenantFixture {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Governance " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName(code + "-admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	endUser, err := client.User.Create().
		SetUsername(code + "-enduser").
		SetEmail(code + "-enduser@example.com").
		SetName(code + "-enduser").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return governanceTenantFixture{tenantID: tenant.ID, adminID: admin.ID, endUserID: endUser.ID}
}
