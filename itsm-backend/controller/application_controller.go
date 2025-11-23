package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type ApplicationController struct {
	service *service.ApplicationService
}

func NewApplicationController(client *ent.Client) *ApplicationController {
	return &ApplicationController{
		service: service.NewApplicationService(client),
	}
}

// CreateApplication 创建应用
func (c *ApplicationController) CreateApplication(ctx *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Code      string `json:"code" binding:"required"`
		Type      string `json:"type"`
		ProjectID int    `json:"project_id"`
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
	app, err := c.service.CreateApplication(ctx.Request.Context(), req.Name, req.Code, req.Type, req.ProjectID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, app)
}

// ListApplications 获取应用列表
func (c *ApplicationController) ListApplications(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	apps, err := c.service.ListApplications(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, apps)
}

// CreateMicroservice 创建微服务
func (c *ApplicationController) CreateMicroservice(ctx *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Code          string `json:"code" binding:"required"`
		Language      string `json:"language"`
		Framework     string `json:"framework"`
		ApplicationID int    `json:"application_id" binding:"required"`
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
	svc, err := c.service.CreateMicroservice(ctx.Request.Context(), req.Name, req.Code, req.Language, req.Framework, req.ApplicationID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, svc)
}

// UpdateApplication 更新应用
func (c *ApplicationController) UpdateApplication(ctx *gin.Context) {
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
		ProjectID *int    `json:"project_id"`
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

	app, err := c.service.UpdateApplication(ctx.Request.Context(), id, req.Name, req.Code, req.Type, req.ProjectID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, app)
}

// DeleteApplication 删除应用
func (c *ApplicationController) DeleteApplication(ctx *gin.Context) {
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

	err = c.service.DeleteApplication(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

// ListMicroservices 获取微服务列表
func (c *ApplicationController) ListMicroservices(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	microservices, err := c.service.ListMicroservices(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, microservices)
}

// UpdateMicroservice 更新微服务
func (c *ApplicationController) UpdateMicroservice(ctx *gin.Context) {
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
		ApplicationID *int    `json:"application_id"`
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

	svc, err := c.service.UpdateMicroservice(ctx.Request.Context(), id, req.Name, req.Code, req.Language, req.Framework, req.ApplicationID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, svc)
}

// DeleteMicroservice 删除微服务
func (c *ApplicationController) DeleteMicroservice(ctx *gin.Context) {
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

	err = c.service.DeleteMicroservice(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}

