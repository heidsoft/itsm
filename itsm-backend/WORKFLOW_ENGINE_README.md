# 工作流引擎实现说明

## 概述

本文档描述了ITSM系统中基础工作流引擎的实现，包括后端核心功能、API接口和前端组件。

## 架构设计

### 核心组件

1. **WorkflowEngine** - 工作流执行引擎
2. **WorkflowApprovalService** - 审批流程管理服务
3. **WorkflowTaskService** - 任务管理服务
4. **WorkflowMonitorService** - 监控和告警服务

### 数据模型

#### WorkflowDefinition (工作流定义)

```go
type WorkflowDefinition struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Version     string                 `json:"version"`
    TenantID    int                    `json:"tenant_id"`
    Steps       []WorkflowStep         `json:"steps"`
    Variables   map[string]interface{} `json:"variables"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

#### WorkflowStep (工作流步骤)

```go
type WorkflowStep struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Type        string   `json:"type"` // start, end, task, approval, condition, action
    Description string   `json:"description"`
    Assignee    string   `json:"assignee"`
    Timeout     int      `json:"timeout"`
    Conditions  []string `json:"conditions"`
    Actions     []string `json:"actions"`
    NextSteps   []string `json:"next_steps"`
}
```

#### WorkflowInstance (工作流实例)

```go
type WorkflowInstance struct {
    ID           string                 `json:"id"`
    DefinitionID string                 `json:"definition_id"`
    Status       string                 `json:"status"`
    CurrentStep  string                 `json:"current_step"`
    Variables    map[string]interface{} `json:"variables"`
    History      []WorkflowHistory      `json:"history"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

## 功能特性

### 1. 工作流执行引擎

- **工作流启动**: 根据定义创建工作流实例
- **步骤执行**: 按顺序执行工作流步骤
- **状态管理**: 跟踪工作流实例的执行状态
- **条件分支**: 支持基于条件的流程分支
- **变量管理**: 支持工作流执行过程中的变量传递

### 2. 审批流程管理

- **审批任务创建**: 自动创建审批任务
- **审批操作**: 支持批准、拒绝、取消操作
- **任务分配**: 支持任务重新分配
- **审批历史**: 记录完整的审批过程

### 3. 任务管理

- **任务创建**: 创建工作流任务
- **任务状态**: 支持开始、完成、失败、取消状态
- **任务分配**: 支持任务重新分配
- **超时处理**: 支持任务超时配置

### 4. 监控和告警

- **性能指标**: 工作流执行统计
- **步骤分析**: 步骤级别的性能分析
- **瓶颈识别**: 自动识别性能瓶颈
- **告警管理**: 基于规则的告警系统

## API接口

### 工作流管理

- `GET /api/v1/workflows` - 获取工作流列表
- `POST /api/v1/workflows` - 创建工作流
- `GET /api/v1/workflows/:id` - 获取工作流详情
- `PUT /api/v1/workflows/:id` - 更新工作流
- `DELETE /api/v1/workflows/:id` - 删除工作流

### 工作流执行

- `POST /api/v1/workflows/:id/start` - 启动工作流
- `POST /api/v1/workflows/:id/execute-step` - 执行工作流步骤
- `POST /api/v1/workflows/:id/complete-step` - 完成工作流步骤

### 审批管理

- `GET /api/v1/workflows/approval-tasks` - 获取审批任务列表
- `POST /api/v1/workflows/approval-tasks/:id/approve` - 批准任务
- `POST /api/v1/workflows/approval-tasks/:id/reject` - 拒绝任务

### 任务管理

- `GET /api/v1/workflows/tasks` - 获取任务列表
- `POST /api/v1/workflows/tasks/:id/start` - 开始任务
- `POST /api/v1/workflows/tasks/:id/complete` - 完成任务

### 监控接口

- `GET /api/v1/workflows/metrics` - 获取工作流指标
- `GET /api/v1/workflows/step-metrics` - 获取步骤指标
- `GET /api/v1/workflows/alerts` - 获取告警信息

## 前端组件

### 1. WorkflowDesigner (工作流设计器)

- **可视化设计**: 拖拽式工作流设计界面
- **步骤配置**: 支持各种类型步骤的详细配置
- **变量管理**: 工作流变量的定义和管理
- **版本控制**: 支持工作流版本管理

### 2. WorkflowMonitor (工作流监控)

