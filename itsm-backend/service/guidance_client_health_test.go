package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A5：sidecar 的 /health 永远返回 200，配置错误只体现在响应体里。
// 旧 HealthCheck 只看状态码，于是 LLM_API_KEY 没配时后端仍认为 sidecar
// 健康、Keyword 降级结果被当成模型决策消费。这里锁住三种响应形态。
func TestGuidanceClient_HealthCheck_ReadsLLMConfigured(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		errContain string
	}{
		{
			name:    "已配置 → 健康",
			body:    `{"status":"healthy","llm":{"provider":"openai","model":"gpt-4","configured":true,"reason":""}}`,
			wantErr: false,
		},
		{
			name:       "凭据缺失 → 必须报错",
			body:       `{"status":"degraded","llm":{"provider":"openai","model":"gpt-4","configured":false,"reason":"LLM_API_KEY not configured"}}`,
			wantErr:    true,
			errContain: "not configured",
		},
		{
			name:       "status=degraded 但 configured 缺省 → 仍按不健康处理",
			body:       `{"status":"degraded","llm":{"provider":"openai"}}`,
			wantErr:    true,
			errContain: "degraded",
		},
		{
			name:    "旧版 sidecar（无 llm 对象）→ 兼容，不误报",
			body:    `{"status":"healthy","service":"itsm-ai-service","version":"1.0.0","llm":{"provider":"openai","model":"gpt-4o-mini"}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewGuidanceClient(srv.URL, zap.NewNop().Sugar())
			err := c.HealthCheck(context.Background())
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

// A4：sidecar 开启服务间鉴权后，Go 侧必须把 AI_SERVICE_TOKEN 放进
// X-Service-Token；否则 /triage 恒 401，分诊整条链路退化成关键词兜底。
func TestGuidanceClient_Triage_SendsServiceToken(t *testing.T) {
	t.Setenv("AI_SERVICE_TOKEN", "unit-test-token")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Service-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"category":"database","priority":"high","confidence":0.8,"explanation":"ok","method":"guidance"}`))
	}))
	defer srv.Close()

	c := NewGuidanceClient(srv.URL, zap.NewNop().Sugar())
	_, err := c.Triage(context.Background(), "MySQL 慢查询", "", 1)
	require.NoError(t, err)
	assert.Equal(t, "unit-test-token", got)
}

func TestGuidanceClient_Triage_NoTokenConfigured_SendsNoHeader(t *testing.T) {
	t.Setenv("AI_SERVICE_TOKEN", "")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Service-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"category":"general","priority":"low","confidence":0.5,"explanation":"ok","method":"guidance-fallback","degraded":true}`))
	}))
	defer srv.Close()

	c := NewGuidanceClient(srv.URL, zap.NewNop().Sugar())
	_, err := c.Triage(context.Background(), "普通咨询", "", 1)
	require.NoError(t, err)
	assert.Empty(t, got, "未配置令牌时不应发送空头，避免看起来已鉴权")
}
