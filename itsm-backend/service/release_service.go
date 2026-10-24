package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/release"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"
	"itsm-backend/internal/datascope"

	"go.uber.org/zap"
)

// ReleaseService 发布管理服务
type ReleaseService struct {
	client         *ent.Client
	logger         *zap.SugaredLogger
	approvalBridge *BPMNApprovalBridge
	// workflowOutboxEnabled 启用后，CreateRelease 在业务事务内以 transactional
	// outbox 落 CommandStartBPMN 命令（幂等键 release:<id>:workflow:start），
	// 由命令总线异步启动 release_approval_flow，保证「发布已建、流程必起」。
	// 未启用时退化为 goroutine fire-and-forget（at-most-once，失败仅错误日志
	// + 补跑指引），仅兼容单测/旧组装；生产组装只走持久化命令路径。
	workflowOutboxEnabled bool
	// processTriggerSvc 仅用于 outbox 未启用时的兼容触发路径。
	processTriggerSvc ProcessTriggerServiceInterface
}

// NewReleaseService 创建发布管理服务
func NewReleaseService(client *ent.Client, logger *zap.SugaredLogger) *ReleaseService {
	svc := &ReleaseService{
		client: client,
		logger: logger,
	}
	if client != nil {
		// 发布审批桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = NewBPMNApprovalBridge(client, logger)
	}
	return svc
}

// EnableWorkflowOutbox 启用发布创建的流程启动事务外发箱：
// CreateRelease 在同一事务内落 CommandStartBPMN 命令，替代 goroutine
// at-most-once 触发（与 ticket/incident 同口径）。
func (s *ReleaseService) EnableWorkflowOutbox() { s.workflowOutboxEnabled = true }

// SetProcessTriggerService 设置流程触发服务（仅 outbox 未启用时的兼容路径使用）。
func (s *ReleaseService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerSvc = triggerService
}

// validateReferences 校验关联变更/负责人属于同一租户。
func (s *ReleaseService) validateReferences(ctx context.Context, tenantID int, changeID, ownerID *int) error {
	if changeID != nil && *changeID > 0 {
		exists, err := s.client.Change.Query().
			Where(change.IDEQ(*changeID), change.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("校验关联变更失败: %w", err)
		}
		if !exists {
			return fmt.Errorf("关联变更不存在或不属于当前租户")
		}
	}
	if ownerID != nil && *ownerID > 0 {
		exists, err := s.client.User.Query().
			Where(user.IDEQ(*ownerID), user.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("校验负责人失败: %w", err)
		}
		if !exists {
			return fmt.Errorf("负责人不存在或不属于当前租户")
		}
	}
	return nil
}

// CreateRelease 创建发布
// P1 修复：
//  1. release_number 由服务端生成，不信任客户端传入值（避免越权/冲突编号）。
//  2. 两段写（先 Create 再对 JSON 字段二次 Save）包进单个事务，失败回滚，不留半成品。
func (s *ReleaseService) CreateRelease(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error) {
	releaseNumber := s.generateReleaseNumber(tenantID)

	if err := s.validateReferences(ctx, tenantID, req.ChangeID, req.OwnerID); err != nil {
		return nil, err
	}

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
		SetStatus(release.Status(dto.ReleaseStatusDraft)).
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

	// 发布创建即启动审批流（release_approval_flow）：outbox 启用时命令与业务
	// 写入同事务，失败整单回滚——不会出现「发布已建、无流程实例」的悬挂态；
	// 幂等键 release:<id>:workflow:start 保证命令重放不重复起流。
	if s.workflowOutboxEnabled {
		if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
			TenantID: tenantID, CommandType: commandbus.CommandStartBPMN,
			AggregateType: "release", AggregateID: releaseEntity.ID,
			IdempotencyKey: fmt.Sprintf("release:%d:workflow:start", releaseEntity.ID),
			Payload:        map[string]interface{}{"businessType": "release", "businessId": releaseEntity.ID},
		}); err != nil {
			s.logger.Errorw("Failed to enqueue release workflow command", "error", err, "release_id", releaseEntity.ID)
			return nil, fmt.Errorf("enqueue release workflow: %w", err)
		}
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

	// 兼容未启用 outbox 的单元测试/旧组装；生产组装只允许持久化命令路径。
	// 与 ticket/incident 同款语义：goroutine at-most-once 触发，失败即该单
	// 永久无流程实例，必须按错误级别暴露并附补跑指引。
	if s.processTriggerSvc != nil && !s.workflowOutboxEnabled {
		go func() {
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForRelease(workflowCtx, releaseEntity.ID, tenantID); err != nil {
				s.logger.Errorw("Failed to trigger workflow for release (non-outbox fire-and-forget, delivery=at-most-once)",
					"error", err, "release_id", releaseEntity.ID,
					"remedy", "启用 ITSM workflow outbox 或人工调用 /process-trigger 重放")
			}
		}()
	}

	s.logger.Infow("Release created successfully", "release_id", releaseEntity.ID, "tenant_id", tenantID, "release_number", releaseNumber)
	return response, nil
}

