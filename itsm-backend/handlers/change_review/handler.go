package change_review

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 变更评审组成员名册 HTTP 层（仅名册管理；评审审批流转由审批链引擎驱动）。
type Handler struct {
	svc *service.ChangeReviewService
}

// NewHandler 创建变更评审组 handler
func NewHandler(svc *service.ChangeReviewService, _ *zap.SugaredLogger) *Handler {
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

// ListMembers GET /api/v1/change-review/members?type=REVIEW|EREVIEW
func (h *Handler) ListMembers(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	boardType := c.DefaultQuery("type", "REVIEW")
	if boardType != "REVIEW" && boardType != "EREVIEW" {
		boardType = "REVIEW"
	}
	members, err := h.svc.ListMembers(c.Request.Context(), boardType, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, members)
}

// AddMember POST /api/v1/change-review/members
func (h *Handler) AddMember(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	var req dto.AddChangeReviewMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	member, err := h.svc.AddMember(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, member)
}

// UpdateMember PUT /api/v1/change-review/members/:id
func (h *Handler) UpdateMember(c *gin.Context) {
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
	var req dto.UpdateChangeReviewMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	member, err := h.svc.UpdateMember(c.Request.Context(), id, tenantID, req.Role, req.IsActive)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, member)
}

// RemoveMember DELETE /api/v1/change-review/members/:id
func (h *Handler) RemoveMember(c *gin.Context) {
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
	if err := h.svc.RemoveMember(c.Request.Context(), id, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, gin.H{"deleted": id})
}
