package incident

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// ProcessTriggerServiceInterface 流程触发服务接口
type ProcessTriggerServiceInterface interface {
	TriggerProcess(ctx context.Context, req interface{}) (interface{}, error)
}

type Service struct {
	repo                  Repository
	productionService     *service.IncidentService
	monitoringService     *service.IncidentMonitoringService
	alertingSvc           *service.IncidentAlertingService
	rootCauseSvc          *service.RootCauseAnalysisService
	logger                *zap.SugaredLogger
	processTriggerService ProcessTriggerServiceInterface
}

// IncidentEventReadModel is the handler-facing projection for an incident event.
// Keeping this projection in the service prevents HTTP handlers from traversing
// Ent edges and accidentally depending on persistence implementation details.
type IncidentEventReadModel struct {
	ID          int
	IncidentID  int
	EventType   string
	EventName   string
	Description string
	OccurredAt  time.Time
	CreatedAt   time.Time
}

type IncidentAlertReadModel struct {
	ID          int
	IncidentID  int
	AlertName   string
	AlertType   string
	Severity    string
	Status      string
	TriggeredAt time.Time
}

type IncidentMetricsReadModel struct {
	ID           int
	CreatedAt    time.Time
	ResolvedAt   time.Time
	MetricsCount int
}

func NewService(repo Repository, productionSvc *service.IncidentService, monitoringSvc *service.IncidentMonitoringService, alertingSvc *service.IncidentAlertingService, rootCauseSvc *service.RootCauseAnalysisService, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:              repo,
		productionService: productionSvc,
		monitoringService: monitoringSvc,
		alertingSvc:       alertingSvc,
		rootCauseSvc:      rootCauseSvc,
		logger:            logger,
	}
}

// acknowledgeGuard 是 ack/resolve/close/reopen 共用的行级守卫。
// P1-DataScope：生命周期写操作与 Update/Delete 同属"操作他人单据"风险面，
// 写权限 ⊆ 读权限——普通角色仅可操作本人报告或受理的事件单。
func (s *Service) acknowledgeGuard(ctx context.Context, id, actorID int, actorRole string, tenantID int) error {
	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.ReporterID, current.AssigneeID) {
		return common.NewForbiddenError("无权限操作该事件单：仅报告人、受理人或管理员可执行生命周期操作")
	}
	return nil
}

func (s *Service) Acknowledge(ctx context.Context, id, userID, tenantID int, actorRole string) error {
	if err := s.acknowledgeGuard(ctx, id, userID, actorRole, tenantID); err != nil {
		return err
	}
	return lifecycleError(s.productionService.AcknowledgeIncident(ctx, id, userID, tenantID))
}

func (s *Service) Resolve(ctx context.Context, id, userID, tenantID int, resolution, rootCause, actorRole string) error {
	if err := s.acknowledgeGuard(ctx, id, userID, actorRole, tenantID); err != nil {
		return err
	}
	return lifecycleError(s.productionService.ResolveIncident(ctx, id, userID, tenantID, resolution, rootCause))
}

func (s *Service) Close(ctx context.Context, id, userID, tenantID int, closeNotes, actorRole string) error {
	if err := s.acknowledgeGuard(ctx, id, userID, actorRole, tenantID); err != nil {
		return err
	}
	return lifecycleError(s.productionService.CloseIncident(ctx, id, userID, tenantID, closeNotes))
}

func (s *Service) Reopen(ctx context.Context, id, userID, tenantID int, actorRole string) error {
	if err := s.acknowledgeGuard(ctx, id, userID, actorRole, tenantID); err != nil {
		return err
	}
	return lifecycleError(s.productionService.ReopenIncident(ctx, id, userID, tenantID))
}

// Assign 不纳入行级守卫：指派是"产生受理关系"的授权动作（受理人正是通过
// assign 产生，对其做 owner/assignee 校验会循环依赖），路由层已有专用
// incident:assign RBAC 门禁（见 router/incident_routes.go）。
func (s *Service) Assign(ctx context.Context, id, assigneeID, tenantID int) error {
	_, err := s.productionService.AssignIncident(ctx, id, assigneeID, tenantID)
	return err
}

