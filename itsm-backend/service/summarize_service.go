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

	result, err := s.gateway.Chat(ctx, "", messages)
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

	result, err := s.gateway.Chat(ctx, "", messages)
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

	result, err := s.gateway.Chat(ctx, "", messages)
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

// SummarizePromptVersion 是摘要的 prompt/口径版本，随响应透出并写入 AI 审计底账。
// 换 prompt 模板就必须换它，否则底账里两代 prompt 的产出无法区分。
const SummarizePromptVersion = "ticket-summary-v1"

// SummaryWithMetadata includes metadata about the summary
//
// Method 是「这份摘要到底怎么来的」，三种取值必须如实区分：
//
//	llm      —— 模型真的生成了摘要
//	direct   —— 原文够短，直接原样返回（**不是摘要**，是原文）
//	fallback —— 模型不可用/调用失败，退化为截断原文（**不是摘要**，是截断）
//
// 区分它们不是为了好看：此前 direct/fallback 产出的东西同样被当作「AI 摘要」
// 呈给用户，而底账里没有任何字段能区分。用户以为自己在评估 AI，实际评估的是
// 一段截断文本——采纳率、有用率这些指标因此从根上就是脏的。
//
// Model 是真实模型名（由 LLMGateway.EffectiveModel 取得，取不到为 "unknown"）；
// llm 以外的路径没有模型参与，字段留空——留空是实情，不是缺失。
//
// Confidence 对摘要**恒为 0 并标注不适用**：摘要是生成类产出，模型不给自评
// 分数，硬填一个数字就是编造。是否可信由操作者的采纳/拒绝翻转来表达（那才是
// 真实信号），不靠一个假分数装点。
type SummaryWithMetadata struct {
	Summary   string        `json:"summary"`
	Method    string        `json:"method"` // "llm" | "direct" | "fallback"
	Latency   time.Duration `json:"latencyMs"`
	CharCount int           `json:"charCount"`
	// 溯源三字段
	PromptVersion string  `json:"promptVersion"`
	Model         string  `json:"model,omitempty"` // 空 = 无模型参与（direct/fallback）
	Confidence    float64 `json:"confidence"`      // 摘要场景恒为 0，见类型注释
}

// SummarizeWithMetadata returns summary with additional metadata
func (s *SummarizeService) SummarizeWithMetadata(ctx context.Context, text string, maxLen int) (*SummaryWithMetadata, error) {
	start := time.Now()

	if maxLen <= 0 {
		maxLen = 200
	}

	result := &SummaryWithMetadata{
		CharCount:     len(text),
		PromptVersion: SummarizePromptVersion,
	}

	if len(text) <= maxLen {
		result.Summary = text
		result.Method = "direct"
		result.Latency = time.Since(start)
		// 原文直出：没有模型参与，Model 留空是有意为之（不是漏填）。
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

	summary, err := s.gateway.Chat(ctx, "", messages)
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
	// 调用成功后再取模型名：本次调用未指定 model，实际用的是 provider 配置的
	// 那一个，只有网关报得出来（与分诊同源，不再硬编码）。
	result.Model = s.gateway.EffectiveModel("")
	result.Latency = time.Since(start)

	return result, nil
}
