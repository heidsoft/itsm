package ticket

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/internal/datascope"
	repoTicket "itsm-backend/repository/ticket"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// ErrTicketInvalidTransition 是工单状态机非法迁移的哨兵错误。
// 与 incident 领域的 ErrIncidentInvalidTransition 对齐：service 层 wrap 后返回，
// handler 层通过 errors.Is 识别并映射为 409 Conflict + BusinessError。
var ErrTicketInvalidTransition = fmt.Errorf("invalid ticket status transition")

// Service handles ticket business logic.
// All operations go through the Repository interface (tenant-isolated).
type Service struct {
	repo          Repository
	productionSvc *service.TicketService
	slaMonitor    *service.SLAMonitorService
	logger        *zap.SugaredLogger
}

// NewService creates a new ticket service.
func NewService(repo Repository, productionSvc *service.TicketService, slaMonitor *service.SLAMonitorService, logger *zap.SugaredLogger) *Service {
	return &Service{repo: repo, productionSvc: productionSvc, slaMonitor: slaMonitor, logger: logger}
}

// PauseSLA 暂停工单 SLA 计时，落到 SLA 领域的权威实现。
func (s *Service) PauseSLA(ctx context.Context, tenantID, ticketID int, reason string) error {
	if s.slaMonitor == nil {
		return common.NewBusinessError(common.ServiceUnavailableCode, "SLA timing service is not connected", "")
	}
	return s.slaMonitor.PauseSLA(ctx, tenantID, "ticket", ticketID, reason)
}

// ResumeSLA 恢复工单 SLA 计时（暂停时长顺延截止时间）。
func (s *Service) ResumeSLA(ctx context.Context, tenantID, ticketID int) error {
	if s.slaMonitor == nil {
		return common.NewBusinessError(common.ServiceUnavailableCode, "SLA timing service is not connected", "")
	}
	return s.slaMonitor.ResumeSLA(ctx, tenantID, "ticket", ticketID)
}

// Create creates a new ticket.
// Delegates to productionSvc.CreateTicket which handles workflow outbox,
// SLA, notifications, automation rules, and other side effects.
func (s *Service) Create(ctx context.Context, tenantID int, params *CreateParams) (*Ticket, error) {
	s.logger.Infow("Creating ticket", "title", params.Title, "tenant_id", tenantID)

	if params.RequesterID == 0 {
		return nil, common.NewBadRequestError("requester_id is required", nil)
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}
	if params.Type == "" {
		params.Type = "incident"
	}

	// Build the DTO for the production service which owns workflow/SLA/outbox logic.
	req := &dto.CreateTicketRequest{
		Title:                 params.Title,
		Description:           params.Description,
		Type:                  params.Type,
		Priority:              params.Priority,
		RequesterID:           params.RequesterID,
		FormFields:            params.FormFields,
		TagIDs:                params.TagIDs,
		WorkflowDefinitionKey: params.WorkflowDefinitionKey,
	}
	if params.AssigneeID != nil {
		req.AssigneeID = *params.AssigneeID
	}
	if params.TicketTypeID != nil {
		req.TicketTypeID = params.TicketTypeID
	}
	if params.CategoryID != nil {
		req.CategoryID = params.CategoryID
	}
	if params.TemplateID != nil {
		req.TemplateID = params.TemplateID
	}
	if params.ParentTicketID != nil {
		req.ParentTicketID = params.ParentTicketID
	}

	// productionSvc 为 nil 时（单元测试 harness 场景）降级为纯 repo 创建，
	// 跳过工作流/SLA/通知等生产侧副作用——生产装配恒传非 nil。
	if s.productionSvc == nil {
		return s.repo.Create(ctx, params, tenantID)
	}

	created, err := s.productionSvc.CreateTicket(ctx, req, tenantID)
	if err != nil {
		return nil, err
	}

	return productionTicketToTicket(created), nil
}

