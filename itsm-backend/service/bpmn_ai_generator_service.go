package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"itsm-backend/dto"
)

// BPMNAIGeneratorService AI驱动的BPMN流程生成服务
type BPMNAIGeneratorService struct {
	llmGateway        *LLMGateway
	deploymentService *BPMNDeploymentService
	templateService   *BPMNTemplateService
	templateCatalog   *BPMNWorkflowTemplateCatalog
	parser            *BPMNParser
}

func (s *BPMNAIGeneratorService) SetTemplateCatalog(catalog *BPMNWorkflowTemplateCatalog) {
	s.templateCatalog = catalog
}

// NewBPMNAIGeneratorService 创建AI生成服务实例
func NewBPMNAIGeneratorService(llmGateway *LLMGateway, deploymentService *BPMNDeploymentService, templateServices ...*BPMNTemplateService) *BPMNAIGeneratorService {
	var templateService *BPMNTemplateService
	if len(templateServices) > 0 {
		templateService = templateServices[0]
	}
	return &BPMNAIGeneratorService{
		llmGateway:        llmGateway,
		deploymentService: deploymentService,
		templateService:   templateService,
		parser:            NewBPMNParser(),
	}
}

// GenerateBPMN 根据用户需求生成BPMN流程
func (s *BPMNAIGeneratorService) GenerateBPMN(ctx context.Context, req *dto.GenerateBPMNRequest, autoDeploy bool) (*dto.GenerateBPMNResponse, error) {
	// 构建Prompt
	prompt := s.buildGenerationPrompt(req)

	// 调用LLM生成BPMN
	messages := []LLMMessage{
		{
			Role:    "system",
			Content: getSystemPrompt(),
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// An empty model delegates selection to the configured provider.  Hard-coding
	// gpt-4o here made BPMN generation incompatible with MiniMax/local deployments.
	response, err := s.llmGateway.Chat(ctx, "", messages)
	if err != nil {
		return nil, fmt.Errorf("调用AI生成BPMN失败: %w", err)
	}

	// 解析响应，提取BPMN XML和元数据
	bpmnXML, metadata, err := s.parseLLMResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	// 验证生成的BPMN XML
	if err := s.parser.ValidateBPMNXML([]byte(bpmnXML)); err != nil {
		return nil, fmt.Errorf("生成的BPMN XML验证失败: %w", err)
	}

	// 解析BPMN提取信息
	definitions, err := s.parser.ParseXML([]byte(bpmnXML))
	if err != nil {
		return nil, fmt.Errorf("解析生成的BPMN XML失败: %w", err)
	}

	processInfo := s.parser.ExtractProcessInfo(definitions)
	if processInfo == nil {
		return nil, fmt.Errorf("无法提取流程信息")
	}

	// 构建响应
	resp := &dto.GenerateBPMNResponse{
		BPMNXML:             bpmnXML,
		ProcessID:           processInfo["id"].(string),
		ProcessName:         processInfo["name"].(string),
		ProcessDescription:  metadata.Explanation,
		Version:             "1.0.0",
		NodeCount:           metadata.NodeCount,
		Complexity:          metadata.Complexity,
		Explanation:         metadata.Explanation,
		CandidateDefinition: buildBusinessProcessCandidate(req),
	}
	if err := validateBusinessProcessCandidate(resp.CandidateDefinition); err != nil {
		return nil, fmt.Errorf("候选业务流程定义校验失败: %w", err)
	}

	// 生成后自动 Lint：语义层检查（连通性/网关/任务配置），
	// 结果随响应返回给调用方；error 级问题阻断自动部署。
	lintResult, err := NewBPMNLintService().LintBPMNXML([]byte(bpmnXML))
	if err != nil {
		// Lint 解析失败不影响返回（前面 ValidateBPMNXML 已通过），记录为解析级问题
		resp.LintResult = &dto.BPMNLintResult{
			HasErrors: true,
			Issues: []*dto.BPMNLintIssue{{
				Severity: "error",
				Category: "structure",
				Message:  "Lint 校验失败: " + err.Error(),
			}},
		}
		resp.LintResult.ErrorCount = 1
	} else {
		resp.LintResult = lintResult
	}

	// 如果需要自动部署（error 级 Lint 问题时拒绝，防止部署引擎无法正确执行的流程）
	if autoDeploy && !resp.LintResult.HasErrors {
		deployReq := &DeployProcessDefinitionRequest{
			Name:        resp.ProcessName,
			Description: resp.ProcessDescription,
			BPMNXML:     bpmnXML,
			TenantID:    req.TenantID,
		}

		deployment, err := s.deploymentService.DeployProcessDefinition(ctx, deployReq)
		if err != nil {
			return nil, fmt.Errorf("自动部署流程失败: %w", err)
		}

		resp.DeploymentID = deployment.DeploymentID
		definition, err := s.deploymentService.GetProcessDefinitionForDeployment(ctx, deployment.ID, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("自动部署成功但无法读取流程定义: %w", err)
		}
		resp.ProcessDefinitionID = definition.ID
		resp.Version = definition.Version
	}

	return resp, nil
}

// SuggestTemplates returns tenant-scoped and immutable built-in template matches.
func (s *BPMNAIGeneratorService) SuggestTemplates(ctx context.Context, tenantID int, keyword, processType string) ([]*dto.BPMNTemplateSuggestion, error) {
	if s.templateService == nil {
		return nil, fmt.Errorf("工作流模板目录未配置")
	}
	result := make([]*dto.BPMNTemplateSuggestion, 0)
	if s.templateCatalog != nil {
		custom, err := s.templateCatalog.List(ctx, tenantID, keyword, processType, "published", 1, 20)
		if err != nil {
			return nil, err
		}
		for _, item := range custom.Items {
			result = append(result, &dto.BPMNTemplateSuggestion{ID: item.Key, Name: item.Name, Description: item.Description, ProcessType: item.Domain, Score: 1})
		}
	}
	builtIns, err := s.templateService.SuggestTemplates(ctx, tenantID, keyword, processType)
	if err != nil {
		return nil, err
	}
	return append(result, builtIns...), nil
}

// buildBusinessProcessCandidate keeps business semantics explicit while the
// AI supplies BPMN structure. The existing designer must review this object
// before publication.
func buildBusinessProcessCandidate(req *dto.GenerateBPMNRequest) *dto.BusinessProcessCandidate {
	form := map[string]interface{}{"fields": []map[string]interface{}{}}
	approval := map[string]interface{}{"required": req.IncludeApprovals, "mode": "sequential", "rules": []interface{}{}}
	bindings := map[string]interface{}{"requester": "user", "organization": "department"}
	switch req.ProcessType {
	case "leave":
		form["fields"] = businessFields("leaveType:enum", "startAt:date", "endAt:date", "reason:string")
		bindings["approver"] = "direct_manager"
	case "expense":
		form["fields"] = businessFields("amount:number", "currency:enum", "costCenter:string", "invoice:file")
		approval["rules"] = []interface{}{map[string]interface{}{"when": "amount < 5000", "role": "department_manager"}, map[string]interface{}{"when": "amount >= 5000", "role": "finance_manager"}}
		bindings["costCenter"] = "cost_center"
	case "hr":
		form["fields"] = businessFields("employee:reference", "department:reference", "effectiveAt:date", "reason:string")
		bindings["employee"] = "user"
	case "procurement":
		form["fields"] = businessFields("item:string", "quantity:number", "amount:number", "vendor:reference", "budgetCode:string")
		approval["rules"] = []interface{}{map[string]interface{}{"when": "amount < 10000", "role": "budget_owner"}, map[string]interface{}{"when": "amount >= 10000", "role": "cfo"}}
		bindings["vendor"] = "supplier"
	case "it":
		form["fields"] = businessFields("service:reference", "environment:enum", "impact:enum", "implementationPlan:string")
		bindings["service"] = "service_catalog"
	}
	sla := map[string]interface{}{}
	if req.IncludeSLA {
		sla = map[string]interface{}{"enabled": true, "responseHours": 24, "resolutionHours": 72}
	}
	return &dto.BusinessProcessCandidate{Domain: req.ProcessType, FormSchema: form, ApprovalPolicy: approval, OntologyBindings: bindings, SLAConfig: sla, RequiresConfirmation: true}
}

func businessFields(specs ...string) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		fields = append(fields, map[string]interface{}{"name": parts[0], "type": parts[1], "required": true})
	}
	return fields
}

