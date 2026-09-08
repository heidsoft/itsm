package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mock 实现 ====================

// MockLLMProvider 模拟 LLM Provider
type MockLLMProvider struct {
	mu          sync.Mutex
	ShouldError bool
	ErrorMsg    string
	Response    string
	CallCount   int
}

func (m *MockLLMProvider) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	m.mu.Lock()
	m.CallCount++
	m.mu.Unlock()
	if m.ShouldError {
		return "", errors.New(m.ErrorMsg)
	}
	return m.Response, nil
}

func (m *MockLLMProvider) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CallCount
}

// MockTokenLimiter 模拟 Token Limiter
type MockTokenLimiter struct {
	mu          sync.Mutex
	ShouldAllow bool
	CheckCount  int
	LastTokens  int
}

func (m *MockTokenLimiter) Allow(nTokens int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CheckCount++
	m.LastTokens = nTokens
	return m.ShouldAllow
}

func (m *MockTokenLimiter) Snapshot() (checkCount, lastTokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CheckCount, m.LastTokens
}

// MockObserver 模拟 Observer
type MockObserver struct {
	mu      sync.Mutex
	Records []Observation
}

type Observation struct {
	Provider string
	Model    string
	Tokens   int
	Latency  time.Duration
	Err      error
}

func (m *MockObserver) Observe(provider string, model string, tokens int, latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Records = append(m.Records, Observation{
		Provider: provider,
		Model:    model,
		Tokens:   tokens,
		Latency:  latency,
		Err:      err,
	})
}

func (m *MockObserver) Snapshot() []Observation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Observation(nil), m.Records...)
}

// ==================== LLMGateway 基本测试 ====================

func TestNewLLMGateway(t *testing.T) {
	gateway := NewLLMGateway(nil, nil, nil, "test-provider")
	require.NotNil(t, gateway)
	assert.Equal(t, "test-provider", gateway.providerName)
}

func TestLLMGateway_Chat_Success(t *testing.T) {
	// 准备 mock
	provider := &MockLLMProvider{
		Response: "Hello, world!",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: true,
	}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "Hello"},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", resp)
	assert.Equal(t, 1, provider.Calls())
	checkCount, _ := limiter.Snapshot()
	assert.Equal(t, 1, checkCount)
	records := observer.Snapshot()
	assert.Len(t, records, 1)
	assert.Equal(t, "openai", records[0].Provider)
	assert.Equal(t, "gpt-4", records[0].Model)
}

func TestLLMGateway_Chat_ProviderError(t *testing.T) {
	provider := &MockLLMProvider{
		ShouldError: true,
		ErrorMsg:    "API error",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: true,
	}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "Hello"},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
	assert.Empty(t, resp)
	records := observer.Snapshot()
	assert.Len(t, records, 1)
	assert.NotNil(t, records[0].Err)
}

func TestLLMGateway_Chat_RateLimited(t *testing.T) {
	provider := &MockLLMProvider{
		Response: "Hello",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: false, // 拒绝请求
	}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "Hello"},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.Error(t, err)
	// 返回值是空字符串
	assert.Empty(t, resp)
	// 验证返回的是 RateLimitError
	var rateLimitErr *RateLimitError
	assert.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 0, provider.Calls()) // provider 不应被调用
	assert.Len(t, observer.Snapshot(), 1)
}

func TestLLMGateway_Chat_NilLimiter(t *testing.T) {
	provider := &MockLLMProvider{
		Response: "Hello",
	}
	observer := &MockObserver{}

	// limiter 为 nil
	gateway := NewLLMGateway(provider, nil, observer, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "Hello"},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.NoError(t, err)
	assert.Equal(t, "Hello", resp)
	assert.Equal(t, 1, provider.Calls())
}