// triggerWorkflowForRelease 为发布触发审批工作流（outbox 未启用时的兼容路径）。
// 流程 key 与变量形状与 WorkflowStartCommandHandler 的 release 分支保持一致，
// 保证两条起流路径（outbox 命令 / 兼容触发）产生同一契约的流程实例。
func (s *ReleaseService) triggerWorkflowForRelease(ctx context.Context, releaseID int, tenantID int) error {
	rel, err := s.client.Release.Query().
		Where(release.IDEQ(releaseID), release.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeRelease,
		BusinessID:           rel.ID,
		ProcessDefinitionKey: "release_approval_flow",
		Variables: map[string]interface{}{
			"release_id": rel.ID, "release_number": rel.ReleaseNumber, "title": rel.Title,
			"description": rel.Description, "type": rel.Type, "severity": rel.Severity,
			"environment": rel.Environment, "status": string(rel.Status), "created_by": rel.CreatedBy,
			"requires_approval": rel.RequiresApproval, "is_emergency": rel.IsEmergency,
		},
		TriggeredBy: fmt.Sprintf("%d", rel.CreatedBy),
		TriggeredAt: time.Now(),
		TenantID:    tenantID,
	}

	if _, err := s.processTriggerSvc.TriggerProcess(ctx, triggerReq); err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow("Workflow triggered for release", "release_id", rel.ID, "release_number", rel.ReleaseNumber, "tenant_id", tenantID)
	return nil
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
		query = query.Where(release.StatusEQ(release.Status(status)))
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

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &dto.ReleaseListResponse{
		Items:      releases,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateRelease 更新发布。
// DataScope 行级写守卫（对齐 ticket/change/incident 写路径批次）：读不到的单据
// 同样不允许写——管理角色全租户可写，其余角色仅创建人或负责人（OwnerID）可写。
func (s *ReleaseService) UpdateRelease(ctx context.Context, id, tenantID int, req *dto.UpdateReleaseRequest, actorID int, actorRole string) (*dto.ReleaseResponse, error) {
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

	if !datascope.CanWriteResource(actorID, actorRole, releaseEntity.CreatedBy, releaseEntity.OwnerID) {
		s.logger.Warnw("Release update rejected by row-level guard",
			"release_id", id, "tenant_id", tenantID, "actor_id", actorID, "actor_role", actorRole)
		return nil, common.NewForbiddenError("无权限操作该发布单：仅创建人、负责人或管理员可操作")
	}

	if err := s.validateReferences(ctx, tenantID, req.ChangeID, req.OwnerID); err != nil {
		return nil, err
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

// UpdateReleaseStatus 更新发布状态，流转必须过 isValidReleaseStatusTransition 白名单，
// 否则持 release:write 的人可以绕过审批门禁：
//   - draft → cancelled（排期 draft → scheduled 不在此列，必须走 /approve 且需 release:approve）
//   - scheduled → in-progress / cancelled
//   - in-progress → completed / failed / rolled_back / cancelled
//   - completed / cancelled / rolled_back / failed 为终态（不可被复活）
//
// DataScope 行级写守卫与 UpdateRelease 同规则，且不排除 rolled_back 目标：状态机允许经
// /status 路由（仅 release:write）直达 rolled_back，把该目标排除出行级守卫等于留下绕行门禁的洞。
// owner/管理员回滚不受影响，非 owner 操作他人发布回滚被 403
// （专用 /rollback 路由的 release:rollback RBAC 仍先行生效）。
func (s *ReleaseService) UpdateReleaseStatus(ctx context.Context, id, tenantID int, status string, actorID int, actorRole string) (*dto.ReleaseResponse, error) {
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

	if !datascope.CanWriteResource(actorID, actorRole, releaseEntity.CreatedBy, releaseEntity.OwnerID) {
		s.logger.Warnw("Release status transition rejected by row-level guard",
			"release_id", id, "tenant_id", tenantID, "actor_id", actorID, "actor_role", actorRole, "target_status", status)
		return nil, common.NewForbiddenError("无权限操作该发布单：仅创建人、负责人或管理员可执行状态流转")
	}

	// 1. 状态机白名单校验
	if !isValidReleaseStatusTransition(string(releaseEntity.Status), status) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidReleaseTransition, releaseEntity.Status, status)
	}

	// 2. CAS 条件更新：谓词锚定读取时看到的状态值，防止并发窗口内
	// （scheduled→in_progress 或 failed→rolled_back 已发生）用陈旧校验结果
	// 覆盖推进后的状态。0 行命中 = 状态已被并发修改 → 409 冲突语义。
	builder := s.client.Release.Update().
		Where(
			release.IDEQ(id),
			release.TenantIDEQ(tenantID),
			release.StatusEQ(releaseEntity.Status),
		).
		SetStatus(release.Status(status))

	// 如果状态是已完成，设置实际发布日期
	if status == string(dto.ReleaseStatusCompleted) {
		now := time.Now()
		builder.SetActualReleaseDate(now)
	}

	affected, err := builder.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release status", "error", err, "release_id", id, "status", status)
		return nil, fmt.Errorf("failed to update release status: %w", err)
	}
	if affected == 0 {
		return nil, common.NewBusinessError(common.ConflictCode,
			"发布状态已被并发修改，请刷新后重试",
			fmt.Sprintf("expected %s", releaseEntity.Status))
	}

	updated, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload release after status update", "error", err, "release_id", id)
		return nil, fmt.Errorf("failed to reload release: %w", err)
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
			// draft 只能取消：排期是审批等效动作，只能经 /approve（release:approve）发生，
			// 否则持 release:write 的用户经 /status 路由即可绕过审批门禁。
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
			// failed 只能回滚或取消：重新排期同样属于审批等效动作，必须经 /approve（release:approve）。
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
// 则中止业务审批，避免发布状态与流程状态分叉。
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

	// 仅草稿态/失败态允许审批：失败态审批即「重新排期(reschedule)」，
	// 使发布历史不断裂、关联变更不丢。
	if releaseEntity.Status != release.Status(dto.ReleaseStatusDraft) && releaseEntity.Status != release.Status(dto.ReleaseStatusFailed) {
		// 非法状态属于 conflict（409），不是服务器故障——必须类型化，
		// 否则 RespondError 分类不到会兜底 500。
		return nil, common.NewBusinessError(common.ConflictCode,
			"当前发布状态不允许审批", string(releaseEntity.Status))
	}

	// 审批人校验：必须是本租户有效用户，且创建人不能审批自己的发布
	exists, err := s.client.User.Query().
		Where(user.ID(actorID), user.TenantID(tenantID), user.Active(true)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("校验审批人失败: %w", err)
	}
	if !exists {
		return nil, common.NewForbiddenError("审批人不存在或已停用")
	}
	if actorID == releaseEntity.CreatedBy {
		// 自审批属于业务拒绝（conflict 语义），必须类型化避免 500 兜底。
		return nil, common.NewBusinessError(common.ConflictCode,
			"发布创建人不能审批自己的发布", "")
	}

	// BPMN 任务完成与业务状态更新必须在同一事务内，否则会出现"流程已批、业务未批"。
	targetStatus := string(dto.ReleaseStatusScheduled)
	if action == "reject" {
		targetStatus = string(dto.ReleaseStatusCancelled)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启审批事务失败: %w", err)
	}
	defer tx.Rollback()

	if s.approvalBridge == nil {
		return nil, common.NewBusinessError(common.ConflictCode, "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态", "")
	}
	completed, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTaskWithClient(
		ctx, tx.Client(), tenantID, actorID, string(dto.BusinessTypeRelease), id, action, comment,
	)
	if bridgeErr != nil {
		return nil, fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
	}
	if !completed {
		return nil, common.NewBusinessError(common.ConflictCode, "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态", "")
	}

	// 审批状态推进同样走 CAS：防止并发窗口里已进入 in_progress 的发布
	// 被并发提交的旧审批结果改回 rejected/cancelled（fail-closed）。
	affected, err := tx.Release.Update().
		Where(
			release.IDEQ(id),
			release.TenantIDEQ(tenantID),
			release.StatusEQ(releaseEntity.Status),
		).
		SetStatus(release.Status(targetStatus)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release approval status", "error", err, "release_id", id, "status", targetStatus)
		return nil, fmt.Errorf("failed to update release status: %w", err)
	}
	if affected == 0 {
		return nil, common.NewBusinessError(common.ConflictCode,
			"发布状态已被并发修改，请刷新后重试",
			fmt.Sprintf("expected %s", releaseEntity.Status))
	}
	updated, err := tx.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update release approval status", "error", err, "release_id", id, "status", targetStatus)
		return nil, fmt.Errorf("failed to update release status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交审批事务失败: %w", err)
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
// DataScope 行级删守卫：幂等语义保留（查不到 = 已删 = 成功），查得到但非 owner 时
// 改为先查后删——403 而非静默删除（对齐其他域 Delete 守卫模式）。
func (s *ReleaseService) DeleteRelease(ctx context.Context, id, tenantID int, actorID int, actorRole string) error {
	releaseEntity, err := s.client.Release.Query().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 幂等：未匹配（已删/不存在/跨租户）= 视为已删除，不报错。
			s.logger.Infow("Release delete matched 0 rows (idempotent, treated as success)",
				"release_id", id, "tenant_id", tenantID)
			return nil
		}
		s.logger.Errorw("Failed to get release for delete", "error", err, "release_id", id)
		return fmt.Errorf("failed to delete release: %w", err)
	}

	if !datascope.CanWriteResource(actorID, actorRole, releaseEntity.CreatedBy, releaseEntity.OwnerID) {
		s.logger.Warnw("Release delete rejected by row-level guard",
			"release_id", id, "tenant_id", tenantID, "actor_id", actorID, "actor_role", actorRole)
		return common.NewForbiddenError("无权限删除该发布单：仅创建人、负责人或管理员可删除")
	}

	deleted, err := s.client.Release.Delete().
		Where(release.IDEQ(id), release.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete release", "error", err, "release_id", id)
		return fmt.Errorf("failed to delete release: %w", err)
	}
	if deleted == 0 {
		// 并发窗口兜底：查询到但删除时已被他方删除，仍按幂等成功处理。
		s.logger.Infow("Release delete matched 0 rows after guard (idempotent, treated as success)",
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
	draft, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusDraft))).Count(ctx)
	scheduled, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusScheduled))).Count(ctx)
	inProgress, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusInProgress))).Count(ctx)
	completed, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusCompleted))).Count(ctx)
	cancelled, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusCancelled))).Count(ctx)
	failed, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusFailed))).Count(ctx)
	rolledBack, _ := s.client.Release.Query().Where(release.TenantIDEQ(tenantID), release.StatusEQ(release.Status(dto.ReleaseStatusRolledBack))).Count(ctx)

	stats.Draft = draft
	stats.Scheduled = scheduled
	stats.InProgress = inProgress
	stats.Completed = completed
	stats.Cancelled = cancelled
	stats.Failed = failed
	stats.RolledBack = rolledBack

	return stats, nil
}
