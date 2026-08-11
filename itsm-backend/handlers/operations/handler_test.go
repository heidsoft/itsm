package operations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/internal/commandbus"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandlerListUsesCamelCaseContractAndTenantScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	createCommand(t, client, 7, commandbus.StatusPending)
	createCommand(t, client, 8, commandbus.StatusDeadLetter)
	handler := NewHandler(NewService(client))
	router := gin.New()
	router.GET("/commands", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		handler.List(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/commands?page=1&pageSize=10", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Items    []map[string]interface{} `json:"items"`
			Total    int                      `json:"total"`
			PageSize int                      `json:"pageSize"`
			Summary  map[string]interface{}   `json:"summary"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, 1, response.Data.Total)
	require.Equal(t, 10, response.Data.PageSize)
	require.Len(t, response.Data.Items, 1)
	require.Contains(t, response.Data.Items[0], "commandType")
	require.NotContains(t, response.Data.Items[0], "command_type")
	require.EqualValues(t, 1, response.Data.Summary["pending"])
}
