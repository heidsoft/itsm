package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ==================== AI 评估器 ====================
// 基于 ai_feedbacks（用户对 AI 建议的 useful/score=confidence×100 标签）与
// ai_llm_calls（平台级 LLM 调用观测）输出评估报告：
//   - 按场景(kind)的有用率与平均置信度
//   - 置信度校准：将 confidence 分桶，比较每桶有用率与桶中点，量化“高置信≠被接受”的偏差
//   - 平台级 LLM 成功率与延迟
//
// 时间边界由 Go 计算后以参数传入，SQL 不依赖 NOW()/INTERVAL，保证 sqlite 测试可用。

const (
	auditBucketCount = 5
	auditMaxPageSize = 200
)

// AIEvaluationReport 是 /api/v1/ai/evaluation 的返回契约（camelCase）。
type AIEvaluationReport struct {
	GeneratedAt          string                `json:"generatedAt"`
	LookbackDays         int                   `json:"lookbackDays"`
	TotalFeedback        int                   `json:"totalFeedback"`
	UsefulRate           float64               `json:"usefulRate"`
	AcceptedRate         float64               `json:"acceptedRate"`
	AvgConfidence        float64               `json:"avgConfidence"`
	HealthScore          float64               `json:"healthScore"`
	HasData              bool                  `json:"hasData"`
	ByScenario           []AIScenarioEval      `json:"byScenario"`
	ConfidenceCalibration []AICalibrationBucket `json:"confidenceCalibration"`
	Platform             LLMPlatformStats      `json:"platform"`
}

// AIScenarioEval 单个 AI 场景（triage/summarize/analyze/rag_search/...）的评估。
type AIScenarioEval struct {
	Kind          string  `json:"kind"`
	Count         int     `json:"count"`
	UsefulRate    float64 `json:"usefulRate"`
	AcceptedRate  float64 `json:"acceptedRate"`
	AvgConfidence float64 `json:"avgConfidence"`
}

// AICalibrationBucket 置信度分桶：中点=桶内置信度均值（评分校准的目标值），
// CalibrationError = |实际有用率 - 中点|，越小说明置信度越可信。
type AICalibrationBucket struct {
	Bucket            string  `json:"bucket"`
	Midpoint          float64 `json:"midpoint"`
	Count             int     `json:"count"`
	UsefulRate        float64 `json:"usefulRate"`
	CalibrationError  float64 `json:"calibrationError"`
}

