package bpmn

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ApprovalChainResolver 审批链求值接口。
// 定义在 bpmn 包以避免 service ↔ service/bpmn 循环导入。
// 由 service.ApprovalChainService 实现（通过适配器）。
type ApprovalChainResolver interface {
	ResolveApprovalPlanRaw(
		ctx context.Context,
		tenantID int,
		entityType string,
		requesterID int,
		priority string,
		amount float64,
		approvals map[int][]int,
	) (chainID int, levelsJSON []byte, passed bool, pendingLevel int, blocked bool, err error)
}

// ApprovalChainServiceTaskHandler 审批链配置读取 Service Task。
// 配置与编排分离：ApprovalChain 表为配置源，本 handler 在 BPMN 流程编排中读取配置并求值。
// 配置读取失败时阻塞流程并告警，不使用默认审批人（SA 决策）。
type ApprovalChainServiceTaskHandler struct {
	HandlerBase
	client   *ent.Client
	logger   *zap.SugaredLogger
	resolver ApprovalChainResolver
}

func NewApprovalChainServiceTaskHandler(
	client *ent.Client,
	logger *zap.SugaredLogger,
) *ApprovalChainServiceTaskHandler {
	return &ApprovalChainServiceTaskHandler{
		client: client,
		logger: logger,
	}
}

// SetApprovalChainResolver 延迟注入审批链求值器，避免循环依赖。
func (h *ApprovalChainServiceTaskHandler) SetApprovalChainResolver(r ApprovalChainResolver) {
	h.resolver = r
}

func (h *ApprovalChainServiceTaskHandler) GetTaskType() string {
	return "approval_chain_task"
}

func (h *ApprovalChainServiceTaskHandler) GetHandlerID() string {
	return "approval_chain_handler"
}

func (h *ApprovalChainServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	if h.resolver == nil {
		return nil, fmt.Errorf("审批链求值器未注入，请检查启动顺序")
	}
	action := GetStringFromVars(variables, "action")
	switch action {
	case "resolve_plan":
		return h.resolvePlan(ctx, variables)
	default:
		return nil, fmt.Errorf("审批链服务任务未知动作: %q", action)
	}
}

func (h *ApprovalChainServiceTaskHandler) Validate(_ context.Context, _ map[string]interface{}) error {
	return nil
}

// resolvePlan 读取审批链配置并求值，返回审批计划。
// BPMN 变量约定：
//   - tenant_id (int, 必填)
//   - entity_type (string, 必填): "ticket" | "change" | "service_request"
//   - requester_id (int, 必填)
//   - priority (string, 可选)
//   - amount (float64, 可选)
//   - approvals (map[int][]int, 可选): 已完成的审批层级
func (h *ApprovalChainServiceTaskHandler) resolvePlan(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	tenantID, err := ResolveTenantID(ctx, variables)
	if err != nil {
		return nil, err
	}

	entityType := GetStringFromVars(variables, "entity_type")
	if entityType == "" {
		return nil, fmt.Errorf("审批链求值缺少 entity_type")
	}

	requesterID := GetIntFromVars(variables, "requester_id")
	if requesterID <= 0 {
		return nil, fmt.Errorf("审批链求值缺少有效的 requester_id")
	}

	priority := GetStringFromVars(variables, "priority")
	var amount float64
	if v, ok := variables["amount"]; ok {
		switch val := v.(type) {
		case float64:
			amount = val
		case int:
			amount = float64(val)
		case int64:
			amount = float64(val)
		}
	}

	var approvals map[int][]int
	if raw, ok := variables["approvals"]; ok && raw != nil {
		if err := decodeApprovals(raw, &approvals); err != nil {
			return nil, fmt.Errorf("解析 approvals 变量失败: %w", err)
		}
	}

	chainID, levelsJSON, passed, pendingLevel, blocked, err := h.resolver.ResolveApprovalPlanRaw(
		ctx, tenantID, entityType, requesterID, priority, amount, approvals,
	)
	if err != nil {
		h.logger.Errorw("审批链配置读取/求值失败，阻塞流程",
			"tenant_id", tenantID, "entity_type", entityType, "requester_id", requesterID, "error", err)
		return nil, fmt.Errorf("审批链配置读取失败，流程阻塞: %w", err)
	}

	h.logger.Infow("审批链求值完成",
		"tenant_id", tenantID, "entity_type", entityType,
		"chain_id", chainID, "passed", passed,
		"pending_level", pendingLevel, "blocked", blocked)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: "审批链求值完成",
		OutputVars: map[string]interface{}{
			"chain_id":      chainID,
			"levels":        string(levelsJSON),
			"passed":        passed,
			"pending_level": pendingLevel,
			"blocked":       blocked,
		},
	}, nil
}

func decodeApprovals(raw interface{}, out *map[int][]int) error {
	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[int][]int, len(rawMap))
	for k, v := range rawMap {
		var level int
		if _, err := fmt.Sscanf(k, "%d", &level); err != nil {
			continue
		}
		arr, ok := v.([]interface{})
		if !ok {
			continue
		}
		ids := make([]int, 0, len(arr))
		for _, item := range arr {
			switch id := item.(type) {
			case float64:
				ids = append(ids, int(id))
			case int:
				ids = append(ids, id)
			case int64:
				ids = append(ids, int(id))
			}
		}
		result[level] = ids
	}
	*out = result
	return nil
}
