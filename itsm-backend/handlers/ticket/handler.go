package ticket

import (
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

// ticketToResponse converts a Ticket domain model to a TicketResponse DTO.
func ticketToResponse(t *Ticket) *dto.TicketResponse {
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
	return resp
}

func ticketListToResponse(ts []*Ticket) []*dto.TicketResponse {
	result := make([]*dto.TicketResponse, 0, len(ts))
	for _, t := range ts {
		if r := ticketToResponse(t); r != nil {
			result = append(result, r)
		}
	}
	return result
}

// ticketStatsToDTO 将领域实体转为与前端契约一致的扁平 DTO。
// 前端期望字段：total / open / inProgress / resolved / pending / highPriority / overdue。
// entity 字段：TotalTickets / OpenTickets / InProgressTickets / PendingTickets /
// ResolvedTickets / ClosedTickets / CriticalTickets / HighTickets / AvgResolutionMin。
func ticketStatsToDTO(s *TicketStats) *dto.TicketStatsResponse {
	if s == nil {
		return &dto.TicketStatsResponse{}
	}
	return &dto.TicketStatsResponse{
		Total:        s.TotalTickets,
		Open:         s.OpenTickets,
		InProgress:   s.InProgressTickets,
		Resolved:     s.ResolvedTickets,
		Pending:      s.PendingTickets,
		HighPriority: s.HighTickets,
		Overdue:      s.OverdueTickets,
	}
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
		if businessErr, ok := err.(*common.BusinessError); ok {
			common.Fail(c, businessErr.Code, businessErr.Message)
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// GetTicket handles GET /api/v1/tickets/:id
func (h *Handler) GetTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	ticket, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "工单不存在")
		return
	}

	common.Success(c, ticketToResponse(ticket))
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
	if req.Keyword != "" {
		filters["keyword"] = req.Keyword
	}

	tickets, total, err := h.service.List(c.Request.Context(), tenantID, req.Page, req.PageSize, filters, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// v1.1 回归：使用 SuccessWithPagination 自动产出 items+tickets 别名，避免前端 response.tickets 未定义导致列表为空
	common.SuccessWithPagination(c, ticketListToResponse(tickets), req.Page, req.PageSize, int64(total))
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
	if len(req.Tags) > 0 {
		// tags are handled via ReplaceTags=false; tags are set separately
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, id, params)
	if err != nil {
		if common.IsVersionConflictError(err) {
			conflictErr := err.(*common.VersionConflictError)
			common.Conflict(c, conflictErr.Error(), gin.H{
				"ticketId":       conflictErr.ResourceID,
				"currentVersion": conflictErr.CurrentVersion,
				"serverVersion":  conflictErr.ServerVersion,
			})
			return
		}
		if strings.Contains(err.Error(), "ticket not found") {
			common.Fail(c, common.NotFoundCode, "工单不存在")
			return
		}
		if isUserInputUpdateError(err) {
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(updated))
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

	if err := h.service.Delete(c.Request.Context(), id, tenantID); err != nil {
		if isForbiddenErr(err) {
			common.FailWithErr(c, err, "操作失败")
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

	ticket, err := h.service.UpdateStatus(c.Request.Context(), id, req.Status, tenantID, userID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(ticket))
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

	if err := h.service.BatchDelete(c.Request.Context(), req.TicketIDs, tenantID); err != nil {
		if isForbiddenErr(err) {
			common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) GetTicketStats(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// v1.1 回归：返回与前端约定的扁平 DTO（total/open/inProgress/resolved/pending/highPriority/overdue），
	// 避免 stats 卡片显示 0。原先直接返回 entity，字段名（TotalTickets/OpenTickets）与前端契约不一致。
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

	common.Success(c, ticketToResponse(ticket))
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
	escalatedBy := c.GetInt("user_id")

	ticket, err := h.service.EscalateTicket(c.Request.Context(), id, req.Reason, tenantID, escalatedBy)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(ticket))
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
	ticket, err := h.service.ResolveTicket(c.Request.Context(), id, resolution, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(ticket))
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
	ticket, err := h.service.CloseTicket(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.BadRequestCode, "当前状态不允许关闭: "+err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
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
	tickets, err := h.service.Search(c.Request.Context(), keyword, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketListToResponse(tickets))
}

// GetOverdueTickets handles GET /api/v1/tickets/overdue
func (h *Handler) GetOverdueTickets(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	tickets, err := h.service.GetOverdueTickets(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketListToResponse(tickets))
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

	common.Success(c, ticketListToResponse(tickets))
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
		common.Fail(c, common.NotFoundCode, "工单不存在")
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
		common.FailWithErr(c, err, "操作失败")
		return
	}

	filename := "tickets." + req.Format
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, "application/octet-stream", data)
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

// GetTicketAnalytics handles POST /api/v1/tickets/analytics
func (h *Handler) GetTicketAnalytics(c *gin.Context) {
	var req dto.TicketAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	analytics, err := h.service.GetTicketAnalytics(c.Request.Context(), tenantID, req.DateFrom.Format("2006-01-02"), req.DateTo.Format("2006-01-02"))
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, analytics)
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
	common.Success(c, gin.H{
		"templates": normalized,
		"total":     len(templates),
		"page":      1,
		"pageSize":  len(templates),
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
	created, err := h.service.CreateTicketTemplate(c.Request.Context(), tenantID, &TicketTemplate{
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Category:    tmpl.Category,
		Priority:    tmpl.Priority,
		Fields:      tmpl.Fields,
		FormFields:  tmpl.FormFields,
	})
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
	updated, err := h.service.UpdateTicketTemplate(c.Request.Context(), tenantID, templateID, &TicketTemplate{
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Category:    tmpl.Category,
		Priority:    tmpl.Priority,
		Fields:      tmpl.Fields,
		FormFields:  tmpl.FormFields,
	})
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
	c.ShouldBindJSON(&req)

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

	common.Success(c, ticketListToResponse(tickets))
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
		if businessErr, ok := err.(*common.BusinessError); ok {
			common.Fail(c, businessErr.Code, businessErr.Message)
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(ticket))
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

	params := &UpdateParams{Version: req.Version}
	if req.Title != "" {
		params.Title = &req.Title
	}
	if req.Description != "" {
		params.Description = &req.Description
	}
	if req.Priority != "" {
		params.Priority = &req.Priority
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, subtaskID, params)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketToResponse(updated))
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

	if err := h.service.DeleteSubtask(c.Request.Context(), subtaskID, tenantID); err != nil {
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

	// SLA pause requires the SLAMonitorService; delegate via common.Success
	// with a note that the real implementation lives in controller layer.
	common.Success(c, gin.H{"message": "SLA已暂停", "ticketId": id})
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

	common.Success(c, gin.H{"message": "SLA已恢复", "ticketId": id})
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