func TestLLMGateway_Chat_NilObserver(t *testing.T) {
	provider := &MockLLMProvider{
		Response: "Hello",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: true,
	}

	// observer 为 nil
	gateway := NewLLMGateway(provider, limiter, nil, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "Hello"},
	}

	// 不应 panic
	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.NoError(t, err)
	assert.Equal(t, "Hello", resp)
}

func TestLLMGateway_Chat_TokenCalculation(t *testing.T) {
	provider := &MockLLMProvider{
		Response: "Hello",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: true,
	}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	// 测试 token 计算
	messages := []LLMMessage{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "What is 2+2?"},
		{Role: "assistant", Content: "4"},
	}

	_, err := gateway.Chat(context.Background(), "gpt-4", messages)
	require.NoError(t, err)

	// 验证 limiter 收到了正确的 token 数量
	// 每个字符约等于 1/4 token
	// "You are a helpful assistant" = 28 字符 ≈ 7 tokens
	// "What is 2+2?" = 13 字符 ≈ 3 tokens
	// "4" = 1 字符 ≈ 0 tokens
	// 总计约 10 tokens
	_, lastTokens := limiter.Snapshot()
	assert.GreaterOrEqual(t, lastTokens, 7)
}

// ==================== Token Limiter 测试 ====================

func TestNewFixedWindowLimiter(t *testing.T) {
	limiter := NewFixedWindowLimiter(100)
	require.NotNil(t, limiter)
	assert.Equal(t, 100, limiter.capacity)
}

func TestFixedWindowLimiter_Allow_WithinCapacity(t *testing.T) {
	limiter := NewFixedWindowLimiter(100)

	assert.True(t, limiter.Allow(50))
	assert.True(t, limiter.Allow(100))
	assert.True(t, limiter.Allow(0))
}

func TestFixedWindowLimiter_Allow_ExceedCapacity(t *testing.T) {
	limiter := NewFixedWindowLimiter(100)

	assert.False(t, limiter.Allow(101))
	assert.False(t, limiter.Allow(200))
}

func TestFixedWindowLimiter_ZeroCapacity(t *testing.T) {
	limiter := NewFixedWindowLimiter(0)

	// capacity=0 时，Allow(0) 返回 true，Allow(1) 返回 false
	assert.True(t, limiter.Allow(0))  // 0 <= 0
	assert.False(t, limiter.Allow(1)) // 1 > 0
}

// ==================== NoopObserver 测试 ====================

func TestNoopObserver_Observe_NoPanic(t *testing.T) {
	observer := NoopObserver{}

	// 不应 panic
	observer.Observe("provider", "model", 100, time.Millisecond, nil)
	observer.Observe("provider", "model", 100, time.Millisecond, errors.New("error"))
}

// ==================== RateLimitError 测试 ====================

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{Message: "rate limited"}

	assert.Equal(t, "rate limited", err.Error())
}

func TestRateLimitError_Is(t *testing.T) {
	err := &RateLimitError{Message: "rate limited"}

	// RateLimitError 没有实现 Is 方法，但 Error() 正确
	assert.Equal(t, "rate limited", err.Error())
	// 可以直接比较错误类型
	assert.Equal(t, ErrRateLimited.Message, err.Message)
}

// ==================== Token 估算测试 ====================

