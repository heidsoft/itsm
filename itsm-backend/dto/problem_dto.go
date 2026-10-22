package dto

import "time"

// CreateProblemRequest 创建问题请求
type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=200"`
	Description string `json:"description" binding:"required,min=10,max=5000"`
	Priority    string `json:"priority" binding:"required"`
	Category    string `json:"category"`
	RootCause   string `json:"rootCause"`
	Impact      string `json:"impact"`
	ImpactScope string `json:"impactScope"` // 影响范围
}

// UpdateProblemRequest 更新问题请求
type UpdateProblemRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=2,max=200"`
	Description *string `json:"description" binding:"omitempty,min=10,max=5000"`
	Priority    *string `json:"priority" binding:"omitempty"`
	Status      *string `json:"status" binding:"omitempty"`
	Category    *string `json:"category" binding:"omitempty"`
	RootCause   *string `json:"rootCause" binding:"omitempty"`
	Impact      *string `json:"impact" binding:"omitempty"`
}

// UpdateProblemRootCauseRequest 记录问题根因。
type UpdateProblemRootCauseRequest struct {
	RootCause string `json:"rootCause" binding:"required"`
}

// UpdateProblemResolutionRequest 记录问题的临时或最终解决方案。
type UpdateProblemResolutionRequest struct {
	Solution   string `json:"solution"`
	Workaround string `json:"workaround"`
	Resolution string `json:"resolution"`
}

// CloseProblemRequest 关闭问题时可同时记录最终解决方案。
type CloseProblemRequest struct {
	Resolution string `json:"resolution"`
}

// ListProblemsRequest 获取问题列表请求
//
// 只有这里声明的字段会参与查询；分页夹紧由 common.GetPaginationFromQuery 单点负责。
// 这里不声明 sortBy/sortOrder/dateFrom/dateTo：handler 与 repository 都不使用它们
// （传与不传结果一致），声明出来等于假契约。
type ListProblemsRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Status   string `json:"status" form:"status"`
	Priority string `json:"priority" form:"priority"`
	Category string `json:"category" form:"category"`
	Keyword  string `json:"keyword" form:"keyword"`
}

// ProblemResponse 问题响应
type ProblemResponse struct {
	ID            int     `json:"id"`
	ProblemNumber string  `json:"problemNumber,omitempty"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	Category      string  `json:"category"`
	RootCause     string  `json:"rootCause"`
	Workaround    string  `json:"workaround"`
	Resolution    string  `json:"resolution"`
	Impact        string  `json:"impact"`
	AssigneeID    *int    `json:"assigneeId,omitempty"`
	AssigneeName  *string `json:"assigneeName,omitempty"`
	// 创建人/报告人字段（ID + 名字。名字由 service 层 join user 表填充，避免前端再调一次 user API）
	CreatedBy     int       `json:"createdBy"`
	CreatedByName *string   `json:"createdByName,omitempty"`
	TenantID      int       `json:"tenantId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	// 关联数据
	AssociatedTickets   []*AssociatedItemResponse `json:"associatedTickets,omitempty"`
	AssociatedIncidents []*AssociatedItemResponse `json:"associatedIncidents,omitempty"`
	AssociatedChanges   []*AssociatedItemResponse `json:"associatedChanges,omitempty"`
}

// AssociatedItemResponse 关联项响应
type AssociatedItemResponse struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Number string `json:"number,omitempty"`
	Type   string `json:"type,omitempty"`
}

// ProblemAssociationRequest 关联管理请求
type ProblemAssociationRequest struct {
	RelatedType string `json:"relatedType" binding:"required,oneof=ticket incident change"`
	RelatedIDs  []int  `json:"relatedIds" binding:"required,min=1"`
}

// ProblemRemoveAssociationRequest 移除关联请求
type ProblemRemoveAssociationRequest struct {
	RelatedType string `json:"relatedType" binding:"required,oneof=ticket incident change"`
	RelatedID   int    `json:"relatedId" binding:"required"`
}

// ProblemAssociationResponse 关联管理响应
type ProblemAssociationResponse struct {
	Tickets   []*AssociatedItemResponse `json:"tickets"`
	Incidents []*AssociatedItemResponse `json:"incidents"`
	Changes   []*AssociatedItemResponse `json:"changes"`
}

