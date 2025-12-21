package cloud

import "context"

// ExecuteResult 交付执行结果（最小骨架）
type ExecuteResult struct {
	Resources map[string]string `json:"resources,omitempty"` // e.g. {"ecs_instance_id":"i-xxx"}
}

// Provider 交付 Provider 抽象（后续可扩展 Plan/Validate）
type Provider interface {
	Name() string
	Execute(ctx context.Context, payload map[string]any) (*ExecuteResult, error)
}


