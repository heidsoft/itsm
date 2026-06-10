package common

import (
	"net/http"
	"strconv"
	"strings"

	"itsm-backend/common"

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

// Auth

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		TenantID   int    `json:"tenant_id"`
		TenantCode string `json:"tenant_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.TenantID, req.TenantCode)
	if err != nil {
		common.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}

	secure := shouldUseSecureCookies(c)
	domain := cookieDomain(c)
	httpOnly := true

	// Browsers reject Secure cookies over plain HTTP. We keep host-only cookies
	// without Secure in local development so the same-origin frontend proxy can
	// persist login state. Production HTTPS requests still get Secure cookies.

	// Access token: 15分钟
	c.SetCookie("access_token", res.AccessToken, 900, "/", domain, secure, httpOnly)
	// Refresh token: 7天
	c.SetCookie("refresh_token", res.RefreshToken, 604800, "/", domain, secure, httpOnly)

	common.Success(c, res)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		common.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}

	secure := shouldUseSecureCookies(c)
	domain := cookieDomain(c)

	// 设置 httpOnly cookies (Secure only on HTTPS requests)
	c.SetCookie("access_token", res.AccessToken, 900, "/", domain, secure, true)
	if res.RefreshToken != "" {
		c.SetCookie("refresh_token", res.RefreshToken, 604800, "/", domain, secure, true)
	}

	common.Success(c, res)
}

// Users

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	u, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "User not found")
		return
	}
	common.Success(c, u)
}

// GetUserTenants 获取用户所属的租户列表（前端登录后需要）
func (h *Handler) GetUserTenants(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenants, err := h.svc.GetUserTenants(c.Request.Context(), userID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{"tenants": tenants})
}

func (h *Handler) ListUsers(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	users, err := h.svc.ListUsers(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, users)
}

// Departments

func (h *Handler) GetDepartmentTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tree, err := h.svc.GetDepartmentTree(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, tree)
}

func (h *Handler) ListDepartments(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	deps, err := h.svc.ListDepartments(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, deps)
}

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
		ParentID    int    `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
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
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid department id")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
		ParentID    int    `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	d := &Department{
		ID:          id,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		ParentID:    req.ParentID,
		TenantID:    tenantID,
	}
	result, err := h.svc.UpdateDepartment(c.Request.Context(), d)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid department id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.svc.DeleteDepartment(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, nil)
}

// Teams

func (h *Handler) ListTeams(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	teams, err := h.svc.ListTeams(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, teams)
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
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
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) UpdateTeam(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid team id")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	t := &Team{
		ID:          id,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		TenantID:    tenantID,
	}
	result, err := h.svc.UpdateTeam(c.Request.Context(), t)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid team id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.svc.DeleteTeam(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, nil)
}

// Tags

func (h *Handler) ListTags(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tags, err := h.svc.ListTags(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, tags)
}

// Audit Logs

func (h *Handler) GetAuditLogs(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	userID, _ := strconv.Atoi(c.Query("user_id"))
	logs, err := h.svc.GetAuditLogs(c.Request.Context(), tenantID, userID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, logs)
}
