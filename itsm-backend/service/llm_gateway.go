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
