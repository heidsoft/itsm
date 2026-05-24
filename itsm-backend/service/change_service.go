package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

// ChangeService 变更管理服务
type ChangeService struct {
	client                *ent.Client
	logger                *zap.SugaredLogger
	processTriggerService ProcessTriggerServiceInterface
	approvalService       *ApprovalService
}

// NewChangeService 创建变更管理服务
func NewChangeService(client *ent.Client, logger *zap.SugaredLogger) *ChangeService {
	return &ChangeService{
		client: client,
		logger: logger,
	}
}

// SetProcessTriggerService 设置流程触发服务
func (s *ChangeService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

// SetApprovalService 设置审批服务
func (s *ChangeService) SetApprovalService(approvalSvc *ApprovalService) {
	s.approvalService = approvalSvc
}

// CreateChange 创建变更
func (s *ChangeService) CreateChange(ctx context.Context, req *dto.CreateChangeRequest, createdBy, tenantID int) (*dto.ChangeResponse, error) {
	// 创建变更记录
	changeEntity, err := s.client.Change.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetJustification(req.Justification).
		SetType(req.Type).
		SetStatus(string(dto.ChangeStatusDraft)).
		SetPriority(req.Priority).
		SetImpactScope(req.ImpactScope).
		SetRiskLevel(req.RiskLevel).
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		SetImplementationPlan(req.ImplementationPlan).
		SetRollbackPlan(req.RollbackPlan).
		SetNillablePlannedStartDate(req.PlannedStartDate).
		SetNillablePlannedEndDate(req.PlannedEndDate).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create change", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create change: %w", err)
	}

	// 设置受影响的配置项
	if len(req.AffectedCIs) > 0 {
		_, err = changeEntity.Update().SetAffectedCis(req.AffectedCIs).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set affected CIs", "error", err, "change_id", changeEntity.ID)
		}
	}

	// 获取创建人信息
	creator, err := s.client.User.Get(ctx, createdBy)
	if err != nil {
		s.logger.Warnw("Failed to get creator info", "error", err, "user_id", createdBy)
	}

	// 构建响应
	response := &dto.ChangeResponse{
		ID:                 changeEntity.ID,
		Title:              changeEntity.Title,
		Description:        changeEntity.Description,
		Justification:      changeEntity.Justification,
		Type:               dto.ChangeType(changeEntity.Type),
		Status:             dto.ChangeStatus(changeEntity.Status),
		Priority:           dto.ChangePriority(changeEntity.Priority),
		ImpactScope:        dto.ChangeImpact(changeEntity.ImpactScope),
		RiskLevel:          dto.ChangeRisk(changeEntity.RiskLevel),
		CreatedBy:          changeEntity.CreatedBy,
		CreatedByName:      creator.Name,
		TenantID:           changeEntity.TenantID,
		PlannedStartDate:   &changeEntity.PlannedStartDate,
		PlannedEndDate:     &changeEntity.PlannedEndDate,
		ImplementationPlan: changeEntity.ImplementationPlan,
		RollbackPlan:       changeEntity.RollbackPlan,
		CreatedAt:          changeEntity.CreatedAt,
		UpdatedAt:          changeEntity.UpdatedAt,
	}

	// 设置受影响的配置项
	if len(changeEntity.AffectedCis) > 0 {
		response.AffectedCIs = changeEntity.AffectedCis
	}

	// 设置相关工单
	if len(changeEntity.RelatedTickets) > 0 {
		response.RelatedTickets = changeEntity.RelatedTickets
	}

	// 触发审批流程（对于紧急或高风险变更）
	if s.approvalService != nil && (string(req.Priority) == "urgent" || string(req.RiskLevel) == "high" || string(req.RiskLevel) == "critical") {
		go func() {
			approvalCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// 生成变更编号
			changeNumber := fmt.Sprintf("CHG-%s-%06d", time.Now().Format("200602"), changeEntity.ID)
			approvalReq := &ApprovalTriggerRequest{
				TicketID:     changeEntity.ID,
				TicketNumber: changeNumber,
				TicketTitle:  changeEntity.Title,
				TicketType:   "change",
				Priority:     string(req.Priority),
				RequesterID:  createdBy,
				TenantID:     tenantID,
			}
			records, err := s.approvalService.TriggerApproval(approvalCtx, approvalReq)
			if err != nil {
				s.logger.Warnw("Failed to trigger approval for change", "error", err, "change_id", changeEntity.ID)
			} else if len(records) > 0 {
				s.logger.Infow("Approval triggered for change", "change_id", changeEntity.ID, "approval_records", len(records))
			}
		}()
	}

	// 触发BPMN工作流（异步执行，不阻塞变更创建）
	if s.processTriggerService != nil {
		go func() {
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForChange(workflowCtx, changeEntity.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to trigger workflow for change", "error", err, "change_id", changeEntity.ID)
			}
		}()
	}

	s.logger.Infow("Change created successfully", "change_id", changeEntity.ID, "tenant_id", tenantID)
	return response, nil
}

