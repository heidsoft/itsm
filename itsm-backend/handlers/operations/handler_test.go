package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent/auditlog"
	"itsm-backend/internal/commandbus"
	"itsm-backend/middleware"

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
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 7})
		c.Set("tenant_id", 7)
		handler.List(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/commands?page=1&pageSize=10", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Code int `json:"code"`
		Data struct {
			Items      []map[string]interface{} `json:"items"`
			Total      int                      `json:"total"`
			Page       int                      `json:"page"`
			PageSize   int                      `json:"pageSize"`
			TotalPages int                      `json:"totalPages"`
			Summary    map[string]interface{}   `json:"summary"`
			ByType     []map[string]interface{} `json:"byType"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, 1, response.Data.Total)
	require.Equal(t, 1, response.Data.Page)
	require.Equal(t, 10, response.Data.PageSize)
	require.Equal(t, 1, response.Data.TotalPages)
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
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 5})
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
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 9})
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
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 9})
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
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 3})
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

func TestHandlerReplayRecoversInterruptedAITool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newOperationsTestClient(t)
	ctx := context.Background()
	invocation := createInterruptedAIToolInvocation(t, client, "single-recovery")
	command, err := client.OperationalCommand.Create().
		SetTenantID(invocation.TenantID).SetCommandType(commandbus.CommandExecuteAITool).
		SetAggregateType("tool_invocation").SetAggregateID(invocation.ID).
		SetIdempotencyKey("tool_invocation:1:execute").SetPayload(map[string]interface{}{"invocationId": invocation.ID}).
		SetStatus(commandbus.StatusDeadLetter).SetAttempt(1).SetMaxAttempts(1).
		SetLastError("tool invocation is not claimable").Save(ctx)
	require.NoError(t, err)

	handler := NewHandler(NewService(client))
	router := gin.New()
	router.POST("/commands/:id/replay", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: invocation.TenantID})
		c.Set("tenant_id", invocation.TenantID)
		c.Set("user_id", invocation.UserID)
		c.Set("request_id", "req-ai-recovery")
		handler.Replay(c)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/commands/1/replay", nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response struct {
		Code int `json:"code"`
		Data struct {
			Status         string `json:"status"`
			IdempotencyKey string `json:"idempotencyKey"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, commandbus.StatusPending, response.Data.Status)
	require.Equal(t, command.IdempotencyKey, response.Data.IdempotencyKey)

	storedInvocation, err := client.ToolInvocation.Get(ctx, invocation.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", storedInvocation.Status)
	require.Equal(t, "操作员确认重放中断执行", *storedInvocation.Error)
	storedCommand, err := client.OperationalCommand.Get(ctx, command.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusPending, storedCommand.Status)
	require.Equal(t, command.FencingToken+1, storedCommand.FencingToken)
	audit, err := client.AuditLog.Query().Where(
		auditlog.TenantIDEQ(invocation.TenantID), auditlog.ActionEQ("replay"), auditlog.ResourceEQ("operational_command"),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, invocation.UserID, audit.UserID)
	require.Contains(t, *audit.RequestBody, `"invocationRecovered":true`)
}

func TestReplayDoesNotRecoverAnotherTenantsInvocation(t *testing.T) {
	client := newOperationsTestClient(t)
	ctx := context.Background()
	invocation := createInterruptedAIToolInvocation(t, client, "cross-tenant-recovery")
	commandTenantID := invocation.TenantID + 1
	command, err := client.OperationalCommand.Create().
		SetTenantID(commandTenantID).SetCommandType(commandbus.CommandExecuteAITool).
		SetAggregateType("tool_invocation").SetAggregateID(invocation.ID).
		SetIdempotencyKey("tool_invocation:cross-tenant:execute").SetPayload(map[string]interface{}{"invocationId": invocation.ID}).
		SetStatus(commandbus.StatusDeadLetter).SetMaxAttempts(1).Save(ctx)
	require.NoError(t, err)

	_, err = NewService(client).Replay(ctx, commandTenantID, command.ID, Actor{UserID: invocation.UserID, Path: "/replay", Method: http.MethodPost})
	require.NoError(t, err)
	unchanged, err := client.ToolInvocation.Get(ctx, invocation.ID)
	require.NoError(t, err)
	require.Equal(t, "running", unchanged.Status)
}
