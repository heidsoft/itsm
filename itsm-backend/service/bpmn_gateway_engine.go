package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
)

// GatewayEngine BPMN网关执行引擎
type GatewayEngine struct {
	client *ent.Client
}

// NewGatewayEngine 创建网关执行引擎实例
func NewGatewayEngine(client *ent.Client) *GatewayEngine {
	return &GatewayEngine{
		client: client,
	}
}

// GatewayType 网关类型
type GatewayType string

const (
	GatewayTypeExclusive GatewayType = "exclusive" // 排他网关
	GatewayTypeParallel  GatewayType = "parallel"  // 并行网关
	GatewayTypeInclusive GatewayType = "inclusive" // 包容网关
)

// GatewayExecutionRequest 网关执行请求
type GatewayExecutionRequest struct {
	ProcessInstanceID string                 `json:"process_instance_id" binding:"required"`
	GatewayID         string                 `json:"gateway_id" binding:"required"`
	GatewayType       GatewayType            `json:"gateway_type" binding:"required"`
	Variables         map[string]interface{} `json:"variables"`
	TenantID          int                    `json:"tenant_id" binding:"required"`
}

// GatewayExecutionResult 网关执行结果
type GatewayExecutionResult struct {
	NextActivities []string               `json:"next_activities"`
	Variables      map[string]interface{} `json:"variables"`
	ExecutedAt     time.Time              `json:"executed_at"`
}

