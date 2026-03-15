package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// MenuController 菜单控制器
type MenuController struct {
	menuService *service.MenuService
}

// NewMenuController 创建菜单控制器
func NewMenuController(menuService *service.MenuService) *MenuController {
	return &MenuController{
		menuService: menuService,
	}
}

// ListMenus 获取菜单列表
// @Summary 获取菜单列表
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus [get]
func (c *MenuController) ListMenus(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	menus, err := c.menuService.ListMenus(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, menus)
}

// GetMenu 获取菜单详情
// @Summary 获取菜单详情
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Param id path int true "菜单ID"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus/{id} [get]
func (c *MenuController) GetMenu(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	menu, err := c.menuService.GetMenu(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, menu)
}

// CreateMenu 创建菜单
// @Summary 创建菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Param request body dto.CreateMenuRequest true "创建请求"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus [post]
func (c *MenuController) CreateMenu(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	var req dto.CreateMenuRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	menu, err := c.menuService.CreateMenu(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, menu)
}

// UpdateMenu 更新菜单
// @Summary 更新菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Param id path int true "菜单ID"
// @Param request body dto.UpdateMenuRequest true "更新请求"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus/{id} [put]
func (c *MenuController) UpdateMenu(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	var req dto.UpdateMenuRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	menu, err := c.menuService.UpdateMenu(ctx.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, menu)
}

// DeleteMenu 删除菜单
// @Summary 删除菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Param id path int true "菜单ID"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus/{id} [delete]
func (c *MenuController) DeleteMenu(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	err = c.menuService.DeleteMenu(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, nil)
}

// GetUserMenus 获取当前用户可见菜单
// @Summary 获取当前用户可见菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response
// @Router /api/v1/auth/menus [get]
func (c *MenuController) GetUserMenus(ctx *gin.Context) {
	// 从上下文获取用户ID和租户ID
	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "用户未认证")
		return
	}
	userID := userIDVal.(int)

	tenantIDVal, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
		return
	}
	tenantID := tenantIDVal.(int)

	menus, err := c.menuService.GetUserMenus(ctx.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, menus)
}

// InitDefaultMenus 初始化默认菜单（仅系统初始化时使用）
// @Summary 初始化默认菜单
// @Tags 菜单管理
// @Produce json
// @Security BearerAuth
// @Param tenant_id path int true "租户ID"
// @Success 200 {object} common.Response
// @Router /api/v1/tenants/{tenant_id}/menus/init [post]
func (c *MenuController) InitDefaultMenus(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Param("tenant_id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的租户ID")
		return
	}

	// 检查是否已有菜单
	existing, err := c.menuService.ListMenus(ctx.Request.Context(), tenantID)
	if err == nil && len(existing) > 0 {
		common.Success(ctx, gin.H{
			"message": "菜单已初始化",
			"count":   len(existing),
		})
		return
	}

	common.Success(ctx, gin.H{
		"message": "请通过种子数据初始化菜单",
	})
}
