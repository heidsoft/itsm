package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/emailconversation"
	"itsm-backend/ent/group"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"
	"itsm-backend/ent/incidentrule"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/supportcontract"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"
	"itsm-backend/middleware"

	"go.uber.org/zap"
)

// 事件领域哨兵错误，供 controller 通过 errors.Is 做精确错误分类，
// 避免依赖易碎的字符串比较（err.Error() == "incident not found"）。
var (
	ErrIncidentNotFound                  = errors.New("incident not found")
	ErrIncidentTerminal                  = errors.New("terminal incident cannot be escalated")
	ErrIncidentEscalationLevelInvalid    = errors.New("escalation level must be between 1 and 5")
	ErrIncidentEscalationLevelNotGreater = errors.New("escalation level must be greater than current level")
	ErrIncidentEscalationReasonRequired  = errors.New("escalation reason is required")
	ErrEmailContractNotActive            = errors.New("support contract is not active")
	ErrIncidentInvalidTransition         = errors.New("invalid incident status transition")
	ErrIncidentResolutionRequired        = errors.New("incident resolution is required")
	ErrIncidentCloseNotesRequired        = errors.New("incident close notes are required")
	ErrIncidentVersionConflict           = errors.New("incident version conflict")
)

type EmailIncidentCommand struct {
	ConversationID    int
	SupportContractID int
	ReporterUserID    int
	AssignmentGroupID *int
	AssigneeID        *int
	Title             string
	Description       string
	Impact            string
	Urgency           string
	Category          string
	Metadata          map[string]interface{}
	OverrideContract  bool
	OverrideReason    string
}

type incidentCreateOptions struct {
	isAutomated           bool
	emailConversationID   *int
	supportContractID     *int
	assignmentGroupID     *int
	allowInactiveContract bool
}

type IncidentService struct {
	priorityMatrixService *PriorityMatrixService
	client                *ent.Client
	logger                *zap.SugaredLogger
	sequenceService       *SequenceService
	processTriggerService ProcessTriggerServiceInterface
	ruleEngine            *IncidentRuleEngine
	rawDB                 *sql.DB // for transactional SELECT FOR UPDATE (S-4 修复)
	workflowOutboxEnabled bool
	rulesOutboxEnabled    bool
	slaSvc                *TicketSLAService
}

// SetSLAService 注入SLA服务，用于创建Incident时自动绑定SLA策略
func (s *IncidentService) SetSLAService(sla *TicketSLAService) {
	s.slaSvc = sla
}

func NewIncidentService(client *ent.Client, logger *zap.SugaredLogger, slaSvc *TicketSLAService) *IncidentService {
	return &IncidentService{
		client: client,
		logger: logger,
		slaSvc: slaSvc,
	}
}

