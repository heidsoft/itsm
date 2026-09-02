// Package ticket_rating — 工单评分 handler.
// 迁移自 controller/ticket_rating_controller.go，保持原有 API 契约不变。
package ticket_rating

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 TicketRatingService 依赖.
type Handler struct {
	ratingService *service.TicketRatingService
	logger        *zap.SugaredLogger
}

// NewHandler 构造 ticket_rating Handler.
func NewHandler(ratingService *service.TicketRatingService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		ratingService: ratingService,
		logger:        logger,
	}
}

// SubmitRating POST /api/v1/tickets/:id/rating
func (h *Handler) SubmitRating(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.SubmitTicketRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	rating, err := h.ratingService.SubmitRating(c.Request.Context(), ticketID, &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to submit rating", "error", err, "ticket_id", ticketID, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rating)
}

// GetRating GET /api/v1/tickets/:id/rating
func (h *Handler) GetRating(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	rating, err := h.ratingService.GetRating(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get rating", "error", err, "ticket_id", ticketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	if rating == nil {
		common.Success(c, nil)
		return
	}

	common.Success(c, rating)
}

// GetRatingStats GET /api/v1/tickets/rating-stats
func (h *Handler) GetRatingStats(c *gin.Context) {
	var req dto.GetRatingStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	req.TenantID = c.GetInt("tenant_id")

	stats, err := h.ratingService.GetRatingStats(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorw("Failed to get rating stats", "error", err, "tenant_id", req.TenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}
