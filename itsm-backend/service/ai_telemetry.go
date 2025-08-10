package service

import (
	"context"
	"database/sql"
	"fmt"
)

type AITelemetryService struct {
	db *sql.DB
}

func NewAITelemetryService(db *sql.DB) *AITelemetryService {
	return &AITelemetryService{db: db}
}

// SaveFeedback saves user feedback on AI suggestions
func (s *AITelemetryService) SaveFeedback(ctx context.Context, tenantID, userID int, reqID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string) error {
	queryStr := `
		INSERT INTO ai_feedbacks (tenant_id, user_id, request_id, kind, query, item_type, item_id, useful, score, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var itemIDVal interface{}
	if itemID != nil {
		itemIDVal = *itemID
	} else {
		itemIDVal = nil
	}

	var scoreVal interface{}
	if score != nil {
		scoreVal = *score
	} else {
		scoreVal = nil
	}

	var notesVal interface{}
	if notes != nil {
		notesVal = *notes
	} else {
		notesVal = nil
	}

	_, err := s.db.ExecContext(ctx, queryStr, tenantID, userID, reqID, kind, query, itemType, itemIDVal, useful, scoreVal, notesVal)
	return err
}

// GetMetrics retrieves AI usage metrics for a tenant
func (s *AITelemetryService) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get total AI requests from audit_logs
	var totalRequests int
	query := `
		SELECT COUNT(*) FROM audit_logs 
		WHERE tenant_id = $1 AND action LIKE '%ai%' 
		AND created_at >= NOW() - INTERVAL '1 day' * $2
	`
	err := s.db.QueryRowContext(ctx, query, tenantID, lookbackDays).Scan(&totalRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to get total requests: %w", err)
	}
	metrics["total_requests"] = totalRequests

	// Get feedback metrics
	var totalFeedback, usefulFeedback int
	query = `
		SELECT COUNT(*), COUNT(CASE WHEN useful THEN 1 END)
		FROM ai_feedbacks 
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
	`
	err = s.db.QueryRowContext(ctx, query, tenantID, lookbackDays).Scan(&totalFeedback, &usefulFeedback)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback metrics: %w", err)
	}

	metrics["total_feedback"] = totalFeedback
	metrics["useful_feedback"] = usefulFeedback

	if totalFeedback > 0 {
		metrics["useful_rate"] = float64(usefulFeedback) / float64(totalFeedback)
	} else {
		metrics["useful_rate"] = 0.0
	}

	// Get metrics by kind
	query = `
		SELECT kind, COUNT(*) 
		FROM ai_feedbacks 
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY kind
	`
	rows, err := s.db.QueryContext(ctx, query, tenantID, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get kind metrics: %w", err)
	}
	defer rows.Close()

	kindMetrics := make(map[string]int)
	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			return nil, fmt.Errorf("failed to scan kind metrics: %w", err)
		}
		kindMetrics[kind] = count
	}
	metrics["by_kind"] = kindMetrics

	// Get average response time (estimated from audit logs)
	var avgResponseTime float64
	query = `
		SELECT AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) 
		FROM audit_logs 
		WHERE tenant_id = $1 AND action LIKE '%ai%' 
		AND created_at >= NOW() - INTERVAL '1 day' * $2
		AND updated_at IS NOT NULL
	`
	err = s.db.QueryRowContext(ctx, query, tenantID, lookbackDays).Scan(&avgResponseTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get avg response time: %w", err)
	}
	if err == sql.ErrNoRows {
		avgResponseTime = 0.0
	}
	metrics["avg_response_time_seconds"] = avgResponseTime

	return metrics, nil
}
