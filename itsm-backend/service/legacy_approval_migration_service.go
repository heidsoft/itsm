package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"
)

type LegacyApprovalMigrationService struct {
	client     *ent.Client
	deployment *BPMNDeploymentService
	binding    *ProcessBindingService
}

func NewLegacyApprovalMigrationService(client *ent.Client) *LegacyApprovalMigrationService {
	return &LegacyApprovalMigrationService{client: client, deployment: NewBPMNDeploymentService(client), binding: NewProcessBindingService(client)}
}

type LegacyApprovalMigrationResult struct {
	WorkflowID           int    `json:"workflowId"`
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	BPMNXML              string `json:"bpmnXml,omitempty"`
	Skipped              bool   `json:"skipped"`
	BindingReplaced      bool   `json:"bindingReplaced"`
}

func (s *LegacyApprovalMigrationService) Migrate(ctx context.Context, workflow *ent.ApprovalWorkflow, dryRun bool) (*LegacyApprovalMigrationResult, error) {
	key := fmt.Sprintf("legacy_approval_%d", workflow.ID)
	bpmnXML, err := buildLegacyApprovalBPMN(key, workflow.Name, workflow.Nodes)
	if err != nil {
		return nil, err
	}
	result := &LegacyApprovalMigrationResult{WorkflowID: workflow.ID, ProcessDefinitionKey: key, BPMNXML: bpmnXML}
	if dryRun {
		return result, nil
	}

	// 1) 部署 BPMN 流程定义（幂等：已存在则跳过）。
	exists, err := s.client.ProcessDefinition.Query().Where(processdefinition.Key(key), processdefinition.TenantID(workflow.TenantID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		if _, err = s.deployment.DeployProcessDefinition(ctx, &DeployProcessDefinitionRequest{Name: workflow.Name, Description: "Migrated from legacy approval workflow", BPMNXML: bpmnXML, TenantID: workflow.TenantID}); err != nil {
			return nil, err
		}
	} else {
		result.Skipped = true
	}

	// 2) 确保存在一条把该业务键路由到迁移流程的绑定。
	//    若同业务键已有激活绑定（激活路由唯一索引限制每键仅一条激活绑定），
	//    原地改指向迁移流而非插入重复行；否则新建。这样“迁移”会真正接管
	//    该业务键的审批路由，而不是因唯一索引冲突而失败。
	// 审批绑定统一挂在 business_type="approval" 之下，subType 携带工单类型，
	// 与工单生命周期绑定（business_type="ticket"）隔离，避免抢占总线解析导致
	// 工单主流程被审批流程顶替。
	businessType := dto.BusinessType("approval")
	subType := workflow.TicketType
	conditions := map[string]interface{}{}
	if workflow.Priority != "" {
		conditions["priority"] = workflow.Priority
	}
	replaced, err := s.upsertBinding(ctx, workflow, key, businessType, subType, conditions)
	if err != nil {
		return nil, err
	}
	result.BindingReplaced = replaced
	return result, nil
}

// upsertBinding 确保 (businessType, "") 业务键的激活绑定指向 key 对应的流程定义。
// 已存在则原地更新（替换路由目标），不存在则新建。
func (s *LegacyApprovalMigrationService) upsertBinding(ctx context.Context, workflow *ent.ApprovalWorkflow, key string, businessType dto.BusinessType, subType string, conditions map[string]interface{}) (bool, error) {
	existing, err := s.client.ProcessBinding.Query().
		Where(
			processbinding.BusinessType(string(businessType)),
			processbinding.BusinessSubType(subType),
			processbinding.TenantID(workflow.TenantID),
			processbinding.IsActive(true),
		).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return false, err
	}
	if existing != nil {
		if _, err = s.client.ProcessBinding.UpdateOneID(existing.ID).
			SetProcessDefinitionKey(key).
			SetIsActive(workflow.IsActive).
			SetPriority(50).
			SetConditions(conditions).
			Save(ctx); err != nil {
			return false, fmt.Errorf("更新已有流程绑定失败: %w", err)
		}
		return true, nil
	}
	if _, err = s.binding.CreateBinding(ctx, &dto.ProcessBinding{
		BusinessType:         businessType,
		BusinessSubType:      subType,
		ProcessDefinitionKey: key,
		ProcessVersion:       1,
		Priority:             50,
		IsActive:             workflow.IsActive,
		TenantID:             workflow.TenantID,
		Conditions:           conditions,
	}); err != nil {
		return false, err
	}
	return false, nil
}

