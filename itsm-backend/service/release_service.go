package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/release"
	"itsm-backend/ent/user"
	"itsm-backend/handlers/common/datascope"

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
// P1 修复：
//  1. release_number 由服务端生成，不信任客户端传入值（避免越权/冲突编号）。
//  2. 两段写（先 Create 再对 JSON 字段二次 Save）包进单个事务，失败回滚，不留半成品。
func (s *ReleaseService) CreateRelease(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error) {
	releaseNumber := s.generateReleaseNumber(tenantID)

	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Errorw("Failed to start transaction", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create release: %w", err)
	}
	defer tx.Rollback()

	releaseEntity, err := tx.Release.Create().
		SetReleaseNumber(releaseNumber).
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

	// 设置关联字段（Update() 返回构造器，必须 Save(ctx) 才落库）
	updater := releaseEntity.Update()
	needsUpdate := false
	if len(req.AffectedSystems) > 0 {
		updater = updater.SetAffectedSystems(req.AffectedSystems)
		needsUpdate = true
	}
	if len(req.AffectedComponents) > 0 {
		updater = updater.SetAffectedComponents(req.AffectedComponents)
		needsUpdate = true
	}
	if len(req.DeploymentSteps) > 0 {
		updater = updater.SetDeploymentSteps(req.DeploymentSteps)
		needsUpdate = true
	}
	if len(req.Tags) > 0 {
		updater = updater.SetTags(req.Tags)
		needsUpdate = true
	}
	if needsUpdate {
		updated, err := updater.Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to set release associations", "error", err, "release_id", releaseEntity.ID)
			return nil, fmt.Errorf("failed to set release associations: %w", err)
		}
		releaseEntity = updated
	}

	if err := tx.Commit(); err != nil {
		s.logger.Errorw("Failed to commit release transaction", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	// 获取创建人信息
	creator, _ := s.client.User.Get(ctx, createdBy)
	creatorName := ""
	if creator != nil {
		creatorName = creator.Name
	}

	response := dto.ToReleaseResponse(releaseEntity)
	response.CreatedByName = creatorName

	s.logger.Infow("Release created successfully", "release_id", releaseEntity.ID, "tenant_id", tenantID, "release_number", releaseNumber)
	return response, nil
}

// generateReleaseNumber 服务端生成发布编号（REL-YYYYMMDD-xxxxxx），
// 不依赖客户端传入，避免越权/编号冲突。
func (s *ReleaseService) generateReleaseNumber(tenantID int) string {
	randPart := fmt.Sprintf("%06d", rand.Intn(1000000))
	return fmt.Sprintf("REL-%s-%s", time.Now().Format("20060102"), randPart)
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

// ListReleases 获取发布列表。推广 ticket 的 DataScope 行级权限：
// 管理角色（super_admin/admin/manager/sysadmin）可见全租户，其余角色仅可见
// 本人创建或分配给自己的发布单。currentUserID/currentRole 由 controller 从
// 鉴权中间件注入的 user_id/role 取得。
func (s *ReleaseService) ListReleases(ctx context.Context, tenantID int, page, pageSize int, status, releaseType string, currentUserID int, currentRole string) (*dto.ReleaseListResponse, error) {
	query := s.client.Release.Query().Where(release.TenantIDEQ(tenantID))

	if status != "" {
		query = query.Where(release.StatusEQ(status))
	}
	if releaseType != "" {
		query = query.Where(release.TypeEQ(releaseType))
	}

	// 行级数据权限（推广自 ticket DataScope 模式）：
	// OwnedOrAssigned 时强制追加 Or(CreatedByEQ(uid), OwnerIDEQ(uid))，
	// 使普通用户只能看到自己创建或分配给自己的发布单。
	// CurrentUserID<=0 时 fail-closed，返回空集而非全量。
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	if dataScope == datascope.DataScopeOwnedOrAssigned {
		if currentUserID <= 0 {
			query = query.Where(release.IDEQ(-1))
		} else {
			query = query.Where(release.Or(
				release.CreatedByEQ(currentUserID),
				release.OwnerIDEQ(currentUserID),
			))
		}
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

// ErrInvalidReleaseTransition 表示发布状态机白名单拒绝了本次状态流转。
// 属于调用方输入错误（应映射为 400），而非服务端内部错误（500）。
var ErrInvalidReleaseTransition = errors.New("非法的发布状态转换")

// UpdateReleaseStatus 更新发布状态
// C-1 修复：新增 isValidReleaseStatusTransition 白名单校验，防止审批被绕过：
//   - draft → cancelled（draft → scheduled 已移除，见 D-5：必须走 /approve 且需 release:approve）
//   - scheduled → in-progress / cancelled
//   - in-progress → completed / failed / rolled_back / cancelled
//   - completed / cancelled / rolled_back / failed 为终态（不可被复活）
func (s *ReleaseService) UpdateReleaseStatus(ctx context.Context, id, tenantID int, status string) (*dto.ReleaseResponse, error) {
	status = func() string { s1 := status; return s1 }()
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

	// 1. 状态机白名单校验
	if !isValidReleaseStatusTransition(releaseEntity.Status, status) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidReleaseTransition, releaseEntity.Status, status)
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

// isValidReleaseStatusTransition 发布状态转换白名单校验
func isValidReleaseStatusTransition(current, newStatus string) bool {
	if current == newStatus {
		// 幂等：同一状态不报错
		return true
	}
	baseTransitions := map[string]map[string]struct{}{
		string(dto.ReleaseStatusDraft): {
			// D-5 修复：draft→scheduled 是审批等效动作，必须经 /approve（release:approve），
			// 不允许持 release:write 的用户经 /status 路由直接排期，否则绕过审批门禁。
			string(dto.ReleaseStatusCancelled): {},
		},
		string(dto.ReleaseStatusScheduled): {
			string(dto.ReleaseStatusInProgress): {},
			string(dto.ReleaseStatusCancelled):  {},
		},
		string(dto.ReleaseStatusInProgress): {
			string(dto.ReleaseStatusCompleted):  {},
			string(dto.ReleaseStatusFailed):     {},
			string(dto.ReleaseStatusRolledBack): {},
			string(dto.ReleaseStatusCancelled):  {},
		},
		string(dto.ReleaseStatusFailed): {
			// D-5 修复：failed→scheduled 为重新排期（审批等效），必须经 /approve（release:approve）。
			string(dto.ReleaseStatusRolledBack): {},
			string(dto.ReleaseStatusCancelled):  {},
		},
		// 终态：completed / cancelled / rolled_back 不允许再转换
		string(dto.ReleaseStatusCompleted):  {},
		string(dto.ReleaseStatusCancelled):  {},
		string(dto.ReleaseStatusRolledBack): {},
	}
	allowed, ok := baseTransitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[newStatus]
	return ok
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

	// 仅草稿态/失败态允许审批：失败态审批即「重新排期(reschedule)」，使发布历史不断裂、
	// 关联变更不丢（P1 修复：放宽此前硬要求 draft 的死锁）。
	if releaseEntity.Status != string(dto.ReleaseStatusDraft) && releaseEntity.Status != string(dto.ReleaseStatusFailed) {
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
			ctx, tenantID, actorID, string(dto.BusinessTypeRelease), id, action, comment,
		); bridgeErr != nil {
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
// P1 修复：release 在 schema 中无 deleted_at（物理删），无法软删；DELETE 应是幂等操作
// （RFC 7231 §4.3.5：连续多次 DELETE 同一资源与一次效果相同），故未匹配到（已删/
// 不存在/跨租户均视为 0 行）时不返回错误，让上游 controller 统一返 200 响应，客户端
// 不会因为“曾被删”再次发送 DELETE 而误判为 5001 内部错误。
func (s *ReleaseService) DeleteRelease(ctx context.Context, id, tenantID int) error {
	deleted, err := s.client.Release.Delete().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete release", "error", err, "release_id", id)
		return fmt.Errorf("failed to delete release: %w", err)
	}
	if deleted == 0 {
		// 幂等：未匹配 = 视为已删除，不报错。
		s.logger.Infow("Release delete matched 0 rows (idempotent, treated as success)",
			"release_id", id, "tenant_id", tenantID)
		return nil
	}

	s.logger.Infow("Release deleted", "release_id", id, "tenant_rows", deleted)
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