// GetChange 获取变更详情
func (s *ChangeService) GetChange(ctx context.Context, id int, tenantID int) (*dto.ChangeResponse, error) {
	changeEntity, err := s.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("change not found")
		}
		s.logger.Errorw("Failed to get change", "error", err, "change_id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to get change: %w", err)
	}

	// 获取创建人信息
	creator, err := s.client.User.Get(ctx, changeEntity.CreatedBy)
	if err != nil {
		s.logger.Warnw("Failed to get creator info", "error", err, "user_id", changeEntity.CreatedBy)
	}

	// 获取处理人信息
	var assigneeName *string
	if changeEntity.AssigneeID > 0 {
		assignee, err := s.client.User.Get(ctx, changeEntity.AssigneeID)
		if err == nil {
			assigneeName = &assignee.Name
		}
	}

	// 构建响应
	response := &dto.ChangeResponse{
		ID:                 changeEntity.ID,
		Title:              changeEntity.Title,
		Description:        changeEntity.Description,
		Justification:      changeEntity.Justification,
		Type:               dto.ChangeType(changeEntity.Type),
		Status:             dto.ChangeStatus(changeEntity.Status),
		Priority:           dto.ChangePriority(changeEntity.Priority),
		ImpactScope:        dto.ChangeImpact(changeEntity.ImpactScope),
		RiskLevel:          dto.ChangeRisk(changeEntity.RiskLevel),
		AssigneeID:         &changeEntity.AssigneeID,
		AssigneeName:       assigneeName,
		CreatedBy:          changeEntity.CreatedBy,
		CreatedByName:      creator.Name,
		TenantID:           changeEntity.TenantID,
		PlannedStartDate:   &changeEntity.PlannedStartDate,
		PlannedEndDate:     &changeEntity.PlannedEndDate,
		ActualStartDate:    &changeEntity.ActualStartDate,
		ActualEndDate:      &changeEntity.ActualEndDate,
		ImplementationPlan: changeEntity.ImplementationPlan,
		RollbackPlan:       changeEntity.RollbackPlan,
		CreatedAt:          changeEntity.CreatedAt,
		UpdatedAt:          changeEntity.UpdatedAt,
	}

	// 设置受影响的配置项
	if len(changeEntity.AffectedCis) > 0 {
		response.AffectedCIs = changeEntity.AffectedCis
	}

	// 设置相关工单
	if len(changeEntity.RelatedTickets) > 0 {
		response.RelatedTickets = changeEntity.RelatedTickets
	}

	return response, nil
}

