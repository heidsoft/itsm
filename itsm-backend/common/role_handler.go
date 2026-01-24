package common

import (
	"net/http"
	"sync"
	"time"

	"itsm-backend/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RoleHandler handles role-related requests
type RoleHandler struct {
	logger *zap.SugaredLogger
	mu     sync.RWMutex
	roles  map[int]*dto.RoleDTO
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(client interface{}, logger *zap.SugaredLogger) *RoleHandler {
	h := &RoleHandler{
		logger: logger,
		roles:  make(map[int]*dto.RoleDTO),
	}
	// Initialize with default roles
	h.initDefaultRoles()
	return h
}

func (h *RoleHandler) initDefaultRoles() {
	h.mu.Lock()
	defer h.mu.Unlock()

	defaultRoles := []dto.RoleDTO{
		{
			ID:          1,
			Name:        "系统管理员",
			Description: "拥有所有权限",
			Permissions: []string{"*"},
			Status:      "active",
			IsSystem:    true,
		},
		{
			ID:          2,
			Name:        "IT支持",
			Description: "IT支持团队",
			Permissions: []string{"tickets:*", "incidents:*", "knowledge:view"},
			Status:      "active",
			IsSystem:    false,
		},
		{
			ID:          3,
			Name:        "普通用户",
			Description: "普通用户角色",
			Permissions: []string{"tickets:create", "tickets:view", "knowledge:view"},
			Status:      "active",
			IsSystem:    false,
		},
	}

	for i := range defaultRoles {
		defaultRoles[i].CreatedAt = time.Now().Format(time.RFC3339)
		defaultRoles[i].UpdatedAt = time.Now().Format(time.RFC3339)
		h.roles[defaultRoles[i].ID] = &defaultRoles[i]
	}
}

// ListRoles handles GET /api/v1/roles
func (h *RoleHandler) ListRoles(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	roles := make([]*dto.RoleDTO, 0, len(h.roles))
	for _, r := range h.roles {
		roles = append(roles, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"roles":     roles,
			"total":     len(roles),
			"page":      1,
			"page_size": 20,
		},
	})
}

// GetRole handles GET /api/v1/roles/:id
func (h *RoleHandler) GetRole(c *gin.Context) {
	id := c.GetInt("role_id")

	h.mu.RLock()
	defer h.mu.RUnlock()

	role, exists := h.roles[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "角色不存在",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    role,
	})
}

// CreateRole handles POST /api/v1/roles
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
			"data":    nil,
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate new ID
	newID := len(h.roles) + 1
	for h.roles[newID] != nil {
		newID++
	}

	newRole := &dto.RoleDTO{
		ID:          newID,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		Status:      req.Status,
		IsSystem:    false,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	h.roles[newID] = newRole

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    newRole,
	})
}

// UpdateRole handles PUT /api/v1/roles/:id
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id := c.GetInt("role_id")

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据",
			"data":    nil,
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	existingRole, exists := h.roles[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "角色不存在",
			"data":    nil,
		})
		return
	}

	if existingRole.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "系统角色不能修改",
			"data":    nil,
		})
		return
	}

	if req.Name != "" {
		existingRole.Name = req.Name
	}
	if req.Description != "" {
		existingRole.Description = req.Description
	}
	if req.Permissions != nil {
		existingRole.Permissions = req.Permissions
	}
	if req.Status != "" {
		existingRole.Status = req.Status
	}

	existingRole.UpdatedAt = time.Now().Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    existingRole,
	})
}

// DeleteRole handles DELETE /api/v1/roles/:id
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id := c.GetInt("role_id")

	h.mu.Lock()
	defer h.mu.Unlock()

	existingRole, exists := h.roles[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "角色不存在",
			"data":    nil,
		})
		return
	}

	if existingRole.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "系统角色不能删除",
			"data":    nil,
		})
		return
	}

	delete(h.roles, id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"deleted": true},
	})
}

// ListPermissions handles GET /api/v1/permissions
func (h *RoleHandler) ListPermissions(c *gin.Context) {
	permissions := []string{
		// Dashboard
		"dashboard:view",
		// Tickets
		"tickets:view", "tickets:create", "tickets:update", "tickets:delete",
		// Incidents
		"incidents:view", "incidents:create", "incidents:update", "incidents:delete",
		// Problems
		"problems:view", "problems:create", "problems:update", "problems:delete",
		// Changes
		"changes:view", "changes:create", "changes:update", "changes:delete",
		// Service Catalog
		"service_catalog:view", "service_catalog:create", "service_catalog:update", "service_catalog:delete",
		// Knowledge Base
		"knowledge:view", "knowledge:create", "knowledge:update", "knowledge:delete",
		// Reports
		"reports:view", "reports:export", "reports:create",
		// CMDB
		"cmdb:view", "cmdb:create", "cmdb:update", "cmdb:delete",
		// SLA
		"sla:view", "sla:create", "sla:update", "sla:delete",
		// Users
		"users:view", "users:create", "users:update", "users:delete",
		// Roles
		"roles:view", "roles:create", "roles:update", "roles:delete",
		// Admin
		"admin:view", "admin:settings",
		// Workflows
		"workflows:view", "workflows:create", "workflows:update", "workflows:delete",
		// System Config
		"system_config:view", "system_config:update",
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"permissions": permissions,
		},
	})
}

// Helper to convert string to int for ID
func getIDParam(c *gin.Context, key string) int {
	idStr := c.Param(key)
	var id int
	for _, r := range idStr {
		if r >= '0' && r <= '9' {
			id = id*10 + int(r-'0')
		}
	}
	return id
}
