package controller

import (
	"fmt"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketController struct {
	ticketService           *service.TicketService
	ticketDependencyService *service.TicketDependencyService
	logger                  *zap.SugaredLogger
}

func NewTicketController(ticketService *service.TicketService, ticketDependencyService *service.TicketDependencyService, logger *zap.SugaredLogger) *TicketController {
	return &TicketController{
		ticketService:           ticketService,
		ticketDependencyService: ticketDependencyService,
		logger:                  logger,
	}
}

// CreateTicket 创建工单
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
	userID := c.GetInt("user_id")
	if userID == 0 {
		// 尝试从中间件获取
		if uid, err := middleware.GetUserID(c); err == nil {
			userID = uid
		}
	}

	// 设置请求者ID为当前用户
	req.RequesterID = userID

	ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticket)
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
		tc.logger.Errorw("Failed to update ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticket)
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

	common.Success(c, ticket)
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

	var req struct {
		AssigneeID int `json:"assignee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	assignedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.AssignTicket(c.Request.Context(), ticketID, req.AssigneeID, tenantID, assignedBy)
	if err != nil {
		tc.logger.Errorw("Failed to assign ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticket)
}

// EscalateTicket 升级工单
func (tc *TicketController) EscalateTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
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

	common.Success(c, ticket)
}

// ResolveTicket 解决工单
func (tc *TicketController) ResolveTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Resolution string `json:"resolution" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	resolvedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.ResolveTicket(c.Request.Context(), ticketID, req.Resolution, tenantID, resolvedBy)
	if err != nil {
		tc.logger.Errorw("Failed to resolve ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticket)
}

// CloseTicket 关闭工单
func (tc *TicketController) CloseTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	closedBy := c.GetInt("user_id")

	ticket, err := tc.ticketService.CloseTicket(c.Request.Context(), ticketID, req.Feedback, tenantID, closedBy)
	if err != nil {
		tc.logger.Errorw("Failed to close ticket", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, ticket)
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

	common.Success(c, tickets)
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

	common.Success(c, tickets)
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

	common.Success(c, tickets)
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
	common.Success(c, templates)
}

// CreateTicketTemplate 创建工单模板
func (tc *TicketController) CreateTicketTemplate(c *gin.Context) {
	var template dto.TicketTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 实现创建工单模板功能
	_, err := tc.ticketService.CreateTicketTemplate(c.Request.Context(), tenantID, template)
	if err != nil {
		tc.logger.Errorw("Create ticket template failed", "error", err, "name", template.Name, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Create ticket template successful", "name", template.Name, "tenant_id", tenantID)
	common.Success(c, gin.H{
		"message":       "工单模板创建成功",
		"template_name": template.Name,
	})
}

// UpdateTicketTemplate 更新工单模板
func (tc *TicketController) UpdateTicketTemplate(c *gin.Context) {
	var template dto.TicketTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 实现更新工单模板功能
	_, err := tc.ticketService.UpdateTicketTemplate(c.Request.Context(), template.ID, template)
	if err != nil {
		tc.logger.Errorw("Update ticket template failed", "error", err, "template_id", template.ID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新模板失败: "+err.Error())
		return
	}

	tc.logger.Infow("Update ticket template successful", "template_id", template.ID, "tenant_id", tenantID)
	common.Success(c, gin.H{
		"message":     "工单模板更新成功",
		"template_id": template.ID,
	})
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

	// 实现删除工单模板功能
	err = tc.ticketService.DeleteTicketTemplate(c.Request.Context(), id)
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

	common.Success(c, ticket)
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

	// 检查parent_ticket_id是否匹配
	if ticket.ParentTicketID == 0 || ticket.ParentTicketID != parentID {
		common.Fail(c, common.ParamErrorCode, "子任务不属于指定的父工单")
		return
	}

	updatedTicket, err := tc.ticketService.UpdateTicket(c.Request.Context(), subtaskID, &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to update subtask", "error", err, "subtask_id", subtaskID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, updatedTicket)
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

	// 检查parent_ticket_id是否匹配
	if ticket.ParentTicketID == 0 || ticket.ParentTicketID != parentID {
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