// productionTicketToTicket 把 productionSvc 的创建结果转为 handler 层实体供响应映射。
// 漏拷的字段会被 TicketResponse 的 omitempty 直接从 API 响应里吞掉
// （ticketTypeId 曾因此从 POST /api/v1/tickets 响应中消失），新增字段必须同步。
func productionTicketToTicket(t *repoTicket.Ticket) *Ticket {
	return &Ticket{
		ID:             t.ID,
		TicketNumber:   t.TicketNumber,
		Title:          t.Title,
		Description:    t.Description,
		Status:         string(t.Status),
		Priority:       string(t.Priority),
		Type:           string(t.Type),
		TicketTypeID:   t.TicketTypeID,
		TicketTypeCode: t.TicketTypeCode,
		TicketTypeName: t.TicketTypeName,
		FormFields:     t.FormFields,
		RequesterID:    t.RequesterID,
		AssigneeID:     t.AssigneeID,
		TenantID:       t.TenantID,
		Version:        t.Version,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

// Get retrieves a ticket by ID.
func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetByNumber retrieves a ticket by its business ticket number (e.g. TKT-202609-000010).
// 用于让 GET /api/v1/tickets/:id 同时支持数字 ID 与业务工单号。
func (s *Service) GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error) {
	return s.repo.GetByNumber(ctx, ticketNumber, tenantID)
}

// dataScopeForRole 返回某角色在工单读取上的行级数据范围。
// 判定已收敛到 datascope.ForTicketReader，工作台 KPI 与工单列表/统计共用同一规则。
func dataScopeForRole(currentRole string) datascope.DataScope {
	return datascope.ForTicketReader(currentRole)
}

// List lists tickets with pagination and filtering.
func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	return s.repo.List(ctx, tenantID, page, size, filters, dataScopeForRole(currentRole), currentUserID)
}

// LoadUserNames 批量投影用户显示名，供响应里的 requesterName/assigneeName 使用。
// 姓名只是展示增强：查询失败返回空映射，让响应退回用户 ID，而不是整页报错。
func (s *Service) LoadUserNames(ctx context.Context, tenantID int, ids []int) map[int]string {
	if s.repo == nil {
		return map[int]string{}
	}
	names, err := s.repo.GetUserNamesByIDs(ctx, tenantID, ids)
	if err != nil {
		s.logger.Warnw("加载用户显示名失败", "tenant_id", tenantID, "error", err)
		return map[int]string{}
	}
	return names
}

// Update updates a ticket.
// 写路径行级校验——写权限 ⊆ 读权限，普通角色仅可修改
// 本人创建或受理的工单（datascope.CanWriteResource）。
func (s *Service) Update(ctx context.Context, tenantID int, id int, params *UpdateParams, actorID int, actorRole string) (*Ticket, error) {
	// Fetch current ticket to get version for optimistic locking
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return nil, common.NewForbiddenError("No permission to modify this work order: only the creator, assignee or administrator can operate")
	}
	if params.Version == 0 {
		params.Version = current.Version
	}
	if params.Status != nil && !common.IsValidTicketStatusTransition(current.Status, *params.Status) {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to '%s'", ErrTicketInvalidTransition, current.Status, *params.Status))
	}
	// params.AssigneeID 可指向任意租户的用户，必须由 productionSvc 做同租户校验。
	if params.AssigneeID != nil && *params.AssigneeID > 0 && s.productionSvc != nil {
		if err := s.productionSvc.ValidateAssignee(ctx, *params.AssigneeID, tenantID); err != nil {
			return nil, err
		}
	}

	// PATCH 语义下客户端常回传整个表单。未变化的字段必须从落库参数里摘掉：
	// 仓储对 status 无条件 SetStatus 并重置 resolved_at/closed_at，回传当前状态
	// 会把已解决工单的解决时间改成此刻；全量无变化时更应直接返回，不推进 version。
	if !params.ReplaceTags {
		params = dropUnchangedUpdateParams(params, current)
		// 版本不一致时不能短路：无变化的载荷同样要返回 409，否则过期客户端
		// 可以用一次空更新「成功」掩盖自己读到的是旧数据。
		if isNoChangeUpdate(params) && params.Version == current.Version {
			return current, nil
		}
	}

	updated, err := s.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update ticket: %w", err)
	}
	return updated, nil
}

// dropUnchangedUpdateParams 把与当前值相同的字段置空，只让真正的变化落库。
func dropUnchangedUpdateParams(params *UpdateParams, current *Ticket) *UpdateParams {
	if params.Title != nil && *params.Title == current.Title {
		params.Title = nil
	}
	if params.Description != nil && *params.Description == current.Description {
		params.Description = nil
	}
	if params.Status != nil && *params.Status == current.Status {
		params.Status = nil
	}
	if params.Priority != nil && *params.Priority == current.Priority {
		params.Priority = nil
	}
	if params.Type != nil && *params.Type == current.Type {
		params.Type = nil
	}
	if params.AssigneeID != nil && assigneeIDValue(current.AssigneeID) == *params.AssigneeID {
		params.AssigneeID = nil
	}
	if params.CategoryID != nil && categoryIDValue(current.CategoryID) == *params.CategoryID {
		params.CategoryID = nil
	}
	if params.Resolution != nil && current.Resolution != nil && *params.Resolution == *current.Resolution {
		params.Resolution = nil
	}
	if params.FormFields != nil && formFieldsEqual(*params.FormFields, current.FormFields) {
		params.FormFields = nil
	}
	return params
}

