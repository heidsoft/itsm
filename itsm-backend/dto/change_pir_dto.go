package dto

import "time"

// CreateChangePIRRequest 创建变更PIR请求
type CreateChangePIRRequest struct {
	ChangeID                    int        `json:"change_id" binding:"required"`
	OverallResult               string     `json:"overall_result" binding:"required,oneof=successful partially_successful failed"`
	ObjectivesAchieved          bool       `json:"objectives_achieved"`
	SuccessSummary              *string    `json:"success_summary"`
	IssuesEncountered           *string    `json:"issues_encountered"`
	LessonsLearned              *string    `json:"lessons_learned"`
	ImprovementRecommendations  *string    `json:"improvement_recommendations"`
	ActualStartTime             *time.Time `json:"actual_start_time"`
	ActualEndTime               *time.Time `json:"actual_end_time"`
	RollbackPerformed           bool       `json:"rollback_performed"`
	RollbackReason              *string    `json:"rollback_reason"`
}

// UpdateChangePIRRequest 更新变更PIR请求
type UpdateChangePIRRequest struct {
	OverallResult               *string    `json:"overall_result" binding:"omitempty,oneof=successful partially_successful failed"`
	ObjectivesAchieved          *bool      `json:"objectives_achieved"`
	SuccessSummary              *string    `json:"success_summary"`
	IssuesEncountered           *string    `json:"issues_encountered"`
	LessonsLearned              *string    `json:"lessons_learned"`
	ImprovementRecommendations  *string    `json:"improvement_recommendations"`
}

// ChangePIRResponse 变更PIR响应
type ChangePIRResponse struct {
	ID                          int        `json:"id"`
	ChangeID                    int        `json:"change_id"`
	ChangeTitle                 string     `json:"change_title"`
	ReviewerID                  int        `json:"reviewer_id"`
	ReviewerName                string     `json:"reviewer_name"`
	OverallResult               string     `json:"overall_result"`
	ObjectivesAchieved          bool       `json:"objectives_achieved"`
	SuccessSummary              *string    `json:"success_summary"`
	IssuesEncountered           *string    `json:"issues_encountered"`
	LessonsLearned              *string    `json:"lessons_learned"`
	ImprovementRecommendations  *string    `json:"improvement_recommendations"`
	ActualStartTime             *time.Time `json:"actual_start_time"`
	ActualEndTime               *time.Time `json:"actual_end_time"`
	ActualDurationMinutes       int        `json:"actual_duration_minutes"`
	RollbackPerformed           bool       `json:"rollback_performed"`
	RollbackReason              *string    `json:"rollback_reason"`
	TenantID                    int        `json:"tenant_id"`
	ReviewDate                  time.Time  `json:"review_date"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

// ChangePIRListResponse 变更PIR列表响应
type ChangePIRListResponse struct {
	Total int                   `json:"total"`
	Items []*ChangePIRResponse `json:"items"`
}