// SetProcessTriggerService 设置流程触发服务
func (s *IncidentService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

func (s *IncidentService) EnableWorkflowOutbox() { s.workflowOutboxEnabled = true }
func (s *IncidentService) EnableRulesOutbox()    { s.rulesOutboxEnabled = true }

// SetSequenceService 设置序列服务（用于 incident_number 生成）
func (s *IncidentService) SetPriorityMatrixService(pms *PriorityMatrixService) {
	s.priorityMatrixService = pms
}

func (s *IncidentService) SetSequenceService(seq *SequenceService) {
	s.sequenceService = seq
}

// SetRawDB 设置原生数据库连接（用于事务性编号生成，S-4 修复）
func (s *IncidentService) SetRawDB(db *sql.DB) {
	s.rawDB = db
}

func (s *IncidentService) SetRuleEngine(engine *IncidentRuleEngine) {
	s.ruleEngine = engine
}

// CreateIncident 创建事件
func (s *IncidentService) CreateIncident(ctx context.Context, req *dto.CreateIncidentRequest, tenantID, userID int) (*dto.IncidentResponse, error) {
	return s.createIncident(ctx, req, tenantID, userID, incidentCreateOptions{})
}

func (s *IncidentService) CreateFromEmail(ctx context.Context, tenantID int, command EmailIncidentCommand) (*dto.IncidentResponse, error) {
	if command.ConversationID <= 0 || command.SupportContractID <= 0 || command.ReporterUserID <= 0 {
		return nil, errors.New("email conversation, support contract and automation reporter are required")
	}
	existing, err := s.client.Incident.Query().Where(
		incident.TenantIDEQ(tenantID), incident.EmailConversationIDEQ(command.ConversationID),
	).Only(ctx)
	if err == nil {
		return dto.ToIncidentResponse(existing), nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("check email incident idempotency: %w", err)
	}
	metadata := make(map[string]interface{}, len(command.Metadata)+2)
	for key, value := range command.Metadata {
		metadata[key] = value
	}
	metadata["emailConversationId"] = command.ConversationID
	metadata["supportContractId"] = command.SupportContractID
	if command.OverrideContract {
		if strings.TrimSpace(command.OverrideReason) == "" {
			return nil, errors.New("contract override reason is required")
		}
		metadata["contractOverride"] = true
		metadata["contractOverrideReason"] = command.OverrideReason
	}
	req := &dto.CreateIncidentRequest{Title: command.Title, Description: command.Description, Type: "incident", Impact: command.Impact, Urgency: command.Urgency, Category: command.Category, Source: "email", AssigneeID: command.AssigneeID, Metadata: metadata}
	response, createErr := s.createIncident(ctx, req, tenantID, command.ReporterUserID, incidentCreateOptions{isAutomated: true, emailConversationID: &command.ConversationID, supportContractID: &command.SupportContractID, assignmentGroupID: command.AssignmentGroupID, allowInactiveContract: command.OverrideContract})
	if createErr != nil && ent.IsConstraintError(createErr) {
		existing, lookupErr := s.client.Incident.Query().Where(incident.TenantIDEQ(tenantID), incident.EmailConversationIDEQ(command.ConversationID)).Only(ctx)
		if lookupErr == nil {
			return dto.ToIncidentResponse(existing), nil
		}
	}
	return response, createErr
}

func (s *IncidentService) createIncident(ctx context.Context, req *dto.CreateIncidentRequest, tenantID, userID int, options incidentCreateOptions) (*dto.IncidentResponse, error) {
	s.logger.Infow("Creating incident", "title", req.Title, "tenant_id", tenantID, "user_id", userID)
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("incident title is required")
	}
	reporterExists, err := s.client.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate reporter: %w", err)
	}
	if !reporterExists {
		return nil, fmt.Errorf("reporter not found or inactive")
	}
	if req.AssigneeID != nil {
		if err := s.validateIncidentAssignee(ctx, *req.AssigneeID, tenantID); err != nil {
			return nil, err
		}
	}
	if options.assignmentGroupID != nil {
		exists, validateErr := s.client.Group.Query().Where(group.IDEQ(*options.assignmentGroupID), group.TenantIDEQ(tenantID)).Exist(ctx)
		if validateErr != nil || !exists {
			return nil, errors.New("assignment group not found in tenant")
		}
	}
	if options.emailConversationID != nil {
		exists, validateErr := s.client.EmailConversation.Query().Where(emailconversation.IDEQ(*options.emailConversationID), emailconversation.TenantIDEQ(tenantID)).Exist(ctx)
		if validateErr != nil || !exists {
			return nil, errors.New("email conversation not found in tenant")
		}
	}
	var configurationItems []*ent.ConfigurationItem
	if len(req.ConfigurationItemIDs) > 0 {
		configurationItems, err = s.client.ConfigurationItem.Query().
			Where(configurationitem.IDIn(req.ConfigurationItemIDs...), configurationitem.TenantIDEQ(tenantID)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to validate configuration items: %w", err)
		}
		if len(configurationItems) != len(req.ConfigurationItemIDs) {
			return nil, fmt.Errorf("one or more configuration items not found")
		}
	}

	// 生成事件编号
	incidentNumber, err := s.generateIncidentNumber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate incident number: %w", err)
	}

	// 设置检测时间
	detectedAt := time.Now()
	if req.DetectedAt != nil {
		detectedAt = *req.DetectedAt
	}

	impact := req.Impact
	if impact == "" {
		impact = "medium"
	}
	urgency := req.Urgency
	if urgency == "" {
		urgency = "medium"
	}
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}

	// 计算优先级
	priority := req.Priority
	if priority == "" && s.priorityMatrixService != nil {
		calculatedPriority, err := s.priorityMatrixService.CalculatePriority(tenantID, impact, urgency)
		if err != nil {
			s.logger.Warnw("Failed to calculate priority, using default medium", "error", err)
			priority = "medium"
		} else {
			priority = calculatedPriority
		}
	}

	// 如果最终priority还是空，使用默认值
	if priority == "" {
		priority = "medium"
	}

	incidentType := req.Type
	if incidentType == "" {
		incidentType = "incident"
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start incident transaction: %w", err)
	}
	rollback := func(cause error) (*dto.IncidentResponse, error) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			s.logger.Errorw("Failed to rollback incident transaction", "error", rollbackErr)
		}
		return nil, cause
	}
	if options.supportContractID != nil {
		contractQuery := tx.SupportContract.Query().Where(supportcontract.IDEQ(*options.supportContractID), supportcontract.TenantIDEQ(tenantID))
		if !options.allowInactiveContract {
			contractQuery.Where(supportcontract.StatusEQ("active"))
		}
		active, contractErr := contractQuery.Exist(ctx)
		if contractErr != nil {
			return rollback(fmt.Errorf("recheck support contract: %w", contractErr))
		}
		if !active {
			return rollback(ErrEmailContractNotActive)
		}
	}
	create := tx.Incident.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetStatus("new").
		SetType(incidentType).
		SetPriority(priority).
		SetSeverity(severity).
		SetImpact(impact).
		SetUrgency(urgency).
		SetIncidentNumber(incidentNumber).
		SetReporterID(userID).
		SetCategory(req.Category).
		SetSubcategory(req.Subcategory).
		SetImpactAnalysis(dto.StructToMap(req.ImpactAnalysis)).
		SetSource(source).
		SetMetadata(req.Metadata).
		SetDetectedAt(detectedAt).
		SetIsAutomated(options.isAutomated).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		AddConfigurationItemIDs(req.ConfigurationItemIDs...)
	if req.AssigneeID != nil {
		create.SetAssigneeID(*req.AssigneeID)
	}
	if options.assignmentGroupID != nil {
		create.SetAssignmentGroupID(*options.assignmentGroupID)
	}
	if options.emailConversationID != nil {
		create.SetEmailConversationID(*options.emailConversationID)
	}
	incidentEntity, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident", "error", err)
		return rollback(fmt.Errorf("failed to create incident: %w", err))
	}

	// 绑定SLA策略（P0-1）：创建Incident时自动匹配SLA并计算截止时间
	if s.slaSvc != nil {
		s.logger.Infow("SLA binding start", "slaSvc_nil", s.slaSvc == nil, "priority", priority)
		slaResult, slaErr := s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, "incident", priority)
		s.logger.Infow("SLA calculated", "definition_id", slaResult.SLADefinitionID, "response_deadline", slaResult.ResponseDeadline, "resolution_deadline", slaResult.ResolutionDeadline, "err", slaErr)
		if slaErr != nil {
			s.logger.Warnw("Failed to calculate SLA for incident, continuing without SLA", "error", slaErr)
		} else {
			slaUpdater := tx.Incident.UpdateOne(incidentEntity)
			if slaResult.SLADefinitionID > 0 {
				slaUpdater.SetSLADefinitionID(slaResult.SLADefinitionID)
			}
			if slaResult.ResponseDeadline != nil {
				slaUpdater.SetSLAResponseDeadline(*slaResult.ResponseDeadline)
			}
			if slaResult.ResolutionDeadline != nil {
				slaUpdater.SetSLAResolutionDeadline(*slaResult.ResolutionDeadline)
			}
			if err := slaUpdater.Exec(ctx); err != nil {
				s.logger.Warnw("Failed to set SLA deadlines on incident, continuing", "error", err)
			}
			s.logger.Infow("SLA update exec done", "err", err)
		}
	}

	_, err = tx.IncidentEvent.Create().
		SetIncidentID(incidentEntity.ID).
		SetEventType("creation").
		SetEventName("事件创建").
		SetDescription(fmt.Sprintf("事件 %s 已创建", incidentNumber)).
		SetStatus("active").
		SetSeverity("info").
		SetSource("system").
		SetUserID(userID).
		SetOccurredAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return rollback(fmt.Errorf("failed to create incident event: %w", err))
	}
	if s.workflowOutboxEnabled {
		_, err = commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
			TenantID: tenantID, CommandType: commandbus.CommandStartBPMN,
			AggregateType: "incident", AggregateID: incidentEntity.ID,
			IdempotencyKey: fmt.Sprintf("incident:%d:workflow:start", incidentEntity.ID),
			Payload:        map[string]interface{}{"businessType": "incident", "businessId": incidentEntity.ID},
		})
		if err != nil {
			return rollback(fmt.Errorf("enqueue incident workflow: %w", err))
		}
	}
	if s.rulesOutboxEnabled {
		_, err = commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
			TenantID: tenantID, CommandType: commandbus.CommandExecuteIncidentRules,
			AggregateType: "incident", AggregateID: incidentEntity.ID,
			IdempotencyKey: fmt.Sprintf("incident:%d:rules:create", incidentEntity.ID),
			Payload:        map[string]interface{}{"event": "created"},
		})
		if err != nil {
			return rollback(fmt.Errorf("enqueue incident rules: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return rollback(fmt.Errorf("failed to commit incident transaction: %w", err))
	}
	incidentEntity.Edges.ConfigurationItems = configurationItems

	// 执行事件规则
	if !s.rulesOutboxEnabled {
		go func() {
			ruleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if s.ruleEngine != nil {
				if err := s.ruleEngine.ExecuteRulesForIncident(ruleCtx, incidentEntity.ID, tenantID); err != nil {
					s.logger.Errorw("Incident rule execution completed with failures", "error", err, "incident_id", incidentEntity.ID)
				}
				return
			}
			s.executeIncidentRules(ruleCtx, incidentEntity.ID, tenantID)
		}()
	}

	// 兼容未启用 outbox 的单元测试/旧组装；生产组装只允许持久化命令路径。
	if s.processTriggerService != nil && !s.workflowOutboxEnabled {
		go func() {
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForIncident(workflowCtx, incidentEntity.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to trigger workflow for incident", "error", err, "incident_id", incidentEntity.ID)
			}
		}()
	}

	s.logger.Infow("Incident created successfully", "id", incidentEntity.ID, "number", incidentNumber)

	// 重新查询以获取 SLA 更新后的最新数据（事务 UPDATE 后 incidentEntity 缓存未刷新）
	freshIncident, err := s.client.Incident.Get(ctx, incidentEntity.ID)
	if err != nil {
		s.logger.Warnw("Failed to refetch incident after SLA update, using original", "error", err)
		freshIncident = incidentEntity
	}
	return s.toIncidentResponse(freshIncident), nil
}

// GetIncident 获取事件
func (s *IncidentService) GetIncident(ctx context.Context, id int, tenantID int) (*dto.IncidentResponse, error) {
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(id),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		WithConfigurationItems().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentNotFound
		}
		s.logger.Errorw("Failed to get incident", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	return s.toIncidentResponse(incidentEntity), nil
}

// ListIncidents 获取事件列表
func (s *IncidentService) ListIncidents(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*dto.IncidentResponse, int, error) {
	query := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil())
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 200 {
		size = 200
	}

	// 应用过滤器
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where(incident.StatusEQ(status))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(incident.PriorityEQ(priority))
	}
	if severity, ok := filters["severity"].(string); ok && severity != "" {
		query = query.Where(incident.SeverityEQ(severity))
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		query = query.Where(incident.CategoryEQ(category))
	}
	if source, ok := filters["source"].(string); ok && source != "" {
		query = query.Where(incident.SourceEQ(source))
	}
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		// 关键词搜索：标题、描述、事件编号
		query = query.Where(
			incident.Or(
				incident.TitleContains(keyword),
				incident.DescriptionContains(keyword),
				incident.IncidentNumberContains(keyword),
			),
		)
	}
	if assigneeID, ok := filters["assignee_id"].(int); ok && assigneeID > 0 {
		query = query.Where(incident.AssigneeIDEQ(assigneeID))
	}
	if isMajor, ok := filters["is_major_incident"].(bool); ok {
		query = query.Where(incident.IsMajorIncidentEQ(isMajor))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count incidents", "error", err)
		return nil, 0, fmt.Errorf("failed to count incidents: %w", err)
	}

	// 分页查询
	incidents, err := query.
		WithConfigurationItems().
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(incident.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list incidents", "error", err)
		return nil, 0, fmt.Errorf("failed to list incidents: %w", err)
	}

	responses := make([]*dto.IncidentResponse, len(incidents))
	for i, incidentEntity := range incidents {
		responses[i] = s.toIncidentResponse(incidentEntity)
	}

	return responses, total, nil
}

