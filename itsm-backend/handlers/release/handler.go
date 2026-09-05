package release

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReleaseHandler HTTP handlers for release domain
type ReleaseHandler struct {
	logger         *zap.SugaredLogger
	releaseService *service.ReleaseService
}

// NewHandler creates a new release handler
func NewHandler(logger *zap.SugaredLogger, releaseService *service.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{
		logger:         logger,
		releaseService: releaseService,
	}
}

// ListReleases 获取发布列表
func (h *ReleaseHandler) ListReleases(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")
	releaseType := c.Query("type")
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得，下传给 service 判定 DataScope。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	releases, err := h.releaseService.ListReleases(c.Request.Context(), tenantID, page, pageSize, status, releaseType, currentUserID, currentRole)
	if err != nil {
		h.logger.Errorw("List releases failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, releases)
}

// CreateRelease 创建发布
func (h *ReleaseHandler) CreateRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	release, err := h.releaseService.CreateRelease(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Create release failed", "error", err, "tenant_id", tenantID, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, release)
}

// GetRelease 获取发布详情
func (h *ReleaseHandler) GetRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的发布ID")
		return
	}

	release, err := h.releaseService.GetReleaseByID(c.Request.Context(), releaseID, tenantID)
	if err != nil {
		h.logger.Errorw("Get release failed", "error", err, "release_id", releaseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// UpdateRelease 更新发布
func (h *ReleaseHandler) UpdateRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的发布ID")
		return
	}

	var req dto.UpdateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	release, err := h.releaseService.UpdateRelease(c.Request.Context(), releaseID, tenantID, &req)
	if err != nil {
		h.logger.Errorw("Update release failed", "error", err, "release_id", releaseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// UpdateReleaseStatus 更新发布状态
func (h *ReleaseHandler) UpdateReleaseStatus(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的发布ID")
		return
	}

	var req dto.ReleaseStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	release, err := h.releaseService.UpdateReleaseStatus(c.Request.Context(), releaseID, tenantID, string(req.Status))
	if err != nil {
		// D-5：状态机白名单拒绝属于调用方输入错误，应返回 400 而非 500，
		// 避免客户端把"被安全策略拦截"误判为服务端故障而重试。
		if errors.Is(err, service.ErrInvalidReleaseTransition) {
			h.logger.Warnw("Release status transition rejected", "error", err, "release_id", releaseID)
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}
		h.logger.Errorw("Update release status failed", "error", err, "release_id", releaseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// ApproveRelease 批准发布并进入排期状态（带审批人校验，并桥接 BPMN 待办任务）
func (h *ReleaseHandler) ApproveRelease(c *gin.Context) {
	var req struct {
		Comment string `json:"comment"`
	}
	// approve 的审批意见可选，允许空请求体；非空但格式错误的请求仍应拒绝。
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}
	}
	h.applyReleaseApproval(c, "approve", req.Comment)
}

// RejectRelease 拒绝发布（带审批人校验，并桥接 BPMN 待办任务）
func (h *ReleaseHandler) RejectRelease(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "拒绝原因不能为空")
		return
	}
	h.applyReleaseApproval(c, "reject", req.Reason)
}

// applyReleaseApproval 校验身份后委托服务层处理发布审批（含 BPMN 桥接）
func (h *ReleaseHandler) applyReleaseApproval(c *gin.Context, action, comment string) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || releaseID <= 0 {
		common.ParamError(c, "无效的发布ID")
		return
	}
	release, err := h.releaseService.ApplyReleaseApproval(c.Request.Context(), releaseID, tenantID, userID, action, comment)
	if err != nil {
		h.logger.Errorw("Release approval failed", "error", err, "release_id", releaseID, "action", action)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}
	h.logger.Infow("Release approval completed",
		"release_id", releaseID, "tenant_id", tenantID, "user_id", userID, "action", action)
	common.Success(c, release)
}

// RollbackRelease 回滚发布
func (h *ReleaseHandler) RollbackRelease(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "回滚原因不能为空")
		return
	}
	h.updateReleaseActionStatus(c, string(dto.ReleaseStatusRolledBack), req.Reason)
}

func (h *ReleaseHandler) updateReleaseActionStatus(c *gin.Context, status, reason string) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil || releaseID <= 0 {
		common.ParamError(c, "无效的发布ID")
		return
	}
	release, err := h.releaseService.UpdateReleaseStatus(c.Request.Context(), releaseID, tenantID, status)
	if err != nil {
		h.logger.Errorw("Release action failed", "error", err, "release_id", releaseID, "status", status)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}
	h.logger.Infow("Release action completed", "release_id", releaseID, "tenant_id", tenantID, "status", status, "reason", reason)
	common.Success(c, release)
}

// DeleteRelease 删除发布
func (h *ReleaseHandler) DeleteRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的发布ID")
		return
	}

	err = h.releaseService.DeleteRelease(c.Request.Context(), releaseID, tenantID)
	if err != nil {
		h.logger.Errorw("Delete release failed", "error", err, "release_id", releaseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetReleaseStats 获取发布统计
func (h *ReleaseHandler) GetReleaseStats(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := h.releaseService.GetReleaseStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Get release stats failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}