func isNoChangeUpdate(params *UpdateParams) bool {
	return params.Title == nil && params.Description == nil && params.Status == nil &&
		params.Priority == nil && params.Type == nil && params.AssigneeID == nil &&
		params.CategoryID == nil && params.Resolution == nil && params.FormFields == nil &&
		!params.ReplaceTags && len(params.TagIDs) == 0
}

func categoryIDValue(id *int) int {
	if id == nil {
		return 0
	}
	return *id
}

func assigneeIDValue(id *int) int {
	if id == nil {
		return 0
	}
	return *id
}

func formFieldsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !reflect.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}

// Delete soft-deletes a ticket.
// 删除同样受行级写权限约束，不能只校验租户隔离。
func (s *Service) Delete(ctx context.Context, id int, tenantID int, actorID int, actorRole string) error {
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return common.NewForbiddenError("No permission to delete this work order: only the creator, assignee or administrator can operate it")
	}
	return s.repo.Delete(ctx, id, tenantID)
}

// BatchDelete deletes multiple tickets.
// 逐单行级校验，任一单据无权限则整体拒绝（全有或全无），
// 避免批量接口被用来越权删除他人工单。
func (s *Service) BatchDelete(ctx context.Context, ids []int, tenantID int, actorID int, actorRole string) error {
	if !datascope.IsDataScopeAllRole(actorRole) {
		for _, id := range ids {
			current, err := s.repo.GetByID(ctx, id, tenantID)
			if err != nil {
				return err
			}
			if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
				return common.NewForbiddenError(fmt.Sprintf("No permission to delete work order %d: only the creator, assignee or administrator can operate", id))
			}
		}
	}
	return s.repo.BatchDelete(ctx, ids, tenantID)
}

// AssignTicket assigns a ticket to a user.
// productionSvc 为 nil（仅单测 harness）时退回裸 repo，同租户/状态机/CAS/副作用校验一并跳过。
func (s *Service) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int) (*Ticket, error) {
	if s.productionSvc == nil {
		return s.repo.AssignTicket(ctx, ticketID, assigneeID, tenantID)
	}
	updated, err := s.productionSvc.AssignTicket(ctx, ticketID, assigneeID, tenantID)
	if err != nil {
		return nil, err
	}
	return productionTicketToTicket(updated), nil
}

// EscalateTicket escalates a ticket.
// 升级是生命周期写操作，与 Update/Delete 同风险面——行级校验
// 写权限 ⊆ 读权限（普通角色仅创建人/受理人/管理员）。
func (s *Service) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int, actorRole string) (*Ticket, error) {
	current, err := s.lifecycleGuard(ctx, ticketID, escalatedBy, actorRole, tenantID, "升级")
	if err != nil {
		return nil, err
	}
	if !common.IsValidTicketStatusTransition(current.Status, "in_progress") {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to 'in_progress'", ErrTicketInvalidTransition, current.Status))
	}
	return s.repo.EscalateTicket(ctx, ticketID, reason, tenantID, escalatedBy)
}

// ResolveTicket resolves a ticket with a resolution note.
// 生命周期写操作做行级校验，不能只校验租户隔离。
func (s *Service) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int, actorID int, actorRole string) (*Ticket, error) {
	current, err := s.lifecycleGuard(ctx, ticketID, actorID, actorRole, tenantID, "解决")
	if err != nil {
		return nil, err
	}
	if !common.IsValidTicketStatusTransition(current.Status, "resolved") {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to 'resolved'", ErrTicketInvalidTransition, current.Status))
	}
	return s.repo.ResolveTicket(ctx, ticketID, resolution, tenantID)
}