// LinkIncidentCIs links configuration items to an incident.
func (s *IncidentService) LinkIncidentCIs(ctx context.Context, incidentID int, ciIDs []int, tenantID int) error {
	incidentEntity, err := s.client.Incident.Query().
		Where(incident.IDEQ(incidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentNotFound
		}
		return fmt.Errorf("failed to get incident: %w", err)
	}
	if len(ciIDs) == 0 {
		return nil
	}

	count, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDIn(ciIDs...), configurationitem.TenantIDEQ(incidentEntity.TenantID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate configuration items: %w", err)
	}
	if count != len(ciIDs) {
		return fmt.Errorf("one or more configuration items not found")
	}

	if _, err := s.client.Incident.UpdateOneID(incidentID).
		Where(incident.TenantIDEQ(incidentEntity.TenantID), incident.DeletedAtIsNil()).
		AddConfigurationItemIDs(ciIDs...).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to link configuration items: %w", err)
	}
	return nil
}

// GetIncidentCIs returns the configuration items linked to an incident.
func (s *IncidentService) GetIncidentCIs(ctx context.Context, incidentID int, tenantID int) ([]dto.CIInfo, error) {
	incidentEntity, err := s.client.Incident.Query().
		Where(incident.IDEQ(incidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		WithConfigurationItems().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to get incident configuration items: %w", err)
	}

	cis := make([]dto.CIInfo, 0, len(incidentEntity.Edges.ConfigurationItems))
	for _, ci := range incidentEntity.Edges.ConfigurationItems {
		cis = append(cis, dto.CIInfo{ID: ci.ID, Name: ci.Name})
	}
	return cis, nil
}

// UpdateIncident 更新事件
func (s *IncidentService) UpdateIncident(ctx context.Context, id int, req *dto.UpdateIncidentRequest, tenantID int) (*dto.IncidentResponse, error) {
	s.logger.Infow("Updating incident", "id", id, "tenant_id", tenantID)
	if req.Force {
		role, ok := middleware.RBACRoleFromContext(ctx)
		if !ok || !middleware.HasResourcePermission(ctx, s.client, role, "incident", "force-update", tenantID) {
			return nil, errors.New("incident:force-update permission required when force=true")
		}
	}

	// 获取当前事件实体
	currentIncident, err := s.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	// 版本检查（乐观锁）- 除非明确强制更新
	if !req.Force && req.Version > 0 && currentIncident.Version != req.Version {
		return nil, common.NewVersionConflictError(
			"事件",
			id,
			req.Version,
			currentIncident.Version,
		)
	}

	// 如果要更新状态，验证状态转换是否合法
	if req.Status != nil {
		// 验证状态转换
		if !isValidIncidentStatusTransition(currentIncident.Status, *req.Status) {
			return nil, fmt.Errorf("invalid status transition from '%s' to '%s'", currentIncident.Status, *req.Status)
		}
		// 解决与关闭必须走专用动作，确保解决说明、关闭备注和审计事件不可被通用更新绕过。
		if *req.Status == common.IncidentStatusResolved || *req.Status == common.IncidentStatusClosed {
			return nil, fmt.Errorf("use the dedicated resolve or close action for this status transition")
		}
	}
	if req.AssigneeID != nil {
		if err := s.validateIncidentAssignee(ctx, *req.AssigneeID, tenantID); err != nil {
			return nil, err
		}
	}

	// 计算优先级：如果用户没有显式传入Priority，但修改了Impact或Urgency，则自动重新计算
	priority := req.Priority
	if priority == nil && s.priorityMatrixService != nil && (req.Impact != nil || req.Urgency != nil) {
		// 使用新的Impact或现有Impact
		impact := currentIncident.Impact
		if req.Impact != nil {
			impact = *req.Impact
		}

		// 使用新的Urgency或现有Urgency
		urgency := currentIncident.Urgency
		if req.Urgency != nil {
			urgency = *req.Urgency
		}

		calculatedPriority, err := s.priorityMatrixService.CalculatePriority(tenantID, impact, urgency)
		if err != nil {
			s.logger.Warnw("Failed to calculate priority during update, keeping current value", "error", err)
		} else {
			priority = &calculatedPriority
		}
	}

	updateQuery := s.client.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		SetUpdatedAt(time.Now())
	if !req.Force && req.Version > 0 {
		updateQuery.Where(incident.VersionEQ(req.Version))
	}

	if req.Title != nil {
		updateQuery.SetTitle(*req.Title)
	}
	if req.Description != nil {
		updateQuery.SetDescription(*req.Description)
	}
	if req.Status != nil {
		updateQuery.SetStatus(*req.Status)
		// 如果状态变更为resolved，设置解决时间
		if *req.Status == common.IncidentStatusResolved {
			now := time.Now()
			updateQuery.SetResolvedAt(now)
			updateQuery.ClearClosedAt()
		}
		// 如果状态变更为closed，设置关闭时间
		if *req.Status == common.IncidentStatusClosed {
			now := time.Now()
			updateQuery.SetClosedAt(now)
		}
		if *req.Status == common.IncidentStatusInProgress && currentIncident.Status == common.IncidentStatusResolved {
			updateQuery.ClearResolvedAt().ClearClosedAt()
		}
	}
	if priority != nil {
		updateQuery.SetPriority(*priority)
	}
	if req.Severity != nil {
		updateQuery.SetSeverity(*req.Severity)
	}
	if req.Impact != nil {
		updateQuery.SetImpact(*req.Impact)
	}
	if req.Urgency != nil {
		updateQuery.SetUrgency(*req.Urgency)
	}
	if req.AssigneeID != nil {
		updateQuery.SetAssigneeID(*req.AssigneeID)
	}
	if req.ImpactAnalysis != nil {
		updateQuery.SetImpactAnalysis(dto.StructToMap(req.ImpactAnalysis))
	}
	if req.RootCause != nil {
		updateQuery.SetRootCause(dto.StructToMap(req.RootCause))
	}
	if req.ResolutionSteps != nil {
		updateQuery.SetResolutionSteps(dto.StructSliceToMapSlice(req.ResolutionSteps))
	}
	if req.Metadata != nil {
		updateQuery.SetMetadata(req.Metadata)
	}
	if req.Category != nil {
		updateQuery.SetCategory(*req.Category)
	}
	if req.Subcategory != nil {
		updateQuery.SetSubcategory(*req.Subcategory)
	}

	// 自动增加版本号
	updateQuery.AddVersion(1)

	incidentEntity, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			if !req.Force && req.Version > 0 {
				latest, lookupErr := s.client.Incident.Query().
					Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
					Only(ctx)
				if lookupErr == nil {
					return nil, common.NewVersionConflictError("事件", id, req.Version, latest.Version)
				}
			}
			return nil, ErrIncidentNotFound
		}
		s.logger.Errorw("Failed to update incident", "error", err)
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	// 记录事件更新活动
	_, err = s.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  id,
		EventType:   "update",
		EventName:   "事件更新",
		Description: "事件信息已更新",
		Status:      "active",
		Severity:    "info",
		Source:      "system",
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create incident event", "error", err)
	}

	s.logger.Infow("Incident updated successfully", "id", id)
	return s.toIncidentResponse(incidentEntity), nil
}

