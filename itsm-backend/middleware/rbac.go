package middleware

import (
    "context"
    "itsm-backend/common"
    "itsm-backend/ent"
    "itsm-backend/ent/ticket"
    "itsm-backend/ent/user"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
)

// Permission 权限结构
type Permission struct {
	Resource string `json:"resource"` // 资源名称，如 "ticket", "user", "dashboard"
	Action   string `json:"action"`   // 操作类型，如 "read", "write", "delete", "admin"
}

// RolePermissions 角色权限映射
var RolePermissions = map[string][]Permission{
    "super_admin": {
        {Resource: "*", Action: "*"}, // 超级管理员拥有所有权限
    },
    "admin": {
        {Resource: "ticket", Action: "read"},
        {Resource: "ticket", Action: "write"},
        {Resource: "ticket", Action: "delete"},
        {Resource: "ticket", Action: "admin"},
        {Resource: "user", Action: "read"},
        {Resource: "user", Action: "write"},
        {Resource: "user", Action: "delete"},
        {Resource: "dashboard", Action: "read"},
        {Resource: "dashboard", Action: "admin"},
        {Resource: "knowledge", Action: "read"},
        {Resource: "knowledge", Action: "write"},
        {Resource: "knowledge", Action: "admin"},
        {Resource: "cmdb", Action: "read"},
        {Resource: "cmdb", Action: "write"},
        {Resource: "incident", Action: "read"},
        {Resource: "incident", Action: "write"},
        {Resource: "incident", Action: "admin"},
        // 审计日志权限：仅管理员及以上可读
        {Resource: "audit_logs", Action: "read"},
        {Resource: "ai", Action: "read"},
        {Resource: "ai", Action: "write"},
    },
    "manager": {
        {Resource: "ticket", Action: "read"},
        {Resource: "ticket", Action: "write"},
        {Resource: "incident", Action: "read"},
        {Resource: "incident", Action: "write"},
        {Resource: "dashboard", Action: "read"},
        {Resource: "knowledge", Action: "read"},
        {Resource: "cmdb", Action: "read"},
        {Resource: "user", Action: "read"}, // 经理可查看用户基本信息
    },
    "agent": {
        {Resource: "ticket", Action: "read"},
        {Resource: "ticket", Action: "write"},
        {Resource: "dashboard", Action: "read"},
        {Resource: "knowledge", Action: "read"},
        {Resource: "knowledge", Action: "write"},
        {Resource: "cmdb", Action: "read"},
        {Resource: "incident", Action: "read"},
        {Resource: "incident", Action: "write"},
    },
    "technician": {
        {Resource: "ticket", Action: "read"},
        {Resource: "ticket", Action: "write"},
        {Resource: "knowledge", Action: "read"},
        {Resource: "cmdb", Action: "read"},
        {Resource: "incident", Action: "read"},
        {Resource: "incident", Action: "write"},
    },
    "end_user": {
        {Resource: "ticket", Action: "read"},
        {Resource: "ticket", Action: "write"}, // 最终用户可以创建和更新自己的工单
        {Resource: "knowledge", Action: "read"},
        {Resource: "dashboard", Action: "read"},
        {Resource: "ai", Action: "read"},
        {Resource: "ai", Action: "write"},
    },
}

