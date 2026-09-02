// Package ticket_comment — 工单评论 handler.
// 迁移自 controller/ticket_comment_controller.go，保持原有 API 契约不变。
package ticket_comment

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 TicketCommentService 依赖.
type Handler struct {
	commentService *service.TicketCommentService
	logger         *zap.SugaredLogger
}

// NewHandler 构造 ticket_comment Handler.
func NewHandler(commentService *service.TicketCommentService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		commentService: commentService,
		logger:         logger,
	}
}

// ListTicketComments GET /api/v1/tickets/:id/comments
func (h *Handler) ListTicketComments(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	comments, err := h.commentService.ListTicketComments(c.Request.Context(), ticketID, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket comments", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ListTicketCommentsResponse{
		Items: comments,
		Total: len(comments),
	})
}

// CreateTicketComment POST /api/v1/tickets/:id/comments
func (h *Handler) CreateTicketComment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.CreateTicketCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	req.Normalize()

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	comment, err := h.commentService.CreateTicketComment(c.Request.Context(), ticketID, &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to create ticket comment", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, comment)
}

// UpdateTicketComment PUT /api/v1/tickets/:id/comments/:comment_id
func (h *Handler) UpdateTicketComment(c *gin.Context) {
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	req.Normalize()

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	comment, err := h.commentService.UpdateTicketComment(c.Request.Context(), ticketID, commentID, &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to update ticket comment", "error", err, "ticket_id", ticketID, "comment_id", commentID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, comment)
}

// DeleteTicketComment DELETE /api/v1/tickets/:id/comments/:comment_id
func (h *Handler) DeleteTicketComment(c *gin.Context) {
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
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	err = h.commentService.DeleteTicketComment(c.Request.Context(), ticketID, commentID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to delete ticket comment", "error", err, "ticket_id", ticketID, "comment_id", commentID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}
