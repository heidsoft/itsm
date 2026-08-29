package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ApprovalServiceInterface 审批服务接口（避免循环依赖）
type ApprovalServiceInterface interface {
	SubmitApproval(ctx context.Context, recordID int, userID int, action string, comment string, delegateToUserID *int, tenantID int) error
}

// ApprovalHandler 审批服务任务处理器
// 通过 BPMN ServiceTask 节点执行审批操作：approve / reject / delegate / escalate
type ApprovalHandler struct {
	HandlerBase
	client          *ent.Client
	logger          *zap.SugaredLogger
	approvalService ApprovalServiceInterface
}

// NewApprovalHandler 创建审批处理器
func NewApprovalHandler(client *ent.Client, logger *zap.SugaredLogger) *ApprovalHandler {
	return &ApprovalHandler{
		client: client,
		logger: logger,
	}
}

// SetApprovalService 设置审批服务（外部注入以避免循环依赖）
func (h *ApprovalHandler) SetApprovalService(svc ApprovalServiceInterface) {
	h.approvalService = svc
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
// 动作通过 variables["action"] 传入，支持：approve / reject / delegate / escalate
func (h *ApprovalHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 1. 租户隔离（强制）：拒绝缺失 tenant_id 的执行
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		h.logger.Errorw("ApprovalHandler: tenantID missing, rejecting execution",
			"error", err, "variables", fmt.Sprintf("%v", variables))
		return nil, fmt.Errorf("拒绝执行：%w", err)
	}

	// 2. 提取操作参数
	action := GetStringFromVars(variables, "action")
	approvalID := GetIntFromVars(variables, "approval_id")
	comment := GetStringFromVars(variables, "comment")
	userID := GetIntFromVars(variables, "user_id")
	delegateToUserID := GetIntFromVars(variables, "delegate_to_user_id")

	switch action {
	case "approve", "reject":
		// approve/reject 需要：approvalID + userID
		if approvalID <= 0 {
			return nil, fmt.Errorf("approval_id 为空，无法执行审批")
		}
		if userID <= 0 {
			return nil, fmt.Errorf("user_id 为空，无法执行审批")
		}
		return h.doApproveReject(ctx, action, approvalID, userID, comment, tenantID)

	case "delegate":
		// delegate 需要：approvalID + userID + delegateToUserID
		if approvalID <= 0 {
			return nil, fmt.Errorf("approval_id 为空，无法委托审批")
		}
		if userID <= 0 {
			return nil, fmt.Errorf("user_id 为空，无法委托审批")
		}
		if delegateToUserID <= 0 {
			return nil, fmt.Errorf("delegate_to_user_id 为空，无法委托审批")
		}
		return h.doDelegate(ctx, approvalID, userID, delegateToUserID, comment, tenantID)

	case "escalate":
		// escalate 需要：approvalID + userID + delegateToUserID
		if approvalID <= 0 {
			return nil, fmt.Errorf("approval_id 为空，无法升级审批")
		}
		if delegateToUserID <= 0 {
			return nil, fmt.Errorf("escalate_to（delegate_to_user_id）为空，无法升级审批")
		}
		return h.doEscalate(ctx, approvalID, userID, delegateToUserID, comment, tenantID)

	default:
		return &dto.ServiceTaskResult{
			Success: true,
			Message: fmt.Sprintf("审批操作 [%s] 无需执行", action),
		}, nil
	}
}

// Validate 验证配置（暂不支持 BPMN Extension Properties 预校验）
func (h *ApprovalHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// doApproveReject 执行 approve 或 reject
func (h *ApprovalHandler) doApproveReject(ctx context.Context, action string, approvalID, userID int, comment string, tenantID int) (*dto.ServiceTaskResult, error) {
	if h.approvalService == nil {
		h.logger.Errorw("ApprovalHandler: approvalService not injected, this handler must be initialized with SetApprovalService before use")
		return nil, fmt.Errorf("审批服务未初始化，请联系管理员")
	}

	h.logger.Infow("Executing approval action via BPMN",
		"action", action,
		"approval_id", approvalID,
		"user_id", userID,
		"tenant_id", tenantID,
	)

	var delegateTo *int
	if err := h.approvalService.SubmitApproval(ctx, approvalID, userID, action, comment, delegateTo, tenantID); err != nil {
		h.logger.Warnw("Approval action failed",
			"action", action, "approval_id", approvalID, "error", err)
		return nil, fmt.Errorf("审批操作失败：%w", err)
	}

	status := "approved"
	if action == "reject" {
		status = "rejected"
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已%s，审批ID: %d", status, approvalID),
		OutputVars: map[string]interface{}{
			"approval_id": approvalID,
			"status":      status,
			"action":      action,
			"approved_at": time.Now(),
		},
	}, nil
}

// doDelegate 执行委托
func (h *ApprovalHandler) doDelegate(ctx context.Context, approvalID, userID, delegateToUserID int, comment string, tenantID int) (*dto.ServiceTaskResult, error) {
	if h.approvalService == nil {
		return nil, fmt.Errorf("审批服务未初始化，请联系管理员")
	}

	h.logger.Infow("Delegating approval via BPMN",
		"approval_id", approvalID,
		"from_user_id", userID,
		"to_user_id", delegateToUserID,
		"tenant_id", tenantID,
	)

	delegateTo := &delegateToUserID
	if err := h.approvalService.SubmitApproval(ctx, approvalID, userID, "delegate", comment, delegateTo, tenantID); err != nil {
		h.logger.Warnw("Delegate action failed", "approval_id", approvalID, "error", err)
		return nil, fmt.Errorf("委托审批失败：%w", err)
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已委托，从用户 %d 到用户 %d", userID, delegateToUserID),
		OutputVars: map[string]interface{}{
			"approval_id":   approvalID,
			"status":        "delegated",
			"delegate_from": userID,
			"delegate_to":   delegateToUserID,
			"delegated_at":  time.Now(),
		},
	}, nil
}

// doEscalate 执行升级（内部实现为带备注的委托）
func (h *ApprovalHandler) doEscalate(ctx context.Context, approvalID, userID, escalateTo int, reason string, tenantID int) (*dto.ServiceTaskResult, error) {
	if h.approvalService == nil {
		return nil, fmt.Errorf("审批服务未初始化，请联系管理员")
	}

	h.logger.Infow("Escalating approval via BPMN",
		"approval_id", approvalID,
		"escalate_to", escalateTo,
		"tenant_id", tenantID,
	)

	escalateToPtr := &escalateTo
	comment := reason
	if comment == "" {
		comment = "系统自动升级"
	}

	if err := h.approvalService.SubmitApproval(ctx, approvalID, userID, "delegate", comment, escalateToPtr, tenantID); err != nil {
		h.logger.Warnw("Escalate action failed", "approval_id", approvalID, "error", err)
		return nil, fmt.Errorf("升级审批失败：%w", err)
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("审批已升级到用户 %d", escalateTo),
		OutputVars: map[string]interface{}{
			"approval_id":  approvalID,
			"status":       "escalated",
			"escalated_to": escalateTo,
			"escalated_at": time.Now(),
		},
	}, nil
}

// Ensure ApprovalHandler implements ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*ApprovalHandler)(nil)
