package ai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
	"itsm-backend/ent/toolinvocation"
	"itsm-backend/handlers/ai"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/invocationid"
)

// 这组测试锁「复核既有调用」的四条判据：
//
//	① 按**原发起主体**判定，不按审批人自己 —— 否则管理员复核永远 allowed=true，
//	   复核沦为仪式；这条是整组存在的理由。
//	② 策略指纹比对由服务端算，四态不可合并（identical / changed / unknown /
//	   not_applicable）—— 把 unknown 归入 identical 会让「策略信息缺失」
//	   伪装成「策略未变」，是审计场景最危险的假阴性。
//	③ 零副作用：不执行工具、不改被复核记录的任何字段、不建待审批行。
//	④ 跨租户不可见：只按 id 查会让任何人复核别人的调用。

type reviewResponse struct {
	Code int                    `json:"code"`
	Data dto.ToolDryRunResponse `json:"data"`
}

// reviewAs 走真实 handler（含参数解析与错误映射），按指定审批人身份发起复核。
func reviewAs(t *testing.T, h *ai.Handler, tenantID, userID int, invocationID int) (int, reviewResponse) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/agent/tools/invocations/:id/review", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "super_admin")
		h.ReviewToolInvocation(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/agent/tools/invocations/%d/review", invocationID), nil)
	r.ServeHTTP(w, req)
	var resp reviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

// pendingInvocationFixture 造一条 AI 发起的待审批记录（pending_agent）。
// 直插而不走 ExecuteTool：这里要验证的是**复核读面**，被复核记录如何产生无关。
func pendingInvocationForReview(t *testing.T, f *dryRunFixture, toolName, policyVersion string) int {
	t.Helper()
	// 写服务端主体信封：AI 发起的调用必须显式声明 agent。
	// 封装后业务参数仍是 {"ticketId":7}，与旧夹具等价，但复核读面
	// 不再依赖 approval_state / policy_version 间接推断主体。
	argsWithIdentity, encErr := invocationid.Encode(map[string]interface{}{"ticketId": float64(7)}, authz.PrincipalAgent)
	require.NoError(t, encErr)
	inv, err := f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).
		SetUserID(f.userID).
		SetToolName(toolName).
		SetArguments(argsWithIdentity).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending_agent").
		SetPermissionCheck("passed").
		SetPermissionReason("principal=agent risk=medium minApprovals=1").
		SetRoleSnapshot("super_admin").
		SetPolicyVersion(policyVersion).
		SetDryRun(false).
		Save(context.Background())
	require.NoError(t, err)
	return inv.ID
}

// 判据 ①：复核一条 AI 发起的调用，必须按 agent 判定，而不是按审批人（human）。
//
// 反向验证的构造：把 ai.Service.ReviewInvocation 里的 originPrincipal 固定成
// authz.PrincipalHuman，本用例应红在 EvaluatedPrincipal 与 ApprovalState 上
// ——human 路径不查 Gate 2.5、审批态会是 pending 而不是 pending_agent。
func TestReview_UsesOriginPrincipalNotApprover(t *testing.T) {
	f := newDryRunFixture(t, "review_principal")
	invID := pendingInvocationForReview(t, f, "probe_write_ticket", "")

	status, resp := reviewAs(t, f.handler, f.tenantID, f.userID, invID)
	require.Equal(t, http.StatusOK, status)
	require.Zero(t, resp.Code, resp.Data.Reason)
	require.True(t, resp.Data.DryRun, "复核结果同样是预演结论，dryRun 恒为 true")
	require.Equal(t, invID, resp.Data.InvocationID, "必须回显被复核的记录 ID")
	require.Equal(t, authz.PrincipalAgent, resp.Data.EvaluatedPrincipal,
		"AI 发起的调用必须按 agent 主体复核；按审批人 human 判定会得出与执行链无关的结论")
	require.True(t, resp.Data.Allowed, "已播种 allowedPrincipals 含 agent，应放行")
	require.Equal(t, "pending_agent", resp.Data.ApprovalState,
		"复核给出的审批态必须与被复核记录同主体语义，不得折叠成 pending")
	require.Equal(t, 0, f.catalog.execCount, "判据 ③：复核绝不执行工具")
}

