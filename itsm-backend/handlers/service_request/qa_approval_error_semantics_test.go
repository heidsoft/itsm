package service_request

import (
	"net/http"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyApproval_UnauthorizedActorReturnsForbiddenNotInternalError 固化 2026-11-04 prod 实测缺陷：
//
// 现状：非该节点审批人（例如 super_admin admin，而节点审批人是别人）点「通过」时，
// BPMN 引擎 authorizeTaskActor 返回普通 error（"当前用户不是该任务的审批人或候选人"），
// 该 error 被 fmt.Errorf("同步流程审批任务失败: %w") 二次包装，AppError 类型丢失；
// handler 的 failServiceRequest 落到 common.RespondError 兜底分支 → HTTP 500「操作失败」。
//
// 问题：① 状态码语义错误——这是「你不是审批人」，应为 403，不是 500；
//
//	② 文案被吞——用户只看到"操作失败"，拿不到"你不是该节点的审批人"这类可行动结论，
//	   且请求 ID 之外没有任何排查线索（对比同文件其他分支均给出可行动文案）。
//
// 已有测试 TestServiceRequestApproval_BridgeFailClosedForUnauthorizedActor 只断言
// "不等于成功码"，500 同样能通过，因此这个缺陷能上线。
func TestApplyApproval_UnauthorizedActorReturnsForbiddenNotInternalError(t *testing.T) {
	r, client, tenantID, uid, catID := srSetupBridge(t)
	id := srCreateOne(t, r, catID)
	// BPMN 任务指派给其他人：当前操作人不是流程任务审批人
	srCreateBPMNFixture(t, client, tenantID, "qa_unauth", id, uid+1000)

	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})

	// 期望：明确的 403 + 可行动文案，而不是 500「操作失败」。
	require.NotEqual(t, http.StatusInternalServerError, resp.Code,
		"非审批人操作审批不能返回 500；实际 body=%s", srStr(resp))
	assert.Equal(t, common.ForbiddenErrorCode, resp.Code,
		"非该节点审批人应返回 403 Forbidden；实际 body=%s", srStr(resp))

	msg, _ := resp.Data.(string)
	assert.NotContains(t, srStr(resp), "Operation failed",
		"错误文案不得被兜底覆盖成「操作失败」，应透传可行动原因")
	_ = msg
}
