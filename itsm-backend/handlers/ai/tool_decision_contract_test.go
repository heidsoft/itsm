package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/handlers/ai"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"
)

func newToolDecisionFixture(t *testing.T) (*ent.Client, int, int, *ai.Service) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:tool_decision_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("ToolDecision").SetCode("TOOL_DECIDE").SetDomain("tool-decide.test").SetStatus("active").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername("approver").SetEmail("approver@example.com").SetName("Approver").
		SetPasswordHash("hash").SetActive(true).SetTenantID(tenant.ID).
		SaveX(ctx)

	svc := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil, service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	return client, tenant.ID, user.ID, svc
}

func createPendingInvocation(t *testing.T, client *ent.Client, tenantID, userID int) *ent.ToolInvocation {
	t.Helper()
	inv, err := client.ToolInvocation.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetToolName("update_ticket").
		SetArguments(`{"ticketId":1}`).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending").
		Save(context.Background())
	require.NoError(t, err)
	return inv
}

// 审批通过必须与待执行命令同事务落库：只改审批状态会让调用永远停在 pending。
func TestApproveToolEnqueuesDurableExecution(t *testing.T) {
	client, tenantID, userID, svc := newToolDecisionFixture(t)
	ctx := context.Background()
	inv := createPendingInvocation(t, client, tenantID, userID)

	state, err := svc.ApproveTool(ctx, inv.ID, tenantID, userID, true, "允许执行")
	require.NoError(t, err)
	require.Equal(t, "approved", state)

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "approved", stored.ApprovalState)
	require.Equal(t, userID, stored.ApprovedBy)
	require.NotNil(t, stored.ApprovedAt)

	commands, err := client.OperationalCommand.Query().
		Where(
			operationalcommand.TenantIDEQ(tenantID),
			operationalcommand.CommandTypeEQ(commandbus.CommandExecuteAITool),
			operationalcommand.AggregateIDEQ(inv.ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, "tool_invocation", commands[0].AggregateType)
	require.Equal(t, "pending", commands[0].Status)
	require.Equal(t, 1, commands[0].MaxAttempts, "不可幂等的工具副作用不能自动重试")
	require.EqualValues(t, inv.ID, commands[0].Payload["invocationId"])
}

// 重复审批必须返回业务冲突，且不能产生第二条执行命令。
func TestApproveToolRejectsDuplicateDecision(t *testing.T) {
	client, tenantID, userID, svc := newToolDecisionFixture(t)
	ctx := context.Background()
	inv := createPendingInvocation(t, client, tenantID, userID)

	_, err := svc.ApproveTool(ctx, inv.ID, tenantID, userID, true, "第一次")
	require.NoError(t, err)

	_, err = svc.ApproveTool(ctx, inv.ID, tenantID, userID, true, "第二次")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr))
	require.Equal(t, common.ConflictCode, bizErr.Code)

	count := client.OperationalCommand.Query().
		Where(
			operationalcommand.TenantIDEQ(tenantID),
			operationalcommand.CommandTypeEQ(commandbus.CommandExecuteAITool),
		).
		CountX(ctx)
	require.Equal(t, 1, count, "重复审批不得重复入箱")
}

func TestRejectToolWritesNoCommand(t *testing.T) {
	client, tenantID, userID, svc := newToolDecisionFixture(t)
	ctx := context.Background()
	inv := createPendingInvocation(t, client, tenantID, userID)

	state, err := svc.ApproveTool(ctx, inv.ID, tenantID, userID, false, "风险过高")
	require.NoError(t, err)
	require.Equal(t, "rejected", state)

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "rejected", stored.ApprovalState)
	require.Equal(t, "风险过高", stored.ApprovalReason)
	require.True(t, stored.ApprovedAt.IsZero())
	require.Zero(t, stored.ApprovedBy, "approved_by 只在批准时写入")

	count := client.OperationalCommand.Query().CountX(ctx)
	require.Zero(t, count, "拒绝审批不能创建执行命令")
}

// 审批接口的状态码必须区分「不存在(404)」与「已审批(409)」，
// 前端才能给出正确提示而不是一律显示操作失败。
func TestApproveToolHTTPContract(t *testing.T) {
	client, tenantID, userID, svc := newToolDecisionFixture(t)
	handler := ai.NewHandler(svc)
	router := gin.New()
	router.POST("/api/v1/agent/tools/:id/approve", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "super_admin")
		handler.ApproveTool(c)
	})
	post := func(id int, body string) (*httptest.ResponseRecorder, int) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agent/tools/%d/approve", id), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var response struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		return rec, response.Code
	}

	inv := createPendingInvocation(t, client, tenantID, userID)
	rec, code := post(inv.ID, `{"approve":true,"reason":"ok"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Zero(t, code)

	rec, code = post(inv.ID, `{"approve":true,"reason":"again"}`)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, common.ConflictCode, code)

	rec, code = post(999999, `{"approve":true}`)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, common.NotFoundCode, code)
}

// 别租户的调用必须按资源不存在处理，且不产生命令。
func TestApproveToolIsTenantScoped(t *testing.T) {
	client, tenantID, userID, svc := newToolDecisionFixture(t)
	ctx := context.Background()
	otherTenant := client.Tenant.Create().
		SetName("Other").SetCode("OTHER_DECIDE").SetDomain("other-decide.test").SetStatus("active").
		SaveX(ctx)
	inv := createPendingInvocation(t, client, otherTenant.ID, userID)

	_, err := svc.ApproveTool(ctx, inv.ID, tenantID, userID, true, "")
	require.Error(t, err)
	require.True(t, ent.IsNotFound(err), "跨租户审批必须表现为资源不存在")

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.ApprovalState)
	require.Zero(t, client.OperationalCommand.Query().CountX(ctx))
}
