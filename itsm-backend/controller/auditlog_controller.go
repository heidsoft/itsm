package controller

import (
    "itsm-backend/common"
    "itsm-backend/dto"
    "itsm-backend/middleware"
    "itsm-backend/service"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type AuditLogController struct {
    service *service.AuditLogService
    logger  *zap.SugaredLogger
}

func NewAuditLogController(s *service.AuditLogService, logger *zap.SugaredLogger) *AuditLogController {
    return &AuditLogController{service: s, logger: logger}
}

// ListAuditLogs godoc
// @Summary 列出审计日志
// @Description 根据过滤条件分页查询审计日志
// @Tags AuditLogs
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param user_id query int false "用户ID"
// @Param resource query string false "资源类型"
// @Param action query string false "动作"
// @Param method query string false "HTTP方法"
// @Param status_code query int false "状态码"
// @Param path query string false "路径前缀"
// @Param request_id query string false "请求ID"
// @Param from query string false "开始时间(RFC3339)"
// @Param to query string false "结束时间(RFC3339)"
// @Success 200 {object} common.Response
// @Router /api/v1/audit-logs [get]
func (c *AuditLogController) ListAuditLogs(ctx *gin.Context) {
    var req dto.ListAuditLogsRequest
    if err := ctx.ShouldBindQuery(&req); err != nil {
        common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
        return
    }

    tenantID, err := middleware.GetTenantID(ctx)
    if err != nil || tenantID <= 0 {
        common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
        return
    }

    resp, err := c.service.ListAuditLogs(ctx, &req, tenantID)
    if err != nil {
        c.logger.Errorw("Failed to list audit logs", "error", err, "tenant_id", tenantID)
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }
    common.Success(ctx, resp)
}