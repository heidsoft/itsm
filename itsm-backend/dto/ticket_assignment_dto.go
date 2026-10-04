package dto

// AutoAssignRequest 自动分配请求
type AutoAssignRequest struct {
	TicketID int `json:"ticketId" binding:"required"`
}

// AutoAssignResponse 自动分配响应
type AutoAssignResponse struct {
	TicketID       int     `json:"ticketId"`
	AssignedTo     *int    `json:"assignedTo,omitempty"`
	AssignmentType string  `json:"assignmentType"` // auto, rule, manual
	Reason         string  `json:"reason"`
	Score          float64 `json:"score,omitempty"`
}

// AssignmentRecommendation 分配推荐
type AssignmentRecommendation struct {
	UserID     int      `json:"userId"`
	Username   string   `json:"username"`
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Score      float64  `json:"score"`
	Reason     string   `json:"reason"`
	Workload   int      `json:"workload"` // 当前工作负载
	Skills     []string `json:"skills,omitempty"`
	Categories []int    `json:"categories,omitempty"`
}

// AssignRecommendationListResponse 分配推荐列表响应
//
// GET /api/v1/tickets/assign-recommendations/:id 实测不分页：service 将租户内全部可用
// 用户排序后整表返回，handler 写 Total=len(items)，因此按 docs/api-reference.md
// 「不分页的列表」保留诚实的两键，不补 page/pageSize/totalPages。
// 结构体名带 List 是为了让它落进 tests/contract 的信封棘轮扫描范围（棘轮只扫名字含
// List 且带 total 的结构体，旧名 GetAssignRecommendationsResponse 因此静默逃逸过守卫）。
type AssignRecommendationListResponse struct {
	Items []*AssignmentRecommendation `json:"items"`
	Total int                         `json:"total"`
}

// AssignmentRuleResponse 分配规则响应
type AssignmentRuleResponse struct {
	ID             int                      `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Priority       int                      `json:"priority"`
	Conditions     []map[string]interface{} `json:"conditions"`
	Actions        map[string]interface{}   `json:"actions"`
	IsActive       bool                     `json:"isActive"`
	ExecutionCount int                      `json:"executionCount"`
	LastExecutedAt *string                  `json:"lastExecutedAt,omitempty"`
	CreatedAt      string                   `json:"createdAt"`
	UpdatedAt      string                   `json:"updatedAt"`
}

// CreateAssignmentRuleRequest 创建分配规则请求
type CreateAssignmentRuleRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Priority    int                      `json:"priority"`
	Conditions  []map[string]interface{} `json:"conditions" binding:"required"`
	Actions     map[string]interface{}   `json:"actions" binding:"required"`
	IsActive    bool                     `json:"isActive"`
}

// UpdateAssignmentRuleRequest 更新分配规则请求
type UpdateAssignmentRuleRequest struct {
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Priority    *int                     `json:"priority,omitempty"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
	Actions     map[string]interface{}   `json:"actions,omitempty"`
	IsActive    *bool                    `json:"isActive,omitempty"`
}

// ListAssignmentRulesResponse 分配规则列表响应
//
// 该端点实测不分页（handler 用 Total: len(rules) 且 service 整表取回），
// 因此只保留诚实的 {items,total} 两键，不补假的 page/pageSize/totalPages。
type ListAssignmentRulesResponse struct {
	Items []*AssignmentRuleResponse `json:"items"`
	Total int                       `json:"total"`
}

// TestAssignmentRuleRequest 测试分配规则请求
type TestAssignmentRuleRequest struct {
	TicketID int `json:"ticketId" binding:"required"`
	RuleID   int `json:"ruleId" binding:"required"`
}

// TestAssignmentRuleResponse 测试分配规则响应
type TestAssignmentRuleResponse struct {
	Matched    bool    `json:"matched"`
	AssignedTo *int    `json:"assignedTo,omitempty"`
	Reason     string  `json:"reason"`
	Score      float64 `json:"score,omitempty"`
}
