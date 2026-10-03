package dto

import "time"

// CreateChangePIRRequest 创建变更PIR请求
type CreateChangePIRRequest struct {
	ChangeID                   int        `json:"changeId"`
	OverallResult              string     `json:"overallResult" binding:"required,oneof=successful partially_successful failed"`
	ObjectivesAchieved         bool       `json:"objectivesAchieved"`
	SuccessSummary             *string    `json:"successSummary"`
	IssuesEncountered          *string    `json:"issuesEncountered"`
	LessonsLearned             *string    `json:"lessonsLearned"`
	ImprovementRecommendations *string    `json:"improvementRecommendations"`
	ActualStartTime            *time.Time `json:"actualStartTime"`
	ActualEndTime              *time.Time `json:"actualEndTime"`
	RollbackPerformed          bool       `json:"rollbackPerformed"`
	RollbackReason             *string    `json:"rollbackReason"`
}

// UpdateChangePIRRequest 更新变更PIR请求
type UpdateChangePIRRequest struct {
	OverallResult              *string `json:"overallResult" binding:"omitempty,oneof=successful partially_successful failed"`
	ObjectivesAchieved         *bool   `json:"objectivesAchieved"`
	SuccessSummary             *string `json:"successSummary"`
	IssuesEncountered          *string `json:"issuesEncountered"`
	LessonsLearned             *string `json:"lessonsLearned"`
	ImprovementRecommendations *string `json:"improvementRecommendations"`
}

// ChangePIRResponse 变更PIR响应
type ChangePIRResponse struct {
	ID                         int        `json:"id"`
	ChangeID                   int        `json:"changeId"`
	ChangeTitle                string     `json:"changeTitle"`
	ReviewerID                 int        `json:"reviewerId"`
	ReviewerName               string     `json:"reviewerName"`
	OverallResult              string     `json:"overallResult"`
	ObjectivesAchieved         bool       `json:"objectivesAchieved"`
	SuccessSummary             *string    `json:"successSummary"`
	IssuesEncountered          *string    `json:"issuesEncountered"`
	LessonsLearned             *string    `json:"lessonsLearned"`
	ImprovementRecommendations *string    `json:"improvementRecommendations"`
	ActualStartTime            *time.Time `json:"actualStartTime"`
	ActualEndTime              *time.Time `json:"actualEndTime"`
	ActualDurationMinutes      int        `json:"actualDurationMinutes"`
	RollbackPerformed          bool       `json:"rollbackPerformed"`
	RollbackReason             *string    `json:"rollbackReason"`
	TenantID                   int        `json:"tenantId"`
	ReviewDate                 time.Time  `json:"reviewDate"`
	CreatedAt                  time.Time  `json:"createdAt"`
	UpdatedAt                  time.Time  `json:"updatedAt"`
}

// ChangePIRListResponse 变更PIR列表响应。
//
// 这个端点是真的分页（service/pir_service.go 用 Count + Offset/Limit），所以必须把
// 「第几页、每页多少、共几页」如实交出去：只带 {items,total} 会让调用方把当前页
// 当成整个结果集。五个键由 common.NewPaginationResponse 统一算出，避免除零溢出。
type ChangePIRListResponse struct {
	Items      []*ChangePIRResponse `json:"items"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}
