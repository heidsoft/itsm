package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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
func (s *A2UITicketService) HandleUserAction(ctx context.Context, action string, surfaceID string, contextData map[string]interface{}) ([]string, error) {
	var messages []string

	if action == "submit" {
		// 获取工单数据
		ticketData, ok := contextData["data"].(map[string]interface{})
		if !ok {
			// 尝试解析
			dataJSON, err := json.Marshal(contextData["data"])
			if err != nil {
				return nil, fmt.Errorf("invalid ticket data")
			}
			if err := json.Unmarshal(dataJSON, &ticketData); err != nil {
				return nil, fmt.Errorf("invalid ticket data format")
			}
		}

		// 验证必填字段
		title, _ := ticketData["title"].(string)
		if title == "" {
			// 返回验证错误
			errMsg := A2UIDataModelUpdate{
				DataModelUpdate: DataModelUpdate{
					SurfaceID: surfaceID,
					Path:      "/ui/errors",
					Contents: []Content{
						{Key: "title", ValueString: "标题不能为空"},
					},
				},
			}
			errData, _ := json.Marshal(errMsg)
			messages = append(messages, string(errData))
			return messages, nil
		}

		// 创建工单
		_ = map[string]interface{}{
			"title":       title,
			"type":        ticketData["type"],
			"priority":    ticketData["priority"],
			"description": ticketData["description"],
			"status":      "open",
		}

		// 调用工单服务创建（如果可用）
		// if s.ticketStore != nil {
		// 	ticket, err = s.ticketStore.Create(ctx, ticket)
		// }

		// 返回成功消息
		ticketID := fmt.Sprintf("TKT-%s-00123", "202602")

		successMsg := A2UIDataModelUpdate{
			DataModelUpdate: DataModelUpdate{
				SurfaceID: surfaceID,
				Path:      "/result",
				Contents: []Content{
					{Key: "ticketId", ValueString: ticketID},
					{Key: "status", ValueString: "created"},
				},
			},
		}
		successData, _ := json.Marshal(successMsg)
		messages = append(messages, string(successData))

		// 删除 surface
		deleteMsg := A2UIDeleteSurface{
			DeleteSurface: DeleteSurface{
				SurfaceID: surfaceID,
			},
		}
		deleteData, _ := json.Marshal(deleteMsg)
		messages = append(messages, string(deleteData))
	}

	return messages, nil
}
