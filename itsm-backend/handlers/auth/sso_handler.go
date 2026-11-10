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
		common.RespondError(c, err, "SSO login initiation failed")
		return
	}
	common.Success(c, response)
}

// SSOCallback 处理 POST /api/v1/auth/sso/callback?provider=xxx。
// 成功时建立会话（cookie），载荷与账号密码登录同构。
func (h *Handler) SSOCallback(c *gin.Context) {
	providerCode := strings.TrimSpace(c.Query("provider"))
	var req dto.SSOCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}

	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	response, err := h.sso.Callback(auditCtx, providerCode, req.Code, req.State)
	if err != nil {
		h.sso.logger.Warnw("sso callback failed", "provider", providerCode, "error", err)
		middleware.RecordLoginAudit(auditCtx, h.sso.client, 0, 0, providerCode, "LOGIN_FAILED", "SSO 回调失败")
		common.AuthFailedWithErr(c, err, "SSO callback failed")
		return
	}

	setAuthCookies(c, response.AccessToken, response.RefreshToken)
	middleware.RecordLoginAudit(auditCtx, h.sso.client, response.User.ID, response.User.TenantID, response.User.Username, "LOGIN_SUCCESS", "")
	common.Success(c, response)
}

// SSOListEnabledProviders 处理 GET /api/v1/auth/sso/providers（登录页数据源）。
func (h *Handler) SSOListEnabledProviders(c *gin.Context) {
	providers, err := h.sso.ListEnabledProviders(c.Request.Context(), c.Query("tenantCode"))
	if err != nil {
		common.InternalError(c, "Failed to query SSO provider")
		return
	}
	common.Success(c, providers)
}

// SSOBind 处理 POST /api/v1/auth/sso/bind（已认证）。
func (h *Handler) SSOBind(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "User not authenticated")
		return
	}
	var req dto.SSOBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}
	if err := h.sso.Bind(c.Request.Context(), userID, req.Provider, req.Code, req.State); err != nil {
		common.AuthFailedWithErr(c, err, "Binding failed")
		return
	}
	common.SuccessWithMessage(c, "Binding successful", nil)
}

// SSOUnbind 处理 POST /api/v1/auth/sso/unbind（已认证）。
func (h *Handler) SSOUnbind(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "User not authenticated")
		return
	}
	var req dto.SSOUnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}
	if err := h.sso.Unbind(c.Request.Context(), userID, req.Provider); err != nil {
		common.AuthFailedWithErr(c, err, "Unbinding failed")
		return
	}
	common.SuccessWithMessage(c, "Unbound", nil)
}

// SSOListIdentities 处理 GET /api/v1/auth/sso/identities（已认证）。
func (h *Handler) SSOListIdentities(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "User not authenticated")
		return
	}
	identities, err := h.sso.ListIdentities(c.Request.Context(), userID)
	if err != nil {
		common.InternalError(c, "Failed to query identity binding")
		return
	}
	common.Success(c, identities)
}

// SSOAdminListProviders 处理 GET /api/v1/admin/sso/providers。
func (h *Handler) SSOAdminListProviders(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	pagination := common.GetPaginationFromQuery(c)
	items, total, err := h.sso.ListProviders(c.Request.Context(), tenantID, pagination.Page, pagination.PageSize)
	if err != nil {
		common.InternalError(c, "Failed to query provider")
		return
	}
	common.SuccessWithList(c, items, total, pagination.Page, pagination.PageSize)
}

// SSOAdminGetProvider 处理 GET /api/v1/admin/sso/providers/:id。
func (h *Handler) SSOAdminGetProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "Parameter error: id is illegal")
		return
	}
	provider, err := h.sso.GetProvider(c.Request.Context(), tenantID, providerID)
	if err != nil {
		common.NotFoundWithErr(c, err, "Provider does not exist")
		return
	}
	common.Success(c, provider)
}

// SSOAdminCreateProvider 处理 POST /api/v1/admin/sso/providers。
func (h *Handler) SSOAdminCreateProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	var req dto.SSOCreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}
	provider, err := h.sso.CreateProvider(c.Request.Context(), tenantID, &req)
	if err != nil {
		common.RespondError(c, err, "Failed to create provider")
		return
	}
	common.SuccessWithMessage(c, "Created", provider)
}

// SSOAdminUpdateProvider 处理 PUT /api/v1/admin/sso/providers/:id。
func (h *Handler) SSOAdminUpdateProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "Parameter error: id is illegal")
		return
	}
	var req dto.SSOUpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}
	provider, err := h.sso.UpdateProvider(c.Request.Context(), tenantID, providerID, &req)
	if err != nil {
		common.RespondError(c, err, "Failed to update provider")
		return
	}
	common.SuccessWithMessage(c, "Updated", provider)
}

// SSOAdminDeleteProvider 处理 DELETE /api/v1/admin/sso/providers/:id。
func (h *Handler) SSOAdminDeleteProvider(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	providerID, err := strconv.Atoi(c.Param("id"))
	if err != nil || providerID <= 0 {
		common.ParamError(c, "Parameter error: id is illegal")
		return
	}
	if err := h.sso.DeleteProvider(c.Request.Context(), tenantID, providerID); err != nil {
		common.RespondError(c, err, "Failed to delete provider")
		return
	}
	common.SuccessWithMessage(c, "Deleted", nil)
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