// ResourceActionMap 路径到资源和操作的映射
var ResourceActionMap = map[string]map[string]Permission{
    "GET": {
		"/api/v1/tickets":           {Resource: "ticket", Action: "read"},
		"/api/v1/tickets/*":         {Resource: "ticket", Action: "read"},
		"/api/v1/users":             {Resource: "user", Action: "read"},
		"/api/v1/users/*":           {Resource: "user", Action: "read"},
		"/api/v1/dashboard":         {Resource: "dashboard", Action: "read"},
		"/api/v1/dashboard/*":       {Resource: "dashboard", Action: "read"},
		"/api/v1/knowledge":         {Resource: "knowledge", Action: "read"},
		"/api/v1/knowledge/*":       {Resource: "knowledge", Action: "read"},
		"/api/v1/cmdb":              {Resource: "cmdb", Action: "read"},
		"/api/v1/cmdb/*":            {Resource: "cmdb", Action: "read"},
		"/api/v1/incidents":         {Resource: "incident", Action: "read"},
		"/api/v1/incidents/*":       {Resource: "incident", Action: "read"},
		"/api/v1/audit-logs":        {Resource: "audit_logs", Action: "read"},
		"/api/v1/ai/*":              {Resource: "ai", Action: "read"},
	},
	"POST": {
		"/api/v1/tickets":           {Resource: "ticket", Action: "write"},
		"/api/v1/tickets/*":         {Resource: "ticket", Action: "write"},
		"/api/v1/users":             {Resource: "user", Action: "write"},
		"/api/v1/knowledge":         {Resource: "knowledge", Action: "write"},
		"/api/v1/cmdb":              {Resource: "cmdb", Action: "write"},
		"/api/v1/incidents":         {Resource: "incident", Action: "write"},
		"/api/v1/ai/*":              {Resource: "ai", Action: "write"},
	},
	"PUT": {
		"/api/v1/tickets/*":         {Resource: "ticket", Action: "write"},
		"/api/v1/users/*":           {Resource: "user", Action: "write"},
		"/api/v1/knowledge/*":       {Resource: "knowledge", Action: "write"},
		"/api/v1/cmdb/*":            {Resource: "cmdb", Action: "write"},
		"/api/v1/incidents/*":       {Resource: "incident", Action: "write"},
	},
	"DELETE": {
		"/api/v1/tickets/*":         {Resource: "ticket", Action: "delete"},
		"/api/v1/users/*":           {Resource: "user", Action: "delete"},
		"/api/v1/knowledge/*":       {Resource: "knowledge", Action: "delete"},
		"/api/v1/cmdb/*":            {Resource: "cmdb", Action: "delete"},
		"/api/v1/incidents/*":       {Resource: "incident", Action: "delete"},
	},
}

// RBACMiddleware RBAC权限控制中间件
func RBACMiddleware(client *ent.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 将 Ent 客户端放入上下文，供资源级别检查使用（例如工单所有权校验）
        if client != nil {
            c.Set("client", client)
        }
        // 获取用户信息
        userIDInterface, exists := c.Get("user_id")
        if !exists {
            common.Fail(c, common.AuthFailedCode, "用户未认证")
            c.Abort()
			return
		}

		userID, ok := userIDInterface.(int)
		if !ok {
			common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
			c.Abort()
			return
		}

		// 从数据库获取用户最新角色信息
		userEntity, err := client.User.Query().
			Where(user.ID(userID)).
			Only(context.Background())
		if err != nil {
			common.Fail(c, common.AuthFailedCode, "用户不存在")
			c.Abort()
			return
		}

		// 检查用户是否被禁用
		if !userEntity.Active {
			common.Fail(c, common.ForbiddenCode, "用户已被禁用")
			c.Abort()
			return
		}

		// 从JWT中获取角色信息
		roleInterface, exists := c.Get("role")
		if !exists {
			common.Fail(c, common.AuthFailedCode, "角色信息缺失")
			c.Abort()
			return
		}

		role, ok := roleInterface.(string)
		if !ok {
			common.Fail(c, common.AuthFailedCode, "角色格式错误")
			c.Abort()
			return
		}

		// 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 检查权限
		if !hasPermission(role, method, path, userID, c) {
			common.Fail(c, common.ForbiddenCode, "权限不足")
			c.Abort()
			return
		}

		// 将用户实体信息存储到上下文中
		c.Set("user_entity", userEntity)

		c.Next()
	}
}

// RequirePermission 要求特定权限的中间件
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			common.Fail(c, common.AuthFailedCode, "用户角色信息缺失")
			c.Abort()
			return
		}

		if !hasResourcePermission(role.(string), resource, action) {
			common.Fail(c, common.ForbiddenCode, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole enforces that the authenticated user role is one of the allowed roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	normalized := make([]string, 0, len(allowedRoles))
	for _, r := range allowedRoles {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(r)))
	}
	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 2003, "message": "缺少角色信息"})
			c.Abort()
			return
		}
		role, _ := roleAny.(string)
		role = strings.ToLower(strings.TrimSpace(role))
		for _, ar := range normalized {
			if role == ar {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"code": 2003, "message": "无权限执行该操作"})
		c.Abort()
	}
}

