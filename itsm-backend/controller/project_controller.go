package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type ProjectController struct {
	service *service.ProjectService
}

func NewProjectController(client *ent.Client) *ProjectController {
	return &ProjectController{
		service: service.NewProjectService(client),
	}
}

// CreateProject 创建项目
// @Summary 创建项目
// @Description 创建新的项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param request body object true "项目信息"
// @Success 200 {object} common.Response
// @Router /api/v1/projects [post]
func (c *ProjectController) CreateProject(ctx *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Code         string `json:"code" binding:"required"`
		DepartmentID int    `json:"department_id"`
		ManagerID    int    `json:"manager_id"`
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
	project, err := c.service.CreateProject(ctx.Request.Context(), req.Name, req.Code, req.DepartmentID, req.ManagerID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, project)
}

// ListProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取所有项目列表
// @Tags 项目管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/projects [get]
func (c *ProjectController) ListProjects(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	projects, err := c.service.ListProjects(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, projects)
}

// UpdateProject 更新项目
func (c *ProjectController) UpdateProject(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的项目ID")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		Code         *string `json:"code"`
		DepartmentID *int    `json:"department_id"`
		ManagerID    *int    `json:"manager_id"`
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

	project, err := c.service.UpdateProject(ctx.Request.Context(), id, req.Name, req.Code, req.DepartmentID, req.ManagerID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, project)
}

// DeleteProject 删除项目
func (c *ProjectController) DeleteProject(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的项目ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = c.service.DeleteProject(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}
