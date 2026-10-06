package common

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func shouldUseSecureCookies(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}

	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func cookieDomain(_ *gin.Context) string {
	// Frontend now calls the backend through same-origin /api proxy, so host-only
	// cookies are the safest default for both localhost and production domains.
	return ""
}

// setSessionCookies 写入服务端签发的凭证。Max-Age 全部从 middleware 的 TTL 常量派生，
// 避免浏览器 cookie 生命周期与服务端校验窗口各自硬编码后分叉。
func setSessionCookies(c *gin.Context, res *AuthResult) {
	secure := shouldUseSecureCookies(c)
	domain := cookieDomain(c)
	// SameSite=Lax 防止跨站请求携带认证 cookie（CSRF 防护）；Secure 仅在生产 HTTPS 下启用。
	// 浏览器会拒绝明文 HTTP 上的 Secure cookie，因此本地开发保持 host-only 且不带 Secure。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookie, res.AccessToken, res.ExpiresIn, "/", domain, secure, true)
	if res.RefreshToken != "" {
		c.SetCookie(middleware.RefreshTokenCookie, res.RefreshToken, int(middleware.RefreshTokenTTL.Seconds()), "/", domain, secure, true)
	}
}

// clearSessionCookies 无条件清除浏览器凭证。登出的第一步必须成功：吊销存储故障时
// 先清 cookie 至少保证本浏览器不再自动续签，剩余风险由响应明确暴露。
func clearSessionCookies(c *gin.Context) {
	secure := shouldUseSecureCookies(c)
	domain := cookieDomain(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookie, "", -1, "/", domain, secure, true)
	c.SetCookie(middleware.RefreshTokenCookie, "", -1, "/", domain, secure, true)
}

