package project

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Handler 项目管理HTTP处理器
type Handler struct {
	service interface {
		CreateProject(ctx interface{}, name, code string, deptID, managerID, tenantID int) (*ent.Project, error)
		ListProjects(ctx interface{}, tenantID int) ([]*ent.Project, error)
		UpdateProject(ctx interface{}, id int, name, code *string, deptID, managerID *int, tenantID int) (*ent.Project, error)
		DeleteProject(ctx interface{}, id int, tenantID int) error
		GetProject(ctx interface{}, id int, tenantID int) (*ent.Project, error)
	}
}

// NewHandler creates a new project handler
func NewHandler(service interface {
	CreateProject(ctx interface{}, name, code string, deptID, managerID, tenantID int) (*ent.Project, error)
	ListProjects(ctx interface{}, tenantID int) ([]*ent.Project, error)
	UpdateProject(ctx interface{}, id int, name, code *string, deptID, managerID *int, tenantID int) (*ent.Project, error)
	DeleteProject(ctx interface{}, id int, tenantID int) error
	GetProject(ctx interface{}, id int, tenantID int) (*ent.Project, error)
}) *Handler {
	return &Handler{service: service}
}

// CreateProject 创建项目
// @Summary 创建项目
// @Description 创建新的项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param name query string true "项目名称"
// @Param code query string true "项目代码"
// @Param deptId query int false "部门ID"
// @Param managerId query int false "项目经理ID"
// @Success 200 {object} common.Response{data=ent.Project}
// @Router /api/v1/projects [post]
func (h *Handler) CreateProject(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	name := c.Query("name")
	code := c.Query("code")
	if name == "" || code == "" {
		common.Fail(c, common.ParamErrorCode, "项目名称和代码不能为空")
		return
	}

	var deptID, managerID int
	if deptIDStr := c.Query("deptId"); deptIDStr != "" {
		deptID, _ = strconv.Atoi(deptIDStr)
	}
	if managerIDStr := c.Query("managerId"); managerIDStr != "" {
		managerID, _ = strconv.Atoi(managerIDStr)
	}

	project, err := h.service.CreateProject(c.Request.Context(), name, code, deptID, managerID, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, project)
}

// GetProject 获取项目详情
// @Summary 获取项目详情
// @Description 获取指定项目的详细信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} common.Response{data=ent.Project}
// @Router /api/v1/projects/{id} [get]
func (h *Handler) GetProject(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的项目ID")
		return
	}

	project, err := h.service.GetProject(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, project)
}

// UpdateProject 更新项目
// @Summary 更新项目
// @Description 更新指定项目的信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param name query string false "项目名称"
// @Param code query string false "项目代码"
// @Param deptId query int false "部门ID"
// @Param managerId query int false "项目经理ID"
// @Success 200 {object} common.Response{data=ent.Project}
// @Router /api/v1/projects/{id} [put]
func (h *Handler) UpdateProject(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的项目ID")
		return
	}

	var name, code *string
	var deptID, managerID *int

	if n := c.Query("name"); n != "" {
		name = &n
	}
	if c := c.Query("code"); c != "" {
		code = &c
	}
	if deptIDStr := c.Query("deptId"); deptIDStr != "" {
		if d, err := strconv.Atoi(deptIDStr); err == nil {
			deptID = &d
		}
	}
	if managerIDStr := c.Query("managerId"); managerIDStr != "" {
		if m, err := strconv.Atoi(managerIDStr); err == nil {
			managerID = &m
		}
	}

	project, err := h.service.UpdateProject(c.Request.Context(), id, name, code, deptID, managerID, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, project)
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
func (h *Handler) DeleteProject(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的项目ID")
		return
	}

	err = h.service.DeleteProject(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}

// ListProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取所有项目列表
// @Tags 项目管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=[]ent.Project}
// @Router /api/v1/projects [get]
func (h *Handler) ListProjects(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	projects, err := h.service.ListProjects(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, projects)
}
