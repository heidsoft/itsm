package auth

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SSO HTTP 层。
//
// 路由分三类，安全边界不同：
//   - 公开（initiate/callback/providers）：建立会话前调用，不挂认证中间件；
//     callback 成功后由 setAuthCookies 写入 httpOnly 凭证，响应体绝不回显 token。
//   - 已认证（bind/unbind/identities）：操作「当前会话」与外部身份的关系，
//     user_id 一律取鉴权中间件写入的上下文，不信任请求体。
//   - 管理（providers CRUD）：租户内配置面，RequirePermission("sso", ...)。

// SSOInitiate 处理 POST /api/v1/auth/sso/initiate。
// 若请求携带有效会话凭证，自动转入绑定模式（state 记录当前用户），
// 回调时校验会话未变，避免「点绑定结果变成重新登录」。
// @Summary 发起 SSO 登录
// @Description 生成授权请求并返回跳转 URL/state；provider 缺省 default。携带有效会话凭证时自动转入绑定模式，回调校验会话未变，避免「点绑定变成重新登录」。公开端点。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param body body dto.SSOInitiateRequest false "SSO 发起请求（可为空）"
// @Success 200 {object} common.Response
// @Failure 400 "提供方未启用"
// @Failure 500 "后端故障"
// @Router /api/v1/auth/sso/initiate [post]
func (h *Handler) SSOInitiate(c *gin.Context) {
	var req dto.SSOInitiateRequest
	// body 允许为空：provider 缺省 "default"。
	_ = c.ShouldBindJSON(&req)

	bindUserID := 0
	if token := requestRawAccessToken(c); token != "" {
		if claims, err := middleware.ValidateAccessToken(token, h.sso.jwtSecret); err == nil && claims.UserID > 0 {
			bindUserID = claims.UserID
		}
	}

	response, err := h.sso.Initiate(c.Request.Context(), req.Provider, req.TenantCode, bindUserID)
	if err != nil {
		common.RespondError(c, err, "SSO 登录发起失败")
		return
	}
	common.Success(c, response)
}

// SSOCallback 处理 POST /api/v1/auth/sso/callback?provider=xxx。
// 成功时建立会话（cookie），载荷与账号密码登录同构。
// @Summary SSO 回调
// @Description 用授权码与 state 完成登录，建立会话（cookie）并记录登录审计；失败审计为 LOGIN_FAILED。公开端点。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param provider query string false "提供方编码"
// @Param body body dto.SSOCallbackRequest true "回调参数(code/state)"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "回调校验失败"
// @Router /api/v1/auth/sso/callback [post]
func (h *Handler) SSOCallback(c *gin.Context) {
	providerCode := strings.TrimSpace(c.Query("provider"))
	var req dto.SSOCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	response, err := h.sso.Callback(auditCtx, providerCode, req.Code, req.State)
	if err != nil {
		h.sso.logger.Warnw("sso callback failed", "provider", providerCode, "error", err)
		middleware.RecordLoginAudit(auditCtx, h.sso.client, 0, 0, providerCode, "LOGIN_FAILED", "SSO 回调失败")
		common.AuthFailedWithErr(c, err, "SSO 回调失败")
		return
	}

	setAuthCookies(c, response.AccessToken, response.RefreshToken)
	middleware.RecordLoginAudit(auditCtx, h.sso.client, response.User.ID, response.User.TenantID, response.User.Username, "LOGIN_SUCCESS", "")
	common.Success(c, response)
}

// SSOListEnabledProviders 处理 GET /api/v1/auth/sso/providers（登录页数据源）。
// @Summary 列出启用的 SSO 提供方
// @Description 登录页数据源：按 tenantCode 列出已启用提供方，不暴露密钥等配置细节。公开端点。
// @Tags 单点登录
// @Produce json
// @Param tenantCode query string false "租户编码"
// @Success 200 {object} common.Response "提供方数组"
// @Failure 500 "查询失败"
// @Router /api/v1/auth/sso/providers [get]
func (h *Handler) SSOListEnabledProviders(c *gin.Context) {
	providers, err := h.sso.ListEnabledProviders(c.Request.Context(), c.Query("tenantCode"))
	if err != nil {
		common.InternalError(c, "查询 SSO 提供方失败")
		return
	}
	common.Success(c, providers)
}

// SSOBind 处理 POST /api/v1/auth/sso/bind（已认证）。
// @Summary 绑定 SSO 身份
// @Description 为当前已认证用户绑定外部 SSO 身份；user_id 取鉴权上下文，不信任请求体。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param body body dto.SSOBindRequest true "绑定请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "未认证"
// @Failure 409 "已被绑定"
// @Security BearerAuth
// @Router /api/v1/sso/bind [post]
func (h *Handler) SSOBind(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "用户未认证")
		return
	}
	var req dto.SSOBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	if err := h.sso.Bind(c.Request.Context(), userID, req.Provider, req.Code, req.State); err != nil {
		common.AuthFailedWithErr(c, err, "绑定失败")
		return
	}
	common.SuccessWithMessage(c, "绑定成功", nil)
}

// SSOUnbind 处理 POST /api/v1/auth/sso/unbind（已认证）。
// @Summary 解绑 SSO 身份
// @Description 解除当前用户的外部 SSO 身份绑定。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param body body dto.SSOUnbindRequest true "解绑请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "未认证"
// @Security BearerAuth
// @Router /api/v1/sso/unbind [post]
func (h *Handler) SSOUnbind(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "用户未认证")
		return
	}
	var req dto.SSOUnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	if err := h.sso.Unbind(c.Request.Context(), userID, req.Provider); err != nil {
		common.AuthFailedWithErr(c, err, "解除绑定失败")
		return
	}
	common.SuccessWithMessage(c, "已解除绑定", nil)
}

