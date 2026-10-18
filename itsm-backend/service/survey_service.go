package service

import (
	"context"
	"fmt"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/survey"
	"itsm-backend/ent/surveyresponse"
	"itsm-backend/ent/ticket"

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
func (s *SurveyService) SubmitResponse(ctx context.Context, req *dto.SubmitSurveyRequest, respondentID, tenantID int) error {
	// surveyId 必须属于当前租户：此前直接落库，他租 survey 可被引用（跨租户
	// 引用注入），响应行虽然打着本租户 tenant_id，survey 边却指向对端。
	surveyExists, err := s.client.Survey.Query().
		Where(survey.ID(req.SurveyID), survey.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check survey for submit", "error", err, "survey_id", req.SurveyID)
		return fmt.Errorf("check survey for submit: %w", err)
	}
	if !surveyExists {
		// 引用型资源缺失（不存在或跨租户）：400/1001，与工单评论提及口径一致。
		return common.NewBusinessError(common.ParamErrorCode, "问卷不存在或不属于当前租户", "")
	}
	// ticketId 同为本租引用面：无效引用会让响应挂在不存在/他租工单上。
	ticketExists, err := s.client.Ticket.Query().
		Where(ticket.ID(req.TicketID), ticket.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket for submit", "error", err, "ticket_id", req.TicketID)
		return fmt.Errorf("check ticket for submit: %w", err)
	}
	if !ticketExists {
		return common.NewBusinessError(common.ParamErrorCode, "评分工单不存在或不属于当前租户", "")
	}

	_, err = s.client.SurveyResponse.Create().
		SetSurveyID(req.SurveyID).
		SetTicketID(req.TicketID).
		SetRespondentID(respondentID). // 来自认证上下文的真实用户（此前硬编码 0）
		SetAnswers(toSchemaAnswers(req.Answers)).
		SetScore(req.Score).
		SetComment(req.Comment).
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

// GetAnalytics retrieves analytics for a survey
func (s *SurveyService) GetAnalytics(ctx context.Context, surveyID int, tenantID int) (*dto.SurveyAnalytics, error) {
	// 先验 survey 归属：此前直接按响应行聚合（响应行带本租谓词），
	// 他租 survey id 会得到 200 + 空统计而非 404——存在性探测面。
	if _, err := s.GetSurvey(ctx, surveyID, tenantID); err != nil {
		return nil, err
	}
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
	// 先验 survey 归属：跨租户 survey id 此前得到 200 + 空列表而非 404。
	if _, err := s.GetSurvey(ctx, surveyID, tenantID); err != nil {
		return nil, err
	}
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
