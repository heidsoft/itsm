package common

import (
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
		TenantID   int    `json:"tenantId"`
		TenantCode string `json:"tenantCode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.TenantID, req.TenantCode)
	if err != nil {
		common.AuthFailed(c, err.Error())
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
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	res, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		common.AuthFailed(c, err.Error())
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

// Logout clears httpOnly auth cookies and returns success.
func (h *Handler) Logout(c *gin.Context) {
	secure := shouldUseSecureCookies(c)
	domain := cookieDomain(c)

	c.SetCookie("access_token", "", -1, "/", domain, secure, true)
	c.SetCookie("refresh_token", "", -1, "/", domain, secure, true)

	common.Success(c, nil)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	u, err := h.svc.GetUser(c.Request.Context(), userID)
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
		common.InternalError(c, "获取用户租户列表失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{"tenants": tenants})
}

func (h *Handler) ListUsers(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	users, err := h.svc.ListUsers(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取用户列表失败: "+err.Error())
		return
	}
	common.Success(c, users)
}

// Departments

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
		common.InternalError(c, "获取部门失败: "+err.Error())
		return
	}
	common.Success(c, dept)
}

func (h *Handler) GetDepartmentTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tree, err := h.svc.GetDepartmentTree(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取部门树失败: "+err.Error())
		return
	}
	common.Success(c, tree)
}

func (h *Handler) ListDepartments(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	deps, err := h.svc.ListDepartments(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取部门列表失败: "+err.Error())
		return
	}
	common.Success(c, deps)
}

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"managerId"`
		ParentID    int    `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
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
		common.InternalError(c, "创建部门失败: "+err.Error())
		return
	}
	common.Success(c, result)
}

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
		common.ParamError(c, "参数错误: "+err.Error())
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
		common.InternalError(c, "更新部门失败: "+err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "invalid department id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.svc.DeleteDepartment(c.Request.Context(), id, tenantID); err != nil {
		common.InternalError(c, "删除部门失败: "+err.Error())
		return
	}
	common.Success(c, nil)
}

// Teams

func (h *Handler) ListTeams(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	teams, err := h.svc.ListTeams(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取团队列表失败: "+err.Error())
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
		common.InternalError(c, "获取团队失败: "+err.Error())
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
		common.ParamError(c, "参数错误: "+err.Error())
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
		common.InternalError(c, "创建团队失败: "+err.Error())
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
		common.ParamError(c, "参数错误: "+err.Error())
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
		common.InternalError(c, "更新团队失败: "+err.Error())
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
		common.InternalError(c, "删除团队失败: "+err.Error())
		return
	}
	common.Success(c, nil)
}

// Tags

func (h *Handler) ListTags(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	tags, err := h.svc.ListTags(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取标签列表失败: "+err.Error())
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
		common.InternalError(c, "获取审计日志失败: "+err.Error())
		return
	}
	common.Success(c, logs)
}
