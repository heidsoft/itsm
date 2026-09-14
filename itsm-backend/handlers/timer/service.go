package timer

import (
	"context"
	"strconv"
	"time"

	"itsm-backend/ent"
	"itsm-backend/service"
)

type Service struct {
	store service.TimerStore
}

func NewService(store service.TimerStore) *Service {
	return &Service{store: store}
}

type TimerDTO struct {
	ID                   int                    `json:"id"`
	TimerID              string                 `json:"timerId"`
	TimerType            string                 `json:"timerType"`
	ProcessDefinitionKey string                 `json:"processDefinitionKey"`
	ProcessInstanceID    int                    `json:"processInstanceId"`
	ActivityID           string                 `json:"activityId"`
	TimerExpression      string                 `json:"timerExpression"`
	ExpressionType       string                 `json:"expressionType"`
	FireAt               string                 `json:"fireAt"`
	FiredAt              string                 `json:"firedAt,omitempty"`
	Status               string                 `json:"status"`
	Version              int                    `json:"version"`
	RetryCount           int                    `json:"retryCount"`
	MaxRetries           int                    `json:"maxRetries"`
	LastFireAttempt      string                 `json:"lastFireAttempt,omitempty"`
	FailureReason        string                 `json:"failureReason,omitempty"`
	ContextVariables     map[string]interface{} `json:"contextVariables,omitempty"`
	TotalDurationSeconds float64                `json:"totalDurationSeconds,omitempty"`
	ElapsedSeconds       float64                `json:"elapsedSeconds,omitempty"`
	PauseState           string                 `json:"pauseState"`
	ParentTimerID        int                    `json:"parentTimerId,omitempty"`
	TenantID             int                    `json:"tenantId"`
	CreatedAt            string                 `json:"createdAt"`
	UpdatedAt            string                 `json:"updatedAt"`
}

type TimerListResponse struct {
	Items      []*TimerDTO `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

func (s *Service) List(ctx context.Context, filter service.TimerListFilter) (*TimerListResponse, error) {
	items, total, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	dtoItems := make([]*TimerDTO, 0, len(items))
	for _, t := range items {
		dtoItems = append(dtoItems, toDTO(t))
	}
	totalPages := 0
	if filter.PageSize > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return &TimerListResponse{
		Items:      dtoItems,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) Get(ctx context.Context, tenantID int, timerID string) (*TimerDTO, error) {
	timer, err := s.store.GetByTimerID(ctx, timerID)
	if err != nil {
		return nil, err
	}
	if timer.TenantID != tenantID {
		return nil, &notFoundError{}
	}
	return toDTO(timer), nil
}

func (s *Service) Stats(ctx context.Context, tenantID int) (*service.TimerStats, error) {
	return s.store.Stats(ctx, tenantID)
}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "timer not found" }

func IsNotFound(err error) bool {
	_, ok := err.(*notFoundError)
	return ok
}

func toDTO(t *ent.ProcessTimer) *TimerDTO {
	dto := &TimerDTO{
		ID:                   t.ID,
		TimerID:              t.TimerID,
		TimerType:            t.TimerType,
		ProcessDefinitionKey: t.ProcessDefinitionKey,
		ProcessInstanceID:    t.ProcessInstanceID,
		ActivityID:           t.ActivityID,
		TimerExpression:      t.TimerExpression,
		ExpressionType:       t.ExpressionType,
		FireAt:               t.FireAt.Format(time.RFC3339),
		Status:               t.Status,
		Version:              t.Version,
		RetryCount:           t.RetryCount,
		MaxRetries:           t.MaxRetries,
		FailureReason:        t.FailureReason,
		ContextVariables:     t.ContextVariables,
		TotalDurationSeconds: t.TotalDurationSeconds,
		ElapsedSeconds:       t.ElapsedSeconds,
		PauseState:           t.PauseState,
		ParentTimerID:        t.ParentTimerID,
		TenantID:             t.TenantID,
		CreatedAt:            t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            t.UpdatedAt.Format(time.RFC3339),
	}
	if !t.FiredAt.IsZero() {
		dto.FiredAt = t.FiredAt.Format(time.RFC3339)
	}
	if !t.LastFireAttempt.IsZero() {
		dto.LastFireAttempt = t.LastFireAttempt.Format(time.RFC3339)
	}
	return dto
}

func parseIntParam(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
