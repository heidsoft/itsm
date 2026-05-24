package service

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/survey"
	"itsm-backend/ent/surveyresponse"

	"go.uber.org/zap"
)

// SurveyService handles survey business logic
type SurveyService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewSurveyService creates a new SurveyService
func NewSurveyService(client *ent.Client, logger *zap.SugaredLogger) *SurveyService {
	return &SurveyService{client: client, logger: logger}
}

// SubmitResponse submits a survey response
func (s *SurveyService) SubmitResponse(ctx context.Context, req *dto.SubmitSurveyRequest, tenantID int) error {
	_, err := s.client.SurveyResponse.Create().
		SetSurveyID(req.SurveyID).
		SetTicketID(req.TicketID).
		SetRespondentID(0). // Would come from auth context
		SetAnswers(toSchemaAnswers(req.Answers)).
		SetScore(req.Score).
		SetComment(req.Comment).
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

// GetAnalytics retrieves analytics for a survey
func (s *SurveyService) GetAnalytics(ctx context.Context, surveyID int, tenantID int) (*dto.SurveyAnalytics, error) {
	responses, err := s.client.SurveyResponse.Query().
		Where(surveyresponse.HasSurveyWith(survey.ID(surveyID))).
		Where(surveyresponse.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	if len(responses) == 0 {
		return &dto.SurveyAnalytics{SurveyID: surveyID}, nil
	}

	totalScore := 0
	for _, r := range responses {
		totalScore += r.Score
	}

	return &dto.SurveyAnalytics{
		SurveyID:      surveyID,
		ResponseCount: len(responses),
		AverageScore:  float64(totalScore) / float64(len(responses)),
		NpsScore:      calculateNPS(responses),
		CsatScore:     float64(totalScore) / float64(len(responses)) * 20, // Convert to percentage
	}, nil
}

// GetSurveys retrieves surveys for a tenant
func (s *SurveyService) GetSurveys(ctx context.Context, tenantID int) ([]*ent.Survey, error) {
	return s.client.Survey.Query().
		Where(survey.TenantID(tenantID)).
		All(ctx)
}

// GetSurvey retrieves a single survey by ID
func (s *SurveyService) GetSurvey(ctx context.Context, surveyID int, tenantID int) (*ent.Survey, error) {
	return s.client.Survey.Query().
		Where(survey.ID(surveyID), survey.TenantID(tenantID)).
		Only(ctx)
}

// CreateSurvey creates a new survey
func (s *SurveyService) CreateSurvey(ctx context.Context, req *dto.CreateSurveyRequest, tenantID int) (*ent.Survey, error) {
	return s.client.Survey.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetSurveyType(req.SurveyType).
		SetQuestions(toSchemaQuestions(req.Questions)).
		SetIsActive(req.IsActive).
		SetNillableStartDate(req.StartDate).
		SetNillableEndDate(req.EndDate).
		SetTenantID(tenantID).
		Save(ctx)
}

// UpdateSurvey updates an existing survey
func (s *SurveyService) UpdateSurvey(ctx context.Context, surveyID int, req *dto.UpdateSurveyRequest, tenantID int) (*ent.Survey, error) {
	survey, err := s.GetSurvey(ctx, surveyID, tenantID)
	if err != nil {
		return nil, err
	}

	update := survey.Update()
	if req.Title != "" {
		update = update.SetTitle(req.Title)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}
	if len(req.Questions) > 0 {
		update = update.SetQuestions(toSchemaQuestions(req.Questions))
	}
	if req.StartDate != nil {
		update = update.SetStartDate(*req.StartDate)
	}
	if req.EndDate != nil {
		update = update.SetEndDate(*req.EndDate)
	}
	if req.IsActive != nil {
		update = update.SetIsActive(*req.IsActive)
	}

	return update.Save(ctx)
}

// GetSurveyResponses retrieves responses for a survey
func (s *SurveyService) GetSurveyResponses(ctx context.Context, surveyID int, tenantID int) ([]*ent.SurveyResponse, error) {
	return s.client.SurveyResponse.Query().
		Where(surveyresponse.HasSurveyWith(survey.ID(surveyID))).
		Where(surveyresponse.TenantID(tenantID)).
		All(ctx)
}

// toSchemaAnswers converts DTO answers to schema.Answer
func toSchemaAnswers(answers []dto.AnswerDTO) []schema.Answer {
	result := make([]schema.Answer, len(answers))
	for i, a := range answers {
		result[i] = schema.Answer{
			QuestionIndex: a.QuestionIndex,
			Value:         a.Value,
		}
	}
	return result
}

// toSchemaQuestions converts DTO questions to schema.SurveyQuestion
func toSchemaQuestions(questions []dto.Question) []schema.SurveyQuestion {
	result := make([]schema.SurveyQuestion, len(questions))
	for i, q := range questions {
		result[i] = schema.SurveyQuestion{
			Question: q.Question,
			Type:     q.Type,
			Options:  q.Options,
			Required: q.Required,
		}
	}
	return result
}

// calculateNPS calculates Net Promoter Score
func calculateNPS(responses []*ent.SurveyResponse) float64 {
	promoters := 0
	detractors := 0
	for _, r := range responses {
		if r.Score >= 9 {
			promoters++
		} else if r.Score <= 6 {
			detractors++
		}
	}
	if len(responses) == 0 {
		return 0
	}
	return float64(promoters-detractors) / float64(len(responses)) * 100
}
