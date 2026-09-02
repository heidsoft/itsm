package common

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationRequest 分页请求参数
type PaginationRequest struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"pageSize" form:"page_size" binding:"min=1,max=100"`
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

// ListResponse 列表响应结构
//
// 标准契约使用扁平字段 (items/total/page/pageSize/totalPages)。同时
// MarshalJSON 自动追加领域名别名（response.tickets/response.incidents/...），
// 让仍依赖旧字段名的前端代码继续工作。
type ListResponse struct {
	Items      interface{}         `json:"items"`
	Pagination *PaginationResponse `json:"pagination,omitempty"`
}

// MarshalJSON 在标准契约之上输出领域别名，让前端无需修改即可继续工作。
func (l *ListResponse) MarshalJSON() ([]byte, error) {
	flat := map[string]interface{}{}
	if l.Items != nil {
		flat["items"] = l.Items
	}
	if l.Pagination != nil {
		flat["total"] = l.Pagination.Total
		flat["page"] = l.Pagination.Page
		flat["pageSize"] = l.Pagination.PageSize
		flat["totalPages"] = l.Pagination.TotalPages
		flat["pagination"] = l.Pagination
	}

	if alias := inferDomainAlias(l.Items); alias != "" {
		flat[alias] = l.Items
	}

	return json.Marshal(flat)
}

// domainAliases 与前端代码里 response.<name> 使用的领域名保持一致。
var domainAliases = []string{
	"tickets",
	"incidents",
	"changes",
	"problems",
	"services",
	"assets",
	"applications",
	"projects",
	"releases",
	"categories",
	"types",
	"records",
	"list",
	"templates",
	"items_list",
}

// inferDomainAlias 根据 reflect 看到的 slice 元素类型名映射到领域别名。
// 优先按业务约定名（domainAliases / singularAliases）查找，未命中时回退到
// 类型名 lowerFirst()。避免出现类似 element=TicketResponse → alias=ticketResponse
// 但前端期望 tickets 的不一致。
func inferDomainAlias(items interface{}) string {
	if items == nil {
		return ""
	}
	t := reflect.TypeOf(items)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return ""
	}
	elem := t.Elem()
	for elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		return ""
	}
	name := elem.Name()
	if name == "" {
		return ""
	}
	lower := lowerFirst(name)
	for _, alias := range domainAliases {
		if alias == lower {
			return alias
		}
	}
	// 业务约定：TicketResponse / IncidentResponse 等领域响应类型映射到复数别名
	for singular, plural := range singularAliases {
		if singular == lower {
			return plural
		}
	}
	return lower
}

// singularAliases 显式映射：领域响应类型（单数）→ 列表字段名（复数）。
// 这是 AGENTS.md 中规定的 API 契约，不允许因为 element 类型名差异而漏掉别名。
var singularAliases = map[string]string{
	"ticketResponse":         "tickets",
	"incidentResponse":       "incidents",
	"changeResponse":         "changes",
	"problemResponse":        "problems",
	"serviceResponse":        "services",
	"assetResponse":          "assets",
	"applicationResponse":    "applications",
	"projectResponse":        "projects",
	"releaseResponse":        "releases",
	"categoryResponse":       "categories",
	"typeDefinition":         "types",
	"ticketTypeDefinition":   "types",
	"knowledgeArticleResp":   "articles",
	"knowledgeArticleRespon": "articles",
	"knowledgeArticle":       "articles",
	"workflowResponse":       "workflows",
	"taskResponse":           "tasks",
	"userResponse":           "users",
	"roleResponse":           "roles",
	"groupResponse":          "groups",
	"departmentResponse":     "departments",
	"approvalResponse":       "approvals",
	"notificationResponse":   "notifications",
	"surveyResponse":         "surveys",
	"slaResponse":            "slas",
	"commentResponse":        "comments",
	"attachmentResponse":     "attachments",
	"auditLogResponse":       "auditLogs",
	"connectorResponse":      "connectors",
	"vendorResponse":         "vendors",
	"contractResponse":       "contracts",
	"ciResponse":             "cis",
	"relationResponse":       "relations",
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'A' && s[0] <= 'Z' {
		b := []byte(s)
		b[0] = b[0] + ('a' - 'A')
		return string(b)
	}
	return s
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
