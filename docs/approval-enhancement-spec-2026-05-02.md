# ITSM 审批功能细化开发文档

## 一、背景

当前审批功能较为基础，无法满足企业级复杂审批场景需求。以下是对现有审批功能的分析与改进建议。

## 二、现状分析

### 2.1 已实现功能

| 功能 | 状态 | 说明 |
|------|------|------|
| 多级审批链 | ✅ | 支持多层级审批节点 |
| 基本操作 | ✅ | approve/reject/delegate |
| 超时处理 | ⚠️ | 仅支持超时通知 |
| 优先级匹配 | ✅ | 基于工单优先级匹配审批流 |
| 条件审批 | ⚠️ | 仅支持简单字段比较 |

### 2.2 当前审批模式

```go
// 当前支持的审批模式
ApprovalMode: "any"  // 任意一人审批即可 (或签)
ApprovalMode: "all"  // 所有人必须审批 (会签)
```

### 2.3 当前审批人指定方式

```go
AssigneeType: "role"  // 根据角色查找用户
AssigneeType: "user"  // 直接指定用户ID
```

## 三、待增强功能

### 3.1 条件审批细化

**现状问题：**

- 仅支持 `field + operator + value` 的简单比较
- 不支持复合条件
- 不支持日期范围、正则表达式

**改进方案：**

```go
// 增强后的条件配置
type ApprovalConditionConfig struct {
    Field          string      // 字段名
    Operator       string      // 操作符: eq, neq, gt, lt, contains, regex, in, between
    Value          interface{} // 比较值
    LogicalOperator string    // 逻辑操作符: "AND", "OR" (用于复合条件)
    GroupID        int        // 条件组ID，用于分组
}

// 支持的场景示例
Conditions: [
    {Field: "priority", Operator: "in", Value: ["high", "critical"], LogicalOperator: "OR"},
    {Field: "amount", Operator: "gt", Value: 10000, GroupID: 1},
    {Field: "department", Operator: "eq", Value: "finance", GroupID: 1, LogicalOperator: "AND"}
]
// 解析为: (priority IN ["high","critical"]) OR (amount > 10000 AND department = "finance")
```

### 3.2 审批人指定增强

**现状问题：**

- 仅支持角色和用户ID两种方式
- 无法实现动态指派

**改进方案：**

| 新类型 | 说明 | 示例 |
|--------|------|------|
| `department_head` | 部门负责人 | 查找工单创建者所在部门负责人 |
| `direct_manager` | 直接主管 | 查找用户在组织架构中的上级 |
| `previous_approver` | 上级审批人 | 沿用上一级审批人 |
| `field_value` | 字段值 | 从工单指定字段获取审批人ID |
| `formulator` | 工单发起人 | 工单创建者自己审批 |
| `group` | 用户组 | 指定用户组中的任意一人 |

```go
// 动态审批人解析
type DynamicApproverConfig struct {
    Type  string  // approver_type
    Value string  // 根据Type不同含义不同
    Fallback string  // 备用方案，当找不到时的处理
}

// 扩展resolveApprover方法
switch assigneeType {
case "department_head":
    // 查找工单创建者所在部门负责人
case "direct_manager":
    // 查找用户在组织架构中的直接上级
case "previous_approver":
    // 返回上一级审批人ID
case "field_value":
    // 从工单字段获取值作为审批人ID
case "formulator":
    // 返回工单创建者ID
case "group":
    // 从用户组中选择一个可用用户
}
```

### 3.3 审批模式增强

**现状问题：**

- 仅支持 `any` 和 `all`
- 无法满足 N/M 投票制等复杂场景

**改进方案：**

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| `any` | 任意一人审批即可 | 快速审批通道 |
| `all` | 所有人必须审批 | 高风险审批 |
| `quota` | N/M 投票制 | 委员会审批 |
| `sequential` | 顺序审批 | 逐级审批 |
| `parallel` | 并行审批 | 多部门会签 |

```go
type ApprovalNode struct {
    Level          int
    Name           string
    ApprovalMode   string  // any, all, quota, sequential, parallel
    MinimumApprovals *int  // quota模式下需要的最少同意票数
    MaximumRejections *int // quota模式下允许的最大反对票数
    TimeoutHours   *int
}
```

