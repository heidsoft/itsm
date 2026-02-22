package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/release"

	"go.uber.org/zap"
)

// ReleaseService 发布管理服务
type ReleaseService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewReleaseService 创建发布管理服务
func NewReleaseService(client *ent.Client, logger *zap.SugaredLogger) *ReleaseService {
	return &ReleaseService{
		client: client,
		logger: logger,
	}
}

// CreateRelease 创建发布
func (s *ReleaseService) CreateRelease(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error) {
	releaseEntity, err := s.client.Release.Create().
		SetReleaseNumber(req.ReleaseNumber).
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetType(req.Type).
		SetStatus(string(dto.ReleaseStatusDraft)).
		SetSeverity(req.Severity).
		SetEnvironment(req.Environment).
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		SetNillableChangeID(req.ChangeID).
		SetNillableOwnerID(req.OwnerID).
		SetNillablePlannedReleaseDate(req.PlannedReleaseDate).
		SetNillablePlannedStartDate(req.PlannedStartDate).
		SetNillablePlannedEndDate(req.PlannedEndDate).
		SetReleaseNotes(req.ReleaseNotes).
		SetRollbackProcedure(req.RollbackProcedure).
		SetValidationCriteria(req.ValidationCriteria).
		SetIsEmergency(req.IsEmergency).
		SetRequiresApproval(req.RequiresApproval).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to create release", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	// 设置关联字段
	if len(req.AffectedSystems) > 0 {
		releaseEntity.Update().SetAffectedSystems(req.AffectedSystems)
	}
	if len(req.AffectedComponents) > 0 {
		releaseEntity.Update().SetAffectedComponents(req.AffectedComponents)
	}
	if len(req.DeploymentSteps) > 0 {
		releaseEntity.Update().SetDeploymentSteps(req.DeploymentSteps)
	}
	if len(req.Tags) > 0 {
		releaseEntity.Update().SetTags(req.Tags)
	}

	// 获取创建人信息
	creator, _ := s.client.User.Get(ctx, createdBy)
	creatorName := ""
	if creator != nil {
		creatorName = creator.Name
	}

	response := dto.ToReleaseResponse(releaseEntity)
	response.CreatedByName = creatorName

	s.logger.Infow("Release created successfully", "release_id", releaseEntity.ID, "tenant_id", tenantID)
	return response, nil
}

// GetReleaseByID 根据ID获取发布
func (s *ReleaseService) GetReleaseByID(ctx context.Context, id, tenantID int) (*dto.ReleaseResponse, error) {
	releaseEntity, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get release", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	response := dto.ToReleaseResponse(releaseEntity)

	// 获取创建人信息
	if releaseEntity.CreatedBy > 0 {
		creator, _ := s.client.User.Get(ctx, releaseEntity.CreatedBy)
		if creator != nil {
			response.CreatedByName = creator.Name
		}
	}

	// 获取负责人信息
	if releaseEntity.OwnerID != nil && *releaseEntity.OwnerID > 0 {
		owner, _ := s.client.User.Get(ctx, *releaseEntity.OwnerID)
		if owner != nil {
			response.OwnerName = &owner.Name
		}
	}

	return response, nil
}

// ListReleases 获取发布列表
func (s *ReleaseService) ListReleases(ctx context.Context, tenantID int, page, pageSize int, status, releaseType string) (*dto.ReleaseListResponse, error) {
	query := s.client.Release.Query().Where(release.TenantIDEQ(tenantID))

	if status != "" {
		query = query.Where(release.StatusEQ(status))
	}
	if releaseType != "" {
		query = query.Where(release.TypeEQ(releaseType))
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count releases", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count releases: %w", err)
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	releaseEntities, err := query.Order(ent.Desc(release.FieldCreatedAt)).All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list releases", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	releases := dto.ToReleaseResponseList(releaseEntities)

	return &dto.ReleaseListResponse{
		Total:    total,
		Releases: releases,
	}, nil
}