func (s *Service) Delete(ctx context.Context, id, tenantID int, actorID int, actorRole string) error {
	// P1-DataScope：删除前行级校验——写权限 ⊆ 读权限，普通角色仅可删除
	// 本人报告或受理的事件单。
	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.ReporterID, current.AssigneeID) {
		return common.NewForbiddenError("无权限删除该事件单：仅报告人、受理人或管理员可操作")
	}
	return lifecycleError(s.productionService.DeleteIncident(ctx, id, tenantID))
}

func (s *Service) PauseSLA(_ context.Context, _, _ int) error {
	return common.NewBusinessError(common.ServiceUnavailableCode, "事件 SLA 暂停尚未接入计时服务", "")
}

func (s *Service) ResumeSLA(_ context.Context, _, _ int) error {
	return common.NewBusinessError(common.ServiceUnavailableCode, "事件 SLA 恢复尚未接入计时服务", "")
}

func (s *Service) CreateIncidentEvent(ctx context.Context, req *dto.CreateIncidentEventRequest, tenantID int) (*dto.IncidentEventResponse, error) {
	return s.productionService.CreateIncidentEvent(ctx, req, tenantID)
}

// SetProcessTriggerService 设置流程触发服务
func (s *Service) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

func (s *Service) Create(ctx context.Context, tenantID int, i *Incident) (*Incident, error) {
	s.logger.Infow("Creating incident", "title", i.Title, "tenant_id", tenantID)

	// Generate number
	year, month, _ := time.Now().Date()
	number, err := s.repo.GenerateIncidentNumber(ctx, tenantID, year, int(month))
	if err != nil {
		return nil, fmt.Errorf("failed to generate incident number: %w", err)
	}
	i.IncidentNumber = number
	i.TenantID = tenantID
	i.Status = "new" // default
	if strings.TrimSpace(i.Priority) == "" {
		i.Priority = inferIncidentPriority(i.Title, i.Description)
	}
	if i.DetectedAt.IsZero() {
		i.DetectedAt = time.Now()
	}

	created, err := s.repo.Create(ctx, i)
	if err != nil {
		return nil, err
	}

	// Audit Log
	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  created.ID,
		EventType:   "creation",
		EventName:   "事件创建",
		Description: fmt.Sprintf("事件 %s 已创建", number),
		Status:      "active",
		Severity:    "info",
		Source:      "system",
		UserID:      i.ReporterID,
		OccurredAt:  time.Now(),
		TenantID:    tenantID,
	})

	// Execute Rules Async（派生带租户上下文的独立 context，避免 RLS enforce 后异步规则失效）
	go s.executeRules(tenantctx.WithTenantID(context.Background(), tenantID), created, tenantID)

	return created, nil
}

// inferIncidentPriority provides a deterministic fallback when an operator or
// integration does not specify priority. Explicit priorities are never
// overwritten; automation rules can still refine the result after creation.
func inferIncidentPriority(title, description string) string {
	content := strings.ToLower(title + " " + description)
	for _, keyword := range []string{"down", "outage", "critical", "production"} {
		if strings.Contains(content, keyword) {
			return "urgent"
		}
	}
	for _, keyword := range []string{"slow", "error", "issue"} {
		if strings.Contains(content, keyword) {
			return "medium"
		}
	}
	return "low"
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Incident, error) {
	return s.repo.Get(ctx, id, tenantID)
}

// List 列出事件单。推广 ticket 的 DataScope 行级权限：
// 管理角色可见全租户，其余角色仅可见本人创建或分配给自己的事件单。
// currentUserID/currentRole 由 handler 从鉴权中间件注入的 user_id/role 取得。
func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, currentUserID int, currentRole string) ([]*Incident, int, error) {
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	return s.repo.List(ctx, tenantID, page, size, filters, dataScope, currentUserID)
}

// GetUserNames 批量查询用户姓名（供列表响应回填 reporterName/assigneeName）。
func (s *Service) GetUserNames(ctx context.Context, tenantID int, ids []int) (map[int]string, error) {
	return s.repo.GetUserNamesByIDs(ctx, tenantID, ids)
}

