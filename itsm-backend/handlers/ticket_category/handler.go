// Package ticket_category — 工单分类 handler.
// 迁移自 controller/ticket_category_controller.go，保持原有 API 契约不变。
package ticket_category

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	categoryService *service.TicketCategoryService
	logger          *zap.SugaredLogger
}

type importRow struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	ParentCode  string `json:"parentCode"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    bool   `json:"isActive"`
}

func NewHandler(categoryService *service.TicketCategoryService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{categoryService: categoryService, logger: logger}
}

// CreateCategory POST /api/v1/ticket-categories
func (h *Handler) CreateCategory(c *gin.Context) {
	var req service.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	req.TenantID = tenantID

	category, err := h.categoryService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorw("Failed to create ticket category", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// UpdateCategory PUT /api/v1/ticket-categories/:id
func (h *Handler) UpdateCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	var req service.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	category, err := h.categoryService.UpdateCategory(c.Request.Context(), categoryID, &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to update ticket category", "error", err, "category_id", categoryID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// DeleteCategory DELETE /api/v1/ticket-categories/:id
func (h *Handler) DeleteCategory(c *gin.Context) {
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

	err = h.categoryService.DeleteCategory(c.Request.Context(), categoryID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to delete ticket category", "error", err, "category_id", categoryID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "分类删除成功"})
}

// GetCategory GET /api/v1/ticket-categories/:id
func (h *Handler) GetCategory(c *gin.Context) {
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

	category, err := h.categoryService.GetCategory(c.Request.Context(), categoryID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get ticket category", "error", err, "category_id", categoryID)
		common.Fail(c, common.NotFoundCode, "分类不存在")
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// ListCategories GET /api/v1/ticket-categories
func (h *Handler) ListCategories(c *gin.Context) {
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

	categories, total, err := h.categoryService.ListCategories(c.Request.Context(), req)
	if err != nil {
		h.logger.Errorw("Failed to list ticket categories", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"categories": dto.ToTicketCategoryResponseList(categories),
		"total":      total,
	})
}

// GetCategoryTree GET /api/v1/ticket-categories/tree
func (h *Handler) GetCategoryTree(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	tree, err := h.categoryService.GetCategoryTree(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get category tree", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, tree)
}

// MoveCategory PUT /api/v1/ticket-categories/:id/move
func (h *Handler) MoveCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分类ID")
		return
	}

	var req service.MoveCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	category, err := h.categoryService.MoveCategory(c.Request.Context(), categoryID, &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to move ticket category", "error", err, "category_id", categoryID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketCategoryResponse(category))
}

// PreviewImport POST /api/v1/ticket-categories/import/preview
func (h *Handler) PreviewImport(c *gin.Context) {
	rows, err := h.parseImportRows(c)
	if err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	common.Success(c, rows)
}

// ExecuteImport POST /api/v1/ticket-categories/import
func (h *Handler) ExecuteImport(c *gin.Context) {
	rows, err := h.parseImportRows(c)
	if err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
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
		_, err := h.categoryService.CreateCategory(c.Request.Context(), &service.CreateCategoryRequest{
			Name:        row.Name,
			Code:        row.Code,
			Description: row.Description,
			SortOrder:   row.SortOrder,
			IsActive:    row.IsActive,
			TenantID:    tenantID,
		})
		if err != nil {
			h.logger.Warnw("Failed to import ticket category", "error", err, "code", row.Code, "tenant_id", tenantID)
			failed++
			continue
		}
		success++
	}

	common.Success(c, gin.H{"success": success, "failed": failed})
}

func (h *Handler) parseImportRows(c *gin.Context) ([]importRow, error) {
	const maxSize = 2 << 20 // 2 MiB
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return nil, err
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxSize {
		return nil, fmt.Errorf("导入文件大小必须在 1 字节到 2 MiB 之间")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxSize {
		return nil, fmt.Errorf("导入文件超过 2 MiB 限制")
	}
	if len(content) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	trimmed := strings.TrimSpace(string(content))
	if strings.HasPrefix(trimmed, "[") {
		var rows []importRow
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
		return []importRow{}, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for i, header := range records[0] {
		headerIndex[strings.ToLower(strings.TrimSpace(header))] = i
	}

	rows := make([]importRow, 0, len(records)-1)
	for _, record := range records[1:] {
		row := importRow{
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

func normalizeImportRows(rows []importRow) []importRow {
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
