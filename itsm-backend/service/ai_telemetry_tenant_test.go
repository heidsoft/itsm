package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// attributedContext 走真实 RequestIDMiddleware 注入 request context，
// 再叠加租户上下文——与生产链（RequestIDMiddleware → TenantMiddleware → handler）一致。
func attributedContext(t *testing.T, tenantID int, reqID string) context.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	captured := make(chan context.Context, 1)
	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.GET("/", func(c *gin.Context) { captured <- c.Request.Context() })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", reqID)
	req = req.WithContext(tenantctx.WithTenantID(req.Context(), tenantID))
	r.ServeHTTP(httptest.NewRecorder(), req)

	return <-captured
}

// TestLLMObserver_WritesTenantAndRequestAttribution 锁定晋升条件之一：
// 网关调用必须把 ctx 中的 tenant/request 归属落到 ai_llm_calls。
// 修复前 Observe 签名不带 ctx，行内没有任何归因字段。
func TestLLMObserver_WritesTenantAndRequestAttribution(t *testing.T) {
	db, _ := newTelemetryTestDB(t)
	observer := NewLLMObserver(db, zaptest.NewLogger(t).Sugar())

	ctx := attributedContext(t, 42, "req-abc")
	observer.Observe(ctx, "doubao", "model-x", 120, 900*time.Millisecond, nil)

	var tenantID int
	var requestID string
	var tokens int
	var latencyMs int
	var success bool
	require.NoError(t, db.QueryRow(
		`SELECT tenant_id, request_id, tokens, latency_ms, success FROM ai_llm_calls`,
	).Scan(&tenantID, &requestID, &tokens, &latencyMs, &success))
	assert.Equal(t, 42, tenantID)
	assert.Equal(t, "req-abc", requestID)
	assert.Equal(t, 120, tokens)
	assert.Equal(t, 900, latencyMs)
	assert.True(t, success)
}

// TestLLMObserver_NoTenantKeepsSentinelZero 无租户上下文的系统调用落哨兵 0，
// 不得伪造归属；同时调用方 ctx 已取消也不得阻止观测行写入（WithoutCancel）。
func TestLLMObserver_NoTenantKeepsSentinelZero(t *testing.T) {
	db, _ := newTelemetryTestDB(t)
	observer := NewLLMObserver(db, zaptest.NewLogger(t).Sugar())

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	observer.Observe(cancelled, "doubao", "model-x", 10, 50*time.Millisecond, nil)

	var count int
	var tenantID int
	var requestID string
	require.NoError(t, db.QueryRow(`SELECT COUNT(*), MIN(tenant_id), MIN(request_id) FROM ai_llm_calls`).
		Scan(&count, &tenantID, &requestID))
	assert.Equal(t, 1, count, "已取消的 ctx 不得吞掉观测行")
	assert.Equal(t, 0, tenantID, "无租户上下文必须落哨兵 0，而不是报错或伪造归属")
	assert.Empty(t, requestID)
}

// TestAggregateTenantLLMStats_ExcludesSentinelAndOtherTenants 租户聚合口径：
// 只统计本租户窗口内行；tenant_id=0 的历史/系统行与其他租户不得混入。
func TestAggregateTenantLLMStats_ExcludesSentinelAndOtherTenants(t *testing.T) {
	db, _ := newTelemetryTestDB(t)
	now := time.Now()
	seed := []struct {
		createdAt time.Time
		tenantID  int
		tokens    int
		latencyMs int
		success   bool
	}{
		{now.Add(-1 * time.Minute), 7, 100, 1000, true},
		{now.Add(-2 * time.Minute), 7, 200, 3000, false},
		{now.Add(-1 * time.Minute), 8, 999, 500, true},      // 其他租户
		{now.Add(-1 * time.Minute), 0, 8888, 4000, true},    // 未归因哨兵
		{now.Add(-30 * 24 * time.Hour), 7, 5555, 100, true}, // 窗口外
	}
	for _, row := range seed {
		_, err := db.Exec(
			`INSERT INTO ai_llm_calls (created_at, tenant_id, request_id, provider, model, tokens, latency_ms, success)
			 VALUES ($1, $2, '', 'p', 'm', $3, $4, $5)`,
			row.createdAt, row.tenantID, row.tokens, row.latencyMs, row.success,
		)
		require.NoError(t, err)
	}

	repo := newAITelemetryRepository(db)
	stats, err := repo.AggregateTenantLLMStats(context.Background(), 7, now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 2, stats.CallCount)
	assert.Equal(t, 1, stats.SuccessCount)
	assert.Equal(t, 300, stats.TotalTokens)
	assert.InDelta(t, 2.0, stats.AvgLatencySeconds, 1e-9)

	// 无任何归因行的租户：全零 + 无错误，由调用方按 responseTimeAvailable=false 如实降级。
	empty, err := repo.AggregateTenantLLMStats(context.Background(), 99, now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, empty.CallCount)
	assert.Equal(t, 0.0, empty.AvgLatencySeconds)
}

// TestGateway_PassesContextToObserver 网关契约：observe 必须拿到发起调用的 ctx，
// 否则租户归因在网关层就被丢掉。
func TestGateway_PassesContextToObserver(t *testing.T) {
	provider := &MockLLMProvider{}
	observer := &MockObserver{}
	gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, observer, "mock")

	ctx := tenantctx.WithTenantID(context.Background(), 11)
	_, err := gateway.Chat(ctx, "m", []LLMMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	require.Len(t, observer.Records, 1)
	assert.Equal(t, 11, observer.Records[0].TenantID, "观察者必须能从请求 ctx 读出租户归因")
}
