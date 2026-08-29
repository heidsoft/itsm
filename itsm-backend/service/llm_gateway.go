package service

import (
	"context"
	"time"
)

// LLMGateway abstracts multiple providers and basic observability/limits
type LLMGateway struct {
	provider     LLMProvider
	limiter      TokenLimiter
	observer     Observer
	providerName string
}

type LLMProvider interface {
	Chat(ctx context.Context, model string, messages []LLMMessage) (string, error)
}

// StreamingLLMProvider is an optional capability. Providers that implement
// ChatStream will be used for token-level streaming; otherwise the gateway
// falls back to a single-shot Chat call.
type StreamingLLMProvider interface {
	ChatStream(ctx context.Context, model string, messages []LLMMessage, callback func(string)) error
}

// ToolCallingStreamProvider is an optional capability: providers that support
// declaring tools (function calling) AND streaming. The provider receives the
// declared tools, streams text deltas through callback, and reports any tool
// calls the model requested through onToolCalls. Providers that do not
// implement this interface degrade gracefully: the gateway's
// ChatStreamWithTools falls back to a plain ChatStream with no tools declared.
type ToolCallingStreamProvider interface {
	ChatStreamWithTools(ctx context.Context, model string, messages []LLMMessage, tools []LLMTool, callback func(string), onToolCalls func([]LLMToolCall)) error
}

// LLMTool 声明可供模型调用的工具（OpenAI function calling 风格）。
type LLMTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// LLMToolCall 模型发起的一次工具调用。
type LLMToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 编码的参数
}

type LLMMessage struct {
	Role    string
	Content string
	// Tools 请求侧：本消息声明可用工具（通常挂载在 user/system 消息上）。
	Tools []LLMTool
	// ToolCalls 响应侧：assistant 消息携带模型请求调用的工具列表。
	ToolCalls []LLMToolCall
	// ToolCallID 工具结果消息：对应被执行的 LLMToolCall.ID（OpenAI "tool" role）。
	ToolCallID string
}

type TokenLimiter interface {
	Allow(nTokens int) bool
}

type Observer interface {
	Observe(provider string, model string, tokens int, latency time.Duration, err error)
}

func NewLLMGateway(p LLMProvider, l TokenLimiter, o Observer, providerName string) *LLMGateway {
	return &LLMGateway{provider: p, limiter: l, observer: o, providerName: providerName}
}

func (g *LLMGateway) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	start := time.Now()
	// naive tokens estimation
	tokens := 0
	for _, m := range messages {
		tokens += len([]rune(m.Content)) / 4
	}
	if g.limiter != nil && !g.limiter.Allow(tokens) {
		if g.observer != nil {
			g.observer.Observe("", model, tokens, time.Since(start), ErrRateLimited)
		}
		return "", ErrRateLimited
	}
	out, err := g.provider.Chat(ctx, model, messages)
	if g.observer != nil {
		g.observer.Observe(g.providerName, model, tokens, time.Since(start), err)
	}
	return out, err
}

// ChatStream streams tokens through the callback. If the underlying provider
// does not implement StreamingLLMProvider, it falls back to a single Chat call
// and emits the full response as one chunk. Callbacks may be invoked with
// empty strings; consumers should handle them gracefully.
func (g *LLMGateway) ChatStream(ctx context.Context, model string, messages []LLMMessage, callback func(string)) error {
	if callback == nil {
		callback = func(string) {}
	}
	start := time.Now()
	tokens := 0
	for _, m := range messages {
		tokens += len([]rune(m.Content)) / 4
	}
	if g.limiter != nil && !g.limiter.Allow(tokens) {
		if g.observer != nil {
			g.observer.Observe(g.providerName, model, tokens, time.Since(start), ErrRateLimited)
		}
		return ErrRateLimited
	}

	if streamer, ok := g.provider.(StreamingLLMProvider); ok {
		err := streamer.ChatStream(ctx, model, messages, callback)
		if g.observer != nil {
			g.observer.Observe(g.providerName, model, tokens, time.Since(start), err)
		}
		return err
	}

	// Fallback: run a normal Chat and emit the whole response as one chunk.
	out, err := g.provider.Chat(ctx, model, messages)
	if g.observer != nil {
		g.observer.Observe(g.providerName, model, tokens, time.Since(start), err)
	}
	if err != nil {
		return err
	}
	if out != "" {
		callback(out)
	}
	return nil
}

// ChatStreamWithTools declares tools and streams the reply. Text deltas are
// delivered through callback; if the model requests tool calls they are
// reported through onToolCalls (so the caller can execute them and continue
// the conversation). Providers without tool-calling support degrade to a plain
// ChatStream with no tools declared. Token limiting/observability behave the
// same as ChatStream.
func (g *LLMGateway) ChatStreamWithTools(ctx context.Context, model string, messages []LLMMessage, tools []LLMTool, callback func(string), onToolCalls func([]LLMToolCall)) error {
	if p, ok := g.provider.(ToolCallingStreamProvider); ok {
		return p.ChatStreamWithTools(ctx, model, messages, tools, callback, onToolCalls)
	}
	// 退化：忽略工具声明，走普通流式；模型不会返回工具调用。
	return g.ChatStream(ctx, model, messages, callback)
}

// Simple implementations
var ErrRateLimited = &RateLimitError{Message: "rate limited"}

type RateLimitError struct{ Message string }

func (e *RateLimitError) Error() string { return e.Message }

type FixedWindowLimiter struct{ capacity int }

func NewFixedWindowLimiter(capacity int) *FixedWindowLimiter {
	return &FixedWindowLimiter{capacity: capacity}
}
func (l *FixedWindowLimiter) Allow(n int) bool { return n <= l.capacity }

type NoopObserver struct{}

func (NoopObserver) Observe(_ string, _ string, _ int, _ time.Duration, _ error) {}
