package workbench

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/internal/datascope"
	"itsm-backend/middleware"

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

	// 行级数据权限：跨域工作台会同时列出工单/事件/问题/变更，
	// 若只靠路由层 dashboard:read 兜底，普通角色能看到全租户单据。
	// 词表单一源：internal/datascope（管理角色看全量，其余只看本人发起或受理）。
	role := ""
	if v, exists := c.Get("role"); exists {
		role, _ = v.(string)
	}
	viewerID := 0
	if v, exists := c.Get("user_id"); exists {
		viewerID, _ = v.(int)
	}
	scope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(role) {
		scope = datascope.DataScopeOwnedOrAssigned
	}

	// 页长唯一所有者：common.GetPaginationFromQuery（缺省 1/20，只采纳 [1,100]，
	// 越界与非数字回落缺省），handler 不再自己判 (0,100]。
	pg := common.GetPaginationFromQuery(c)
	query := WorkbenchQuery{
		TenantID:  tenantID,
		Page:      pg.Page,
		PageSize:  pg.PageSize,
		DataScope: scope,
		ViewerID:  viewerID,
		// 域级权限过滤：工作台跨域聚合工单/事件/问题/变更，路由层的
		// dashboard:read 只说明能看仪表盘，不代表能看各业务域。
		Roles: middleware.GetContextRoles(c),
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