// ListChanges 获取变更列表
func (s *ChangeService) ListChanges(ctx context.Context, tenantID int, page, pageSize int, status, search string) (*dto.ChangeListResponse, error) {
	query := s.client.Change.Query().Where(change.TenantID(tenantID))

	// 状态筛选
	if status != "" && status != "全部" {
		query = query.Where(change.Status(status))
	}

	// 搜索筛选
	if search != "" {
		query = query.Where(
			change.Or(
				change.TitleContains(search),
				change.DescriptionContains(search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count changes", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count changes: %w", err)
	}

	// 分页查询
	changes, err := query.
		Order(ent.Desc(change.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list changes", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list changes: %w", err)
	}

	// 构建响应列表
	var changeResponses []dto.ChangeResponse
	for _, changeEntity := range changes {
		// 获取创建人信息
		creator, err := s.client.User.Get(ctx, changeEntity.CreatedBy)
		if err != nil {
			s.logger.Warnw("Failed to get creator info", "error", err, "user_id", changeEntity.CreatedBy)
		}

		// 获取处理人信息
		var assigneeName *string
		if changeEntity.AssigneeID > 0 {
			assignee, err := s.client.User.Get(ctx, changeEntity.AssigneeID)
			if err == nil {
				assigneeName = &assignee.Name
			}
		}

		response := dto.ChangeResponse{
			ID:                 changeEntity.ID,
			Title:              changeEntity.Title,
			Description:        changeEntity.Description,
			Justification:      changeEntity.Justification,
			Type:               dto.ChangeType(changeEntity.Type),
			Status:             dto.ChangeStatus(changeEntity.Status),
			Priority:           dto.ChangePriority(changeEntity.Priority),
			ImpactScope:        dto.ChangeImpact(changeEntity.ImpactScope),
			RiskLevel:          dto.ChangeRisk(changeEntity.RiskLevel),
			AssigneeID:         &changeEntity.AssigneeID,
			AssigneeName:       assigneeName,
			CreatedBy:          changeEntity.CreatedBy,
			CreatedByName:      creator.Name,
			TenantID:           changeEntity.TenantID,
			PlannedStartDate:   &changeEntity.PlannedStartDate,
			PlannedEndDate:     &changeEntity.PlannedEndDate,
			ActualStartDate:    &changeEntity.ActualStartDate,
			ActualEndDate:      &changeEntity.ActualEndDate,
			ImplementationPlan: changeEntity.ImplementationPlan,
			RollbackPlan:       changeEntity.RollbackPlan,
			CreatedAt:          changeEntity.CreatedAt,
			UpdatedAt:          changeEntity.UpdatedAt,
		}

		// 设置受影响的配置项
		if len(changeEntity.AffectedCis) > 0 {
			response.AffectedCIs = changeEntity.AffectedCis
		}

		// 设置相关工单
		if len(changeEntity.RelatedTickets) > 0 {
			response.RelatedTickets = changeEntity.RelatedTickets
		}

		changeResponses = append(changeResponses, response)
	}

	return &dto.ChangeListResponse{
		Total:   total,
		Changes: changeResponses,
	}, nil
}

// UpdateChange 更新变更
func (s *ChangeService) UpdateChange(ctx context.Context, id int, req *dto.UpdateChangeRequest, tenantID int) (*dto.ChangeResponse, error) {
	// 检查变更是否存在
	changeEntity, err := s.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("change not found")
		}
		s.logger.Errorw("Failed to get change for update", "error", err, "change_id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to get change: %w", err)
	}

	// 构建更新字段
	update := changeEntity.Update()

	if req.Title != nil {
		update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Justification != nil {
		update.SetJustification(*req.Justification)
	}
	if req.Type != nil {
		update.SetType(string(*req.Type))
	}
	if req.Priority != nil {
		update.SetPriority(string(*req.Priority))
	}
	if req.ImpactScope != nil {
		update.SetImpactScope(string(*req.ImpactScope))
	}
	if req.RiskLevel != nil {
		update.SetRiskLevel(string(*req.RiskLevel))
	}
	if req.PlannedStartDate != nil {
		update.SetPlannedStartDate(*req.PlannedStartDate)
	}
	if req.PlannedEndDate != nil {
		update.SetPlannedEndDate(*req.PlannedEndDate)
	}
	if req.ImplementationPlan != nil {
		update.SetImplementationPlan(*req.ImplementationPlan)
	}
	if req.RollbackPlan != nil {
		update.SetRollbackPlan(*req.RollbackPlan)
	}

	// 更新受影响的配置项
	if req.AffectedCIs != nil {
		update.SetAffectedCis(req.AffectedCIs)
	}

	// 更新相关工单
	if req.RelatedTickets != nil {
		update.SetRelatedTickets(req.RelatedTickets)
	}

	// 执行更新
	_, err = update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update change", "error", err, "change_id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to update change: %w", err)
	}

	// 获取更新后的完整信息
	return s.GetChange(ctx, id, tenantID)
}

// DeleteChange 删除变更
func (s *ChangeService) DeleteChange(ctx context.Context, id int, tenantID int) error {
	// 检查变更是否存在
	exists, err := s.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check change existence", "error", err, "change_id", id, "tenant_id", tenantID)
		return fmt.Errorf("failed to check change existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("change not found")
	}

	// 删除变更
	err = s.client.Change.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete change", "error", err, "change_id", id, "tenant_id", tenantID)
		return fmt.Errorf("failed to delete change: %w", err)
	}

	s.logger.Infow("Change deleted successfully", "change_id", id, "tenant_id", tenantID)
	return nil
}