// CloseTicket closes a ticket.
// 生命周期写操作做行级校验，不能只校验租户隔离。
func (s *Service) CloseTicket(ctx context.Context, ticketID int, tenantID int, actorID int, actorRole string) (*Ticket, error) {
	current, err := s.lifecycleGuard(ctx, ticketID, actorID, actorRole, tenantID, "关闭")
	if err != nil {
		return nil, err
	}
	if !common.IsValidTicketStatusTransition(current.Status, "closed") {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to 'closed'", ErrTicketInvalidTransition, current.Status))
	}
	return s.repo.CloseTicket(ctx, ticketID, tenantID)
}

// UpdateStatus updates the status of a ticket.
// 状态流转是生命周期写操作，行级校验对齐 Update/Delete
// （写权限 ⊆ 读权限，普通角色仅创建人/受理人/管理员）。
func (s *Service) UpdateStatus(ctx context.Context, ticketID int, status string, tenantID int, userID int, actorRole string) (*Ticket, error) {
	current, err := s.lifecycleGuard(ctx, ticketID, userID, actorRole, tenantID, "变更状态")
	if err != nil {
		return nil, err
	}
	if !common.IsValidTicketStatusTransition(current.Status, status) {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to '%s'", ErrTicketInvalidTransition, current.Status, status))
	}
	return s.repo.UpdateStatus(ctx, ticketID, status, tenantID)
}

// lifecycleGuard 是 resolve/close/escalate/updateStatus 共用的行级守卫。
// 与 Update/Delete 的 CanWriteResource 校验语义一致：加载单据后校验
// 写权限 ⊆ 读权限，拒绝返回 403 Forbidden AppError。
// 返回加载的单据以避免调用方重复查询（调用方还需要当前状态做状态机校验）。
func (s *Service) lifecycleGuard(ctx context.Context, ticketID, actorID int, actorRole string, tenantID int, action string) (*Ticket, error) {
	current, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return nil, common.NewForbiddenError(fmt.Sprintf("No permission %s for this work order: only the creator, assignee or administrator can operate it", action))
	}
	return current, nil
}

// lifecycleError 将状态机哨兵错误映射为 409 BusinessError，
// 与 incident 领域的 lifecycleError 对齐。
func lifecycleError(err error) error {
	if errors.Is(err, ErrTicketInvalidTransition) {
		return common.NewBusinessError(common.ConflictCode, "The current work order status does not allow this operation", "")
	}
	return err
}

// Search searches tickets by keyword.
// currentUserID/currentRole 与 List 同源：关键字搜索只是多一个过滤条件，
// 行级范围必须一致，否则普通角色可借 /tickets/search 读到全租户工单。
func (s *Service) Search(ctx context.Context, keyword string, tenantID int, page, size int, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	return s.repo.Search(ctx, keyword, tenantID, page, size, dataScopeForRole(currentRole), currentUserID)
}

// GetOverdueTickets returns overdue tickets.
// 同 Search：逾期列表是工单列表的过滤视图，沿用同一套 DataScope。
func (s *Service) GetOverdueTickets(ctx context.Context, tenantID int, page, size int, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	return s.repo.FindOverdue(ctx, tenantID, page, size, dataScopeForRole(currentRole), currentUserID)
}

// GetTicketsByAssignee returns tickets assigned to a specific user.
func (s *Service) GetTicketsByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error) {
	return s.repo.FindByAssignee(ctx, assigneeID, tenantID)
}

// GetStats 返回登录者数据范围内的工单统计，是工单统计的唯一入口。
//
// window 为 nil 时保持旧口径：统计可见范围内全部工单，不带趋势与 SLA 读数
// （工作台卡片依赖这一点，不能在这里塞默认窗口让 KPI 凭空变小）。
// window 非 nil 时计数器、超期数、分布与趋势共用同一个 created_at 窗口，
// 并按租户全量补一份 SLA 达标读数。
func (s *Service) GetStats(ctx context.Context, tenantID int, actorID int, actorRole string, window *repoTicket.StatsWindow) (*TicketStats, error) {
	scope := dataScopeForRole(actorRole)
	stats, err := s.repo.GetStats(ctx, tenantID, scope, actorID, window)
	if err != nil {
		return nil, err
	}
	stats.Scope = StatsScopeLabel(scope)
	stats.Window = StatsWindowEcho(window)
	if window != nil {
		s.attachSLACompliance(ctx, tenantID, *window, stats)
	}
	return stats, nil
}