// ListProblemsResponse 问题列表响应
//
// 集合键只能是 items：写成 problems 会让前端同时存在 problems、items?、
// problems?+items? 三套类型声明，以及 resp.problems || resp.items || [] 的 fallback。
type ListProblemsResponse struct {
	Items      []*ProblemResponse `json:"items"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	TotalPages int                `json:"totalPages"`
}

// ProblemStatsResponse 问题统计响应
type ProblemStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	HighPriority int `json:"highPriority"`
	// ByStatus/ByPriority 为租户全量分组计数，只含真实存在的取值，并按词表顺序返回。
	ByStatus   []ProblemStatusCount   `json:"byStatus"`
	ByPriority []ProblemPriorityCount `json:"byPriority"`
}

// ProblemStatusCount 问题状态分布项
type ProblemStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// ProblemPriorityCount 问题优先级分布项
type ProblemPriorityCount struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

// ProblemDetailResponse 问题详情响应
type ProblemDetailResponse struct {
	Problem ProblemResponse `json:"problem"`
}

// CreateKnownErrorFromProblemRequest 从问题创建已知错误请求
type CreateKnownErrorFromProblemRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	Symptoms         *string  `json:"symptoms"`
	RootCause        *string  `json:"rootCause"`
	Workaround       *string  `json:"workaround"`
	Resolution       *string  `json:"resolution"`
	Category         *string  `json:"category"`
	Severity         *string  `json:"severity" binding:"omitempty,oneof=critical high medium low"`
	AffectedProducts []string `json:"affectedProducts"`
	AffectedCIs      []string `json:"affectedCis"`
	Keywords         []string `json:"keywords"`
}

// ─────────────────────────────────────────────────────────────────────────────
// 问题评论（2026-10-21 GA 阻断项落地：GET/POST /api/v1/problems/:id/comments
// 从显式 503 unready 转为真实契约）
// ─────────────────────────────────────────────────────────────────────────────

// CreateProblemCommentRequest 新增问题评论请求。
// isInternal 区分「处理纪要」（团队内部）与「可对报障人可见的进展说明」，
// 与工单评论同一语义；mentions/attachments 为可选的 ID 引用列表。
type CreateProblemCommentRequest struct {
	Content     string `json:"content" binding:"required,min=1,max=5000"`
	IsInternal  bool   `json:"isInternal"`
	Mentions    []int  `json:"mentions"`
	Attachments []int  `json:"attachments"`
}

// ProblemCommentResponse 问题评论响应。
// userName 为服务端投影的显示名（缺省时前端回退 `用户 #id`），
// 与工单姓名单一投影口径一致——不要求前端再拉一次通讯录。
type ProblemCommentResponse struct {
	ID          int       `json:"id"`
	ProblemID   int       `json:"problemId"`
	UserID      int       `json:"userId"`
	UserName    string    `json:"userName,omitempty"`
	Content     string    `json:"content"`
	IsInternal  bool      `json:"isInternal"`
	Mentions    []int     `json:"mentions"`
	Attachments []int     `json:"attachments"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ListProblemCommentsResponse 评论列表信封。
// 键名与工单评论列表一致（items/total/page/size），前端可复用同一渲染组件。
type ListProblemCommentsResponse struct {
	Items []*ProblemCommentResponse `json:"items"`
	Total int                       `json:"total"`
	Page  int                       `json:"page"`
	Size  int                       `json:"size"`
}

// ProblemSLAResponse 问题 SLA 读面（GET /api/v1/problems/:id/sla）。
//
// bound=false 表示该问题没有绑定任何 SLA 定义（租户未配置 problem 类型定义，
// 或优先级无匹配）——这是「没配 SLA」的如实回答，不是「SLA 正常且未超时」；
// 调用方必须把 bound=false 与 status=active 区分开（旧 503 桩的契约理由）。
// status 取值：unbound / active / warning / breached / paused / met。
type ProblemSLAResponse struct {
	ProblemID          int        `json:"problemId"`
	ProblemNumber      string     `json:"problemNumber,omitempty"`
	Priority           string     `json:"priority"`
	Bound              bool       `json:"bound"`
	SLADefinitionID    int        `json:"slaDefinitionId,omitempty"`
	Status             string     `json:"status"`
	ResponseDeadline   *time.Time `json:"responseDeadline,omitempty"`
	ResolutionDeadline *time.Time `json:"resolutionDeadline,omitempty"`
	FirstResponseAt    *time.Time `json:"firstResponseAt,omitempty"`
	ResolvedAt         *time.Time `json:"resolvedAt,omitempty"`
	PausedDurationSec  int        `json:"pausedDurationSeconds,omitempty"`
}

// ── 知识沉淀闭环（2026-10-22）─────────────────────────────────────────────────

// PromoteProblemKnowledgeRequest 问题沉淀为知识草稿的请求体。
// 全部字段可选：空值用问题单内容兜底（标题=问题标题、分类=故障排查、
// 内容=现象/根因/临时措施/解决方案模板化组装）。
type PromoteProblemKnowledgeRequest struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// ListProblemKnowledgeResponse 问题沉淀知识反查列表（GET /api/v1/problems/:id/knowledge）。
// items 为知识域文章 DTO（含 sourceProblemId 回显），含草稿——沉淀当事人
// 需要立刻看到沉淀结果，草稿也要走评审流。
type ListProblemKnowledgeResponse struct {
	Items []*KnowledgeArticleResponse `json:"items"`
	Total int                         `json:"total"`
}
