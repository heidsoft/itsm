package department

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler 部门管理HTTP处理器
type Handler struct {
	departmentService *service.DepartmentService
}

// NewHandler creates a new department handler
func NewHandler(svc *service.DepartmentService) *Handler {
	return &Handler{departmentService: svc}
}

// CreateDepartment 创建部门
// @Summary 创建部门
// @Description 创建新的部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param name query string true "部门名称"
// @Param code query string true "部门代码"
// @Param description query string false "部门描述"
// @Param managerId query int false "部门经理ID"
// @Param parentId query int false "父部门ID"
// @Success 200 {object} common.Response{data=ent.Department}
// @Router /api/v1/departments [post]
func (h *Handler) CreateDepartment(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	name := c.Query("name")
	code := c.Query("code")
	if name == "" || code == "" {
		common.Fail(c, common.ParamErrorCode, "部门名称和代码不能为空")
		return
	}

	description := c.Query("description")
	var managerID, parentID int
	if managerIDStr := c.Query("managerId"); managerIDStr != "" {
		managerID, _ = strconv.Atoi(managerIDStr)
	}
	if parentIDStr := c.Query("parentId"); parentIDStr != "" {
		parentID, _ = strconv.Atoi(parentIDStr)
	}

	dept, err := h.departmentService.CreateDepartment(c.Request.Context(), name, code, description, managerID, parentID, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dept)
}

// GetDepartment 获取部门详情
// @Summary 获取部门详情
// @Description 获取指定部门的详细信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path int true "部门ID"
// @Success 200 {object} common.Response{data=ent.Department}
// @Router /api/v1/departments/{id} [get]
func (h *Handler) GetDepartment(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的部门ID")
		return
	}

	dept, err := h.departmentService.GetDepartmentByID(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dept)
}

// UpdateDepartment 更新部门
// @Summary 更新部门
// @Description 更新指定部门的信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path int true "部门ID"
// @Param name query string false "部门名称"
// @Param code query string false "部门代码"
// @Param description query string false "部门描述"
// @Param managerId query int false "部门经理ID"
// @Param parentId query int false "父部门ID"
// @Success 200 {object} common.Response{data=ent.Department}
// @Router /api/v1/departments/{id} [put]
func (h *Handler) UpdateDepartment(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的部门ID")
		return
	}

	var name, code, description string
	var managerID, parentID int
	var hasName, hasCode, hasDesc bool

	if n := c.Query("name"); n != "" {
		name = n
		hasName = true
	}
	if c := c.Query("code"); c != "" {
		code = c
		hasCode = true
	}
	if d := c.Query("description"); d != "" {
		description = d
		hasDesc = true
	}
	if managerIDStr := c.Query("managerId"); managerIDStr != "" {
		managerID, _ = strconv.Atoi(managerIDStr)
	}
	if parentIDStr := c.Query("parentId"); parentIDStr != "" {
		parentID, _ = strconv.Atoi(parentIDStr)
	}

	dept, err := h.departmentService.UpdateDepartment(c.Request.Context(), id, name, code, description, managerID, parentID, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	_ = hasName && hasCode && hasDesc
	common.Success(c, dept)
}

// DeleteDepartment 删除部门
// @Summary 删除部门
// @Description 删除指定部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path int true "部门ID"
// @Success 200 {object} common.Response
// @Router /api/v1/departments/{id} [delete]
func (h *Handler) DeleteDepartment(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的部门ID")
		return
	}

	err = h.departmentService.DeleteDepartment(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}

// GetDepartmentTree 获取部门树
// @Summary 获取部门树
// @Description 获取部门树形结构
// @Tags 部门管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/departments/tree [get]
func (h *Handler) GetDepartmentTree(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	tree, err := h.departmentService.GetDepartmentTree(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tree)
}
