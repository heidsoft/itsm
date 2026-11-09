package a2ui

import "context"

// TicketFormService 是 A2UI 工单表单 handler 真正消费的最小能力面。
// 表单消息结构属于域协议，已下沉到 dto 包；契约只保留两个生成/解析能力，
// 不把共享层的 LLM 网关或消息构造细节带进域包。
type TicketFormService interface {
	GenerateFormMessages(ctx context.Context, userIntent string, surfaceID string) ([]string, error)
	ParseTicketIntent(ctx context.Context, description string) (map[string]interface{}, error)
}
