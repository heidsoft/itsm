package common

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationRequest 分页请求参数
type PaginationRequest struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"page_size" form:"page_size" binding:"min=1,max=100"`
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ListResponse 列表响应结构
type ListResponse struct {
	Data       interface{}         `json:"data"`
	Pagination *PaginationResponse `json:"pagination"`
}

// GetPaginationFromQuery 从查询参数中获取分页信息
func GetPaginationFromQuery(c *gin.Context) *PaginationRequest {
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	return &PaginationRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// GetOffset 计算数据库查询的偏移量
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取查询限制数量
func (p *PaginationRequest) GetLimit() int {
	return p.PageSize
}

// NewPaginationResponse 创建分页响应
func NewPaginationResponse(page, pageSize int, total int64) *PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	
	return &PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// NewListResponse 创建列表响应
func NewListResponse(data interface{}, pagination *PaginationResponse) *ListResponse {
	return &ListResponse{
		Data:       data,
		Pagination: pagination,
	}
}

// SuccessWithPagination 带分页的成功响应
func SuccessWithPagination(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	pagination := NewPaginationResponse(page, pageSize, total)
	response := NewListResponse(data, pagination)
	Success(c, response)
}

// 常用分页配置
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// ValidatePagination 验证分页参数
func ValidatePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = DefaultPage
	}
	
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	
	return page, pageSize
}

// PaginationMeta 分页元数据（用于数据库查询）
type PaginationMeta struct {
	Offset int
	Limit  int
	Page   int
	Size   int
}

// NewPaginationMeta 创建分页元数据
func NewPaginationMeta(page, pageSize int) *PaginationMeta {
	page, pageSize = ValidatePagination(page, pageSize)
	
	return &PaginationMeta{
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
		Page:   page,
		Size:   pageSize,
	}
}

// GetPaginationMeta 从Gin上下文获取分页元数据
func GetPaginationMeta(c *gin.Context) *PaginationMeta {
	pagination := GetPaginationFromQuery(c)
	return NewPaginationMeta(pagination.Page, pagination.PageSize)
}