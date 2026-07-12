package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
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
	exists, err := s.client.ProcessDefinition.Query().Where(processdefinition.Key(key), processdefinition.TenantID(workflow.TenantID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		result.Skipped = true
		return result, nil
	}
	_, err = s.deployment.DeployProcessDefinition(ctx, &DeployProcessDefinitionRequest{Name: workflow.Name, Description: "Migrated from legacy approval workflow", BPMNXML: bpmnXML, TenantID: workflow.TenantID})
	if err != nil {
		return nil, err
	}
	businessType := dto.BusinessType(workflow.TicketType)
	if businessType == "" {
		businessType = dto.BusinessTypeTicket
	}
	conditions := map[string]interface{}{}
	if workflow.Priority != "" {
		conditions["priority"] = workflow.Priority
	}
	_, err = s.binding.CreateBinding(ctx, &dto.ProcessBinding{BusinessType: businessType, ProcessDefinitionKey: key, ProcessVersion: 1, Priority: 50, IsActive: workflow.IsActive, TenantID: workflow.TenantID, Conditions: conditions})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func buildLegacyApprovalBPMN(key, name string, nodes []map[string]interface{}) (string, error) {
	if strings.TrimSpace(key) == "" || len(nodes) == 0 {
		return "", fmt.Errorf("legacy workflow must have a key and at least one node")
	}
	sort.SliceStable(nodes, func(i, j int) bool { return intValue(nodes[i]["step_order"]) < intValue(nodes[j]["step_order"]) })
	escape := func(v string) string { var b strings.Builder; _ = xml.EscapeText(&b, []byte(v)); return b.String() }
	var tasks, flows strings.Builder
	previous := "StartEvent_1"
	for i, node := range nodes {
		id := fmt.Sprintf("Approval_%d", i+1)
		assigneeType, _ := node["assignee_type"].(string)
		assignee := fmt.Sprint(node["assignee_value"])
		attribute := "assignee"
		if assigneeType == "role" || assigneeType == "group" {
			attribute = "candidateGroups"
		}
		fmt.Fprintf(&tasks, `<bpmn:userTask id="%s" name="%s" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:%s="%s" itsm:commentRequiredOnReject="true"/>`, id, escape(fmt.Sprint(node["name"])), attribute, escape(assignee))
		fmt.Fprintf(&flows, `<bpmn:sequenceFlow id="Flow_%d" sourceRef="%s" targetRef="%s"/>`, i+1, previous, id)
		previous = id
	}
	fmt.Fprintf(&flows, `<bpmn:sequenceFlow id="Flow_%d" sourceRef="%s" targetRef="EndEvent_1"/>`, len(nodes)+1, previous)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="%s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/>%s<bpmn:endEvent id="EndEvent_1"/>%s</bpmn:process></bpmn:definitions>`, escape(key), escape(key), escape(name), tasks.String(), flows.String()), nil
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