// requestAccessToken 取原始 access token。AuthMiddleware 在时读它写入的上下文值；
// 登出这类不再要求有效凭证的路由回退到 cookie 与 Authorization header。
func requestAccessToken(c *gin.Context) string {
	if tok := c.GetString("token"); tok != "" {
		return tok
	}
	if tok, err := c.Cookie(middleware.AccessTokenCookie); err == nil && tok != "" {
		return tok
	}
	if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

// Auth

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		TenantID   int    `json:"tenantId"`
		TenantCode string `json:"tenantCode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	res, err := h.svc.Login(auditCtx, req.Username, req.Password, req.TenantID, req.TenantCode)
	if err != nil {
		// 重名账号需要租户消歧属于请求参数问题，和凭证错误分开返回；
		// 两者不能混成一个消息，否则前端无法引导用户补租户。
		if errors.Is(err, ErrLoginTenantRequired) {
			h.svc.logger.Warnw("login requires tenant disambiguation")
			common.ParamError(c, "该用户名在多个租户下存在，请选择租户后登录")
			return
		}
		h.svc.logger.Errorw("login failed", "error", err)
		common.AuthFailed(c, "用户名或密码错误")
		return
	}

	setSessionCookies(c, res)

	common.Success(c, res)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	// 浏览器端 refresh_token 存于 httpOnly cookie，JS 无法读取，前端以空 body 调用；
	// 此处回退读取 cookie。否则刷新永远 400，access_token(15min) 过期后会话必然丢失。
	_ = c.ShouldBindJSON(&req)
	if req.RefreshToken == "" {
		if cookieToken, err := c.Cookie(middleware.RefreshTokenCookie); err == nil && cookieToken != "" {
			req.RefreshToken = cookieToken
		}
	}
	if req.RefreshToken == "" {
		common.ParamError(c, "参数错误: refreshToken 不能为空")
		return
	}

	res, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.svc.logger.Errorw("refresh token failed", "error", err)
		common.AuthFailed(c, "登录状态已过期，请重新登录")
		return
	}

	setSessionCookies(c, res)

	common.Success(c, res)
}

// Logout 清除浏览器凭证并吊销本次会话的两类 token。
// 本路由刻意不挂 AuthMiddleware：access token 过期（15 分钟）恰恰是最常见的登出场景，
// 若要求有效凭证，登出会 401、7 天的 refresh cookie 原样留在浏览器并把用户重新登录进去。
func (h *Handler) Logout(c *gin.Context) {
	accessTok := requestAccessToken(c)
	refreshTok, _ := c.Cookie(middleware.RefreshTokenCookie)
	clearSessionCookies(c)

	ctx := c.Request.Context()
	var revokeErr error
	if accessTok != "" {
		if claims, err := middleware.ValidateAccessToken(accessTok, h.svc.jwtSecret); err == nil && claims.ExpiresAt != nil {
			if err := middleware.RevokeAccessToken(ctx, accessTok, claims.ExpiresAt.Time); err != nil {
				revokeErr = err
			}
		}
	}
	if refreshTok != "" {
		// refresh token 必须在登出时吊销，否则泄露/遗留的 7 天凭证仍可换新 access token。
		if claims, err := middleware.ValidateRefreshToken(refreshTok, h.svc.jwtSecret); err == nil && claims.ExpiresAt != nil {
			if _, err := middleware.RevokeRefreshToken(ctx, refreshTok, claims.ExpiresAt.Time); err != nil {
				revokeErr = err
			}
		}
	}
	if revokeErr != nil {
		h.svc.logger.Errorw("failed to revoke session tokens on logout", "error", revokeErr)
		common.Fail(c, common.ServiceUnavailableCode, "会话凭证已清除，但服务端吊销未完成；请稍后重试，或由管理端强制下线该账号")
		return
	}

	common.Success(c, nil)
}

// accessTokenRemainingSeconds 返回当前 access token 的剩余有效期（服务端时钟，秒）。
// 前端据此安排刷新，不再依赖浏览器时间或固定间隔推断会话是否即将过期。
func (h *Handler) accessTokenRemainingSeconds(c *gin.Context) int {
	tok := requestAccessToken(c)
	if tok == "" {
		return 0
	}
	claims, err := middleware.ValidateAccessToken(tok, h.svc.jwtSecret)
	if err != nil || claims.ExpiresAt == nil {
		return 0
	}
	if remaining := int(time.Until(claims.ExpiresAt.Time).Seconds()); remaining > 0 {
		return remaining
	}
	return 0
}

// GetSession 返回前端唯一的会话真相。
// @Summary 获取当前会话
// @Description 返回认证用户身份、可切换租户列表，以及服务端时钟计算的 access token 剩余秒数
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=common.SessionResponse}
// @Failure 401 {object} common.Response
// @Router /api/v1/auth/session [get]
func (h *Handler) GetSession(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetSession(c.Request.Context(), userID, tenantID, h.accessTokenRemainingSeconds(c))
	if err != nil {
		h.svc.logger.Errorw("failed to resolve session identity", "user_id", userID, "error", err)
		common.AuthFailed(c, "会话不可用，请重新登录")
		return
	}
	common.Success(c, res)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")
	u, err := h.svc.GetUser(c.Request.Context(), userID, tenantID)
	if err != nil {
		common.NotFound(c, "User not found")
		return
	}
	common.Success(c, u)
}

// GetUserTenants 获取用户所属的租户列表（前端登录后需要）
func (h *Handler) GetUserTenants(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenants, err := h.svc.GetUserTenants(c.Request.Context(), userID)
	if err != nil {
		common.RespondError(c, err, "获取用户租户列表失败")
		return
	}
	common.Success(c, gin.H{"items": tenants})
}

func (h *Handler) ListUsers(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	users, err := h.svc.ListUsers(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取用户列表失败")
		return
	}
	common.Success(c, users)
}

// Departments

// GetDepartment 获取部门详情
// @Summary 获取部门详情
// @Description 按 ID 读取当前租户下的部门
// @Tags 组织管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/v1/org/departments/{id} [get]
func (h *Handler) GetDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(c, "无效的部门ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	dept, err := h.svc.GetDepartment(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "获取部门失败")
		return
	}
	common.Success(c, dept)
}

// GetDepartmentTree 获取部门树
// @Summary 获取部门树
// @Description 返回当前租户的部门层级树
// @Tags 组织管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response
// @Router /api/v1/org/departments/tree [get]
// @Router /api/v1/departments/tree [get]
func (h *Handler) GetDepartmentTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tree, err := h.svc.GetDepartmentTree(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取部门树失败")
		return
	}
	common.Success(c, tree)
}

// ListDepartments 获取部门列表
// @Summary 获取部门列表
// @Description 返回当前租户的平铺部门列表
// @Tags 组织管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response
// @Router /api/v1/departments [get]
func (h *Handler) ListDepartments(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	deps, err := h.svc.ListDepartments(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取部门列表失败")
		return
	}
	common.Success(c, deps)
}

// CreateDepartment 创建部门
// @Summary 创建部门
// @Description 在当前租户下创建部门（name、code 必填）
// @Tags 组织管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "部门信息（name、code、description、managerId、parentId）"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/v1/org/departments [post]
func (h *Handler) CreateDepartment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"managerId"`
		ParentID    int    `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	d := &Department{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		ParentID:    req.ParentID,
		TenantID:    tenantID,
	}
	result, err := h.svc.CreateDepartment(c.Request.Context(), d)
	if err != nil {
		common.RespondError(c, err, "创建部门失败")
		return
	}
	common.Success(c, result)
}

// UpdateDepartment 更新部门
// @Summary 更新部门
// @Description 更新当前租户下的部门；只覆盖请求中出现的非空字段
// @Tags 组织管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门 ID"
// @Param request body object true "部门信息（name、code、description、managerId、parentId）"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/org/departments/{id} [put]
func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "invalid department id")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
		ManagerID   int    `json:"managerId"`
		ParentID    int    `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	// 先读取现有部门，避免部分更新时把 name/code 覆盖为空
	existing, err := h.svc.GetDepartment(c.Request.Context(), id, tenantID)
	if err != nil || existing == nil {
		common.NotFound(c, "部门不存在")
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Code != "" {
		existing.Code = req.Code
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.ManagerID != 0 {
		existing.ManagerID = req.ManagerID
	}
	if req.ParentID != 0 {
		existing.ParentID = req.ParentID
	}
	existing.TenantID = tenantID
	result, err := h.svc.UpdateDepartment(c.Request.Context(), existing)
	if err != nil {
		common.RespondError(c, err, "更新部门失败")
		return
	}
	common.Success(c, result)
}

// DeleteDepartment 删除部门
// @Summary 删除部门
// @Description 删除当前租户下的部门
// @Tags 组织管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "部门 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/v1/org/departments/{id} [delete]
func (h *Handler) DeleteDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "invalid department id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.svc.DeleteDepartment(c.Request.Context(), id, tenantID); err != nil {
		common.RespondError(c, err, "删除部门失败")
		return
	}
	common.Success(c, nil)
}

