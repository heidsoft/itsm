package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common/tenantctx"

	_ "github.com/mattn/go-sqlite3"
)

// 本文件守卫「租户 token 配额与单价口径」的性质。
//
// 背景：AI 晋升条件之一是「token 用量只有计量、没有配额告警与单价口径」。
// 设计上最要紧的三条诚实原则，每条都有对应测试锁：
//
//  1. 未配置配额 = 不限制，但如实报 quotaConfigured=false（不装有限额）；
//  2. 单价未配置 = estimatedCost=0 且 costConfigured=false（**绝不给猜的单价**：
//     成本数字会被拿去做预算决策，猜错比没有更糟）；
//  3. 配额评估失败 fail-open（配额是预算保护不是安全边界，DB 抖动不该让
//     AI 全体不可用）。

// newQuotaTestDB 建内存库并写入本月计量行。
func newQuotaTestDB(t *testing.T, tenantID int, tokens []int) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:quota_"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ai_llm_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP NOT NULL,
		provider TEXT, model TEXT, tokens INTEGER, latency_ms INTEGER, success BOOLEAN,
		tenant_id INTEGER NOT NULL DEFAULT 0, request_id TEXT)`)
	require.NoError(t, err)
	now := time.Now()
	for _, tk := range tokens {
		_, err := db.Exec(`INSERT INTO ai_llm_calls (created_at, provider, model, tokens, latency_ms, success, tenant_id)
			VALUES (?, 'p', 'm', ?, 10, 1, ?)`, now, tk, tenantID)
		require.NoError(t, err)
	}
	return db
}

// TestQuotaUnconfiguredIsHonest：未配置配额时不限制、不装有限额。
func TestQuotaUnconfiguredIsHonest(t *testing.T) {
	db := newQuotaTestDB(t, 1, []int{100, 200})
	q := NewAIQuotaService(db)                  // 无环境变量
	t.Setenv("ITSM_AI_TOKEN_MONTHLY_QUOTA", "") // 显式清空

	usage, err := q.Evaluate(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, usage.QuotaConfigured, "未配置就是未配置，不得装作有配额")
	assert.Equal(t, int64(300), usage.TokensUsed, "计量照实聚合")
	assert.Empty(t, usage.Alert)

	// Check 恒放行
	allowed, _, err := q.Check(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, allowed)
}

// TestQuotaWarningAtThreshold：达到预警阈值（默认 0.8）时 Alert=warning。
func TestQuotaWarningAtThreshold(t *testing.T) {
	db := newQuotaTestDB(t, 1, []int{850}) // 850/1000 = 85% > 80%
	q := NewAIQuotaService(db)
	q.SetQuota(1000)

	usage, err := q.Evaluate(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, usage.QuotaConfigured)
	assert.Equal(t, "warning", usage.Alert)
	assert.Contains(t, usage.AlertMessage, "85%")
	assert.Equal(t, int64(150), usage.TokensRemaining)
}

// TestQuotaExceededBlocks：超限后 Check 拒绝（网关会把它变成一次拒绝的调用）。
func TestQuotaExceededBlocks(t *testing.T) {
	db := newQuotaTestDB(t, 1, []int{1200}) // 1200 > 1000
	q := NewAIQuotaService(db)
	q.SetQuota(1000)

	usage, err := q.Evaluate(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "exceeded", usage.Alert)
	assert.Zero(t, usage.TokensRemaining, "剩余不得为负")

	allowed, _, err := q.Check(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, allowed, "超限必须拒绝")
}

// TestQuotaMonthWindowIsCurrentMonthOnly：上月用量不计入本月配额。
// 用真实日期边界：只插「上月」的行，本月应为 0。
func TestQuotaMonthWindowIsCurrentMonthOnly(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:quota_lastmonth?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE ai_llm_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT, created_at TIMESTAMP NOT NULL,
		provider TEXT, model TEXT, tokens INTEGER, latency_ms INTEGER, success BOOLEAN,
		tenant_id INTEGER NOT NULL DEFAULT 0, request_id TEXT)`)
	require.NoError(t, err)

	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	lastMonthEnd := thisMonthStart.Add(-time.Hour) // 上月最后一小时
	_, err = db.Exec(`INSERT INTO ai_llm_calls (created_at, provider, model, tokens, latency_ms, success, tenant_id)
		VALUES (?, 'p', 'm', 99999, 10, 1, 1)`, lastMonthEnd)
	require.NoError(t, err)

	q := NewAIQuotaService(db)
	q.SetQuota(100)
	usage, err := q.Evaluate(context.Background(), 1)
	require.NoError(t, err)
	assert.Zero(t, usage.TokensUsed, "上月用量不得计入本月配额窗口")
}