// Update 更新事件单。P1-DataScope：写路径行级校验——写权限 ⊆ 读权限，
// 普通角色仅可修改本人报告或受理的事件单（datascope.CanWriteResource）。
//
// 所有错误在边界统一经 lifecycleError 归一化：handler 走 common.RespondError，
// 它只识别 *AppError 与 *BusinessError，裸的领域哨兵（非法状态迁移、版本冲突、
// ent not-found）会掉进 500 兜底。集中在一处归一化可避免漏掉某条 return 路径。
func (s *Service) Update(ctx context.Context, tenantID int, id int, updates *Incident, actorID int, actorRole string) (*Incident, error) {
	updated, err := s.updateIncident(ctx, tenantID, id, updates, actorID, actorRole)
	if err != nil {
		return nil, lifecycleError(err)
	}
	return updated, nil
}

func (s *Service) updateIncident(ctx context.Context, tenantID int, id int, updates *Incident, actorID int, actorRole string) (*Incident, error) {
	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.ReporterID, current.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该事件单：仅报告人、受理人或管理员可操作")
	}

	// 客户端乐观锁：updates.Version 由 handler 从 UpdateIncidentRequest.Version 注入。
	// 该字段是指针语义的可选值，仅在客户端确实带了版本（>0）时校验；未带版本的旧
	// 调用方仍受仓储层 VersionEQ 条件更新保护，不会退化成静默覆盖。
	if updates.Version > 0 && updates.Version != current.Version {
		return nil, fmt.Errorf("%w: client=%d current=%d", service.ErrIncidentVersionConflict, updates.Version, current.Version)
	}

	// Apply updates
	// Note: This logic depends on what 'updates' contains.
	// In a real separate domain, we might pass specific fields or a map.
	// For simplicity, we assume 'updates' has non-zero values for fields to change.

	if updates.Title != "" {
		current.Title = updates.Title
	}
	if updates.Description != "" {
		current.Description = updates.Description
	}
	if updates.Status != "" {
		// 阻断6 修复：必须走事件状态机白名单校验，禁止裸写 SetStatus。
		// 旧逻辑允许终态（closed/cancelled）事件被任意改写回 active，
		// 也允许从 new 直接跳到 resolved 绕过处理流程。
		if !common.IsValidIncidentStatusTransition(current.Status, updates.Status) {
			return nil, fmt.Errorf("%w: from '%s' to '%s'", service.ErrIncidentInvalidTransition, current.Status, updates.Status)
		}
		current.Status = updates.Status
		if updates.Status == "resolved" {
			now := time.Now()
			current.ResolvedAt = &now
		}
		if updates.Status == "closed" {
			now := time.Now()
			current.ClosedAt = &now
		}
	}
	if updates.Priority != "" {
		current.Priority = updates.Priority
	}
	if updates.Severity != "" {
		current.Severity = updates.Severity
	}
	if updates.AssigneeID != nil {
		current.AssigneeID = updates.AssigneeID
	}
	// 修复：handler 已把 Category/Subcategory/Metadata/ImpactAnalysis/RootCause/
	// ResolutionSteps 从请求映射到 updates，此处却只留了 "// ... other fields"
	// 占位注释，导致 PUT 返回 code 0 但这六个字段被静默丢弃（IncidentDetail 的
	// 「保存事件分类」与事件编辑页的分类字段因此完全失效）。
	// map/slice 用 nil 判断，使客户端显式传空值仍能清空字段；string 沿用本函数
	// 既有的真值约定（清空文本字段的能力见 Update 的指针化改造事项）。
	if updates.Category != "" {
		current.Category = updates.Category
	}
	if updates.Subcategory != "" {
		current.Subcategory = updates.Subcategory
	}
	if updates.Metadata != nil {
		current.Metadata = updates.Metadata
	}
	if updates.ImpactAnalysis != nil {
		current.ImpactAnalysis = updates.ImpactAnalysis
	}
	if updates.RootCause != nil {
		current.RootCause = updates.RootCause
	}
	if updates.ResolutionSteps != nil {
		current.ResolutionSteps = updates.ResolutionSteps
	}

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  id,
		EventType:   "update",
		EventName:   "事件更新",
		Description: "事件信息已更新",
		OccurredAt:  time.Now(),
		TenantID:    tenantID,
		Source:      "system",
	})

	return updated, nil
}

