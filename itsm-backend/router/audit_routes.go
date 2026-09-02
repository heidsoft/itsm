package router

import (
	auditlogHandler "itsm-backend/handlers/auditlog"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAuditLogRoutes 设置审计日志相关路由
// 优先使用 AuditLogHandler（支持过滤/分页，返回 {logs,total,page,pageSize} 契约），
// 未装配时回退到 CommonHandler 的基础实现。
func SetupAuditLogRoutes(
	tenant *gin.RouterGroup,
	auditLogHandler *auditlogHandler.Handler,
	commonHandler *domainCommon.Handler,
) {
	// Audit Logs (short path for frontend compatibility)
	if auditLogHandler != nil {
		tenant.GET("/audit-logs", middleware.RequirePermission("audit", "read"), auditLogHandler.ListAuditLogs)
	} else if commonHandler != nil {
		tenant.GET("/audit-logs", middleware.RequirePermission("audit", "read"), commonHandler.GetAuditLogs)
	}
}
