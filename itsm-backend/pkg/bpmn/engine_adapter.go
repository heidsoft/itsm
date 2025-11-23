package bpmn

import (
	"fmt"

	"github.com/nitram509/lib-bpmn-engine/pkg/bpmn_engine"
)

// EngineAdapter 封装 lib-bpmn-engine
type EngineAdapter struct {
	engine bpmn_engine.BpmnEngineState
}

// NewEngineAdapter 创建新的引擎适配器
func NewEngineAdapter() *EngineAdapter {
	engine := bpmn_engine.New()
	return &EngineAdapter{
		engine: engine,
	}
}

// LoadDefinition 加载 BPMN XML 定义
func (e *EngineAdapter) LoadDefinition(xmlData []byte) (*bpmn_engine.ProcessInfo, error) {
	processInfo, err := e.engine.LoadFromBytes(xmlData)
	if err != nil {
		return nil, fmt.Errorf("解析 BPMN XML 失败: %w", err)
	}
	return processInfo, nil
}

// CreateInstance 创建并启动流程实例
// 返回实例ID和可能的错误
func (e *EngineAdapter) CreateInstance(processDefinitionKey int64, variables map[string]interface{}) (int64, error) {
	instance, err := e.engine.CreateAndRunInstance(processDefinitionKey, variables)
	if err != nil {
		return 0, fmt.Errorf("启动流程实例失败: %w", err)
	}
	return instance.GetInstanceKey(), nil
}

// ExportState 导出引擎状态（用于持久化）
func (e *EngineAdapter) ExportState() []byte {
	return e.engine.Export()
}

// RestoreState 恢复引擎状态
func (e *EngineAdapter) RestoreState(state []byte) error {
	if len(state) == 0 {
		return nil
	}
	return e.engine.Load(state)
}

// RegisterTaskHandler 注册任务处理器（用于User Task回调）
// 实际场景中，User Task通常是等待状态，但我们可以在这里注册Handler来捕获进入Task的事件
func (e *EngineAdapter) RegisterTaskHandler(taskID string, handler func(job bpmn_engine.ActivatedJob)) {
	e.engine.NewTaskHandler().Id(taskID).Handler(handler)
}

// FindProcessInfoByKey 根据Key查找ProcessInfo
func (e *EngineAdapter) FindProcessInfoByKey(key string) *bpmn_engine.ProcessInfo {
	for _, p := range e.engine.GetProcessCache() {
		if p.BpmnProcessId == key {
			return &p
		}
	}
	return nil
}
