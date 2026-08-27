package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicerequest"

	"go.uber.org/zap"
)

// ServiceRequestServiceTaskHandler 服务请求服务任务处理器
type ServiceRequestServiceTaskHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewServiceRequestServiceTaskHandler 创建服务请求处理器
func NewServiceRequestServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *ServiceRequestServiceTaskHandler {
	return &ServiceRequestServiceTaskHandler{
		client: client,
		logger: logger,
	}
}

// GetTaskType 返回任务类型
func (h *ServiceRequestServiceTaskHandler) GetTaskType() string {
	return "service_request_task"
}

// GetHandlerID 返回处理器标识
func (h *ServiceRequestServiceTaskHandler) GetHandlerID() string {
	return "service_request_handler"
}

// Execute 执行服务请求任务
func (h *ServiceRequestServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_request":
		return h.createRequest(ctx, variables)
	case "update_request":
		return h.updateRequest(ctx, variables)
	case "approve_request":
		return h.approveRequest(ctx, variables)
	case "reject_request":
		return h.rejectRequest(ctx, variables)
	case "assign_request":
		return h.assignRequest(ctx, variables)
	case "provision_resource":
		return h.provisionResource(ctx, variables)
	case "complete_request":
		return h.completeRequest(ctx, variables)
	case "cancel_request":
		return h.cancelRequest(ctx, variables)
	default:
		return &dto.ServiceTaskResult{Success: true, Message: "无操作执行"}, nil
	}
}

// Validate 验证配置
func (h *ServiceRequestServiceTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// getRequestByIDWithTenant 按 ID + TenantID 查询，找不到或不属于该租户均返回错误。
func (h *ServiceRequestServiceTaskHandler) getRequestByIDWithTenant(ctx context.Context, requestID, tenantID int) (*ent.ServiceRequest, error) {
	if requestID <= 0 {
		return nil, fmt.Errorf("无效的请求ID: %d", requestID)
	}
	sr, err := h.client.ServiceRequest.Query().
		Where(
			servicerequest.ID(requestID),
			servicerequest.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("服务请求不存在或不属于当前租户: %d (tenant=%d)", requestID, tenantID)
	}
	return sr, nil
}

// createRequest 创建服务请求
func (h *ServiceRequestServiceTaskHandler) createRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	catalogID := GetIntFromVars(variables, "catalog_id")
	requesterID := GetIntFromVars(variables, "requester_id")
	reason, _ := variables["reason"].(string)

	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		h.logger.Errorw("BPMN createRequest 缺少租户上下文", "error", err)
		return nil, fmt.Errorf("创建服务请求失败: %w", err)
	}

	if title == "" {
		return nil, fmt.Errorf("请求标题不能为空")
	}
	if catalogID <= 0 {
		return nil, fmt.Errorf("服务目录ID不能为空")
	}
	if requesterID <= 0 {
		return nil, fmt.Errorf("申请人ID不能为空")
	}

	sr, err := h.client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalogID).
		SetRequesterID(requesterID).
		SetTitle(title).
		SetReason(reason).
		SetStatus("submitted").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest created via BPMN",
		"request_id", sr.ID, "title", title, "catalog_id", catalogID, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("服务请求 %d 已创建", sr.ID),
		OutputVars: map[string]interface{}{"request_id": sr.ID, "tenant_id": tenantID},
	}, nil
}

// updateRequest 更新服务请求
func (h *ServiceRequestServiceTaskHandler) updateRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("更新服务请求失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	updateQuery := h.client.ServiceRequest.UpdateOneID(requestID)

	if title, ok := variables["title"].(string); ok && title != "" {
		updateQuery.SetTitle(title)
	}
	if reason, ok := variables["reason"].(string); ok && reason != "" {
		updateQuery.SetReason(reason)
	}
	if status, ok := variables["status"].(string); ok && status != "" {
		updateQuery.SetStatus(status)
	}

	_, err = updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest updated via BPMN", "request_id", requestID, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已更新", requestID),
	}, nil
}

// approveRequest 审批通过
func (h *ServiceRequestServiceTaskHandler) approveRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	comment, _ := variables["approver_comment"].(string)
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("审批服务请求失败: %w", err)
	}

	sr, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newLevel := sr.CurrentLevel + 1
	newStatus := "approved"
	if newLevel <= sr.TotalLevels {
		newStatus = "in_progress"
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetStatus(newStatus).
		SetCurrentLevel(newLevel).
		SetApprovedAt(now).
		SetApproverComment(comment).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("审批服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest approved via BPMN",
		"request_id", requestID, "level", newLevel, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 第 %d 级审批已通过", requestID, sr.CurrentLevel),
	}, nil
}

// rejectRequest 驳回服务请求
func (h *ServiceRequestServiceTaskHandler) rejectRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	reason, _ := variables["reject_reason"].(string)
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("驳回服务请求失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetStatus("rejected").
		SetApproverComment(reason).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("驳回服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest rejected via BPMN",
		"request_id", requestID, "reason", reason, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已被驳回: %s", requestID, reason),
	}, nil
}

// assignRequest 分配服务请求
func (h *ServiceRequestServiceTaskHandler) assignRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	assigneeID := GetIntFromVars(variables, "assignee_id")
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("分配服务请求失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetProcessorID(assigneeID).
		SetStatus("in_progress").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest assigned via BPMN",
		"request_id", requestID, "assignee_id", assigneeID, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已分配给用户 %d", requestID, assigneeID),
	}, nil
}

// provisionResource 资源预置（标记开始实施）
func (h *ServiceRequestServiceTaskHandler) provisionResource(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	resourceType, _ := variables["resource_type"].(string)
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("预置资源失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetStatus("in_progress").
		SetStartedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("预置资源失败: %w", err)
	}

	h.logger.Infow("ServiceRequest resource provisioning via BPMN",
		"request_id", requestID, "resource_type", resourceType, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 资源 %s 正在供应中", requestID, resourceType),
	}, nil
}

// completeRequest 完成服务请求
func (h *ServiceRequestServiceTaskHandler) completeRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	note, _ := variables["completion_note"].(string)
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("完成服务请求失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetStatus("completed").
		SetCompletedAt(time.Now()).
		SetCompletionNote(note).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("完成服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest completed via BPMN",
		"request_id", requestID, "note", note, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已完成", requestID),
	}, nil
}

// cancelRequest 取消服务请求
func (h *ServiceRequestServiceTaskHandler) cancelRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := GetIntFromVars(variables, "request_id")
	reason, _ := variables["cancel_reason"].(string)
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, fmt.Errorf("取消服务请求失败: %w", err)
	}

	if _, err := h.getRequestByIDWithTenant(ctx, requestID, tenantID); err != nil {
		return nil, err
	}

	_, err = h.client.ServiceRequest.UpdateOneID(requestID).
		SetStatus("cancelled").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("取消服务请求失败: %w", err)
	}

	h.logger.Infow("ServiceRequest cancelled via BPMN",
		"request_id", requestID, "reason", reason, "tenant_id", tenantID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已取消: %s", requestID, reason),
	}, nil
}

// Ensure ServiceRequestServiceTaskHandler implements ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*ServiceRequestServiceTaskHandler)(nil)