// GetChangeStats 获取变更统计
func (s *ChangeService) GetChangeStats(ctx context.Context, tenantID int) (*dto.ChangeStatsResponse, error) {
	// 获取总变更数
	total, err := s.client.Change.Query().Where(change.TenantID(tenantID)).Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total changes", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count total changes: %w", err)
	}

	// 获取各状态的变更数
	stats := &dto.ChangeStatsResponse{Total: total}

	// 待审批 - draft/submitted 状态是待审批前的状态
	// pending 和 submitted 是实际待审批状态
	draftOrSubmitted, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.StatusIn(string(dto.ChangeStatusDraft), "submitted")).
		Count(ctx)
	if err == nil {
		stats.Pending = draftOrSubmitted
	}

	// 已批准
	approved, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusApproved))).
		Count(ctx)
	if err == nil {
		stats.Approved = approved
	}

	// 实施中 - 包含 in_progress 和 scheduled
	inProgress, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusInProgress))).
		Count(ctx)
	if err == nil {
		stats.InProgress = inProgress
	}

	// 已完成
	completed, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusCompleted))).
		Count(ctx)
	if err == nil {
		stats.Completed = completed
	}

	// 已回滚
	rolledBack, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusRolledBack))).
		Count(ctx)
	if err == nil {
		stats.RolledBack = rolledBack
	}

	// 已拒绝
	rejected, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusRejected))).
		Count(ctx)
	if err == nil {
		stats.Rejected = rejected
	}

	// 已取消
	cancelled, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusCancelled))).
		Count(ctx)
	if err == nil {
		stats.Cancelled = cancelled
	}

	// 验证：确保各状态之和等于总数
	calculatedTotal := stats.Pending + stats.Approved + stats.InProgress + stats.Completed +
		stats.RolledBack + stats.Rejected + stats.Cancelled
	if calculatedTotal != total {
		s.logger.Warnw("Change stats sum mismatch",
			"total", total,
			"calculated_sum", calculatedTotal,
			"tenant_id", tenantID)
	}

	return stats, nil
}

// UpdateChangeStatus 更新变更状态
func (s *ChangeService) UpdateChangeStatus(ctx context.Context, id int, status dto.ChangeStatus, tenantID int) error {
	// 获取当前变更状态，验证租户所有权
	changeEntity, err := s.client.Change.Query().
		Where(
			change.ID(id),
			change.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("cross-tenant access denied: change not found")
		}
		return fmt.Errorf("failed to get change: %w", err)
	}

	// 验证状态转换
	if !isValidChangeStatusTransition(changeEntity.Status, string(status)) {
		return fmt.Errorf("invalid status transition from '%s' to '%s'", changeEntity.Status, status)
	}

	// 使用租户过滤进行更新，防止跨租户更新
	result, err := s.client.Change.Update().
		Where(
			change.ID(id),
			change.TenantID(tenantID),
		).
		SetStatus(string(status)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update change status", "error", err, "change_id", id, "status", status, "tenant_id", tenantID)
		return fmt.Errorf("failed to update change status: %w", err)
	}

	if result == 0 {
		return fmt.Errorf("cross-tenant access denied: change not found or access denied")
	}

	s.logger.Infow("Change status updated successfully", "change_id", id, "status", status, "tenant_id", tenantID)
	return nil
}

// isValidChangeStatusTransition 检查变更状态转换是否合法
// Change状态转换规则:
// draft -> submitted, cancelled
// submitted -> approved, rejected, cancelled
// approved -> scheduled, cancelled
// rejected -> (不允许转换到其他状态)
// scheduled -> in_progress, cancelled
// in_progress -> completed, failed, cancelled
// completed -> (不允许转换到其他状态)
// failed -> scheduled, cancelled
// cancelled -> (不允许转换到其他状态)
func isValidChangeStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusCancelled},
		common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
		common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusCancelled},
		common.ChangeStatusRejected:   {}, // 被拒绝后不允许转换
		common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
		common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, common.ChangeStatusCancelled},
		common.ChangeStatusCompleted:  {}, // 已完成不允许转换
		common.ChangeStatusFailed:     {common.ChangeStatusScheduled, common.ChangeStatusCancelled},
		common.ChangeStatusCancelled:  {}, // 已取消不允许转换
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		// 未知状态，允许转换（保守策略）
		return true
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