func TestLLMGateway_TokenEstimation(t *testing.T) {
	testCases := []struct {
		name       string
		messages   []LLMMessage
		expectedGt int // 至少应该有这么多 tokens
	}{
		{
			name: "空消息",
			messages: []LLMMessage{
				{Role: "user", Content: ""},
			},
			expectedGt: 0,
		},
		{
			name: "短消息",
			messages: []LLMMessage{
				{Role: "user", Content: "Hi"},
			},
			expectedGt: 0, // 2字符 / 4 = 0
		},
		{
			name: "中等消息",
			messages: []LLMMessage{
				{Role: "user", Content: "Hello, how are you today?"},
			},
			expectedGt: 5, // 至少 5 tokens (20字符 / 4 = 5)
		},
		{
			name: "长消息",
			messages: []LLMMessage{
				{Role: "user", Content: "This is a longer message that contains more text for testing token estimation logic."},
			},
			expectedGt: 15, // 至少 15 tokens
		},
		{
			name: "多消息",
			messages: []LLMMessage{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Question?"},
				{Role: "assistant", Content: "Answer"},
			},
			expectedGt: 3, // 至少 3 tokens
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MockLLMProvider{Response: "ok"}
			limiter := &MockTokenLimiter{ShouldAllow: true}
			observer := &MockObserver{}

			gateway := NewLLMGateway(provider, limiter, observer, "test")

			_, err := gateway.Chat(context.Background(), "model", tt.messages)
			require.NoError(t, err)

			_, lastTokens := limiter.Snapshot()
			assert.GreaterOrEqual(t, lastTokens, tt.expectedGt)
		})
	}
}

// ==================== 并发安全测试 ====================

func TestLLMGateway_ConcurrentCalls(t *testing.T) {
	provider := &MockLLMProvider{
		Response: "response",
	}
	limiter := &MockTokenLimiter{
		ShouldAllow: true,
	}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	messages := []LLMMessage{
		{Role: "user", Content: "test"},
	}

	// 并发调用
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := gateway.Chat(context.Background(), "gpt-4", messages)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, provider.Calls())
}

// ==================== 错误处理边界测试 ====================

func TestLLMGateway_EmptyMessages(t *testing.T) {
	provider := &MockLLMProvider{Response: "ok"}
	limiter := &MockTokenLimiter{ShouldAllow: true}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	// 空消息列表
	resp, err := gateway.Chat(context.Background(), "gpt-4", []LLMMessage{})

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.Equal(t, 1, provider.Calls())
}

