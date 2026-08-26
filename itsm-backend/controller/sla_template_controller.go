package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// SLATemplateController SLA模板控制器
type SLATemplateController struct {
	templateService *service.SLATemplateService
}

// NewSLATemplateController 创建SLA模板控制器
func NewSLATemplateController(templateService *service.SLATemplateService) *SLATemplateController {
	return &SLATemplateController{templateService: templateService}
}

// RegisterRoutes 注册路由
func (c *SLATemplateController) RegisterRoutes(r *gin.RouterGroup) {
	templates := r.Group("/sla/templates")
	{
		templates.GET("", c.ListTemplates)
		templates.GET("/:key", c.GetTemplate)
		templates.POST("/:key/install", c.InstallTemplate)
	}
}

// ListTemplates 列出所有预置SLA模板
// GET /api/v1/sla/templates
func (c *SLATemplateController) ListTemplates(ctx *gin.Context) {
	templates := c.templateService.ListTemplates()
	common.Success(ctx, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}

// GetTemplate 获取单个SLA模板详情
// GET /api/v1/sla/templates/:key
func (c *SLATemplateController) GetTemplate(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		common.Fail(ctx, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tmpl, err := c.templateService.GetTemplate(key)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, err.Error())
		return
	}

	common.Success(ctx, tmpl)
}

// InstallTemplate 将模板安装到当前租户
// POST /api/v1/sla/templates/:key/install
func (c *SLATemplateController) InstallTemplate(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		common.Fail(ctx, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tenantIDVal, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "未授权访问")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(ctx, common.InternalErrorCode, "租户ID类型错误")
		return
	}

	result, err := c.templateService.InstallTemplate(ctx.Request.Context(), key, tenantID)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, err.Error())
		return
	}

	common.Success(ctx, result)
}
