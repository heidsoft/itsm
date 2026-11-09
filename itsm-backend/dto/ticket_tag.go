package dto

import "errors"

// 标签域的失败分类。重复名、在用删除若以普通 error 上抛，handler 只能一律
// 回 500/5001「操作失败」，调用方无法区分「名字重了」「标签还在用」和后端故障。
var (
	// ErrTicketTagNameExists 表示同租户内已有同名标签。ticket_tags 没有 (tenant_id, name)
	// 唯一索引，重名只靠查询前置检查，所以归为 409 冲突而不是数据库故障。
	ErrTicketTagNameExists = errors.New("ticket tag name already exists")
	// ErrTicketTagInUse 表示标签仍被本租户工单引用，删除会被拒绝。
	ErrTicketTagInUse = errors.New("ticket tag is in use")
	// ErrTicketTagNameBlank 表示名称只含空白，trim 后为空。
	ErrTicketTagNameBlank = errors.New("ticket tag name is blank")
	// ErrTicketTagNotFound 表示绑定/解绑引用了本租户不存在的标签。
	ErrTicketTagNotFound = errors.New("ticket tag not found")
)

// CreateTagRequest 创建标签请求。指针字段与 json tag 共同决定 Gin 绑定语义，
// 下沉到 dto 时必须原样保留，否则前端表单行为会漂移。
type CreateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
	// 指针：未传 = 启用；显式 false = 停用。非指针会让 omit 与 false 同义。
	IsActive *bool `json:"isActive"`
	// 租户只能来自认证上下文，禁止客户端自报；binding:"required" 会让不带
	// tenantId 的合法请求直接 400，页面上「新建标签」点了就失败。
	TenantID int `json:"-"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	IsActive    *bool  `json:"isActive"`
}

// ListTagsRequest 获取标签列表请求
type ListTagsRequest struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"pageSize" form:"pageSize"`
	IsActive *bool `json:"isActive" form:"isActive"`
	TenantID int   `json:"tenantId" form:"tenantId"`
}
