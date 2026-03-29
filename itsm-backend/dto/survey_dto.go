package dto

import "time"

// Question represents a survey question
type Question struct {
	Question string   `json:"question"`
	Type     string   `json:"type"` // rating, text, choice
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
}

// SurveyResponse represents a survey response
type SurveyDTO struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SurveyType  string     `json:"surveyType"`
	IsActive    bool       `json:"isActive"`
	Questions   []Question `json:"questions"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	TenantID    int        `json:"tenantId"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// SubmitSurveyRequest represents a request to submit a survey response
type SubmitSurveyRequest struct {
	SurveyID int        `json:"surveyId" binding:"required"`
	TicketID int        `json:"ticketId" binding:"required"`
	Answers  []AnswerDTO `json:"answers" binding:"required"`
	Score    int        `json:"score"`
	Comment  string     `json:"comment"`
}

// AnswerDTO represents an answer to a survey question
type AnswerDTO struct {
	QuestionIndex int         `json:"questionIndex"`
	Value         interface{} `json:"value"`
}

// SurveyAnalytics represents survey analytics data
type SurveyAnalytics struct {
	SurveyID      int     `json:"surveyId"`
	ResponseCount int     `json:"responseCount"`
	AverageScore  float64 `json:"averageScore"`
	NpsScore      float64 `json:"npsScore"`
	CsatScore     float64 `json:"csatScore"`
	ResponseRate  float64 `json:"responseRate"`
}

// SurveyListResponse represents a list of surveys
type SurveyListResponse struct {
	Surveys  []*SurveyDTO `json:"surveys"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// CreateSurveyRequest represents a request to create a survey
type CreateSurveyRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	SurveyType  string     `json:"surveyType" binding:"required,oneof=NPS CSAT CES"`
	Questions   []Question `json:"questions" binding:"required"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	IsActive    bool       `json:"isActive"`
}

// UpdateSurveyRequest represents a request to update a survey
type UpdateSurveyRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	IsActive    *bool      `json:"isActive"`
}