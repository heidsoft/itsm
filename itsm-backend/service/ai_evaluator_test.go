package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTelemetryTestDB 建立 sqlite 内存库的最小 ai_feedbacks / ai_llm_calls 表，
// 用于评估器与审计日志查询的单元测试（与 database.go 的 PG 结构字段一致）。
func newTelemetryTestDB(t *testing.T) (*sql.DB, *AITelemetryService) {
	t.Helper()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_feedbacks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TIMESTAMP,
			tenant_id INT NOT NULL,
			user_id INT NOT NULL,
			request_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			query TEXT,
			item_type TEXT,
			item_id INT,
			useful BOOLEAN NOT NULL,
			score INT,
			notes TEXT
		);
		CREATE TABLE IF NOT EXISTS ai_llm_calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TIMESTAMP,
			provider TEXT,
			model TEXT,
			tokens INT,
			latency_ms INT,
			success BOOLEAN
		);
	`)
	require.NoError(t, err)
	return db, NewAITelemetryService(db)
}

type feedbackSeed struct {
	tenantID int
	userID   int
	kind     string
	query    string
	itemType string
	useful   bool
	score    int
	notes    string
}

func seedFeedback(t *testing.T, db *sql.DB, f feedbackSeed) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO ai_feedbacks (created_at, tenant_id, user_id, request_id, kind, query, item_type, useful, score, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, time.Now(), f.tenantID, f.userID, fmt.Sprintf("req_%d", time.Now().UnixNano()), f.kind, f.query, f.itemType, f.useful, f.score, f.notes)
	require.NoError(t, err)
}

func seedLLMCall(t *testing.T, db *sql.DB, latencyMS int, success bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO ai_llm_calls (created_at, provider, model, tokens, latency_ms, success)
		VALUES (?, 'test', 'doubao-test', 100, ?, ?)
	`, time.Now(), latencyMS, success)
	require.NoError(t, err)
}

// seedStandardEvaluationFixture 写入租户1的四条反馈样本 + 平台四条 LLM 调用：
//
//	triage     useful=true  score=80
//	summarize  useful=false score=40
//	analyze    useful=true  score=90 (ai_audit, accepted)
//	analyze    useful=false score=95 (ai_audit, rejected)
//	LLM: 成功 2000ms / 成功 1000ms / 失败 3000ms / 成功 500ms
func seedStandardEvaluationFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	notes := `{"prompt_version":"v1","model":"doubao-pro","suggestion":{"category":"network"}}`
	seedFeedback(t, db, feedbackSeed{tenantID: 1, userID: 1, kind: "triage", query: "网络中断", itemType: "", useful: true, score: 80})
	seedFeedback(t, db, feedbackSeed{tenantID: 1, userID: 1, kind: "summarize", query: "总结工单", itemType: "", useful: false, score: 40})
	seedFeedback(t, db, feedbackSeed{tenantID: 1, userID: 1, kind: "analyze", query: "ticket_1", itemType: "ai_audit", useful: true, score: 90, notes: notes})
	seedFeedback(t, db, feedbackSeed{tenantID: 1, userID: 1, kind: "analyze", query: "ticket_2", itemType: "ai_audit", useful: false, score: 95, notes: notes})
	seedFeedback(t, db, feedbackSeed{tenantID: 2, userID: 2, kind: "triage", query: "隔离验证", itemType: "", useful: true, score: 100})
	seedLLMCall(t, db, 2000, true)
	seedLLMCall(t, db, 1000, true)
	seedLLMCall(t, db, 3000, false)
	seedLLMCall(t, db, 500, true)
}

func TestEvaluate_OverallMetrics(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)
	assert.True(t, report.HasData)
	assert.Equal(t, 4, report.TotalFeedback)
	// useful: triage+analyze 两条 → 0.5
	assert.Equal(t, 0.5, report.UsefulRate)
	// ai_audit 2 条，accepted 1 条 → 0.5
	assert.Equal(t, 0.5, report.AcceptedRate)
	// (80+40+90+95)/4/100 = 0.7625
	assert.Equal(t, 0.7625, report.AvgConfidence)
	// 健康分 = 0.5*40 + 0.5*30 + 0.75*30 = 57.5
	assert.Equal(t, 57.5, report.HealthScore)
	// 平台统计：4 次调用，3 成功，平均延迟 (2000+1000+3000+500)/4 = 1625ms
	assert.Equal(t, 4, report.Platform.LLMCallCount)
	assert.Equal(t, 0.75, report.Platform.SuccessRate)
	assert.Equal(t, 1625.0, report.Platform.AvgLatencyMs)
}

