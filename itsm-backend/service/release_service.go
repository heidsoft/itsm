package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/release"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// ReleaseService 发布管理服务
type ReleaseService struct {
	client         *ent.Client
	logger         *zap.SugaredLogger
	approvalBridge *BPMNApprovalBridge
}

// NewReleaseService 创建发布管理服务
func NewReleaseService(client *ent.Client, logger *zap.SugaredLogger) *ReleaseService {
	svc := &ReleaseService{
		client: client,
		logger: logger,
	}
	if client != nil {
		// P0-1：发布审批桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = NewBPMNApprovalBridge(client, logger)
	}
	return svc
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

// ApplyReleaseApproval 处理发布审批（approve/reject）：校验审批人身份后先桥接完成对应的
// BPMN 待办任务（以流程任务为权威审批来源），再更新发布状态：
//   - approve: draft → scheduled
//   - reject:  draft → cancelled
//
// 无关联运行中流程实例时回退纯业务审批；若存在待办流程任务但完成失败，
// 则中止业务审批，避免发布状态与流程状态分叉（P0-1 双轨审批收敛）。
func (s *ReleaseService) ApplyReleaseApproval(ctx context.Context, id, tenantID, actorID int, action, comment string) (*dto.ReleaseResponse, error) {
	if action != "approve" && action != "reject" {
		return nil, fmt.Errorf("无效的审批操作: %s", action)
	}

	releaseEntity, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get release for approval", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	// 仅草稿态允许审批，防止已排期/已执行的发布被重复审批
	if releaseEntity.Status != string(dto.ReleaseStatusDraft) {
		return nil, fmt.Errorf("当前发布状态不允许审批: %s", releaseEntity.Status)
	}

	// 审批人校验：必须是本租户有效用户，且创建人不能审批自己的发布
	exists, err := s.client.User.Query().
		Where(user.ID(actorID), user.TenantID(tenantID), user.Active(true)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("校验审批人失败: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("审批人不存在或已停用")
	}
	if actorID == releaseEntity.CreatedBy {
		return nil, fmt.Errorf("发布创建人不能审批自己的发布")
	}

	// P0-1：审批先桥接完成对应的 BPMN 待办任务，失败则中止（fail-closed）
	if s.approvalBridge != nil {
		if _, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTask(
			ctx, tenantID, actorID, string(dto.BusinessTypeRelease), id, action, comment); bridgeErr != nil {
			return nil, fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
		}
	}

	targetStatus := string(dto.ReleaseStatusScheduled)
	if action == "reject" {
		targetStatus = string(dto.ReleaseStatusCancelled)
	}
	updated, err := releaseEntity.Update().SetStatus(targetStatus).Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release approval status", "error", err, "release_id", id, "status", targetStatus)
		return nil, fmt.Errorf("failed to update release status: %w", err)
	}

	s.logger.Infow("Release approval applied",
		"release_id", id, "tenant_id", tenantID, "actor_id", actorID, "action", action, "status", targetStatus)
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
