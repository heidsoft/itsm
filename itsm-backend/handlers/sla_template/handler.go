// Package sla_template 是 SLA 模板域的 HTTP handler 层（域切片架构）。
// 自 controller/sla_template_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.SLATemplateService 承载，本包只做参数解析与响应封装。
package sla_template

import (
	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler SLA 模板 HTTP handler
type Handler struct {
	templateService *service.SLATemplateService
}

// NewHandler 创建 SLA 模板 handler 实例
func NewHandler(templateService *service.SLATemplateService) *Handler {
	return &Handler{templateService: templateService}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	tid, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "未授权访问")
		return 0, false
	}
	id, ok := tid.(int)
	if !ok {
		common.Fail(c, common.InternalErrorCode, "租户ID类型错误")
		return 0, false
	}
	return id, true
}

// ListTemplates 列出所有预置 SLA 模板
func (h *Handler) ListTemplates(ctx *gin.Context) {
	templates := h.templateService.ListTemplates()
	common.Success(ctx, gin.H{
		"items": templates,
		"total": len(templates),
	})
}

// GetTemplate 获取单个 SLA 模板详情
func (h *Handler) GetTemplate(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		common.Fail(ctx, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tmpl, err := h.templateService.GetTemplate(key)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, err.Error())
		return
	}

	common.Success(ctx, tmpl)
}

// InstallTemplate 将模板安装到当前租户
func (h *Handler) InstallTemplate(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		common.Fail(ctx, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tid, ok := tenantID(ctx)
	if !ok {
		return
	}

	result, err := h.templateService.InstallTemplate(ctx.Request.Context(), key, tid)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, err.Error())
		return
	}

	common.Success(ctx, result)
}

// RegisterRoutes 注册路由（兼容旧接口）
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	templates := r.Group("/sla/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.GET("/:key", h.GetTemplate)
		templates.POST("/:key/install", h.InstallTemplate)
	}
}
