package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type AITelemetryService struct {
	db *sql.DB
}

func NewAITelemetryService(db *sql.DB) *AITelemetryService {
	return &AITelemetryService{db: db}
}

// LLMObserver implements the LLMGateway Observer interface. Every gateway call
// (success, rate-limited or failed) is recorded into ai_llm_calls so that
// GetMetrics can report a real avg_response_time_seconds instead of a constant.
// The gateway is tenant-agnostic, so the table intentionally carries no
// tenant_id; latency metrics are platform-level, while tenant-scoped counters
// keep coming from audit_logs / ai_feedbacks.
type LLMObserver struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewLLMObserver(db *sql.DB, logger *zap.SugaredLogger) *LLMObserver {
	return &LLMObserver{db: db, logger: logger}
}

func (o *LLMObserver) Observe(provider string, model string, tokens int, latency time.Duration, err error) {
	const insertSQL = `
		INSERT INTO ai_llm_calls (provider, model, tokens, latency_ms, success)
		VALUES ($1, $2, $3, $4, $5)
	`
	success := err == nil
	if _, insertErr := o.db.Exec(insertSQL, provider, model, tokens, latency.Milliseconds(), success); insertErr != nil {
		o.logger.Warnw("failed to record LLM call metric", "error", insertErr, "provider", provider)
	}
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

	// Average LLM latency from ai_llm_calls (platform-level, recorded by LLMObserver).
	// Kept tenant-agnostic: the gateway is wired before any tenant context exists.
	var avgLatencySeconds float64
	var llmCallCount int
	latencyQuery := `
		SELECT COALESCE(AVG(latency_ms)::float / 1000.0, 0), COUNT(*)
		FROM ai_llm_calls
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	`
	if err := s.db.QueryRowContext(ctx, latencyQuery, lookbackDays).Scan(&avgLatencySeconds, &llmCallCount); err != nil {
		return nil, fmt.Errorf("failed to get latency metrics: %w", err)
	}
	metrics["avg_response_time_seconds"] = avgLatencySeconds
	metrics["llm_call_count"] = llmCallCount
	metrics["response_time_available"] = llmCallCount > 0

	return metrics, nil
}
