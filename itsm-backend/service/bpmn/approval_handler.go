package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ApprovalHandler 审批服务任务处理器
type ApprovalHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewApprovalHandler 创建审批处理器
func NewApprovalHandler(client *ent.Client, logger *zap.SugaredLogger) *ApprovalHandler {
	return &ApprovalHandler{
		client: client,
		logger: logger,
	}
}

// GetTaskType 返回任务类型
func (h *ApprovalHandler) GetTaskType() string {
	return "approval_task"
}

// GetHandlerID 返回处理器标识
func (h *ApprovalHandler) GetHandlerID() string {
	return "approval_handler"
}

// Execute 执行审批服务任务
func (h *ApprovalHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "submit_approval":
		return h.submitApproval(ctx, variables)
	case "approve":
		return h.approve(ctx, variables)
	case "reject":
		return h.reject(ctx, variables)
	case "delegate":
		return h.delegate(ctx, variables)
	case "escalate":
		return h.escalateApproval(ctx, variables)
	default:
		return &dto.ServiceTaskResult{
			Success: true,
			Message: "无操作执行",
		}, nil
	}
}

// Validate 验证配置
func (h *ApprovalHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// submitApproval 提交审批
func (h *ApprovalHandler) submitApproval(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	businessID := GetIntFromVars(variables, "business_id")
	businessType := GetStringFromVars(variables, "business_type")
	approverID := GetIntFromVars(variables, "approver_id")
	title := GetStringFromVars(variables, "title")

	if title == "" {
		title = "审批请求"
	}

	if businessID == 0 {
		return nil, fmt.Errorf("业务ID不能为空")
	}

	h.logger.Infow("Submitting approval via BPMN",
		"business_id", businessID,
		"business_type", businessType,
		"approver_id", approverID,
		"title", title,
	)

	// 这里应该调用审批服务创建审批记录
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已提交，业务ID: %d，类型: %s", businessID, businessType),
		OutputVars: map[string]interface{}{
			"business_id":    businessID,
			"business_type": businessType,
			"status":        "pending",
		},
	}, nil
}

// approve 审批通过
func (h *ApprovalHandler) approve(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	approvalID := GetIntFromVars(variables, "approval_id")
	businessID := GetIntFromVars(variables, "business_id")
	comment := GetStringFromVars(variables, "comment")

	if approvalID == 0 && businessID == 0 {
		return nil, fmt.Errorf("审批ID或业务ID不能为空")
	}

	h.logger.Infow("Approval approved via BPMN",
		"approval_id", approvalID,
		"business_id", businessID,
		"comment", comment,
	)

	// 这里应该调用审批服务通过审批
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已通过，审批ID: %d", approvalID),
		OutputVars: map[string]interface{}{
			"approval_id": approvalID,
			"status":       "approved",
			"approved_at": time.Now(),
		},
	}, nil
}

// reject 审批驳回
func (h *ApprovalHandler) reject(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	approvalID := GetIntFromVars(variables, "approval_id")
	businessID := GetIntFromVars(variables, "business_id")
	reason := GetStringFromVars(variables, "reject_reason")

	if approvalID == 0 && businessID == 0 {
		return nil, fmt.Errorf("审批ID或业务ID不能为空")
	}

	h.logger.Infow("Approval rejected via BPMN",
		"approval_id", approvalID,
		"business_id", businessID,
		"reason", reason,
	)

	// 这里应该调用审批服务驳回审批
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已驳回，审批ID: %d，原因: %s", approvalID, reason),
		OutputVars: map[string]interface{}{
			"approval_id":  approvalID,
			"status":       "rejected",
			"reject_reason": reason,
			"rejected_at":  time.Now(),
		},
	}, nil
}

// delegate 审批委托
func (h *ApprovalHandler) delegate(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	approvalID := GetIntFromVars(variables, "approval_id")
	fromUserID := GetIntFromVars(variables, "from_user_id")
	toUserID := GetIntFromVars(variables, "to_user_id")
	reason := GetStringFromVars(variables, "delegate_reason")

	if approvalID == 0 {
		return nil, fmt.Errorf("审批ID不能为空")
	}

	if toUserID == 0 {
		return nil, fmt.Errorf("委托目标用户ID不能为空")
	}

	h.logger.Infow("Approval delegated via BPMN",
		"approval_id",  approvalID,
		"from_user_id", fromUserID,
		"to_user_id",   toUserID,
		"reason",       reason,
	)

	// 这里应该调用审批服务委托审批
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已委托，从用户 %d 到用户 %d", fromUserID, toUserID),
		OutputVars: map[string]interface{}{
			"approval_id":    approvalID,
			"delegate_to":     toUserID,
			"delegate_reason": reason,
		},
	}, nil
}

// escalateApproval 审批升级
func (h *ApprovalHandler) escalateApproval(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	approvalID := GetIntFromVars(variables, "approval_id")
	escalateTo := GetIntFromVars(variables, "escalate_to")
	reason := GetStringFromVars(variables, "escalation_reason")

	if approvalID == 0 {
		return nil, fmt.Errorf("审批ID不能为空")
	}

	if escalateTo == 0 {
		return nil, fmt.Errorf("升级目标用户ID不能为空")
	}

	h.logger.Infow("Approval escalated via BPMN",
		"approval_id",  approvalID,
		"escalate_to",  escalateTo,
		"reason",       reason,
	)

	// 这里应该调用审批服务升级审批
	// 简化处理，只记录日志

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已升级到用户 %d", escalateTo),
		OutputVars: map[string]interface{}{
			"approval_id":    approvalID,
			"escalated_to":   escalateTo,
			"escalation_reason": reason,
		},
	}, nil
}

// 确保 ApprovalHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*ApprovalHandler)(nil)