// LLMPlatformStats 平台级 LLM 调用统计（ai_llm_calls，无租户维度）。
type LLMPlatformStats struct {
	LLMCallCount int     `json:"llmCallCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
}

// Evaluate 输出租户在 lookbackDays 窗口内的 AI 评估报告。
func (s *AITelemetryService) Evaluate(ctx context.Context, tenantID int, lookbackDays int) (*AIEvaluationReport, error) {
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	since := time.Now().AddDate(0, 0, -lookbackDays)
	report := &AIEvaluationReport{
		GeneratedAt:  time.Now().Format(time.RFC3339),
		LookbackDays: lookbackDays,
		ConfidenceCalibration: make([]AICalibrationBucket, 0, auditBucketCount),
		ByScenario:   make([]AIScenarioEval, 0),
	}

	// 1) 窗口内所有反馈样本（含审计记录），一次取回后在内存聚合，保证跨库兼容。
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(useful, false), COALESCE(score, 0), COALESCE(kind, ''), COALESCE(item_type, '')
		FROM ai_feedbacks
		WHERE tenant_id = $1 AND created_at >= $2
	`, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to load feedback samples: %w", err)
	}
	defer rows.Close()

	type sample struct {
		useful   bool
		score    int
		kind     string
		itemType string
	}
	var samples []sample
	for rows.Next() {
		var s sample
		if err := rows.Scan(&s.useful, &s.score, &s.kind, &s.itemType); err != nil {
			return nil, fmt.Errorf("failed to scan feedback sample: %w", err)
		}
		samples = append(samples, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating feedback samples: %w", err)
	}

	if len(samples) == 0 {
		// 无样本：健康分 0 并标记 hasData=false，前端展示空态而非误导性数值。
		report.Platform = s.loadPlatformStats(ctx, lookbackDays)
		return report, nil
	}
	report.HasData = true
	report.TotalFeedback = len(samples)

	// 2) 总体指标：有用率（全部样本）、被采纳率（仅 ai_audit）、平均置信度。
	usefulCount := 0
	acceptedCount, auditCount := 0, 0
	scoreSum := 0
	byKind := map[string]*kindAgg{}
	for _, smp := range samples {
		if smp.useful {
			usefulCount++
			if smp.itemType == "ai_audit" {
				acceptedCount++
			}
		}
		if smp.itemType == "ai_audit" {
			auditCount++
		}
		scoreSum += smp.score
		agg := byKind[smp.kind]
		if agg == nil {
			agg = &kindAgg{}
			byKind[smp.kind] = agg
		}
		agg.count++
		agg.usefulCount += boolToInt(smp.useful)
		if smp.itemType == "ai_audit" {
			agg.auditCount++
			agg.acceptedCount += boolToInt(smp.useful)
		}
		agg.scoreSum += smp.score
	}
	report.UsefulRate = round4(float64(usefulCount) / float64(len(samples)))
	report.AvgConfidence = round4(float64(scoreSum) / float64(len(samples)) / 100.0)
	if auditCount > 0 {
		report.AcceptedRate = round4(float64(acceptedCount) / float64(auditCount))
	}

	// 3) 按场景分解。
	for kind, agg := range byKind {
		scenario := AIScenarioEval{
			Kind:          kind,
			Count:         agg.count,
			UsefulRate:    round4(float64(agg.usefulCount) / float64(agg.count)),
			AvgConfidence: round4(float64(agg.scoreSum) / float64(agg.count) / 100.0),
		}
		if agg.auditCount > 0 {
			scenario.AcceptedRate = round4(float64(agg.acceptedCount) / float64(agg.auditCount))
		}
		report.ByScenario = append(report.ByScenario, scenario)
	}
	sortScenariosByCount(report.ByScenario)

	// 4) 置信度校准：score(0-100) 均分为 5 桶，比较桶内有用率与置信度中点。
	bucketAggs := make([]*calibrationAgg, auditBucketCount)
	for i := range bucketAggs {
		bucketAggs[i] = &calibrationAgg{}
	}
	for _, smp := range samples {
		idx := smp.score / 20
		if idx < 0 {
			idx = 0
		}
		if idx >= auditBucketCount {
			idx = auditBucketCount - 1
		}
		bucketAggs[idx].count++
		bucketAggs[idx].usefulCount += boolToInt(smp.useful)
		bucketAggs[idx].scoreSum += smp.score
	}
	for i, agg := range bucketAggs {
		low := i * 20
		high := low + 20
		bucket := AICalibrationBucket{
			Bucket:   fmt.Sprintf("%d-%d%%", low, high),
			Midpoint: float64(low+high) / 2.0 / 100.0,
			Count:    agg.count,
		}
		if agg.count > 0 {
			usefulRate := float64(agg.usefulCount) / float64(agg.count)
			bucket.UsefulRate = round4(usefulRate)
			bucket.CalibrationError = round4(absFloat(usefulRate - bucket.Midpoint))
		}
		report.ConfidenceCalibration = append(report.ConfidenceCalibration, bucket)
	}

	// 5) 健康分（0-100）：有用率 40% + 被采纳率 30% + LLM 成功率 30%。
	report.Platform = s.loadPlatformStats(ctx, lookbackDays)
	report.HealthScore = round4(report.UsefulRate*40.0 + report.AcceptedRate*30.0 + report.Platform.SuccessRate*30.0)
	return report, nil
}

