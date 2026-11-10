package sla_template

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

func TestTemplateCatalogEnglishLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(service.NewSLATemplateService(nil, nil))
	r := gin.New()
	r.GET("/api/v1/sla/templates", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 7})
		c.Next()
	}, h.ListTemplates)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/sla/templates", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Len(t, response.Data.Items, 6)
	require.Equal(t, "incident_p1_critical", response.Data.Items[0]["key"])
	require.Equal(t, "Incident P1 Critical", response.Data.Items[0]["name"])
	require.Equal(t, float64(15), response.Data.Items[0]["responseTime"])
	require.Equal(t, false, response.Data.Items[0]["installed"])
	for _, ch := range string(w.Body.Bytes()) {
		require.False(t, unicode.Is(unicode.Han, ch), "built-in catalog must use English labels")
	}
}
