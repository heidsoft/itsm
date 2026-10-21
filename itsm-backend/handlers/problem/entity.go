package problem

import (
	"time"
)

// Problem domain entity
type Problem struct {
	ID            int
	ProblemNumber string
	Title         string
	Description   string
	Status        string
	Priority      string
	Category      string
	RootCause     string
	Workaround    string
	Resolution    string
	Impact        string
	AssigneeID    *int
	CreatedBy     int
	TenantID      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ResolvedAt    *time.Time
	ClosedAt      *time.Time
	// 关联数据 (eager-loaded)
	Tickets   []*AssociatedItem
	Incidents []*AssociatedItem
	Changes   []*AssociatedItem
}

// AssociatedItem 关联项
type AssociatedItem struct {
	ID     int
	Title  string
	Status string
	Number string
	Type   string
}

// ProblemStats domain entity
type ProblemStats struct {
	Total        int
	Open         int
	InProgress   int
	Resolved     int
	Closed       int
	HighPriority int
	// ByStatus/ByPriority 是租户全量的分组计数，只含真实存在的取值。
	// 折叠后的单值字段无法还原分布（identified 从来不在任何桶里），
	// 报表要分布必须用这两项，禁止前端拉一页列表自己数。
	ByStatus   []StatusCount
	ByPriority []PriorityCount
}

// StatusCount 状态分布项
type StatusCount struct {
	Status string
	Count  int
}

// PriorityCount 优先级分布项
type PriorityCount struct {
	Priority string
	Count    int
}

// CategoryCount represents a category and its count.
type CategoryCount struct {
	Category string
	Count    int
}

// MonthlyCount represents a monthly count for trend analysis.
type MonthlyCount struct {
	Month    string
	Count    int
	Resolved int
	Open     int
}

// ProblemTrendData holds trend analytics data.
type ProblemTrendData struct {
	Period                 string
	TotalProblems          int
	ResolvedProblems       int
	OpenProblems           int
	ResolutionRate         float64
	AvgResolutionTimeHours float64
	CategoryBreakdown      map[string]int
	PriorityBreakdown      map[string]int
	TrendDirection         string
	TopCategories          []CategoryCount
	MonthlyTrend           []MonthlyCount
}

// ProblemHotspotData holds hotspot analytics data.
type ProblemHotspotData struct {
	PeriodStart       string
	PeriodEnd         string
	CategoryBreakdown map[string]int
	PriorityBreakdown map[string]int
	Hotspots          []string
	AvgPerCategory    float64
}

// ProblemComment 问题评论/处理纪要（2026-10-21 GA 阻断项落地）。
// 与工单评论同构但独立成表：问题评论的读者是处理团队，可见性规则与
// 含客户沟通的工单评论不同（详见 ent/schema/problem_comment.go 头注）。
type ProblemComment struct {
	ID         int
	ProblemID  int
	UserID     int
	Content    string
	IsInternal bool
	Mentions   []int
	Attachments []int
	TenantID   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// 展示字段（非持久化）：列表查询时批量投影，缺省时前端回退 `用户 #id`
	UserName string
}

// ProblemSLAInfo 问题 SLA 状态读面（GET /api/v1/problems/:id/sla）。
//
// 与工单的 TicketSLAInfo 同口径：截止时间与状态以 sla_states 表为唯一真相
// （aggregate_type="problem"）。Bound=false 表示该问题没有绑定任何 SLA 定义
// （租户未配置 problem 类型定义或优先级无匹配）——这是「没配 SLA」的如实回答，
// 不是「SLA 正常」；调用方不得把 Bound=false 读成未超时。
type ProblemSLAInfo struct {
	ProblemID         int
	ProblemNumber     string
	Priority          string
	Bound             bool
	SLADefinitionID   int
	Status            string // pending/active/paused/breached/met
	ResponseDeadline  *time.Time
	ResolutionDeadline *time.Time
	FirstResponseAt   *time.Time
	ResolvedAt        *time.Time
	PausedDurationSec int
}
