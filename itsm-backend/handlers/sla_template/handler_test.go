package sla_template

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestListTemplatesUsesStandardListResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(service.NewSLATemplateService(nil, nil))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sla/templates", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, common.SuccessCode, response.Code)
	require.Equal(t, "success", response.Message)
	require.NotEmpty(t, response.Data.Items)
	require.Equal(t, len(response.Data.Items), response.Data.Total)
	require.NotContains(t, recorder.Body.String(), `"templates":`)
}
