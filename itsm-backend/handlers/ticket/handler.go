package ticket

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for tickets.
type Handler struct {
	service *Service
}

// NewHandler creates a new ticket handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// buildTicketResponse 只做字段映射，不查库。
func buildTicketResponse(t *Ticket) *dto.TicketResponse {
	if t == nil {
		return nil
	}
	resp := &dto.TicketResponse{
		ID:             t.ID,
		TicketNumber:   t.TicketNumber,
		Title:          t.Title,
		Description:    t.Description,
		Status:         t.Status,
		Priority:       t.Priority,
		Type:           t.Type,
		TicketTypeID:   0,
		TicketTypeCode: t.TicketTypeCode,
		TicketTypeName: t.TicketTypeName,
		FormFields:     t.FormFields,
		RequesterID:    t.RequesterID,
		TenantID:       t.TenantID,
		Version:        t.Version,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
	if t.AssigneeID != nil {
		resp.AssigneeID = *t.AssigneeID
	}
	if t.TicketTypeID != nil {
		resp.TicketTypeID = *t.TicketTypeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = *t.CategoryID
	}
	if t.Resolution != nil {
		resp.Resolution = *t.Resolution
	}
	// Lifecycle timestamps and resolution text are part of the operator-visible
	// contract: resolve/close persist them, and the UI shows them on the detail
	// page. Dropping them here made a resolved ticket look unresolved to any
	// consumer that only reads the API (found by `itsm flow run ticket-lifecycle`).
	resp.ResolvedAt = t.ResolvedAt
	resp.ClosedAt = t.ClosedAt
	resp.FirstResponseAt = t.FirstResponseAt
	resp.SLAResponseDeadline = t.SLAResponseDeadline
	resp.SLAResolutionDeadline = t.SLAResolutionDeadline
	return resp
}

// ticketToResponse 转 DTO 并投影报障人/处理人显示名。
// 姓名投影必须由服务端完成：dto 里曾经声明过 requester/assignee 对象却从未赋值，
// 前端列表因此把每个已派工单渲染成「未分配」或裸用户 ID。
func (h *Handler) ticketToResponse(ctx context.Context, tenantID int, t *Ticket) *dto.TicketResponse {
	resp := buildTicketResponse(t)
	if resp == nil {
		return nil
	}
	h.attachUserNames(ctx, tenantID, []*dto.TicketResponse{resp})
	return resp
}

func (h *Handler) ticketListToResponse(ctx context.Context, tenantID int, ts []*Ticket) []*dto.TicketResponse {
	result := make([]*dto.TicketResponse, 0, len(ts))
	for _, t := range ts {
		if r := buildTicketResponse(t); r != nil {
			result = append(result, r)
		}
	}
	h.attachUserNames(ctx, tenantID, result)
	return result
}

// attachUserNames 用一次 IN 查询回填 requesterName/assigneeName；
// 未命中映射的工单保持空姓名，由前端回退显示用户 ID。
func (h *Handler) attachUserNames(ctx context.Context, tenantID int, items []*dto.TicketResponse) {
	if len(items) == 0 {
		return
	}
	ids := make([]int, 0, len(items)*2)
	seen := make(map[int]struct{}, len(items)*2)
	add := func(id int) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, item := range items {
		add(item.RequesterID)
		add(item.AssigneeID)
	}
	if len(ids) == 0 {
		return
	}
	names := h.service.LoadUserNames(ctx, tenantID, ids)
	for _, item := range items {
		if name, ok := names[item.RequesterID]; ok {
			item.RequesterName = name
		}
		if name, ok := names[item.AssigneeID]; ok {
			item.AssigneeName = name
		}
	}
}

// ticketStatsToDTO 将领域实体转为与前端契约一致的扁平 DTO。
// 前端期望字段：total / open / inProgress / resolved / pending / highPriority / overdue。
// entity 字段：TotalTickets / OpenTickets / InProgressTickets / PendingTickets /
// ResolvedTickets / ClosedTickets / CriticalTickets / HighTickets / AvgResolutionMin。
func ticketStatsToDTO(s *TicketStats) *dto.TicketStatsResponse {
	if s == nil {
		return &dto.TicketStatsResponse{ByStatus: []dto.TicketStatusCount{}, ByPriority: []dto.TicketPriorityCount{}}
	}
	out := &dto.TicketStatsResponse{
		Total:        s.TotalTickets,
		Open:         s.OpenTickets,
		InProgress:   s.InProgressTickets,
		Resolved:     s.ResolvedTickets,
		Closed:       s.ClosedTickets,
		Pending:      s.PendingTickets,
		HighPriority: s.HighTickets,
		Overdue:      s.OverdueTickets,
		ByStatus:     make([]dto.TicketStatusCount, 0, len(s.ByStatus)),
		ByPriority:   make([]dto.TicketPriorityCount, 0, len(s.ByPriority)),
	}
	for _, item := range s.ByStatus {
		out.ByStatus = append(out.ByStatus, dto.TicketStatusCount{Status: item.Status, Count: item.Count})
	}
	for _, item := range s.ByPriority {
		out.ByPriority = append(out.ByPriority, dto.TicketPriorityCount{Priority: item.Priority, Count: item.Count})
	}
	return out
}

// CreateTicket handles POST /api/v1/tickets
func (h *Handler) CreateTicket(c *gin.Context) {
	var req dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if tenantID == 0 {
		common.Fail(c, common.AuthErrorCode, "Tenant ID missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.AuthErrorCode, "User ID missing")
		return
	}

	params := &CreateParams{
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		RequesterID: userID,
		FormFields:  req.FormFields,

		WorkflowDefinitionKey: req.WorkflowDefinitionKey,
	}
	if req.AssigneeID > 0 {
		params.AssigneeID = &req.AssigneeID
	}
	if req.TicketTypeID != nil {
		params.TicketTypeID = req.TicketTypeID
	}
	if req.CategoryID != nil {
		params.CategoryID = req.CategoryID
	}
	if req.TemplateID != nil {
		params.TemplateID = req.TemplateID
	}
	if req.ParentTicketID != nil {
		params.ParentTicketID = req.ParentTicketID
	}
	if len(req.TagIDs) > 0 {
		params.TagIDs = req.TagIDs
	}

	ticket, err := h.service.Create(c.Request.Context(), tenantID, params)
	if err != nil {
		// 统一错误出口：AppError/BusinessError 按语义分流，其余兜底 500。
		common.RespondError(c, err, "创建工单失败")
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// GetTicket handles GET /api/v1/tickets/:id
// id 同时接受数字 ID(9)和业务工单号(TKT-202609-000010);
// 前端 [ticketId] 动态路由会把两者都传进来,需要 fallback 到 GetByNumber。
func (h *Handler) GetTicket(c *gin.Context) {
	idStr := c.Param("id")

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	var (
		ticket *Ticket
		err    error
	)
	if id, atoiErr := strconv.Atoi(idStr); atoiErr == nil {
		ticket, err = h.service.Get(ctx, id, tenantID)
	} else {
		// 非数字 ID 视为业务工单号,例如 TKT-202609-000010
		ticket, err = h.service.GetByNumber(ctx, idStr, tenantID)
	}
	if err != nil {
		// 读取失败不等于「不存在」：驱动故障必须按统一分类上报为 500，
		// 全部压成 404 会让真实故障在监控里表现为大量客户端错误。
		common.RespondError(c, err, "获取工单失败")
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// ListTickets handles GET /api/v1/tickets
func (h *Handler) ListTickets(c *gin.Context) {
	var req dto.ListTicketsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	filters := make(map[string]interface{})
	if req.Status != "" {
		filters["status"] = req.Status
	}
	if req.Priority != "" {
		filters["priority"] = req.Priority
	}
	if req.Type != "" {
		filters["type"] = req.Type
	}
	if req.Keyword != "" {
		filters["keyword"] = req.Keyword
	}
	if req.SortBy != "" {
		filters["sortBy"] = req.SortBy
	}
	if req.SortOrder != "" {
		filters["sortOrder"] = req.SortOrder
	}
	if req.IsOverdue {
		filters["is_overdue"] = true
	}
	// 指针字段必须显式判空下传：仓储层按 filters["assignee_id"] 等键位取过滤条件，
	// 这里漏一个键不会报错，只会让筛选静默失效并返回租户全量工单：
	// 前端工单列表页会发送 assigneeId，漏下传时结果看着正常、范围却是错的。
	if req.AssigneeID != nil && *req.AssigneeID > 0 {
		filters["assignee_id"] = *req.AssigneeID
	}
	if req.RequesterID != nil && *req.RequesterID > 0 {
		filters["requester_id"] = *req.RequesterID
	}
	if req.CategoryID != nil && *req.CategoryID > 0 {
		filters["category_id"] = *req.CategoryID
	}
	if req.ParentTicketID != nil && *req.ParentTicketID > 0 {
		filters["parent_ticket_id"] = *req.ParentTicketID
	}
	if req.TemplateID != nil && *req.TemplateID > 0 {
		filters["template_id"] = *req.TemplateID
	}

	// 分页参数必须在使用前归一化：客户端省略 page/pageSize 时它们是零值，
	// 直接下传会让仓储算出负偏移，直接回显又会让信封的 page/pageSize 变成 0。
	page, pageSize := common.ValidatePagination(req.Page, req.PageSize)

	tickets, total, err := h.service.List(c.Request.Context(), tenantID, page, pageSize, filters, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 标准信封：data.items + total/page/pageSize/totalPages
	common.SuccessWithPagination(c, h.ticketListToResponse(c.Request.Context(), tenantID, tickets), page, pageSize, int64(total))
}

// UpdateTicket handles PUT /api/v1/tickets/:id
func (h *Handler) UpdateTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	params, paramErr := buildUpdateParams(&req)
	if paramErr != nil {
		common.RespondError(c, paramErr, "请求参数错误")
		return
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, id, params, actorID, actorRole)
	if err != nil {
		respondTicketUpdateError(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, updated))
}

// DeleteTicket handles DELETE /api/v1/tickets/:id
func (h *Handler) DeleteTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	if err := h.service.Delete(c.Request.Context(), id, tenantID, actorID, actorRole); err != nil {
		if isForbiddenErr(err) {
			common.RespondError(c, err, "操作失败")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单删除成功"})
}

// UpdateStatus handles PUT /api/v1/tickets/:id/status
func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	ticket, err := h.service.UpdateStatus(c.Request.Context(), id, req.Status, tenantID, userID, actorRole)
	if err != nil {
		failTicketOperation(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// BatchDelete handles POST /api/v1/tickets/batch-delete
func (h *Handler) BatchDeleteTickets(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	if err := h.service.BatchDelete(c.Request.Context(), req.TicketIDs, tenantID, actorID, actorRole); err != nil {
		if isForbiddenErr(err) {
			common.RespondError(c, err, "操作失败")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":      "批量删除成功",
		"deletedCount": len(req.TicketIDs),
	})
}

// GetStats handles GET /api/v1/tickets/stats
// 身份只取认证上下文，统计范围与 GET /api/v1/tickets 列表一致。
func (h *Handler) GetTicketStats(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	stats, err := h.service.GetStats(c.Request.Context(), tenantID, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 统计必须经 ticketStatsToDTO 转成 camelCase DTO 再返回；直接序列化 Ent 模型会带出
	// TotalTickets/OpenTickets 这类字段名，前端解析不到，stats 卡片全是 0。
	common.Success(c, ticketStatsToDTO(stats))
}

// AssignTicket handles POST /api/v1/tickets/:id/assign
func (h *Handler) AssignTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	if req.AssigneeID <= 0 {
		common.Fail(c, common.ParamErrorCode, "assigneeId 必填")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	ticket, err := h.service.AssignTicket(c.Request.Context(), id, req.AssigneeID, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// EscalateTicket handles POST /api/v1/tickets/:id/escalate
func (h *Handler) EscalateTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.EscalateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	ticket, err := h.service.EscalateTicket(c.Request.Context(), id, req.Reason, tenantID, actorID, actorRole)
	if err != nil {
		failTicketOperation(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// ResolveTicket handles POST /api/v1/tickets/:id/resolve
func (h *Handler) ResolveTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.ResolveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	resolution := req.Resolution
	if resolution == "" {
		resolution = req.Solution
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	ticket, err := h.service.ResolveTicket(c.Request.Context(), id, resolution, tenantID, actorID, actorRole)
	if err != nil {
		failTicketOperation(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// CloseTicket handles POST /api/v1/tickets/:id/close
func (h *Handler) CloseTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	ticket, err := h.service.CloseTicket(c.Request.Context(), id, tenantID, actorID, actorRole)
	if err != nil {
		failTicketOperation(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// SearchTickets handles GET /api/v1/tickets/search
func (h *Handler) SearchTickets(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		common.Fail(c, common.ParamErrorCode, "搜索关键词不能为空")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	pagination := common.GetPaginationFromQuery(c)
	// 搜索沿用列表的行级数据范围（DataScope），避免 /tickets/search 成为绕权后门。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")
	tickets, total, err := h.service.Search(c.Request.Context(), keyword, tenantID, pagination.Page, pagination.PageSize, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.SuccessWithPagination(c, h.ticketListToResponse(c.Request.Context(), tenantID, tickets),
		pagination.Page, pagination.PageSize, int64(total))
}

// GetOverdueTickets handles GET /api/v1/tickets/overdue
func (h *Handler) GetOverdueTickets(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	pagination := common.GetPaginationFromQuery(c)
	// 逾期列表沿用与列表相同的行级数据范围；此前缺失 DataScope，普通角色可读全租户逾期工单。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")
	tickets, total, err := h.service.GetOverdueTickets(c.Request.Context(), tenantID, pagination.Page, pagination.PageSize, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.SuccessWithPagination(c, h.ticketListToResponse(c.Request.Context(), tenantID, tickets),
		pagination.Page, pagination.PageSize, int64(total))
}

// GetTicketsByAssignee handles GET /api/v1/tickets/assignee/:assignee_id
func (h *Handler) GetTicketsByAssignee(c *gin.Context) {
	assigneeID, err := strconv.Atoi(c.Param("assignee_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的处理人ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	tickets, err := h.service.GetTicketsByAssignee(c.Request.Context(), assigneeID, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 未分页的服务端全量返回：page=1、pageSize=total 表达"单页全量"，
	// 空结果由 NewPaginationResponse 归一化（pageSize→默认值、totalPages→0）。
	common.SuccessWithPagination(c, h.ticketListToResponse(c.Request.Context(), tenantID, tickets),
		1, len(tickets), int64(len(tickets)))
}

// GetTicketSLAInfo handles GET /api/v1/tickets/:id/sla
func (h *Handler) GetTicketSLAInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	slaInfo, err := h.service.GetTicketSLAInfo(c.Request.Context(), id, tenantID)
	if err != nil {
		// 与 GetTicket 同一口径：驱动故障必须按统一分类上报为 500。全部压成
		// 404 会让 DB 故障在监控里表现为一片客户端错误，也会让调用方误判
		// 「工单不存在」而反复重试。
		common.RespondError(c, err, "获取工单 SLA 信息失败")
		return
	}

	common.Success(c, slaInfo)
}

// ExportTickets handles POST /api/v1/tickets/export
func (h *Handler) ExportTickets(c *gin.Context) {
	var req dto.TicketExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	filters := map[string]interface{}{
		"status":   req.Filters.Status,
		"priority": req.Filters.Priority,
	}
	data, err := h.service.ExportTickets(c.Request.Context(), tenantID, filters, req.Format)
	if err != nil {
		common.FailWithErr(c, err, "工单导出失败")
		return
	}

	contentType, filename := ticketExportDisposition(req.Format)
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, contentType, data)
}

// ticketExportDisposition 返回导出格式的 Content-Type 与下载文件名。
// 格式集合与 dto.TicketExportRequest 的 oneof、service 层实现保持一致。
func ticketExportDisposition(format string) (contentType, filename string) {
	switch format {
	case "excel":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "tickets.xlsx"
	default:
		return "text/csv; charset=utf-8", "tickets.csv"
	}
}

// ImportTickets handles POST /api/v1/tickets/import
func (h *Handler) ImportTickets(c *gin.Context) {
	var req dto.TicketImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	fileData := []byte(req.File)
	if err := h.service.ImportTickets(c.Request.Context(), tenantID, fileData, req.Format); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单导入成功", "format": req.Format})
}

// AssignTickets handles POST /api/v1/tickets/assign
func (h *Handler) AssignTickets(c *gin.Context) {
	var req dto.TicketAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if err := h.service.AssignTickets(c.Request.Context(), tenantID, req.TicketIDs, req.AssigneeID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":       "工单分配成功",
		"assignedCount": len(req.TicketIDs),
	})
}

// GetTicketTemplates handles GET /api/v1/tickets/templates
func (h *Handler) GetTicketTemplates(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	templates, err := h.service.GetTicketTemplates(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	normalized := make([]gin.H, 0, len(templates))
	for _, tmpl := range templates {
		normalized = append(normalized, normalizeTemplate(tmpl))
	}
	// 模板列表不分页，因此信封只带 items + total；历史实现伪造了 page=1、
	// pageSize=len(templates)，让前端以为存在一个不存在的分页协议。
	common.Success(c, gin.H{
		"items": normalized,
		"total": len(normalized),
	})
}

// GetTicketTemplate handles GET /api/v1/tickets/templates/:id
func (h *Handler) GetTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	tmpl, err := h.service.GetTicketTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, normalizeTemplate(tmpl))
}

// CreateTicketTemplate handles POST /api/v1/tickets/templates
func (h *Handler) CreateTicketTemplate(c *gin.Context) {
	var tmpl dto.TicketTemplate
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	created, err := h.service.CreateTicketTemplate(c.Request.Context(), tenantID, &tmpl)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, normalizeTemplate(created))
}

// UpdateTicketTemplate handles PUT /api/v1/tickets/templates/:id
func (h *Handler) UpdateTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var tmpl dto.TicketTemplate
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	updated, err := h.service.UpdateTicketTemplate(c.Request.Context(), tenantID, templateID, &tmpl)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, normalizeTemplate(updated))
}

// DeleteTicketTemplate handles DELETE /api/v1/tickets/templates/:id
func (h *Handler) DeleteTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTicketTemplate(c.Request.Context(), tenantID, templateID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单模板删除成功", "templateId": templateID})
}

// UpdateTicketTemplateStatus handles PUT /api/v1/tickets/templates/:id/status
func (h *Handler) UpdateTicketTemplateStatus(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var req struct {
		IsActive *bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	if req.IsActive == nil {
		common.Fail(c, common.ParamErrorCode, "isActive 字段不能为空")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	tmpl, err := h.service.UpdateTicketTemplateStatus(c.Request.Context(), tenantID, templateID, *req.IsActive)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, normalizeTemplate(tmpl))
}

// CopyTicketTemplate handles POST /api/v1/tickets/templates/:id/copy
func (h *Handler) CopyTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	// 绑定失败必须显式拦截：否则非法 body 会让 req 保持零值，调用方指定的
	// 副本名被静默丢弃（服务层回退为「源模板 - 副本」），复制照样成功并返回
	// 200——请求被部分执行，而调用方以为自己的参数生效了（gosec G104）。
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	tmpl, err := h.service.CopyTicketTemplate(c.Request.Context(), tenantID, templateID, req.Name)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, normalizeTemplate(tmpl))
}

// GetTicketTemplateCategories handles GET /api/v1/tickets/templates/categories
func (h *Handler) GetTicketTemplateCategories(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	categories, err := h.service.GetTicketTemplateCategories(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, categories)
}

// GetSubtasks handles GET /api/v1/tickets/:id/subtasks
func (h *Handler) GetSubtasks(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	tickets, _, err := h.service.GetSubtasks(c.Request.Context(), parentID, tenantID, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 未分页的服务端全量返回：page=1、pageSize=total 表达"单页全量"，
	// 空结果由 NewPaginationResponse 归一化（pageSize→默认值、totalPages→0）。
	common.SuccessWithPagination(c, h.ticketListToResponse(c.Request.Context(), tenantID, tickets),
		1, len(tickets), int64(len(tickets)))
}

// CreateSubtask handles POST /api/v1/tickets/:id/subtasks
func (h *Handler) CreateSubtask(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CreateSubtaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")

	params := &CreateParams{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Type:        req.Type,
		RequesterID: userID,
		FormFields:  req.FormFields,
	}
	if req.AssigneeID > 0 {
		params.AssigneeID = &req.AssigneeID
	}

	ticket, err := h.service.CreateSubtask(c.Request.Context(), tenantID, parentID, params)
	if err != nil {
		// errors.As 而非顶层类型断言：service 的校验错误常被 fmt.Errorf("%w")
		// 或事务包装一层，断言会漏判并把 409/403 的业务拒绝降级成 500。
		var businessErr *common.BusinessError
		if errors.As(err, &businessErr) {
			common.Fail(c, businessErr.Code, businessErr.Message)
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, ticket))
}

// UpdateSubtask handles PUT /api/v1/tickets/:id/subtasks/:subtask_id
func (h *Handler) UpdateSubtask(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的父工单ID")
		return
	}
	subtaskID, err := strconv.Atoi(c.Param("subtask_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的子任务ID")
		return
	}

	var req dto.UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	// Verify subtask belongs to parent
	current, err := h.service.Get(c.Request.Context(), subtaskID, tenantID)
	if err != nil {
		common.RespondError(c, err, "子任务不存在")
		return
	}
	if current.ParentTicketID == nil || *current.ParentTicketID != parentID {
		common.Fail(c, common.ParamErrorCode, "子任务不属于指定的父工单")
		return
	}

	params, paramErr := buildUpdateParams(&req)
	if paramErr != nil {
		common.RespondError(c, paramErr, "请求参数错误")
		return
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, subtaskID, params, actorID, actorRole)
	if err != nil {
		respondTicketUpdateError(c, err)
		return
	}

	common.Success(c, h.ticketToResponse(c.Request.Context(), tenantID, updated))
}

// DeleteSubtask handles DELETE /api/v1/tickets/:id/subtasks/:subtask_id
func (h *Handler) DeleteSubtask(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的父工单ID")
		return
	}
	subtaskID, err := strconv.Atoi(c.Param("subtask_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的子任务ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	actorID := c.GetInt("user_id")
	actorRole := c.GetString("role")

	// Verify subtask belongs to parent
	current, err := h.service.Get(c.Request.Context(), subtaskID, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "子任务不存在")
		return
	}
	if current.ParentTicketID == nil || *current.ParentTicketID != parentID {
		common.Fail(c, common.ParamErrorCode, "子任务不属于指定的父工单")
		return
	}

	if err := h.service.DeleteSubtask(c.Request.Context(), subtaskID, tenantID, actorID, actorRole); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "子任务删除成功"})
}

// PauseSLA handles PUT /api/v1/tickets/:id/sla/pause
// Note: SLA pause/resume require service.SLAMonitorService which is not part of
// the handlers/ticket Service struct. These handlers delegate to the legacy
// controller for now.
func (h *Handler) PauseSLA(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的工单ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "暂停原因为必填项")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if tenantID == 0 {
		common.AuthFailed(c, "缺少租户上下文")
		return
	}

	if err := h.service.PauseSLA(c.Request.Context(), tenantID, id, req.Reason); err != nil {
		common.RespondError(c, err, "暂停SLA失败")
		return
	}
	common.Success(c, gin.H{"message": "SLA已暂停", "ticketId": id, "slaStatus": "paused"})
}

// ResumeSLA handles PUT /api/v1/tickets/:id/sla/resume
func (h *Handler) ResumeSLA(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的工单ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if tenantID == 0 {
		common.AuthFailed(c, "缺少租户上下文")
		return
	}

	if err := h.service.ResumeSLA(c.Request.Context(), tenantID, id); err != nil {
		common.RespondError(c, err, "恢复SLA失败")
		return
	}
	common.Success(c, gin.H{"message": "SLA已恢复", "ticketId": id, "slaStatus": "active"})
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

func normalizeTemplate(tmpl *TicketTemplate) gin.H {
	if tmpl == nil {
		return gin.H{}
	}
	formFields := tmpl.FormFields
	if formFields == nil {
		formFields = map[string]interface{}{}
	}
	return gin.H{
		"id":            tmpl.ID,
		"name":          tmpl.Name,
		"description":   tmpl.Description,
		"category":      tmpl.Category,
		"priority":      tmpl.Priority,
		"fields":        tmpl.Fields,
		"formFields":    formFields,
		"workflowSteps": tmpl.WorkflowSteps,
		"isActive":      tmpl.IsActive,
		"createdAt":     tmpl.CreatedAt,
		"updatedAt":     tmpl.UpdatedAt,
	}
}

func isForbiddenErr(err error) bool {
	var appErr *common.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == common.ErrCodeForbidden
	}
	return false
}

// failTicketOperation 统一生命周期写操作（resolve/close/escalate/updateStatus）
// 的错误出口：403 AppError 必须按语义分流（对齐错误映射铁律），不得掉入
// FailWithErr 兜底被吞成 500。
func failTicketOperation(c *gin.Context, err error) {
	if isForbiddenErr(err) {
		common.RespondError(c, err, "操作失败")
		return
	}
	var bizErr *common.BusinessError
	if errors.As(err, &bizErr) {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.FailWithErr(c, err, "操作失败")
}

// buildUpdateParams 把更新请求转换成仓储参数。工单与子任务共用同一份映射，
// 避免子任务端点只落 title/description/priority、其余字段静默丢弃。
func buildUpdateParams(req *dto.UpdateTicketRequest) (*UpdateParams, error) {
	if len(req.Tags) > 0 {
		// 本端点没有标签名解析能力；静默丢弃会让调用方以为标签已写入。
		return nil, common.NewBusinessError(common.ParamErrorCode, "工单标签请使用 PATCH /api/v1/tickets/:id/relations（tagIds）", "")
	}
	params := &UpdateParams{Version: req.Version}
	if req.Title != "" {
		params.Title = &req.Title
	}
	if req.Description != "" {
		params.Description = &req.Description
	}
	if req.Status != "" {
		params.Status = &req.Status
	}
	if req.Priority != "" {
		params.Priority = &req.Priority
	}
	if req.Type != "" {
		params.Type = &req.Type
	}
	if req.AssigneeID > 0 {
		params.AssigneeID = &req.AssigneeID
	}
	if req.CategoryID != nil {
		params.CategoryID = req.CategoryID
	}
	if req.FormFields != nil {
		params.FormFields = &req.FormFields
	}
	if req.Resolution != "" {
		params.Resolution = &req.Resolution
	}
	return params, nil
}

// respondTicketUpdateError 是工单更新（含子任务）的统一错误出口。
// 版本冲突必须用 errors.As 解包：service 层包装后顶层类型断言会漏判，
// 乐观锁冲突会掉进 500。其余错误交给统一分类器（403/404/409/500）。
func respondTicketUpdateError(c *gin.Context, err error) {
	var conflictErr *common.VersionConflictError
	if errors.As(err, &conflictErr) {
		common.Conflict(c, conflictErr.Error(), gin.H{
			"ticketId":       conflictErr.ResourceID,
			"currentVersion": conflictErr.CurrentVersion,
			"serverVersion":  conflictErr.ServerVersion,
		})
		return
	}
	if isUserInputUpdateError(err) {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	common.RespondError(c, err, "操作失败")
}

func isUserInputUpdateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	patterns := []string{
		"工单分类不存在或不可用",
		"处理人不存在或不可用",
		"验证处理人失败",
		"验证工单分类失败",
		"无法解析工单分类",
		"解析工单标签失败",
		"无法解析工单标签",
		"工单分类名称不存在",
	}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

func (h *Handler) GetTicketActivity(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.GetTicketActivity(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "获取活动日志失败")
		return
	}
	common.Success(c, result)
}
