package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketWorkflowController struct {
	workflowService *service.TicketWorkflowService
	logger          *zap.SugaredLogger
}

func NewTicketWorkflowController(workflowService *service.TicketWorkflowService, logger *zap.SugaredLogger) *TicketWorkflowController {
	return &TicketWorkflowController{
		workflowService: workflowService,
		logger:          logger,
	}
}

// AcceptTicket 接单
// @Summary 接单
// @Description 接收并开始处理工单
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.AcceptTicketRequest true "接单请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/accept [post]
func (tc *TicketWorkflowController) AcceptTicket(c *gin.Context) {
	var req dto.AcceptTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.AcceptTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to accept ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "接单成功"})
}

// RejectTicket 驳回工单
// @Summary 驳回工单
// @Description 驳回工单并说明原因
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.RejectTicketRequest true "驳回请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/reject [post]
func (tc *TicketWorkflowController) RejectTicket(c *gin.Context) {
	var req dto.RejectTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.RejectTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to reject ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "驳回成功"})
}

// WithdrawTicket 撤回工单
// @Summary 撤回工单
// @Description 撤回已提交的工单
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.WithdrawTicketRequest true "撤回请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/withdraw [post]
func (tc *TicketWorkflowController) WithdrawTicket(c *gin.Context) {
	var req dto.WithdrawTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.WithdrawTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to withdraw ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "撤回成功"})
}

// ForwardTicket 转发工单
// @Summary 转发工单
// @Description 将工单转发给其他人处理
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.ForwardTicketRequest true "转发请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/forward [post]
func (tc *TicketWorkflowController) ForwardTicket(c *gin.Context) {
	var req dto.ForwardTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.ForwardTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to forward ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "转发成功"})
}

// CCTicket 抄送工单
// @Summary 抄送工单
// @Description 将工单抄送给其他人
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.CCTicketRequest true "抄送请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/cc [post]
func (tc *TicketWorkflowController) CCTicket(c *gin.Context) {
	var req dto.CCTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.CCTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to cc ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "抄送成功"})
}

// ApproveTicket 审批工单
// @Summary 审批工单
// @Description 审批工单（通过/拒绝/委派）
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.ApproveTicketRequest true "审批请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/approve [post]
func (tc *TicketWorkflowController) ApproveTicket(c *gin.Context) {
	var req dto.ApproveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.ApproveTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to approve ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	var message string
	switch req.Action {
	case "approve":
		message = "审批通过"
	case "reject":
		message = "审批拒绝"
	case "delegate":
		message = "已委派"
	default:
		message = "操作成功"
	}

	common.Success(c, gin.H{"message": message})
}

// ResolveTicket 解决工单
// @Summary 解决工单
// @Description 标记工单为已解决
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.ResolveTicketRequest true "解决请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/resolve [post]
func (tc *TicketWorkflowController) ResolveTicket(c *gin.Context) {
	var req dto.ResolveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.ResolveTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to resolve ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单已解决"})
}

// CloseTicket 关闭工单
// @Summary 关闭工单
// @Description 关闭已解决的工单
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.CloseTicketRequest true "关闭请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/close [post]
func (tc *TicketWorkflowController) CloseTicket(c *gin.Context) {
	var req dto.CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.CloseTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to close ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单已关闭"})
}

// ReopenTicket 重开工单
// @Summary 重开工单
// @Description 重新打开已关闭的工单
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param request body dto.ReopenTicketRequest true "重开请求"
// @Success 200 {object} common.Response{data=map[string]string}
// @Router /api/v1/tickets/workflow/reopen [post]
func (tc *TicketWorkflowController) ReopenTicket(c *gin.Context) {
	var req dto.ReopenTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tc.workflowService.ReopenTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to reopen ticket", "error", err, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单已重开"})
}

// GetTicketWorkflowState 获取工单流转状态
// @Summary 获取工单流转状态
// @Description 获取工单的当前流转状态和可执行操作
// @Tags 工单流转
// @Accept json
// @Produce json
// @Param id path int true "工单ID"
// @Success 200 {object} common.Response{data=dto.TicketWorkflowState}
// @Router /api/v1/tickets/:id/workflow/state [get]
func (tc *TicketWorkflowController) GetTicketWorkflowState(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	state, err := tc.workflowService.GetTicketWorkflowState(c.Request.Context(), ticketID, userID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket workflow state", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, state)
}

