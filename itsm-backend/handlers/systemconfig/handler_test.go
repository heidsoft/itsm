package systemconfig

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestHandler(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	dbName := "file:sc_test_" + t.Name() + "?mode=memory&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	configService := service.NewSystemConfigService(client, logger)
	h := NewHandler(configService, logger)

	r := gin.New()
	r.Use(gin.Recovery())

	// 注入 *middleware.TenantContext（GetTenantID 的读取来源）
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h == "" {
			// default
			_ = tenantID
		}
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Next()
	})

	// 注册路由 - mirror router.go 契约 /system-configs 与 /configs（两个路径组同构）
	sc := r.Group("/api/v1/system-configs")
	{
		sc.GET("", h.ListConfigs)
		sc.GET("/:id", h.GetConfig)
		sc.GET("/key/:key", h.GetConfigByKey)
		sc.PUT("/:id", h.UpdateConfig)
		sc.PUT("/batch", h.BatchUpdateConfigs)
	}

	return r
}

// 默认配置只能由 ListConfigs 懒初始化。曾经存在 GET /system-configs/init：
// 按 system_config:read 守卫却整表写入密码策略/2FA/SMTP，而 end_user 持该读权限，
// 一次 GET 就能把 passwordMinLength=8、enable2FA=false 落进租户配置。
func TestHandler_ListConfigs_LazyInitializesDefaults(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/system-configs", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items    []map[string]interface{} `json:"items"`
			Total    int                      `json:"total"`
			Page     int                      `json:"page"`
			PageSize int                      `json:"pageSize"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	assert.Equal(t, 0, listResp.Code)
	assert.Greater(t, listResp.Data.Total, 0, "空表首次读取应懒初始化出默认配置")
	assert.Equal(t, 1, listResp.Data.Page)
	assert.Equal(t, 20, listResp.Data.PageSize)
}

// GET /system-configs/init 不再是写入口：路径只剩 /:id 兜底，不得再返回初始化成功。
// 路由本身缺席由 router 生产路由表测试锁定（含 /configs 别名）。
func TestHandler_InitRouteIsRemoved(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/system-configs/init", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "init 不得再作为可调用入口：%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "默认配置初始化成功")
}

func TestHandler_GetConfig_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/system-configs/invalid", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetConfig_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/system-configs/99999", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "service 层查不到返回错误 → NotFoundWithErr 404")
}

func TestHandler_GetConfigByKey_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/system-configs/key/nonexistent_key", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateConfig_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"value":"new_value"}`)
	req, err := http.NewRequest("PUT", "/api/v1/system-configs/99999", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "service 更新失败 → FailWithErr 500（与旧契约一致）")
}

func TestHandler_BatchUpdateConfigs_BadRequest(t *testing.T) {
	r := setupTestHandler(t)

	// 非法 JSON → 绑定失败
	body := []byte(`{invalid}`)
	req, err := http.NewRequest("PUT", "/api/v1/system-configs/batch", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MissingTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:sc_notenant_"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	configService := service.NewSystemConfigService(client, logger)
	h := NewHandler(configService, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	// 不注入 TenantContext → GetTenantID 报错 → 401
	r.GET("/api/v1/system-configs", h.ListConfigs)

	req, err := http.NewRequest("GET", "/api/v1/system-configs", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "缺租户上下文应 401（旧契约 UnauthorizedCode）")
}