// triggerWorkflowForChange 为变更触发工作流
func (s *ChangeService) triggerWorkflowForChange(ctx context.Context, changeID int, tenantID int) error {
	// 获取变更信息
	ch, err := s.client.Change.Query().
		Where(
			change.ID(changeID),
			change.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get change: %w", err)
	}

	// 构建流程变量
	variables := map[string]interface{}{
		"change_id":    ch.ID,
		"title":        ch.Title,
		"description":  ch.Description,
		"type":         ch.Type,
		"priority":     ch.Priority,
		"status":       ch.Status,
		"impact_scope": ch.ImpactScope,
		"risk_level":   ch.RiskLevel,
		"created_by":   ch.CreatedBy,
		"assignee_id":  ch.AssigneeID,
	}

	// 根据变更类型选择不同的流程
	processKey := "change_normal_flow"
	if ch.Type == "emergency" {
		processKey = "change_emergency_flow"
	}

	// 触发流程
	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeChange,
		BusinessID:           changeID,
		ProcessDefinitionKey: processKey,
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", ch.CreatedBy),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerService.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow(
		"Workflow triggered for change",
		"change_id", changeID,
		"process_instance_id", resp.ProcessInstanceID,
		"process_key", processKey,
	)

	return nil
}

// GetWorkflowStatus 获取变更关联的流程状态
func (s *ChangeService) GetWorkflowStatus(ctx context.Context, changeID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	businessKey := fmt.Sprintf("change:%d", changeID)

	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("未找到变更关联的流程实例")
		}
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	processDefName := ""
	if processInstance.Edges.Definition != nil {
		processDefName = processInstance.Edges.Definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     processInstance.ID,
		ProcessDefinitionKey:  processInstance.ProcessDefinitionKey,
		ProcessDefinitionName: processDefName,
		BusinessKey:           processInstance.BusinessKey,
		Status:                s.mapProcessStatus(processInstance.Status),
		CurrentActivityID:     processInstance.CurrentActivityID,
		CurrentActivityName:   processInstance.CurrentActivityName,
		StartTime:             processInstance.StartTime,
		EndTime:               &processInstance.EndTime,
	}, nil
}

// mapProcessStatus 映射流程状态
func (s *ChangeService) mapProcessStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}

// GetCalendarView 获取变更日历视图数据
func (s *ChangeService) GetCalendarView(ctx context.Context, tenantID int, startDate, endDate string, status string) (*dto.ChangeCalendarResponse, error) {
	// 解析日期
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}
	// 设置结束日期为当天结束
	end = end.Add(24*time.Hour - time.Second)

	// 构建查询: 计划时间在日期范围内
	query := s.client.Change.Query().
		Where(
			change.TenantID(tenantID),
			change.Or(
				change.And(
					change.PlannedStartDateGTE(start),
					change.PlannedStartDateLTE(end),
				),
				change.And(
					change.PlannedEndDateGTE(start),
					change.PlannedEndDateLTE(end),
				),
			),
		)

	// 状态过滤
	if status != "" {
		query = query.Where(change.Status(status))
	}

	// 获取变更列表
	changes, err := query.Order(ent.Asc(change.FieldPlannedStartDate)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	// 构建响应
	items := make([]dto.ChangeCalendarItem, 0, len(changes))
	for _, c := range changes {
		item := dto.ChangeCalendarItem{
			ID:           c.ID,
			Title:        c.Title,
			ChangeNumber: fmt.Sprintf("C-%d", c.ID),
			Status:       c.Status,
			RiskLevel:    c.RiskLevel,
			Category:     string(c.Type),
			PlannedStart: c.PlannedStartDate,
			PlannedEnd:   c.PlannedEndDate,
		}

		// 获取处理人姓名
		if c.AssigneeID > 0 {
			user, err := s.client.User.Get(ctx, c.AssigneeID)
			if err == nil {
				item.AssigneeName = user.Name
			}
		}

		items = append(items, item)
	}

	return &dto.ChangeCalendarResponse{
		Items: items,
		Total: len(items),
	}, nil
}