func buildLegacyApprovalBPMN(key, name string, nodes []map[string]interface{}) (string, error) {
	if strings.TrimSpace(key) == "" || len(nodes) == 0 {
		return "", fmt.Errorf("legacy workflow must have a key and at least one node")
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return legacyApprovalNodeOrder(nodes[i], i) < legacyApprovalNodeOrder(nodes[j], j)
	})
	escape := func(v string) string { var b strings.Builder; _ = xml.EscapeText(&b, []byte(v)); return b.String() }
	var tasks, flows strings.Builder
	previous := "StartEvent_1"
	for i, node := range nodes {
		id := fmt.Sprintf("Approval_%d", i+1)
		attribute, assignee := resolveLegacyApprovalAssignee(node)
		attr := ""
		if assignee != "" {
			attr = fmt.Sprintf(` itsm:%s="%s"`, attribute, escape(assignee))
		}
		fmt.Fprintf(&tasks, `<bpmn:userTask id="%s" name="%s" itsm:taskPurpose="approval" itsm:approvalMode="single"%s itsm:commentRequiredOnReject="true"/>`, id, escape(fmt.Sprint(node["name"])), attr)
		fmt.Fprintf(&flows, `<bpmn:sequenceFlow id="Flow_%d" sourceRef="%s" targetRef="%s"/>`, i+1, previous, id)
		previous = id
	}
	fmt.Fprintf(&flows, `<bpmn:sequenceFlow id="Flow_%d" sourceRef="%s" targetRef="EndEvent_1"/>`, len(nodes)+1, previous)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="%s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/>%s<bpmn:endEvent id="EndEvent_1"/>%s</bpmn:process></bpmn:definitions>`, escape(key), escape(key), escape(name), tasks.String(), flows.String()), nil
}

// resolveLegacyApprovalAssignee extracts the BPMN assignee/candidateGroups from a
// legacy approval node. It tolerates the three node shapes that actually exist in the
// database: the canonical API schema (assigneeType/assigneeValue), the snake_case
// variant (assignee_type/assignee_value), and the older seeder schema
// (approver_type + role). Returns ("assignee"|"candidateGroups", value).
func resolveLegacyApprovalAssignee(node map[string]interface{}) (string, string) {
	if at, ok := node["assigneeType"].(string); ok && at != "" {
		av := fmt.Sprint(node["assigneeValue"])
		if at == "role" || at == "group" {
			return "candidateGroups", av
		}
		return "assignee", av
	}
	if at, ok := node["assignee_type"].(string); ok && at != "" {
		av := fmt.Sprint(node["assignee_value"])
		if at == "role" || at == "group" {
			return "candidateGroups", av
		}
		return "assignee", av
	}
	if at, ok := node["approver_type"].(string); ok && at != "" {
		if at == "role" {
			return "candidateGroups", fmt.Sprint(node["role"])
		}
		// 动态角色（manager / dept_manager / team_leader / project_manager 等）
		// 作为候选组挂到任务上，由 BPMN 引擎按角色解析审批人。
		return "candidateGroups", at
	}
	return "assignee", ""
}

// legacyApprovalNodeOrder returns a stable sort key for a legacy approval node.
// It prefers an explicit step_order, then level, then falls back to the slice index.
func legacyApprovalNodeOrder(node map[string]interface{}, idx int) int {
	if v := intValue(node["step_order"]); v > 0 {
		return v
	}
	if v := intValue(node["level"]); v > 0 {
		return v
	}
	return idx + 1
}

func intValue(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
