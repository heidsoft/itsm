package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/ai"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/services"
)

// 这组测试锁「执行回执与落库审批态一致」。
//
// 缺陷背景（2026-11-21 审计发现）：`POST /api/v1/agent/tools/execute` 在
// invocationID > 0 时把响应里的 approvalState 硬编码为 "pending"，
// 而 Service 落库的审批态可能是 pending_agent / pending_agent_two。
// 调用方（前端审批页、MCP 客户端、Agent 编排器）据此误判「一人批完即放行」，
// 而实际需要非发起人批准，或需要两名不同审批人依次批准
// （双人审批首票只进 awaiting_second，绝不放行）。
//
// 判据：回执的 approvalState 必须等于库里的 approval_state，
// 且摘要与下一步能区分单人 / 双人审批语义。
//
// 反向验证：把 handler 改回硬编码 "pending"，本文件必须变红，
// 且红在断言上（不是编译错误）。

type toolReceiptResponse struct {
	Code int `json:"code"`
	Data struct {
		Status        string   `json:"status"`
		Summary       string   `json:"summary"`
		InvocationID  int      `json:"invocationId"`
		ApprovalState string   `json:"approvalState"`
		NextActions   []string `json:"nextActions"`
	} `json:"data"`
}

// executeToolAs 以指定主体发起一次工具执行，返回 HTTP 状态码与解析后的信封。
func executeToolAs(t *testing.T, h *ai.Handler, tenantID, userID int, principal, body string) (int, toolReceiptResponse) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/agent/tools/execute", func(c *gin.Context) {
		// principal 走 request context（与 middleware/auth.go 同口径）：
		// ExecuteTool 的 Gate 2.5 只从 authz.PrincipalFromContext 读主体。
		c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "super_admin")
		h.ExecuteTool(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tools/execute", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp toolReceiptResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

func newToolReceiptService(t *testing.T, dbName string) (*ai.Handler, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName)
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("ToolReceipt").SetCode("TOOL_RECEIPT").SetDomain(dbName + ".test").SetStatus("active").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername(dbName + "-requester").SetEmail(dbName + "@example.com").SetName("Requester").
		SetPasswordHash("hash").SetActive(true).SetTenantID(tenant.ID).
		SaveX(ctx)

	// Gate 2.5 是 fail-closed 的：DB 里没有 ticket:write 这行，就没收任何主体。
	// 测试要覆盖「agent 被授权后的回执」，必须显式播种运行期策略
	// （与 internal/authz/registry_test.go 同一做法：DerivePolicy + EncodeDefinition）。
	seedPermission(t, client, tenant.ID, "ticket", "write", true)
	seedPermission(t, client, tenant.ID, "cmdb", "write", true)

	svc := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil,
		service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(client)
	// Gate 2.5 默认读 authz.Shared()，测试进程里那是未装配 client 的空注册表
	// （fail-closed）。这里注入指向本测试库的注册表实例——只影响本 fixture，
	// 不 ConfigureShared，避免污染同进程的其他测试。
	svc.SetPermissionCatalog(authz.NewRegistry(client))
	return ai.NewHandler(svc), tenant.ID, user.ID
}

// seedPermission 播种一条权限码，可选是否把 agent 加入 allowedPrincipals。
// 中风险动作（write）在 agent 获准时按策略要求 Mn=1（单人审批）。
func seedPermission(t *testing.T, client *ent.Client, tenantID int, resource, action string, allowAgent bool) {
	t.Helper()
	ctx := context.Background()
	policy := authz.DerivePolicy(resource, action)
	if allowAgent {
		policy.AllowedPrincipals = []string{authz.PrincipalHuman, authz.PrincipalAgent}
	}
	encoded, err := authz.EncodeDefinition(resource+" "+action, policy)
	require.NoError(t, err)
	client.Permission.Create().
		SetCode(resource + ":" + action).
		SetName(resource + " " + action).
		SetDescription(encoded).
		SetResource(resource).SetAction(action).SetTenantID(tenantID).
		SaveX(ctx)
}

func joinActions(items []string) string { return strings.Join(items, " | ") }

// 人主体发起写工具：单人审批，回执应为 pending（保留既有语义）。
func TestToolReceipt_HumanWriteToolStaysSingleApproval(t *testing.T) {
	h, tenantID, userID := newToolReceiptService(t, "receipt_human")

	status, resp := executeToolAs(t, h, tenantID, userID, authz.PrincipalHuman,
		`{"name":"create_ticket","args":{"title":"receipt"}}`)
	require.Equal(t, http.StatusOK, status)
	require.Zero(t, resp.Code)
	require.Equal(t, "pending", resp.Data.Status)
	require.Equal(t, "pending", resp.Data.ApprovalState, "人发起写工具应保留单人审批语义")
	require.NotZero(t, resp.Data.InvocationID)
}

// AI 主体发起写工具：回执必须如实报 pending_agent，不得折叠成 pending。
// 这是本批修复的核心判据。
func TestToolReceipt_AgentWriteToolReportsAgentState(t *testing.T) {
	h, tenantID, userID := newToolReceiptService(t, "receipt_agent")

	status, resp := executeToolAs(t, h, tenantID, userID, authz.PrincipalAgent,
		`{"name":"create_ticket","args":{"title":"agent receipt"}}`)
	require.Equal(t, http.StatusOK, status)
	require.Zero(t, resp.Code)
	require.NotZero(t, resp.Data.InvocationID, "AI 发起写工具应创建审批 invocation")
	require.Equal(t, "pending_agent", resp.Data.ApprovalState,
		"AI 发起写工具的回执必须是 pending_agent（不得硬编码 pending）")
	require.Contains(t, resp.Data.Summary, "AI 发起", "摘要应能区分 AI 发起与人发起")
	require.Contains(t, joinActions(resp.Data.NextActions), "非发起人",
		"下一步应提示「非发起人审批」，避免调用方误以为可自批")
}

// 回执与落库一致性：回读的 approvalState 必须等于 DB 里的值。
// 这条断言把「回执读库」而不是「回执凭推断」锁死。
func TestToolReceipt_MatchesPersistedApprovalState(t *testing.T) {
	h, tenantID, userID := newToolReceiptService(t, "receipt_persist")

	_, resp := executeToolAs(t, h, tenantID, userID, authz.PrincipalAgent,
		`{"name":"update_ticket","args":{"ticketId":1,"status":"resolved"}}`)
	require.NotZero(t, resp.Data.InvocationID)

	// 通过 handler 暴露的读面回读同一条 invocation，确认两处口径一致。
	status, detail := getInvocation(t, h, tenantID, resp.Data.InvocationID)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, resp.Data.ApprovalState, detail.Data.ApprovalState,
		"执行回执与详情读面的 approvalState 必须一致（同一落库值）")
}

type invocationDetail struct {
	Code int `json:"code"`
	Data struct {
		ApprovalState string `json:"approvalState"`
		RiskLevel     string `json:"riskLevel"`
		MinApprovals  int    `json:"minApprovals"`
	} `json:"data"`
}

func getInvocation(t *testing.T, h *ai.Handler, tenantID, id int) (int, invocationDetail) {
	t.Helper()
	r := gin.New()
	r.GET("/api/v1/agent/tools/:id", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		h.GetToolInvocation(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/agent/tools/%d", id), nil)
	r.ServeHTTP(w, req)

	var d invocationDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d), w.Body.String())
	return w.Code, d
}