### 3.4 超时处理增强

**现状问题：**

- 仅记录超时状态，无自动处理

**改进方案：**

| 超时动作 | 说明 |
|----------|------|
| `notify` | 发送超时提醒 |
| `escalate` | 自动升级给指定人 |
| `auto_approve` | 满足条件时自动通过 |
| `auto_reject` | 满足条件时自动拒绝 |
| `suspend` | 暂停流程，等待确认 |

```go
type TimeoutConfig struct {
    TimeoutHours    int
    Actions         []string  // 超时后执行的动作列表
    EscalateTo      *int      // 升级目标用户ID
    EscalateLevel   *int      // 升级层级数
    Conditions      []ApprovalConditionConfig // 自动审批的条件
    ReminderHours   []int     // 提醒时间点，如 [24, 48] 表示24小时和48小时提醒
}

// 配置示例
{
    "timeout_hours": 48,
    "timeout_actions": ["escalate", "notify"],
    "escalate_to": 101,  // 升级给管理员
    "escalate_level": 1,
    "conditions": [
        {Field: "priority", Operator: "neq", Value: "critical"}
    ],
    "reminder_hours": [24, 48]
}
```

### 3.5 审批历史记录增强

**现状问题：**

- 仅记录审批结果，缺少详细操作日志

**改进方案：**

```go
// 审批操作日志表
type ApprovalActionLog struct {
    ID              int
    RecordID        int       // 关联审批记录ID
    Action          string    // approve, reject, delegate, escalate, remind, timeout
    OperatorID      int       // 操作人ID
    OperatorName    string
    OperatorIP      string    // 操作人IP地址
    Comment         string
    FromStatus      string    // 操作前状态
    ToStatus        string    // 操作后状态
    RelatedUsers    []int     // 涉及的其他用户（如委托目标）
    CreatedAt       time.Time
}

// SLA跟踪
type ApprovalSLARecord struct {
    ID                int
    RecordID          int
    SLAStartTime      time.Time
    SLAEndTime        time.Time
    DueTime           time.Time
    FirstReminderTime time.Time
    ReminderCount     int
    Escalated         bool
    EscalatedAt       *time.Time
    CompletedAt       *time.Time
    Status            string    // on_track, at_risk, breached
}
```

### 3.6 委托功能增强

**现状问题：**

- 简单委托，无规则和历史追踪

**改进方案：**

| 功能 | 说明 |
|------|------|
| 临时委托规则 | 设置周期性委托，如每月指定委托人 |
| 委托链追踪 | 记录完整委托路径 |
| 委托时效 | 委托可设置有效期 |
| 委托审批 | 敏感操作需原审批人确认 |

```go
// 委托规则
type DelegationRule struct {
    ID            int
    UserID        int        // 委托人
    DelegateTo    int        // 被委托人
    TicketTypes   []string   // 允许委托的工单类型
    PriorityRange []string   // 允许委托的优先级范围
    ValidFrom     time.Time
    ValidTo       *time.Time
    Active        bool
}

// 委托记录
type DelegationLog struct {
    ID              int
    OriginalApprover int
    OriginalRecord  int
    DelegateTo      int
    DelegatorReason string
    DelegatedAt     time.Time
    OriginalApproverConfirmed bool  // 原审批人是否确认
    ConfirmedAt     *time.Time
}
```

## 四、数据库变更

### 4.1 新增表

