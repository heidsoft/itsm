package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"

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
	if err := s.validateCreateChange(ctx, req, createdBy, tenantID); err != nil {
		return nil, err
	}
	affectedCIs := uniqueNonEmptyStrings(req.AffectedCIs)
	relatedTickets := uniqueNonEmptyStrings(req.RelatedTickets)
	initialStatus := dto.ChangeStatusDraft
	if req.Type == string(dto.ChangeTypeStandard) && req.RiskLevel == string(dto.ChangeRiskLow) &&
		strings.TrimSpace(req.ImplementationPlan) != "" && strings.TrimSpace(req.RollbackPlan) != "" {
		initialStatus = dto.ChangeStatusApproved
	}

	changeEntity, err := s.client.Change.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetJustification(req.Justification).
		SetType(req.Type).
		SetStatus(string(initialStatus)).
		SetPriority(req.Priority).
		SetImpactScope(req.ImpactScope).
		SetRiskLevel(req.RiskLevel).
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		SetImplementationPlan(req.ImplementationPlan).
		SetRollbackPlan(req.RollbackPlan).
		SetNillablePlannedStartDate(req.PlannedStartDate).
		SetNillablePlannedEndDate(req.PlannedEndDate).
		SetAffectedCis(affectedCIs).
		SetRelatedTickets(relatedTickets).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create change", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create change: %w", err)
	}

	// 构建响应（先填充实体数据，状态后续可能更新）
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
		TenantID:           changeEntity.TenantID,
		PlannedStartDate:   optionalTime(changeEntity.PlannedStartDate),
		PlannedEndDate:     optionalTime(changeEntity.PlannedEndDate),
		ImplementationPlan: changeEntity.ImplementationPlan,
		RollbackPlan:       changeEntity.RollbackPlan,
		CreatedAt:          changeEntity.CreatedAt,
		UpdatedAt:          changeEntity.UpdatedAt,
	}

	if req.Type == string(dto.ChangeTypeEmergency) {
		// 紧急变更：记录特殊标记，后续走ECAB快速审批流程
		s.logger.Infow("Emergency change created per ITIL standard - requires ECAB approval", "change_id", changeEntity.ID, "tenant_id", tenantID)
	}

	// 获取创建人信息
	creator, err := s.client.User.Query().Where(user.IDEQ(createdBy), user.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		s.logger.Warnw("Failed to get creator info", "error", err, "user_id", createdBy)
	}

	if creator != nil {
		response.CreatedByName = creator.Name
	}

	// 设置受影响的配置项
	if len(changeEntity.AffectedCis) > 0 {
		response.AffectedCIs = changeEntity.AffectedCis
	}

	// 设置相关工单
	if len(changeEntity.RelatedTickets) > 0 {
		response.RelatedTickets = changeEntity.RelatedTickets
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

func (s *ChangeService) validateCreateChange(ctx context.Context, req *dto.CreateChangeRequest, createdBy, tenantID int) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("变更标题不能为空")
	}
	if !isValidChangeType(req.Type) {
		return fmt.Errorf("无效的变更类型: %s", req.Type)
	}
	if !isValidChangePriority(req.Priority) || !isValidChangeImpact(req.ImpactScope) || !isValidChangeRisk(req.RiskLevel) {
		return fmt.Errorf("变更优先级、影响范围或风险等级无效")
	}
	if req.PlannedStartDate != nil && req.PlannedEndDate != nil && !req.PlannedStartDate.Before(*req.PlannedEndDate) {
		return fmt.Errorf("计划结束时间必须晚于计划开始时间")
	}
	creatorExists, err := s.client.User.Query().
		Where(user.IDEQ(createdBy), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("验证变更创建人失败: %w", err)
	}
	if !creatorExists {
		return fmt.Errorf("变更创建人不存在或不可用")
	}
	if err := s.validateChangeReferences(ctx, req.AffectedCIs, req.RelatedTickets, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *ChangeService) validateChangeReferences(ctx context.Context, affectedCIs, relatedTickets []string, tenantID int) error {
	ciNames := uniqueNonEmptyStrings(affectedCIs)
	if len(ciNames) > 0 {
		items, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.TenantIDEQ(tenantID), configurationitem.NameIn(ciNames...)).
			Select(configurationitem.FieldName).All(ctx)
		if err != nil {
			return fmt.Errorf("验证受影响配置项失败: %w", err)
		}
		found := make(map[string]struct{}, len(items))
		for _, item := range items {
			found[item.Name] = struct{}{}
		}
		for _, name := range ciNames {
			if _, ok := found[name]; !ok {
				return fmt.Errorf("受影响配置项不存在: %s", name)
			}
		}
	}
	ticketNumbers := uniqueNonEmptyStrings(relatedTickets)
	if len(ticketNumbers) > 0 {
		count, err := s.client.Ticket.Query().
			Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil(), ticket.TicketNumberIn(ticketNumbers...)).Count(ctx)
		if err != nil {
			return fmt.Errorf("验证相关工单失败: %w", err)
		}
		if count != len(ticketNumbers) {
			return fmt.Errorf("相关工单不存在或不属于当前租户")
		}
	}
	return nil
}

