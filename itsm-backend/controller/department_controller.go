package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type DepartmentController struct {
	service *service.DepartmentService
}

func NewDepartmentController(client *ent.Client) *DepartmentController {
	return &DepartmentController{
		service: service.NewDepartmentService(client),
	}
}

// CreateDepartment 创建部门
func (c *DepartmentController) CreateDepartment(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
		ParentID    int    `json:"parent_id"`
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
	dept, err := c.service.CreateDepartment(ctx.Request.Context(), req.Name, req.Code, req.Description, req.ManagerID, req.ParentID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dept)
}

// GetDepartmentTree 获取部门树
func (c *DepartmentController) GetDepartmentTree(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}
	tree, err := c.service.GetDepartmentTree(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, tree)
}

// UpdateDepartment 更新部门
func (c *DepartmentController) UpdateDepartment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的部门ID")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
		ManagerID   int    `json:"manager_id"`
		ParentID    int    `json:"parent_id"`
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

	dept, err := c.service.UpdateDepartment(ctx.Request.Context(), id, req.Name, req.Code, req.Description, req.ManagerID, req.ParentID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, dept)
}

// DeleteDepartment 删除部门
func (c *DepartmentController) DeleteDepartment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的部门ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = c.service.DeleteDepartment(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除成功"})
}


