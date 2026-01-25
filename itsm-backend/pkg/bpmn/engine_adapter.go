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
	engine := bpmn_engine.New("itsm-engine")
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
// 注意：lib-bpmn-engine v0.2.4 不提供状态导出功能
// 如果需要持久化，建议使用数据库存储流程实例状态
func (e *EngineAdapter) ExportState() []byte {
	// 状态导出需要依赖数据库中的流程实例表
	// 当前通过 ProcessInstance 表管理状态
	return nil
}

// RestoreState 恢复引擎状态
// 注意：lib-bpmn-engine v0.2.4 不提供状态恢复功能
// 建议从数据库重新加载流程定义并创建新实例
func (e *EngineAdapter) RestoreState(state []byte) error {
	if len(state) == 0 {
		return nil
	}
	// 状态恢复通过重新创建流程实例实现
	return nil
}

// RegisterTaskHandler 注册任务处理器（用于User Task回调）
func (e *EngineAdapter) RegisterTaskHandler(taskID string, handler func(job bpmn_engine.ActivatedJob)) {
	e.engine.AddTaskHandler(taskID, handler)
}

// FindProcessInfoByKey 根据Key查找ProcessInfo
// 注意：需要通过已加载的流程定义来查找
func (e *EngineAdapter) FindProcessInfoByKey(key string) *bpmn_engine.ProcessInfo {
	// lib-bpmn-engine 不提供 GetProcessCache 方法
	// 需要通过其他方式查找，例如从数据库查询已加载的流程定义
	// 当前实现：流程定义从 ProcessDefinition 表加载
	return nil
}