// attachSLACompliance 补齐窗口内的 SLA 达标读数，失败时保持 nil。
//
// nil 与 0% 必须区分：达标率 0 会被读成「这个租户所有工单都超时」，
// 而计时服务未接入/查询失败是运维问题，前端要显示「不可用」而不是假数字。
func (s *Service) attachSLACompliance(ctx context.Context, tenantID int, window repoTicket.StatsWindow, stats *TicketStats) {
	if s.slaMonitor == nil {
		return
	}
	metrics, err := s.slaMonitor.CalculateSLAMetrics(ctx, tenantID, window.From, window.To)
	if err != nil {
		s.logger.Warnw("计算工单 SLA 达标率失败", "tenant_id", tenantID, "error", err)
		return
	}
	echo := StatsWindowEcho(&window)
	windowDays := 0
	if echo != nil {
		windowDays = echo.Days
	}
	stats.SLA = &TicketSLACompliance{
		ComplianceRate:     metrics.SLACompliance,
		SampledTickets:     metrics.TotalTickets,
		ViolatedTickets:    metrics.ViolatedTickets,
		AvgResponseMinutes: metrics.AvgResponseMinutes,
		AvgResolutionHours: metrics.AvgResolutionHours,
		WindowDays:         windowDays,
		Scope:              SLAScopeTenant,
	}
}

// ExportTickets 导出工单，委派给生产服务层（其查询已带 TenantID 谓词）。
func (s *Service) ExportTickets(ctx context.Context, tenantID int, filters map[string]interface{}, format string) ([]byte, error) {
	if s.productionSvc == nil {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, "The work order export service is not connected", "")
	}
	return s.productionSvc.ExportTickets(ctx, tenantID, filters, format)
}

// ImportTickets imports tickets (placeholder).
func (s *Service) ImportTickets(ctx context.Context, tenantID int, data []byte, format string) error {
	return fmt.Errorf("import not implemented in handlers layer")
}

// AssignTickets assigns multiple tickets to a user.
func (s *Service) AssignTickets(ctx context.Context, tenantID int, ticketIDs []int, assigneeID int) error {
	for _, id := range ticketIDs {
		if _, err := s.AssignTicket(ctx, id, assigneeID, tenantID); err != nil {
			return err
		}
	}
	return nil
}

// ==================== 工单模板 ====================
//
// 模板的业务规则与持久化唯一所有者是 productionSvc（service.TicketService 的
// 模板 CRUD，对照 ent TicketTemplate schema）。此前这组方法委托给 Repository
// 层的 stub——CreateTemplate 原样返回入参、GetTemplate 恒 not found，属于
// "伪成功"：前端显示创建/编辑成功，库里实际没有记录，违反"能力未实现必须
// 如实暴露，禁止包装成空成功"的错误语义禁令。现在统一接通真实现；
// productionSvc 为 nil（纯单测 harness）时 fail-closed 报错，不再伪造成功。

// dtoTemplateToDomain 把 service.TicketService 返回的 dto.TicketTemplate 转回
// handler 层模板实体，供 normalizeTemplate 统一映射响应。
func dtoTemplateToDomain(t *dto.TicketTemplate) *TicketTemplate {
	if t == nil {
		return nil
	}
	return &TicketTemplate{
		ID:            t.ID,
		Name:          t.Name,
		Description:   t.Description,
		Category:      t.Category,
		Priority:      t.Priority,
		Fields:        t.Fields,
		FormFields:    t.FormFields,
		WorkflowSteps: t.WorkflowSteps,
		IsActive:      t.IsActive != nil && *t.IsActive,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

// GetTicketTemplates returns all ticket templates.
func (s *Service) GetTicketTemplates(ctx context.Context, tenantID int) ([]*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	items, err := s.productionSvc.GetTicketTemplates(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]*TicketTemplate, 0, len(items))
	for _, item := range items {
		if t, ok := item.(*dto.TicketTemplate); ok {
			result = append(result, dtoTemplateToDomain(t))
		}
	}
	return result, nil
}

// GetTicketTemplate returns a single template.
func (s *Service) GetTicketTemplate(ctx context.Context, tenantID int, templateID int) (*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	t, err := s.productionSvc.GetTicketTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, err
	}
	return dtoTemplateToDomain(t), nil
}

// CreateTicketTemplate creates a new template.
func (s *Service) CreateTicketTemplate(ctx context.Context, tenantID int, req *dto.TicketTemplate) (*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	created, err := s.productionSvc.CreateTicketTemplate(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}
	t, ok := created.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("Return type exception when creating work order template")
	}
	return dtoTemplateToDomain(t), nil
}

