package dto

import (
	"context"
	"errors"
)

// LLM 网关的传输值类型与消费面端口。
//
// 这些 struct 原定义在共享层 service 包，但域代码（AI 流式问答、邮件抽取）
// 只消费其数据形状。按依赖倒置把值类型放在 dto、把网关能力收敛为端口接口：
// service 侧保留同名别名，因此运行时类型、JSON 形状与 errors.Is 身份都不变。

type LLMTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

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

// LLMGatewayPort 是域侧真正消费的网关能力面。
//
// 只列域代码会调用的方法，不复制网关内部的配额/重试/观测实现。
// *service.LLMGateway 隐式满足本接口；RAG 服务的方法参数也统一到该端口，
// 使域内 ragStreamer 契约的签名与具体实现完全一致（Go 接口要求精确匹配）。
type LLMGatewayPort interface {
	Chat(ctx context.Context, model string, messages []LLMMessage) (string, error)
	ChatStream(ctx context.Context, model string, messages []LLMMessage, callback func(string)) error
	ChatStreamWithTools(ctx context.Context, model string, messages []LLMMessage, tools []LLMTool, callback func(string), onToolCalls func([]LLMToolCall)) error
	SupportsToolCalling() bool
	EffectiveModel(model string) string
	ProviderName() string
}

// RateLimitError 单请求 token 超限。与网关配额错误分开：前者是限流，
// 后者是租户月度预算耗尽，调用方分别映射为 429 与 429 + 配额提示。
type RateLimitError struct{ Message string }

func (e *RateLimitError) Error() string { return e.Message }

var ErrRateLimited = &RateLimitError{Message: "rate limited"}

var ErrQuotaExceeded = errors.New("tenant monthly token quota exceeded")
