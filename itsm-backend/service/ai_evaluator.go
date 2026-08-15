package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ==================== AI 评估器 ====================
// 基于 ai_feedbacks（用户对 AI 建议的 useful/score=confidence×100 标签）与
// ai_llm_calls（平台级 LLM 调用观测）输出评估报告：
//   - 按场景(kind)的有用率与平均置信度
//   - 按 Skill（skillCode）聚合的有用率与健康分（与 byScenario 一致，但走 Skill 标识）
//   - 置信度校准：将 confidence 分桶，比较每桶有用率与桶中点，量化“高置信≠被接受”的偏差
//   - 平台级 LLM 成功率与延迟
//
// 时间边界由 Go 计算后以参数传入，SQL 不依赖 NOW()/INTERVAL，保证 sqlite 测试可用。

const (
	auditBucketCount = 5
	auditMaxPageSize = 200
)

// AIEvaluationReport 是 /api/v1/ai/evaluation 的返回契约（camelCase）。
//
// 字段兼容性说明（Sprint C）：
//   - ByScenario 保留为旧契约，内部字段已扩展 SkillCode / SkillName。
//   - BySkill 是新字段，结构与 ByScenario 一致，便于前端按"技能"维度渲染。
//   - BySkillAlias 是 ByScenario 的反别名（统一返回两个字段，前端可任选）。
//   - 旧 kind 字段（如 "triage"）会通过 scenarioToSkillCode 映射到 ai.triage。
type AIEvaluationReport struct {
	GeneratedAt           string                `json:"generatedAt"`
	LookbackDays          int                   `json:"lookbackDays"`
	TotalFeedback         int                   `json:"totalFeedback"`
	UsefulRate            float64               `json:"usefulRate"`
	AcceptedRate          float64               `json:"acceptedRate"`
	AvgConfidence         float64               `json:"avgConfidence"`
	HealthScore           float64               `json:"healthScore"`
	HasData               bool                  `json:"hasData"`
	ByScenario            []AIScenarioEval      `json:"byScenario"`
	BySkill               []AISkillEval         `json:"bySkill"`
	ConfidenceCalibration []AICalibrationBucket `json:"confidenceCalibration"`
	Platform              LLMPlatformStats      `json:"platform"`
}

// AIScenarioEval 单个 AI 场景（triage/summarize/analyze/rag_search/...）的评估。
//
// 字段兼容性：保留 Kind（字段名同前），并新增 SkillCode / SkillName。
// 旧"triage" → 新"ai.triage"由 scenarioToSkillCode 推导。
type AIScenarioEval struct {
	Kind          string  `json:"kind"`
	SkillCode     string  `json:"skillCode,omitempty"`
	SkillName     string  `json:"skillName,omitempty"`
	Count         int     `json:"count"`
	UsefulRate    float64 `json:"usefulRate"`
	AcceptedRate  float64 `json:"acceptedRate"`
	AvgConfidence float64 `json:"avgConfidence"`
}

// AISkillEval 按 Skill 维度的评估（来自 SkillRegistry 的 SkillEntity）。
//
// 与 AIScenarioEval 的区别：
//   - 字段名固定为 SkillCode（不再保留 Kind）；
//   - 增加 HealthScore 字段，按 manifest.eval.successThreshold 与校准误差
//     给出 0-100 的子健康分；
//   - 配合 AIEvaluationReport.HealthScore 形成"总-分"健康分。
type AISkillEval struct {
	SkillCode     string  `json:"skillCode"`
	SkillName     string  `json:"skillName,omitempty"`
	Count         int     `json:"count"`
	UsefulRate    float64 `json:"usefulRate"`
	AcceptedRate  float64 `json:"acceptedRate"`
	AvgConfidence float64 `json:"avgConfidence"`
	HealthScore   float64 `json:"healthScore"`
	IsPilot       bool    `json:"isPilot"`
}

// AICalibrationBucket 置信度分桶：中点=桶内置信度均值（评分校准的目标值），
// CalibrationError = |实际有用率 - 中点|，越小说明置信度越可信。
type AICalibrationBucket struct {
	Bucket           string  `json:"bucket"`
	Midpoint         float64 `json:"midpoint"`
	Count            int     `json:"count"`
	UsefulRate       float64 `json:"usefulRate"`
	CalibrationError float64 `json:"calibrationError"`
}