// ExecuteGateway 执行网关逻辑
func (e *GatewayEngine) ExecuteGateway(ctx context.Context, req *GatewayExecutionRequest) (*GatewayExecutionResult, error) {
	switch req.GatewayType {
	case GatewayTypeExclusive:
		return e.executeExclusiveGateway(ctx, req)
	case GatewayTypeParallel:
		return e.executeParallelGateway(ctx, req)
	case GatewayTypeInclusive:
		return e.executeInclusiveGateway(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的网关类型: %s", req.GatewayType)
	}
}

// executeExclusiveGateway 执行排他网关
func (e *GatewayEngine) executeExclusiveGateway(ctx context.Context, req *GatewayExecutionRequest) (*GatewayExecutionResult, error) {
	// 获取流程实例
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(req.ProcessInstanceID)).
		Where(processinstance.TenantID(req.TenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 获取流程定义
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(instance.ProcessDefinitionID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 解析流程定义中的网关信息
	gatewayInfo, err := e.extractGatewayInfo(definition.BpmnXML, req.GatewayID)
	if err != nil {
		return nil, fmt.Errorf("提取网关信息失败: %w", err)
	}

	// 评估条件表达式，选择下一个活动
	nextActivity, err := e.evaluateExclusiveGatewayConditions(ctx, req, gatewayInfo)
	if err != nil {
		return nil, fmt.Errorf("评估排他网关条件失败: %w", err)
	}

	// 记录网关执行历史
	err = e.recordGatewayExecution(ctx, req, []string{nextActivity}, "exclusive")
	if err != nil {
		return nil, fmt.Errorf("记录网关执行历史失败: %w", err)
	}

	return &GatewayExecutionResult{
		NextActivities: []string{nextActivity},
		Variables:      req.Variables,
		ExecutedAt:     time.Now(),
	}, nil
}

// executeParallelGateway 执行并行网关
func (e *GatewayEngine) executeParallelGateway(ctx context.Context, req *GatewayExecutionRequest) (*GatewayExecutionResult, error) {
	// 获取流程实例
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(req.ProcessInstanceID)).
		Where(processinstance.TenantID(req.TenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 获取流程定义
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(instance.ProcessDefinitionID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 解析流程定义中的网关信息
	gatewayInfo, err := e.extractGatewayInfo(definition.BpmnXML, req.GatewayID)
	if err != nil {
		return nil, fmt.Errorf("提取网关信息失败: %w", err)
	}

	// 并行网关激活所有输出流
	nextActivities := e.getParallelGatewayOutputs(gatewayInfo)

	// 记录网关执行历史
	err = e.recordGatewayExecution(ctx, req, nextActivities, "parallel")
	if err != nil {
		return nil, fmt.Errorf("记录网关执行历史失败: %w", err)
	}

	return &GatewayExecutionResult{
		NextActivities: nextActivities,
		Variables:      req.Variables,
		ExecutedAt:     time.Now(),
	}, nil
}

// executeInclusiveGateway 执行包容网关
func (e *GatewayEngine) executeInclusiveGateway(ctx context.Context, req *GatewayExecutionRequest) (*GatewayExecutionResult, error) {
	// 获取流程实例
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(req.ProcessInstanceID)).
		Where(processinstance.TenantID(req.TenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 获取流程定义
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(instance.ProcessDefinitionID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 解析流程定义中的网关信息
	gatewayInfo, err := e.extractGatewayInfo(definition.BpmnXML, req.GatewayID)
	if err != nil {
		return nil, fmt.Errorf("提取网关信息失败: %w", err)
	}

	// 评估条件表达式，选择满足条件的输出流
	nextActivities, err := e.evaluateInclusiveGatewayConditions(ctx, req, gatewayInfo)
	if err != nil {
		return nil, fmt.Errorf("评估包容网关条件失败: %w", err)
	}

	// 记录网关执行历史
	err = e.recordGatewayExecution(ctx, req, nextActivities, "inclusive")
	if err != nil {
		return nil, fmt.Errorf("记录网关执行历史失败: %w", err)
	}

	return &GatewayExecutionResult{
		NextActivities: nextActivities,
		Variables:      req.Variables,
		ExecutedAt:     time.Now(),
	}, nil
}

// extractGatewayInfo 从流程定义中提取网关信息
func (e *GatewayEngine) extractGatewayInfo(definition interface{}, gatewayID string) (map[string]interface{}, error) {
	// 这里需要根据实际的流程定义格式来解析
	// 暂时返回一个示例结构
	return map[string]interface{}{
		"id":          gatewayID,
		"type":        "gateway",
		"outputs":     []string{"activity1", "activity2", "activity3"},
		"conditions":  []string{"condition1", "condition2", "condition3"},
		"defaultFlow": "activity3",
	}, nil
}

// evaluateExclusiveGatewayConditions 评估排他网关条件
func (e *GatewayEngine) evaluateExclusiveGatewayConditions(ctx context.Context, req *GatewayExecutionRequest, gatewayInfo map[string]interface{}) (string, error) {
	// 获取输出流和条件
	outputs, ok := gatewayInfo["outputs"].([]string)
	if !ok {
		return "", fmt.Errorf("网关输出流格式错误")
	}

	conditions, ok := gatewayInfo["conditions"].([]string)
	if !ok {
		return "", fmt.Errorf("网关条件格式错误")
	}

	defaultFlow, ok := gatewayInfo["defaultFlow"].(string)
	if !ok {
		defaultFlow = ""
	}

	// 评估每个条件
	for i, condition := range conditions {
		if i >= len(outputs) {
			break
		}

		// 这里应该实现条件表达式评估器
		// 暂时使用简单的字符串匹配作为示例
		if e.evaluateCondition(condition, req.Variables) {
			return outputs[i], nil
		}
	}

	// 如果没有条件满足，使用默认流
	if defaultFlow != "" {
		return defaultFlow, nil
	}

	// 如果没有默认流，选择第一个输出
	if len(outputs) > 0 {
		return outputs[0], nil
	}

	return "", fmt.Errorf("排他网关没有可用的输出流")
}

// evaluateInclusiveGatewayConditions 评估包容网关条件
func (e *GatewayEngine) evaluateInclusiveGatewayConditions(ctx context.Context, req *GatewayExecutionRequest, gatewayInfo map[string]interface{}) ([]string, error) {
	// 获取输出流和条件
	outputs, ok := gatewayInfo["outputs"].([]string)
	if !ok {
		return nil, fmt.Errorf("网关输出流格式错误")
	}

	conditions, ok := gatewayInfo["conditions"].([]string)
	if !ok {
		return nil, fmt.Errorf("网关条件格式错误")
	}

	var nextActivities []string

	// 评估每个条件，满足条件的都激活
	for i, condition := range conditions {
		if i >= len(outputs) {
			break
		}

		if e.evaluateCondition(condition, req.Variables) {
			nextActivities = append(nextActivities, outputs[i])
		}
	}

	// 如果没有条件满足，使用默认流
	if len(nextActivities) == 0 {
		defaultFlow, ok := gatewayInfo["defaultFlow"].(string)
		if ok && defaultFlow != "" {
			nextActivities = append(nextActivities, defaultFlow)
		}
	}

	// 如果仍然没有输出，选择第一个输出
	if len(nextActivities) == 0 && len(outputs) > 0 {
		nextActivities = append(nextActivities, outputs[0])
	}

	if len(nextActivities) == 0 {
		return nil, fmt.Errorf("包容网关没有可用的输出流")
	}

	return nextActivities, nil
}

// getParallelGatewayOutputs 获取并行网关的所有输出
func (e *GatewayEngine) getParallelGatewayOutputs(gatewayInfo map[string]interface{}) []string {
	outputs, ok := gatewayInfo["outputs"].([]string)
	if !ok {
		return []string{}
	}
	return outputs
}

// evaluateCondition 评估条件表达式
func (e *GatewayEngine) evaluateCondition(condition string, variables map[string]interface{}) bool {
	// 这里应该实现一个完整的条件表达式评估器
	// 暂时使用简单的字符串匹配作为示例

	// 检查条件是否包含变量引用
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			varName := strings.TrimSpace(parts[0])
			varValue := strings.TrimSpace(parts[1])

			// 移除引号
			varValue = strings.Trim(varValue, `"'`)

			if actualValue, exists := variables[varName]; exists {
				// 简单的字符串比较
				return fmt.Sprintf("%v", actualValue) == varValue
			}
		}
	}

	// 检查布尔值
	if condition == "true" {
		return true
	}
	if condition == "false" {
		return false
	}

	// 检查变量值
	if value, exists := variables[condition]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
		// 非零值视为true
		if value != nil && value != "" && value != 0 {
			return true
		}
	}

	return false
}

// recordGatewayExecution 记录网关执行历史
func (e *GatewayEngine) recordGatewayExecution(ctx context.Context, req *GatewayExecutionRequest, nextActivities []string, gatewayType string) error {
	// 这里应该记录到流程执行历史表
	// 暂时使用日志记录
	executionLog := map[string]interface{}{
		"process_instance_id": req.ProcessInstanceID,
		"gateway_id":          req.GatewayID,
		"gateway_type":        gatewayType,
		"next_activities":     nextActivities,
		"variables":           req.Variables,
		"executed_at":         time.Now(),
		"tenant_id":           req.TenantID,
	}

	// 序列化日志
	logBytes, err := json.Marshal(executionLog)
	if err != nil {
		return fmt.Errorf("序列化执行日志失败: %w", err)
	}

	// TODO: 保存到流程执行历史表
	_ = logBytes

	return nil
}

// GetGatewayExecutionHistory 获取网关执行历史
func (e *GatewayEngine) GetGatewayExecutionHistory(ctx context.Context, processInstanceID string, tenantID int) ([]map[string]interface{}, error) {
	// 这里应该从流程执行历史表查询
	// 暂时返回空结果
	return []map[string]interface{}{}, nil
}

// ValidateGatewayConfiguration 验证网关配置
func (e *GatewayEngine) ValidateGatewayConfiguration(ctx context.Context, gatewayConfig map[string]interface{}) error {
	// 验证网关类型
	gatewayType, ok := gatewayConfig["type"].(string)
	if !ok {
		return fmt.Errorf("网关类型缺失")
	}

	switch GatewayType(gatewayType) {
	case GatewayTypeExclusive, GatewayTypeParallel, GatewayTypeInclusive:
		// 类型有效
	default:
		return fmt.Errorf("不支持的网关类型: %s", gatewayType)
	}

	// 验证输出流
	outputs, ok := gatewayConfig["outputs"].([]string)
	if !ok || len(outputs) == 0 {
		return fmt.Errorf("网关必须包含至少一个输出流")
	}

	// 验证条件表达式（对于排他和包容网关）
	if gatewayType == string(GatewayTypeExclusive) || gatewayType == string(GatewayTypeInclusive) {
		conditions, ok := gatewayConfig["conditions"].([]string)
		if !ok || len(conditions) == 0 {
			return fmt.Errorf("排他网关和包容网关必须包含条件表达式")
		}

		// 检查条件和输出流的数量是否匹配
		if len(conditions) != len(outputs) {
			return fmt.Errorf("条件表达式数量与输出流数量不匹配")
		}
	}

	return nil
}