// TestQuotaFailOpenOnDBError：评估失败放行——配额是预算保护不是安全边界。
// 同时守网关行为：配额服务评估失败时调用必须照常放行（不是 429）。
func TestQuotaFailOpenOnDBError(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:quota_dead?mode=memory")
	require.NoError(t, err)
	// 不建表：查询必失败
	q := NewAIQuotaService(db)
	q.SetQuota(1)

	allowed, _, err := q.Check(context.Background(), 1)
	require.Error(t, err, "评估错误应透传给调用方记录")
	assert.True(t, allowed, "评估失败必须放行（fail-open）")

	// 网关层同样放行：DB 抖动不得让 AI 全体不可用。
	g := NewLLMGateway(&modelReportingProvider{model: "m", response: "ok"}, nil, nil, "p")
	g.SetAIQuotaService(q)
	ctx := tenantctx.WithTenantID(context.Background(), 1)
	out, gwErr := g.Chat(ctx, "", []LLMMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, gwErr, "配额评估失败时网关必须放行（fail-open）")
	assert.Equal(t, "ok", out)
}

// TestCostEstimateNeverGuessesPrice：单价未配置时给 0 + costConfigured=false，
// 绝不给一个猜的数字。
func TestCostEstimateNeverGuessesPrice(t *testing.T) {
	db := newQuotaTestDB(t, 1, nil)
	q := NewAIQuotaService(db) // 未设单价

	est := q.EstimateCost(50000)
	assert.False(t, est.CostConfigured)
	assert.Zero(t, est.EstimatedCost, "未配置单价时成本必须是 0，不是猜测值")
	assert.Equal(t, int64(50000), est.TotalTokens, "用量照实")
}

// TestCostEstimateComputesWhenConfigured：配置单价后按 千token×单价 计算。
func TestCostEstimateComputesWhenConfigured(t *testing.T) {
	db := newQuotaTestDB(t, 1, nil)
	t.Setenv("ITSM_AI_TOKEN_PRICE_PER_1K", "0.002")
	t.Setenv("ITSM_AI_TOKEN_PRICE_CURRENCY", "CNY")
	q := NewAIQuotaService(db) // 构造时读环境变量

	est := q.EstimateCost(50000)
	require.True(t, est.CostConfigured)
	assert.InDelta(t, 0.1, est.EstimatedCost, 0.0001, "50000 tokens × 0.002/千 = 0.1")
	assert.Equal(t, "CNY", est.Currency)
}

// TestGatewayQuotaBlocksAndObserves：网关在配额超限时拒绝调用，且被观察者
// 记录为一次失败的调用（配额拒绝不是静默消失，计量里必须留痕）。
func TestGatewayQuotaBlocksAndObserves(t *testing.T) {
	db := newQuotaTestDB(t, 1, []int{5000}) // 已用 5000 > 配额 1000
	q := NewAIQuotaService(db)
	q.SetQuota(1000)

	rec := &recordingObserver{}
	g := NewLLMGateway(&modelReportingProvider{model: "m", response: "ok"}, nil, rec, "p")
	g.SetAIQuotaService(q)

	ctx := tenantctx.WithTenantID(context.Background(), 1)
	_, err := g.Chat(ctx, "", []LLMMessage{{Role: "user", Content: "hi"}})
	require.ErrorIs(t, err, ErrQuotaExceeded)
	assert.Equal(t, 1, rec.calls, "配额拒绝必须被观察者记录，不得静默")
	assert.False(t, rec.lastSuccess)
}

// TestGatewayQuotaSkippedWithoutTenant：无租户归属（系统调用）不受配额限制。
func TestGatewayQuotaSkippedWithoutTenant(t *testing.T) {
	db := newQuotaTestDB(t, 1, []int{5000})
	q := NewAIQuotaService(db)
	q.SetQuota(1000)

	g := NewLLMGateway(&modelReportingProvider{model: "m", response: "ok"}, nil, nil, "p")
	g.SetAIQuotaService(q)

	// 不带租户上下文
	out, err := g.Chat(context.Background(), "", []LLMMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err, "系统调用不受租户配额限制")
	assert.Equal(t, "ok", out)
}

// TestGatewayWithoutQuotaServiceUnchanged：未注入配额服务时行为与旧版完全一致。
func TestGatewayWithoutQuotaServiceUnchanged(t *testing.T) {
	g := NewLLMGateway(&modelReportingProvider{model: "m", response: "ok"}, nil, nil, "p")
	out, err := g.Chat(context.Background(), "", []LLMMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "ok", out)
}

// ---- 测试辅助 ----

type recordingObserver struct {
	calls       int
	lastSuccess bool
}

func (r *recordingObserver) Observe(_ context.Context, _, _ string, _ int, _ time.Duration, err error) {
	r.calls++
	r.lastSuccess = err == nil
}
