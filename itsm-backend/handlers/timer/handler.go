package timer

import (
	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page := parseIntParam(c.DefaultQuery("page", "1"), 1)
	pageSize := parseIntParam(c.DefaultQuery("pageSize", "20"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	filter := service.TimerListFilter{
		TenantID:             c.GetInt("tenant_id"),
		Status:               c.Query("status"),
		TimerType:            c.Query("timerType"),
		ProcessDefinitionKey: c.Query("processDefinitionKey"),
		Page:                 page,
		PageSize:             pageSize,
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		common.InternalError(c, "failed to list timers")
		return
	}
	common.Success(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	timerID := c.Param("id")
	if timerID == "" {
		common.ParamError(c, "timer id is required")
		return
	}

	result, err := h.service.Get(c.Request.Context(), c.GetInt("tenant_id"), timerID)
	if err != nil {
		if IsNotFound(err) {
			common.NotFound(c, "timer not found")
			return
		}
		common.InternalError(c, "failed to get timer")
		return
	}
	common.Success(c, result)
}

func (h *Handler) Stats(c *gin.Context) {
	result, err := h.service.Stats(c.Request.Context(), c.GetInt("tenant_id"))
	if err != nil {
		common.InternalError(c, "failed to get timer stats")
		return
	}
	common.Success(c, result)
}