func TestEvaluate_ScenarioBreakdown(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)

	byKind := map[string]AIScenarioEval{}
	for _, s := range report.ByScenario {
		byKind[s.Kind] = s
	}
	require.Len(t, byKind, 3)
	assert.Equal(t, 1, byKind["triage"].Count)
	assert.Equal(t, 1.0, byKind["triage"].UsefulRate)
	assert.Equal(t, 0.8, byKind["triage"].AvgConfidence)
	assert.Equal(t, 0.0, byKind["summarize"].UsefulRate)
	assert.Equal(t, 2, byKind["analyze"].Count)
	assert.Equal(t, 0.5, byKind["analyze"].UsefulRate)
	assert.Equal(t, 0.5, byKind["analyze"].AcceptedRate)
	assert.Equal(t, 0.925, byKind["analyze"].AvgConfidence)
}

func TestEvaluate_ConfidenceCalibration(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)
	require.Len(t, report.ConfidenceCalibration, 5)

	// score=80/90/95 → [80-100] 桶（idx 4）；score=40 → [40-60] 桶（idx 2）
	high := report.ConfidenceCalibration[4]
	assert.Equal(t, "80-100%", high.Bucket)
	assert.Equal(t, 3, high.Count)
	assert.InDelta(t, 2.0/3.0, high.UsefulRate, 0.0001)
	// |0.6667 - 0.9| = 0.2333
	assert.InDelta(t, 0.2333, high.CalibrationError, 0.0001)

	mid := report.ConfidenceCalibration[2]
	assert.Equal(t, "40-60%", mid.Bucket)
	assert.Equal(t, 1, mid.Count)
	assert.Equal(t, 0.0, mid.UsefulRate)
	assert.Equal(t, 0.5, mid.Midpoint)
}

func TestEvaluate_TenantIsolation(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	// 租户2只有一条 useful=true 的 triage，不应看到租户1的数据。
	report, err := svc.Evaluate(context.Background(), 2, 30)
	require.NoError(t, err)
	assert.Equal(t, 1, report.TotalFeedback)
	assert.Equal(t, 1.0, report.UsefulRate)
	assert.Equal(t, 1.0, report.AvgConfidence)
}

func TestEvaluate_EmptyState(t *testing.T) {
	_, svc := newTelemetryTestDB(t)
	// 空库：无样本时不返回误导性数值
	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)
	assert.False(t, report.HasData)
	assert.Equal(t, 0, report.TotalFeedback)
	assert.Equal(t, 0.0, report.HealthScore)
	assert.Empty(t, report.ByScenario)
	// Sprint C：bySkill 字段在空库时仍以空数组返回（与 ByScenario 对齐）。
	assert.NotNil(t, report.BySkill)
	assert.Empty(t, report.BySkill)
}

// newRegistryWithFakeSkill 构造一个只包含单个 fake Skill 的最小 SkillRegistry，
// 用于评估器“带 SkillRegistry”的场景。Skill 的 Manifest.Category 决定 IsPilot。
func newRegistryWithFakeSkill(code, name, category string) *SkillRegistry {
	reg := NewSkillRegistry()
	fs := &fakeSkill{code: code, name: name, category: category}
	// 直接绕开 register 的并发保护（测试中调用次序固定）。
	reg.skills[code] = fs
	return reg
}

// fakeSkill 是 SkillRegistry 测试用的最小实现：仅返回 Manifest/Name。
type fakeSkill struct {
	code     string
	name     string
	category string
}

func (f *fakeSkill) Code() string   { return f.code }
func (f *fakeSkill) Name() string   { return f.name }
func (f *fakeSkill) Tags() []string { return nil }
func (f *fakeSkill) Manifest() SkillManifest {
	return SkillManifest{Name: f.code, Version: "v1", Category: f.category}
}
func (f *fakeSkill) GetMetrics() SkillMetrics                                      { return SkillMetrics{} }
func (f *fakeSkill) Validate(_ interface{}) error                                  { return nil }
func (f *fakeSkill) Execute(_ context.Context, _ interface{}) (interface{}, error) { return nil, nil }

func TestEvaluate_BySkillDimension_WithoutRegistry(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	// 未注入 SkillRegistry：bySkill 仍以 skillCode 为 key 聚合。
	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)
	// 4 条样本 (triage/summarize/analyze/analyze) 分别映射到
	// ai.triage / ai.summarize / ai.analyze。租户2的 1 条 triage 同映射为 ai.triage，
	// 但只计租户1 的 4 条。
	require.Len(t, report.BySkill, 3)
	byCode := map[string]AISkillEval{}
	for _, sk := range report.BySkill {
		byCode[sk.SkillCode] = sk
	}
	// ai.triage：1 条 useful，accepted 0；usefulRate=1.0，acceptedRate=0.0
	triage, ok := byCode["ai.triage"]
	require.True(t, ok, "ai.triage should be present")
	assert.Equal(t, 1, triage.Count)
	assert.Equal(t, 1.0, triage.UsefulRate)
	assert.Equal(t, 0.0, triage.AcceptedRate)
	assert.Equal(t, 0.8, triage.AvgConfidence)
	assert.Equal(t, false, triage.IsPilot) // 未注入 registry → IsPilot=false
	// ai.analyze：2 条，usefulRate=0.5
	analyze, ok := byCode["ai.analyze"]
	require.True(t, ok)
	assert.Equal(t, 2, analyze.Count)
	assert.Equal(t, 0.5, analyze.UsefulRate)
	assert.Equal(t, 0.5, analyze.AcceptedRate)
	assert.Equal(t, 0.925, analyze.AvgConfidence)
	// HealthScore = 0.5*40 + 0.5*30 + (1-|0.5-0.925|)*30 = 20+15+(0.575*30) = 52.25
	assert.Equal(t, 52.25, analyze.HealthScore)
}

