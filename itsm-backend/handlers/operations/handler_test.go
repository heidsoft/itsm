package operations

import (
	"bytes"
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
			ByType   []map[string]interface{} `json:"byType"`
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
	require.NotEmpty(t, response.Data.ByType)
}

func TestHandlerListAcceptsCommandTypeAndAggregateTypeFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	createCommand(t, client, 5, commandbus.StatusPending)
	handler := NewHandler(NewService(client))
	router := gin.New()
	router.GET("/commands", func(c *gin.Context) {
		c.Set("tenant_id", 5)
		handler.List(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/commands?commandType=workflow.start&aggregateType=incident", nil,
	))
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 1, response.Data.Total)
}

func TestHandlerBulkReplayExposesAuditAndMatchedIds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	dead := createCommandAt(t, client, 9, commandbus.StatusDeadLetter, 101)
	createCommandAt(t, client, 9, commandbus.StatusPending, 102)

	handler := NewHandler(NewService(client))
	router := gin.New()
	router.POST("/bulk-replay", func(c *gin.Context) {
		c.Set("tenant_id", 9)
		c.Set("user_id", 42)
		c.Set("request_id", "req-bulk-1")
		handler.BulkReplay(c)
	})

	body, _ := json.Marshal(map[string]interface{}{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost, "/bulk-replay", bytes.NewReader(body),
	))
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Updated    int   `json:"updated"`
			MatchedIDs []int `json:"matchedIds"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, 1, response.Data.Updated)
	require.Equal(t, []int{dead.ID}, response.Data.MatchedIDs)
}

func TestHandlerBulkReplayRejectsEmptyFilterWith404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	handler := NewHandler(NewService(client))
	router := gin.New()
	router.POST("/bulk-replay", func(c *gin.Context) {
		c.Set("tenant_id", 9)
		handler.BulkReplay(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost, "/bulk-replay", bytes.NewReader([]byte("{}")),
	))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestHandlerBulkCancelRejectsWrongStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	createCommand(t, client, 3, commandbus.StatusSucceeded)
	handler := NewHandler(NewService(client))
	router := gin.New()
	router.POST("/bulk-cancel", func(c *gin.Context) {
		c.Set("tenant_id", 3)
		c.Set("user_id", 7)
		handler.BulkCancel(c)
	})

	body, _ := json.Marshal(map[string]interface{}{"status": "succeeded"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost, "/bulk-cancel", bytes.NewReader(body),
	))
	require.Equal(t, http.StatusConflict, recorder.Code)
}