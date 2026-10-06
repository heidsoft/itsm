package project

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler 项目管理HTTP处理器
type Handler struct {
	service *service.ProjectService
}

// NewHandler creates a new project handler
func NewHandler(svc *service.ProjectService) *Handler {
	return &Handler{service: svc}
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
func (h *Handler) CreateProject(ctx *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Code         string `json:"code" binding:"required"`
		DepartmentID int    `json:"departmentId"`
		ManagerID    int    `json:"managerId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	project, err := h.service.CreateProject(ctx.Request.Context(), req.Name, req.Code, req.DepartmentID, req.ManagerID, tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
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
func (h *Handler) ListProjects(ctx *gin.Context) {
	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	projects, err := h.service.ListProjects(ctx.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}

	common.Success(ctx, projects)
}

// GetProject 获取单个项目
// @Summary 获取单个项目
// @Description 获取指定项目的详细信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} common.Response
// @Router /api/v1/projects/{id} [get]
func (h *Handler) GetProject(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的项目ID")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	project, err := h.service.GetProject(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}

	common.Success(ctx, project)
}

// UpdateProject 更新项目
// @Summary 更新项目
// @Description 更新项目信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param request body object true "项目信息"
// @Success 200 {object} common.Response
// @Router /api/v1/projects/{id} [put]
func (h *Handler) UpdateProject(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的项目ID")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		Code         *string `json:"code"`
		DepartmentID *int    `json:"departmentId"`
		ManagerID    *int    `json:"managerId"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	project, err := h.service.UpdateProject(ctx.Request.Context(), id, req.Name, req.Code, req.DepartmentID, req.ManagerID, tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}

	common.Success(ctx, project)
}

// DeleteProject 删除项目
// @Summary 删除项目
// @Description 删除指定项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} common.Response
// @Router /api/v1/projects/{id} [delete]
func (h *Handler) DeleteProject(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的项目ID")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	err = h.service.DeleteProject(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(ctx, err, "操作失败")
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}