// AssignIncident 分配事件
func (s *IncidentService) AssignIncident(ctx context.Context, id int, assigneeID int, tenantID int) (*dto.IncidentResponse, error) {
	s.logger.Infow("Assigning incident", "id", id, "assignee_id", assigneeID, "tenant_id", tenantID)

	// 获取当前事件
	current, err := s.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	if err := s.validateIncidentAssignee(ctx, assigneeID, tenantID); err != nil {
		return nil, err
	}

	// 更新分配人（版本条件更新，防止并发分配静默覆盖）
	updatedIncident, err := s.client.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(current.Version)).
		SetAssigneeID(assigneeID).
		SetUpdatedAt(time.Now()).
		AddVersion(1).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentVersionConflict
		}
		s.logger.Errorw("Failed to assign incident", "error", err, "id", id)
		return nil, fmt.Errorf("failed to assign incident: %w", err)
	}

	// 记录分配活动
	s.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  id,
		EventType:   "assignment",
		EventName:   "事件分配",
		Description: fmt.Sprintf("事件已分配给用户 %d", assigneeID),
		Status:      "active",
		Severity:    "info",
		Source:      "user",
	}, tenantID)

	s.logger.Infow("Incident assigned successfully", "id", id, "assignee_id", assigneeID)
	return s.toIncidentResponse(updatedIncident), nil
}

func (s *IncidentService) validateIncidentAssignee(ctx context.Context, assigneeID, tenantID int) error {
	if assigneeID <= 0 {
		return fmt.Errorf("invalid assignee id")
	}
	assigneeExists, err := s.client.User.Query().
		Where(user.IDEQ(assigneeID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate assignee: %w", err)
	}
	if !assigneeExists {
		return fmt.Errorf("assignee not found or inactive")
	}
	return nil
}

func (s *IncidentService) ensureActiveIncident(ctx context.Context, incidentID, tenantID int) error {
	exists, err := s.client.Incident.Query().
		Where(incident.IDEQ(incidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate incident: %w", err)
	}
	if !exists {
		return ErrIncidentNotFound
	}
	return nil
}

// DeleteIncident 软删除事件，保留事件、活动、告警与指标用于审计。
func (s *IncidentService) DeleteIncident(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting incident", "id", id, "tenant_id", tenantID)

	// First verify the incident belongs to the current tenant
	updated, err := s.client.Incident.Update().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		AddVersion(1).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete incident: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("cross-tenant access denied: incident not found")
	}

	s.logger.Infow("Incident deleted successfully", "id", id)
	return nil
}

// CreateIncidentEvent 创建事件活动记录
func (s *IncidentService) CreateIncidentEvent(ctx context.Context, req *dto.CreateIncidentEventRequest, tenantID int) (*dto.IncidentEventResponse, error) {
	s.logger.Infow("Creating incident event", "incident_id", req.IncidentID, "type", req.EventType)
	if err := s.ensureActiveIncident(ctx, req.IncidentID, tenantID); err != nil {
		return nil, err
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}

	eventBuilder := s.client.IncidentEvent.Create().
		SetIncidentID(req.IncidentID).
		SetEventType(req.EventType).
		SetEventName(req.EventName).
		SetDescription(req.Description).
		SetStatus(req.Status).
		SetSeverity(req.Severity).
		SetData(req.Data).
		SetOccurredAt(occurredAt).
		SetSource(req.Source).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if req.UserID != nil {
		eventBuilder.SetUserID(*req.UserID)
	}

	event, err := eventBuilder.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident event", "error", err)
		return nil, fmt.Errorf("failed to create incident event: %w", err)
	}

	s.logger.Infow("Incident event created successfully", "id", event.ID)
	return s.toIncidentEventResponse(event), nil
}

func createIncidentEventTx(ctx context.Context, tx *ent.Tx, req *dto.CreateIncidentEventRequest, tenantID int) error {
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(req.IncidentID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("validate incident for event: %w", err)
	}
	if incidentEntity.TenantID != tenantID {
		return fmt.Errorf("validate incident for event: incident does not belong to tenant")
	}
	if req.UserID != nil {
		actorExists, err := tx.User.Query().
			Where(user.IDEQ(*req.UserID), user.TenantIDEQ(incidentEntity.TenantID), user.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("validate incident event actor: %w", err)
		}
		if !actorExists {
			return fmt.Errorf("validate incident event actor: actor not found in tenant or inactive")
		}
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}
	builder := tx.IncidentEvent.Create().SetIncidentID(req.IncidentID).SetEventType(req.EventType).
		SetEventName(req.EventName).SetDescription(req.Description).SetStatus(req.Status).
		SetSeverity(req.Severity).SetData(req.Data).SetOccurredAt(occurredAt).SetSource(req.Source).
		SetMetadata(req.Metadata).SetTenantID(tenantID).SetCreatedAt(time.Now()).SetUpdatedAt(time.Now())
	if req.UserID != nil {
		builder.SetUserID(*req.UserID)
	}
	_, err = builder.Save(ctx)
	return err
}

func createIncidentAlertTx(ctx context.Context, tx *ent.Tx, req *dto.CreateIncidentAlertRequest, tenantID int) error {
	incidentExists, err := tx.Incident.Query().
		Where(incident.IDEQ(req.IncidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("validate incident for alert: %w", err)
	}
	if !incidentExists {
		return ErrIncidentNotFound
	}

	triggeredAt := time.Now()
	if req.TriggeredAt != nil {
		triggeredAt = *req.TriggeredAt
	}
	_, err = tx.IncidentAlert.Create().
		SetIncidentID(req.IncidentID).
		SetAlertType(req.AlertType).
		SetAlertName(req.AlertName).
		SetMessage(req.Message).
		SetSeverity(req.Severity).
		SetStatus("active").
		SetChannels(req.Channels).
		SetRecipients(req.Recipients).
		SetTriggeredAt(triggeredAt).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// CreateIncidentAlert 创建事件告警
func (s *IncidentService) CreateIncidentAlert(ctx context.Context, req *dto.CreateIncidentAlertRequest, tenantID int) (*dto.IncidentAlertResponse, error) {
	s.logger.Infow("Creating incident alert", "incident_id", req.IncidentID, "type", req.AlertType)
	if err := s.ensureActiveIncident(ctx, req.IncidentID, tenantID); err != nil {
		return nil, err
	}

	triggeredAt := time.Now()
	if req.TriggeredAt != nil {
		triggeredAt = *req.TriggeredAt
	}

	alert, err := s.client.IncidentAlert.Create().
		SetIncidentID(req.IncidentID).
		SetAlertType(req.AlertType).
		SetAlertName(req.AlertName).
		SetMessage(req.Message).
		SetSeverity(req.Severity).
		SetStatus("active").
		SetChannels(req.Channels).
		SetRecipients(req.Recipients).
		SetTriggeredAt(triggeredAt).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident alert", "error", err)
		return nil, fmt.Errorf("failed to create incident alert: %w", err)
	}

	s.logger.Infow("Incident alert created successfully", "id", alert.ID)
	return s.toIncidentAlertResponse(alert), nil
}

// CreateIncidentMetric 创建事件指标
func (s *IncidentService) CreateIncidentMetric(ctx context.Context, req *dto.CreateIncidentMetricRequest, tenantID int) (*dto.IncidentMetricResponse, error) {
	s.logger.Infow("Creating incident metric", "incident_id", req.IncidentID, "type", req.MetricType)
	if err := s.ensureActiveIncident(ctx, req.IncidentID, tenantID); err != nil {
		return nil, err
	}

	measuredAt := time.Now()
	if req.MeasuredAt != nil {
		measuredAt = *req.MeasuredAt
	}

	metric, err := s.client.IncidentMetric.Create().
		SetIncidentID(req.IncidentID).
		SetMetricType(req.MetricType).
		SetMetricName(req.MetricName).
		SetMetricValue(req.MetricValue).
		SetUnit(req.Unit).
		SetMeasuredAt(measuredAt).
		SetTags(req.Tags).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident metric", "error", err)
		return nil, fmt.Errorf("failed to create incident metric: %w", err)
	}

	s.logger.Infow("Incident metric created successfully", "id", metric.ID)
	return s.toIncidentMetricResponse(metric), nil
}