// SSOListIdentities 处理 GET /api/v1/auth/sso/identities（已认证）。
// @Summary 列出我的 SSO 身份
// @Description 列出当前用户已绑定的外部身份（不包含凭据）。
// @Tags 单点登录
// @Produce json
// @Success 200 {object} common.Response "身份数组"
// @Failure 401 "未认证"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/sso/identities [get]
func (h *Handler) SSOListIdentities(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "用户未认证")
		return
	}
	identities, err := h.sso.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		common.InternalError(c, "查询身份绑定失败")
		return
	}
	common.Success(c, identities)
}

// SSOAdminListProviders 处理 GET /api/v1/admin/sso/providers。
// @Summary 列出 SSO 提供方配置
// @Description 租户内 SSO 配置管理面，需要 sso:read 权限。
// @Tags 单点登录
// @Produce json
// @Success 200 {object} common.Response "提供方配置数组"
// @Failure 401 "未认证"
// @Failure 403 "权限不足"
// @Security BearerAuth
// @Router /api/v1/sso/providers [get]
func (h *Handler) SSOAdminListProviders(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	pagination := common.GetPaginationFromQuery(c)
	items, total, err := h.sso.ListProviders(c.Request.Context(), tenantID, pagination.Page, pagination.PageSize)
	if err != nil {
		common.InternalError(c, "查询提供方失败")
		return
	}
	common.SuccessWithList(c, items, total, pagination.Page, pagination.PageSize)
}

// SSOAdminGetProvider 处理 GET /api/v1/admin/sso/providers/:id。
// @Summary 获取 SSO 提供方配置
// @Description 按 ID 查询单个提供方配置，需要 sso:read 权限。
// @Tags 单点登录
// @Produce json
// @Param id path int true "提供方ID"
// @Success 200 {object} common.Response{data=dto.SSOProviderResponse}
// @Failure 400 "无效ID"
// @Failure 403 "权限不足"
// @Failure 404 "提供方不存在"
// @Security BearerAuth
// @Router /api/v1/sso/providers/{id} [get]
func (h *Handler) SSOAdminGetProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "参数错误: id 非法")
		return
	}
	provider, err := h.sso.GetProvider(c.Request.Context(), tenantID, providerID)
	if err != nil {
		common.NotFoundWithErr(c, err, "提供方不存在")
		return
	}
	common.Success(c, provider)
}

// SSOAdminCreateProvider 处理 POST /api/v1/admin/sso/providers。
// @Summary 创建 SSO 提供方
// @Description 新建租户级 SSO 提供方配置，需要 sso:write 权限。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param body body dto.SSOCreateProviderRequest true "提供方配置"
// @Success 200 {object} common.Response{data=dto.SSOProviderResponse}
// @Failure 400 "参数错误"
// @Failure 403 "权限不足"
// @Security BearerAuth
// @Router /api/v1/sso/providers [post]
func (h *Handler) SSOAdminCreateProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	var req dto.SSOCreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	provider, err := h.sso.CreateProvider(c.Request.Context(), tenantID, &req)
	if err != nil {
		common.RespondError(c, err, "创建提供方失败")
		return
	}
	common.SuccessWithMessage(c, "创建成功", provider)
}

// SSOAdminUpdateProvider 处理 PUT /api/v1/admin/sso/providers/:id。
// @Summary 更新 SSO 提供方
// @Description 局部更新提供方配置，需要 sso:write 权限。
// @Tags 单点登录
// @Accept json
// @Produce json
// @Param id path int true "提供方ID"
// @Param body body dto.SSOUpdateProviderRequest true "提供方配置"
// @Success 200 {object} common.Response{data=dto.SSOProviderResponse}
// @Failure 400 "参数错误"
// @Failure 403 "权限不足"
// @Failure 404 "提供方不存在"
// @Security BearerAuth
// @Router /api/v1/sso/providers/{id} [put]
func (h *Handler) SSOAdminUpdateProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "参数错误: id 非法")
		return
	}
	var req dto.SSOUpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	provider, err := h.sso.UpdateProvider(c.Request.Context(), tenantID, providerID, &req)
	if err != nil {
		common.RespondError(c, err, "更新提供方失败")
		return
	}
	common.SuccessWithMessage(c, "更新成功", provider)
}

// SSOAdminDeleteProvider 处理 DELETE /api/v1/admin/sso/providers/:id。
// @Summary 删除 SSO 提供方
// @Description 删除租户级 SSO 提供方配置，需要 sso:write 权限。
// @Tags 单点登录
// @Produce json
// @Param id path int true "提供方ID"
// @Success 200 {object} common.Response
// @Failure 400 "无效ID"
// @Failure 403 "权限不足"
// @Failure 404 "提供方不存在"
// @Security BearerAuth
// @Router /api/v1/sso/providers/{id} [delete]
func (h *Handler) SSOAdminDeleteProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "参数错误: id 非法")
		return
	}
	if err := h.sso.DeleteProvider(c.Request.Context(), tenantID, providerID); err != nil {
		common.RespondError(c, err, "删除提供方失败")
		return
	}
	common.SuccessWithMessage(c, "删除成功", nil)
}

// requestRawAccessToken 与 handlers/common 的 requestAccessToken 同口径：
// 优先上下文（鉴权中间件已解析），回退 cookie 与 Authorization header。
// initiate 在公开组上，拿不到中间件写入的上下文，只能走后两条路径。
func requestRawAccessToken(c *gin.Context) string {
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