// UpdateTicketTemplate updates a template.
func (s *Service) UpdateTicketTemplate(ctx context.Context, tenantID int, templateID int, req *dto.TicketTemplate) (*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	updated, err := s.productionSvc.UpdateTicketTemplate(ctx, tenantID, templateID, req)
	if err != nil {
		return nil, err
	}
	t, ok := updated.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("Work order template update return type exception")
	}
	return dtoTemplateToDomain(t), nil
}

// DeleteTicketTemplate deletes a template.
func (s *Service) DeleteTicketTemplate(ctx context.Context, tenantID int, templateID int) error {
	if s.productionSvc == nil {
		return fmt.Errorf("The ticket template service is unavailable")
	}
	return s.productionSvc.DeleteTicketTemplate(ctx, tenantID, templateID)
}

// UpdateTicketTemplateStatus enables or disables a template.
func (s *Service) UpdateTicketTemplateStatus(ctx context.Context, tenantID int, templateID int, isActive bool) (*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	t, err := s.productionSvc.UpdateTicketTemplateStatus(ctx, tenantID, templateID, isActive)
	if err != nil {
		return nil, err
	}
	return dtoTemplateToDomain(t), nil
}

// CopyTicketTemplate copies a template with a new name.
func (s *Service) CopyTicketTemplate(ctx context.Context, tenantID int, templateID int, newName string) (*TicketTemplate, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	t, err := s.productionSvc.CopyTicketTemplate(ctx, tenantID, templateID, newName)
	if err != nil {
		return nil, err
	}
	return dtoTemplateToDomain(t), nil
}

// GetTicketTemplateCategories returns template categories.
func (s *Service) GetTicketTemplateCategories(ctx context.Context, tenantID int) ([]string, error) {
	if s.productionSvc == nil {
		return nil, fmt.Errorf("The ticket template service is unavailable")
	}
	return s.productionSvc.GetTicketTemplateCategories(ctx, tenantID)
}

// GetSubtasks returns child tickets of a parent ticket.
func (s *Service) GetSubtasks(ctx context.Context, parentID int, tenantID int, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	filters := map[string]interface{}{
		"parent_ticket_id": parentID,
	}
	return s.repo.List(ctx, tenantID, 1, 100, filters, datascope.DataScopeOwnedOrAssigned, currentUserID)
}

// CreateSubtask creates a child ticket.
func (s *Service) CreateSubtask(ctx context.Context, tenantID int, parentID int, params *CreateParams) (*Ticket, error) {
	params.ParentTicketID = &parentID
	return s.repo.Create(ctx, params, tenantID)
}

// UpdateSubtask updates a child ticket.
// 子任务写路径同样受行级写权限约束。
func (s *Service) UpdateSubtask(ctx context.Context, tenantID int, subtaskID int, params *UpdateParams, actorID int, actorRole string) (*Ticket, error) {
	// Verify subtask belongs to parent
	current, err := s.repo.GetByID(ctx, subtaskID, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return nil, common.NewForbiddenError("No permission to modify this subtask: only the creator, assignee or administrator can operate")
	}
	// 修复：原代码比较 *current.ParentTicketID != tenantID —— ParentTicketID 是父工单 ID，
	// 恒不等于 tenantID，导致所有子任务更新都被误判为"不属于父工单"。正确语义是：
	// 仅当该工单不是子任务（无父工单）时拒绝。
	if current.ParentTicketID == nil {
		return nil, common.NewBadRequestError("subtask does not belong to parent", nil)
	}
	if params.Status != nil && !common.IsValidTicketStatusTransition(current.Status, *params.Status) {
		return nil, lifecycleError(fmt.Errorf("%w: from '%s' to '%s'", ErrTicketInvalidTransition, current.Status, *params.Status))
	}
	return s.repo.Update(ctx, subtaskID, params, tenantID)
}

// DeleteSubtask deletes a child ticket.
// 子任务删除同样受行级写权限约束。
func (s *Service) DeleteSubtask(ctx context.Context, subtaskID int, tenantID int, actorID int, actorRole string) error {
	current, err := s.repo.GetByID(ctx, subtaskID, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return common.NewForbiddenError("No permission to delete this subtask: only the creator, assignee or administrator can operate")
	}
	return s.repo.Delete(ctx, subtaskID, tenantID)
}

