package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/release"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

// WorkflowStartCommandHandler starts BPMN processes idempotently for durable commands.
type WorkflowStartCommandHandler struct {
	client   *ent.Client
	trigger  ProcessTriggerServiceInterface
	resolver *ProcessResolver
	logger   *zap.SugaredLogger
}

func NewWorkflowStartCommandHandler(client *ent.Client, trigger ProcessTriggerServiceInterface, resolver *ProcessResolver, logger *zap.SugaredLogger) *WorkflowStartCommandHandler {
	return &WorkflowStartCommandHandler{client: client, trigger: trigger, resolver: resolver, logger: logger}
}

func (h *WorkflowStartCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if cmd == nil || cmd.TenantID <= 0 || cmd.AggregateID <= 0 {
		return fmt.Errorf("invalid workflow command")
	}
	businessKey := fmt.Sprintf("%s:%d", cmd.AggregateType, cmd.AggregateID)
	exists, err := h.client.ProcessInstance.Query().
		Where(processinstance.TenantIDEQ(cmd.TenantID), processinstance.BusinessKeyEQ(businessKey)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check existing workflow: %w", err)
	}
	if exists {
		h.logger.Infow("workflow start command already applied", "command_id", cmd.ID, "business_key", businessKey)
		return nil
	}

	var req *dto.ProcessTriggerRequest
	switch cmd.AggregateType {
	case "ticket":
		tkt, err := h.client.Ticket.Query().Where(ticket.IDEQ(cmd.AggregateID), ticket.TenantIDEQ(cmd.TenantID), ticket.DeletedAtIsNil()).Only(ctx)
		if err != nil {
			return fmt.Errorf("load ticket for workflow: %w", err)
		}
		processKey, _ := cmd.Payload["workflowDefinitionKey"].(string)
		// 工单类型/分类的绑定可能在流程定义被停用或删除后仍保留，直接用它会让
		// workflow.start 反复重试并卡住工单生命周期；视为未配置，回落到路由表与
		// 内置兜底并告警（与变更域同一处理）。
		if processKey != "" {
			available, derr := h.definitionAvailable(ctx, cmd.TenantID, processKey)
			if derr != nil {
				return fmt.Errorf("check ticket process definition: %w", derr)
			}
			if !available {
				h.logger.Warnw("ticket workflow binding points to a missing or inactive process definition, falling back to routing",
					"command_id", cmd.ID, "ticket_id", cmd.AggregateID, "tenant_id", cmd.TenantID, "bound_key", processKey)
				processKey = ""
			}
		}
		if processKey == "" && h.resolver != nil {
			processKey, err = h.resolver.ResolveWithPriority(ctx, tkt, "")
			if err != nil {
				return fmt.Errorf("resolve ticket workflow: %w", err)
			}
		}
		if processKey == "" {
			processKey = "ticket_general_flow"
		}
		// 解析器（尤其 ResolveWithPriority 的 ticket_urgent_flow 兜底）与绑定表都可能
		// 指向租户未部署的 key；统一在此校验并回退通用工单流程，避免 workflow.start
		// 反复重试后进死信、工单永远没有流程实例。
		processKey, err = h.pickExistingDefinition(ctx, cmd.TenantID, processKey, "ticket_general_flow", "ticket_assignment_flow")
		if err != nil {
			return err
		}
		variables := map[string]interface{}{
			"ticket_id": tkt.ID, "ticket_number": tkt.TicketNumber, "title": tkt.Title,
			"description": tkt.Description, "priority": tkt.Priority, "status": tkt.Status,
			"requester_id": tkt.RequesterID,
		}
		if tkt.AssigneeID > 0 {
			variables["assignee_id"] = tkt.AssigneeID
		}
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeTicket, BusinessID: tkt.ID, ProcessDefinitionKey: processKey,
			Variables: variables, TriggeredBy: fmt.Sprintf("%d", tkt.RequesterID), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "incident":
		inc, err := h.client.Incident.Query().Where(incident.IDEQ(cmd.AggregateID), incident.TenantIDEQ(cmd.TenantID), incident.DeletedAtIsNil()).Only(ctx)
		if err != nil {
			return fmt.Errorf("load incident for workflow: %w", err)
		}
		processKey := "incident_emergency_flow"
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeIncident, BusinessID: inc.ID, ProcessDefinitionKey: processKey,
			Variables: map[string]interface{}{
				"incident_id": inc.ID, "incident_number": inc.IncidentNumber, "title": inc.Title,
				"description": inc.Description, "priority": inc.Priority, "severity": inc.Severity,
				"status": inc.Status, "category": inc.Category, "reporter_id": inc.ReporterID, "assignee_id": inc.AssigneeID,
			},
			TriggeredBy: fmt.Sprintf("%d", inc.ReporterID), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "problem":
		item, err := h.client.Problem.Query().Where(problem.IDEQ(cmd.AggregateID), problem.TenantIDEQ(cmd.TenantID), problem.DeletedAtIsNil()).Only(ctx)
		if err != nil {
			return fmt.Errorf("load problem for workflow: %w", err)
		}
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeProblem, BusinessID: item.ID, ProcessDefinitionKey: "problem_management_flow",
			Variables: map[string]interface{}{
				"problem_id": item.ID, "title": item.Title, "description": item.Description,
				"priority": item.Priority, "status": item.Status, "category": item.Category,
				"root_cause": item.RootCause, "impact": item.Impact, "created_by": item.CreatedBy,
			},
			TriggeredBy: fmt.Sprintf("%d", item.CreatedBy), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "change":
		ch, err := h.client.Change.Query().Where(change.IDEQ(cmd.AggregateID), change.TenantIDEQ(cmd.TenantID)).Only(ctx)
		if err != nil {
			return fmt.Errorf("load change for workflow: %w", err)
		}
		fallbackKey := "change_normal_flow"
		if ch.Type == "emergency" {
			fallbackKey = "change_emergency_flow"
		}
		processKey, _ := cmd.Payload["workflowDefinitionKey"].(string)
		if processKey == "" && h.resolver != nil {
			// P1（issue #92）：变更域接入流程路由——payload 显式指定 >
			// process_bindings 路由（按变更类型/风险等级）> 内置兜底。
			routed, rerr := h.resolver.ResolveForChange(ctx, ch, "")
			if rerr != nil {
				return fmt.Errorf("resolve change workflow: %w", rerr)
			}
			processKey = routed
		}
		if processKey == "" {
			processKey = fallbackKey
		}
		// 路由/显式指定可能指向租户未部署的自定义流程定义，内置 key 也可能被租户
		// 停用——两种都统一校验：不存在则回退（emergency 先试 change_emergency_flow，
		// 再退 change_normal_flow），避免 workflow.start 反复重试卡死变更生命周期。
		processKey, err = h.pickExistingDefinition(ctx, cmd.TenantID, processKey, fallbackKey, "change_normal_flow")
		if err != nil {
			return err
		}
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeChange, BusinessID: ch.ID, ProcessDefinitionKey: processKey,
			Variables: map[string]interface{}{
				"change_id": ch.ID, "title": ch.Title, "description": ch.Description, "type": ch.Type,
				"priority": ch.Priority, "status": ch.Status, "impact_scope": ch.ImpactScope,
				"risk_level": ch.RiskLevel, "created_by": ch.CreatedBy, "assignee_id": ch.AssigneeID,
				"need_approval": ch.Type != "emergency",
			},
			TriggeredBy: fmt.Sprintf("%d", ch.CreatedBy), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "service_request":
		sr, err := h.client.ServiceRequest.Query().Where(
			servicerequest.IDEQ(cmd.AggregateID),
			servicerequest.TenantIDEQ(cmd.TenantID),
			servicerequest.DeletedAtIsNil(),
		).Only(ctx)
		if err != nil {
			return fmt.Errorf("load service request for workflow: %w", err)
		}
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeServiceRequest, BusinessID: sr.ID,
			ProcessDefinitionKey: "service_request_flow",
			Variables: map[string]interface{}{
				"service_request_id": sr.ID, "title": sr.Title, "reason": sr.Reason,
				"status": sr.Status, "requester_id": sr.RequesterID, "catalog_id": sr.CatalogID,
				"current_level": sr.CurrentLevel, "total_levels": sr.TotalLevels,
			},
			TriggeredBy: fmt.Sprintf("%d", sr.RequesterID), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	case "release":
		rel, err := h.client.Release.Query().Where(release.IDEQ(cmd.AggregateID), release.TenantIDEQ(cmd.TenantID)).Only(ctx)
		if err != nil {
			return fmt.Errorf("load release for workflow: %w", err)
		}
		// 发布域无路由绑定 resolver，固定走默认绑定 release → release_approval_flow
		//（见 bpmn_process_binding_service.go 默认绑定表）；与 problem 分支同口径。
		// 发布创建即进入审批流：未显式指定时按默认绑定启动，使 /approve 的 BPMN
		// 任务完成桥接（ApplyReleaseApproval）有流程实例可落。
		req = &dto.ProcessTriggerRequest{
			BusinessType: dto.BusinessTypeRelease, BusinessID: rel.ID, ProcessDefinitionKey: "release_approval_flow",
			Variables: map[string]interface{}{
				"release_id": rel.ID, "release_number": rel.ReleaseNumber, "title": rel.Title,
				"description": rel.Description, "type": rel.Type, "severity": rel.Severity,
				"environment": rel.Environment, "status": string(rel.Status), "created_by": rel.CreatedBy,
				"requires_approval": rel.RequiresApproval, "is_emergency": rel.IsEmergency,
			},
			TriggeredBy: fmt.Sprintf("%d", rel.CreatedBy), TriggeredAt: time.Now(), TenantID: cmd.TenantID,
		}
	default:
		return fmt.Errorf("unsupported workflow aggregate type %q", cmd.AggregateType)
	}
	if _, err := h.trigger.TriggerProcess(ctx, req); err != nil {
		return fmt.Errorf("start %s workflow: %w", cmd.AggregateType, err)
	}
	return nil
}

