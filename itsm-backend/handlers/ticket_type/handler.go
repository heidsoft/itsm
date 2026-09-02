// Package ticket_type — 工单类型 handler.
// 迁移自 controller/ticket_type_controller.go，保持原有 API 契约不变。
package ticket_type

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	ticketTypeService *service.TicketTypeService
	logger            *zap.SugaredLogger
}

func NewHandler(ticketTypeService *service.TicketTypeService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		ticketTypeService: ticketTypeService,
		logger:            logger,
	}
}

// CreateTicketType POST /api/v1/ticket-types
func (h *Handler) CreateTicketType(c *gin.Context) {
	var req dto.CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	ticketType, err := h.ticketTypeService.CreateTicketType(c.Request.Context(), &req, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to create ticket type", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketType)
}

// UpdateTicketType PUT /api/v1/ticket-types/:id
func (h *Handler) UpdateTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	var req dto.UpdateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	ticketType, err := h.ticketTypeService.UpdateTicketType(c.Request.Context(), id, &req, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to update ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticketType)
}

// GetTicketType GET /api/v1/ticket-types/:id
func (h *Handler) GetTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	ticketType, err := h.ticketTypeService.GetTicketType(c.Request.Context(), id, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "工单类型不存在")
		return
	}

	common.Success(c, ticketType)
}

// ListTicketTypes GET /api/v1/ticket-types
func (h *Handler) ListTicketTypes(c *gin.Context) {
	var req dto.ListTicketTypesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	tenantID := c.GetInt("tenant_id")

	response, err := h.ticketTypeService.ListTicketTypes(c.Request.Context(), &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket types", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, response)
}

// DeleteTicketType DELETE /api/v1/ticket-types/:id
func (h *Handler) DeleteTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	err = h.ticketTypeService.DeleteTicketType(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to delete ticket type", "error", err, "id", id, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单类型已归档"})
}

// EnableTicketType POST /api/v1/ticket-types/:id/enable
func (h *Handler) EnableTicketType(c *gin.Context) {
	h.setStatus(c, dto.TicketTypeStatusActive)
}

// DisableTicketType POST /api/v1/ticket-types/:id/disable
func (h *Handler) DisableTicketType(c *gin.Context) {
	h.setStatus(c, dto.TicketTypeStatusInactive)
}

func (h *Handler) setStatus(c *gin.Context, status dto.TicketTypeStatus) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}
	result, err := h.ticketTypeService.SetStatus(c.Request.Context(), id, c.GetInt("tenant_id"), c.GetInt("user_id"), status)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

// CloneTicketType POST /api/v1/ticket-types/:id/clone
func (h *Handler) CloneTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	result, err := h.ticketTypeService.CloneTicketType(c.Request.Context(), id, c.GetInt("tenant_id"), c.GetInt("user_id"), req.Code, req.Name)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

// RestoreTicketType POST /api/v1/ticket-types/:id/restore
func (h *Handler) RestoreTicketType(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单类型ID")
		return
	}
	result, err := h.ticketTypeService.RestoreTicketType(c.Request.Context(), id, c.GetInt("tenant_id"), c.GetInt("user_id"))
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

// ListPresets GET /api/v1/ticket-type-presets
func (h *Handler) ListPresets(c *gin.Context) {
	common.Success(c, h.ticketTypeService.ListPresets())
}

// InstallPreset POST /api/v1/ticket-type-presets/:presetId/install
func (h *Handler) InstallPreset(c *gin.Context) {
	var req dto.InstallTicketTypePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	result, err := h.ticketTypeService.InstallPreset(c.Request.Context(), c.Param("presetId"), &req, c.GetInt("tenant_id"), c.GetInt("user_id"))
	if err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	common.Success(c, result)
}
