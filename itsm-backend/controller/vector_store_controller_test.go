package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newVectorTestEngine(vc *VectorStoreController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/system/vector-store", vc.GetStatus)
	r.POST("/api/v1/system/vector-store/test", vc.TestConnection)
	return r
}

func TestMaskSecret(t *testing.T) {
	cases := []struct{ in, want string }{
		{"postgres://itsm:s3cret@localhost:5432/itsm?sslmode=disable", "postgres://itsm:***@localhost:5432/itsm?sslmode=disable"},
		{"user:pass@tcp(127.0.0.1:3306)/db", "user:***@tcp(127.0.0.1:3306)/db"},
		{"no-secret-here", "no-secret-here"},
		{"", ""},
	}
	for _, tc := range cases {
		got := maskSecret(tc.in)
		assert.NotContains(t, got, "s3cret")
		assert.NotContains(t, got, "pass")
		if tc.want != "" {
			assert.Equal(t, tc.want, got)
		}
	}
}

func TestVectorStoreStatus_Unconfigured(t *testing.T) {
	t.Setenv("VECTOR_STORE_CONFIG", "")
	vc := NewVectorStoreController(nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	newVectorTestEngine(vc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/system/vector-store", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int                       `json:"code"`
		Data VectorStoreStatusResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	// 能力语义：未配置必须报告 unconfigured，禁止伪装成空成功
	assert.Equal(t, "unconfigured", body.Data.Capability)
	assert.False(t, body.Data.Configured)
	assert.Equal(t, "keyword", body.Data.Backend)
}

func TestVectorStoreStatus_KeywordReady(t *testing.T) {
	t.Setenv("VECTOR_STORE_CONFIG", "backend: keyword\ncollection: test_chunks\nfallback: true\n")
	vc := NewVectorStoreController(nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	newVectorTestEngine(vc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/system/vector-store", nil))

	var body struct {
		Code int                       `json:"code"`
		Data VectorStoreStatusResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "ready", body.Data.Capability)
	assert.Equal(t, "test_chunks", body.Data.Collection)
	assert.True(t, body.Data.FallbackEnabled)
}

func TestVectorStoreStatus_InvalidConfigIsObservableError(t *testing.T) {
	t.Setenv("VECTOR_STORE_CONFIG", "backend: [unclosed")
	vc := NewVectorStoreController(nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	newVectorTestEngine(vc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/system/vector-store", nil))

	var body struct {
		Code int                       `json:"code"`
		Data VectorStoreStatusResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	// 配置损坏必须可观测，且不得回显原始 YAML 错误
	assert.Equal(t, "error", body.Data.Capability)
	assert.NotContains(t, body.Data.Message, "unclosed")
}

func TestVectorStoreStatus_DegradedWithoutReachablePrimary(t *testing.T) {
	// pgvector 指向无法连通的 DSN：启用回退时报 degraded
	t.Setenv("VECTOR_STORE_CONFIG", "backend: pgvector\ncollection: k\nfallback: true\nconfig:\n  dsn: postgres://itsm:secret@127.0.0.1:1/none?sslmode=disable\n")
	vc := NewVectorStoreController(nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	newVectorTestEngine(vc).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/system/vector-store", nil))

	var body struct {
		Code int                       `json:"code"`
		Data VectorStoreStatusResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "degraded", body.Data.Capability)
	assert.NotContains(t, w.Body.String(), "secret", "响应不得泄漏 DSN 密码")
}

func TestVectorStoreTestEndpoint(t *testing.T) {
	t.Setenv("VECTOR_STORE_CONFIG", "backend: keyword\nfallback: true\n")
	vc := NewVectorStoreController(nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/vector-store/test", nil)
	newVectorTestEngine(vc).ServeHTTP(w, req)

	var body struct {
		Code int                     `json:"code"`
		Data VectorStoreTestResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.True(t, body.Data.OK)
	assert.Equal(t, "keyword", body.Data.Backend)
}

func TestMaskSettings(t *testing.T) {
	out := maskSettings(map[string]interface{}{
		"dsn":       "postgres://u:p@h:5432/db",
		"dimension": 1536,
		"api_key":   "sk-abcdef",
	})
	assert.NotContains(t, out["dsn"].(string), ":p@")
	assert.NotContains(t, out["api_key"].(string), "abcdef")
	assert.Equal(t, 1536, out["dimension"])
}