- **实时监控**: 工作流实例的实时状态监控
- **性能分析**: 详细的性能指标和趋势分析
- **告警管理**: 告警信息的展示和处理
- **瓶颈识别**: 性能瓶颈的可视化展示

## 使用示例

### 创建工作流

```go
// 创建工作流定义
workflow := &WorkflowDefinition{
    Name:        "变更审批流程",
    Description: "IT变更的标准审批流程",
    Version:     "1.0.0",
    TenantID:    1,
    Steps: []WorkflowStep{
        {
            ID:          "step_1",
            Name:        "需求确认",
            Type:        "task",
            Description: "确认变更需求",
            Assignee:    "张三",
            Timeout:     24,
        },
        {
            ID:          "step_2",
            Name:        "技术评审",
            Type:        "approval",
            Description: "技术可行性评审",
            Assignee:    "李四",
            Timeout:     48,
        },
        {
            ID:          "step_3",
            Name:        "实施部署",
            Type:        "task",
            Description: "执行变更实施",
            Assignee:    "王五",
            Timeout:     72,
        },
    },
    Variables: map[string]interface{}{
        "change_type": "infrastructure",
        "priority":    "high",
    },
}
```

### 启动工作流

```go
// 启动工作流实例
instance, err := workflowEngine.StartWorkflow(ctx, workflow.ID, map[string]interface{}{
    "change_request_id": "CR-001",
    "requester":         "user123",
})
```

### 完成工作流步骤

```go
// 完成工作流步骤
err := workflowEngine.CompleteWorkflowStep(ctx, &CompleteWorkflowStepRequest{
    InstanceID: instance.ID,
    StepID:     "step_1",
    Action:     "complete",
    Variables: map[string]interface{}{
        "approval_result": "approved",
        "comments":        "需求确认完成",
    },
})
```

## 配置说明

### 环境变量

- `WORKFLOW_ENGINE_ENABLED`: 启用工作流引擎 (默认: true)
- `WORKFLOW_TIMEOUT_DEFAULT`: 默认超时时间(小时) (默认: 24)
- `WORKFLOW_MAX_RETRIES`: 最大重试次数 (默认: 3)

### 数据库配置

工作流引擎使用以下数据表：

- `workflows`: 工作流定义
- `workflow_instances`: 工作流实例
- `workflow_histories`: 执行历史
- `approval_tasks`: 审批任务
- `workflow_tasks`: 工作流任务

## 性能优化

### 1. 缓存策略

- 工作流定义缓存
- 实例状态缓存
- 审批任务缓存

### 2. 异步处理

- 工作流步骤异步执行
- 审批通知异步发送
- 监控数据异步收集

### 3. 批量操作

- 批量创建任务
- 批量更新状态
- 批量处理告警

## 安全考虑

### 1. 权限控制

- 基于角色的访问控制
- 工作流级别的权限管理
- 步骤级别的权限控制

### 2. 数据验证

- 输入参数验证
- 工作流定义验证
- 执行权限验证

### 3. 审计日志

- 操作日志记录
- 变更历史追踪
- 安全事件监控

## 扩展性

### 1. 插件系统

- 自定义步骤类型
- 自定义条件表达式
- 自定义动作处理器

### 2. 集成接口

- 外部系统集成
- 消息队列集成
- 监控系统集成

### 3. 多租户支持

- 租户隔离
- 资源配额管理
- 自定义配置

## 故障排除

### 常见问题

1. **工作流卡住**: 检查步骤配置和条件设置
2. **审批任务丢失**: 检查任务分配和通知配置
3. **性能下降**: 检查数据库索引和缓存配置
4. **告警不触发**: 检查告警规则和阈值设置

### 日志分析

- 工作流执行日志
- 错误和异常日志
- 性能监控日志
- 安全审计日志

## 未来规划

### 短期目标

- [ ] 完善条件表达式引擎
- [ ] 添加更多步骤类型
- [ ] 优化性能监控
- [ ] 增强告警功能

### 长期目标

- [ ] 支持复杂业务规则
- [ ] 集成机器学习优化
- [ ] 支持分布式部署
- [ ] 提供可视化流程分析

## 贡献指南

欢迎提交Issue和Pull Request来改进工作流引擎。请确保：

1. 遵循代码规范
2. 添加适当的测试
3. 更新相关文档
4. 通过代码审查

## 许可证

本项目采用MIT许可证，详见LICENSE文件。