// validateBusinessProcessCandidate is deliberately deterministic. It checks
// the contract envelope before an AI-produced BPMN definition can be
// auto-deployed; policy expressions are restricted to the supported amount
// comparison grammar for now.
func validateBusinessProcessCandidate(candidate *dto.BusinessProcessCandidate) error {
	if candidate == nil || candidate.Domain == "" || candidate.FormSchema == nil || candidate.ApprovalPolicy == nil || candidate.OntologyBindings == nil {
		return fmt.Errorf("缺少业务域、表单、本体或审批策略")
	}
	if fields, ok := candidate.FormSchema["fields"].([]map[string]interface{}); !ok {
		return fmt.Errorf("formSchema.fields 必须是结构化字段列表")
	} else {
		for _, field := range fields {
			name, _ := field["name"].(string)
			typ, _ := field["type"].(string)
			if name == "" || typ == "" || field["required"] != true {
				return fmt.Errorf("字段定义不完整")
			}
		}
	}
	if rules, exists := candidate.ApprovalPolicy["rules"]; exists {
		items, ok := rules.([]interface{})
		if !ok {
			return fmt.Errorf("approvalPolicy.rules 必须是数组")
		}
		for _, item := range items {
			rule, ok := item.(map[string]interface{})
			when, _ := rule["when"].(string)
			role, _ := rule["role"].(string)
			if !ok || role == "" || !regexp.MustCompile(`^amount\s*(<|<=|>=|>)\s*[0-9]+$`).MatchString(when) {
				return fmt.Errorf("审批条件仅支持 amount 数值比较")
			}
		}
	}
	return nil
}

