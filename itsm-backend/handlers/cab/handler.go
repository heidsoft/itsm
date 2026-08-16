package cab

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler CAB 成员名册 HTTP 层（仅名册管理；CAB 审批流转由审批链引擎驱动）。
type Handler struct {
	svc *service.CABService
}

// NewHandler 创建 CAB handler
func NewHandler(svc *service.CABService, _ *zap.SugaredLogger) *Handler {
	return &Handler{svc: svc}
}

// tenantIDFromCtx 从 gin 上下文提取租户ID（鉴权中间件已注入）。
func tenantIDFromCtx(c *gin.Context) (int, bool) {
	v, ok := c.Get("tenant_id")
	if !ok {
		return 0, false
	}
	tid, ok := v.(int)
	return tid, ok
}

// ListCABMembers GET /api/v1/cab/members?type=CAB|ECAB
func (h *Handler) ListCABMembers(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	boardType := c.DefaultQuery("type", "CAB")
	if boardType != "CAB" && boardType != "ECAB" {
		boardType = "CAB"
	}
	members, err := h.svc.ListCABMembers(c.Request.Context(), boardType, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, members)
}

// AddCABMember POST /api/v1/cab/members
func (h *Handler) AddCABMember(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	var req dto.AddCABMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, err.Error())
		return
	}
	member, err := h.svc.AddCABMember(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, member)
}

// UpdateCABMember PUT /api/v1/cab/members/:id
func (h *Handler) UpdateCABMember(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的成员ID")
		return
	}
	var req dto.UpdateCABMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, err.Error())
		return
	}
	member, err := h.svc.UpdateCABMember(c.Request.Context(), id, tenantID, req.Role, req.IsActive)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, member)
}

// RemoveCABMember DELETE /api/v1/cab/members/:id
func (h *Handler) RemoveCABMember(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的成员ID")
		return
	}
	if err := h.svc.RemoveCABMember(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"deleted": id})
}