// UpdateRelease 更新发布
func (s *ReleaseService) UpdateRelease(ctx context.Context, id, tenantID int, req *dto.UpdateReleaseRequest) (*dto.ReleaseResponse, error) {
	releaseEntity, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get release", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	update := releaseEntity.Update()

	if req.Title != nil {
		update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Type != nil {
		update.SetType(*req.Type)
	}
	if req.Environment != nil {
		update.SetEnvironment(*req.Environment)
	}
	if req.Severity != nil {
		update.SetSeverity(*req.Severity)
	}
	if req.ChangeID != nil {
		update.SetChangeID(*req.ChangeID)
	}
	if req.OwnerID != nil {
		update.SetOwnerID(*req.OwnerID)
	}
	if req.PlannedReleaseDate != nil {
		update.SetPlannedReleaseDate(*req.PlannedReleaseDate)
	}
	if req.PlannedStartDate != nil {
		update.SetPlannedStartDate(*req.PlannedStartDate)
	}
	if req.PlannedEndDate != nil {
		update.SetPlannedEndDate(*req.PlannedEndDate)
	}
	if req.ActualReleaseDate != nil {
		update.SetActualReleaseDate(*req.ActualReleaseDate)
	}
	if req.ReleaseNotes != nil {
		update.SetReleaseNotes(*req.ReleaseNotes)
	}
	if req.RollbackProcedure != nil {
		update.SetRollbackProcedure(*req.RollbackProcedure)
	}
	if req.ValidationCriteria != nil {
		update.SetValidationCriteria(*req.ValidationCriteria)
	}
	if req.IsEmergency != nil {
		update.SetIsEmergency(*req.IsEmergency)
	}
	if req.RequiresApproval != nil {
		update.SetRequiresApproval(*req.RequiresApproval)
	}

	if len(req.AffectedSystems) > 0 {
		update.SetAffectedSystems(req.AffectedSystems)
	}
	if len(req.AffectedComponents) > 0 {
		update.SetAffectedComponents(req.AffectedComponents)
	}
	if len(req.DeploymentSteps) > 0 {
		update.SetDeploymentSteps(req.DeploymentSteps)
	}
	if len(req.Tags) > 0 {
		update.SetTags(req.Tags)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to update release: %w", err)
	}

	return dto.ToReleaseResponse(updated), nil
}

// UpdateReleaseStatus 更新发布状态
func (s *ReleaseService) UpdateReleaseStatus(ctx context.Context, id, tenantID int, status string) (*dto.ReleaseResponse, error) {
	releaseEntity, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get release", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	update := releaseEntity.Update().SetStatus(status)

	// 如果状态是已完成，设置实际发布日期
	if status == string(dto.ReleaseStatusCompleted) {
		now := time.Now()
		update.SetActualReleaseDate(now)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release status", "error", err, "release_id", id, "status", status)
		return nil, fmt.Errorf("failed to update release status: %w", err)
	}

	s.logger.Infow("Release status updated", "release_id", id, "status", status)
	return dto.ToReleaseResponse(updated), nil
}

// DeleteRelease 删除发布
func (s *ReleaseService) DeleteRelease(ctx context.Context, id, tenantID int) error {
	_, err := s.client.Release.Delete().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		Exec(ctx)

	if err != nil {
		s.logger.Errorw("Failed to delete release", "error", err, "release_id", id)
		return fmt.Errorf("failed to delete release: %w", err)
	}

	s.logger.Infow("Release deleted", "release_id", id)
	return nil
}

// GetReleaseStats 获取发布统计
func (s *ReleaseService) GetReleaseStats(ctx context.Context, tenantID int) (*dto.ReleaseStatsResponse, error) {
	stats := &dto.ReleaseStatsResponse{}

	total, err := s.client.Release.Query().Where(release.TenantIDEQ(tenantID)).Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count releases", "error", err)
		return nil, err
	}
	stats.Total = total

	// 统计各状态数量
	draft, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusDraft))).Count(ctx)
	scheduled, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusScheduled))).Count(ctx)
	inProgress, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusInProgress))).Count(ctx)
	completed, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusCompleted))).Count(ctx)
	cancelled, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusCancelled))).Count(ctx)
	failed, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusFailed))).Count(ctx)
	rolledBack, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(string(dto.ReleaseStatusRolledBack))).Count(ctx)

	stats.Draft = draft
	stats.Scheduled = scheduled
	stats.InProgress = inProgress
	stats.Completed = completed
	stats.Cancelled = cancelled
	stats.Failed = failed
	stats.RolledBack = rolledBack

	return stats, nil
}
