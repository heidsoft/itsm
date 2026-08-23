package bpmn

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// bpmnTenantIDKey is the unexported context key used to store the BPMN tenant ID.
type bpmnTenantIDKey struct{}

// BPMNTenantIDContextKey is the exported context key for the BPMN tenant ID.
// Using a typed key (instead of a plain string) prevents collisions with other
// packages that store values under the same context.
var BPMNTenantIDContextKey = bpmnTenantIDKey{}

type bpmnUserIDKey struct{}

// BPMNUserIDContextKey carries the authenticated actor into workflow services.
// It must only be populated from trusted authentication middleware.
var BPMNUserIDContextKey = bpmnUserIDKey{}

// ServiceTaskHandlerInterface 服务任务处理器接口
// 定义所有服务任务处理器需要实现的方法
type ServiceTaskHandlerInterface interface {
	// GetTaskType 返回处理器支持的任务类型
	GetTaskType() string

	// Execute 执行服务任务
	Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error)

	// Validate 验证参数
	Validate(ctx context.Context, config map[string]interface{}) error

	// GetHandlerID 返回处理器标识
	GetHandlerID() string
}

// HandlerBase 处理器基类
// 提供通用的辅助方法
type HandlerBase struct {
	// 可以在这里添加通用的字段
}

// GetIntFromVars 从变量中提取整数
func GetIntFromVars(variables map[string]interface{}, key string) int {
	if v, ok := variables[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return 0
}

// GetIntSliceFromVars 从变量中提取整数切片
func GetIntSliceFromVars(variables map[string]interface{}, key string) []int {
	if v, ok := variables[key]; ok {
		if val, ok := v.([]interface{}); ok {
			res := make([]int, 0, len(val))
			for _, item := range val {
				switch i := item.(type) {
				case float64:
					res = append(res, int(i))
				case int:
					res = append(res, i)
				case int64:
					res = append(res, int(i))
				}
			}
			return res
		}
	}
	return []int{}
}

// GetBoolFromVars 从变量中提取布尔值
func GetBoolFromVars(variables map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := variables[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case int:
			return val != 0
		case float64:
			return val != 0
		case string:
			return val == "true" || val == "1" || val == "yes"
		}
	}
	return defaultValue
}

// GetStringFromVars 从变量中提取字符串
func GetStringFromVars(variables map[string]interface{}, key string) string {
	if v, ok := variables[key]; ok {
		if val, ok := v.(string); ok {
			return val
		}
	}
	return ""
}

// GetTenantIDFromVars 从流程变量中提取租户ID。
// 缺失或非法时返回错误，禁止回退到默认租户（tenant_id=1），
// 否则会把无租户上下文的流程数据写入 admin 租户，造成越权。
func GetTenantIDFromVars(variables map[string]interface{}) (int, error) {
	if id := GetIntFromVars(variables, "tenant_id"); id > 0 {
		return id, nil
	}
	return 0, fmt.Errorf("流程变量缺少有效的 tenant_id，拒绝执行以避免跨租户写入")
}

// GetTenantIDFromContext 从 context 中提取 BPMN 租户ID，作为流程变量的补充来源。
func GetTenantIDFromContext(ctx context.Context) (int, bool) {
	if v := ctx.Value(BPMNTenantIDContextKey); v != nil {
		switch id := v.(type) {
		case int:
			if id > 0 {
				return id, true
			}
		case int64:
			if id > 0 {
				return int(id), true
			}
		case float64:
			if id > 0 {
				return int(id), true
			}
		}
	}
	return 0, false
}

// ResolveTenantID 优先使用流程变量中的租户ID，其次回退到 context 中经认证注入的租户ID。
// 两者都不可用时返回错误，绝不静默落到租户 1。
func ResolveTenantID(ctx context.Context, variables map[string]interface{}) (int, error) {
	if id := GetIntFromVars(variables, "tenant_id"); id > 0 {
		return id, nil
	}
	if id, ok := GetTenantIDFromContext(ctx); ok {
		return id, nil
	}
	return 0, fmt.Errorf("无法确定租户ID：流程变量与上下文均缺少 tenant_id")
}

// HandlerRegistry 处理器注册中心
// 负责管理所有服务任务处理器的注册和获取
type HandlerRegistry struct {
	handlers map[string]ServiceTaskHandlerInterface
}

// NewHandlerRegistry 创建新的处理器注册中心
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]ServiceTaskHandlerInterface),
	}
}

// Register 注册处理器
func (r *HandlerRegistry) Register(handler ServiceTaskHandlerInterface) {
	r.handlers[handler.GetHandlerID()] = handler
}

// Unregister 注销处理器
func (r *HandlerRegistry) Unregister(handlerID string) {
	delete(r.handlers, handlerID)
}

// GetHandler 获取处理器
func (r *HandlerRegistry) GetHandler(handlerID string) ServiceTaskHandlerInterface {
	return r.handlers[handlerID]
}

// GetHandlerByTaskType 根据任务类型获取处理器
func (r *HandlerRegistry) GetHandlerByTaskType(taskType string) ServiceTaskHandlerInterface {
	// 精确匹配
	if handler, ok := r.handlers[taskType]; ok {
		return handler
	}

	// 通配匹配
	for _, handler := range r.handlers {
		if handler.GetTaskType() == taskType {
			return handler
		}
	}

	return nil
}

// ListHandlers 列出所有已注册的处理器
func (r *HandlerRegistry) ListHandlers() []ServiceTaskHandlerInterface {
	handlers := make([]ServiceTaskHandlerInterface, 0, len(r.handlers))
	for _, h := range r.handlers {
		handlers = append(handlers, h)
	}
	return handlers
}
