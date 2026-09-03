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
		sc.GET("/init", h.InitDefaultConfigs)
		sc.GET("/:id", h.GetConfig)
		sc.GET("/key/:key", h.GetConfigByKey)
		sc.PUT("/:id", h.UpdateConfig)
		sc.PUT("/batch", h.BatchUpdateConfigs)
	}

	return r
}

func TestHandler_InitDefaultConfigs_ThenList(t *testing.T) {
	r := setupTestHandler(t)

	// 初始化默认配置
	req, err := http.NewRequest("GET", "/api/v1/system-configs/init", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var initResp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &initResp))
	assert.Equal(t, 0, initResp.Code)

	// 列表应有配置
	req2, err := http.NewRequest("GET", "/api/v1/system-configs", nil)
	require.NoError(t, err)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "body=%s", w2.Body.String())

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items    []map[string]interface{} `json:"items"`
			Total    int                      `json:"total"`
			Page     int                      `json:"page"`
			PageSize int                      `json:"pageSize"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	assert.Equal(t, 0, listResp.Code)
	assert.Greater(t, listResp.Data.Total, 0, "初始化后应至少有一条默认配置")
	assert.Equal(t, 1, listResp.Data.Page)
	assert.Equal(t, 20, listResp.Data.PageSize)
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
