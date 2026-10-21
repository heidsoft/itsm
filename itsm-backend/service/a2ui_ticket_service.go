package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// ErrA2UISubmitNotImplemented：A2UI 表单「提交」动作的诚实语义哨兵（UPGRADE §1.48）。
// A2UI submit 从未真正创建工单（伪造 TKT-xxx 编号 + status=created 假成功）；
// 真创建必须经 ticket 领域公开 service 接口完成 requester/审计/SLA/工作流绑定，
// 属产品待办。在此之前未实现的渠道必须显式 501（NotImplementedCode），不得伪装成功。
var ErrA2UISubmitNotImplemented = errors.New("a2ui ticket submit channel not wired into ticket domain")

// ==================== A2UI 消息结构 ====================

// SurfaceUpdate 消息
type A2UISurfaceUpdate struct {
	SurfaceUpdate SurfaceUpdate `json:"surfaceUpdate"`
}

type SurfaceUpdate struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

type Component struct {
	ID        string       `json:"id"`
	Component ComponentDef `json:"component"`
}

type ComponentDef map[string]interface{}

// DataModelUpdate 消息
type A2UIDataModelUpdate struct {
	DataModelUpdate DataModelUpdate `json:"dataModelUpdate"`
}

type DataModelUpdate struct {
	SurfaceID string    `json:"surfaceId"`
	Path      string    `json:"path"`
	Contents  []Content `json:"contents"`
}

type Content struct {
	Key         string `json:"key"`
	ValueString string `json:"valueString,omitempty"`
	ValueNumber *int   `json:"valueNumber,omitempty"`
	ValueBool   *bool  `json:"valueBoolean,omitempty"`
}

// BeginRendering 消息
type A2UIBeginRendering struct {
	BeginRendering BeginRendering `json:"beginRendering"`
}

type BeginRendering struct {
	SurfaceID string `json:"surfaceId"`
	Root      string `json:"root"`
}

// DeleteSurface 消息
type A2UIDeleteSurface struct {
	DeleteSurface DeleteSurface `json:"deleteSurface"`
}

type DeleteSurface struct {
	SurfaceID string `json:"surfaceId"`
}

// ==================== 工单表单服务 ====================

type A2UITicketService struct {
	llmGateway *LLMGateway
}

// 新的工单表单服务
func NewA2UITicketService(llmGateway *LLMGateway) *A2UITicketService {
	return &A2UITicketService{
		llmGateway: llmGateway,
	}
}

// 生成工单表单消息序列
func (s *A2UITicketService) GenerateFormMessages(ctx context.Context, userIntent string, surfaceID string) ([]string, error) {
	var messages []string

	// 1. 解析用户意图，提取字段
	extracted := s.extractFieldsFromIntent(userIntent)

	// 2. 生成 surfaceUpdate - 定义表单结构
	surfaceMsg := s.buildSurfaceUpdateMessage(surfaceID, extracted)
	messages = append(messages, surfaceMsg)

	// 3. 生成 dataModelUpdate - 初始化数据模型
	initMsg := s.buildInitialDataModelMessage(surfaceID, extracted)
	messages = append(messages, initMsg)

	// 4. 生成 beginRendering - 触发渲染
	renderMsg := s.buildBeginRenderingMessage(surfaceID)
	messages = append(messages, renderMsg)

	return messages, nil
}