// Teams

func (h *Handler) ListTeams(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	teams, err := h.svc.ListTeams(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取团队列表失败")
		return
	}
	common.Success(c, teams)
}

func (h *Handler) GetTeam(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的团队ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	t, err := h.svc.GetTeam(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "获取团队失败")
		return
	}
	common.Success(c, t)
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"managerId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	t := &Team{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		TenantID:    tenantID,
	}
	result, err := h.svc.CreateTeam(c.Request.Context(), t)
	if err != nil {
		common.RespondError(c, err, "创建团队失败")
		return
	}
	common.Success(c, result)
}

func (h *Handler) UpdateTeam(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "invalid team id")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
		ManagerID   int    `json:"managerId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	existing, err := h.svc.GetTeam(c.Request.Context(), id, tenantID)
	if err != nil || existing == nil {
		common.NotFound(c, "团队不存在")
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Code != "" {
		existing.Code = req.Code
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.ManagerID != 0 {
		existing.ManagerID = req.ManagerID
	}
	existing.TenantID = tenantID
	result, err := h.svc.UpdateTeam(c.Request.Context(), existing)
	if err != nil {
		common.RespondError(c, err, "更新团队失败")
		return
	}
	common.Success(c, result)
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "invalid team id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.svc.DeleteTeam(c.Request.Context(), id, tenantID); err != nil {
		common.RespondError(c, err, "删除团队失败")
		return
	}
	common.Success(c, nil)
}

// Tags

func (h *Handler) ListTags(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tags, err := h.svc.ListTags(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取标签列表失败")
		return
	}
	common.Success(c, tags)
}

// Audit Logs

func (h *Handler) GetAuditLogs(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID, _ := strconv.Atoi(c.Query("userId"))
	logs, err := h.svc.GetAuditLogs(c.Request.Context(), tenantID, userID)
	if err != nil {
		common.RespondError(c, err, "获取审计日志失败")
		return
	}
	common.Success(c, logs)
}
