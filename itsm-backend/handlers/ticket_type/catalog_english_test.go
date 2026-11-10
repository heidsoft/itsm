package ticket_type

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/service"
)

func TestPresetCatalogEnglishLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(service.NewTicketTypeService(nil, nil), nil)
	r := gin.New()
	r.GET("/api/v1/ticket-type-presets", h.ListPresets)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ticket-type-presets", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Len(t, response.Data, 4)
	require.Equal(t, "general-incident", response.Data[0]["id"])
	require.Equal(t, "General Incident", response.Data[0]["name"])
	for _, ch := range string(w.Body.Bytes()) {
		require.False(t, unicode.Is(unicode.Han, ch), "built-in catalog must use English labels")
	}
}
