package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketCategoryController struct {
	categoryService *service.TicketCategoryService
	logger          *zap.SugaredLogger
}

func NewTicketCategoryController(categoryService *service.TicketCategoryService, logger *zap.SugaredLogger) *TicketCategoryController {
	return &TicketCategoryController{
		categoryService: categoryService,
		logger:          logger,
	}
}

// CreateCategory 创建工单分类
func (tc *TicketCategoryController) CreateCategory(c *gin.Context) {
	var req service.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.TenantID = tenantID

	category, err := tc.categoryService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket category", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, category)
}

// UpdateCategory 更新工单分类
func (tc *TicketCategoryController) UpdateCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	var req service.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	category, err := tc.categoryService.UpdateCategory(c.Request.Context(), categoryID, &req)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, category)
}

// DeleteCategory 删除工单分类
func (tc *TicketCategoryController) DeleteCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	err = tc.categoryService.DeleteCategory(c.Request.Context(), categoryID)
	if err != nil {
		tc.logger.Errorw("Failed to delete ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "分类删除成功"})
}

// GetCategory 获取分类详情
func (tc *TicketCategoryController) GetCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	category, err := tc.categoryService.GetCategory(c.Request.Context(), categoryID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.NotFoundCode, "分类不存在")
		return
	}

	common.Success(c, category)
}

// ListCategories 获取分类列表
func (tc *TicketCategoryController) ListCategories(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	parentIDStr := c.Query("parent_id")
	levelStr := c.Query("level")
	activeStr := c.Query("active")

	var parentID *int
	var level int
	var active *bool

	if parentIDStr != "" {
		if id, err := strconv.Atoi(parentIDStr); err == nil {
			parentID = &id
		}
	}

	if levelStr != "" {
		if l, err := strconv.Atoi(levelStr); err == nil {
			level = l
		}
	}

	if activeStr != "" {
		if a, err := strconv.ParseBool(activeStr); err == nil {
			active = &a
		}
	}

	req := &service.ListCategoriesRequest{
		Page:      1,
		PageSize:  100,
		ParentID:  parentID,
		Level:     level,
		IsActive:  active,
		TenantID:  tenantID,
	}

	categories, total, err := tc.categoryService.ListCategories(c.Request.Context(), req)
	if err != nil {
		tc.logger.Errorw("Failed to list ticket categories", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"categories": categories,
		"total":      total,
	})
}

// GetCategoryTree 获取分类树形结构
func (tc *TicketCategoryController) GetCategoryTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	tree, err := tc.categoryService.GetCategoryTree(c.Request.Context(), tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get category tree", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tree)
}

// MoveCategory 移动分类位置
func (tc *TicketCategoryController) MoveCategory(c *gin.Context) {
	_, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	var req struct {
		NewParentID  *int `json:"new_parent_id"`
		NewSortOrder *int `json:"new_sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 注意：MoveCategory 方法尚未在服务中实现
	common.Fail(c, common.InternalErrorCode, "分类移动功能尚未实现")
}
