// Package sla_template 是 SLA 模板域的 HTTP handler 层（域切片架构）。
// 模板目录与安装规则由 service.SLATemplateService 承载，本包只做租户上下文、
// DTO 投影与错误映射。
package sla_template

import (
	"errors"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/middleware"
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

func (h *Handler) toResponse(tmpl service.SLATemplate, state service.SLATemplateInstallInfo) *dto.SLATemplateResponse {
	resp := &dto.SLATemplateResponse{
		Key:              tmpl.Key,
		Name:             tmpl.Name,
		Description:      tmpl.Description,
		ServiceType:      tmpl.ServiceType,
		Priority:         tmpl.Priority,
		ResponseTime:     tmpl.ResponseTime,
		ResolutionTime:   tmpl.ResolutionTime,
		BusinessHours:    dto.ProjectSLABusinessHours(tmpl.BusinessHours),
		EscalationLevels: dto.ProjectSLAEscalationLevels(tmpl.EscalationRules),
		Conditions:       tmpl.Conditions,
		Industry:         tmpl.Industry,
		Recommended:      tmpl.Recommended,
		Installed:        state.SLADefinitionID > 0,
		SLADefinitionID:  state.SLADefinitionID,
	}
	if state.SLADefinitionID > 0 {
		installedAt := state.InstalledAt
		resp.InstalledAt = &installedAt
	}
	if resp.Conditions == nil {
		resp.Conditions = map[string]interface{}{}
	}
	return resp
}

// resolveTenant 统一 fail-closed：缺租户上下文时不返回「谁都没装过」的空状态。
// ResolveTenantID 解析失败时已经写入 401/403，这里只补它没写响应的分支。
func resolveTenant(c *gin.Context) (int, bool) {
	tid, ok := handlerctx.ResolveTenantID(c)
	if ok && tid > 0 {
		return tid, true
	}
	if !c.Writer.Written() {
		common.Fail(c, common.AuthFailedCode, "未授权访问")
	}
	return 0, false
}

// ListTemplates 列出预置 SLA 模板及其在当前租户的安装状态
func (h *Handler) ListTemplates(c *gin.Context) {
	tid, ok := resolveTenant(c)
	if !ok {
		return
	}

	states, err := h.templateService.InstallStates(c.Request.Context(), tid)
	if err != nil {
		common.RespondError(c, err, "查询 SLA 模板安装状态失败")
		return
	}

	templates := h.templateService.ListTemplates()
	items := make([]*dto.SLATemplateResponse, 0, len(templates))
	for _, tmpl := range templates {
		items = append(items, h.toResponse(tmpl, states[tmpl.Key]))
	}

	common.Success(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

// GetTemplate 获取单个模板详情（含安装状态）
func (h *Handler) GetTemplate(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		common.Fail(c, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tid, ok := resolveTenant(c)
	if !ok {
		return
	}

	tmpl, err := h.templateService.GetTemplate(key)
	if err != nil {
		if errors.Is(err, service.ErrSLATemplateNotFound) {
			common.NotFoundWithErr(c, err, "SLA 模板不存在")
			return
		}
		common.RespondError(c, err, "查询 SLA 模板失败")
		return
	}

	states, err := h.templateService.InstallStates(c.Request.Context(), tid)
	if err != nil {
		common.RespondError(c, err, "查询 SLA 模板安装状态失败")
		return
	}

	common.Success(c, h.toResponse(*tmpl, states[key]))
}

// InstallTemplate 将模板安装到当前租户
func (h *Handler) InstallTemplate(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		common.Fail(c, common.ParamErrorCode, "模板 key 不能为空")
		return
	}

	tid, ok := resolveTenant(c)
	if !ok {
		return
	}

	result, err := h.templateService.InstallTemplate(c.Request.Context(), key, tid)
	if err != nil {
		if errors.Is(err, service.ErrSLATemplateNotFound) {
			common.NotFoundWithErr(c, err, "SLA 模板不存在")
			return
		}
		common.RespondError(c, err, "安装 SLA 模板失败")
		return
	}

	common.Success(c, result)
}

// RegisterRoutes 注册路由（挂在 /api/v1/sla/templates 下）。
// 三个端点都必须过 RequirePermission：安装是写操作，只给列表挂权限会让任何登录用户
// 都能往租户里创建 SLA 定义。
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	templates := r.Group("/sla/templates")
	{
		templates.GET("", middleware.RequirePermission("sla", "read"), h.ListTemplates)
		templates.GET("/:key", middleware.RequirePermission("sla", "read"), h.GetTemplate)
		templates.POST("/:key/install", middleware.RequirePermission("sla", "write"), h.InstallTemplate)
	}
}
