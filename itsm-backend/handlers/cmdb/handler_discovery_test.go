package cmdb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDiscoveryJobFailsClosedUntilProductionReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(nil)
	router.POST("/api/v1/cmdb/discovery/jobs", handler.CreateDiscoveryJob)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/cmdb/discovery/jobs",
		strings.NewReader(`{"sourceId":"source-1"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.JSONEq(t, `{
		"code": 5003,
		"message": "云资源自动发现尚未通过生产验收，当前服务不可用"
	}`, recorder.Body.String())
}
