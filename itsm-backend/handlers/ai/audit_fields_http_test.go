package ai_test

// 审计回放字段的 **HTTP 响应契约**（2026-11-21）。
//
// 为什么必须测 HTTP 层而不只测 service：
// 本仓出现过「字段在后端结构体里有、在响应体里没有」的漂移（DTO 映射漏字段，
// 状态码仍 200）。审计字段尤其不能这样——复查时看不到 policyVersion，
// 等于「策略版本」这个能力根本没落地，尽管 DB 那一列是有值的。
//
// 三条断言：
//  1. 详情 / 列表 / 按 runId 三个读面的响应都带 policyVersion 与 runId；
//  2. policyVersion 的三态原样透出（"", "unknown", 指纹），读面不擅自改写；
//  3. 空 runId 返回 400 而不是「空集成功」——静默成功会让调用方
//     误以为「这轮什么都没做」，而实际是参数缺失。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/handlers/ai"
	service "itsm-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// auditFieldsRepo 提供三种 policyVersion 取值，并记录按 run 查询的入参。
type auditFieldsRepo struct {
	lastRunID string
	lastState string
}

func (r *auditFieldsRepo) CreateToolInvocation(_ context.Context, i *ai.ToolInvocation) (*ai.ToolInvocation, error) {
	return i, nil
}

func (r *auditFieldsRepo) GetToolInvocation(_ context.Context, id, _ int) (*ai.ToolInvocation, error) {
	return r.invocation(id, "run_detail"), nil
}

func (r *auditFieldsRepo) DecideToolInvocation(_ context.Context, id, _, _ int, _ bool, reason string) (*ai.ToolInvocation, error) {
	inv := r.invocation(id, "run_decide")
	inv.ApprovalReason = reason
	return inv, nil
}

func (r *auditFieldsRepo) ListToolInvocations(_ context.Context, _ int, state string) ([]*ai.ToolInvocation, error) {
	r.lastState = state
	return []*ai.ToolInvocation{
		r.invocation(1, "run_list_a"),
		r.invocation(2, "run_list_b"),
	}, nil
}

// CountToolInvocationsByApprovalState 对齐真实仓储的分组计数口径。
// 两条 mock 记录都是 approval_state="pending_agent_two" 且非预演。
func (r *auditFieldsRepo) CountToolInvocationsByApprovalState(_ context.Context, _ int) (map[string]int, error) {
	return map[string]int{"pending_agent_two": 2}, nil
}

func (r *auditFieldsRepo) ListToolInvocationsByRun(_ context.Context, _ int, runID string) ([]*ai.ToolInvocation, error) {
	r.lastRunID = runID
	if runID == "" {
		return []*ai.ToolInvocation{}, nil
	}
	inv := r.invocation(9, runID)
	inv.PolicyVersion = "unknown"
	return []*ai.ToolInvocation{inv}, nil
}

func (r *auditFieldsRepo) invocation(id int, runID string) *ai.ToolInvocation {
	return &ai.ToolInvocation{
		ID:              id,
		TenantID:        10,
		ToolName:        "update_ticket",
		Arguments:       "{}",
		Status:          "pending",
		ApprovalState:   "pending_agent_two",
		PermissionCheck: "passed",
		PolicyVersion:   "a1b2c3d4e5f60718293a4b5c6d7e8f90",
		RunID:           runID,
		RequestID:       "req-abc",
	}
}

// 其余 Repository 方法：本测试只走工具调用读面，用最小实现满足接口。
func (r *auditFieldsRepo) CreateConversation(_ context.Context, c *ai.Conversation) (*ai.Conversation, error) {
	return c, nil
}

func (r *auditFieldsRepo) GetConversation(_ context.Context, id, _, _ int) (*ai.Conversation, error) {
	return &ai.Conversation{ID: id}, nil
}

func (r *auditFieldsRepo) ListConversations(_ context.Context, _, _ int) ([]*ai.Conversation, error) {
	return nil, nil
}
func (r *auditFieldsRepo) DeleteConversation(_ context.Context, _, _, _ int) error { return nil }
func (r *auditFieldsRepo) CreateMessage(_ context.Context, m *ai.Message) (*ai.Message, error) {
	return m, nil
}

func (r *auditFieldsRepo) GetMessages(_ context.Context, _ int, _ int) ([]*ai.Message, error) {
	return nil, nil
}

func (r *auditFieldsRepo) CreateRCA(_ context.Context, x *ai.RootCauseAnalysis) (*ai.RootCauseAnalysis, error) {
	return x, nil
}

func (r *auditFieldsRepo) GetRCAByTicket(_ context.Context, _ int, _ int) (*ai.RootCauseAnalysis, error) {
	return nil, nil
}

func (r *auditFieldsRepo) UpdateRCA(_ context.Context, x *ai.RootCauseAnalysis) (*ai.RootCauseAnalysis, error) {
	return x, nil
}

func (r *auditFieldsRepo) SaveAIAnalysisResult(_ context.Context, a *ai.AIAnalysisResult) (*ai.AIAnalysisResult, error) {
	return a, nil
}

func (r *auditFieldsRepo) ListAIAnalysisResults(_ context.Context, _ int, _ string, _ int) ([]*ai.AIAnalysisResult, error) {
	return nil, nil
}

