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

// ResolveForChange 解析变更应使用的 BPMN 流程 Key（issue #92：变更域接入流程路由）。
// 优先级：1. 请求/命令显式指定 2. process_bindings 路由
// （business_sub_type = 变更类型 normal|standard|emergency，conditions 可按 risk_level/priority 匹配）
// 3. 内置兜底（emergency → change_emergency_flow，其余 → change_normal_flow）。
// 注意：路由命中返回的 key 不校验流程定义是否存在——由调用方（WorkflowStartCommandHandler）
// 对非内置 key 做存在性校验并告警回退，避免租户误配导致 workflow.start 命令反复重试。
func (r *ProcessResolver) ResolveForChange(ctx context.Context, ch *ent.Change, reqKey string) (string, error) {
	// 优先级 1：请求参数显式指定
	if reqKey != "" {
		return reqKey, nil
	}

	// 优先级 2：ProcessBinding 表路由（按变更类型匹配）
	if r.routing != nil {
		route, err := r.routing.FindBestRoute(ctx, &RoutingContext{
			BusinessType:    string(dto.BusinessTypeChange),
			BusinessSubType: ch.Type,
			TenantID:        ch.TenantID,
			Variables: map[string]interface{}{
				"change_type": ch.Type,
				"risk_level":  ch.RiskLevel,
				"priority":    ch.Priority,
			},
		})
		if err != nil {
			return "", err
		}
		if route != nil && route.ProcessDefinitionKey != "" {
			return route.ProcessDefinitionKey, nil
		}
	}

	// 优先级 3：内置兜底
	if ch.Type == "emergency" {
		return "change_emergency_flow", nil
	}
	return "change_normal_flow", nil
}
