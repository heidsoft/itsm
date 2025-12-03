package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketCommentController struct {
	commentService *service.TicketCommentService
	logger          *zap.SugaredLogger
}

func NewTicketCommentController(commentService *service.TicketCommentService, logger *zap.SugaredLogger) *TicketCommentController {
	return &TicketCommentController{
		commentService: commentService,
		logger:          logger,
	}
}

// ListTicketComments 获取工单评论列表
// GET /api/v1/tickets/:id/comments
func (tcc *TicketCommentController) ListTicketComments(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	comments, err := tcc.commentService.ListTicketComments(c.Request.Context(), ticketID, tenantID, userID)
	if err != nil {
		tcc.logger.Errorw("Failed to list ticket comments", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListTicketCommentsResponse{
		Comments: comments,
		Total:    len(comments),
	})
}

// CreateTicketComment 创建工单评论
// POST /api/v1/tickets/:id/comments
func (tcc *TicketCommentController) CreateTicketComment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CreateTicketCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	comment, err := tcc.commentService.CreateTicketComment(c.Request.Context(), ticketID, &req, userID, tenantID)
	if err != nil {
		tcc.logger.Errorw("Failed to create ticket comment", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, comment)
}

// UpdateTicketComment 更新工单评论
// PUT /api/v1/tickets/:id/comments/:comment_id
func (tcc *TicketCommentController) UpdateTicketComment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	commentID, err := strconv.Atoi(c.Param("comment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的评论ID")
		return
	}

	var req dto.UpdateTicketCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	comment, err := tcc.commentService.UpdateTicketComment(c.Request.Context(), ticketID, commentID, &req, userID, tenantID)
	if err != nil {
		tcc.logger.Errorw("Failed to update ticket comment", "error", err, "ticket_id", ticketID, "comment_id", commentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, comment)
}

// DeleteTicketComment 删除工单评论
// DELETE /api/v1/tickets/:id/comments/:comment_id
func (tcc *TicketCommentController) DeleteTicketComment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	commentID, err := strconv.Atoi(c.Param("comment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的评论ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	err = tcc.commentService.DeleteTicketComment(c.Request.Context(), ticketID, commentID, userID, tenantID)
	if err != nil {
		tcc.logger.Errorw("Failed to delete ticket comment", "error", err, "ticket_id", ticketID, "comment_id", commentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

