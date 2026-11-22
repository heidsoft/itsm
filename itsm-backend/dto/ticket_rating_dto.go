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
//
// 日期字段必须带 binding：此前只有 `form` 标签，非法日期会一路传到 service，
// 被 `time.Parse` 静默丢弃后查询退化成「全历史评分」——报表使用者以为筛了
// 日期，看到的却是所有历史数据。绑定 `datetime=2006-01-02` 与前端日期选择器
// 的提交格式一致；`omitempty` 保留「不传日期 = 查全部」这个正当语义。
// 注意字段是 *string：不传时为 nil，传空串时 binding 会按 `datetime` 校验失败，
// 因此带 `?startDate=` 的空值也会被拒（前端不应提交空串）。
type GetRatingStatsRequest struct {
	AssigneeID *int    `form:"assigneeId"`                                        // 按处理人筛选
	CategoryID *int    `form:"categoryId"`                                        // 按分类筛选
	StartDate  *string `form:"startDate" binding:"omitempty,datetime=2006-01-02"` // 开始日期
	EndDate    *string `form:"endDate" binding:"omitempty,datetime=2006-01-02"`   // 结束日期
	TenantID   int     `form:"tenantId"`
}