// GetIncidentMonitoring 获取事件监控数据
func (s *IncidentService) GetIncidentMonitoring(ctx context.Context, req *dto.IncidentMonitoringRequest, tenantID int) (*dto.IncidentMonitoringResponse, error) {
	s.logger.Infow("Getting incident monitoring data", "tenant_id", tenantID)

	// 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
	}

	query := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
			incident.CreatedAtGTE(startTime),
			incident.CreatedAtLTE(endTime),
		)

	// 应用过滤器
	if req.IncidentID != nil {
		query = query.Where(incident.IDEQ(*req.IncidentID))
	}
	if req.Category != nil {
		query = query.Where(incident.CategoryEQ(*req.Category))
	}
	if req.Priority != nil {
		query = query.Where(incident.PriorityEQ(*req.Priority))
	}
	if req.Status != nil {
		query = query.Where(incident.StatusEQ(*req.Status))
	}

	// 获取事件列表
	incidents, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incidents", "error", err)
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	// 计算统计数据
	totalIncidents := len(incidents)
	var openIncidents, resolvedIncidents, closedIncidents, criticalIncidents, highPriorityIncidents int
	var totalResolutionTime float64
	var resolvedCount int

	for _, incidentEntity := range incidents {
		switch incidentEntity.Status {
		case "new", "in_progress":
			openIncidents++
		case "resolved":
			resolvedIncidents++
			if !incidentEntity.ResolvedAt.IsZero() {
				resolutionTime := incidentEntity.ResolvedAt.Sub(incidentEntity.CreatedAt).Hours()
				totalResolutionTime += resolutionTime
				resolvedCount++
			}
		case "closed":
			closedIncidents++
		}

		if incidentEntity.Severity == "critical" {
			criticalIncidents++
		}
		if incidentEntity.Priority == "high" || incidentEntity.Priority == "urgent" {
			highPriorityIncidents++
		}
	}

	// 计算平均解决时间
	var averageResolutionTime float64
	if resolvedCount > 0 {
		averageResolutionTime = totalResolutionTime / float64(resolvedCount)
	}

	// 计算解决率
	var resolutionRate float64
	if totalIncidents > 0 {
		resolutionRate = float64(resolvedIncidents+closedIncidents) / float64(totalIncidents) * 100
	}

	// 计算升级率
	var escalationRate float64
	if totalIncidents > 0 {
		var escalatedCount int
		for _, incidentEntity := range incidents {
			if incidentEntity.EscalationLevel > 0 {
				escalatedCount++
			}
		}
		escalationRate = float64(escalatedCount) / float64(totalIncidents) * 100
	}

	// 构建响应
	response := &dto.IncidentMonitoringResponse{
		TotalIncidents:        totalIncidents,
		OpenIncidents:         openIncidents,
		ResolvedIncidents:     resolvedIncidents,
		ClosedIncidents:       closedIncidents,
		CriticalIncidents:     criticalIncidents,
		HighPriorityIncidents: highPriorityIncidents,
		AverageResolutionTime: averageResolutionTime,
		ResolutionRate:        resolutionRate,
		EscalationRate:        escalationRate,
	}

	// 转换事件列表
	response.Incidents = make([]dto.IncidentResponse, len(incidents))
	for i, incidentEntity := range incidents {
		response.Incidents[i] = *s.toIncidentResponse(incidentEntity)
	}

	return response, nil
}

// EscalateIncident 升级事件
func (s *IncidentService) EscalateIncident(ctx context.Context, req *dto.IncidentEscalationRequest, tenantID int) (*dto.IncidentEscalationResponse, error) {
	s.logger.Infow("Escalating incident", "incident_id", req.IncidentID, "level", req.EscalationLevel)
	if req.EscalationLevel < 1 || req.EscalationLevel > 5 {
		return nil, ErrIncidentEscalationLevelInvalid
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, ErrIncidentEscalationReasonRequired
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin incident escalation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 在同一事务内读取并以版本条件更新，升级、事件和告警必须共同成功。
	current, err := tx.Incident.Query().
		Where(
			incident.IDEQ(req.IncidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	if current.Status == common.IncidentStatusResolved || current.Status == common.IncidentStatusClosed || current.Status == common.IncidentStatusCancelled {
		return nil, ErrIncidentTerminal
	}
	if req.EscalationLevel <= current.EscalationLevel {
		return nil, fmt.Errorf("%w: %d", ErrIncidentEscalationLevelNotGreater, current.EscalationLevel)
	}

	// 更新事件升级信息
	now := time.Now()
	incidentEntity, err := tx.Incident.UpdateOneID(req.IncidentID).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(current.Version)).
		SetEscalationLevel(req.EscalationLevel).
		SetEscalatedAt(now).
		SetUpdatedAt(now).
		AddVersion(1).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrIncidentVersionConflict
		}
		s.logger.Errorw("Failed to escalate incident", "error", err)
		return nil, fmt.Errorf("failed to escalate incident: %w", err)
	}

	// 记录升级活动
	err = createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID:  req.IncidentID,
		EventType:   "escalation",
		EventName:   "事件升级",
		Description: fmt.Sprintf("事件升级到级别 %d: %s", req.EscalationLevel, req.Reason),
		Status:      "active",
		Severity:    "high",
		Data: map[string]interface{}{
			"escalation_level": req.EscalationLevel,
			"reason":           req.Reason,
		},
		Source: "system",
	}, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create escalation event: %w", err)
	}

	// 创建升级告警
	err = createIncidentAlertTx(ctx, tx, &dto.CreateIncidentAlertRequest{
		IncidentID: req.IncidentID,
		AlertType:  "escalation",
		AlertName:  "事件升级告警",
		Message:    fmt.Sprintf("事件 %s 已升级到级别 %d", incidentEntity.IncidentNumber, req.EscalationLevel),
		Severity:   "high",
		Channels:   []string{"email", "sms"},
		Recipients: []string{"manager@company.com"},
		Metadata: map[string]interface{}{
			"escalation_level": req.EscalationLevel,
			"reason":           req.Reason,
		},
	}, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create escalation alert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit incident escalation: %w", err)
	}

	// 构建响应
	response := &dto.IncidentEscalationResponse{
		ID:              incidentEntity.ID,
		IncidentID:      req.IncidentID,
		EscalationLevel: req.EscalationLevel,
		Reason:          req.Reason,
		Status:          "active",
		NotifiedUsers:   req.NotifyUsers,
		AutoAssigned:    req.AutoAssign,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.logger.Infow("Incident escalated successfully", "incident_id", req.IncidentID, "level", req.EscalationLevel)
	return response, nil
}

// generateIncidentNumber 生成事件编号，优先使用 Redis 序列
//
// S-4 补充修复：incident_number 是**全局唯一**约束（不含 tenant_id），而 Redis 序列
// key 只按年月分片、不区分租户来源，一旦某个编号已被其他写入方（含历史数据迁移、
// 其他租户、人工补数）占用，序列返回的候选值就会与之碰撞，导致本次创建直接失败。
// 因此这里对候选编号做全局存在性校验并向前跳号，把"必然失败"降级为"跳过占用号"。
func (s *IncidentService) generateIncidentNumber(ctx context.Context, tenantID int) (string, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	expiredAt := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	key := fmt.Sprintf("sequence:incident:%d%02d", year, month)

	// 优先使用 Redis 序列（原子递增，避免并发重复）
	if s.sequenceService != nil {
		// 最多跳号 maxProbe 次；超过则落到唯一后缀兜底，避免热点月份被长期占用时死循环
		const maxProbe = 20
		for i := 0; i < maxProbe; i++ {
			seq, err := s.sequenceService.GetNextSequenceWithExpiry(ctx, key, expiredAt)
			if err != nil {
				s.logger.Warnw("Redis sequence failed for incident, fallback to DB", "error", err)
				break
			}
			candidate := fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq)

			// 全局（跨租户）存在性校验：命中则继续取下一个序列值
			taken, existErr := s.client.Incident.Query().
				Where(incident.IncidentNumberEQ(candidate)).
				Exist(ctx)
			if existErr != nil {
				// 校验失败不阻断创建，交由数据库唯一约束兜底
				s.logger.Warnw("Incident number existence check failed, accepting candidate",
					"error", existErr, "candidate", candidate)
				return candidate, nil
			}
			if !taken {
				return candidate, nil
			}
			s.logger.Warnw("Incident number already taken globally, probing next",
				"candidate", candidate, "tenant_id", tenantID, "attempt", i+1)
		}
		s.logger.Warnw("Incident number probing exhausted, using unique suffix",
			"tenant_id", tenantID)
		return fmt.Sprintf("INC-%04d%02d-%s", year, month, uniqueFallbackSuffix()), nil
	}

	// 备用方案：数据库查询
	return s.generateIncidentNumberWithDB(ctx, tenantID, year, month)
}