// 构建表单组件定义
func (s *A2UITicketService) buildSurfaceUpdateMessage(surfaceID string, fields map[string]string) string {
	components := []Component{
		// 根布局
		{
			ID: "root",
			Component: ComponentDef{
				"Column": map[string]interface{}{
					"children": map[string]interface{}{
						"explicitList": []string{"header", "form-card"},
					},
				},
			},
		},
		// Header
		{
			ID: "header",
			Component: ComponentDef{
				"Text": map[string]interface{}{
					"text":      map[string]string{"literalString": "创建工单"},
					"usageHint": "h1",
				},
			},
		},
		// Form Card
		{
			ID: "form-card",
			Component: ComponentDef{
				"Card": map[string]interface{}{
					"child": "form-column",
				},
			},
		},
		// Form Column
		{
			ID: "form-column",
			Component: ComponentDef{
				"Column": map[string]interface{}{
					"children": map[string]interface{}{
						"explicitList": []string{
							"title-field", "type-field", "priority-field",
							"desc-field", "submit-row",
						},
					},
				},
			},
		},
		// 标题字段
		{
			ID: "title-field",
			Component: ComponentDef{
				"TextField": map[string]interface{}{
					"label":       map[string]string{"literalString": "标题"},
					"text":        map[string]string{"path": "/ticket/title"},
					"placeholder": map[string]string{"literalString": "请输入工单标题"},
				},
			},
		},
		// 类型选择
		{
			ID: "type-field",
			Component: ComponentDef{
				"PickerSelect": map[string]interface{}{
					"label":     map[string]string{"literalString": "工单类型"},
					"selection": map[string]string{"path": "/ticket/type"},
					"options": map[string]interface{}{
						"explicitList": []map[string]string{
							{"id": "hardware", "text": "硬件申请"},
							{"id": "software", "text": "软件安装"},
							{"id": "network", "text": "网络问题"},
							{"id": "access", "text": "权限申请"},
							{"id": "other", "text": "其他"},
						},
					},
				},
			},
		},
		// 优先级选择
		{
			ID: "priority-field",
			Component: ComponentDef{
				"PickerSelect": map[string]interface{}{
					"label":     map[string]string{"literalString": "优先级"},
					"selection": map[string]string{"path": "/ticket/priority"},
					"options": map[string]interface{}{
						"explicitList": []map[string]string{
							{"id": "low", "text": "低"},
							{"id": "medium", "text": "中"},
							{"id": "high", "text": "高"},
							{"id": "urgent", "text": "紧急"},
						},
					},
				},
			},
		},
		// 描述字段
		{
			ID: "desc-field",
			Component: ComponentDef{
				"TextAreaField": map[string]interface{}{
					"label": map[string]string{"literalString": "详细描述"},
					"text":  map[string]string{"path": "/ticket/description"},
					"rows":  4,
				},
			},
		},
		// 提交按钮行
		{
			ID: "submit-row",
			Component: ComponentDef{
				"Row": map[string]interface{}{
					"children": map[string]interface{}{
						"explicitList": []string{"submit-btn"},
					},
				},
			},
		},
		// 提交按钮
		{
			ID: "submit-btn",
			Component: ComponentDef{
				"Button": map[string]interface{}{
					"child": "submit-text",
					"action": map[string]interface{}{
						"name": "submit",
						"context": []map[string]interface{}{
							{"key": "data", "value": map[string]string{"path": "/ticket"}},
						},
					},
				},
			},
		},
		// 按钮文字
		{
			ID: "submit-text",
			Component: ComponentDef{
				"Text": map[string]interface{}{
					"text": map[string]string{"literalString": "提交工单"},
				},
			},
		},
	}

	msg := A2UISurfaceUpdate{
		SurfaceUpdate: SurfaceUpdate{
			SurfaceID:  surfaceID,
			Components: components,
		},
	}

	data, _ := json.Marshal(msg)
	return string(data)
}

// 构建初始数据模型消息
func (s *A2UITicketService) buildInitialDataModelMessage(surfaceID string, fields map[string]string) string {
	var contents []Content

	// 默认值
	contents = append(contents, Content{Key: "title", ValueString: ""})
	contents = append(contents, Content{Key: "type", ValueString: "hardware"})
	contents = append(contents, Content{Key: "priority", ValueString: "medium"})
	contents = append(contents, Content{Key: "description", ValueString: ""})

	// 从意图提取的值
	if v, ok := fields["title"]; ok && v != "" {
		contents = append(contents, Content{Key: "title", ValueString: v})
	}
	if v, ok := fields["type"]; ok && v != "" {
		contents = append(contents, Content{Key: "type", ValueString: v})
	}
	if v, ok := fields["priority"]; ok && v != "" {
		contents = append(contents, Content{Key: "priority", ValueString: v})
	}
	if v, ok := fields["description"]; ok && v != "" {
		contents = append(contents, Content{Key: "description", ValueString: v})
	}

	msg := A2UIDataModelUpdate{
		DataModelUpdate: DataModelUpdate{
			SurfaceID: surfaceID,
			Path:      "/ticket",
			Contents:  contents,
		},
	}

	data, _ := json.Marshal(msg)
	return string(data)
}

// 构建渲染消息
func (s *A2UITicketService) buildBeginRenderingMessage(surfaceID string) string {
	msg := A2UIBeginRendering{
		BeginRendering: BeginRendering{
			SurfaceID: surfaceID,
			Root:      "root",
		},
	}

	data, _ := json.Marshal(msg)
	return string(data)
}