```sql
-- 审批操作日志表
CREATE TABLE approval_action_logs (
    id SERIAL PRIMARY KEY,
    record_id INT NOT NULL REFERENCES approval_records(id),
    action VARCHAR(50) NOT NULL,
    operator_id INT NOT NULL,
    operator_name VARCHAR(100),
    operator_ip VARCHAR(50),
    comment TEXT,
    from_status VARCHAR(20),
    to_status VARCHAR(20),
    related_users INT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 委托规则表
CREATE TABLE delegation_rules (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    delegate_to INT NOT NULL,
    ticket_types VARCHAR(50)[],
    priority_range VARCHAR(20)[],
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- SLA记录表
CREATE TABLE approval_sla_records (
    id SERIAL PRIMARY KEY,
    record_id INT UNIQUE NOT NULL REFERENCES approval_records(id),
    sla_start_time TIMESTAMP NOT NULL,
    due_time TIMESTAMP NOT NULL,
    first_reminder_time TIMESTAMP,
    reminder_count INT DEFAULT 0,
    escalated BOOLEAN DEFAULT FALSE,
    escalated_at TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'on_track'
);

-- 复合条件组表
CREATE TABLE approval_condition_groups (
    id SERIAL PRIMARY KEY,
    workflow_id INT NOT NULL REFERENCES approval_workflows(id),
    node_level INT,
    group_id INT NOT NULL,
    logical_operator VARCHAR(10) DEFAULT 'AND',
    display_order INT DEFAULT 0
);
```

### 4.2 表字段变更

```sql
-- approval_workflows.nodes 增加新字段
ALTER TABLE approval_workflows ADD COLUMN IF NOT EXISTS timeout_config JSONB;
ALTER TABLE approval_workflows ADD COLUMN IF NOT EXISTS sla_config JSONB;

-- approval_records 增加字段
ALTER TABLE approval_records ADD COLUMN IF NOT EXISTS sla_record_id INT REFERENCES approval_sla_records(id);
```

## 五、API 扩展

### 5.1 新增端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/approval-logs` | 获取审批操作日志 |
| GET | `/api/v1/delegation-rules` | 获取委托规则列表 |
| POST | `/api/v1/delegation-rules` | 创建委托规则 |
| PUT | `/api/v1/delegation-rules/:id` | 更新委托规则 |
| DELETE | `/api/v1/delegation-rules/:id` | 删除委托规则 |
| GET | `/api/v1/approval-sla/:record_id` | 获取审批SLA状态 |

### 5.2 DTO 变更

```go
// 增强的审批节点DTO
type ApprovalNodeRequest struct {
    Level          int                      `json:"level"`
    Name           string                   `json:"name"`
    ApproverType   string                   `json:"approver_type"`
    ApproverIDs    []int                    `json:"approver_ids"`
    ApprovalMode   string                   `json:"approval_mode"`
    AllowReject    bool                     `json:"allow_reject"`
    AllowDelegate  bool                     `json:"allow_delegate"`
    RejectAction   string                   `json:"reject_action"`
    MinimumApprovals *int                   `json:"minimum_approvals,omitempty"`
    TimeoutHours   *int                     `json:"timeout_hours,omitempty"`
    TimeoutConfig  *TimeoutConfig           `json:"timeout_config,omitempty"`
    Conditions     []ApprovalConditionConfig `json:"conditions,omitempty"`
    ConditionGroups []ConditionGroupConfig   `json:"condition_groups,omitempty"`
}
```

## 六、实施计划

### 阶段一：基础增强（优先级：高）

- [ ] 扩展条件审批，支持复合条件 AND/OR
- [ ] 新增审批人动态解析（department_head, direct_manager, field_value）
- [ ] 增强审批模式（quota 模式）

### 阶段二：超时处理（优先级：中）

- [ ] 实现超时自动升级
- [ ] 添加超时提醒机制
- [ ] SLA 状态跟踪

### 阶段三：委托管理（优先级：中）

- [ ] 临时委托规则
- [ ] 委托历史追踪
- [ ] 敏感操作二次确认

### 阶段四：审计日志（优先级：低）

- [ ] 完整操作日志记录
- [ ] 操作追溯查询
- [ ] 合规报表生成

## 七、风险与注意事项

1. **向后兼容**：新增字段和功能需保证与现有数据兼容
2. **性能考虑**：复合条件解析可能影响审批触发性能，需优化
3. **安全审计**：委托和升级操作需严格权限控制
4. **用户体验**：前端需同步更新以支持新功能

## 八、验收标准

- [ ] 复合条件可以正确解析和执行
- [ ] 动态审批人可以正确解析
- [ ] quota 模式按预期工作
- [ ] 超时升级可以正确触发
- [ ] 委托规则可以按时间生效
- [ ] 操作日志完整记录所有审批动作
- [ ] 与现有审批流程向后兼容
