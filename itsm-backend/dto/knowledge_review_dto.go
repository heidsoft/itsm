package dto

import "time"

// SubmitReviewRequest 提交审核请求
type SubmitReviewRequest struct {
	ArticleID int `json:"articleId" binding:"required"`
}

// ApproveArticleRequest 审核通过请求
type ApproveArticleRequest struct {
	ArticleID int    `json:"articleId" binding:"required"`
	Comment   string `json:"comment"`
}

// RejectArticleRequest 审核拒绝请求
type RejectArticleRequest struct {
	ArticleID int    `json:"articleId" binding:"required"`
	Comment   string `json:"comment" binding:"required"`
}

// KnowledgeReviewResponse 审核记录响应
type KnowledgeReviewResponse struct {
	ID           int        `json:"id"`
	ArticleID    int        `json:"articleId"`
	ArticleTitle string     `json:"articleTitle"`
	ReviewerID   int        `json:"reviewerId"`
	ReviewerName string     `json:"reviewerName"`
	Status       string     `json:"status"`
	Comment      string     `json:"comment"`
	ReviewedAt   *time.Time `json:"reviewedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// PendingReviewsResponse 待审核文章列表响应
type PendingReviewsResponse struct {
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
	Items    []*ArticleBasicInfo `json:"items"`
}

// ArticleBasicInfo 文章基本信息
type ArticleBasicInfo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	AuthorID  int       `json:"authorId"`
	CreatedAt time.Time `json:"createdAt"`
}
