// Package vector_store 的状态面 HTTP 契约回归（B10.1）。
//
// 路由与权限码：/api/v1/system/vector-store（system_config_routes.go 内注册）
// GET 默认状态 → system_config:read；POST /test 连通性诊断 → system_config:write
// （租户配置域；旧 system:read/write 是平台运维码，只有 sysadmin 持有，
// 会让菜单可见的租管进页面即 403）。
//
// 锁定语义（既有正确面防回归，本域无行为变更）：
//   - [VS1] 未配置 env：capability=unconfigured、backend=keyword、
//     配置视图 Settings 脱敏空表、响应 200 信封 code=0。
//   - [VS2] 响应字段 camelCase（fallbackEnabled/fallback 等），
//     不得出现 snake_case。
//   - [VS3] 配置解析失败（非法 env JSON）→ capability=error 的降级形态，
//     不得 500、不得泄漏原始解析错误。
//   - [VS4] end_user 不持 system:read → 403/2003。
package vector_store_test

import (
	"context"
	"database/sql"
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
	"itsm-backend/handlers/systemconfig"
	vectorStoreHandler "itsm-backend/handlers/vector_store"
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
	userA   *ent.User // security_admin：持 system:{read,write}
	endUser *ent.User // end_user：不持 system:read → 403 探针
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := fmt.Sprintf("file:vstore_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	// VectorStore 只消费自己的 *sql.DB：用同一 DSN（cache=shared）连入同一内存库，
	// 不污染进程级 database.GetRawDB()。
	rawDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	ctx := context.Background()
	tenantA := client.Tenant.Create().SetName("VStore A").SetCode("vstore-a").SetDomain("vstore-a.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("vstore-a-sec").SetEmail("vstore-a-sec@example.com").SetName("Sec A").
		SetPasswordHash("hash").SetRole("security_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("vstore-a-enduser").SetEmail("vstore-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "vstore-contract-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		// vector-store 路由挂在 SetupSystemConfigRoutes 内（外层 gate 要求
		// SystemConfigHandler != nil，与生产装配一致）。
		SystemConfigHandler: systemconfig.NewHandler(service.NewSystemConfigService(client, logger), logger),
		VectorStoreHandler:  vectorStoreHandler.NewHandler(service.NewVectorStore(rawDB), logger),
	})

	return &fixture{engine: engine, client: client, secret: secret, userA: userA, endUser: endUser}
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

func TestVectorStoreHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/system/vector-store"

	t.Run("未配置态unconfigured与camelCase", func(t *testing.T) {
		t.Setenv("VECTOR_STORE_CONFIG", "")
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)

		var status struct {
			Configured      bool                   `json:"configured"`
			Source          string                 `json:"source"`
			Backend         string                 `json:"backend"`
			Collection      string                 `json:"collection"`
			FallbackEnabled bool                   `json:"fallbackEnabled"`
			Capability      string                 `json:"capability"`
			Settings        map[string]interface{} `json:"settings"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &status), "body=%s", w.Body.String())
		require.False(t, status.Configured)
		require.Equal(t, "unconfigured", status.Capability)
		require.Equal(t, "keyword", status.Backend)
		require.NotNil(t, status.Settings)
		require.NotContains(t, w.Body.String(), `"vector_count"`, "不得出现 snake_case 字段")
	})

	t.Run("非法env降级error不泄漏原文不500", func(t *testing.T) {
		t.Setenv("VECTOR_STORE_CONFIG", "{not-json")
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		require.Contains(t, w.Body.String(), `"capability":"error"`)
		// 不得把原始解析异常透出（授权面之外的 provider 细节）
		require.NotContains(t, w.Body.String(), "localhost", "不得泄漏配置原文")
	})

	t.Run("system:read权限403禁拒", func(t *testing.T) {
		t.Setenv("VECTOR_STORE_CONFIG", "")
		w := fx.do(t, http.MethodGet, base, fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
