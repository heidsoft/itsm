package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketRatingController struct {
	ratingService *service.TicketRatingService
	logger        *zap.SugaredLogger
}

func NewTicketRatingController(ratingService *service.TicketRatingService, logger *zap.SugaredLogger) *TicketRatingController {
	return &TicketRatingController{
		ratingService: ratingService,
		logger:        logger,
	}
}

// SubmitRating 提交工单评分
// POST /api/v1/tickets/:id/rating
func (trc *TicketRatingController) SubmitRating(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.SubmitTicketRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	rating, err := trc.ratingService.SubmitRating(c.Request.Context(), ticketID, &req, userID, tenantID)
	if err != nil {
		trc.logger.Errorw("Failed to submit rating", "error", err, "ticket_id", ticketID, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rating)
}

// GetRating 获取工单评分
// GET /api/v1/tickets/:id/rating
func (trc *TicketRatingController) GetRating(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	rating, err := trc.ratingService.GetRating(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		trc.logger.Errorw("Failed to get rating", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	if rating == nil {
		common.Success(c, nil)
		return
	}

	common.Success(c, rating)
}

// GetRatingStats 获取评分统计
// GET /api/v1/tickets/rating-stats
func (trc *TicketRatingController) GetRatingStats(c *gin.Context) {
	var req dto.GetRatingStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.TenantID = c.GetInt("tenant_id")

	stats, err := trc.ratingService.GetRatingStats(c.Request.Context(), &req)
	if err != nil {
		trc.logger.Errorw("Failed to get rating stats", "error", err, "tenant_id", req.TenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, stats)
}

