package dto

import (
	"time"

	"itsm-backend/ent"
)

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
	SurveyID int         `json:"surveyId" binding:"required"`
	TicketID int         `json:"ticketId" binding:"required"`
	Answers  []AnswerDTO `json:"answers" binding:"required"`
	Score    int         `json:"score"`
	Comment  string      `json:"comment"`
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

// ===================================
// Survey Mappers（B10.1：handler 禁止直返 ent 模型，snake_case 泄漏收敛）
// ===================================

// ToSurveyDTO converts an ent.Survey to SurveyDTO (camelCase contract).
func ToSurveyDTO(s *ent.Survey) *SurveyDTO {
	if s == nil {
		return nil
	}
	out := &SurveyDTO{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		SurveyType:  s.SurveyType,
		IsActive:    s.IsActive,
		TenantID:    s.TenantID,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
	out.Questions = make([]Question, 0, len(s.Questions))
	for _, q := range s.Questions {
		out.Questions = append(out.Questions, Question{
			Question: q.Question,
			Type:     q.Type,
			Options:  q.Options,
			Required: q.Required,
		})
	}
	if !s.StartDate.IsZero() {
		out.StartDate = &s.StartDate
	}
	if !s.EndDate.IsZero() {
		out.EndDate = &s.EndDate
	}
	return out
}

// ToSurveyDTOList converts a slice of ent.Survey to SurveyDTO slice.
func ToSurveyDTOList(surveys []*ent.Survey) []*SurveyDTO {
	responses := make([]*SurveyDTO, 0, len(surveys))
	for _, s := range surveys {
		responses = append(responses, ToSurveyDTO(s))
	}
	return responses
}

// SurveyAnswerDTO represents a single answer in a survey response (camelCase).
type SurveyAnswerDTO struct {
	QuestionIndex int         `json:"questionIndex"`
	Value         interface{} `json:"value"`
}

// SurveyResponseItemDTO represents a submitted survey response (camelCase).
type SurveyResponseItemDTO struct {
	ID           int               `json:"id"`
	SurveyID     int               `json:"surveyId"`
	TicketID     int               `json:"ticketId"`
	RespondentID int               `json:"respondentId"`
	Score        int               `json:"score"`
	Comment      string            `json:"comment"`
	Answers      []SurveyAnswerDTO `json:"answers"`
	SubmittedAt  time.Time         `json:"submittedAt"`
}

// ToSurveyResponseItemDTO converts an ent.SurveyResponse to SurveyResponseItemDTO.
func ToSurveyResponseItemDTO(r *ent.SurveyResponse) *SurveyResponseItemDTO {
	if r == nil {
		return nil
	}
	answers := make([]SurveyAnswerDTO, 0, len(r.Answers))
	for _, a := range r.Answers {
		answers = append(answers, SurveyAnswerDTO{
			QuestionIndex: a.QuestionIndex,
			Value:         a.Value,
		})
	}
	return &SurveyResponseItemDTO{
		ID:           r.ID,
		SurveyID:     r.SurveyID,
		TicketID:     r.TicketID,
		RespondentID: r.RespondentID,
		Score:        r.Score,
		Comment:      r.Comment,
		Answers:      answers,
		SubmittedAt:  r.SubmittedAt,
	}
}

// ToSurveyResponseItemDTOList converts a slice of ent.SurveyResponse.
func ToSurveyResponseItemDTOList(rows []*ent.SurveyResponse) []*SurveyResponseItemDTO {
	out := make([]*SurveyResponseItemDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToSurveyResponseItemDTO(r))
	}
	return out
}