func TestLLMGateway_LongContent(t *testing.T) {
	provider := &MockLLMProvider{Response: "ok"}
	limiter := &MockTokenLimiter{ShouldAllow: true}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	// 长内容
	longContent := make([]byte, 10000)
	for i := range longContent {
		longContent[i] = 'a'
	}

	messages := []LLMMessage{
		{Role: "user", Content: string(longContent)},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	_, lastTokens := limiter.Snapshot()
	assert.Greater(t, lastTokens, 2000) // 10000 / 4 = 2500
}

func TestLLMGateway_UnicodeContent(t *testing.T) {
	provider := &MockLLMProvider{Response: "ok"}
	limiter := &MockTokenLimiter{ShouldAllow: true}
	observer := &MockObserver{}

	gateway := NewLLMGateway(provider, limiter, observer, "openai")

	// Unicode 内容（每个字符算一个 rune）
	messages := []LLMMessage{
		{Role: "user", Content: "你好世界！这是测试。🎉"},
	}

	resp, err := gateway.Chat(context.Background(), "gpt-4", messages)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// ==================== SupportsToolCalling 能力闸门（回归 bug：minimax 伪工具调用）====================

// toolCapableProvider 同时实现 Chat 与 ChatStreamWithTools，用于验证能力探测返回 true。
type toolCapableProvider struct {
	MockLLMProvider
	toolCallReceived int
}

func (p *toolCapableProvider) ChatStreamWithTools(_ context.Context, _ string, _ []LLMMessage, _ []LLMTool, callback func(string), onToolCalls func([]LLMToolCall)) error {
	if callback != nil {
		callback("partial answer")
	}
	if onToolCalls != nil {
		p.toolCallReceived++
		onToolCalls([]LLMToolCall{{ID: "call_1", Name: "list_tickets", Arguments: "{}"}})
	}
	return nil
}

// TestLLMGateway_SupportsToolCalling_ReflectsProviderCapability 验证 provider 能力探测：
//   - 未实现 ChatStreamWithTools 的 provider。如 MiniMax。必须返回 false，调用方据此不注入
//     tool-driven system prompt，避免模型在文本中假装“正在调用 list_tickets 工具...”但永远拿不到结果。
//   - 实现了 ChatStreamWithTools 的 provider（如 OpenAI）必须返回 true。
//   - nil gateway 与 nil provider 不能 panic，返回 false。
func TestLLMGateway_SupportsToolCalling_ReflectsProviderCapability(t *testing.T) {
	t.Run("plain provider without tools capability returns false", func(t *testing.T) {
		provider := &MockLLMProvider{Response: "ok"}
		gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, &MockObserver{}, "minimax")
		assert.False(t, gateway.SupportsToolCalling(), "MiniMax 类仅 Chat provider 应该被判为不支持 tool calling")
	})

	t.Run("tool-capable provider returns true", func(t *testing.T) {
		provider := &toolCapableProvider{}
		provider.Response = "ok"
		gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, &MockObserver{}, "openai")
		assert.True(t, gateway.SupportsToolCalling(), "OpenAI 类 provider 必须被判为支持 tool calling")
	})

	t.Run("nil gateway and nil provider are safe", func(t *testing.T) {
		var nilGateway *LLMGateway
		assert.False(t, nilGateway.SupportsToolCalling(), "nil gateway 必须 false 且不能 panic")
		g := NewLLMGateway(nil, nil, nil, "")
		assert.False(t, g.SupportsToolCalling(), "provider 为 nil 时必须 false")
	})

	t.Run("ChatStreamWithTools actually invokes provider on capability hit", func(t *testing.T) {
		provider := &toolCapableProvider{}
		gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, &MockObserver{}, "openai")
		var streamed string
		var gotCalls []LLMToolCall
		err := gateway.ChatStreamWithTools(
			context.Background(), "gpt-4",
			[]LLMMessage{{Role: "user", Content: "hi"}},
			[]LLMTool{{Name: "list_tickets"}},
			func(s string) { streamed += s },
			func(tcs []LLMToolCall) { gotCalls = tcs },
		)
		require.NoError(t, err)
		assert.Equal(t, "partial answer", streamed)
		require.Len(t, gotCalls, 1)
		assert.Equal(t, "list_tickets", gotCalls[0].Name)
		assert.Equal(t, 1, provider.toolCallReceived)
	})
}

// ==================== 重试语义测试（2026-09-08 UAT Q-2 根因修复） ====================

// flakyProvider 前几次返回瞬时错误，之后成功——模拟网络抖动/上游 5xx。
type flakyProvider struct {
	MockLLMProvider
	mu          sync.Mutex
	failures    int
	errToReturn error
}

func (p *flakyProvider) Chat(_ context.Context, _ string, _ []LLMMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CallCount++
	if p.CallCount <= p.failures {
		return "", p.errToReturn
	}
	return p.Response, nil
}

func (p *flakyProvider) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.CallCount
}

