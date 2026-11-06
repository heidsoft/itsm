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
