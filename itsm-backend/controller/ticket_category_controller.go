package controller

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketCategoryController struct {
	categoryService *service.TicketCategoryService
	logger          *zap.SugaredLogger
}

type ticketCategoryImportRow struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	ParentCode  string `json:"parentCode"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    bool   `json:"isActive"`
}

func NewTicketCategoryController(categoryService *service.TicketCategoryService, logger *zap.SugaredLogger) *TicketCategoryController {
	return &TicketCategoryController{
		categoryService: categoryService,
		logger:          logger,
	}
}

// CreateCategory 创建工单分类
// @Summary 创建工单分类
// @Description 创建新的工单分类
// @Tags 工单分类
// @Accept json
// @Produce json
// @Param request body service.CreateCategoryRequest true "分类信息"
// @Success 200 {object} common.Response
// @Router /api/v1/ticket-categories [post]
func (tc *TicketCategoryController) CreateCategory(c *gin.Context) {
	var req service.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	req.TenantID = tenantID

	category, err := tc.categoryService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		tc.logger.Errorw("Failed to create ticket category", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// UpdateCategory 更新工单分类
// @Summary 更新工单分类
// @Description 更新工单分类信息
// @Tags 工单分类
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Param request body service.CreateCategoryRequest true "分类信息"
// @Success 200 {object} common.Response
// @Router /api/v1/ticket-categories/{id} [put]
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

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	category, err := tc.categoryService.UpdateCategory(c.Request.Context(), categoryID, &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to update ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// DeleteCategory 删除工单分类
func (tc *TicketCategoryController) DeleteCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	err = tc.categoryService.DeleteCategory(c.Request.Context(), categoryID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to delete ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "分类删除成功"})
}

// PreviewImport 预览工单分类导入数据
func (tc *TicketCategoryController) PreviewImport(c *gin.Context) {
	rows, err := tc.parseImportRows(c)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	common.Success(c, rows)
}

// ExecuteImport 执行工单分类导入
func (tc *TicketCategoryController) ExecuteImport(c *gin.Context) {
	rows, err := tc.parseImportRows(c)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	success, failed := 0, 0
	for _, row := range rows {
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Code) == "" {
			failed++
			continue
		}
		_, err := tc.categoryService.CreateCategory(c.Request.Context(), &service.CreateCategoryRequest{
			Name:        row.Name,
			Code:        row.Code,
			Description: row.Description,
			SortOrder:   row.SortOrder,
			IsActive:    row.IsActive,
			TenantID:    tenantID,
		})
		if err != nil {
			tc.logger.Warnw("Failed to import ticket category", "error", err, "code", row.Code, "tenant_id", tenantID)
			failed++
			continue
		}
		success++
	}

	common.Success(c, gin.H{"success": success, "failed": failed})
}

func (tc *TicketCategoryController) parseImportRows(c *gin.Context) ([]ticketCategoryImportRow, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	trimmed := strings.TrimSpace(string(content))
	if strings.HasPrefix(trimmed, "[") {
		var rows []ticketCategoryImportRow
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, err
		}
		return normalizeImportRows(rows), nil
	}

	reader := csv.NewReader(strings.NewReader(trimmed))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return []ticketCategoryImportRow{}, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for i, header := range records[0] {
		headerIndex[strings.ToLower(strings.TrimSpace(header))] = i
	}

	rows := make([]ticketCategoryImportRow, 0, len(records)-1)
	for _, record := range records[1:] {
		row := ticketCategoryImportRow{
			Name:        importCell(record, headerIndex, "name"),
			Code:        importCell(record, headerIndex, "code"),
			Description: importCell(record, headerIndex, "description"),
			ParentCode:  importCell(record, headerIndex, "parent_code"),
			SortOrder:   parseImportInt(importCell(record, headerIndex, "sort_order")),
			IsActive:    parseImportBool(importCell(record, headerIndex, "is_active"), true),
		}
		rows = append(rows, row)
	}

	return normalizeImportRows(rows), nil
}

func normalizeImportRows(rows []ticketCategoryImportRow) []ticketCategoryImportRow {
	for i := range rows {
		rows[i].Name = strings.TrimSpace(rows[i].Name)
		rows[i].Code = strings.TrimSpace(rows[i].Code)
		rows[i].Description = strings.TrimSpace(rows[i].Description)
		rows[i].ParentCode = strings.TrimSpace(rows[i].ParentCode)
		if rows[i].Code == "" {
			rows[i].Code = rows[i].Name
		}
	}
	return rows
}

func importCell(record []string, headerIndex map[string]int, key string) string {
	index, ok := headerIndex[key]
	if !ok || index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func parseImportInt(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseImportBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetCategory 获取分类详情
func (tc *TicketCategoryController) GetCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	category, err := tc.categoryService.GetCategory(c.Request.Context(), categoryID, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to get ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.NotFoundCode, "分类不存在")
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
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
		Page:     1,
		PageSize: 100,
		ParentID: parentID,
		Level:    level,
		IsActive: active,
		TenantID: tenantID,
	}

	categories, total, err := tc.categoryService.ListCategories(c.Request.Context(), req)
	if err != nil {
		tc.logger.Errorw("Failed to list ticket categories", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"categories": dto.ToTicketCategoryResponseList(categories),
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
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	var req service.MoveCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	category, err := tc.categoryService.MoveCategory(c.Request.Context(), categoryID, &req, tenantID)
	if err != nil {
		tc.logger.Errorw("Failed to move ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}