// generateIncidentNumberWithDB 使用数据库查询生成事件编号（备用方案）
// S-4 修复：incident_number 在库中为全局唯一约束（非按租户），原实现按租户各自求 max 后再落库，
// 两租户极易算出同一编号，先建者占用全局唯一值、后者永久创建失败（跨租户死锁）。
// 改为：在事务内对全表当月最大编号加锁（FOR UPDATE SKIP LOCKED）后递增，保证全局单调递增且唯一；
// 当月无记录时用唯一后缀兜底，彻底消除跨租户碰撞。
func (s *IncidentService) generateIncidentNumberWithDB(ctx context.Context, tenantID int, year, month int) (string, error) {
	prefix := fmt.Sprintf("INC-%04d%02d-", year, month)

	if s.rawDB != nil {
		tx, err := s.rawDB.BeginTx(ctx, nil)
		if err == nil {
			// 不加租户过滤：incident_number 全局唯一，必须跨租户协调最大号
			query := `SELECT incident_number FROM incidents WHERE incident_number LIKE $1 AND incident_number IS NOT NULL AND incident_number != '' ORDER BY incident_number DESC LIMIT 1 FOR UPDATE SKIP LOCKED`
			var maxNum string
			scanErr := tx.QueryRowContext(ctx, query, prefix+"%").Scan(&maxNum)
			if scanErr == nil {
				seq := 0
				if idx := strings.LastIndex(maxNum, "-"); idx >= 0 {
					fmt.Sscanf(maxNum[idx+1:], "%d", &seq)
				}
				candidate := fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq+1)
				_ = tx.Commit()
				return candidate, nil
			}
			if scanErr == sql.ErrNoRows {
				// 全表当月无记录：用唯一后缀避免两租户同时落到 000001 而碰撞
				candidate := fmt.Sprintf("INC-%04d%02d-%s", year, month, uniqueFallbackSuffix())
				_ = tx.Commit()
				return candidate, nil
			}
			// 其他查询错误：回滚后走最终兜底
			_ = tx.Rollback()
			s.logger.Warnw("Incident number lock query failed, using random fallback", "error", scanErr)
		} else {
			s.logger.Warnw("Incident number tx begin failed, using random fallback", "error", err)
		}
	}

	// 最终兜底（无 rawDB 或事务异常）：唯一后缀保证全局唯一约束不被打破
	return fmt.Sprintf("INC-%04d%02d-%s", year, month, uniqueFallbackSuffix()), nil
}

func (s *IncidentService) executeIncidentRules(ctx context.Context, incidentID int, tenantID int) {
	s.logger.Infow("Executing incident rules", "incident_id", incidentID)

	// 获取激活的事件规则
	rules, err := s.client.IncidentRule.Query().
		Where(
			incidentrule.TenantIDEQ(tenantID),
			incidentrule.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident rules", "error", err)
		return
	}

	// 获取事件信息
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident", "error", err)
		return
	}

	// 执行每个规则
	for _, rule := range rules {
		if s.evaluateRuleConditions(rule.Conditions, incidentEntity) {
			s.executeRuleActions(ctx, rule, incidentEntity, tenantID)
		}
	}
}

func (s *IncidentService) evaluateRuleConditions(conditions map[string]interface{}, incident *ent.Incident) bool {
	// 简化的条件评估逻辑
	if priority, ok := conditions["priority"].([]string); ok {
		for _, p := range priority {
			if incident.Priority == p {
				return true
			}
		}
	}
	if severity, ok := conditions["severity"].([]string); ok {
		for _, s := range severity {
			if incident.Severity == s {
				return true
			}
		}
	}
	if status, ok := conditions["status"].(string); ok {
		return incident.Status == status
	}
	return false
}

func (s *IncidentService) executeRuleActions(ctx context.Context, rule *ent.IncidentRule, incident *ent.Incident, tenantID int) {
	s.logger.Infow("Executing rule actions", "rule_id", rule.ID, "incident_id", incident.ID)

	// 记录规则执行
	execution, err := s.client.IncidentRuleExecution.Create().
		SetRuleID(rule.ID).
		SetIncidentID(incident.ID).
		SetStatus("running").
		SetStartedAt(time.Now()).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create rule execution", "error", err)
		return
	}

	// 执行动作
	for _, action := range rule.Actions {
		if actionType, ok := action["type"].(string); ok {
			switch actionType {
			case "escalate":
				s.executeEscalationAction(ctx, action, incident, tenantID)
			case "notify":
				s.executeNotificationAction(ctx, action, incident, tenantID)
			case "assign":
				s.executeAssignmentAction(ctx, action, incident, tenantID)
			}
		}
	}

	// 更新规则执行状态
	_, err = s.client.IncidentRuleExecution.UpdateOneID(execution.ID).
		SetStatus("completed").
		SetCompletedAt(time.Now()).
		SetResult("Rule executed successfully").
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update rule execution", "error", err)
	}

	// 更新规则执行次数
	_, err = s.client.IncidentRule.UpdateOneID(rule.ID).
		SetExecutionCount(rule.ExecutionCount + 1).
		SetLastExecutedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update rule execution count", "error", err)
	}
}

func (s *IncidentService) executeEscalationAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	if level, ok := action["level"].(int); ok {
		_, err := s.EscalateIncident(ctx, &dto.IncidentEscalationRequest{
			IncidentID:      incident.ID,
			EscalationLevel: level,
			Reason:          "自动升级",
			NotifyUsers:     []int{},
			AutoAssign:      false,
		}, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to execute escalation action", "error", err)
		}
	}
}

func (s *IncidentService) executeNotificationAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	channels := []string{"email"}
	if ch, ok := action["channels"].([]string); ok {
		channels = ch
	}

	recipients := []string{"admin@company.com"}
	if rec, ok := action["recipients"].([]string); ok {
		recipients = rec
	}

	message := "事件需要关注"
	if msg, ok := action["message"].(string); ok {
		message = msg
	}

	_, err := s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incident.ID,
		AlertType:  "notification",
		AlertName:  "规则触发通知",
		Message:    message,
		Severity:   "medium",
		Channels:   channels,
		Recipients: recipients,
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to execute notification action", "error", err)
	}
}

func (s *IncidentService) executeAssignmentAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	if assigneeID, ok := action["assignee_id"].(int); ok {
		_, err := s.UpdateIncident(ctx, incident.ID, &dto.UpdateIncidentRequest{
			AssigneeID: &assigneeID,
		}, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to execute assignment action", "error", err)
		}
	}
}

// isValidIncidentStatusTransition 检查事件状态转换是否合法。
// 阻断6 修复：委托给 common.IsValidIncidentStatusTransition，保持单一事实来源，
// 避免 service 层与 handlers/incident 层两套白名单漂移。
func isValidIncidentStatusTransition(currentStatus, newStatus string) bool {
	return common.IsValidIncidentStatusTransition(currentStatus, newStatus)
}

// 转换为响应DTO
func (s *IncidentService) toIncidentResponse(incident *ent.Incident) *dto.IncidentResponse {
	return dto.ToIncidentResponse(incident)
}