// PreviewBPMN 预览流程结构，不生成完整XML
func (s *BPMNAIGeneratorService) PreviewBPMN(ctx context.Context, req *dto.PreviewBPMNRequest) (*dto.PreviewBPMNResponse, error) {
	// 构建预览Prompt
	prompt := s.buildPreviewPrompt(req)

	// 调用LLM
	messages := []LLMMessage{
		{
			Role:    "system",
			Content: getPreviewSystemPrompt(),
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Keep preview on the same deployment-configured model as generation.
	response, err := s.llmGateway.Chat(ctx, "", messages)
	if err != nil {
		return nil, fmt.Errorf("调用AI预览流程失败: %w", err)
	}

	// 解析JSON响应
	var previewResp dto.PreviewBPMNResponse
	if err := json.Unmarshal([]byte(response), &previewResp); err != nil {
		return nil, fmt.Errorf("解析预览响应失败: %w", err)
	}

	return &previewResp, nil
}

// buildGenerationPrompt 构建生成Prompt
func (s *BPMNAIGeneratorService) buildGenerationPrompt(req *dto.GenerateBPMNRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("业务需求描述: %s\n", req.Requirement))
	sb.WriteString(fmt.Sprintf("流程类型: %s\n", req.ProcessType))
	sb.WriteString(fmt.Sprintf("企业类型: %s\n", req.EnterpriseType))
	sb.WriteString(fmt.Sprintf("包含SLA配置: %t\n", req.IncludeSLA))
	sb.WriteString(fmt.Sprintf("包含通知配置: %t\n", req.IncludeNotifications))
	sb.WriteString(fmt.Sprintf("包含审批节点: %t\n", req.IncludeApprovals))
	sb.WriteString("\n请根据以上信息生成符合BPMN 2.0规范的工作流定义。\n")

	return sb.String()
}

// buildPreviewPrompt 构建预览Prompt
func (s *BPMNAIGeneratorService) buildPreviewPrompt(req *dto.PreviewBPMNRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("业务需求描述: %s\n", req.Requirement))
	sb.WriteString(fmt.Sprintf("流程类型: %s\n", req.ProcessType))
	sb.WriteString(fmt.Sprintf("企业类型: %s\n", req.EnterpriseType))
	sb.WriteString("\n请分析这个业务流程，返回结构化的预览信息。\n")

	return sb.String()
}

// parseLLMResponse 解析LLM响应，提取BPMN XML和元数据
func (s *BPMNAIGeneratorService) parseLLMResponse(response string) (string, *GenerationMetadata, error) {
	// 提取BPMN XML部分
	bpmnRegex := regexp.MustCompile(`(?s)<\?xml.*?<bpmn:definitions.*?</bpmn:definitions>`)
	bpmnMatches := bpmnRegex.FindStringSubmatch(response)
	if len(bpmnMatches) == 0 {
		return "", nil, fmt.Errorf("未在响应中找到BPMN XML内容")
	}
	bpmnXML := bpmnMatches[0]

	// 提取元数据部分
	metadataRegex := regexp.MustCompile("(?s)```json\\n(.*?)\\n```")
	metadataMatches := metadataRegex.FindStringSubmatch(response)
	var metadata GenerationMetadata
	if len(metadataMatches) > 0 {
		if err := json.Unmarshal([]byte(metadataMatches[1]), &metadata); err != nil {
			// 如果解析失败，尝试从BPMN中提取基本信息
			metadata = GenerationMetadata{
				NodeCount:   strings.Count(bpmnXML, "<bpmn:"),
				Complexity:  "medium",
				Explanation: "AI生成的BPMN流程",
			}
		}
	} else {
		// 没有元数据部分，从BPMN中提取
		metadata = GenerationMetadata{
			NodeCount:   strings.Count(bpmnXML, "<bpmn:"),
			Complexity:  "medium",
			Explanation: "AI生成的BPMN流程",
		}
	}

	return bpmnXML, &metadata, nil
}

// GenerationMetadata 生成的元数据
type GenerationMetadata struct {
	NodeCount   int    `json:"nodeCount"`
	Complexity  string `json:"complexity"`
	Explanation string `json:"explanation"`
}

