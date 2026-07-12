package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ConnectorController 不需要数据库（Manager/Registry 均为内存结构）。
// 这里覆盖“GA 就绪”视角下的只读/健康端点，验证路由接线、
// DTO 脱敏与租户作用域不会 panic —— 正是部署就绪检查关注的点。
func setupConnectorController(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()

	reg := connector.NewRegistry()
	mgr := connector.NewManager(reg, logger)
	mkt := marketplace.New()
	ctrl := NewConnectorController(mgr, reg, mkt, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(1, 1))
	r.GET("/api/v1/connectors", ctrl.ListMarket)
	r.GET("/api/v1/connectors/configs", ctrl.ListConfigs)
	r.GET("/api/v1/connectors/health", ctrl.Health)
	r.GET("/api/v1/connectors/lifecycle", ctrl.Lifecycle)
	return r
}

func TestConnectorController_ListMarket(t *testing.T) {
	r := setupConnectorController(t)
	resp := doReq(t, r, "GET", "/api/v1/connectors", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "items")
	assert.Contains(t, data, "total")
}

func TestConnectorController_ListConfigs(t *testing.T) {
	r := setupConnectorController(t)
	resp := doReq(t, r, "GET", "/api/v1/connectors/configs", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Health(t *testing.T) {
	r := setupConnectorController(t)
	resp := doReq(t, r, "GET", "/api/v1/connectors/health", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Lifecycle(t *testing.T) {
	r := setupConnectorController(t)
	resp := doReq(t, r, "GET", "/api/v1/connectors/lifecycle", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "items")
	// 空管理器下也应返回生命周期视图（不依赖外部连接器）
	require.NotNil(t, data["total"])
}

// --- 配置校验分支：Provision / Revoke / Test ---

// provisionFake 是仅用于测试的轻量连接器，注册到 registry 后即可走通
// Provision 的成功 / 失败 / 禁用 分支。
type provisionFake struct{ cfg connector.Config }

func (f *provisionFake) Manifest() connector.Manifest {
	return connector.Manifest{Name: "fakeconn", Title: "Fake Conn", Provider: "fakep", Type: connector.TypeCustom}
}
func (f *provisionFake) Init(_ context.Context, cfg connector.Config) error { f.cfg = cfg; return nil }
func (f *provisionFake) Send(_ context.Context, _ *connector.Message) error { return nil }
func (f *provisionFake) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (f *provisionFake) Close() error { return nil }

// setupConnectorControllerWithProvision 在空 registry 基础上注册一个 fakeconn，
// 以便测试 Provision / Revoke / Test 的配置校验分支。
func setupConnectorControllerWithProvision(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	reg := connector.NewRegistry()
	reg.Register(func() connector.Connector { return &provisionFake{} })
	mgr := connector.NewManager(reg, logger)
	mkt := marketplace.New()
	ctrl := NewConnectorController(mgr, reg, mkt, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(1, 1))
	r.POST("/api/v1/connectors/configs", ctrl.Provision)
	r.DELETE("/api/v1/connectors/configs/:name", ctrl.Revoke)
	r.POST("/api/v1/connectors/:name/test", ctrl.Test)
	return r
}

// doReqRaw 发送原始 body，用于触发 JSON 解析失败分支
func doReqRaw(t *testing.T, r *gin.Engine, method, path, raw string) *common.Response {
	t.Helper()
	req, err := http.NewRequest(method, path, strings.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

func TestConnectorController_Provision_InvalidJSON(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// 畸形 JSON：解析失败 → ParamErrorCode
	resp := doReqRaw(t, r, "POST", "/api/v1/connectors/configs", `{"name":`)
	assert.EqualValues(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Provision_MissingName(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// 缺少 binding:"required" 的 name → 校验失败 → ParamErrorCode
	resp := doReq(t, r, "POST", "/api/v1/connectors/configs", map[string]interface{}{"enabled": true}, false)
	assert.EqualValues(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Provision_NotRegistered(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// name 未在 registry 注册 → manager 返回 error → InternalErrorCode
	resp := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{Name: "ghost", Enabled: true}, false)
	assert.EqualValues(t, common.InternalErrorCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Provision_Success(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	resp := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{
			Name:        "fakeconn",
			Provider:    "fakep",
			Enabled:     true,
			Credentials: map[string]string{"token": "secret"},
			Settings:    map[string]interface{}{},
		}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "fakeconn", data["name"])
	// 凭据脱敏：返回的是键名而非明文
	creds, ok := data["credentials"].(map[string]interface{})
	require.True(t, ok, "credentials should be a masked map")
	_, hasToken := creds["token"]
	assert.True(t, hasToken, "masked credentials should preserve key name")
}

func TestConnectorController_Provision_DisabledRevokes(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// Enabled=false：Provision 走 Revoke 分支，返回成功（不会新建实例）
	resp := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{Name: "fakeconn", Enabled: false}, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Revoke(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// 先成功 provision
	prov := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{Name: "fakeconn", Enabled: true}, false)
	require.Equal(t, common.SuccessCode, prov.Code, "body=%s", mustString(prov))
	// 再 revoke
	resp := doReq(t, r, "DELETE", "/api/v1/connectors/configs/fakeconn", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "fakeconn", data["name"])
	assert.Equal(t, true, data["revoked"])
}

func TestConnectorController_Test_MissingDebugChannel(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// provision 但不配置 debug_channel
	prov := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{Name: "fakeconn", Enabled: true, Settings: map[string]interface{}{}}, false)
	require.Equal(t, common.SuccessCode, prov.Code)
	// 触发 test：缺少 debug_channel → ParamErrorCode
	resp := doReq(t, r, "POST", "/api/v1/connectors/fakeconn/test", nil, false)
	assert.EqualValues(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
}

func TestConnectorController_Test_Success(t *testing.T) {
	r := setupConnectorControllerWithProvision(t)
	// provision 时配置 debug_channel
	prov := doReq(t, r, "POST", "/api/v1/connectors/configs",
		dto.ProvisionConnectorRequest{
			Name:     "fakeconn",
			Enabled:  true,
			Settings: map[string]interface{}{"debug_channel": "ch-test"},
		}, false)
	require.Equal(t, common.SuccessCode, prov.Code, "body=%s", mustString(prov))
	// 触发 test：debug_channel 已配置 → 走 Send 成功分支
	resp := doReq(t, r, "POST", "/api/v1/connectors/fakeconn/test", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "ch-test", data["channel"])
}