func (s *IncidentService) toIncidentEventResponse(event *ent.IncidentEvent) *dto.IncidentEventResponse {
	return &dto.IncidentEventResponse{
		ID:          event.ID,
		IncidentID:  event.IncidentID,
		EventType:   event.EventType,
		EventName:   event.EventName,
		Description: event.Description,
		Status:      event.Status,
		Severity:    event.Severity,
		Data:        event.Data,
		OccurredAt:  event.OccurredAt,
		UserID:      &event.UserID,
		Source:      event.Source,
		Metadata:    event.Metadata,
		TenantID:    event.TenantID,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
}

func (s *IncidentService) toIncidentAlertResponse(alert *ent.IncidentAlert) *dto.IncidentAlertResponse {
	return &dto.IncidentAlertResponse{
		ID:             alert.ID,
		IncidentID:     alert.IncidentID,
		AlertType:      alert.AlertType,
		AlertName:      alert.AlertName,
		Message:        alert.Message,
		Severity:       alert.Severity,
		Status:         alert.Status,
		Channels:       alert.Channels,
		Recipients:     alert.Recipients,
		TriggeredAt:    alert.TriggeredAt,
		AcknowledgedAt: &alert.AcknowledgedAt,
		ResolvedAt:     &alert.ResolvedAt,
		AcknowledgedBy: &alert.AcknowledgedBy,
		Metadata:       alert.Metadata,
		TenantID:       alert.TenantID,
		CreatedAt:      alert.CreatedAt,
		UpdatedAt:      alert.UpdatedAt,
	}
}

func (s *IncidentService) toIncidentMetricResponse(metric *ent.IncidentMetric) *dto.IncidentMetricResponse {
	return &dto.IncidentMetricResponse{
		ID:          metric.ID,
		IncidentID:  metric.IncidentID,
		MetricType:  metric.MetricType,
		MetricName:  metric.MetricName,
		MetricValue: metric.MetricValue,
		Unit:        metric.Unit,
		MeasuredAt:  metric.MeasuredAt,
		Tags:        metric.Tags,
		Metadata:    metric.Metadata,
		TenantID:    metric.TenantID,
		CreatedAt:   metric.CreatedAt,
		UpdatedAt:   metric.UpdatedAt,
	}
}

// GetIncidentStats 获取事件统计信息

// AcknowledgeIncident 流转事件状态到 acknowledged
func (s *IncidentService) AcknowledgeIncident(ctx context.Context, id, userID, tenantID int) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start acknowledge transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// 获取当前事件状态进行验证
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return err
	}
	// 验证状态转换是否合法
	if !isValidIncidentStatusTransition(incidentEntity.Status, common.IncidentStatusAcknowledged) {
		return fmt.Errorf("%w: from '%s' to '%s'", ErrIncidentInvalidTransition, incidentEntity.Status, common.IncidentStatusAcknowledged)
	}

	now := time.Now()
	err = tx.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(incidentEntity.Version)).
		SetStatus(common.IncidentStatusAcknowledged).
		SetUpdatedAt(now).
		AddVersion(1).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentVersionConflict
		}
		return err
	}
	if err := createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "acknowledgement", EventName: "事件确认",
		Description: fmt.Sprintf("事件由用户 %d 确认", userID), Status: "active", Severity: "info",
		UserID: &userID, Source: "user",
	}, tenantID); err != nil {
		return fmt.Errorf("create acknowledgement event: %w", err)
	}
	return tx.Commit()
}

// ResolveIncident 流转事件状态到 resolved
func (s *IncidentService) ResolveIncident(ctx context.Context, id, userID, tenantID int, resolution, rootCause string) error {
	if strings.TrimSpace(resolution) == "" {
		return ErrIncidentResolutionRequired
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start resolve transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// 获取当前事件状态进行验证
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return err
	}
	actorExists, err := tx.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(incidentEntity.TenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("validate incident actor: %w", err)
	}
	if !actorExists {
		return fmt.Errorf("validate incident actor: actor not found in tenant or inactive")
	}

	// 验证状态转换是否合法
	if !isValidIncidentStatusTransition(incidentEntity.Status, common.IncidentStatusResolved) {
		return fmt.Errorf("%w: from '%s' to '%s'", ErrIncidentInvalidTransition, incidentEntity.Status, common.IncidentStatusResolved)
	}

	now := time.Now()
	rootCauseData := incidentEntity.RootCause
	if rootCauseData == nil {
		rootCauseData = make(map[string]interface{})
	}
	if strings.TrimSpace(rootCause) != "" {
		rootCauseData["rootCause"] = strings.TrimSpace(rootCause)
		rootCauseData["status"] = "confirmed"
	}
	resolutionSteps := incidentEntity.ResolutionSteps
	resolutionSteps = append(resolutionSteps, map[string]interface{}{
		"step": len(resolutionSteps) + 1, "description": strings.TrimSpace(resolution),
		"executedBy": fmt.Sprintf("%d", userID), "executedAt": now, "status": "completed",
	})
	err = tx.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(incidentEntity.Version)).
		SetStatus(common.IncidentStatusResolved).
		SetResolvedAt(now).
		ClearClosedAt().
		SetRootCause(rootCauseData).
		SetResolutionSteps(resolutionSteps).
		SetUpdatedAt(now).
		AddVersion(1).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentVersionConflict
		}
		return err
	}
	if err := createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "resolution", EventName: "事件解决",
		Description: strings.TrimSpace(resolution), Status: "active", Severity: "info",
		Data:   map[string]interface{}{"rootCause": strings.TrimSpace(rootCause)},
		UserID: &userID, Source: "user",
	}, tenantID); err != nil {
		return fmt.Errorf("create resolution event: %w", err)
	}
	return tx.Commit()
}

// CloseIncident 流转事件状态到 closed
func (s *IncidentService) CloseIncident(ctx context.Context, id, userID, tenantID int, closeNotes string) error {
	if strings.TrimSpace(closeNotes) == "" {
		return ErrIncidentCloseNotesRequired
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start close transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// 获取当前事件状态进行验证
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return err
	}

	// 验证状态转换是否合法
	if !isValidIncidentStatusTransition(incidentEntity.Status, common.IncidentStatusClosed) {
		return fmt.Errorf("%w: from '%s' to '%s'", ErrIncidentInvalidTransition, incidentEntity.Status, common.IncidentStatusClosed)
	}

	now := time.Now()
	err = tx.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(incidentEntity.Version)).
		SetStatus(common.IncidentStatusClosed).
		SetClosedAt(now).
		SetUpdatedAt(now).
		AddVersion(1).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentVersionConflict
		}
		return err
	}
	if err := createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "closure", EventName: "事件关闭",
		Description: strings.TrimSpace(closeNotes), Status: "active", Severity: "info",
		UserID: &userID, Source: "user",
	}, tenantID); err != nil {
		return fmt.Errorf("create closure event: %w", err)
	}
	return tx.Commit()
}

// ReopenIncident 将已解决或已关闭的事件重新流转到 in_progress
func (s *IncidentService) ReopenIncident(ctx context.Context, id, userID, tenantID int) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start reopen transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return err
	}

	if incidentEntity.Status != common.IncidentStatusResolved && incidentEntity.Status != common.IncidentStatusClosed {
		return fmt.Errorf("%w: only resolved or closed incidents can be reopened", ErrIncidentInvalidTransition)
	}

	now := time.Now()
	err = tx.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(incidentEntity.Version)).
		SetStatus(common.IncidentStatusInProgress).
		ClearResolvedAt().
		ClearClosedAt().
		SetUpdatedAt(now).
		AddVersion(1).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentVersionConflict
		}
		return err
	}
	if err := createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "reopen", EventName: "事件重新打开",
		Description: fmt.Sprintf("事件由用户 %d 重新打开", userID), Status: "active", Severity: "info",
		UserID: &userID, Source: "user",
	}, tenantID); err != nil {
		return fmt.Errorf("create reopen event: %w", err)
	}
	return tx.Commit()
}