// loadPlatformStats 读取平台级 LLM 调用统计（成功率和平均延迟）。
func (s *AITelemetryService) loadPlatformStats(ctx context.Context, lookbackDays int) LLMPlatformStats {
	since := time.Now().AddDate(0, 0, -lookbackDays)
	stats := LLMPlatformStats{}
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(AVG(CASE WHEN success THEN 1 ELSE 0 END), 0), COALESCE(AVG(latency_ms), 0)
		FROM ai_llm_calls
		WHERE created_at >= $1
	`, since).Scan(&stats.LLMCallCount, &stats.SuccessRate, &stats.AvgLatencyMs)
	if err != nil {
		// 平台统计失败不阻塞评估报告主体。
		return stats
	}
	stats.SuccessRate = round4(stats.SuccessRate)
	stats.AvgLatencyMs = round2(stats.AvgLatencyMs)
	return stats
}

// ==================== AI 审计日志查询 ====================

// AIAuditEntry 是 /api/v1/ai/audit-logs 列表项（camelCase）。
// notes 字段为 JSON（prompt_version/model/confidence/suggestion/notes），
// 由 RecordAudit 写入 ai_feedbacks.notes，此处解析为结构化字段。
type AIAuditEntry struct {
	ID            int64                  `json:"id"`
	CreatedAt     string                 `json:"createdAt"`
	TenantID      int                    `json:"tenantId"`
	UserID        int                    `json:"userId"`
	RequestID     string                 `json:"requestId"`
	Scenario      string                 `json:"scenario"`
	InputRef      string                 `json:"inputRef"`
	PromptVersion string                 `json:"promptVersion"`
	Model         string                 `json:"model"`
	Confidence    float64                `json:"confidence"`
	Accepted      bool                   `json:"accepted"`
	Suggestion    map[string]interface{} `json:"suggestion"`
	Notes         string                 `json:"notes"`
}

// ListAuditLogs 分页查询租户的 AI 审计记录（item_type='ai_audit'）。
// kind 过滤通过占位符参数化，查询骨架为固定字符串，不拼接用户输入。
func (s *AITelemetryService) ListAuditLogs(ctx context.Context, tenantID int, page, pageSize int, kind string, lookbackDays int) ([]AIAuditEntry, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > auditMaxPageSize {
		pageSize = 20
	}
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	since := time.Now().AddDate(0, 0, -lookbackDays)

	const countSQL = "SELECT COUNT(*) FROM ai_feedbacks WHERE tenant_id = $1 AND item_type = 'ai_audit' AND created_at >= $2"
	const countSQLKind = "SELECT COUNT(*) FROM ai_feedbacks WHERE tenant_id = $1 AND item_type = 'ai_audit' AND created_at >= $2 AND kind = $3"
	const listSQL = "SELECT id, created_at, tenant_id, user_id, request_id, kind, COALESCE(query, ''), useful, COALESCE(score, 0), COALESCE(notes, '') FROM ai_feedbacks WHERE tenant_id = $1 AND item_type = 'ai_audit' AND created_at >= $2 ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4"
	const listSQLKind = "SELECT id, created_at, tenant_id, user_id, request_id, kind, COALESCE(query, ''), useful, COALESCE(score, 0), COALESCE(notes, '') FROM ai_feedbacks WHERE tenant_id = $1 AND item_type = 'ai_audit' AND created_at >= $2 AND kind = $3 ORDER BY created_at DESC, id DESC LIMIT $4 OFFSET $5"

	var total int
	var err error
	if kind != "" {
		err = s.db.QueryRowContext(ctx, countSQLKind, tenantID, since, kind).Scan(&total)
	} else {
		err = s.db.QueryRowContext(ctx, countSQL, tenantID, since).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	var rows interface {
		Next() bool
		Scan(...interface{}) error
		Err() error
		Close() error
	}
	if kind != "" {
		rows, err = s.db.QueryContext(ctx, listSQLKind, tenantID, since, kind, pageSize, (page-1)*pageSize)
	} else {
		rows, err = s.db.QueryContext(ctx, listSQL, tenantID, since, pageSize, (page-1)*pageSize)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	entries := make([]AIAuditEntry, 0)
	for rows.Next() {
		var e AIAuditEntry
		var createdAt time.Time
		var score int
		var notes string
		if err := rows.Scan(&e.ID, &createdAt, &e.TenantID, &e.UserID, &e.RequestID, &e.Scenario, &e.InputRef, &e.Accepted, &score, &notes); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		e.CreatedAt = createdAt.Format(time.RFC3339)
		e.Confidence = round4(float64(score) / 100.0)
		e.Notes = notes
		// notes 中冗余了 confidence 等字段；解析时仅提取结构化元数据，原始 notes 保留展示。
		if notes != "" {
			var meta struct {
				PromptVersion string                 `json:"prompt_version"`
				Model         string                 `json:"model"`
				Suggestion    map[string]interface{} `json:"suggestion"`
			}
			if err := json.Unmarshal([]byte(notes), &meta); err == nil {
				e.PromptVersion = meta.PromptVersion
				e.Model = meta.Model
				e.Suggestion = meta.Suggestion
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed iterating audit logs: %w", err)
	}
	return entries, total, nil
}

// ==================== 内部聚合工具 ====================

type kindAgg struct {
	count         int
	usefulCount   int
	auditCount    int
	acceptedCount int
	scoreSum      int
}

type calibrationAgg struct {
	count       int
	usefulCount int
	scoreSum    int
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func round4(f float64) float64 {
	return float64(int(f*10000+0.5)) / 10000.0
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100.0
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func sortScenariosByCount(scenarios []AIScenarioEval) {
	for i := 1; i < len(scenarios); i++ {
		for j := i; j > 0 && scenarios[j].Count > scenarios[j-1].Count; j-- {
			scenarios[j], scenarios[j-1] = scenarios[j-1], scenarios[j]
		}
	}
}
