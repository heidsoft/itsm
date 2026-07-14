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
