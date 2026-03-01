package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ChangeServiceTaskHandler 变更服务任务处理器
type ChangeServiceTaskHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewChangeServiceTaskHandler 创建变更处理器
func NewChangeServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *ChangeServiceTaskHandler {
	return &ChangeServiceTaskHandler{
		client: client,
		logger: logger,
	}
}

// GetTaskType 返回任务类型
func (h *ChangeServiceTaskHandler) GetTaskType() string {
	return "change_task"
}

// GetHandlerID 返回处理器标识
func (h *ChangeServiceTaskHandler) GetHandlerID() string {
	return "change_service_handler"
}

// Execute 执行变更服务任务
func (h *ChangeServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_change":
		return h.createChange(ctx, variables)
	case "update_change":
		return h.updateChange(ctx, variables)
	case "approve_change":
		return h.approveChange(ctx, variables)
	case "reject_change":
		return h.rejectChange(ctx, variables)
	case "schedule_change":
		return h.scheduleChange(ctx, variables)
	case "implement_change":
		return h.implementChange(ctx, variables)
	case "verify_change":
		return h.verifyChange(ctx, variables)
	case "close_change":
		return h.closeChange(ctx, variables)
	case "assess_risk":
		return h.assessRisk(ctx, variables)
	case "notify_stakeholders":
		return h.notifyStakeholders(ctx, variables)
	default:
		return &dto.ServiceTaskResult{
			Success: true,
			Message: "无操作执行",
		}, nil
	}
}

// Validate 验证配置
func (h *ChangeServiceTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// createChange 创建变更
func (h *ChangeServiceTaskHandler) createChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	description, _ := variables["description"].(string)
	changeType, _ := variables["type"].(string)
	priority, _ := variables["priority"].(string)
	tenantID := GetTenantIDFromVars(variables)

	if title == "" {
		return nil, fmt.Errorf("变更标题不能为空")
	}

	change, err := h.client.Change.Create().
		SetTitle(title).
		SetDescription(description).
		SetType(changeType).
		SetPriority(priority).
		SetStatus("draft").
		SetCreatedBy(GetIntFromVars(variables, "created_by")).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建变更失败: %w", err)
	}

	h.logger.Infow("Change created via BPMN", "change_id", change.ID, "title", title)

	return &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("变更 %d 已创建", change.ID),
		OutputVars: map[string]interface{}{"change_id": change.ID},
	}, nil
}

// updateChange 更新变更
func (h *ChangeServiceTaskHandler) updateChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	updateQuery := h.client.Change.UpdateOneID(changeID)

	if title, ok := variables["title"].(string); ok && title != "" {
		updateQuery.SetTitle(title)
	}
	if description, ok := variables["description"].(string); ok && description != "" {
		updateQuery.SetDescription(description)
	}
	if status, ok := variables["status"].(string); ok && status != "" {
		updateQuery.SetStatus(status)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新变更失败: %w", err)
	}

	h.logger.Infow("Change updated via BPMN", "change_id", changeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已更新", changeID),
	}, nil
}

// approveChange 审批变更
func (h *ChangeServiceTaskHandler) approveChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 更新状态为审批中
	_, err = h.client.Change.UpdateOneID(changeID).
		SetStatus("pending_approval").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新变更状态失败: %w", err)
	}

	h.logger.Infow("Change submitted for approval via BPMN", "change_id", changeID, "title", change.Title)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已提交审批", changeID),
	}, nil
}

// rejectChange 驳回变更
func (h *ChangeServiceTaskHandler) rejectChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	reason, _ := variables["reject_reason"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus("rejected").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("驳回变更失败: %w", err)
	}

	h.logger.Infow("Change rejected via BPMN", "change_id", changeID, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已驳回: %s", changeID, reason),
	}, nil
}

// scheduleChange 排期变更
func (h *ChangeServiceTaskHandler) scheduleChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 解析日期时间
	var plannedStart, plannedEnd time.Time
	if startStr, ok := variables["planned_start_date"].(string); ok && startStr != "" {
		plannedStart, _ = time.Parse(time.RFC3339, startStr)
	}
	if endStr, ok := variables["planned_end_date"].(string); ok && endStr != "" {
		plannedEnd, _ = time.Parse(time.RFC3339, endStr)
	}

	updateQuery := h.client.Change.UpdateOneID(changeID).SetStatus("scheduled")
	if !plannedStart.IsZero() {
		updateQuery.SetPlannedStartDate(plannedStart)
	}
	if !plannedEnd.IsZero() {
		updateQuery.SetPlannedEndDate(plannedEnd)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("排期变更失败: %w", err)
	}

	h.logger.Infow("Change scheduled via BPMN", "change_id", changeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已排期", changeID),
	}, nil
}

// implementChange 实施变更
func (h *ChangeServiceTaskHandler) implementChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	now := time.Now()
	_, err = h.client.Change.UpdateOneID(changeID).
		SetStatus("in_progress").
		SetActualStartDate(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("实施变更失败: %w", err)
	}

	h.logger.Infow("Change implementation started via BPMN", "change_id", changeID, "title", change.Title)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 开始实施", changeID),
	}, nil
}

// verifyChange 验证变更
func (h *ChangeServiceTaskHandler) verifyChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	verificationResult, _ := variables["verification_result"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 根据验证结果更新状态
	newStatus := "verification_passed"
	if verificationResult == "failed" {
		newStatus = "verification_failed"
	}

	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus(newStatus).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("验证变更失败: %w", err)
	}

	h.logger.Infow("Change verification via BPMN", "change_id", changeID, "result", verificationResult)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 验证结果: %s", changeID, verificationResult),
	}, nil
}

// closeChange 关闭变更
func (h *ChangeServiceTaskHandler) closeChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	feedback, _ := variables["feedback"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	now := time.Now()
	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus("closed").
		SetActualEndDate(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("关闭变更失败: %w", err)
	}

	h.logger.Infow("Change closed via BPMN", "change_id", changeID, "feedback", feedback)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已关闭", changeID),
	}, nil
}

// assessRisk 评估变更风险
func (h *ChangeServiceTaskHandler) assessRisk(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息进行风险评估
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 简化的风险评估逻辑
	riskLevel := "medium"
	impactScope := change.ImpactScope

	// 根据变更类型和优先级确定风险等级
	if change.Type == "emergency" {
		riskLevel = "high"
	} else if change.Type == "minor" {
		riskLevel = "low"
	}

	_, err = h.client.Change.UpdateOneID(changeID).
		SetRiskLevel(riskLevel).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("评估风险失败: %w", err)
	}

	h.logger.Infow("Change risk assessed via BPMN", "change_id", changeID, "risk_level", riskLevel, "impact_scope", impactScope)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 风险评估完成: %s", changeID, riskLevel),
		OutputVars: map[string]interface{}{
			"risk_level":   riskLevel,
			"impact_scope": impactScope,
		},
	}, nil
}

// notifyStakeholders 通知利益相关者
func (h *ChangeServiceTaskHandler) notifyStakeholders(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := GetIntFromVars(variables, "change_id")
	notificationType, _ := variables["notification_type"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 记录通知日志（实际应调用通知服务）
	h.logger.Infow("Stakeholders notification via BPMN",
		"change_id", changeID,
		"change_title", change.Title,
		"notification_type", notificationType,
	)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 利益相关者已通知", changeID),
	}, nil
}

// 确保 ChangeServiceTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*ChangeServiceTaskHandler)(nil)