// hasPermission 检查用户是否有权限访问指定资源
func hasPermission(role, method, path string, userID int, c *gin.Context) bool {
	// 超级管理员拥有所有权限
	if role == "super_admin" {
		return true
	}

	// 获取路径对应的资源和操作
	permission := getPermissionFromPath(method, path)
	if permission == nil {
		// 如果没有找到对应的权限配置，默认拒绝访问
		return false
	}

	// 检查角色是否有该资源的操作权限
	if !hasResourcePermission(role, permission.Resource, permission.Action) {
		return false
	}

	// 资源级别的权限检查
	return checkResourceLevelPermission(role, permission.Resource, permission.Action, userID, c)
}

// hasResourcePermission 检查角色是否有指定资源的操作权限
func hasResourcePermission(role, resource, action string) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, perm := range permissions {
		// 检查通配符权限
		if perm.Resource == "*" && perm.Action == "*" {
			return true
		}
		if perm.Resource == "*" && perm.Action == action {
			return true
		}
		if perm.Resource == resource && perm.Action == "*" {
			return true
		}
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}

	return false
}

// getPermissionFromPath 从路径获取权限信息
func getPermissionFromPath(method, path string) *Permission {
	methodMap, exists := ResourceActionMap[method]
	if !exists {
		return nil
	}

	// 精确匹配
	if perm, exists := methodMap[path]; exists {
		return &perm
	}

	// 通配符匹配
	for pattern, perm := range methodMap {
		if matchPath(pattern, path) {
			return &perm
		}
	}

	return nil
}

// checkResourceLevelPermission 检查资源级别的权限
func checkResourceLevelPermission(role, resource, action string, userID int, c *gin.Context) bool {
	// 对于某些资源，需要检查用户是否只能访问自己的数据
	switch resource {
	case "ticket":
		return checkTicketPermission(role, action, userID, c)
	case "user":
		return checkUserPermission(role, action, userID, c)
	default:
		return true
	}
}

// checkTicketPermission 检查工单权限
func checkTicketPermission(role, action string, userID int, c *gin.Context) bool {
    // 具备全局工单访问权限的角色
    if role == "admin" || role == "agent" || role == "manager" || role == "technician" {
        return true
    }

    // 最终用户（end_user）只能访问自己创建的工单
    if role == "end_user" {
        // 创建自己的工单：允许 POST /api/v1/tickets
        if c.Request.Method == "POST" && strings.HasPrefix(c.Request.URL.Path, "/api/v1/tickets") {
            return true
        }
        // 其他操作需要检查工单的请求人是否为当前用户
        ticketIDStr := c.Param("id")
        if ticketIDStr != "" {
            ticketID, err := strconv.Atoi(ticketIDStr)
            if err != nil {
                return false
            }
            return checkTicketOwnership(ticketID, userID, c)
        }
        // 当未提供具体工单ID时，默认拒绝（例如尝试访问他人列表）
        return false
    }

    // 其他角色默认允许（已通过资源操作权限检查）
    return true
}

// checkUserPermission 检查用户权限
func checkUserPermission(role, action string, userID int, c *gin.Context) bool {
    // 管理员可以访问所有用户
    if role == "admin" {
        return true
    }
    // 经理可读取所有用户基本信息（仅限 read 操作）
    if role == "manager" && action == "read" {
        return true
    }

    // 其他角色只能访问自己的用户信息
    targetUserIDStr := c.Param("id")
    if targetUserIDStr != "" {
        targetUserID, err := strconv.Atoi(targetUserIDStr)
        if err != nil {
            return false
        }
        return targetUserID == userID
    }

    // 对 /api/v1/users 列表等未指定ID的访问，非管理员/经理默认拒绝
    return false
}

// checkTicketOwnership 检查工单所有权
func checkTicketOwnership(ticketID, userID int, c *gin.Context) bool {
    // 从上下文中获取Ent客户端
    clientAny, exists := c.Get("client")
    if !exists {
        return false
    }
    client, ok := clientAny.(*ent.Client)
    if !ok || client == nil {
        return false
    }
    // 查询工单并校验请求人
    t, err := client.Ticket.Query().
        Where(ticket.ID(ticketID)).
        Only(context.Background())
    if err != nil {
        return false
    }
    return t.RequesterID == userID
}

// matchPath 匹配路径（支持通配符）
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// 支持 * 通配符
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return false
}
