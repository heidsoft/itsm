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
	parser            *BPMNParser
}

// NewBPMNAIGeneratorService 创建AI生成服务实例
func NewBPMNAIGeneratorService(llmGateway *LLMGateway, deploymentService *BPMNDeploymentService) *BPMNAIGeneratorService {
	return &BPMNAIGeneratorService{
		llmGateway:        llmGateway,
		deploymentService: deploymentService,
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

	response, err := s.llmGateway.Chat(ctx, "gpt-4o", messages)
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
		BPMNXML:            bpmnXML,
		ProcessID:          processInfo["id"].(string),
		ProcessName:        processInfo["name"].(string),
		ProcessDescription: metadata.Explanation,
		Version:            "1.0.0",
		NodeCount:          metadata.NodeCount,
		Complexity:         metadata.Complexity,
		Explanation:        metadata.Explanation,
	}

	// 如果需要自动部署
	if autoDeploy {
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
		// 这里需要查询对应的流程定义ID，暂时留空
		// resp.ProcessDefinitionID = processDef.ID
	}

	return resp, nil
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

	response, err := s.llmGateway.Chat(ctx, "gpt-4o", messages)
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
	NodeCount   int    `json:"node_count"`
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
