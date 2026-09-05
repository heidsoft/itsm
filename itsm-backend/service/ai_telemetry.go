package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"

	"go.uber.org/zap"
)

// AITelemetryService AI 遥测指标服务。
//
// 写路径（ObserveLLMCall / SaveFeedback）：调用 aiTelemetryRepository（封装层）。
// 读路径（GetMetrics 中的 AI 调用次数）：
//   - audit_logs 已经具备 Ent schema → 走 ent.AuditLog.Query()；
//   - ai_feedbacks / ai_llm_calls 没有 Ent → 走 aiTelemetryRepository。
//
// 所有 SQL 集中在 ai_telemetry_repository.go，未来表结构变更只改一处。
//
// db / repo 双字段保留：repo 承载 5 个写/聚合方法（ObserveLLMCall/SaveFeedback/
// AggregateFeedback/CountAIAuditLogs/AggregateLLMLatency），db 给 ai_evaluator.go
// 的若干读路径（loadFeedbackSamples / loadPlatformStats / listFeedback / countFeedback）
// 提供直连，避免在本次重构外引入额外 repository 拆解。
type AITelemetryService struct {
	db   *sql.DB
	repo *aiTelemetryRepository
	// skillRegistry 是可选的 SkillRegistry 引用（由 bootstrap 在服务组装阶段注入）。
	// 设置后，Evaluate 会把 AIScenarioEval.Kind 映射到 Skill.code/Skill.name，
	// 并额外生成 bySkill 维度的健康分；为 nil 时仍按旧"kind"维度聚合，向后兼容。
	skillRegistry *SkillRegistry
	// entClient 可选；为 nil 时 GetMetrics 的 audit_logs 聚合回退到 sql 计数。
	entClient *ent.Client
}

// NewAITelemetryService 创建 AI 遥测服务。
func NewAITelemetryService(db *sql.DB) *AITelemetryService {
	return &AITelemetryService{db: db, repo: newAITelemetryRepository(db)}
}

// SetEntClient 注入 Ent client，供 GetMetrics 中 audit_logs 聚合使用。
func (s *AITelemetryService) SetEntClient(client *ent.Client) {
	s.entClient = client
}

// SetSkillRegistry 注入 SkillRegistry，使 AI 评估报告能够按 Skill 维度聚合。
// 该方法在 bootstrap 期间调用；调用后 Evaluate 输出的 bySkill 字段将包含
// Skill 元数据（code/name）与子健康分（HealthScore）。可在测试中保持 nil 以
// 验证旧契约，向后兼容。
func (s *AITelemetryService) SetSkillRegistry(reg *SkillRegistry) {
	s.skillRegistry = reg
}

// LLMObserver implements the LLMGateway Observer interface. Every gateway call
// (success, rate-limited or failed) is recorded into ai_llm_calls so that
// GetMetrics can report a real avg_response_time_seconds instead of a constant.
// The gateway is tenant-agnostic, so the table intentionally carries no
// tenant_id; latency metrics are platform-level, while tenant-scoped counters
// keep coming from audit_logs / ai_feedbacks.
type LLMObserver struct {
	repo   *aiTelemetryRepository
	logger *zap.SugaredLogger
}

// NewLLMObserver 创建 LLM 观察者。
func NewLLMObserver(db *sql.DB, logger *zap.SugaredLogger) *LLMObserver {
	return &LLMObserver{repo: newAITelemetryRepository(db), logger: logger}
}

func (o *LLMObserver) Observe(provider string, model string, tokens int, latency time.Duration, err error) {
	success := err == nil
	if obsErr := o.repo.ObserveLLMCall(context.Background(), provider, model, tokens, latency.Milliseconds(), success); obsErr != nil {
		safeLog(o.logger, "failed to record LLM call metric", "error", obsErr, "provider", provider)
	}
}

// SaveFeedback saves user feedback on AI suggestions
func (s *AITelemetryService) SaveFeedback(ctx context.Context, tenantID, userID int, reqID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string) error {
	return s.repo.SaveFeedback(ctx, tenantID, userID, reqID, kind, query, itemType, itemID, useful, score, notes)
}

// GetMetrics retrieves AI usage metrics for a tenant
func (s *AITelemetryService) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	totalRequests, err := s.countAIAuditLogs(ctx, tenantID, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get total requests: %w", err)
	}
	metrics["total_requests"] = totalRequests

	feedback, err := s.repo.AggregateFeedback(ctx, tenantID, lookbackDays)
	if err != nil {
		return nil, err
	}
	metrics["total_feedback"] = feedback.TotalFeedback
	metrics["useful_feedback"] = feedback.UsefulFeedback
	if feedback.TotalFeedback > 0 {
		metrics["useful_rate"] = float64(feedback.UsefulFeedback) / float64(feedback.TotalFeedback)
	} else {
		metrics["useful_rate"] = 0.0
	}
	metrics["by_kind"] = feedback.ByKind

	// Average LLM latency from ai_llm_calls (platform-level, recorded by LLMObserver).
	// Kept tenant-agnostic: the gateway is wired before any tenant context exists.
	latencyAgg, err := s.repo.AggregateLLMLatency(ctx, lookbackDays)
	if err != nil {
		return nil, err
	}
	metrics["avg_response_time_seconds"] = latencyAgg.AvgLatencySeconds
	metrics["llm_call_count"] = latencyAgg.CallCount
	metrics["response_time_available"] = latencyAgg.CallCount > 0

	return metrics, nil
}

// countAIAuditLogs 通过 Ent 在 audit_logs 中统计"近 lookbackDays 内、含 ai 动作"的条数。
// 当未注入 entClient 时回退到 aiTelemetryRepository（仅在新旧 bootstrap 共存期）。
func (s *AITelemetryService) countAIAuditLogs(ctx context.Context, tenantID, lookbackDays int) (int, error) {
	if s.entClient != nil {
		since := time.Now().AddDate(0, 0, -lookbackDays)
		return s.entClient.AuditLog.Query().
			Where(
				auditlog.TenantIDEQ(tenantID),
				auditlog.ActionContains("ai"),
				auditlog.CreatedAtGTE(since),
			).
		Count(ctx)
	}
	if s.repo == nil {
		return 0, fmt.Errorf("ai telemetry repository not initialised")
	}
	return s.repo.CountAIAuditLogs(ctx, tenantID, lookbackDays)
}