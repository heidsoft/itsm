package workbench

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"

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
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	// 页长唯一所有者：common.GetPaginationFromQuery（缺省 1/20，只采纳 [1,100]，
	// 越界与非数字回落缺省）。台账 E4-47④：此前 handler 手抄 (0,100] 判定。
	pg := common.GetPaginationFromQuery(c)
	query := WorkbenchQuery{
		TenantID: tenantID,
		Page:     pg.Page,
		PageSize: pg.PageSize,
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

	response, err := h.service.Query(c.Request.Context(), query)
	if err != nil {
		common.RespondError(c, err, "query workbench failed")
		return
	}

	common.Success(c, response)
}
