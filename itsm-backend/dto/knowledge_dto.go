package dto

import "time"

// 创建知识库文章请求
type CreateKnowledgeArticleRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content"`
	Category string   `json:"category" binding:"required"`
	Tags     []string `json:"tags"`
}

// 更新知识库文章请求
type UpdateKnowledgeArticleRequest struct {
	Title    *string  `json:"title"`
	Content  *string  `json:"content"`
	Category *string  `json:"category"`
	Status   *string  `json:"status"`
	Tags     []string `json:"tags"`
}

// 知识库文章响应
type KnowledgeArticleResponse struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	Author    string    `json:"author"`
	Views     int       `json:"views"`
	Tags      []string  `json:"tags"`
	TenantID  int       `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 知识库文章列表请求
type ListKnowledgeArticlesRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Category string `form:"category"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// 知识库文章列表响应
type KnowledgeArticleListResponse struct {
	Articles []KnowledgeArticleResponse `json:"articles"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	Size     int                        `json:"size"`
}
