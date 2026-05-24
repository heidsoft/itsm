package dto

import "time"

// SubmitReviewRequest 提交审核请求
type SubmitReviewRequest struct {
	ArticleID int `json:"article_id" binding:"required"`
}

// ApproveArticleRequest 审核通过请求
type ApproveArticleRequest struct {
	ArticleID int    `json:"article_id" binding:"required"`
	Comment   string `json:"comment"`
}

// RejectArticleRequest 审核拒绝请求
type RejectArticleRequest struct {
	ArticleID int    `json:"article_id" binding:"required"`
	Comment   string `json:"comment" binding:"required"`
}

// KnowledgeReviewResponse 审核记录响应
type KnowledgeReviewResponse struct {
	ID           int        `json:"id"`
	ArticleID    int        `json:"article_id"`
	ArticleTitle string     `json:"article_title"`
	ReviewerID   int        `json:"reviewer_id"`
	ReviewerName string     `json:"reviewer_name"`
	Status       string     `json:"status"`
	Comment      string     `json:"comment"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// PendingReviewsResponse 待审核文章列表响应
type PendingReviewsResponse struct {
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Items    []*ArticleBasicInfo `json:"items"`
}

// ArticleBasicInfo 文章基本信息
type ArticleBasicInfo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	AuthorID  int       `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}
