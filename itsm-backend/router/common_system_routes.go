package router

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupCommonSystemRoutes 注册 Common & System 域路由（auth/me、users、groups、org、
// projects、applications、system、RBAC roles/permissions/menus、tenants、notifications）。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// 需要 *gin.Engine 是因为 auth 子组挂在 /api/v1/auth（r 层）而非 tenant 层。
func SetupCommonSystemRoutes(r *gin.Engine, tenant *gin.RouterGroup, config *RouterConfig) {
	if config.CommonHandler != nil {
		// Auth scoped
		// 修复 P0：auth/me, auth/tenants, auth/menus 不应在 tenant 分组内
		// 否则 tenant RBAC 中间件会拦截端用户，导致 layout 崩溃
		authGrp := r.Group("/api/v1/auth")
		{
			authGrp.GET("/me", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe)
			authGrp.GET("/tenants", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetUserTenants)
			authGrp.POST("/logout", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.Logout)
			if config.AuthHandler != nil {
				authGrp.POST("/switch-tenant", middleware.AuthMiddleware(config.JWTSecret), config.AuthHandler.SwitchTenant)
			}
		}

		// User Menu (no permission required, will be filtered by role)
		if config.RBACHandler != nil {
			authGrp.GET("/menus", middleware.AuthMiddleware(config.JWTSecret), config.RBACHandler.GetUserMenus)
		}

		// ==================== Audit Logs ====================
		SetupAuditLogRoutes(tenant, config.AuditLogHandler, config.CommonHandler)

		// Users
		if config.UserHandler != nil {
			users := tenant.Group("/users")
			{
				users.GET("", middleware.RequirePermission("user", "read"), config.UserHandler.ListUsers)
				users.POST("", middleware.RequirePermission("user", "write"), config.UserHandler.CreateUser)
				users.GET("/profile", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe) // 获取当前用户信息（需认证）
				users.GET("/me", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe)      // alias of /profile
				users.GET("/:id", middleware.RequirePermission("user", "read"), config.UserHandler.GetUser)
				users.PUT("/:id", middleware.RequirePermission("user", "write"), config.UserHandler.UpdateUser)
				users.DELETE("/:id", middleware.RequirePermission("user", "delete"), config.UserHandler.DeleteUser)
				users.PUT("/:id/status", middleware.RequirePermission("user", "write"), config.UserHandler.ChangeUserStatus)
				users.PUT("/:id/reset-password", middleware.RequirePermission("user", "write"), config.UserHandler.ResetPassword)
				users.GET("/stats", middleware.RequirePermission("user", "read"), config.UserHandler.GetUserStats)
				users.POST("/batch", middleware.RequirePermission("user", "write"), config.UserHandler.BatchUpdateUsers)
			}
		} else {
			users := tenant.Group("/users")
			{
				users.GET("", middleware.RequirePermission("user", "read"), config.CommonHandler.ListUsers)
			}
		}

		// Groups
		if config.GroupHandler != nil {
			groups := tenant.Group("/groups")
			{
				groups.GET("", middleware.RequirePermission("groups", "read"), config.GroupHandler.ListGroups)
				groups.POST("", middleware.RequirePermission("groups", "write"), config.GroupHandler.CreateGroup)
				groups.GET("/:id", middleware.RequirePermission("groups", "read"), config.GroupHandler.GetGroup)
				groups.PUT("/:id", middleware.RequirePermission("groups", "write"), config.GroupHandler.UpdateGroup)
				groups.DELETE("/:id", middleware.RequirePermission("groups", "write"), config.GroupHandler.DeleteGroup)
				groups.POST("/:id/members", middleware.RequirePermission("groups", "write"), config.GroupHandler.AddUserToGroup)
				groups.DELETE("/:id/members", middleware.RequirePermission("groups", "write"), config.GroupHandler.RemoveUserFromGroup)
				groups.GET("/:id/members", middleware.RequirePermission("groups", "read"), config.GroupHandler.GetGroupMembers)
			}
		}

		// Organization
		org := tenant.Group("/org")
		{
			org.GET("/departments/tree", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartmentTree)
			org.GET("/departments/:id", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartment)
			org.POST("/departments", middleware.RequirePermission("department", "create"), config.CommonHandler.CreateDepartment)
			org.PUT("/departments/:id", middleware.RequirePermission("department", "update"), config.CommonHandler.UpdateDepartment)
			org.DELETE("/departments/:id", middleware.RequirePermission("department", "delete"), config.CommonHandler.DeleteDepartment)
			org.GET("/teams", middleware.RequirePermission("team", "read"), config.CommonHandler.ListTeams)
			org.GET("/teams/:id", middleware.RequirePermission("team", "read"), config.CommonHandler.GetTeam)
			org.POST("/teams", middleware.RequirePermission("team", "create"), config.CommonHandler.CreateTeam)
			org.PUT("/teams/:id", middleware.RequirePermission("team", "update"), config.CommonHandler.UpdateTeam)
			org.DELETE("/teams/:id", middleware.RequirePermission("team", "delete"), config.CommonHandler.DeleteTeam)
		}

		// Projects
		if config.ProjectHandler != nil {
			projects := tenant.Group("/projects")
			{
				projects.GET("", middleware.RequirePermission("project", "read"), config.ProjectHandler.ListProjects)
				projects.POST("", middleware.RequirePermission("project", "write"), config.ProjectHandler.CreateProject)
				projects.GET("/:id", middleware.RequirePermission("project", "read"), config.ProjectHandler.GetProject)
				projects.PUT("/:id", middleware.RequirePermission("project", "write"), config.ProjectHandler.UpdateProject)
				projects.DELETE("/:id", middleware.RequirePermission("project", "write"), config.ProjectHandler.DeleteProject)
			}
		}

		// Applications
		if config.ApplicationHandler != nil {
			applications := tenant.Group("/applications")
			{
				applications.GET("", middleware.RequirePermission("application", "read"), config.ApplicationHandler.ListApplications)
				applications.POST("", middleware.RequirePermission("application", "write"), config.ApplicationHandler.CreateApplication)
				applications.GET("/microservices", middleware.RequirePermission("application", "read"), config.ApplicationHandler.ListMicroservices)
				applications.POST("/microservices", middleware.RequirePermission("application", "write"), config.ApplicationHandler.CreateMicroservice)
			}
		}

		sys := tenant.Group("/system")
		{
			sys.GET("/tags", middleware.RequirePermission("tag", "read"), config.CommonHandler.ListTags)
			sys.GET("/audit-logs", middleware.RequirePermission("audit_log", "read"), config.CommonHandler.GetAuditLogs)
		}

		// B7/Bug10: 顶层路由别名（兼容前端 /api/v1/{departments,teams,tags,admin/tenants} 旧路径）
		// 实际业务仍在 /org/* 与 /system/* 与 /tenants，旧路径保持只读 GET，避免破坏现有调用方
		{
			tenant.GET("/departments", middleware.RequirePermission("department", "read"), config.CommonHandler.ListDepartments)
			tenant.GET("/departments/tree", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartmentTree)
			tenant.GET("/teams", middleware.RequirePermission("team", "read"), config.CommonHandler.ListTeams)
			tenant.GET("/tags", middleware.RequirePermission("tag", "read"), config.CommonHandler.ListTags)
		}
		if config.TenantHandler != nil {
			admin := tenant.Group("/admin")
			{
				admin.GET("/tenants", middleware.RequirePermission("tenant", "read"), config.TenantHandler.ListTenants)
			}
		}

		// Role & Permission Controllers (database-backed with tenant isolation)
		if config.RBACHandler != nil {
			roles := tenant.Group("/roles")
			{
				roles.GET("", middleware.RequirePermission("role", "read"), config.RBACHandler.ListRoles)
				roles.POST("", middleware.RequirePermission("role", "create"), config.RBACHandler.CreateRole)
				roles.GET("/:id", middleware.RequirePermission("role", "read"), config.RBACHandler.GetRole)
				roles.PUT("/:id", middleware.RequirePermission("role", "update"), config.RBACHandler.UpdateRole)
				roles.DELETE("/:id", middleware.RequirePermission("role", "delete"), config.RBACHandler.DeleteRole)
				roles.POST("/:id/permissions", middleware.RequirePermission("role", "update"), config.RBACHandler.AssignPermissions)
			}
		}

		if config.RBACHandler != nil {
			permissions := tenant.Group("/permissions")
			{
				permissions.GET("", middleware.RequirePermission("permission", "read"), config.RBACHandler.ListPermissions)
				permissions.POST("", middleware.RequirePermission("permission", "create"), config.RBACHandler.CreatePermission)
				permissions.POST("/init", middleware.RequirePermission("permission", "create"), config.RBACHandler.InitDefaultPermissions)
			}
		}

		// Menu Controllers (database-backed with tenant isolation)
		if config.RBACHandler != nil {
			menus := tenant.Group("/menus")
			{
				menus.GET("", middleware.RequirePermission("menu", "read"), config.RBACHandler.ListMenus)
				menus.POST("", middleware.RequirePermission("menu", "create"), config.RBACHandler.CreateMenu)
				menus.GET("/:id", middleware.RequirePermission("menu", "read"), config.RBACHandler.GetMenu)
				menus.PUT("/:id", middleware.RequirePermission("menu", "update"), config.RBACHandler.UpdateMenu)
				menus.DELETE("/:id", middleware.RequirePermission("menu", "delete"), config.RBACHandler.DeleteMenu)
				menus.POST("/init", middleware.RequirePermission("menu", "create"), config.RBACHandler.InitDefaultMenus)
			}
		}

		// Tenant Management (admin only)
		if config.TenantHandler != nil {
			tenants := tenant.Group("/tenants")
			{
				tenants.GET("", middleware.RequirePermission("tenant", "read"), config.TenantHandler.ListTenants)
				tenants.POST("", middleware.RequirePermission("tenant", "create"), config.TenantHandler.CreateTenant)
				tenants.GET("/:id", middleware.RequirePermission("tenant", "read"), config.TenantHandler.GetTenant)
				tenants.PUT("/:id", middleware.RequirePermission("tenant", "update"), config.TenantHandler.UpdateTenant)
				tenants.DELETE("/:id", middleware.RequirePermission("tenant", "delete"), config.TenantHandler.DeleteTenant)
				tenants.PUT("/:id/status", middleware.RequirePermission("tenant", "update"), config.TenantHandler.UpdateTenantStatus)
			}
		}

		// Notification Preferences
		config.Logger.Info("NotificationHandler check:", zap.Any("handler", config.NotificationHandler))
		if config.NotificationHandler != nil {
			config.Logger.Info("Registering notification-preferences routes")
			notifPrefs := tenant.Group("/notification-preferences")
			{
				notifPrefs.GET("", middleware.RequirePermission("notification", "read"), config.NotificationHandler.ListPreferences)
				notifPrefs.GET("/event-types", middleware.RequirePermission("notification", "read"), config.NotificationHandler.ListEventTypes)
				notifPrefs.GET("/:event_type", middleware.RequirePermission("notification", "read"), config.NotificationHandler.GetPreference)
				notifPrefs.POST("", middleware.RequirePermission("notification", "create"), config.NotificationHandler.CreateOrUpdatePreference)
				notifPrefs.PUT("", middleware.RequirePermission("notification", "update"), config.NotificationHandler.BulkUpdatePreferences)
				notifPrefs.DELETE("/:event_type", middleware.RequirePermission("notification", "delete"), config.NotificationHandler.DeletePreference)
				notifPrefs.POST("/reset", middleware.RequirePermission("notification", "update"), config.NotificationHandler.ResetPreferences)
				notifPrefs.POST("/init", middleware.RequirePermission("notification", "create"), config.NotificationHandler.InitializeDefaultPreferences)
			}
		}

		// Notifications
		if config.NotificationHandler != nil {
			notifications := tenant.Group("/notifications")
			{
				notifications.GET("", middleware.RequirePermission("notification", "read"), config.NotificationHandler.GetNotifications)
				notifications.GET("/unread-count", middleware.RequirePermission("notification", "read"), config.NotificationHandler.GetUnreadCount)
				notifications.PUT("/:id/read", middleware.RequirePermission("notification", "update"), config.NotificationHandler.MarkNotificationRead)
				notifications.PUT("/read-all", middleware.RequirePermission("notification", "update"), config.NotificationHandler.MarkAllNotificationsRead)
				notifications.PUT("/batch/read", middleware.RequirePermission("notification", "update"), config.NotificationHandler.MarkNotificationsRead)
				notifications.DELETE("/batch", middleware.RequirePermission("notification", "delete"), config.NotificationHandler.DeleteNotifications)
				notifications.DELETE("/:id", middleware.RequirePermission("notification", "delete"), config.NotificationHandler.DeleteNotification)
				notifications.POST("", middleware.RequirePermission("notification", "create"), config.NotificationHandler.CreateNotification)
			}
		}
	}
}
