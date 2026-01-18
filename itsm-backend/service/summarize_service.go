package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SummarizeService provides LLM-powered text summarization
type SummarizeService struct {
	gateway *LLMGateway
	logger  *zap.Logger
}

// NewSummarizeService creates a new SummarizeService with LLM gateway
func NewSummarizeService(gateway *LLMGateway, logger *zap.Logger) *SummarizeService {
	return &SummarizeService{
		gateway: gateway,
		logger:  logger,
	}
}

// Summarize generates a concise summary of the given text using LLM
func (s *SummarizeService) Summarize(ctx context.Context, text string, maxLen int) (string, error) {
	if maxLen <= 0 {
		maxLen = 200
	}

	// If text is short enough, return as-is
	if len(text) <= maxLen {
		return text, nil
	}

	// If no LLM gateway configured, fall back to simple truncation
	if s.gateway == nil {
		s.logger.Warn("SummarizeService: no LLM gateway configured, using fallback")
		return simpleTruncate(text, maxLen), nil
	}

	// Build prompt for summarization
	prompt := fmt.Sprintf(`请用简洁的语言总结以下IT工单内容，摘要长度不超过%d字：

---
%s
---

请只输出摘要内容，不需要任何额外解释。`, maxLen, text)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的IT服务管理助手，负责将工单内容浓缩成简洁的摘要。"},
		{Role: "user", Content: prompt},
	}

	result, err := s.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		s.logger.Error("SummarizeService: LLM call failed", zap.Error(err))
		// Fallback to simple truncation on error
		return simpleTruncate(text, maxLen), nil
	}

	// Clean up the result
	summary := strings.TrimSpace(result)
	// Remove any potential markdown formatting
	summary = strings.TrimPrefix(summary, "```")
	summary = strings.TrimSuffix(summary, "```")
	summary = strings.TrimSpace(summary)

	// Ensure we don't exceed maxLen
	if len(summary) > maxLen {
		summary = simpleTruncate(summary, maxLen)
	}

	return summary, nil
}

// SummarizeWithContext generates a summary using relevant context
func (s *SummarizeService) SummarizeWithContext(ctx context.Context, text, contextInfo string, maxLen int) (string, error) {
	if maxLen <= 0 {
		maxLen = 200
	}

	if s.gateway == nil {
		return s.Summarize(ctx, text, maxLen)
	}

	prompt := fmt.Sprintf(`请用简洁的语言总结以下IT工单内容，摘要长度不超过%d字。

相关上下文信息：
%s

工单内容：
---
%s
---

请只输出摘要内容，不需要任何额外解释。`, maxLen, contextInfo, text)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的IT服务管理助手，负责将工单内容浓缩成简洁的摘要。"},
		{Role: "user", Content: prompt},
	}

	result, err := s.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		s.logger.Error("SummarizeService: LLM call with context failed", zap.Error(err))
		return s.Summarize(ctx, text, maxLen)
	}

	summary := strings.TrimSpace(result)
	summary = strings.TrimPrefix(summary, "```")
	summary = strings.TrimSuffix(summary, "```")
	summary = strings.TrimSpace(summary)

	if len(summary) > maxLen {
		summary = simpleTruncate(summary, maxLen)
	}

	return summary, nil
}

// GenerateActionItems extracts actionable items from the ticket
func (s *SummarizeService) GenerateActionItems(ctx context.Context, text string) ([]string, error) {
	if s.gateway == nil {
		return []string{}, nil
	}

	prompt := fmt.Sprintf(`请从以下IT工单中提取可执行的操作项，只输出列表格式，每行一个：

---
%s
---

操作项：`, text)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个IT服务管理助手，负责从工单中提取可执行的行动项。"},
		{Role: "user", Content: prompt},
	}

	result, err := s.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		s.logger.Error("SummarizeService: action items extraction failed", zap.Error(err))
		return []string{}, nil
	}

	// Parse the result into a list
	lines := strings.Split(result, "\n")
	actions := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove list markers
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "1. ")
		line = strings.TrimPrefix(line, "2. ")
		line = strings.TrimPrefix(line, "3. ")
		if line != "" && !strings.Contains(strings.ToLower(line), "操作项") {
			actions = append(actions, line)
		}
	}

	return actions, nil
}

// simpleTruncate provides a fallback when LLM is not available
func simpleTruncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// SummaryWithMetadata includes metadata about the summary
type SummaryWithMetadata struct {
	Summary   string        `json:"summary"`
	Method    string        `json:"method"`    // "llm" or "fallback"
	Latency   time.Duration `json:"latency_ms"`
	CharCount int           `json:"char_count"`
}

// SummarizeWithMetadata returns summary with additional metadata
func (s *SummarizeService) SummarizeWithMetadata(ctx context.Context, text string, maxLen int) (*SummaryWithMetadata, error) {
	start := time.Now()

	if maxLen <= 0 {
		maxLen = 200
	}

	result := &SummaryWithMetadata{
		CharCount: len(text),
	}

	if len(text) <= maxLen {
		result.Summary = text
		result.Method = "direct"
		result.Latency = time.Since(start)
		return result, nil
	}

	if s.gateway == nil {
		result.Summary = simpleTruncate(text, maxLen)
		result.Method = "fallback"
		result.Latency = time.Since(start)
		return result, nil
	}

	prompt := fmt.Sprintf(`请用简洁的语言总结以下IT工单内容，摘要长度不超过%d字：

---
%s
---

请只输出摘要内容，不需要任何额外解释。`, maxLen, text)

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的IT服务管理助手，负责将工单内容浓缩成简洁的摘要。"},
		{Role: "user", Content: prompt},
	}

	summary, err := s.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		s.logger.Error("SummarizeService: LLM call failed", zap.Error(err))
		result.Summary = simpleTruncate(text, maxLen)
		result.Method = "fallback"
		result.Latency = time.Since(start)
		return result, nil
	}

	// Clean up
	summary = strings.TrimSpace(summary)
	summary = strings.TrimPrefix(summary, "```")
	summary = strings.TrimSuffix(summary, "```")
	summary = strings.TrimSpace(summary)
	if len(summary) > maxLen {
		summary = simpleTruncate(summary, maxLen)
	}

	result.Summary = summary
	result.Method = "llm"
	result.Latency = time.Since(start)

	return result, nil
}