// GetTicketSLAInfo 返回工单的 SLA 信息。
//
// 生产装配下委托给 productionSvc：SLA 字段以 sla_states 表为准、缺记录时降级到
// 工单内联字段的规则由该领域唯一实现持有。handler 层此前复制了一份 ad-hoc map，
// 字段会随一侧演进而漏掉（slaDefinitionName / ticketNumber / 剩余时间曾因此丢失）。
//
// productionSvc 为 nil（单元测试或降级装配）时用 repo 构造同一契约，字段必须与
// service.TicketSLAInfo 保持一致。
func (s *Service) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*service.TicketSLAInfo, error) {
	if s.productionSvc != nil {
		return s.productionSvc.GetTicketSLAInfo(ctx, ticketID, tenantID)
	}

	t, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	info := &service.TicketSLAInfo{
		TicketID:     t.ID,
		TicketNumber: t.TicketNumber,
		Priority:     t.Priority,
	}

	// 优先从 sla_states 读取
	if s.slaMonitor != nil {
		if slaSt, slaErr := s.slaMonitor.GetSLAState(ctx, tenantID, "ticket", ticketID); slaErr == nil && slaSt != nil {
			info.SLADefinitionID = slaSt.SLADefinitionID
			if !slaSt.ResponseDeadline.IsZero() {
				info.ResponseDeadline = slaSt.ResponseDeadline
			}
			if !slaSt.ResolutionDeadline.IsZero() {
				info.ResolutionDeadline = slaSt.ResolutionDeadline
			}
			if !slaSt.FirstResponseAt.IsZero() {
				info.FirstResponseAt = &slaSt.FirstResponseAt
			}
			if !slaSt.ResolvedAt.IsZero() {
				info.ResolvedAt = &slaSt.ResolvedAt
			}
			s.fillSLAComputedFields(ctx, tenantID, info)
			return info, nil
		}
	}

	// 降级：使用工单内联字段
	if t.SLADefinitionID != nil {
		info.SLADefinitionID = *t.SLADefinitionID
	}
	if t.SLAResponseDeadline != nil {
		info.ResponseDeadline = *t.SLAResponseDeadline
	}
	if t.SLAResolutionDeadline != nil {
		info.ResolutionDeadline = *t.SLAResolutionDeadline
	}
	info.FirstResponseAt = t.FirstResponseAt
	info.ResolvedAt = t.ResolvedAt
	s.fillSLAComputedFields(ctx, tenantID, info)
	return info, nil
}

// fillSLAComputedFields 补齐 SLA 定义名称与响应/解决的剩余分钟数与违约标记。
// 定义名称按租户查询，避免跨租户名称泄漏；定义已删除时留空，由响应显示 "-"。
func (s *Service) fillSLAComputedFields(ctx context.Context, tenantID int, info *service.TicketSLAInfo) {
	if info.SLADefinitionID > 0 {
		if name, err := s.repo.GetSLADefinitionName(ctx, tenantID, info.SLADefinitionID); err == nil {
			info.SLADefinitionName = name
		} else if s.logger != nil {
			s.logger.Warnw("加载 SLA 定义名称失败", "tenant_id", tenantID, "sla_definition_id", info.SLADefinitionID, "error", err)
		}
	}

	now := time.Now()
	if info.FirstResponseAt != nil {
		info.ResponseTimeLeft = 0
		info.IsResponseBreached = false
	} else if !info.ResponseDeadline.IsZero() {
		if now.After(info.ResponseDeadline) {
			info.IsResponseBreached = true
		} else {
			info.ResponseTimeLeft = int(info.ResponseDeadline.Sub(now).Minutes())
		}
	}

	if info.ResolvedAt != nil {
		info.ResolutionTimeLeft = 0
		info.IsResolutionBreached = false
	} else if !info.ResolutionDeadline.IsZero() {
		if now.After(info.ResolutionDeadline) {
			info.IsResolutionBreached = true
		} else {
			info.ResolutionTimeLeft = int(info.ResolutionDeadline.Sub(now).Minutes())
		}
	}
}

// GetTicketActivity retrieves ticket activity log (merged comments, attachments, status changes)
func (s *Service) GetTicketActivity(ctx context.Context, ticketID, tenantID int) ([]map[string]interface{}, error) {
	return s.productionSvc.GetTicketActivity(ctx, ticketID, tenantID)
}
