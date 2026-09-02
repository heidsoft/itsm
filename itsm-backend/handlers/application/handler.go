package application

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler 应用管理HTTP处理器
type Handler struct {
	service *service.ApplicationService
}

// NewHandler creates a new application handler
func NewHandler(client *ent.Client) *Handler {
	return &Handler{service: service.NewApplicationService(client)}
}

// CreateApplication 创建应用
// @Summary 创建应用
// @Description 创建新的应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param request body object true "应用信息"
// @Success 200 {object} common.Response
// @Router /api/v1/applications [post]
func (h *Handler) CreateApplication(ctx *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Code      string `json:"code" binding:"required"`
		Type      string `json:"type"`
		ProjectID int    `json:"projectId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	app, err := h.service.CreateApplication(ctx.Request.Context(), req.Name, req.Code, req.Type, req.ProjectID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, app)
}

// ListApplications 获取应用列表
// @Summary 获取应用列表
// @Description 获取所有应用列表
// @Tags 应用管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/applications [get]
func (h *Handler) ListApplications(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	apps, err := h.service.ListApplications(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, apps)
}

// CreateMicroservice 创建微服务
// @Summary 创建微服务
// @Description 创建新的微服务
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param request body object true "微服务信息"
// @Success 200 {object} common.Response
// @Router /api/v1/applications/microservices [post]
func (h *Handler) CreateMicroservice(ctx *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Code          string `json:"code" binding:"required"`
		Language      string `json:"language"`
		Framework     string `json:"framework"`
		ApplicationID int    `json:"applicationId" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	svc, err := h.service.CreateMicroservice(ctx.Request.Context(), req.Name, req.Code, req.Language, req.Framework, req.ApplicationID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, svc)
}

// UpdateApplication 更新应用
// @Summary 更新应用
// @Description 更新应用信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Param request body object true "应用信息"
// @Success 200 {object} common.Response
// @Router /api/v1/applications/{id} [put]
func (h *Handler) UpdateApplication(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的应用ID")
		return
	}

	var req struct {
		Name      *string `json:"name"`
		Code      *string `json:"code"`
		Type      *string `json:"type"`
		ProjectID *int    `json:"projectId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	app, err := h.service.UpdateApplication(ctx.Request.Context(), id, req.Name, req.Code, req.Type, req.ProjectID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, app)
}

// DeleteApplication 删除应用
// @Summary 删除应用
// @Description 删除指定应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} common.Response
// @Router /api/v1/applications/{id} [delete]
func (h *Handler) DeleteApplication(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的应用ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = h.service.DeleteApplication(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

// ListMicroservices 获取微服务列表
// @Summary 获取微服务列表
// @Description 获取所有微服务列表
// @Tags 应用管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/applications/microservices [get]
func (h *Handler) ListMicroservices(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	microservices, err := h.service.ListMicroservices(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, microservices)
}

// UpdateMicroservice 更新微服务
// @Summary 更新微服务
// @Description 更新微服务信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param id path int true "微服务ID"
// @Param request body object true "微服务信息"
// @Success 200 {object} common.Response
// @Router /api/v1/applications/microservices/{id} [put]
func (h *Handler) UpdateMicroservice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的微服务ID")
		return
	}

	var req struct {
		Name          *string `json:"name"`
		Code          *string `json:"code"`
		Language      *string `json:"language"`
		Framework     *string `json:"framework"`
		ApplicationID *int    `json:"applicationId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	svc, err := h.service.UpdateMicroservice(ctx.Request.Context(), id, req.Name, req.Code, req.Language, req.Framework, req.ApplicationID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, svc)
}

// DeleteMicroservice 删除微服务
// @Summary 删除微服务
// @Description 删除指定微服务
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param id path int true "微服务ID"
// @Success 200 {object} common.Response
// @Router /api/v1/applications/microservices/{id} [delete]
func (h *Handler) DeleteMicroservice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的微服务ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = h.service.DeleteMicroservice(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}