func TestLLMGateway_RetryTransientErrors(t *testing.T) {
	t.Run("network error retried then succeeds (transient)", func(t *testing.T) {
		provider := &flakyProvider{
			failures:        2,
			errToReturn:     errors.New("MiniMax API error: connection reset by peer"),
			MockLLMProvider: MockLLMProvider{Response: "ok after retry"},
		}
		observer := &MockObserver{}
		gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, observer, "minimax")

		out, err := gateway.Chat(context.Background(), "m", []LLMMessage{{Role: "user", Content: "hi"}})
		require.NoError(t, err)
		assert.Equal(t, "ok after retry", out)
		assert.Equal(t, 3, provider.calls(), "首次 + 2 次重试")
		// 观测语义：最终结果只 Observe 一次
		require.Len(t, observer.Records, 1)
		assert.NoError(t, observer.Records[0].Err)
	})

	t.Run("5xx retried, 401 auth error NOT retried", func(t *testing.T) {
		// 5xx：可重试
		p5xx := &flakyProvider{
			failures:        1,
			errToReturn:     errors.New("MiniMax API error: status 502, message: bad gateway"),
			MockLLMProvider: MockLLMProvider{Response: "ok"},
		}
		gw5xx := NewLLMGateway(p5xx, nil, nil, "minimax")
		_, err := gw5xx.Chat(context.Background(), "m", []LLMMessage{{Role: "user", Content: "hi"}})
		require.NoError(t, err)
		assert.Equal(t, 2, p5xx.calls(), "5xx 应重试 1 次后成功")

		// 401：不可重试（重试只会重复失败拖长等待）
		p401 := &flakyProvider{
			failures:        3,
			errToReturn:     errors.New("OpenAI API error: error, status code: 401"),
			MockLLMProvider: MockLLMProvider{Response: "ok"},
		}
		gw401 := NewLLMGateway(p401, nil, nil, "openai")
		_, err = gw401.Chat(context.Background(), "m", []LLMMessage{{Role: "user", Content: "hi"}})
		require.Error(t, err)
		assert.Equal(t, 1, p401.calls(), "401 必须立即返回，不重试")
	})

	t.Run("persistent transient failure exhausts retries and fails", func(t *testing.T) {
		provider := &flakyProvider{
			failures:    99,
			errToReturn: errors.New("MiniMax API error: status 503, message: overloaded"),
		}
		observer := &MockObserver{}
		gateway := NewLLMGateway(provider, nil, observer, "minimax")

		_, err := gateway.Chat(context.Background(), "m", []LLMMessage{{Role: "user", Content: "hi"}})
		require.Error(t, err)
		assert.Equal(t, 1+llmMaxRetries, provider.calls())
		require.Len(t, observer.Records, 1, "重试中间态不 Observe，只记最终结果")
		assert.Error(t, observer.Records[0].Err)
	})

	t.Run("caller context cancelled during backoff stops retrying", func(t *testing.T) {
		provider := &flakyProvider{
			failures:    99,
			errToReturn: errors.New("MiniMax API error: status 500"),
		}
		gateway := NewLLMGateway(provider, nil, nil, "minimax")

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		_, err := gateway.Chat(ctx, "m", []LLMMessage{{Role: "user", Content: "hi"}})
		require.Error(t, err)
		assert.LessOrEqual(t, provider.calls(), 2, "ctx 取消后应停止重试")
	})
}

// ==================== ChatStreamWithTools 门禁/观测补齐测试 ====================

func TestLLMGateway_ChatStreamWithTools_GateAndObserve(t *testing.T) {
	t.Run("tool path respects token limiter (previously bypassed)", func(t *testing.T) {
		provider := &toolCapableProvider{}
		limiter := &MockTokenLimiter{ShouldAllow: false}
		gateway := NewLLMGateway(provider, limiter, &MockObserver{}, "openai")

		err := gateway.ChatStreamWithTools(
			context.Background(), "gpt-4",
			[]LLMMessage{{Role: "user", Content: "hi"}},
			[]LLMTool{{Name: "list_tickets"}},
			func(string) {}, func([]LLMToolCall) {},
		)
		require.ErrorIs(t, err, ErrRateLimited)
		assert.Equal(t, 0, provider.toolCallReceived, "限流时不得触达 provider")
	})

	t.Run("tool path emits exactly one observation (previously zero)", func(t *testing.T) {
		provider := &toolCapableProvider{}
		observer := &MockObserver{}
		gateway := NewLLMGateway(provider, &MockTokenLimiter{ShouldAllow: true}, observer, "openai")

		err := gateway.ChatStreamWithTools(
			context.Background(), "gpt-4",
			[]LLMMessage{{Role: "user", Content: "hi"}},
			nil, func(string) {}, func([]LLMToolCall) {},
		)
		require.NoError(t, err)
		require.Len(t, observer.Records, 1, "工具调用路径必须进 ai_llm_calls 观测")
		assert.Equal(t, "openai", observer.Records[0].Provider)
	})
}
