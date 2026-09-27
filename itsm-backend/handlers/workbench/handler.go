package workbench

import (
	"strconv"
	"strings"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for workbench endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new workbench handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Query handles GET /api/v1/workbench
func (h *Handler) Query(c *gin.Context) {
	tenantID, exists := c.Get("tenantID")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "tenant context missing")
		return
	}
	tenantIDInt, ok := tenantID.(int)
	if !ok {
		common.Fail(c, common.InternalErrorCode, "invalid tenant context")
		return
	}

	query := WorkbenchQuery{
		TenantID: tenantIDInt,
		Page:     1,
		PageSize: 20,
	}

	if assigneeIDStr := c.Query("assigneeId"); assigneeIDStr != "" {
		if id, err := strconv.Atoi(assigneeIDStr); err == nil {
			query.AssigneeID = &id
		}
	}

	if phaseStr := c.Query("phase"); phaseStr != "" {
		query.Phase = strings.Split(phaseStr, ",")
	}

	if priorityStr := c.Query("priority"); priorityStr != "" {
		query.Priority = strings.Split(priorityStr, ",")
	}

	if recordTypeStr := c.Query("recordType"); recordTypeStr != "" {
		query.RecordType = strings.Split(recordTypeStr, ",")
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			query.Page = page
		}
	}

	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			query.PageSize = pageSize
		}
	}

	response, err := h.service.Query(c.Request.Context(), query)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "query workbench failed: "+err.Error())
		return
	}

	common.Success(c, response)
}