// Escalate 升级事件。与 Update 同理，错误在边界统一经 lifecycleError 归一化，
// 避免行级守卫的 ent not-found 与仓储条件更新未命中退化成 500。
func (s *Service) Escalate(ctx context.Context, tenantID int, id int, level int, reason string, actorID int, actorRole string) (*Incident, error) {
	escalated, err := s.escalateIncident(ctx, tenantID, id, level, reason, actorID, actorRole)
	if err != nil {
		return nil, lifecycleError(err)
	}
	return escalated, nil
}

func (s *Service) escalateIncident(ctx context.Context, tenantID int, id int, level int, reason string, actorID int, actorRole string) (*Incident, error) {
	// P1-DataScope：升级是生命周期写操作，与 ack/resolve/close/reopen 同风险面，
	// 行级校验对齐 acknowledgeGuard（写权限 ⊆ 读权限）。
	if err := s.acknowledgeGuard(ctx, id, actorID, actorRole, tenantID); err != nil {
		return nil, err
	}

	current, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	current.EscalationLevel = level
	now := time.Now()
	current.EscalatedAt = &now

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.repo.CreateEvent(ctx, &IncidentEvent{
		IncidentID:  id,
		EventType:   "escalation",
		EventName:   "事件升级",
		Description: fmt.Sprintf("事件升级到级别 %d: %s", level, reason),
		Data: map[string]interface{}{
			"level":  level,
			"reason": reason,
		},
		OccurredAt: time.Now(),
		TenantID:   tenantID,
		Source:     "system",
	})

	return updated, nil
}

// executeRules Logic
func (s *Service) executeRules(ctx context.Context, incident *Incident, tenantID int) {
	rules, err := s.repo.ListActiveRules(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to list active rules", "error", err)
		return
	}

	for _, rule := range rules {
		if s.evaluateCondition(rule.Conditions, incident) {
			s.executeAction(ctx, rule, incident, tenantID)
		}
	}
}

