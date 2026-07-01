package controller

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/repository/ticket"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketController struct {
	ticketService           *service.TicketService
	ticketDependencyService *service.TicketDependencyService
	db                      *sql.DB
	logger                  *zap.SugaredLogger
}

func NewTicketController(ticketService *service.TicketService, ticketDependencyService *service.TicketDependencyService, db *sql.DB, logger *zap.SugaredLogger) *TicketController {
	return &TicketController{
		ticketService:           ticketService,
		ticketDependencyService: ticketDependencyService,
		db:                      db,
		logger:                  logger,
	}
}

// ticketToResponse 将 V2 领域模型 ticket.Ticket 转换为 DTO TicketResponse
// V2 负责业务编排，DTO 负责接口呈现；中间需要一层 adapter
func ticketToResponse(t *ticket.Ticket) *dto.TicketResponse {
	if t == nil {
		return nil
	}
	resp := &dto.TicketResponse{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Description:  t.Description,
		Status:       string(t.Status),
		Priority:     string(t.Priority),
		Type:         string(t.Type),
		RequesterID:  t.RequesterID,
		TenantID:     t.TenantID,
		Version:      t.Version,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.AssigneeID != nil {
		resp.AssigneeID = *t.AssigneeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = *t.CategoryID
	}
	return resp
}

func ticketListToResponse(ts []*ticket.Ticket) []*dto.TicketResponse {
	result := make([]*dto.TicketResponse, 0, len(ts))
	for _, t := range ts {
		if r := ticketToResponse(t); r != nil {
			result = append(result, r)
		}
	}
	return result
}

// CreateTicket 创建工单
// @Summary 创建工单
// @Description 创建新的工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body dto.CreateTicketRequest true "创建工单请求"
// @Success 200 {object} common.Response
// @Router /api/v1/tickets [post]
func (tc *TicketController) CreateTicket(c *gin.Context) {
	var req dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取租户ID（从中间件注入）
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		// 尝试从TenantContext获取
		if tenantCtx, ok := middleware.GetTenantContext(c); ok {
			tenantID = tenantCtx.TenantID
		}
	}
	if tenantID == 0 {
		common.Fail(c, common.ParamErrorCode, "租户ID无效")
		return
	}

	userID := c.GetInt("user_id")
	if userID == 0 {
		// 尝试从中间件获取
		if uid, err := middleware.GetUserID(c); err == nil {
			userID = uid
		}
	}
	if userID == 0 {
		common.Fail(c, common.ParamErrorCode, "无法获取当前用户信息，请重新登录")
		return
	}

	// 设置请求者ID为当前用户
	req.RequesterID = userID

	ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// UpdateTicket 更新工单
func (tc *TicketController) UpdateTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.UserID = c.GetInt("user_id")

	ticket, err := tc.ticketService.UpdateTicket(c.Request.Context(), ticketID, &req, tenantID)
	if err != nil {
		// 处理版本冲突错误
		if common.IsVersionConflictError(err) {
			conflictErr := err.(*common.VersionConflictError)
			tc.logger.Warnw("Version conflict", "error", err, "ticket_id", ticketID)
			common.Conflict(c, conflictErr.Error(), gin.H{
				"ticketId":       conflictErr.ResourceID,
				"currentVersion": conflictErr.CurrentVersion,
				"serverVersion":  conflictErr.ServerVersion,
			})
			return
		}
		tc.logger.Errorw("Failed to update ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// GetTicket 获取工单详情
func (tc *TicketController) GetTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	ticket, err := tc.ticketService.GetTicket(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "工单不存在")
		return
	}
	resp := ticketToResponse(ticket)
	dto.EnrichTicketResponse(c.Request.Context(), tc.db, resp, tenantID)
	common.Success(c, resp)
}

// GetTicketSLAInfo 获取工单SLA信息
func (tc *TicketController) GetTicketSLAInfo(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	slaInfo, err := tc.ticketService.GetTicketSLAInfo(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket SLA info", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "工单不存在")
		return
	}

	common.Success(c, slaInfo)
}

// ListTickets 获取工单列表
func (tc *TicketController) ListTickets(c *gin.Context) {
	var req dto.ListTicketsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	response, err := tc.ticketService.ListTickets(c.Request.Context(), &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to list tickets", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, response)
}

// DeleteTicket 删除工单
func (tc *TicketController) DeleteTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = tc.ticketService.DeleteTicket(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to delete ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单删除成功"})
}

// UpdateTicketStatus 更新工单状态
func (tc *TicketController) UpdateTicketStatus(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	ticket, err := tc.ticketService.UpdateTicketStatus(c.Request.Context(), ticketID, req.Status, tenantID, userID)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket status", "error", err, "ticket_id", ticketID, "tenant_id", tenantID, "status", req.Status, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// BatchDeleteTickets 批量删除工单
func (tc *TicketController) BatchDeleteTickets(c *gin.Context) {
	var req dto.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	err := tc.ticketService.BatchDeleteTickets(c.Request.Context(), req.TicketIDs, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to batch delete tickets", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":       "批量删除成功",
		"deleted_count": len(req.TicketIDs),
	})
}

// GetTicketStats 获取工单统计
func (tc *TicketController) GetTicketStats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	stats, err := tc.ticketService.GetTicketStats(c.Request.Context(), tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket stats", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, stats)
}

// AssignTicket 分配工单
func (tc *TicketController) AssignTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	// 兼容 snake_case 字段名 assignee_id
	assigneeID := req.AssigneeID
	if assigneeID == 0 && req.AssigneeIDAlt != 0 {
		assigneeID = req.AssigneeIDAlt
	}
	if assigneeID <= 0 {
		common.Fail(c, common.ParamErrorCode, "assigneeId 必填")
		return
	}

	tenantID := c.GetInt("tenant_id")
	assignedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.AssignTicket(c.Request.Context(), ticketID, assigneeID, tenantID)
	_ = assignedBy // V2 简化：审计参数由 repository / notification 服务处理
	if err != nil {
		tc.logger.Errorw("Failed to assign ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// EscalateTicket 升级工单
func (tc *TicketController) EscalateTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.EscalateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	escalatedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.EscalateTicket(c.Request.Context(), ticketID, req.Reason, tenantID, escalatedBy)
	if err != nil {
		tc.logger.Errorw("Failed to escalate ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// ResolveTicket 解决工单
func (tc *TicketController) ResolveTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.ResolveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	resolvedBy := c.GetInt("user_id")

	// 兼容前端发送的 solution 字段
	resolution := req.Resolution
	if resolution == "" {
		resolution = req.Solution
	}

	ticket, err := tc.ticketService.ResolveTicket(c.Request.Context(), ticketID, resolution, tenantID)
	_ = resolvedBy // V2 简化
	if err != nil {
		tc.logger.Errorw("Failed to resolve ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// CloseTicket 关闭工单
func (tc *TicketController) CloseTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	closedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.CloseTicket(c.Request.Context(), ticketID, tenantID, req.Feedback)
	_ = closedBy // V2 简化
	if err != nil {
		tc.logger.Errorw("Failed to close ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// SearchTickets 搜索工单
func (tc *TicketController) SearchTickets(c *gin.Context) {
	searchTerm := c.Query("q")
	if searchTerm == "" {
		common.Fail(c, common.ParamErrorCode, "搜索关键词不能为空")
		return
	}

	tenantID := c.GetInt("tenant_id")

	tickets, err := tc.ticketService.SearchTickets(c.Request.Context(), searchTerm, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to search tickets", "error", err, "search_term", searchTerm, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketListToResponse(tickets))
}

// GetOverdueTickets 获取逾期工单
func (tc *TicketController) GetOverdueTickets(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	tickets, err := tc.ticketService.GetOverdueTickets(c.Request.Context(), tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get overdue tickets", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketListToResponse(tickets))
}

// GetTicketsByAssignee 获取指定处理人的工单
func (tc *TicketController) GetTicketsByAssignee(c *gin.Context) {
	assigneeID, err := strconv.Atoi(c.Param("assignee_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的处理人ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	tickets, err := tc.ticketService.GetTicketsByAssignee(c.Request.Context(), assigneeID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get tickets by assignee", "error", err, "assignee_id", assigneeID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketListToResponse(tickets))
}

// GetTicketActivity 获取工单活动日志
func (tc *TicketController) GetTicketActivity(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	activities, err := tc.ticketService.GetTicketActivity(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket activity", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, activities)
}

// ExportTickets 导出工单
func (tc *TicketController) ExportTickets(c *gin.Context) {
	var req dto.TicketExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 实现导出功能
	filters := map[string]interface{}{
		"status":   req.Filters.Status,
		"priority": req.Filters.Priority,
	}
	data, err := tc.ticketService.ExportTickets(c.Request.Context(), tenantID, filters, req.Format)
	if err != nil {
		tc.logger.Errorw("Export tickets failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "导出失败: "+err.Error())
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("tickets_%s.%s", time.Now().Format("20060102"), req.Format)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", data)
}

// ImportTickets 导入工单
func (tc *TicketController) ImportTickets(c *gin.Context) {
	var req dto.TicketImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 解析文件数据
	fileData := []byte(req.File) // 这里应该从文件上传中获取

	// 实现导入功能
	err := tc.ticketService.ImportTickets(c.Request.Context(), tenantID, fileData, req.Format)
	if err != nil {
		tc.logger.Errorw("Import tickets failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "导入失败: "+err.Error())
		return
	}

	tc.logger.Infow("Import tickets successful", "format", req.Format, "tenant_id", tenantID)
	common.Success(c, gin.H{
		"message": "工单导入成功",
		"format":  req.Format,
	})
}

// AssignTickets 分配工单
func (tc *TicketController) AssignTickets(c *gin.Context) {
	var req dto.TicketAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 实现工单分配功能
	err := tc.ticketService.AssignTickets(c.Request.Context(), tenantID, req.TicketIDs, req.AssigneeID)
	if err != nil {
		tc.logger.Errorw("Assign tickets failed", "error", err, "ticket_ids", req.TicketIDs, "assignee_id", req.AssigneeID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "分配失败: "+err.Error())
		return
	}

	tc.logger.Infow("Assign tickets successful", "ticket_ids", req.TicketIDs, "assignee_id", req.AssigneeID, "tenant_id", tenantID)
	common.Success(c, gin.H{
		"message":        "工单分配成功",
		"assigned_count": len(req.TicketIDs),
	})
}

// GetTicketAnalytics 获取工单分析
func (tc *TicketController) GetTicketAnalytics(c *gin.Context) {
	var req dto.TicketAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 实现工单分析功能
	analytics, err := tc.ticketService.GetTicketAnalytics(c.Request.Context(), tenantID, req.DateFrom, req.DateTo)
	if err != nil {
		tc.logger.Errorw("Get ticket analytics failed", "error", err, "date_from", req.DateFrom, "date_to", req.DateTo, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取分析数据失败: "+err.Error())
		return
	}

	tc.logger.Infow("Get ticket analytics successful", "date_from", req.DateFrom, "date_to", req.DateTo, "tenant_id", tenantID)
	common.Success(c, analytics)
}

// GetTicketTemplates 获取工单模板
func (tc *TicketController) GetTicketTemplates(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		// 尝试从TenantContext获取
		if tenantCtx, ok := middleware.GetTenantContext(c); ok {
			tenantID = tenantCtx.TenantID
		}
	}

	// 获取真实模板数据
	templates, err := tc.ticketService.GetTicketTemplates(c.Request.Context(), tenantID)
	if err != nil {
		tc.logger.Errorw("Get ticket templates failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Get ticket templates successful", "tenant_id", tenantID, "count", len(templates))
	page := 1
	pageSize := len(templates)
	if pageSize == 0 {
		pageSize = 100
	}
	common.Success(c, gin.H{
		"templates": normalizeTicketTemplateList(templates),
		"total":     len(templates),
		"page":      page,
		"page_size": pageSize,
		"pageSize":  pageSize,
	})
}

// GetTicketTemplate 获取单个工单模板
func (tc *TicketController) GetTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	template, err := tc.ticketService.GetTicketTemplate(c.Request.Context(), tenantID, templateID)
	if err != nil {
		tc.logger.Errorw("Get ticket template failed", "error", err, "template_id", templateID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取模板失败: "+err.Error())
		return
	}

	common.Success(c, normalizeTicketTemplate(template))
}

// CreateTicketTemplate 创建工单模板
func (tc *TicketController) CreateTicketTemplate(c *gin.Context) {
	var template dto.TicketTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 实现创建工单模板功能
	created, err := tc.ticketService.CreateTicketTemplate(c.Request.Context(), tenantID, &template)
	if err != nil {
		tc.logger.Errorw("Create ticket template failed", "error", err, "name", template.Name, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Create ticket template successful", "name", template.Name, "tenant_id", tenantID)
	common.Success(c, normalizeTicketTemplate(created))
}

// UpdateTicketTemplate 更新工单模板
func (tc *TicketController) UpdateTicketTemplate(c *gin.Context) {
	templateIDStr := c.Param("id")
	templateID, err := strconv.Atoi(templateIDStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var template dto.TicketTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 实现更新工单模板功能
	updated, err := tc.ticketService.UpdateTicketTemplate(c.Request.Context(), tenantID, templateID, &template)
	if err != nil {
		tc.logger.Errorw("Update ticket template failed", "error", err, "template_id", templateID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Update ticket template successful", "template_id", templateID, "tenant_id", tenantID)
	common.Success(c, normalizeTicketTemplate(updated))
}

// DeleteTicketTemplate 删除工单模板
func (tc *TicketController) DeleteTicketTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		common.Fail(c, common.ParamErrorCode, "模板ID不能为空")
		return
	}

	id, err := strconv.Atoi(templateID)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 实现删除工单模板功能
	err = tc.ticketService.DeleteTicketTemplate(c.Request.Context(), tenantID, id)
	if err != nil {
		tc.logger.Errorw("Delete ticket template failed", "error", err, "template_id", id, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "删除模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Delete ticket template successful", "template_id", id, "tenant_id", tenantID)
	common.Success(c, gin.H{
		"message":     "工单模板删除成功",
		"template_id": id,
	})
}

// UpdateTicketTemplateStatus 启用或停用工单模板
func (tc *TicketController) UpdateTicketTemplateStatus(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var req struct {
		IsActive    *bool `json:"isActive"`
		IsActiveAlt *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	isActive := req.IsActive
	if isActive == nil {
		isActive = req.IsActiveAlt
	}
	if isActive == nil {
		common.Fail(c, common.ParamErrorCode, "is_active is required")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	template, err := tc.ticketService.UpdateTicketTemplateStatus(c.Request.Context(), tenantID, templateID, *isActive)
	if err != nil {
		tc.logger.Errorw("Update ticket template status failed", "error", err, "template_id", templateID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新模板状态失败: "+err.Error())
		return
	}

	common.Success(c, normalizeTicketTemplate(template))
}

// CopyTicketTemplate 复制工单模板
func (tc *TicketController) CopyTicketTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的模板ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	template, err := tc.ticketService.CopyTicketTemplate(c.Request.Context(), tenantID, templateID, req.Name)
	if err != nil {
		tc.logger.Errorw("Copy ticket template failed", "error", err, "template_id", templateID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "复制模板失败: "+err.Error())
		return
	}

	common.Success(c, normalizeTicketTemplate(template))
}

// GetTicketTemplateCategories 获取工单模板分类
func (tc *TicketController) GetTicketTemplateCategories(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	categories, err := tc.ticketService.GetTicketTemplateCategories(c.Request.Context(), tenantID)
	if err != nil {
		tc.logger.Errorw("Get ticket template categories failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取模板分类失败: "+err.Error())
		return
	}

	common.Success(c, categories)
}

func normalizeTicketTemplateList(templates []interface{}) []gin.H {
	result := make([]gin.H, 0, len(templates))
	for _, template := range templates {
		result = append(result, normalizeTicketTemplate(template))
	}
	return result
}

func normalizeTicketTemplate(template interface{}) gin.H {
	tmpl, ok := template.(*dto.TicketTemplate)
	if !ok || tmpl == nil {
		return gin.H{}
	}
	formFields := tmpl.FormFields
	if formFields == nil {
		formFields = tmpl.FormFieldsAlt
	}
	if formFields == nil {
		formFields = gin.H{}
	}
	isActive := tmpl.IsActive
	if tmpl.IsActiveAlt != nil {
		isActive = *tmpl.IsActiveAlt
	}
	return gin.H{
		"id":             tmpl.ID,
		"name":           tmpl.Name,
		"description":    tmpl.Description,
		"category":       tmpl.Category,
		"priority":       tmpl.Priority,
		"fields":         tmpl.Fields,
		"formFields":     formFields,
		"form_fields":    formFields,
		"workflow_steps": tmpl.WorkflowSteps,
		"isActive":       isActive,
		"is_active":      isActive,
		"createdAt":      tmpl.CreatedAt,
		"created_at":     tmpl.CreatedAt,
		"updatedAt":      tmpl.UpdatedAt,
		"updated_at":     tmpl.UpdatedAt,
	}
}

// GetSubtasks 获取子任务列表
func (tc *TicketController) GetSubtasks(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 查询所有parent_ticket_id等于当前工单ID的工单
	tickets, err := tc.ticketService.ListTickets(c.Request.Context(), &dto.ListTicketsRequest{
		ParentTicketID: &parentID,
	}, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get subtasks", "error", err, "parent_id", parentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tickets.Tickets)
}

// CreateSubtask 创建子任务
func (tc *TicketController) CreateSubtask(c *gin.Context) {
	parentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	// 设置父工单ID和请求者ID
	req.ParentTicketID = &parentID
	req.RequesterID = userID

	ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to create subtask", "error", err, "parent_id", parentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(ticket))
}

// UpdateSubtask 更新子任务
func (tc *TicketController) UpdateSubtask(c *gin.Context) {
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
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.UserID = c.GetInt("user_id")

	// 验证子任务是否属于指定的父工单
	ticket, err := tc.ticketService.GetTicket(c.Request.Context(), subtaskID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get subtask", "error", err, "subtask_id", subtaskID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "子任务不存在")
		return
	}

	// 检查parent_ticket_id是否匹配（V2 是 *int）
	if ticket.ParentTicketID == nil || *ticket.ParentTicketID != parentID {
		common.Fail(c, common.ParamErrorCode, "子任务不属于指定的父工单")
		return
	}

	updatedTicket, err := tc.ticketService.UpdateTicket(c.Request.Context(), subtaskID, &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to update subtask", "error", err, "subtask_id", subtaskID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticketToResponse(updatedTicket))
}

// DeleteSubtask 删除子任务
func (tc *TicketController) DeleteSubtask(c *gin.Context) {
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

	tenantID := c.GetInt("tenant_id")

	// 验证子任务是否属于指定的父工单
	ticket, err := tc.ticketService.GetTicket(c.Request.Context(), subtaskID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get subtask", "error", err, "subtask_id", subtaskID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "子任务不存在")
		return
	}

	// 检查parent_ticket_id是否匹配（V2 是 *int）
	if ticket.ParentTicketID == nil || *ticket.ParentTicketID != parentID {
		common.Fail(c, common.ParamErrorCode, "子任务不属于指定的父工单")
		return
	}

	err = tc.ticketService.DeleteTicket(c.Request.Context(), subtaskID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to delete subtask", "error", err, "subtask_id", subtaskID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "子任务删除成功"})
}
