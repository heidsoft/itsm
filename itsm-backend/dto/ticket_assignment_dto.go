package dto

// AutoAssignRequest 自动分配请求
type AutoAssignRequest struct {
	TicketID int `json:"ticket_id" binding:"required"`
}

// AutoAssignResponse 自动分配响应
type AutoAssignResponse struct {
	TicketID       int    `json:"ticket_id"`
	AssignedTo     *int   `json:"assigned_to,omitempty"`
	AssignmentType string `json:"assignment_type"` // auto, rule, manual
	Reason         string `json:"reason"`
	Score          float64 `json:"score,omitempty"`
}

// AssignmentRecommendation 分配推荐
type AssignmentRecommendation struct {
	UserID    int     `json:"user_id"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
	Workload  int     `json:"workload"` // 当前工作负载
	Skills     []string `json:"skills,omitempty"`
	Categories []int   `json:"categories,omitempty"`
}

// GetAssignRecommendationsResponse 获取分配推荐响应
type GetAssignRecommendationsResponse struct {
	Recommendations []*AssignmentRecommendation `json:"recommendations"`
	Total           int                         `json:"total"`
}

// AssignmentRuleResponse 分配规则响应
type AssignmentRuleResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Priority        int                    `json:"priority"`
	Conditions      []map[string]interface{} `json:"conditions"`
	Actions         map[string]interface{} `json:"actions"`
	IsActive        bool                   `json:"is_active"`
	ExecutionCount  int                    `json:"execution_count"`
	LastExecutedAt  *string                `json:"last_executed_at,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// CreateAssignmentRuleRequest 创建分配规则请求
type CreateAssignmentRuleRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Conditions  []map[string]interface{} `json:"conditions" binding:"required"`
	Actions     map[string]interface{} `json:"actions" binding:"required"`
	IsActive    bool                   `json:"is_active"`
}

// UpdateAssignmentRuleRequest 更新分配规则请求
type UpdateAssignmentRuleRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Priority    *int                   `json:"priority,omitempty"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
	Actions     map[string]interface{} `json:"actions,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

// ListAssignmentRulesResponse 分配规则列表响应
type ListAssignmentRulesResponse struct {
	Rules []*AssignmentRuleResponse `json:"rules"`
	Total int                       `json:"total"`
}

// TestAssignmentRuleRequest 测试分配规则请求
type TestAssignmentRuleRequest struct {
	TicketID int `json:"ticket_id" binding:"required"`
	RuleID   int `json:"rule_id" binding:"required"`
}

// TestAssignmentRuleResponse 测试分配规则响应
type TestAssignmentRuleResponse struct {
	Matched     bool   `json:"matched"`
	AssignedTo  *int   `json:"assigned_to,omitempty"`
	Reason      string `json:"reason"`
	Score       float64 `json:"score,omitempty"`
}