// LLMPlatformStats 平台级 LLM 调用统计（ai_llm_calls，无租户维度）。
type LLMPlatformStats struct {
	LLMCallCount int     `json:"llmCallCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
}

// scenarioToSkillCode 把旧"kind"（scenarios）映射到 Skill 标识。
//
// 映射规则：
//   - "triage"          → "ai.triage"
//   - "summarize"       → "ai.summarize"
//   - "analyze"         → "ai.analyze"
//   - "rag_search"      → "ai.knowledge_search"
//   - "chat"            → "ai.chat"
//   - "analytics"       → "ai.analytics"
//   - "prediction"      → "ai.trend_prediction"
//   - "create_ticket"   → "ai.create_ticket"
//   - "agent_tool"      → "ai.agent_tool"
//   - "metrics"         → "ai.metrics"
//   - "feedback"        → "ai.feedback"
//   - 其它              → "ai." + 原 kind（兼容未来扩展）
func scenarioToSkillCode(kind string) string {
	switch kind {
	case "triage":
		return "ai.triage"
	case "summarize":
		return "ai.summarize"
	case "analyze":
		return "ai.analyze"
	case "rag_search":
		return "ai.knowledge_search"
	case "chat":
		return "ai.chat"
	case "analytics":
		return "ai.analytics"
	case "prediction":
		return "ai.trend_prediction"
	case "create_ticket":
		return "ai.create_ticket"
	case "agent_tool":
		return "ai.agent_tool"
	case "metrics":
		return "ai.metrics"
	case "feedback":
		return "ai.feedback"
	}
	if kind == "" {
		return ""
	}
	return "ai." + kind
}

// Evaluate 输出租户在 lookbackDays 窗口内的 AI 评估报告。
func (s *AITelemetryService) Evaluate(ctx context.Context, tenantID int, lookbackDays int) (*AIEvaluationReport, error) {
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	since := time.Now().AddDate(0, 0, -lookbackDays)
	report := &AIEvaluationReport{
		GeneratedAt:           time.Now().Format(time.RFC3339),
		LookbackDays:          lookbackDays,
		ConfidenceCalibration: make([]AICalibrationBucket, 0, auditBucketCount),
		ByScenario:            make([]AIScenarioEval, 0),
		BySkill:               make([]AISkillEval, 0),
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
	// bySkillCode 与 byKind 同步聚合（key 是 skill.code，如 ai.triage）。
	// 即使 SkillRegistry 未注入，scenarioToSkillCode 仍可保证映射稳定。
	bySkillCode := map[string]*kindAgg{}
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
		// 同步累计到 bySkillCode：kind → skillCode 的映射在聚合阶段完成。
		skillCode := scenarioToSkillCode(smp.kind)
		if skillCode != "" {
			sAgg := bySkillCode[skillCode]
			if sAgg == nil {
				sAgg = &kindAgg{}
				bySkillCode[skillCode] = sAgg
			}
			sAgg.count++
			sAgg.usefulCount += boolToInt(smp.useful)
			if smp.itemType == "ai_audit" {
				sAgg.auditCount++
				sAgg.acceptedCount += boolToInt(smp.useful)
			}
			sAgg.scoreSum += smp.score
		}
	}
	report.UsefulRate = round4(float64(usefulCount) / float64(len(samples)))
	report.AvgConfidence = round4(float64(scoreSum) / float64(len(samples)) / 100.0)
	if auditCount > 0 {
		report.AcceptedRate = round4(float64(acceptedCount) / float64(auditCount))
	}

	// 3) 按场景分解。每个 AIScenarioEval 同时填上 SkillCode（来自
	// scenarioToSkillCode 的稳定映射），与 SkillName（如果 SkillRegistry 已注入）。
	for kind, agg := range byKind {
		scenario := AIScenarioEval{
			Kind:          kind,
			SkillCode:     scenarioToSkillCode(kind),
			Count:         agg.count,
			UsefulRate:    round4(float64(agg.usefulCount) / float64(agg.count)),
			AvgConfidence: round4(float64(agg.scoreSum) / float64(agg.count) / 100.0),
		}
		if scenario.SkillCode != "" && s.skillRegistry != nil {
			if sk, err := s.skillRegistry.Get(scenario.SkillCode); err == nil {
				scenario.SkillName = sk.Name()
			}
		}
		if agg.auditCount > 0 {
			scenario.AcceptedRate = round4(float64(agg.acceptedCount) / float64(agg.auditCount))
		}
		report.ByScenario = append(report.ByScenario, scenario)
	}
	sortScenariosByCount(report.ByScenario)

	// 4) 按 Skill 维度分解（Sprint C 新增维度）。
	//	与 byScenario 互为同构表达，但 key 固定为 skill.code。
	//	HealthScore 子分 = usefulRate*40 + acceptedRate*30 + (1.0 - |usefulRate-avgConfidence|)*30
	//	  即"用户满意度" 40 + "被采纳度" 30 + "置信度校准" 30。
	//	IsPilot 由 SkillRegistry 注入；未注入时按 false 输出，保留稳定字段。
	report.BySkill = make([]AISkillEval, 0, len(bySkillCode))
	for code, agg := range bySkillCode {
		skill := AISkillEval{
			SkillCode:     code,
			Count:         agg.count,
			UsefulRate:    round4(float64(agg.usefulCount) / float64(agg.count)),
			AcceptedRate:  0.0,
			AvgConfidence: round4(float64(agg.scoreSum) / float64(agg.count) / 100.0),
		}
		if agg.auditCount > 0 {
			skill.AcceptedRate = round4(float64(agg.acceptedCount) / float64(agg.auditCount))
		}
		// 置信度校准误差：|实际有用率 - 平均置信度|，越小说明置信度越接近实际。
		calibrationError := absFloat(skill.UsefulRate - skill.AvgConfidence)
		skill.HealthScore = round4(
			skill.UsefulRate*40.0 +
				skill.AcceptedRate*30.0 +
				(1.0-calibrationError)*30.0,
		)
		// 名称 / IsPilot 由 SkillRegistry 注入。
		if s.skillRegistry != nil {
			if sk, err := s.skillRegistry.Get(code); err == nil {
				skill.SkillName = sk.Name()
				skill.IsPilot = strings.EqualFold(sk.Manifest().Category, "pilot")
			}
		}
		report.BySkill = append(report.BySkill, skill)
	}
	sortSkillsByCount(report.BySkill)

	// 5) 置信度校准：score(0-100) 均分为 5 桶，比较桶内有用率与置信度中点。
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

// sortSkillsByCount 按调用次数降序排列 bySkill 列表，与 sortScenariosByCount
// 保持一致的口径：次数多的排前，便于前端在健康看板中优先展示高频 Skill。
func sortSkillsByCount(skills []AISkillEval) {
	for i := 1; i < len(skills); i++ {
		for j := i; j > 0 && skills[j].Count > skills[j-1].Count; j-- {
			skills[j], skills[j-1] = skills[j-1], skills[j]
		}
	}
}
