package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"

	"go.uber.org/zap"
)

// ChangeService 变更管理服务
type ChangeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewChangeService 创建变更管理服务
func NewChangeService(client *ent.Client, logger *zap.SugaredLogger) *ChangeService {
	return &ChangeService{
		client: client,
		logger: logger,
	}
}

// CreateChange 创建变更
func (s *ChangeService) CreateChange(ctx context.Context, req *dto.CreateChangeRequest, createdBy, tenantID int) (*dto.ChangeResponse, error) {
	// 创建变更记录
	changeEntity, err := s.client.Change.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetJustification(req.Justification).
		SetType(string(req.Type)).
		SetStatus(string(dto.ChangeStatusDraft)).
		SetPriority(string(req.Priority)).
		SetImpactScope(string(req.ImpactScope)).
		SetRiskLevel(string(req.RiskLevel)).
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

	// 设置相关工单
	if len(req.RelatedTickets) > 0 {
		_, err = changeEntity.Update().SetRelatedTickets(req.RelatedTickets).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set related tickets", "error", err, "change_id", changeEntity.ID)
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

	// 待审批
	pending, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusPending))).
		Count(ctx)
	if err == nil {
		stats.Pending = pending
	}

	// 已批准
	approved, err := s.client.Change.Query().
		Where(change.TenantID(tenantID), change.Status(string(dto.ChangeStatusApproved))).
		Count(ctx)
	if err == nil {
		stats.Approved = approved
	}

	// 实施中
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

	return stats, nil
}

// UpdateChangeStatus 更新变更状态
func (s *ChangeService) UpdateChangeStatus(ctx context.Context, id int, status dto.ChangeStatus, tenantID int) error {
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

	// 更新状态
	_, err = s.client.Change.UpdateOneID(id).SetStatus(string(status)).Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update change status", "error", err, "change_id", id, "status", status, "tenant_id", tenantID)
		return fmt.Errorf("failed to update change status: %w", err)
	}

	s.logger.Infow("Change status updated successfully", "change_id", id, "status", status, "tenant_id", tenantID)
	return nil
}