// 判据 ①（反向）：人发起的调用必须按 human 复核，不能被误判成 agent。
// pending 是人发起才可能出现的态（agent 一定是 pending_agent*）。
func TestReview_HumanOriginatedInvocationEvaluatedAsHuman(t *testing.T) {
	f := newDryRunFixture(t, "review_human_origin")
	inv, err := f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).
		SetUserID(f.userID).
		SetToolName("probe_write_ticket").
		SetArguments(`{"ticketId":9}`).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending").
		SetPermissionCheck("passed").
		SetRoleSnapshot("super_admin").
		SetDryRun(false).
		Save(context.Background())
	require.NoError(t, err)

	status, resp := reviewAs(t, f.handler, f.tenantID, f.userID, inv.ID)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, authz.PrincipalHuman, resp.Data.EvaluatedPrincipal,
		"approval_state=pending 是人发起的唯一形态，必须按 human 复核")
	require.Equal(t, "pending", resp.Data.ApprovalState)
}

// 判据 ②：四态策略比对。这是复核相对「只看落库 policyVersion」的全部增量价值。
func TestReview_PolicyComparisonFourStates(t *testing.T) {
	// 取当前策略的真实指纹：用它构造「未变」与「已变」两种输入，
	// 而不是硬编码一个假指纹——假指纹会让 identical 分支永远无法真实命中。
	f := newDryRunFixture(t, "review_policy_cmp")
	_, first := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalAgent, "super_admin", "probe_write_ticket")
	require.NotEmpty(t, first.Data.PolicyVersion, "agent 路径必须给出实际策略指纹")
	require.NotEqual(t, authz.PolicyVersionUnknown, first.Data.PolicyVersion)

	cases := []struct {
		name     string
		baseline string
		want     string
	}{
		{
			name:     "同一指纹 → identical",
			baseline: first.Data.PolicyVersion,
			want:     dto.PolicyComparisonIdentical,
		},
		{
			name:     "旧指纹 → changed",
			baseline: "0123456789abcdef0123456789abcdef",
			want:     dto.PolicyComparisonChanged,
		},
		{
			name:     "基线 unknown → unknown（不得归入 identical）",
			baseline: authz.PolicyVersionUnknown,
			want:     dto.PolicyComparisonUnknown,
		},
		{
			name:     "基线空串而当前有指纹 → changed（无策略→有策略，判据实质已变）",
			baseline: "",
			want:     dto.PolicyComparisonChanged,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 每条用例独立 fixture：共享库会让 dryRunRows 的计数互相污染。
			ff := newDryRunFixture(t, "review_cmp_"+tc.want)
			invID := pendingInvocationForReview(t, ff, "probe_write_ticket", tc.baseline)
			status, resp := reviewAs(t, ff.handler, ff.tenantID, ff.userID, invID)
			require.Equal(t, http.StatusOK, status)
			require.Equal(t, tc.want, resp.Data.PolicyComparison, resp.Data.Reason)
			require.Equal(t, tc.baseline, resp.Data.BaselinePolicyVersion,
				"必须回显被复核记录当时的指纹，审批人才能自己核实比对结论")
		})
	}
}

// 判据 ②（human/human）：两侧都无策略参与时返回 not_applicable，不返回 identical。
// 这看起来只是措辞差别，但它决定了前端会不会画出「策略未变」这个断言——
// 而 human 路径根本没有策略可比，断言「未变」是纯粹的虚构。
func TestReview_PolicyComparisonNotApplicableWhenNoPolicyInvolved(t *testing.T) {
	f := newDryRunFixture(t, "review_cmp_na")
	inv, err := f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).
		SetUserID(f.userID).
		SetToolName("probe_write_ticket").
		SetArguments(`{"ticketId":3}`).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending").
		SetPermissionCheck("passed").
		SetRoleSnapshot("super_admin").
		SetPolicyVersion(""). // 当年无策略概念
		SetDryRun(false).
		Save(context.Background())
	require.NoError(t, err)

	status, resp := reviewAs(t, f.handler, f.tenantID, f.userID, inv.ID)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, dto.PolicyComparisonNotApplicable, resp.Data.PolicyComparison,
		"两侧都无策略参与时不得断言「未变」——那是虚构的结论")
}

