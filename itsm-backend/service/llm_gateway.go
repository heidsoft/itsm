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

type LLMMessage struct {
	Role    string
	Content string
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
