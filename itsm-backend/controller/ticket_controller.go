package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketController struct {
	ticketService *service.TicketService
	logger        *zap.SugaredLogger
}

func NewTicketController(ticketService *service.TicketService, logger *zap.SugaredLogger) *TicketController {
	return &TicketController{
		ticketService: ticketService,
		logger:        logger,
	}
}

// CreateTicket 创建工单
// @Summary 创建工单
// @Description 创建新的工单
// @Tags tickets
// @Accept json
// @Produce json
// @Param ticket body dto.CreateTicketRequest true "工单信息"
// @Success 200 {object} common.Response{data=dto.TicketResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets [post]
func (tc *TicketController) CreateTicket(c *gin.Context) {
	var req dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorw("Invalid request parameters", "error", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID（需要认证中间件）
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	requesterID, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, requesterID)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	response := dto.ToTicketResponse(ticket)
	common.Success(c, response)
}

// GetTicket 获取工单详情
// @Summary 获取工单详情
// @Description 根据ID获取工单详细信息
// @Tags tickets
// @Produce json
// @Param id path int true "工单ID"
// @Success 200 {object} common.Response{data=dto.TicketResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets/{id} [get]
func (tc *TicketController) GetTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID格式错误")
		return
	}

	ticket, err := tc.ticketService.GetTicketByID(c.Request.Context(), id)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket", "ticket_id", id, "error", err)
		if err.Error() == "工单不存在" {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	response := dto.ToTicketResponse(ticket)
	common.Success(c, response)
}

// UpdateTicket 更新工单
// @Summary 更新工单
// @Description 更新工单状态和信息
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path int true "工单ID"
// @Param ticket body dto.UpdateTicketRequest true "更新信息"
// @Success 200 {object} common.Response{data=dto.TicketResponse}
// @Failure 400 {object} common.Response
// @Failure 403 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets/{id} [patch]
func (tc *TicketController) UpdateTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID格式错误")
		return
	}

	var req dto.UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorw("Invalid request parameters", "error", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	ticket, err := tc.ticketService.UpdateTicket(c.Request.Context(), id, &req, userIDInt)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket", "ticket_id", id, "error", err)
		if err.Error() == "工单不存在" {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else if err.Error() == "无权限更新此工单" {
			common.Fail(c, common.ForbiddenCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	response := dto.ToTicketResponse(ticket)
	common.Success(c, response)
}

// ApproveTicket 审批工单
// @Summary 审批工单
// @Description 对工单进行审批操作
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path int true "工单ID"
// @Param approval body dto.ApprovalRequest true "审批信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets/{id}/approve [post]
func (tc *TicketController) ApproveTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID格式错误")
		return
	}

	var req dto.ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorw("Invalid request parameters", "error", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	approverID, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	err = tc.ticketService.ApproveTicket(c.Request.Context(), id, &req, approverID)
	if err != nil {
		tc.logger.Errorw("Failed to approve ticket", "ticket_id", id, "error", err)
		if err.Error() == "工单不存在" {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	common.Success(c, gin.H{"message": "审批成功"})
}

// AddComment 添加评论
// @Summary 添加评论
// @Description 为工单添加评论
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path int true "工单ID"
// @Param comment body dto.CommentRequest true "评论内容"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets/{id}/comment [post]
func (tc *TicketController) AddComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID格式错误")
		return
	}

	var req dto.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorw("Invalid request parameters", "error", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	err = tc.ticketService.AddComment(c.Request.Context(), id, &req, userIDInt)
	if err != nil {
		tc.logger.Errorw("Failed to add comment", "ticket_id", id, "error", err)
		if err.Error() == "工单不存在" {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	common.Success(c, gin.H{"message": "评论添加成功"})
}

// GetTickets 获取工单列表
// @Summary 获取工单列表
// @Description 获取工单列表，支持分页和筛选
// @Tags tickets
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "状态筛选"
// @Param priority query string false "优先级筛选"
// @Success 200 {object} common.Response{data=dto.TicketListResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets [get]
func (tc *TicketController) GetTickets(c *gin.Context) {
	// 获取查询参数
	page := 1
	size := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			size = s
		}
	}

	status := c.Query("status")
	priority := c.Query("priority")

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	req := &dto.GetTicketsRequest{
		Page:     page,
		Size:     size,
		Status:   status,
		Priority: priority,
		UserID:   userIDInt,
	}

	result, err := tc.ticketService.GetTickets(c.Request.Context(), req)
	if err != nil {
		tc.logger.Errorw("Failed to get tickets", "error", err)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, result)
}

// UpdateTicketStatus 更新工单状态
// @Summary 更新工单状态
// @Description 更新工单状态
// @Tags tickets
// @Accept json
// @Produce json
// @Param id path int true "工单ID"
// @Param status body dto.UpdateStatusRequest true "状态信息"
// @Success 200 {object} common.Response{data=dto.TicketResponse}
// @Failure 400 {object} common.Response
// @Failure 403 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /tickets/{id}/status [put]
func (tc *TicketController) UpdateTicketStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID格式错误")
		return
	}

	var req dto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tc.logger.Errorw("Invalid request parameters", "error", err)
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
		return
	}

	ticket, err := tc.ticketService.UpdateTicketStatus(c.Request.Context(), id, req.Status, userIDInt)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket status", "ticket_id", id, "error", err)
		if err.Error() == "工单不存在" {
			common.Fail(c, common.NotFoundCode, err.Error())
		} else if err.Error() == "无权限更新此工单" {
			common.Fail(c, common.ForbiddenCode, err.Error())
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	response := dto.ToTicketResponse(ticket)
	common.Success(c, response)
}