package operations

import (
	"errors"
	"strconv"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page := positiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := positiveInt(c.DefaultQuery("pageSize", "20"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	result, err := h.service.List(c.Request.Context(), ListRequest{
		TenantID: c.GetInt("tenant_id"), Status: c.Query("status"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		common.InternalError(c, "failed to list operational commands")
		return
	}
	common.Success(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := commandID(c)
	if !ok {
		return
	}
	result, err := h.service.Get(c.Request.Context(), c.GetInt("tenant_id"), id)
	h.respond(c, result, err)
}

func (h *Handler) Replay(c *gin.Context) { h.mutate(c, true) }
func (h *Handler) Cancel(c *gin.Context) { h.mutate(c, false) }

func (h *Handler) mutate(c *gin.Context, replay bool) {
	id, ok := commandID(c)
	if !ok {
		return
	}
	actor := Actor{
		UserID: c.GetInt("user_id"), RequestID: c.GetString("request_id"),
		IP: c.ClientIP(), Path: c.Request.URL.Path, Method: c.Request.Method,
	}
	var result *CommandDTO
	var err error
	if replay {
		result, err = h.service.Replay(c.Request.Context(), c.GetInt("tenant_id"), id, actor)
	} else {
		result, err = h.service.Cancel(c.Request.Context(), c.GetInt("tenant_id"), id, actor)
	}
	h.respond(c, result, err)
}

func (h *Handler) respond(c *gin.Context, result *CommandDTO, err error) {
	switch {
	case err == nil:
		common.Success(c, result)
	case errors.Is(err, ErrCommandNotFound):
		common.NotFound(c, "operational command not found")
	case errors.Is(err, ErrInvalidState), errors.Is(err, ErrConcurrentWrite):
		common.Fail(c, common.ConflictCode, "operational command state changed; refresh and retry")
	default:
		common.InternalError(c, "operational command operation failed")
	}
}

func commandID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "invalid command id")
		return 0, false
	}
	return id, true
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