// EscalateToMajorIncident 将事件升级为重大事件（Major Incident）
// 写入影响评估信息，提升严重程度，并记录审计事件
func (s *IncidentService) EscalateToMajorIncident(ctx context.Context, id, userID, tenantID int, req *dto.EscalateMajorIncidentRequest) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin major incident escalation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	incidentEntity, err := tx.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return err
	}

	if incidentEntity.IsMajorIncident {
		return fmt.Errorf("incident is already a major incident")
	}
	if incidentEntity.Status == common.IncidentStatusResolved || incidentEntity.Status == common.IncidentStatusClosed {
		return fmt.Errorf("resolved or closed incidents cannot be escalated to major incident")
	}

	now := time.Now()
	impactAnalysis := incidentEntity.ImpactAnalysis
	if impactAnalysis == nil {
		impactAnalysis = make(map[string]interface{})
	}
	impactAnalysis["majorIncident"] = map[string]interface{}{
		"impactScope":       req.ImpactScope,
		"businessImpact":    strings.TrimSpace(req.BusinessImpact),
		"communicationPlan": strings.TrimSpace(req.CommunicationPlan),
		"escalatedBy":       userID,
		"escalatedAt":       now,
	}

	err = tx.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.VersionEQ(incidentEntity.Version)).
		SetIsMajorIncident(true).
		SetSeverity("critical").
		SetImpactAnalysis(impactAnalysis).
		SetEscalatedAt(now).
		AddEscalationLevel(1).
		SetUpdatedAt(now).
		AddVersion(1).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrIncidentVersionConflict
		}
		return err
	}
	if err := createIncidentEventTx(ctx, tx, &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "major_incident_escalation", EventName: "升级为重大事件",
		Description: strings.TrimSpace(req.BusinessImpact), Status: "active", Severity: "critical",
		Data: map[string]interface{}{
			"impactScope":       req.ImpactScope,
			"communicationPlan": strings.TrimSpace(req.CommunicationPlan),
		},
		UserID: &userID, Source: "user",
	}, tenantID); err != nil {
		return fmt.Errorf("create major incident event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit major incident escalation: %w", err)
	}
	return nil
}

func (s *IncidentService) GetIncidentStats(ctx context.Context, tenantID int) (*dto.IncidentStatsResponse, error) {
	s.logger.Infow("Getting incident stats", "tenant_id", tenantID)

	// 获取总事件数
	totalIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total incidents", "error", err)
		return nil, fmt.Errorf("failed to count total incidents: %w", err)
	}

	// 获取开放事件数（new, in_progress）
	openIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
			incident.StatusIn("new", "acknowledged", "assigned", "triaged", "in_progress", "on_hold", "escalated"),
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count open incidents", "error", err)
		return nil, fmt.Errorf("failed to count open incidents: %w", err)
	}

	// 获取关键事件数（severity = critical）
	criticalIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil(), incident.SeverityEQ("critical")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count critical incidents", "error", err)
		return nil, fmt.Errorf("failed to count critical incidents: %w", err)
	}

	// 获取主要事件数（使用 severity = critical 或 priority = high/urgent 作为主要事件）
	majorIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
			incident.Or(
				incident.SeverityEQ("critical"),
				incident.PriorityIn("high", "urgent"),
			),
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count major incidents", "error", err)
		return nil, fmt.Errorf("failed to count major incidents: %w", err)
	}

	// 获取已解决的事件，计算平均解决时间
	resolvedIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
			incident.StatusEQ("resolved"),
			incident.ResolvedAtNotNil(),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get resolved incidents", "error", err)
		return nil, fmt.Errorf("failed to get resolved incidents: %w", err)
	}

	var totalResolutionTime float64
	var totalAcknowledgeTime float64
	resolvedCount := len(resolvedIncidents)
	acknowledgedCount := 0

	for _, inc := range resolvedIncidents {
		if !inc.ResolvedAt.IsZero() && !inc.DetectedAt.IsZero() {
			resolutionTime := inc.ResolvedAt.Sub(inc.DetectedAt).Hours()
			totalResolutionTime += resolutionTime
		}
		// 使用 detected_at 到 created_at 的时间差作为确认时间（简化实现）
		if !inc.DetectedAt.IsZero() && !inc.CreatedAt.IsZero() {
			acknowledgeTime := inc.DetectedAt.Sub(inc.CreatedAt).Hours()
			if acknowledgeTime > 0 {
				totalAcknowledgeTime += acknowledgeTime
				acknowledgedCount++
			}
		}
	}

	var avgResolutionTime float64
	if resolvedCount > 0 {
		avgResolutionTime = totalResolutionTime / float64(resolvedCount)
	}

	var mtta float64
	if acknowledgedCount > 0 {
		mtta = totalAcknowledgeTime / float64(acknowledgedCount)
	}

	var mttr float64 = avgResolutionTime

	return &dto.IncidentStatsResponse{
		TotalIncidents:    totalIncidents,
		OpenIncidents:     openIncidents,
		CriticalIncidents: criticalIncidents,
		MajorIncidents:    majorIncidents,
		AvgResolutionTime: avgResolutionTime,
		MTTA:              mtta,
		MTTR:              mttr,
	}, nil
}

// GetIncidentEvents 获取指定事件的活动记录
func (s *IncidentService) GetIncidentEvents(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentEventResponse, error) {
	s.logger.Infow("Getting incident events", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的活动记录
	events, err := s.client.IncidentEvent.Query().
		Where(
			incidentevent.IncidentIDEQ(incident.ID),
			incidentevent.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident events: %w", err)
	}

	responses := make([]dto.IncidentEventResponse, len(events))
	for i, event := range events {
		responses[i] = *s.toIncidentEventResponse(event)
	}

	return responses, nil
}

// GetIncidentAlerts 获取指定事件的告警
func (s *IncidentService) GetIncidentAlerts(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentAlertResponse, error) {
	s.logger.Infow("Getting incident alerts", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的告警
	alerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.IncidentIDEQ(incident.ID),
			incidentalert.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident alerts: %w", err)
	}

	responses := make([]dto.IncidentAlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = *s.toIncidentAlertResponse(alert)
	}

	return responses, nil
}

// GetIncidentMetrics 获取指定事件的指标
func (s *IncidentService) GetIncidentMetrics(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentMetricResponse, error) {
	s.logger.Infow("Getting incident metrics", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的指标
	metrics, err := s.client.IncidentMetric.Query().
		Where(
			incidentmetric.IncidentIDEQ(incident.ID),
			incidentmetric.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident metrics: %w", err)
	}

	responses := make([]dto.IncidentMetricResponse, len(metrics))
	for i, metric := range metrics {
		responses[i] = *s.toIncidentMetricResponse(metric)
	}

	return responses, nil
}

// triggerWorkflowForIncident 为事件触发工作流
func (s *IncidentService) triggerWorkflowForIncident(ctx context.Context, incidentID int, tenantID int) error {
	// 获取事件信息
	inc, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get incident: %w", err)
	}

	// 构建流程变量
	variables := map[string]interface{}{
		"incident_id":     inc.ID,
		"incident_number": inc.IncidentNumber,
		"title":           inc.Title,
		"description":     inc.Description,
		"priority":        inc.Priority,
		"severity":        inc.Severity,
		"status":          inc.Status,
		"category":        inc.Category,
		"reporter_id":     inc.ReporterID,
		"assignee_id":     inc.AssigneeID,
	}

	// 根据严重程度选择不同的流程
	// 注意: incident_general_flow 不存在，使用 incident_emergency_flow 作为默认
	processKey := "incident_emergency_flow"
	if inc.Severity == "critical" || inc.Priority == "urgent" {
		processKey = "incident_emergency_flow"
	}

	// 触发流程
	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeIncident,
		BusinessID:           incidentID,
		ProcessDefinitionKey: processKey,
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", inc.ReporterID),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerService.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow(
		"Workflow triggered for incident",
		"incident_id", incidentID,
		"process_instance_id", resp.ProcessInstanceID,
		"process_key", processKey,
	)

	return nil
}

// GetWorkflowStatus 获取事件关联的流程状态
func (s *IncidentService) GetWorkflowStatus(ctx context.Context, incidentID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	businessKey := fmt.Sprintf("incident:%d", incidentID)

	// 直接查询流程实例
	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("未找到事件关联的流程实例")
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
func (s *IncidentService) mapProcessStatus(status string) dto.ProcessStatus {
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