// getSystemPrompt 获取系统提示词
func getSystemPrompt() string {
	return "你是一个专业的BPMN 2.0流程生成专家，专为企业级ITSM系统生成符合规范的工作流定义。\n\n" +
		"输出要求：\n" +
		"1. 首先输出完整的BPMN 2.0 XML，必须包含完整的命名空间声明\n" +
		"2. 然后输出JSON格式的元数据，包含node_count, complexity, explanation字段\n" +
		"3. BPMN XML必须符合以下规范：\n" +
		"   - 使用<bpmn:definitions>作为根节点，包含所有必需的命名空间\n" +
		"   - 流程ID使用小写字母和下划线，例如\"incident_escalation_flow\"\n" +
		"   - 节点ID有明确含义，例如\"StartEvent_Report\", \"Activity_AssignEngineer\"\n" +
		"   - 包含必要的扩展元数据，如category, version, description\n" +
		"   - 支持lib-bpmn-engine引擎的执行\n" +
		"   - 对于服务任务，使用implementation=\"##WebService\"\n" +
		"   - 对于用户任务，包含适当的角色分配配置\n" +
		"   - 网关节点有明确的条件表达式\n\n" +
		"BPMN结构要求：\n" +
		"- 必须包含一个开始事件和至少一个结束事件\n" +
		"- 流程逻辑完整，没有断开的连接线\n" +
		"- 包含适当的网关处理分支逻辑\n" +
		"- 如果需要SLA，在节点元数据中添加sla_minutes字段\n" +
		"- 如果需要通知，在节点元数据中添加notification_channel和notify_roles字段\n" +
		"- 如果需要审批，添加userTask节点并配置审批角色\n\n" +
		"命名空间要求：\n" +
		"xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\"\n" +
		"xmlns:bpmndi=\"http://www.omg.org/spec/BPMN/20100524/DI\"\n" +
		"xmlns:dc=\"http://www.omg.org/spec/DD/20100524/DC\"\n" +
		"xmlns:camunda=\"http://camunda.org/schema/1.0/bpmn\"\n\n" +
		"输出格式示例：\n" +
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<bpmn:definitions xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\"\n" +
		"                 xmlns:bpmndi=\"http://www.omg.org/spec/BPMN/20100524/DI\"\n" +
		"                 xmlns:dc=\"http://www.omg.org/spec/DD/20100524/DC\"\n" +
		"                 xmlns:camunda=\"http://camunda.org/schema/1.0/bpmn\"\n" +
		"                 targetNamespace=\"http://bpmn.io/schema/bpmn\">\n" +
		"  <bpmn:process id=\"process_id\" name=\"流程名称\" isExecutable=\"true\">\n" +
		"    <!-- 流程内容 -->\n" +
		"  </bpmn:process>\n" +
		"</bpmn:definitions>\n\n" +
		"```json\n" +
		"{\n" +
		"  \"node_count\": 15,\n" +
		"  \"complexity\": \"medium\",\n" +
		"  \"explanation\": \"生成的流程包含故障上报、自动分派、工程师处理、审批、关闭等完整节点，适用于企业级IT服务管理场景\"\n" +
		"}\n" +
		"```\n\n" +
		"请确保输出的XML是完整且可执行的，符合lib-bpmn-engine的要求。"
}

// getPreviewSystemPrompt 获取预览系统提示词
func getPreviewSystemPrompt() string {
	return "你是一个专业的业务流程分析专家，请根据用户的业务需求描述，分析并返回结构化的流程预览信息。\n\n" +
		"输出要求：\n" +
		"1. 仅返回JSON格式，不包含其他内容\n" +
		"2. JSON结构如下：\n" +
		"{\n" +
		"  \"structure_description\": \"流程结构的详细描述\",\n" +
		"  \"nodes\": [\n" +
		"    {\n" +
		"      \"id\": \"节点ID\",\n" +
		"      \"name\": \"节点名称\",\n" +
		"      \"type\": \"节点类型\",\n" +
		"      \"description\": \"节点描述\",\n" +
		"      \"assignee_role\": \"处理角色（可选）\",\n" +
		"      \"sla_minutes\": SLA时间（可选）\n" +
		"    }\n" +
		"  ],\n" +
		"  \"complexity\": \"low/medium/high\",\n" +
		"  \"estimated_node_count\": 预估节点数量,\n" +
		"  \"use_cases\": \"适用场景说明\",\n" +
		"  \"suggestions\": [\"优化建议1\", \"优化建议2\"]\n" +
		"}\n\n" +
		"节点类型可选值：startEvent, endEvent, userTask, serviceTask, exclusiveGateway, parallelGateway, scriptTask, receiveTask, sendTask\n\n" +
		"请根据业务需求合理设计流程结构，确保符合企业级ITSM的最佳实践。"
}
