package service

import (
	"context"
	"errors"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
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

// llmRetryPolicy 控制瞬时错误的自动重试。
//
// 背景（2026-09-08 UAT Q-2 根因）：prod 33 次 LLM 调用 6 次失败（18%），
// 其中 openai 3 次为"未配置 API Key 时Embedder/Provider 回退到 openai"的 401
// （不可重试，已在配置层修复），minimax 3 次为网络抖动/上游 5xx（可重试）。
// 网关层对可重试错误做有限退避重试，把演示/验收场景的失败率压到 5% 以内；
// 不可重试错误（401/403/400）立即返回，避免拖长用户等待。
const (
	llmMaxRetries       = 2                      // 首次调用外最多重试 2 次（共 3 次尝试）
	llmRetryBaseBackoff = 300 * time.Millisecond // 指数退避基值：300ms, 600ms
	llmRetryMaxLatency  = 45 * time.Second       // 累计耗时超阈值（慢失败）不再重试
)

// isTransientLLMError 判断错误是否值得重试：
//   - 网络层错误（连接拒绝/DNS/超时）——url.Error、context deadline（非调用方取消）
//   - HTTP 429（限流）与 5xx（上游故障）
//
// 不可重试：400（请求错误）、401/403（鉴权）——重试只会重复失败并拖长延迟。
func isTransientLLMError(err error) bool {
	if err == nil {
		return false
	}
	// go-openai APIError：按状态码分类（OpenAI/Azure/兼容网关都走这个类型）
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		switch code := apiErr.HTTPStatusCode; {
		case code == 429:
			return true
		case code >= 500:
			return true
		default:
			return false
		}
	}
	// MiniMax/Local provider 的错误消息里带 "status %d"（无结构化类型），按文本分类。
	// 注意顺序：先匹配 5xx/429，再排除 4xx 明确不可重试的状态。
	msg := err.Error()
	if strings.Contains(msg, "status 429") || strings.Contains(msg, "status 5") {
		return true
	}
	for _, code := range []string{"status 400", "status 401", "status 403", "status 404"} {
		if strings.Contains(msg, code) {
			return false
		}
	}
	// 网络/超时类：url.Error 包装（provider 层 %w 包装了 http 错误）或
	// connection reset/refused、EOF、broken pipe、timeout 等网络层关键词
	var urlErr interface{ Unwrap() error }
	if errors.As(err, &urlErr) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	for _, kw := range []string{
		"connection reset", "connection refused", "broken pipe",
		"EOF", "timeout", "TLS handshake", "no such host", "i/o timeout",
	} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// retryableChat 执行一次带退避重试的 provider.Chat。
// 观测语义：只在「最终结果」上 Observe 一次（重试中间态不写 ai_llm_calls，
// 保持调用计数与业务请求一一对应，避免重试把失败率/延迟统计进一步打乱）。
func (g *LLMGateway) retryableChat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	start := time.Now()
	var (
		out string
		err error
	)
	for attempt := 0; attempt <= llmMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := llmRetryBaseBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		out, err = g.provider.Chat(ctx, model, messages)
		if err == nil {
			return out, nil
		}
		// 调用方取消/超时：尊重 ctx，不重试
		if ctx.Err() != nil {
			return "", err
		}
		if !isTransientLLMError(err) {
			return "", err
		}
		// 慢失败（如 30s+ 超时）重试会成倍拖长用户等待：已耗时超阈值直接放弃
		if time.Since(start) > llmRetryMaxLatency {
			return "", err
		}
	}
	return "", err
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
	out, err := g.retryableChat(ctx, model, messages)
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
		// 流式调用不做自动重试：部分 token 可能已通过 callback 下发，
		// 重试会造成内容重复。瞬时错误交由调用方（前端断线重连）兜底。
		err := streamer.ChatStream(ctx, model, messages, callback)
		if g.observer != nil {
			g.observer.Observe(g.providerName, model, tokens, time.Since(start), err)
		}
		return err
	}

	// Fallback: run a normal Chat and emit the whole response as one chunk.
	out, err := g.retryableChat(ctx, model, messages)
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
		// 修复（2026-09-08）：原实现直接透传 provider，绕过 limiter 与 observer——
		// 工具调用路径的 LLM 用量既不受 token_cap 约束，也不进 ai_llm_calls，
		// 导致 /ai/metrics 的调用数/失败率/平均延迟全部失真（UAT Q-1/Q-2 的 33 次
		// 采样若混入工具路径则不可信）。现补齐与 ChatStream 相同的门禁与观测。
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
		err := p.ChatStreamWithTools(ctx, model, messages, tools, callback, onToolCalls)
		if g.observer != nil {
			g.observer.Observe(g.providerName, model, tokens, time.Since(start), err)
		}
		return err
	}
	// 退化：忽略工具声明，走普通流式；模型不会返回工具调用。
	return g.ChatStream(ctx, model, messages, callback)
}

// SupportsToolCalling reports whether the bound provider can declare tools and
// return real tool_calls in the response. Callers (e.g. AI ChatStream) use this
// to decide whether to inject tool-driven system prompt instructions: when the
// provider returns false, telling the model "you must call tools" causes it to
// fabricate text like "正在调用 list_tickets 工具..." without ever executing
// anything, which is exactly the bug we are guarding against.
func (g *LLMGateway) SupportsToolCalling() bool {
	if g == nil || g.provider == nil {
		return false
	}
	_, ok := g.provider.(ToolCallingStreamProvider)
	return ok
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
