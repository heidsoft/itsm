package bpmn

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

type WorkflowTemplateHandler struct {
	catalog *service.BPMNWorkflowTemplateCatalog
}

func (h *WorkflowTemplateHandler) RegisterRoutes(r *gin.RouterGroup) {
	routes := r.Group("/bpmn/ai/templates")
	routes.GET("", middleware.RequirePermission("workflow", "read"), h.List)
	routes.GET("/:key", middleware.RequirePermission("workflow", "read"), h.Get)
	routes.GET("/:key/versions", middleware.RequirePermission("workflow", "read"), h.Versions)
	routes.POST("", middleware.RequirePermission("workflow", "create"), h.Create)
	routes.PUT("/:key", middleware.RequirePermission("workflow", "update"), h.Update)
	routes.POST("/:key/publish", middleware.RequirePermission("workflow", "update"), h.Publish)
	routes.POST("/:key/archive", middleware.RequirePermission("workflow", "update"), h.Archive)
	routes.POST("/:key/reload", middleware.RequirePermission("workflow", "update"), h.Reload)
}

func NewWorkflowTemplateHandler(catalog *service.BPMNWorkflowTemplateCatalog) *WorkflowTemplateHandler {
	return &WorkflowTemplateHandler{catalog: catalog}
}

func (h *WorkflowTemplateHandler) List(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	result, err := h.catalog.List(ctx.Request.Context(), tenantID, ctx.Query("keyword"), ctx.Query("domain"), ctx.Query("status"), page, pageSize)
	if err != nil {
		common.Fail(ctx, common.ServiceUnavailableCode, "模板目录暂不可用")
		return
	}
	common.Success(ctx, result)
}

func (h *WorkflowTemplateHandler) Get(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	result, err := h.catalog.Get(ctx.Request.Context(), tenantID, ctx.Param("key"), ctx.Query("version"))
	if errors.Is(err, service.ErrWorkflowTemplateNotFound) {
		common.Fail(ctx, common.NotFoundCode, "模板不存在")
		return
	}
	if err != nil {
		common.Fail(ctx, common.ServiceUnavailableCode, "模板目录暂不可用")
		return
	}
	common.Success(ctx, result)
}

func (h *WorkflowTemplateHandler) Create(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	userID, err := middleware.GetUserID(ctx)
	if err != nil || userID <= 0 {
		common.AuthFailed(ctx, "未授权访问：缺少有效操作者")
		return
	}
	var req dto.CreateWorkflowTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	result, err := h.catalog.CreateDraft(ctx.Request.Context(), tenantID, userID, &req)
	if errors.Is(err, service.ErrWorkflowTemplateConflict) {
		common.Fail(ctx, common.ConflictCode, "模板 key 已存在")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateInvalid) {
		common.Fail(ctx, common.ValidationError, "模板参数或 BPMN 内容无效")
		return
	}
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建模板失败")
		return
	}
	ctx.JSON(http.StatusCreated, common.Response{Code: common.SuccessCode, Message: "success", Data: result})
}

func (h *WorkflowTemplateHandler) Update(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	var req dto.UpdateWorkflowTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	result, err := h.catalog.UpdateDraft(ctx.Request.Context(), tenantID, ctx.Param("key"), &req)
	if errors.Is(err, service.ErrWorkflowTemplateNotFound) {
		common.Fail(ctx, common.NotFoundCode, "模板不存在")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateConflict) {
		common.Fail(ctx, common.ConflictCode, "只有草稿版本可以编辑")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateInvalid) {
		common.Fail(ctx, common.ValidationError, "模板参数无效")
		return
	}
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新模板失败")
		return
	}
	common.Success(ctx, result)
}

func (h *WorkflowTemplateHandler) Publish(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	result, err := h.catalog.Publish(ctx.Request.Context(), tenantID, ctx.Param("key"))
	if errors.Is(err, service.ErrWorkflowTemplateNotFound) {
		common.Fail(ctx, common.NotFoundCode, "模板不存在")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateConflict) {
		common.Fail(ctx, common.ConflictCode, "只有草稿版本可以发布")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateInvalid) {
		common.Fail(ctx, common.ValidationError, "模板未通过 BPMN 校验")
		return
	}
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "发布模板失败")
		return
	}
	common.Success(ctx, result)
}

func (h *WorkflowTemplateHandler) Archive(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	err := h.catalog.Archive(ctx.Request.Context(), tenantID, ctx.Param("key"))
	if errors.Is(err, service.ErrWorkflowTemplateNotFound) {
		common.Fail(ctx, common.NotFoundCode, "已发布模板不存在")
		return
	}
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "停用模板失败")
		return
	}
	common.Success(ctx, gin.H{"key": ctx.Param("key"), "status": "archived"})
}

// Reload 重新部署已发布模板为 BPMN 流程定义。
// 仅取已发布（status='published'）的最新版本，不会把 draft 内容写入运行时。
// 返回新部署的 deployment 与 process_definition 信息，便于前端展示版本跃迁。
func (h *WorkflowTemplateHandler) Reload(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	key := ctx.Param("key")
	result, err := h.catalog.Reload(ctx.Request.Context(), tenantID, key)
	if errors.Is(err, service.ErrWorkflowTemplateNotFound) {
		common.Fail(ctx, common.NotFoundCode, "已发布模板不存在，无法重载")
		return
	}
	if errors.Is(err, service.ErrWorkflowTemplateInvalid) {
		common.Fail(ctx, common.ValidationError, err.Error())
		return
	}
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "重载模板失败")
		return
	}
	common.Success(ctx, result)
}

func (h *WorkflowTemplateHandler) Versions(ctx *gin.Context) {
	tenantID, ok := templateTenant(ctx)
	if !ok {
		return
	}
	result, err := h.catalog.Versions(ctx.Request.Context(), tenantID, ctx.Param("key"))
	if err != nil {
		common.Fail(ctx, common.ServiceUnavailableCode, "模板版本暂不可用")
		return
	}
	common.Success(ctx, result)
}

func templateTenant(ctx *gin.Context) (int, bool) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID <= 0 {
		common.AuthFailed(ctx, "未授权访问：缺少有效租户上下文")
		return 0, false
	}
	return tenantID, true
}
