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
	TenantID  int       `json:"tenantId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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

// KnowledgeStatsResponse 知识库统计响应
type KnowledgeStatsResponse struct {
	Total      int      `json:"total"`               // 总文章数
	Published  int      `json:"published"`           // 已发布文章数
	Draft      int      `json:"draft"`               // 草稿数
	Views      int64    `json:"views"`               // 总浏览次数
	Rating     float64  `json:"rating"`              // 平均评分 (基于点赞数)
	Categories []CategoryStats `json:"categories"` // 按分类统计
}

// CategoryStats 分类统计
type CategoryStats struct {
	Name  string `json:"name"`  // 分类名称
	Count int    `json:"count"` // 文章数量
}