// definitionAvailable 判断流程定义在本租户是否存在且处于启用状态。
func (h *WorkflowStartCommandHandler) definitionAvailable(ctx context.Context, tenantID int, key string) (bool, error) {
	return h.client.ProcessDefinition.Query().Where(
		processdefinition.TenantIDEQ(tenantID),
		processdefinition.IsActive(true),
		processdefinition.Key(key),
	).Exist(ctx)
}

// pickExistingDefinition 返回 key 与 fallbacks 中第一个在租户下真实存在且启用的流程 key。
//
// 为什么需要这道闸（2026-10-24 本地生产栈实测）：
//
//	ProcessTriggerService.TriggerProcess 在定义缺失时只返回普通 error，
//	commandbus 一律视为可重试（默认 8 次）后进 dead_letter。而业务单据此时
//	已经创建成功——用户看到「工单建了但永远没有流程」，运维看到一条既无单据
//	也无可操作建议的死信。两处真实故障都是这条路径：
//	  · ticket 高/紧急优先级 → ProcessResolver.ResolveWithPriority 兜底
//	    ticket_urgent_flow，而 service/bpmn/ 下从来没有这个模板；
//	  · 紧急变更 → ResolveForChange 兜底 change_emergency_flow，同样没有模板
//	    （本轮已补 bpmn/change_emergency_flow.bpmn，但租户可能自行停用/删除，
//	     仍需兜底）。
//
// 语义：只有「解析出的 key 与所有 fallback 都不可用」才算真失败；此时返回的
// error 会列出全部候选 key 与租户，让死信本身具备可操作性，而不是重复 8 次
// 同一句「流程定义 X 不存在」。
func (h *WorkflowStartCommandHandler) pickExistingDefinition(ctx context.Context, tenantID int, key string, fallbacks ...string) (string, error) {
	candidates := make([]string, 0, len(fallbacks)+1)
	seen := make(map[string]bool, len(fallbacks)+1)
	for _, candidate := range append([]string{key}, fallbacks...) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no workflow definition key resolved for tenant %d", tenantID)
	}

	for _, candidate := range candidates {
		available, err := h.definitionAvailable(ctx, tenantID, candidate)
		if err != nil {
			return "", fmt.Errorf("check process definition %s: %w", candidate, err)
		}
		if available {
			if candidate != key {
				h.logger.Warnw("workflow definition unavailable, falling back to builtin",
					"tenant_id", tenantID, "resolved_key", key, "fallback_key", candidate)
			}
			return candidate, nil
		}
	}
	return "", fmt.Errorf(
		"tenant %d 没有任何可用的流程定义（候选 %v 均不存在或已停用）："+
			"请确认内置模板已部署（启动日志 bpmn-template-bootstrap-deploy），或为该业务类型配置 process_bindings",
		tenantID, candidates,
	)
}
