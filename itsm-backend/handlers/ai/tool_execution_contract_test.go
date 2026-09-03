package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/ai"
	"itsm-backend/service"
)

func TestExecuteToolReportsPendingApproval(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	tenant := client.Tenant.Create().SetName("Tools").SetCode("tools").SetDomain("tools.test").SaveX(context.Background())
	user := client.User.Create().SetUsername("operator").SetEmail("operator@example.com").SetName("Operator").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(context.Background())
	svc := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil, service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(client)
	h := ai.NewHandler(svc)
	r := gin.New()
	r.POST("/api/v1/agent/tools/execute", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", user.ID)
		c.Set("role", "super_admin")
		h.ExecuteTool(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tools/execute", bytes.NewBufferString(`{"name":"create_ticket","args":{"title":"review"}}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		Code int `json:"code"`
		Data struct {
			Status        string `json:"status"`
			InvocationID  int    `json:"invocationId"`
			ApprovalState string `json:"approvalState"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, "pending", response.Data.Status)
	require.Equal(t, "pending", response.Data.ApprovalState)
	invocation := client.ToolInvocation.GetX(context.Background(), response.Data.InvocationID)
	require.Equal(t, "pending", invocation.Status)
	require.Equal(t, tenant.ID, invocation.TenantID)
	require.Equal(t, user.ID, invocation.UserID)
	require.Zero(t, client.Ticket.Query().CountX(context.Background()), "approval request must not execute the write")
}
