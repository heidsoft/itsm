package dto

import "time"

// SubmitTicketRatingRequest 提交工单评分请求
type SubmitTicketRatingRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"` // 1-5星评分
	Comment string `json:"comment,omitempty"`                     // 评分评论（可选）
}

// TicketRatingResponse 工单评分响应
type TicketRatingResponse struct {
	Rating      int        `json:"rating"`                // 1-5星评分
	Comment     string     `json:"comment"`               // 评分评论
	RatedAt     *time.Time `json:"ratedAt"`               // 评分时间
	RatedBy     int        `json:"ratedBy"`               // 评分人ID
	RatedByName string     `json:"ratedByName,omitempty"` // 评分人姓名
}

// RatingStatsResponse 评分统计响应
type RatingStatsResponse struct {
	TotalRatings       int                          `json:"totalRatings"`         // 总评分数量
	AverageRating      float64                      `json:"averageRating"`        // 平均评分
	RatingDistribution map[int]int                  `json:"ratingDistribution"`   // 评分分布 {1: 10, 2: 5, 3: 8, 4: 20, 5: 30}
	ByAssignee         map[int]*AssigneeRatingStats `json:"byAssignee,omitempty"` // 按处理人统计
	ByCategory         map[int]*CategoryRatingStats `json:"byCategory,omitempty"` // 按分类统计
}

// AssigneeRatingStats 处理人评分统计
type AssigneeRatingStats struct {
	AssigneeID    int     `json:"assigneeId"`
	AssigneeName  string  `json:"assigneeName"`
	TotalRatings  int     `json:"totalRatings"`
	AverageRating float64 `json:"averageRating"`
}

// CategoryRatingStats 分类评分统计
type CategoryRatingStats struct {
	CategoryID    int     `json:"categoryId"`
	CategoryName  string  `json:"categoryName"`
	TotalRatings  int     `json:"totalRatings"`
	AverageRating float64 `json:"averageRating"`
}

// GetRatingStatsRequest 获取评分统计请求
type GetRatingStatsRequest struct {
	AssigneeID *int    `form:"assignee_id"` // 按处理人筛选
	CategoryID *int    `form:"category_id"` // 按分类筛选
	StartDate  *string `form:"start_date"`  // 开始日期
	EndDate    *string `form:"end_date"`    // 结束日期
	TenantID   int     `form:"tenant_id" binding:"required"`
}
