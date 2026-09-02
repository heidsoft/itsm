package automation_rule

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestGetAutomationRule_MissingRuleReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewHandler(service.NewTicketAutomationRuleService(client, logger), logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Next()
	})
	// Match the production route registered in router/router.go.
	router.GET("/api/v1/tickets/automation-rules/:id", handler.GetAutomationRule)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/automation-rules/1", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	var response common.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, common.NotFoundCode, response.Code)
	require.Equal(t, "自动化规则不存在或无权访问", response.Message)
}