func (s *Service) evaluateCondition(conditions map[string]interface{}, incident *Incident) bool {
	// Simplified evaluation logic
	if priority, ok := conditions["priority"].([]string); ok {
		match := false
		for _, p := range priority {
			if incident.Priority == p {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	if status, ok := conditions["status"].(string); ok {
		if incident.Status != status {
			return false
		}
	}
	// Add more conditions as needed
	return true
}

func (s *Service) executeAction(ctx context.Context, rule *IncidentRule, incident *Incident, tenantID int) {
	// Execute implementation
	// ... (Simplification: updating stats and logging for now to avoid circular service dependencies if action updates incident again)
	s.logger.Infow("Rule Executed", "rule_id", rule.ID, "incident_id", incident.ID)

	// Update stats
	s.repo.UpdateRuleStats(ctx, rule.ID, rule.ExecutionCount+1, time.Now())
}

// GetStats 聚合事件统计（按 tenant 隔离）
// 该方法仅做租户透传，业务规则（如缓存、敏感字段脱敏）由调用层决定；
// 当前实现直接返回仓储聚合结果，后续若启用 Redis 缓存（见方案「不在本方案范围」）
// 可在此层加缓存读写与租户失效。
func (s *Service) GetStats(ctx context.Context, tenantID int) (*IncidentStats, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("invalid tenant id: %d", tenantID)
	}
	return s.repo.GetStats(ctx, tenantID)
}

// ─── 子资源读模型（自 handler 下沉）────────────────────────────────

// GetIncidentEvents 返回事件的全部活动记录（按 Edge 加载）。
func (s *Service) GetIncidentEvents(ctx context.Context, id, tenantID int) ([]IncidentEventReadModel, error) {
	item, err := s.repo.GetIncidentWithEdges(ctx, id, tenantID, "events")
	if err != nil {
		return nil, err
	}
	result := make([]IncidentEventReadModel, 0, len(item.Edges.IncidentEvents))
	for _, event := range item.Edges.IncidentEvents {
		result = append(result, IncidentEventReadModel{ID: event.ID, IncidentID: event.IncidentID, EventType: event.EventType, EventName: event.EventName, Description: event.Description, OccurredAt: event.OccurredAt, CreatedAt: event.CreatedAt})
	}
	return result, nil
}

// GetIncidentAlerts 返回事件的全部告警（按 Edge 加载）。
func (s *Service) GetIncidentAlerts(ctx context.Context, id, tenantID int) ([]IncidentAlertReadModel, error) {
	item, err := s.repo.GetIncidentWithEdges(ctx, id, tenantID, "alerts")
	if err != nil {
		return nil, err
	}
	result := make([]IncidentAlertReadModel, 0, len(item.Edges.IncidentAlerts))
	for _, alert := range item.Edges.IncidentAlerts {
		result = append(result, IncidentAlertReadModel{ID: alert.ID, IncidentID: alert.IncidentID, AlertName: alert.AlertName, AlertType: alert.AlertType, Severity: alert.Severity, Status: alert.Status, TriggeredAt: alert.TriggeredAt})
	}
	return result, nil
}

// GetIncidentMetricsData 返回事件及其指标边，供指标计算。
func (s *Service) GetIncidentMetricsData(ctx context.Context, id, tenantID int) (*IncidentMetricsReadModel, error) {
	item, err := s.repo.GetIncidentWithEdges(ctx, id, tenantID, "metrics")
	if err != nil {
		return nil, err
	}
	return &IncidentMetricsReadModel{ID: item.ID, CreatedAt: item.CreatedAt, ResolvedAt: item.ResolvedAt, MetricsCount: len(item.Edges.IncidentMetrics)}, nil
}

// CountTenantSLAViolations 统计租户级 SLA 违规数（指标接口用）。
func (s *Service) CountTenantSLAViolations(ctx context.Context, tenantID int) int {
	count, err := s.repo.CountTenantSLAViolations(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("GetIncidentMetrics: failed to count SLA violations", "error", err)
		return 0
	}
	return count
}

// ListIncidentComments 返回事件评论（event_type=comment 的 IncidentEvent）。
func (s *Service) ListIncidentComments(ctx context.Context, incidentID, tenantID int) ([]*ent.IncidentEvent, error) {
	return s.repo.ListIncidentComments(ctx, incidentID, tenantID)
}

// CreateIncidentComment 校验事件归属后写入一条评论。
func (s *Service) CreateIncidentComment(ctx context.Context, incidentID, tenantID, userID int, content string, isInternal bool) (*ent.IncidentEvent, error) {
	if _, err := s.repo.GetIncidentWithEdges(ctx, incidentID, tenantID); err != nil {
		return nil, err
	}
	return s.repo.CreateIncidentCommentEvent(ctx, &ent.IncidentEvent{
		IncidentID:  incidentID,
		EventType:   "comment",
		EventName:   "用户评论",
		Description: content,
		Status:      "active",
		UserID:      userID,
		Source:      "user",
		Data:        map[string]interface{}{"isInternal": isInternal},
		TenantID:    tenantID,
		OccurredAt:  time.Now(),
	})
}

// lifecycleError preserves domain failure semantics without exposing provider errors.
func lifecycleError(err error) error {
	switch {
	case errors.Is(err, service.ErrIncidentInvalidTransition):
		return common.NewBusinessError(common.ConflictCode, "当前事件状态不允许此操作", "")
	case errors.Is(err, service.ErrIncidentVersionConflict):
		return common.NewBusinessError(common.ConflictCode, "事件已被修改，请刷新后重试", "")
	case errors.Is(err, ErrStaleVersion):
		// 仓储条件更新未命中：读取快照后行已被并发写入推进。与客户端版本不符
		// 同属冲突语义，必须提示刷新重试而不是 500。
		return common.NewBusinessError(common.ConflictCode, "事件已被修改，请刷新后重试", "")
	case errors.Is(err, service.ErrIncidentResolutionRequired):
		return common.NewBusinessError(common.ParamErrorCode, "请填写解决方案", "")
	case errors.Is(err, service.ErrIncidentCloseNotesRequired):
		return common.NewBusinessError(common.ParamErrorCode, "请填写关闭说明", "")
	case errors.Is(err, service.ErrIncidentNotFound):
		return common.NewBusinessError(common.NotFoundCode, "事件不存在", "")
	case ent.IsNotFound(err):
		// 仓储读/条件更新未命中：资源不存在或跨租户（fail closed 后同样表现为
		// 不存在）。必须映射成 404/4004，否则经 common.RespondError 的写路径
		//（Update/Escalate）会掉进 500 兜底，与 failIncidentOperation 不一致。
		return common.NewBusinessError(common.NotFoundCode, "事件不存在", "")
	default:
		return err
	}
}