// 判据 ③：复核不得改动被复核记录，也不得建新待审批行。
// 「复核只读」如果只写在注释里，某次重构就可能把它写成「顺便更新审批理由」。
func TestReview_MutatesNothing(t *testing.T) {
	f := newDryRunFixture(t, "review_no_side_effect")
	invID := pendingInvocationForReview(t, f, "probe_write_ticket", "")

	before, err := f.client.ToolInvocation.Get(context.Background(), invID)
	require.NoError(t, err)
	pendingBefore := f.client.ToolInvocation.Query().CountX(context.Background())

	status, _ := reviewAs(t, f.handler, f.tenantID, f.userID, invID)
	require.Equal(t, http.StatusOK, status)

	after, err := f.client.ToolInvocation.Get(context.Background(), invID)
	require.NoError(t, err)
	require.Equal(t, before.ApprovalState, after.ApprovalState, "复核不得改动审批态")
	require.Equal(t, before.Status, after.Status, "复核不得改动执行状态")
	require.Equal(t, before.ApprovalReason, after.ApprovalReason)

	// 待审批行数：只有被复核那一条。复核自身落的是 dry_run=true 行，
	// 不进入审批态集合，因此这里的计数不应变化。
	nonDryRunPending := f.client.ToolInvocation.Query().
		Where(toolinvocation.DryRunEQ(false)).
		CountX(context.Background())
	require.Equal(t, pendingBefore, nonDryRunPending)
}

// 判据 ④：跨租户复核必须表现为「记录不存在」，而不是读到别人的调用。
func TestReview_IsTenantScoped(t *testing.T) {
	f := newDryRunFixture(t, "review_tenant")
	other, err := f.client.Tenant.Create().
		SetName("OtherReview").SetCode("OTHER_REVIEW").SetDomain("other-review.test").SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)

	otherInv, err := f.client.ToolInvocation.Create().
		SetTenantID(other.ID).
		SetUserID(f.userID).
		SetToolName("probe_write_ticket").
		SetArguments(`{"ticketId":4}`).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending_agent").
		SetDryRun(false).
		Save(context.Background())
	require.NoError(t, err)

	status, resp := reviewAs(t, f.handler, f.tenantID, f.userID, otherInv.ID)
	require.Equal(t, http.StatusNotFound, status, "跨租户复核必须表现为资源不存在")
	require.NotZero(t, resp.Code)
}

// 重放语义：同一个调用可以反复复核，每次都落一条独立审计。
// 「复核一次就够了」是错误直觉——审批人可能在策略变更前后各查一次，
// 而两次结论的差异正是需要留痕的证据。
func TestReview_RepeatedReviewAuditsEachTime(t *testing.T) {
	f := newDryRunFixture(t, "review_repeat")
	invID := pendingInvocationForReview(t, f, "probe_write_ticket", "")

	var auditIDs []int
	for i := 0; i < 2; i++ {
		status, resp := reviewAs(t, f.handler, f.tenantID, f.userID, invID)
		require.Equal(t, http.StatusOK, status)
		require.NotZero(t, resp.Data.AuditID)
		auditIDs = append(auditIDs, resp.Data.AuditID)
	}
	require.NotEqual(t, auditIDs[0], auditIDs[1], "每次复核各落一条审计行")

	rows := dryRunRows(t, f.client, f.tenantID)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.True(t, r.DryRun)
		require.Equal(t, "none", r.ApprovalState, "复核审计不得进入审批态集合")
	}
}
