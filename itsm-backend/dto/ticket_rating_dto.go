package dto

import "time"

// SubmitTicketRatingRequest 提交工单评分请求
type SubmitTicketRatingRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"` // 1-5星评分
	Comment string `json:"comment,omitempty"`                      // 评分评论（可选）
}

// TicketRatingResponse 工单评分响应
type TicketRatingResponse struct {
	Rating      int        `json:"rating"`       // 1-5星评分
	Comment     string     `json:"comment"`      // 评分评论
	RatedAt     *time.Time `json:"rated_at"`    // 评分时间
	RatedBy     int        `json:"rated_by"`    // 评分人ID
	RatedByName string     `json:"rated_by_name,omitempty"` // 评分人姓名
}

// RatingStatsResponse 评分统计响应
type RatingStatsResponse struct {
	TotalRatings    int                        `json:"total_ratings"`     // 总评分数量
	AverageRating   float64                   `json:"average_rating"`    // 平均评分
	RatingDistribution map[int]int             `json:"rating_distribution"` // 评分分布 {1: 10, 2: 5, 3: 8, 4: 20, 5: 30}
	ByAssignee      map[int]*AssigneeRatingStats `json:"by_assignee,omitempty"` // 按处理人统计
	ByCategory      map[int]*CategoryRatingStats  `json:"by_category,omitempty"` // 按分类统计
}

// AssigneeRatingStats 处理人评分统计
type AssigneeRatingStats struct {
	AssigneeID    int     `json:"assignee_id"`
	AssigneeName string  `json:"assignee_name"`
	TotalRatings  int     `json:"total_ratings"`
	AverageRating float64 `json:"average_rating"`
}

// CategoryRatingStats 分类评分统计
type CategoryRatingStats struct {
	CategoryID    int     `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	TotalRatings  int     `json:"total_ratings"`
	AverageRating float64 `json:"average_rating"`
}

// GetRatingStatsRequest 获取评分统计请求
type GetRatingStatsRequest struct {
	AssigneeID *int   `form:"assignee_id"` // 按处理人筛选
	CategoryID *int   `form:"category_id"` // 按分类筛选
	StartDate  *string `form:"start_date"` // 开始日期
	EndDate    *string `form:"end_date"`   // 结束日期
	TenantID   int     `form:"tenant_id" binding:"required"`
}

