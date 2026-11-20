package release

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReleaseHandler HTTP handlers for release domain
type ReleaseHandler struct {
	logger         *zap.SugaredLogger
	releaseService ReleaseService
}

// NewHandler creates a new release handler
func NewHandler(logger *zap.SugaredLogger, releaseService ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{
		logger:         logger,
		releaseService: releaseService,
	}
}

// ListReleases handles GET /api/v1/releases
//
//	@Summary	获取发布列表
//	@Description	分页获取发布列表，支持按状态与类型过滤。行级数据权限：管理角色可见全租户，其他角色仅可见本人创建或负责的发布；actor 缺失时 fail-closed 返回空集。
//	@Tags	发布管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	page	query	int	false	"页码"	default(1)
//	@Param	pageSize	query	int	false	"每页数量"	default(20)
//	@Param	status	query	string	false	"按状态过滤（draft/scheduled/in-progress/completed/cancelled/failed/rolled_back）"
//	@Param	type	query	string	false	"按发布类型过滤（major/minor/patch/hotfix）"
//	@Success	200	{object}	common.Response{data=dto.ReleaseListResponse}
//	@Failure	401	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases [get]
func (h *ReleaseHandler) ListReleases(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	pg := common.GetPaginationFromQuery(c)
	status := c.Query("status")
	releaseType := c.Query("type")
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得，下传给 service 判定 DataScope。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	releases, err := h.releaseService.ListReleases(c.Request.Context(), tenantID, pg.Page, pg.PageSize, status, releaseType, currentUserID, currentRole)
	if err != nil {
		h.logger.Errorw("List releases failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, releases)
}

// CreateRelease handles POST /api/v1/releases
//
//	@Summary	创建发布
//	@Description	创建发布单，初始状态为 draft。发布编号由服务端按 ID 派生（REL-YYYYMMDD-NNNNNN），不接受客户端传入；关联变更与负责人必须属于同一租户，否则 400。生产配置启用 outbox 时，业务写入与流程启动命令同事务落库，保证「发布已建、审批流必起」。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	request	body	dto.CreateReleaseRequest	true	"创建发布请求"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases [post]
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

// GetRelease handles GET /api/v1/releases/:id
//
//	@Summary	获取发布详情
//	@Description	按 ID 获取发布详情，含创建人与负责人姓名投影。跨租户或不存在的发布统一 404，不暴露 ID 可枚举信号。
//	@Tags	发布管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases/{id} [get]
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

// UpdateRelease handles PUT /api/v1/releases/:id
//
//	@Summary	更新发布
//	@Description	部分更新发布字段（指针字段区分未传与传零值）。行级写权限：仅创建人、负责人或管理员可改，其他角色 403；跨租户/不存在返回 404。关联的变更与负责人须属同一租户。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Param	request	body	dto.UpdateReleaseRequest	true	"更新发布请求"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"行级权限不足"
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases/{id} [put]
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

	release, err := h.releaseService.UpdateRelease(c.Request.Context(), releaseID, tenantID, &req, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		if isForbiddenAppErr(err) {
			common.RespondError(c, err, "操作失败")
			return
		}
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

// UpdateReleaseStatus handles PUT /api/v1/releases/:id/status
//
//	@Summary	更新发布状态
//	@Description	按状态机白名单推进发布状态：draft→cancelled；scheduled→in-progress/cancelled；in-progress→completed/failed/rolled_back/cancelled；completed/cancelled/rolled_back/failed 为终态。排期（draft→scheduled）属审批等效动作，必须走 /approve，不能经本接口绕过 release:approve 门禁。非法流转返回 409（业务码 4090），并发 CAS 冲突也返回 409。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Param	request	body	dto.ReleaseStatusUpdateRequest	true	"状态更新请求"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"行级权限不足"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"非法状态流转或并发冲突（业务码 4090）"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases/{id}/status [put]
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

	release, err := h.releaseService.UpdateReleaseStatus(c.Request.Context(), releaseID, tenantID, string(req.Status), c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		// 状态机白名单拒绝属于与服务器状态相关的业务冲突：按全局错误语义规范
		// 映射 409(4090)，不能伪装成 400 参数错误（触发客户端"修正输入"重试），
		// 更不能掉进 500 兜底（客户端会误判为服务端故障而盲目重试）。
		if errors.Is(err, dto.ErrInvalidReleaseTransition) {
			h.logger.Warnw("Release status transition rejected", "error", err, "release_id", releaseID)
			common.Conflict(c, "当前发布状态不允许此操作", gin.H{"detail": err.Error()})
			return
		}
		if isForbiddenAppErr(err) {
			common.RespondError(c, err, "操作失败")
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

// ApproveRelease handles POST /api/v1/releases/:id/approve
//
//	@Summary	批准发布
//	@Description	批准草稿/失败态发布并进入 scheduled（失败态审批即重新排期）。请求体可选：空 body 表示无审批意见。校验审批人须为本租户有效用户且非发布创建人；先在事务内完成 BPMN 待办任务再推进业务状态，无关联待办则 409，不产生「流程已批、业务未批」的分叉。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Param	request	body	object	false	"审批请求（可选 comment 审批意见）"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"审批人无效"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"状态不允许审批 / 自审批 / 无 BPMN 待办 / 并发冲突"
//	@Router	/api/v1/releases/{id}/approve [post]
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

// RejectRelease handles POST /api/v1/releases/:id/reject
//
//	@Summary	拒绝发布
//	@Description	拒绝草稿/失败态发布，状态置为 cancelled。拒绝原因必填（空原因 400，先于 service 调用拦截，避免无理由拒绝进入审计链）。与批准共用审批人校验与 BPMN 待办桥接。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Param	request	body	object	true	"拒绝请求（必填 reason 拒绝原因）"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response	"缺少拒绝原因或请求体非法"
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"审批人无效"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"状态不允许审批 / 自审批 / 无 BPMN 待办 / 并发冲突"
//	@Router	/api/v1/releases/{id}/reject [post]
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
		common.RespondError(c, err, "操作失败")
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

// RollbackRelease handles POST /api/v1/releases/:id/rollback
//
//	@Summary	回滚发布
//	@Description	将 in-progress / failed 发布推进为 rolled_back。回滚原因必填。经状态机白名单校验，非法流转（如终态发布再回滚）返回 409（业务码 4090）而非 500；行级写守卫仍生效（非创建人/负责人/管理员回滚他人发布返回 403）。
//	@Tags	发布管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Param	request	body	object	true	"回滚请求（必填 reason 回滚原因）"
//	@Success	200	{object}	common.Response{data=dto.ReleaseResponse}
//	@Failure	400	{object}	common.Response	"缺少回滚原因或请求体非法"
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"行级权限不足"
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response	"非法状态流转或并发冲突（业务码 4090）"
//	@Router	/api/v1/releases/{id}/rollback [post]
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
	release, err := h.releaseService.UpdateReleaseStatus(c.Request.Context(), releaseID, tenantID, status, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		// 与 UpdateReleaseStatus 端点保持同一语义：状态机白名单拒绝是 409 业务冲突，
		// 不能让 rollback/cancel/complete 等动作端点掉进 500 兜底。
		if errors.Is(err, dto.ErrInvalidReleaseTransition) {
			h.logger.Warnw("Release action rejected by state machine", "error", err, "release_id", releaseID, "status", status)
			common.Conflict(c, "当前发布状态不允许此操作", gin.H{"detail": err.Error()})
			return
		}
		if isForbiddenAppErr(err) {
			common.RespondError(c, err, "操作失败")
			return
		}
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

// DeleteRelease handles DELETE /api/v1/releases/:id
//
//	@Summary	删除发布
//	@Description	物理删除发布。DELETE 幂等：已删除/不存在/跨租户均视为成功（200），不误报 5001。行级删守卫仍生效——查得到但非创建人/负责人/管理员时返回 403。
//	@Tags	发布管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"发布ID"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	401	{object}	common.Response
//	@Failure	403	{object}	common.Response	"行级权限不足"
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/releases/{id} [delete]
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

	err = h.releaseService.DeleteRelease(c.Request.Context(), releaseID, tenantID, c.GetInt("user_id"), c.GetString("role"))
	if err != nil {
		if isForbiddenAppErr(err) {
			common.RespondError(c, err, "操作失败")
			return
		}
		h.logger.Errorw("Delete release failed", "error", err, "release_id", releaseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetReleaseStats handles GET /api/v1/releases/stats
//
//	@Summary	获取发布统计
//	@Description	按租户统计发布总数与各状态数量（draft/scheduled/in-progress/completed/cancelled/failed/rolled_back）。统计口径 fail-closed：任一计数查询失败即返回 500，绝不把 DB 错误吞成 0 而呈现假健康看板。
//	@Tags	发布管理
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	common.Response{data=dto.ReleaseStatsResponse}
//	@Failure	401	{object}	common.Response
//	@Failure	500	{object}	common.Response	"统计查询失败"
//	@Router	/api/v1/releases/stats [get]
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

// isForbiddenAppErr 判定是否为 403 Forbidden AppError（DataScope 行级守卫）。
// 与 change 域同名助手同实现：FailWithErr 会把 AppError 兜底吞成 5001（错误映射
// 铁律），行级拒绝必须走 RespondError 语义分流映射为 HTTP 403。
func isForbiddenAppErr(err error) bool {
	var appErr *common.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == common.ErrCodeForbidden
	}
	return false
}
