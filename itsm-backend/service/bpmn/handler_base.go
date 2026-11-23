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

type bpmnSystemAdvanceKey struct{}

// WithBPMNSystemAdvance marks ctx as a trusted system transition: the engine
// skips per-actor task ownership checks while still keeping the tenant scope
// intact for RLS. Do not use tenantctx system bypass for this — that would
// route DB access through a BYPASSRLS role.
func WithBPMNSystemAdvance(ctx context.Context) context.Context {
	return context.WithValue(ctx, bpmnSystemAdvanceKey{}, true)
}

// IsBPMNSystemAdvance reports whether ctx is a trusted BPMN system transition.
func IsBPMNSystemAdvance(ctx context.Context) bool {
	v, _ := ctx.Value(bpmnSystemAdvanceKey{}).(bool)
	return v
}

// ServiceTaskHandlerInterface 服务任务处理器接口
// 定义所有服务任务处理器需要实现的方法
type ServiceTaskHandlerInterface interface {
	// GetTaskType 返回处理器支持的任务类型
	GetTaskType() string

	// Execute 执行服务任务
	Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error)

	// Validate 验证参数
	Validate(ctx context.Context, config map[string]interface{}) error

	GetHandlerID() string
}

// CompensationCapableHandler 是「可被补偿」的显式能力声明（codex 审计 P1-09）。
//
// 背景：引擎原先对所有 ServiceTask 无条件登记补偿范围，补偿时取原 serviceRef、
// 加 `_compensate=true` 后调用同一个 handler。而绝大多数 handler 的 Execute 只按
// action 执行**正向**动作（webhook 重发、通知重发、工单状态重写），于是「补偿」
// 变成把刚做过的副作用再做一遍——远端操作既不会被撤销，事务回滚也管不到已发出的
// 调用。注释里「整单回滚，不留下半补偿状态」只描述数据库簿记，不能证明业务可回滚。
//
// 因此补偿能力改为 **opt-in**：只有显式实现本接口并返回 true 的 handler 才会被
// 登记为可补偿、才会在补偿阶段被允许执行。未实现本接口（或返回 false）的 handler
// 一律视为**不支持补偿**——不登记 = 不承诺可回滚，而不是先承诺再把正向动作重做一遍。
//
// 真正实现补偿语义（独立 undo handler / 幂等 undo 输入 / 外部回执）后才应返回 true。
type CompensationCapableHandler interface {
	ServiceTaskHandlerInterface

	// SupportsCompensation 报告该 handler 是否实现了真正的撤销语义。
	SupportsCompensation() bool
}

// HandlerSupportsCompensation 判断 handler 是否声明了可补偿能力。
// fail-closed：未实现 CompensationCapableHandler 的 handler 一律返回 false，
// 不因为「实现了基础接口」就被当成可回滚。
func HandlerSupportsCompensation(handler ServiceTaskHandlerInterface) bool {
	capable, ok := handler.(CompensationCapableHandler)
	return ok && capable.SupportsCompensation()
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

// GetTenantIDFromContext 从 context 中提取 BPMN 租户ID（命令认证注入的唯一权威来源）。
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

// ResolveTenantID 返回命令认证 context 注入的租户ID——这是 ServiceTask handler
// 唯一权威的租户来源（codex 审计 P1-01）。
//
// 流程变量中的 tenant_id 只允许与 context 一致（后端 TriggerProcess 启动时写入
// 的同值副本）；不一致视为跨租户注入，直接拒绝。context 缺失同样拒绝（fail
// closed），绝不回退到变量或默认租户——旧实现「变量优先」曾允许租户 A 的流程
// 携带 B 的 tenant_id 把 handler 的查询/更新指向 B。
func ResolveTenantID(ctx context.Context, variables map[string]interface{}) (int, error) {
	ctxID, ok := GetTenantIDFromContext(ctx)
	if !ok {
		return 0, fmt.Errorf("缺少认证租户上下文，拒绝执行以避免跨租户写入")
	}
	if id := GetIntFromVars(variables, "tenant_id"); id > 0 && id != ctxID {
		return 0, fmt.Errorf("流程变量 tenant_id=%d 与认证上下文租户 %d 冲突，拒绝执行以避免跨租户写入", id, ctxID)
	}
	return ctxID, nil
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