// 从用户意图提取字段（基于规则 + LLM）
func (s *A2UITicketService) extractFieldsFromIntent(intent string) map[string]string {
	result := make(map[string]string)

	// 预处理
	intentLower := strings.ToLower(intent)

	// 优先级识别
	if strings.Contains(intentLower, "紧急") || strings.Contains(intentLower, "urgent") ||
		strings.Contains(intentLower, "asap") || strings.Contains(intentLower, "尽快") {
		result["priority"] = "urgent"
	} else if strings.Contains(intentLower, "重要") || strings.Contains(intentLower, "急") ||
		strings.Contains(intentLower, "高优先级") {
		result["priority"] = "high"
	} else if strings.Contains(intentLower, "普通") || strings.Contains(intentLower, "一般") {
		result["priority"] = "medium"
	} else {
		result["priority"] = "low"
	}

	// 类型识别
	if strings.Contains(intentLower, "电脑") || strings.Contains(intentLower, "笔记本") ||
		len(strings.Fields(intentLower)) > 0 {
		// 硬件相关
		hardwareKeywords := []string{"电脑", "笔记本", "显示器", "键盘", "鼠标", "打印机", "硬件"}
		for _, kw := range hardwareKeywords {
			if strings.Contains(intentLower, kw) {
				result["type"] = "hardware"
				goto typeMatched
			}
		}
	}

	if strings.Contains(intentLower, "软件") || strings.Contains(intentLower, "安装") ||
		strings.Contains(intentLower, "license") || strings.Contains(intentLower, "许可证") {
		result["type"] = "software"
		goto typeMatched
	}

	if strings.Contains(intentLower, "网络") || strings.Contains(intentLower, "wifi") ||
		strings.Contains(intentLower, "上网") || strings.Contains(intentLower, "网线") {
		result["type"] = "network"
		goto typeMatched
	}

	if strings.Contains(intentLower, "权限") || strings.Contains(intentLower, "账号") ||
		strings.Contains(intentLower, "账户") || strings.Contains(intentLower, "vpn") {
		result["type"] = "access"
		goto typeMatched
	}

	// 默认类型
	result["type"] = "other"

typeMatched:

	// 标题提取 - 简化处理
	if len(intent) > 0 && len(intent) < 100 {
		// 尝试提取关键词生成标题
		title := intent
		if len(title) > 50 {
			// 截取
			title = title[:50]
		}
		result["title"] = title
	}

	// 描述
	result["description"] = intent

	return result
}

// 处理用户提交操作
// B11（UPGRADE §1.48）：「submit」动作为未接线能力。此前实现从未真正创建工单，
// 而是伪造 TKT-yyyymm-00123 编号并回传 status=created 假成功，让 /tickets/ai-create
// 页面展示「工单创建成功！」——违反能力状态语义（未实现不得伪装 ready）。
// 真创建需经 ticket 领域公开 service 接口完成 requester/审计/SLA/工作流绑定，属产品待办。
func (s *A2UITicketService) HandleUserAction(ctx context.Context, action string, surfaceID string, contextData map[string]interface{}) ([]string, error) {
	var messages []string

	if action == "submit" {
		// 未接线能力显式失败：handler 映射为 501/NotImplementedCode。
		return nil, ErrA2UISubmitNotImplemented
	}

	return messages, nil
}

// ParseTicketIntent B9: 解析自然语言描述，返回结构化字段供前端预填表单
// 不直接创建工单，保持用户可控
func (s *A2UITicketService) ParseTicketIntent(ctx context.Context, description string) (map[string]interface{}, error) {
	fields := s.extractFieldsFromIntent(description)

	// 提取 title：取 description 第一行前 30 字符
	title := description
	if idx := strings.Index(description, "\n"); idx > 0 && idx < 100 {
		title = strings.TrimSpace(description[:idx])
	}
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[:30])
	}
	if fields["title"] == "" {
		fields["title"] = title
	}

	// confidence 基于是否提取到 category/priority
	confidence := 0.55
	if fields["category"] != "" {
		confidence += 0.15
	}
	if fields["priority"] != "" {
		confidence += 0.15
	}
	if fields["title"] != "" {
		confidence += 0.10
	}
	if confidence > 0.95 {
		confidence = 0.95
	}

	// B11（UPGRADE §1.48）：API 边界字段统一 camelCase（此前为 suggested_title 等
	// snake_case 键，违反 HTTP 字段命名规范；前端当前无该接口消费方，无消费兼容影响）。
	return map[string]interface{}{
		"suggestedTitle":    fields["title"],
		"suggestedCategory": fields["category"],
		"suggestedPriority": fields["priority"],
		"description":       description,
		"confidence":        confidence,
		"reasoning":         "A2UI intent parser (keyword + length heuristics)",
		"surfaceId":         "ticket-create",
		"actionUrl":         "/tickets/create",
	}, nil
}