func TestEvaluate_BySkillDimension_WithRegistry(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	// 注入一个含 pilot 分类的 fake ai.triage。
	reg := newRegistryWithFakeSkill("ai.triage", "Triage Skill", "pilot")
	svc.SetSkillRegistry(reg)

	report, err := svc.Evaluate(context.Background(), 1, 30)
	require.NoError(t, err)

	byCode := map[string]AISkillEval{}
	for _, sk := range report.BySkill {
		byCode[sk.SkillCode] = sk
	}
	// 找到的 ai.triage 应当带上 Name + IsPilot=true
	triage, ok := byCode["ai.triage"]
	require.True(t, ok)
	assert.Equal(t, "Triage Skill", triage.SkillName)
	assert.Equal(t, true, triage.IsPilot)

	// ai.analyze 未注册：SkillName 为空，IsPilot=false
	analyze, ok := byCode["ai.analyze"]
	require.True(t, ok)
	assert.Empty(t, analyze.SkillName)
	assert.Equal(t, false, analyze.IsPilot)
}

func TestEvaluate_ScenarioToSkillCodeMapping(t *testing.T) {
	// 验证 AIScenarioEval.Kind → SkillCode 的稳定映射。
	cases := map[string]string{
		"triage":        "ai.triage",
		"summarize":     "ai.summarize",
		"analyze":       "ai.analyze",
		"rag_search":    "ai.knowledge_search",
		"chat":          "ai.chat",
		"analytics":     "ai.analytics",
		"prediction":    "ai.trend_prediction",
		"create_ticket": "ai.create_ticket",
		"agent_tool":    "ai.agent_tool",
		"":              "",
		"custom_kind":   "ai.custom_kind", // 未识别 kind 走 "ai." + 原 kind
	}
	for kind, want := range cases {
		assert.Equal(t, want, scenarioToSkillCode(kind), "kind=%q", kind)
	}
}

func TestListAIAuditLogs_PaginationAndNotes(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	entries, total, err := svc.ListAuditLogs(context.Background(), 1, 1, 20, "", 90)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
	// 最新在前：ticket_2 的 analyze 记录
	assert.Equal(t, "analyze", entries[0].Scenario)
	assert.False(t, entries[0].Accepted)
	assert.Equal(t, 0.95, entries[0].Confidence)
	assert.Equal(t, "v1", entries[0].PromptVersion)
	assert.Equal(t, "doubao-pro", entries[0].Model)
	require.NotNil(t, entries[0].Suggestion)
	assert.Equal(t, "network", entries[0].Suggestion["category"])
	assert.Equal(t, "ticket_2", entries[0].InputRef)
	// notes 原文保留
	assert.NotEmpty(t, entries[0].Notes)
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(entries[0].Notes), &raw))
	assert.Equal(t, "v1", raw["prompt_version"])
}

func TestListAIAuditLogs_KindFilterAndPaging(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	// kind 过滤：analyze 2 条
	entries, total, err := svc.ListAuditLogs(context.Background(), 1, 1, 1, "analyze", 90)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 1)

	// 不存在的 kind → 0 条且无错误
	entries, total, err = svc.ListAuditLogs(context.Background(), 1, 1, 20, "triage", 90)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)

	// 租户隔离：租户2 无审计记录（其反馈 item_type 为空）
	entries, total, err = svc.ListAuditLogs(context.Background(), 2, 1, 20, "", 90)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestListAIAuditLogs_TimeWindow(t *testing.T) {
	db, svc := newTelemetryTestDB(t)
	seedStandardEvaluationFixture(t, db)

	// lookbackDays=0 非法值 → 回退 90 天，仍能查到
	entries, total, err := svc.ListAuditLogs(context.Background(), 1, 1, 20, "", 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, entries, 2)

	// 极端小的窗口（1 天前边界，样本都在今天）仍能查到；构造过期样本后应被排除
	_, err = db.Exec(`
		INSERT INTO ai_feedbacks (created_at, tenant_id, user_id, request_id, kind, query, item_type, useful, score)
		VALUES (?, 1, 1, 'old', 'analyze', 'old_ticket', 'ai_audit', 1, 70)
	`, time.Now().AddDate(0, 0, -200))
	require.NoError(t, err)
	entries, total, err = svc.ListAuditLogs(context.Background(), 1, 1, 20, "", 90)
	require.NoError(t, err)
	assert.Equal(t, 2, total) // 200 天前的记录被时间窗口排除
	assert.Len(t, entries, 2)
}
