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

// ChangePIRListResponse 变更PIR列表响应
type ChangePIRListResponse struct {
	Total int                  `json:"total"`
	Items []*ChangePIRResponse `json:"items"`
}
