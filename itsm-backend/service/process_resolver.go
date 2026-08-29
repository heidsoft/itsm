package service

import (
	"context"

	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
)

// ProcessResolver 解析工单应该使用哪个 BPMN 流程
// 优先级：1.请求指定 2.ProcessBinding 3.兜底
type ProcessResolver struct {
	client  *ent.Client
	routing *ProcessRoutingService
}

// NewProcessResolver 创建流程解析器
func NewProcessResolver(client *ent.Client, bindingService ProcessBindingServiceInterface) *ProcessResolver {
	return &ProcessResolver{
		client:  client,
		routing: NewProcessRoutingService(client, zap.NewNop().Sugar()),
	}
}

// Resolve 解析工单应该使用的流程 Key
func (r *ProcessResolver) Resolve(ctx context.Context, ticket *ent.Ticket, reqKey string) (string, error) {
	// 优先级 1：请求参数显式指定
	if reqKey != "" {
		return reqKey, nil
	}

	// 优先级 2：ProcessBinding 表查询（按工单类型匹配）
	if r.routing != nil {
		route, err := r.routing.FindBestRoute(ctx, &RoutingContext{
			BusinessType: string(dto.BusinessTypeTicket), BusinessSubType: ticket.Type,
			TenantID:  ticket.TenantID,
			Variables: map[string]interface{}{"priority": ticket.Priority},
		})
		if err != nil {
			return "", err
		}
		if route != nil {
			return route.ProcessDefinitionKey, nil
		}
	}

	// 优先级 3：兜底默认
	return "ticket_general_flow", nil
}

// ResolveWithPriority 考虑优先级的解析（通用工单场景）
func (r *ProcessResolver) ResolveWithPriority(ctx context.Context, ticket *ent.Ticket, reqKey string) (string, error) {
	// 先走标准解析
	processKey, err := r.Resolve(ctx, ticket, reqKey)
	if err != nil {
		return "", err
	}

	// 如果是通用工单（没有匹配到特定类型），根据优先级调整
	if processKey == "ticket_general_flow" {
		if ticket.Priority == "high" || ticket.Priority == "urgent" {
			return "ticket_urgent_flow", nil
		}
	}

	return processKey, nil
}
