package common

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationRequest 分页请求参数
type PaginationRequest struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"pageSize" form:"pageSize" binding:"min=1,max=100"`
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	HasNext    bool  `json:"hasNext"`
	HasPrev    bool  `json:"hasPrev"`
}

// ListResponse 列表响应结构（唯一权威信封）：
// data = {items,total,page,pageSize,totalPages}。
//
// 历史上这里还会做两件事，均已删除：
//   - 按 slice 元素类型追加领域名别名（items 与 tickets 同时返回）；
//   - 再嵌套一个 pagination 对象，把同一组分页事实返回两遍。
//
// 两者都属于 AGENTS.md 禁止的双轨契约，消费方一律读平铺的 items/total/page/pageSize/totalPages。
type ListResponse struct {
	Items      interface{}         `json:"items"`
	Pagination *PaginationResponse `json:"-"`
}

// MarshalJSON 平铺分页元数据。键集合固定，不随元素类型变化。
func (l *ListResponse) MarshalJSON() ([]byte, error) {
	flat := map[string]interface{}{
		"items": l.Items,
	}
	if l.Pagination != nil {
		flat["total"] = l.Pagination.Total
		flat["page"] = l.Pagination.Page
		flat["pageSize"] = l.Pagination.PageSize
		flat["totalPages"] = l.Pagination.TotalPages
	}

	return json.Marshal(flat)
}

// GetPaginationFromQuery 从查询参数中获取分页信息
// 契约：查询参数统一使用 camelCase（pageSize），与请求 DTO 的 form tag 一致；
// 旧的 page_size 形态不再解析，存量调用方需按 API 契约迁移。
func GetPaginationFromQuery(c *gin.Context) *PaginationRequest {
	page := 1
	pageSize := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
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
//
// page/pageSize 必须先归一化再参与计算：调用方普遍把 ShouldBindQuery / strconv.Atoi 的结果
// 直接传进来，客户端省略或写错分页参数时它们是 0，而 float64 除零得到 +Inf，
// int(+Inf) 在 amd64 上是 int64 最小值——列表接口会把 totalPages: -9223372036854775808
// 当成真实总页数返回给前端（2026-10-02 在 GET /api/v1/tickets 实测命中）。
func NewPaginationResponse(page, pageSize int, total int64) *PaginationResponse {
	page, pageSize = ValidatePagination(page, pageSize)
	if total < 0 {
		total = 0
	}
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
func NewListResponse(items interface{}, pagination *PaginationResponse) *ListResponse {
	return &ListResponse{
		Items:      items,
		Pagination: pagination,
	}
}

// SuccessWithPagination 带分页的成功响应
func SuccessWithPagination(c *gin.Context, items interface{}, page, pageSize int, total int64) {
	pagination := NewPaginationResponse(page, pageSize, total)
	response := NewListResponse(items, pagination)
	Success(c, response)
}

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