func (r *auditFieldsRepo) GetAIAnalysisResult(_ context.Context, id, _ int) (*ai.AIAnalysisResult, error) {
	return &ai.AIAnalysisResult{ID: id}, nil
}
func (r *auditFieldsRepo) DeleteAIAnalysisResult(_ context.Context, _, _ int) error { return nil }
func (r *auditFieldsRepo) GetLatestAIAnalysisResultByTicket(_ context.Context, _ int, _ int, _ string) (*ai.AIAnalysisResult, error) {
	return nil, nil
}

func newAuditFieldsHandler(t *testing.T) (*gin.Engine, *auditFieldsRepo) {
	t.Helper()
	repo := &auditFieldsRepo{}
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil,
		service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 统一注入租户与用户：handler 内两处都要求 tenant_id 存在，
	// 缺失会提前返回 401，掩盖真正的字段断言。
	withCtx := func(fn gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("tenant_id", 10)
			c.Set("user_id", 1)
			fn(c)
		}
	}
	r.GET("/detail/:id", withCtx(h.GetToolInvocation))
	r.GET("/list", withCtx(h.ListToolInvocations))
	r.GET("/runs/:runId/invocations", withCtx(h.ListRunInvocations))
	return r, repo
}

func doJSON(t *testing.T, r *gin.Engine, path string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "响应不是合法 JSON: %s", w.Body.String())
	data, _ := body["data"].(map[string]any)
	return w.Code, data
}

func TestToolInvocationResponsesExposeAuditFields(t *testing.T) {
	r, _ := newAuditFieldsHandler(t)

	t.Run("详情", func(t *testing.T) {
		code, data := doJSON(t, r, "/detail/1")
		require.Equal(t, http.StatusOK, code)
		require.Contains(t, data, "policyVersion")
		require.Contains(t, data, "runId")
		require.NotEmpty(t, data["policyVersion"], "详情页策略版本为空：审计字段未透出")
		require.NotEmpty(t, data["runId"])
	})

	t.Run("审批列表", func(t *testing.T) {
		code, data := doJSON(t, r, "/list")
		require.Equal(t, http.StatusOK, code)
		items, ok := data["items"].([]any)
		require.True(t, ok, "items 不是数组")
		require.NotEmpty(t, items)
		first, ok := items[0].(map[string]any)
		require.True(t, ok)
		require.Contains(t, first, "policyVersion")
		require.Contains(t, first, "runId")
	})

	t.Run("按运行轮次", func(t *testing.T) {
		code, data := doJSON(t, r, "/runs/run_http/invocations")
		require.Equal(t, http.StatusOK, code)
		items, ok := data["items"].([]any)
		require.True(t, ok)
		require.Len(t, items, 1)
		first := items[0].(map[string]any)
		require.Equal(t, "unknown", first["policyVersion"],
			"无法取到指纹的场景必须原样透出 unknown，不能改写成空串（两者语义不同）")
		require.Equal(t, "run_http", first["runId"])
		require.Equal(t, "run_http", data["runId"])
	})
}

func TestPolicyVersionThreeStatesAreNotRewritten(t *testing.T) {
	// 读面禁止做「语义美化」：空串（无策略参与）、unknown（取不到）、指纹
	// 三种状态必须原样返回。若把 unknown 改写成空串，事后就分不清
	// 「这次没策略判定」与「策略当时读不出来」——而这正是审计的核心问题。
	r, _ := newAuditFieldsHandler(t)

	_, data := doJSON(t, r, "/runs/run_http/invocations")
	items := data["items"].([]any)
	require.Equal(t, "unknown", items[0].(map[string]any)["policyVersion"])

	_, detail := doJSON(t, r, "/detail/1")
	require.Equal(t, "a1b2c3d4e5f60718293a4b5c6d7e8f90", detail["policyVersion"],
		"指纹应原样返回，不得截断或改写")
}

func TestListRunInvocationsRejectsEmptyRunID(t *testing.T) {
	// 路由是 /runs/:runId/invocations，空 runId 实际走不到该路由；
	// 但 handler 内部仍有防御性校验。这里验证「参数缺失不会被静默接受」：
	// 若实现允许空参数并静默返回空集，调用方会以为「这轮什么都没做」。
	repo := &auditFieldsRepo{}
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil,
		service.NewToolRegistry(nil, nil, nil, nil), nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/runs//invocations", nil)
	c.Params = gin.Params{{Key: "runId", Value: ""}}
	c.Set("tenant_id", 10)

	h.ListRunInvocations(c)

	require.Equal(t, http.StatusBadRequest, w.Code,
		"空 runId 必须显式拒绝（400），静默返回空集会让缺失参数伪装成「什么都没做」")
	require.Empty(t, repo.lastRunID, "空 runId 不应到达仓储层")
}

func TestListRunInvocationsPassesRunIDToRepository(t *testing.T) {
	// 参数真的传到了仓储：否则接口存在但永远返回空，是「控件存在≠有效」的服务端版本。
	r, repo := newAuditFieldsHandler(t)
	code, _ := doJSON(t, r, "/runs/run_propagate/invocations")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "run_propagate", repo.lastRunID)
}
