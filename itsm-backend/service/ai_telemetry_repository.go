package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

// aiTelemetryRepository 封装 AI 遥测相关 raw SQL。
//
// 设计原因：ai_feedbacks / ai_llm_calls 两张表暂无 Ent schema（platform-level
// 数据，刻意不带 tenant_id），保留 raw SQL 但集中在一个 repository 内，便于：
//  1. 强制时间窗口与平台级聚合一致；
//  2. 表结构变更只改本文件；
//  3. 单元测试只需 mock 本仓储。
type aiTelemetryRepository struct {
	db *sql.DB
}

func newAITelemetryRepository(db *sql.DB) *aiTelemetryRepository {
	if db == nil {
		return nil
	}
	return &aiTelemetryRepository{db: db}
}

// ObserveLLMCall 在 ai_llm_calls 写一条 LLM 调用记录。
func (r *aiTelemetryRepository) ObserveLLMCall(ctx context.Context, provider, model string, tokens int, latencyMs int64, success bool) error {
	const query = `
		INSERT INTO ai_llm_calls (provider, model, tokens, latency_ms, success)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := r.db.ExecContext(ctx, query, provider, model, tokens, latencyMs, success); err != nil {
		return fmt.Errorf("observe llm call: %w", err)
	}
	return nil
}

// SaveFeedback 在 ai_feedbacks 写一条用户反馈。
//
// item_type='ai_audit' 时 accepted 是操作者 accept/reject 决策（v1.7
// Sprint 3 Task 1：解 useful overload），其他 item_type 时 useful 是
// 操作者对答案的赞踩。accepted 与 useful 互不覆盖，运营中可以分别
// 读两个维度。
func (r *aiTelemetryRepository) SaveFeedback(ctx context.Context, tenantID, userID int, requestID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string, accepted *bool) error {
	const sqlStr = `
		INSERT INTO ai_feedbacks (tenant_id, user_id, request_id, kind, query, item_type, item_id, useful, score, notes, accepted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	if _, err := r.db.ExecContext(ctx, sqlStr,
		tenantID, userID, requestID, kind, query, itemType,
		nullableInt(itemID), useful, nullableInt(score), nullableString(notes),
		nullableBool(accepted),
	); err != nil {
		return fmt.Errorf("save ai feedback: %w", err)
	}
	return nil
}

// UpdateFeedbackAccepted 翻转一条 ai_audit 记录的 accept/reject 决策。
// 幂等：同值重复调用无副作用。仅对 item_type='ai_audit' 生效。
func (r *aiTelemetryRepository) UpdateFeedbackAccepted(ctx context.Context, tenantID int64, id int64, accepted bool) error {
	const sqlStr = `
		UPDATE ai_feedbacks
		SET accepted = $3
		WHERE id = $1 AND tenant_id = $2 AND item_type = 'ai_audit'
	`
	res, err := r.db.ExecContext(ctx, sqlStr, id, tenantID, accepted)
	if err != nil {
		return fmt.Errorf("update ai feedback accepted: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrFeedbackNotFound
	}
	return nil
}

// ErrFeedbackNotFound 当 UpdateFeedbackAccepted 没找到匹配行（id 错、
// 租户错、或 item_type 不是 ai_audit）。
var ErrFeedbackNotFound = errors.New("ai feedback row not found")

// FeedbackAggregate 是按租户聚合的反馈指标。
type FeedbackAggregate struct {
	TotalFeedback  int
	UsefulFeedback int
	ByKind         map[string]int
	LookbackDays   int
}

// CountFeedbacks 返回按 kind 聚合的反馈计数与 useful 计数。
func (r *aiTelemetryRepository) AggregateFeedback(ctx context.Context, tenantID, lookbackDays int) (*FeedbackAggregate, error) {
	const summaryQuery = `
		SELECT COUNT(*), COUNT(CASE WHEN useful THEN 1 END)
		FROM ai_feedbacks
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
	`
	const byKindQuery = `
		SELECT kind, COUNT(*)
		FROM ai_feedbacks
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY kind
	`

	agg := &FeedbackAggregate{
		ByKind:       map[string]int{},
		LookbackDays: lookbackDays,
	}
	if err := r.db.QueryRowContext(ctx, summaryQuery, tenantID, lookbackDays).Scan(&agg.TotalFeedback, &agg.UsefulFeedback); err != nil {
		return nil, fmt.Errorf("aggregate feedback summary: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, byKindQuery, tenantID, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("aggregate feedback by kind: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var count int
		if scanErr := rows.Scan(&kind, &count); scanErr != nil {
			return nil, fmt.Errorf("scan feedback by kind: %w", scanErr)
		}
		agg.ByKind[kind] = count
	}
	return agg, nil
}

// LLMCallAggregate 平台级（无 tenant）LLM 调用延迟聚合。
type LLMCallAggregate struct {
	AvgLatencySeconds float64
	CallCount         int
}

// CountAIAuditLogs 通过 audit_logs 在指定租户下统计"近 lookbackDays 内、含 ai 动作"的条数。
// 优先由调用方走 Ent，仅在 entClient 不可用时回退到本方法。
func (r *aiTelemetryRepository) CountAIAuditLogs(ctx context.Context, tenantID, lookbackDays int) (int, error) {
	const query = `
		SELECT COUNT(*) FROM audit_logs
		WHERE tenant_id = $1 AND action LIKE '%ai%'
		AND created_at >= NOW() - INTERVAL '1 day' * $2
	`
	var total int
	if err := r.db.QueryRowContext(ctx, query, tenantID, lookbackDays).Scan(&total); err != nil {
		return 0, fmt.Errorf("count ai audit logs: %w", err)
	}
	return total, nil
}

// AggregateLLMLatency 计算全平台 LLM 平均延迟（秒）。
func (r *aiTelemetryRepository) AggregateLLMLatency(ctx context.Context, lookbackDays int) (*LLMCallAggregate, error) {
	const query = `
		SELECT COALESCE(AVG(latency_ms)::float / 1000.0, 0), COUNT(*)
		FROM ai_llm_calls
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	`
	agg := &LLMCallAggregate{}
	if err := r.db.QueryRowContext(ctx, query, lookbackDays).Scan(&agg.AvgLatencySeconds, &agg.CallCount); err != nil {
		return nil, fmt.Errorf("aggregate llm latency: %w", err)
	}
	return agg, nil
}

func nullableInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullableString(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullableBool(p *bool) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// safeLog 防止 nil logger 时 panic；调用方要么传 zap.NewN() 进来，要么允许 nil。
func safeLog(logger *zap.SugaredLogger, msg string, kv ...interface{}) {
	if logger == nil {
		return
	}
	logger.Warnw(msg, kv...)
}
