package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TicketAssignmentController 工单分配控制器
type TicketAssignmentController struct {
	assignmentService *service.TicketAssignmentService
	logger            *zap.Logger
}

// NewTicketAssignmentController 创建工单分配控制器
func NewTicketAssignmentController(assignmentService *service.TicketAssignmentService, logger *zap.Logger) *TicketAssignmentController {
	return &TicketAssignmentController{
		assignmentService: assignmentService,
		logger:            logger,
	}
}

// AssignTicket 智能分配工单
func (tac *TicketAssignmentController) AssignTicket(c *gin.Context) {
	var req service.AssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.TenantID = tenantID

	assignment, err := tac.assignmentService.AssignTicket(c.Request.Context(), &req)
	if err != nil {
		tac.logger.Error("Failed to assign ticket", zap.Error(err), zap.Int("ticket_id", req.TicketID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, assignment)
}

// GetUserWorkload 获取用户工作负载
func (tac *TicketAssignmentController) GetUserWorkload(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的用户ID")
		return
	}

	workload, err := tac.assignmentService.GetUserWorkload(c.Request.Context(), userID)
	if err != nil {
		tac.logger.Error("Failed to get user workload", zap.Error(err), zap.Int("user_id", userID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, workload)
}

// GetTeamWorkload 获取团队工作负载
func (tac *TicketAssignmentController) GetTeamWorkload(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	workloads, err := tac.assignmentService.GetTeamWorkload(c.Request.Context(), tenantID)
	if err != nil {
		tac.logger.Error("Failed to get team workload", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, workloads)
}

// ReassignTicket 重新分配工单
func (tac *TicketAssignmentController) ReassignTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		NewAssigneeID int    `json:"new_assignee_id" binding:"required"`
		Reason        string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	err = tac.assignmentService.ReassignTicket(c.Request.Context(), ticketID, req.NewAssigneeID, req.Reason)
	if err != nil {
		tac.logger.Error("Failed to reassign ticket", zap.Error(err), zap.Int("ticket_id", ticketID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工单重新分配成功"})
}

// LoadBalance 负载均衡
func (tac *TicketAssignmentController) LoadBalance(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	err := tac.assignmentService.LoadBalance(c.Request.Context(), tenantID)
	if err != nil {
		tac.logger.Error("Failed to load balance", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "负载均衡完成"})
}