func isValidChangeType(value string) bool {
	return value == string(dto.ChangeTypeNormal) || value == string(dto.ChangeTypeStandard) || value == string(dto.ChangeTypeEmergency)
}

func isValidChangePriority(value string) bool {
	return value == string(dto.ChangePriorityLow) || value == string(dto.ChangePriorityMedium) || value == string(dto.ChangePriorityHigh) || value == string(dto.ChangePriorityCritical)
}

func isValidChangeImpact(value string) bool {
	return value == string(dto.ChangeImpactLow) || value == string(dto.ChangeImpactMedium) || value == string(dto.ChangeImpactHigh)
}

func isValidChangeRisk(value string) bool {
	return value == string(dto.ChangeRiskLow) || value == string(dto.ChangeRiskMedium) || value == string(dto.ChangeRiskHigh)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

// API 使用 pending 表达待审批，持久化层和 BPMN 流程统一使用 submitted。
func persistedChangeStatus(status string) string {
	if status == string(dto.ChangeStatusPending) {
		return common.ChangeStatusSubmitted
	}
	return status
}

func apiChangeStatus(status string) dto.ChangeStatus {
	if status == common.ChangeStatusSubmitted {
		return dto.ChangeStatusPending
	}
	return dto.ChangeStatus(status)
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
	creator, err := s.client.User.Query().Where(user.IDEQ(changeEntity.CreatedBy), user.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		s.logger.Warnw("Failed to get creator info", "error", err, "user_id", changeEntity.CreatedBy)
	}

	// 获取处理人信息
	var assigneeName *string
	if changeEntity.AssigneeID > 0 {
		assignee, err := s.client.User.Query().Where(user.IDEQ(changeEntity.AssigneeID), user.TenantIDEQ(tenantID)).Only(ctx)
		if err == nil {
			assigneeName = &assignee.Name
		}
	}

	// 构建响应
	createdByName := ""
	if creator != nil {
		createdByName = creator.Name
	}
	response := &dto.ChangeResponse{
		ID:                 changeEntity.ID,
		Title:              changeEntity.Title,
		Description:        changeEntity.Description,
		Justification:      changeEntity.Justification,
		Type:               dto.ChangeType(changeEntity.Type),
		Status:             apiChangeStatus(changeEntity.Status),
		Priority:           dto.ChangePriority(changeEntity.Priority),
		ImpactScope:        dto.ChangeImpact(changeEntity.ImpactScope),
		RiskLevel:          dto.ChangeRisk(changeEntity.RiskLevel),
		AssigneeID:         optionalInt(changeEntity.AssigneeID),
		AssigneeName:       assigneeName,
		CreatedBy:          changeEntity.CreatedBy,
		CreatedByName:      createdByName,
		TenantID:           changeEntity.TenantID,
		PlannedStartDate:   optionalTime(changeEntity.PlannedStartDate),
		PlannedEndDate:     optionalTime(changeEntity.PlannedEndDate),
		ActualStartDate:    optionalTime(changeEntity.ActualStartDate),
		ActualEndDate:      optionalTime(changeEntity.ActualEndDate),
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
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	query := s.client.Change.Query().Where(change.TenantID(tenantID))

	// 状态筛选
	if status != "" && status != "全部" {
		query = query.Where(change.Status(persistedChangeStatus(status)))
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
		creator, err := s.client.User.Query().Where(user.IDEQ(changeEntity.CreatedBy), user.TenantIDEQ(tenantID)).Only(ctx)
		if err != nil {
			s.logger.Warnw("Failed to get creator info", "error", err, "user_id", changeEntity.CreatedBy)
		}

		// 获取处理人信息
		var assigneeName *string
		if changeEntity.AssigneeID > 0 {
			assignee, err := s.client.User.Query().Where(user.IDEQ(changeEntity.AssigneeID), user.TenantIDEQ(tenantID)).Only(ctx)
			if err == nil {
				assigneeName = &assignee.Name
			}
		}

		createdByName := ""
		if creator != nil {
			createdByName = creator.Name
		}
		response := dto.ChangeResponse{
			ID:                 changeEntity.ID,
			Title:              changeEntity.Title,
			Description:        changeEntity.Description,
			Justification:      changeEntity.Justification,
			Type:               dto.ChangeType(changeEntity.Type),
			Status:             apiChangeStatus(changeEntity.Status),
			Priority:           dto.ChangePriority(changeEntity.Priority),
			ImpactScope:        dto.ChangeImpact(changeEntity.ImpactScope),
			RiskLevel:          dto.ChangeRisk(changeEntity.RiskLevel),
			AssigneeID:         optionalInt(changeEntity.AssigneeID),
			AssigneeName:       assigneeName,
			CreatedBy:          changeEntity.CreatedBy,
			CreatedByName:      createdByName,
			TenantID:           changeEntity.TenantID,
			PlannedStartDate:   optionalTime(changeEntity.PlannedStartDate),
			PlannedEndDate:     optionalTime(changeEntity.PlannedEndDate),
			ActualStartDate:    optionalTime(changeEntity.ActualStartDate),
			ActualEndDate:      optionalTime(changeEntity.ActualEndDate),
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

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &dto.ChangeListResponse{
		Total:      total,
		Changes:    changeResponses,
		PageSize:   pageSize,
		TotalPages: totalPages,
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
	if changeEntity.Status != string(dto.ChangeStatusDraft) {
		return nil, fmt.Errorf("只有草稿状态的变更可以修改")
	}
	if req.Type != nil && !isValidChangeType(string(*req.Type)) {
		return nil, fmt.Errorf("无效的变更类型: %s", *req.Type)
	}
	if req.Priority != nil && !isValidChangePriority(string(*req.Priority)) {
		return nil, fmt.Errorf("无效的变更优先级: %s", *req.Priority)
	}
	if req.ImpactScope != nil && !isValidChangeImpact(string(*req.ImpactScope)) {
		return nil, fmt.Errorf("无效的影响范围: %s", *req.ImpactScope)
	}
	if req.RiskLevel != nil && !isValidChangeRisk(string(*req.RiskLevel)) {
		return nil, fmt.Errorf("无效的风险等级: %s", *req.RiskLevel)
	}
	plannedStart, plannedEnd := changeEntity.PlannedStartDate, changeEntity.PlannedEndDate
	if req.PlannedStartDate != nil {
		plannedStart = *req.PlannedStartDate
	}
	if req.PlannedEndDate != nil {
		plannedEnd = *req.PlannedEndDate
	}
	if !plannedStart.IsZero() && !plannedEnd.IsZero() && !plannedStart.Before(plannedEnd) {
		return nil, fmt.Errorf("计划结束时间必须晚于计划开始时间")
	}
	if err := s.validateChangeReferences(ctx, req.AffectedCIs, req.RelatedTickets, tenantID); err != nil {
		return nil, err
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
		update.SetAffectedCis(uniqueNonEmptyStrings(req.AffectedCIs))
	}

	// 更新相关工单
	if req.RelatedTickets != nil {
		update.SetRelatedTickets(uniqueNonEmptyStrings(req.RelatedTickets))
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
	changeEntity, err := s.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("change not found")
		}
		return fmt.Errorf("failed to get change: %w", err)
	}
	if changeEntity.Status != string(dto.ChangeStatusDraft) && changeEntity.Status != string(dto.ChangeStatusCancelled) {
		return fmt.Errorf("只有草稿或已取消的变更可以删除")
	}

	deleted, err := s.client.Change.Delete().
		Where(change.IDEQ(id), change.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete change", "error", err, "change_id", id, "tenant_id", tenantID)
		return fmt.Errorf("failed to delete change: %w", err)
	}
	if deleted != 1 {
		return fmt.Errorf("change not found")
	}

	s.logger.Infow("Change deleted successfully", "change_id", id, "tenant_id", tenantID)
	return nil
}

// GetChangeStats 获取变更统计
func (s *ChangeService) GetChangeStats(ctx context.Context, tenantID int) (*dto.ChangeStatsResponse, error) {
	rows, err := s.client.Change.Query().Where(change.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query change statistics: %w", err)
	}
	stats := &dto.ChangeStatsResponse{Total: len(rows)}
	for _, item := range rows {
		switch item.Status {
		case string(dto.ChangeStatusDraft), string(dto.ChangeStatusPending), common.ChangeStatusSubmitted:
			stats.Pending++
		case string(dto.ChangeStatusApproved):
			stats.Approved++
		case string(dto.ChangeStatusScheduled):
			stats.Scheduled++
		case string(dto.ChangeStatusInProgress):
			stats.InProgress++
		case string(dto.ChangeStatusCompleted):
			stats.Completed++
		case string(dto.ChangeStatusFailed):
			stats.Failed++
		case string(dto.ChangeStatusRolledBack):
			stats.RolledBack++
		case string(dto.ChangeStatusRejected):
			stats.Rejected++
		case string(dto.ChangeStatusCancelled):
			stats.Cancelled++
		default:
			return nil, fmt.Errorf("unknown change status %q for change %d", item.Status, item.ID)
		}
	}
	return stats, nil
}

// UpdateChangeStatus 更新变更状态
func (s *ChangeService) UpdateChangeStatus(ctx context.Context, id int, status dto.ChangeStatus, tenantID int) error {
	persistedStatus := persistedChangeStatus(string(status))
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
	if !isValidChangeStatusTransition(changeEntity.Status, persistedStatus, changeEntity.Type) {
		return fmt.Errorf("invalid status transition from '%s' to '%s'", changeEntity.Status, status)
	}

	// 使用租户过滤进行更新，防止跨租户更新
	update := s.client.Change.Update().
		Where(
			change.ID(id),
			change.TenantID(tenantID),
		).
		SetStatus(persistedStatus)
	now := time.Now()
	if status == dto.ChangeStatusInProgress && changeEntity.ActualStartDate.IsZero() {
		update.SetActualStartDate(now)
	}
	if status == dto.ChangeStatusCompleted || status == dto.ChangeStatusFailed || status == dto.ChangeStatusRolledBack {
		update.SetActualEndDate(now)
	}
	result, err := update.Save(ctx)
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
// isValidChangeStatusTransition 检查变更状态转换是否合法
// 根据ITIL标准，不同类型的变更有不同的状态转换规则
func isValidChangeStatusTransition(currentStatus, newStatus, changeType string) bool {
	// 基础转换规则（适用于所有变更类型）
	baseTransitions := map[string][]string{
		common.ChangeStatusRejected:        {}, // 被拒绝后不允许转换
		common.ChangeStatusCompleted:       {}, // 已完成不允许转换
		common.ChangeStatusCancelled:       {}, // 已取消不允许转换
		string(dto.ChangeStatusRolledBack): {},
	}

	// 不同变更类型的特殊转换规则
	var typeSpecificTransitions map[string][]string
	switch changeType {
	case string(dto.ChangeTypeStandard):
		// 标准变更：预授权，可以跳过审批步骤
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	case string(dto.ChangeTypeEmergency):
		// 紧急变更：可以跳过多个步骤，快速实施
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusApproved, common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	default: // 普通变更：严格的ITIL流程
		typeSpecificTransitions = map[string][]string{
			common.ChangeStatusDraft:      {common.ChangeStatusSubmitted, common.ChangeStatusCancelled},
			common.ChangeStatusSubmitted:  {common.ChangeStatusApproved, common.ChangeStatusRejected, common.ChangeStatusCancelled},
			common.ChangeStatusApproved:   {common.ChangeStatusScheduled, common.ChangeStatusCancelled},
			common.ChangeStatusScheduled:  {common.ChangeStatusInProgress, common.ChangeStatusCancelled},
			common.ChangeStatusInProgress: {common.ChangeStatusCompleted, common.ChangeStatusFailed, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
			common.ChangeStatusFailed:     {common.ChangeStatusScheduled, string(dto.ChangeStatusRolledBack), common.ChangeStatusCancelled},
		}
	}

	// 合并基础规则和类型特定规则
	validTransitions := make(map[string][]string)
	for k, v := range baseTransitions {
		validTransitions[k] = v
	}
	for k, v := range typeSpecificTransitions {
		validTransitions[k] = v
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		// 未知状态必须失败关闭，避免绕过变更生命周期约束。
		return false
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
			change.PlannedStartDateLTE(end),
			change.PlannedEndDateGTE(start),
		)

	// 状态过滤
	if status != "" {
		query = query.Where(change.Status(persistedChangeStatus(status)))
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
			Status:       string(apiChangeStatus(c.Status)),
			RiskLevel:    c.RiskLevel,
			Category:     string(c.Type),
			PlannedStart: c.PlannedStartDate,
			PlannedEnd:   c.PlannedEndDate,
		}

		// 获取处理人姓名
		if c.AssigneeID > 0 {
			assignee, err := s.client.User.Query().Where(user.IDEQ(c.AssigneeID), user.TenantIDEQ(tenantID)).Only(ctx)
			if err == nil {
				item.AssigneeName = assignee.Name
			}
		}

		items = append(items, item)
	}

	return &dto.ChangeCalendarResponse{
		Items: items,
		Total: len(items),
	}, nil
}
