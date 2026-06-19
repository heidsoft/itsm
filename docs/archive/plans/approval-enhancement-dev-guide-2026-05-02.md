# ITSM 审批功能细化开发指南

## 一、文档说明

本文档为 IT 服务管理系统（ITSM）审批功能的细化开发指南，包含完整的实现方案、代码示例和验收标准。

---

## 二、需求概述

### 2.1 现状问题

当前审批功能仅支持基础能力，无法满足企业复杂审批场景：

| 问题 | 现状 | 影响 |
|------|------|------|
| 条件审批过简 | 仅支持 `field + operator + value` 单条件 | 无法处理复合业务规则 |
| 审批人固定 | 仅支持角色ID或用户ID | 无法动态指派审批人 |
| 审批模式单一 | 仅支持 `any` 或 `all` | 无法实现 N/M 投票制 |
| 超时处理缺失 | 仅记录超时，无自动处理 | 工单可能无限期等待 |
| 委托功能弱 | 仅支持一次性委托 | 无法设置周期性委托 |

### 2.2 目标

实现企业级审批能力：

- 支持复合条件判断（AND/OR 逻辑）
- 支持动态审批人解析
- 支持多种审批模式（投票制、顺序审批等）
- 支持超时自动升级和提醒
- 支持委托规则管理

---

## 三、数据库设计

### 3.1 新增表

```sql
-- 审批操作日志表（记录每次操作）
CREATE TABLE approval_action_logs (
    id SERIAL PRIMARY KEY,
    record_id INT NOT NULL REFERENCES approval_records(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,  -- approve, reject, delegate, escalate, remind, timeout
    operator_id INT NOT NULL,
    operator_name VARCHAR(100),
    operator_ip VARCHAR(50),
    comment TEXT,
    from_status VARCHAR(20),
    to_status VARCHAR(20),
    related_users INT[],  -- 委托场景记录被委托人
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_approval_logs_record ON approval_action_logs(record_id);
CREATE INDEX idx_approval_logs_operator ON approval_action_logs(operator_id);

-- 委托规则表（支持周期性委托）
CREATE TABLE delegation_rules (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,  -- 委托人
    delegate_to INT NOT NULL,  -- 被委托人
    ticket_types VARCHAR(50)[],  -- 允许委托的工单类型，空表示全部
    priority_range VARCHAR(20)[],  -- 允许委托的优先级
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP,  -- NULL表示永久有效
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_delegation_user ON delegation_rules(user_id);
CREATE INDEX idx_delegation_valid ON delegation_rules(valid_from, valid_to);

-- SLA记录表（跟踪审批时效）
CREATE TABLE approval_sla_records (
    id SERIAL PRIMARY KEY,
    record_id INT UNIQUE NOT NULL REFERENCES approval_records(id) ON DELETE CASCADE,
    sla_start_time TIMESTAMP NOT NULL,
    due_time TIMESTAMP NOT NULL,
    first_reminder_time TIMESTAMP,  -- 首次提醒时间
    reminder_count INT DEFAULT 0,
    escalated BOOLEAN DEFAULT FALSE,
    escalated_at TIMESTAMP,
    escalated_to INT,  -- 升级给谁
    completed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'on_track'  -- on_track, at_risk, breached
);
CREATE INDEX idx_sla_due ON approval_sla_records(due_time);
CREATE INDEX idx_sla_status ON approval_sla_records(status);
```

### 3.2 表字段变更

```sql
-- approval_workflows 表增加字段
ALTER TABLE approval_workflows ADD COLUMN IF NOT EXISTS sla_config JSONB;
-- sla_config 示例: {"first_reminder_hours": 24, "second_reminder_hours": 48}

-- approval_records 表增加字段
ALTER TABLE approval_records ADD COLUMN IF NOT EXISTS sla_record_id INT REFERENCES approval_sla_records(id);
```

---

## 四、核心数据结构

### 4.1 审批条件（复合条件支持）

```go
// 文件：itsm-backend/dto/approval.go

// 单个条件
type ApprovalCondition struct {
    Field    string      `json:"field"`    // 字段名，如 "priority", "amount"
    Operator string      `json:"operator"` // 操作符：eq, neq, gt, gte, lt, lte, contains, in, between, regex
    Value    interface{} `json:"value"`     // 比较值
}

// 条件组（用于复合条件）
type ApprovalConditionGroup struct {
    GroupID         int                   `json:"group_id"`         // 组编号
    Conditions      []ApprovalCondition   `json:"conditions"`       // 组内条件（默认AND）
    LogicalOperator string                `json:"logical_operator"` // 组间操作符：AND, OR
}

// 审批节点完整配置
type ApprovalNodeConfig struct {
    Level            int                      `json:"level"`
    Name             string                   `json:"name"`
    ApproverType     string                   `json:"approver_type"`     // role, user, department_head, direct_manager, field_value, formulator, group
    ApproverIDs      []int                   `json:"approver_ids"`
    ApprovalMode     string                   `json:"approval_mode"`    // any, all, quota
    MinimumApprovals *int                    `json:"minimum_approvals"` // quota模式需要票数
    AllowReject      bool                     `json:"allow_reject"`
    AllowDelegate    bool                     `json:"allow_delegate"`
    RejectAction     string                   `json:"reject_action"`     // terminate, return_to_submitter
    TimeoutHours     *int                     `json:"timeout_hours"`
    TimeoutConfig    *TimeoutConfig            `json:"timeout_config"`
    Conditions       []ApprovalCondition      `json:"conditions"`        // 简单条件（向后兼容）
    ConditionGroups  []ApprovalConditionGroup  `json:"condition_groups"`  // 复合条件
}

// 超时配置
type TimeoutConfig struct {
    Actions        []string `json:"actions"`         // notify, escalate, auto_approve, auto_reject
    EscalateTo      *int     `json:"escalate_to"`      // 升级目标用户ID
    EscalateLevel   *int     `json:"escalate_level"`   // 升级层级数
    ReminderHours   []int    `json:"reminder_hours"`   // 提醒时间点：[24, 48]
    AutoApproveConditions []ApprovalCondition `json:"auto_approve_conditions"` // 自动审批条件
}
```

### 4.2 委托规则

```go
// 文件：itsm-backend/ent/delegationrule.go（新建）

type DelegationRule struct {
    ID            int
    UserID        int       // 委托人
    DelegateTo    int       // 被委托人
    TicketTypes   []string  // 允许的工单类型
    PriorityRange []string  // 允许的优先级
    ValidFrom     time.Time
    ValidTo       *time.Time
    Active        bool
}
```

### 4.3 审批日志

```go
// 文件：itsm-backend/ent/approvalactionlog.go（新建）

type ApprovalActionLog struct {
    ID           int
    RecordID     int       // 关联审批记录ID
    Action       string    // approve, reject, delegate, escalate, remind, timeout
    OperatorID   int
    OperatorName string
    OperatorIP   string
    Comment      string
    FromStatus   string
    ToStatus     string
    RelatedUsers []int
    CreatedAt    time.Time
}
```

---

## 五、功能实现

### 5.1 复合条件解析

#### 5.1.1 新增文件

```go
// 文件：itsm-backend/service/approval_condition.go

package service

import (
    "fmt"
    "regexp"
    "strings"
    "time"
    
    "itsm-backend/ent"
)

// ApprovalConditionEvaluator 审批条件评估器
type ApprovalConditionEvaluator struct {
    ticket *ent.Ticket
}

// NewApprovalConditionEvaluator 创建评估器
func NewApprovalConditionEvaluator(ticket *ent.Ticket) *ApprovalConditionEvaluator {
    return &ApprovalConditionEvaluator{ticket: ticket}
}

// EvaluateSimpleConditions 评估简单条件（向后兼容）
func (e *ApprovalConditionEvaluator) EvaluateSimpleConditions(conditions []map[string]interface{}) (bool, error) {
    if len(conditions) == 0 {
        return true, nil
    }
    
    for _, cond := range conditions {
        result, err := e.evaluateSingleCondition(cond)
        if err != nil {
            return false, err
        }
        if !result {
            return false, nil
        }
    }
    return true, nil
}

// EvaluateConditionGroups 评估复合条件组
func (e *ApprovalConditionEvaluator) EvaluateConditionGroups(groups []map[string]interface{}) (bool, error) {
    if len(groups) == 0 {
        return true, nil
    }
    
    // 按group_id分组
    groupMap := make(map[int][]map[string]interface{})
    for _, cond := range groups {
        groupID := 1
        if gid, ok := cond["group_id"].(float64); ok {
            groupID = int(gid)
        } else if gid, ok := cond["group_id"].(int); ok {
            groupID = gid
        }
        groupMap[groupID] = append(groupMap[groupID], cond)
    }
    
    // 组内AND，组间默认AND（可通过logical_operator调整）
    for groupID, conditions := range groupMap {
        groupResult := true
        for _, cond := range conditions {
            result, err := e.evaluateSingleCondition(cond)
            if err != nil {
                return false, err
            }
            if !result {
                groupResult = false
                break
            }
        }
        if !groupResult {
            return false, nil
        }
        
        // 检查是否有组间OR逻辑
        if groupID > 1 {
            // 实现组间OR：只要有一组满足即可
            return true, nil
        }
    }
    
    return true, nil
}

// evaluateSingleCondition 评估单个条件
func (e *ApprovalConditionEvaluator) evaluateSingleCondition(cond map[string]interface{}) (bool, error) {
    field := getStringField(cond, "field")
    operator := getStringField(cond, "operator")
    value := cond["value"]
    
    if field == "" || operator == "" {
        return true, nil // 跳过无效条件
    }
    
    fieldValue := e.getFieldValue(field)
    
    switch operator {
    case "eq":
        return equals(fieldValue, value), nil
    case "neq":
        return !equals(fieldValue, value), nil
    case "gt":
        return compare(fieldValue, value) > 0, nil
    case "gte":
        return compare(fieldValue, value) >= 0, nil
    case "lt":
        return compare(fieldValue, value) < 0, nil
    case "lte":
        return compare(fieldValue, value) <= 0, nil
    case "contains":
        return strings.Contains(toString(fieldValue), toString(value)), nil
    case "in":
        return containsIn(value, fieldValue), nil
    case "not_in":
        return !containsIn(value, fieldValue), nil
    case "between":
        return isBetween(fieldValue, value), nil
    case "regex":
        return matchRegex(toString(fieldValue), toString(value)), nil
    case "is_null":
        return fieldValue == nil || fieldValue == "", nil
    case "is_not_null":
        return fieldValue != nil && fieldValue != "", nil
    default:
        return false, fmt.Errorf("unknown operator: %s", operator)
    }
}

// getFieldValue 从工单获取字段值
func (e *ApprovalConditionEvaluator) getFieldValue(field string) interface{} {
    switch field {
    case "priority":
        return e.ticket.Priority
    case "status":
        return e.ticket.Status
    case "category":
        return e.ticket.Category
    case "requester_id":
        return e.ticket.RequesterID
    case "assigned_to":
        return e.ticket.AssignedTo
    case "title":
        return e.ticket.Title
    case "description":
        return e.ticket.Description
    default:
        // 从扩展字段获取
        if e.ticket.Extra != nil {
            if val, ok := e.ticket.Extra[field]; ok {
                return val
            }
        }
        return nil
    }
}

// 辅助函数
func getStringField(m map[string]interface{}, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

func toString(v interface{}) string {
    if v == nil {
        return ""
    }
    return fmt.Sprintf("%v", v)
}

func equals(a, b interface{}) bool {
    return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compare(a, b interface{}) int {
    aNum := toFloat64(a)
    bNum := toFloat64(b)
    if aNum > bNum {
        return 1
    } else if aNum < bNum {
        return -1
    }
    return 0
}

func toFloat64(v interface{}) float64 {
    switch val := v.(type) {
    case float64:
        return val
    case int:
        return float64(val)
    case int64:
        return float64(val)
    case string:
        var f float64
        fmt.Sscanf(val, "%f", &f)
        return f
    default:
        return 0
    }
}

func containsIn(value, item interface{}) bool {
    arr, ok := value.([]interface{})
    if !ok {
        arr2, ok := value.([]string)
        if ok {
            itemStr := toString(item)
            for _, v := range arr2 {
                if v == itemStr {
                    return true
                }
            }
        }
        return false
    }
    itemStr := toString(item)
    for _, v := range arr {
        if toString(v) == itemStr {
            return true
        }
    }
    return false
}

func isBetween(value, rangeValue interface{}) bool {
    arr, ok := rangeValue.([]interface{})
    if !ok || len(arr) < 2 {
        return false
    }
    v := toFloat64(value)
    min := toFloat64(arr[0])
    max := toFloat64(arr[1])
    return v >= min && v <= max
}

func matchRegex(text, pattern string) bool {
    matched, _ := regexp.MatchString(pattern, text)
    return matched
}
```

#### 5.1.2 单元测试

```go
// 文件：itsm-backend/service/approval_condition_test.go

package service

import (
    "testing"
)

func TestEvaluateSimpleConditions(t *testing.T) {
    ticket := &ent.Ticket{
        Priority: "high",
        Status:   "open",
    }
    evaluator := NewApprovalConditionEvaluator(ticket)
    
    tests := []struct {
        name       string
        conditions []map[string]interface{}
        expect     bool
    }{
        {
            name: "优先级等于high",
            conditions: []map[string]interface{}{
                {"field": "priority", "operator": "eq", "value": "high"},
            },
            expect: true,
        },
        {
            name: "优先级等于low（不满足）",
            conditions: []map[string]interface{}{
                {"field": "priority", "operator": "eq", "value": "low"},
            },
            expect: false,
        },
        {
            name: "优先级in列表",
            conditions: []map[string]interface{}{
                {"field": "priority", "operator": "in", "value": []interface{}{"high", "critical"}},
            },
            expect: true,
        },
        {
            name: "多条件AND",
            conditions: []map[string]interface{}{
                {"field": "priority", "operator": "eq", "value": "high"},
                {"field": "status", "operator": "eq", "value": "open"},
            },
            expect: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := evaluator.EvaluateSimpleConditions(tt.conditions)
            if err != nil {
                t.Errorf("unexpected error: %v", err)
                return
            }
            if result != tt.expect {
                t.Errorf("expected %v, got %v", tt.expect, result)
            }
        })
    }
}
```

---

### 5.2 动态审批人解析

#### 5.2.1 修改 resolveApprover 方法

```go
// 文件：itsm-backend/service/approval_service.go
// 修改现有 resolveApprover 方法，增加新类型支持

// resolveApprover 解析审批人
func (s *ApprovalService) resolveApprover(ctx context.Context, assigneeType, assigneeValue string, ticket *ent.Ticket, tenantID int) (int, string, error) {
    switch assigneeType {
    case "role":
        // 根据角色查找用户
        return s.resolveByRole(ctx, assigneeValue, tenantID)
        
    case "user":
        // 根据用户ID查找
        return s.resolveByUserID(ctx, assigneeValue, tenantID)
        
    case "department_head":
        // 查找工单创建者所在部门负责人
        return s.resolveByDepartmentHead(ctx, ticket, tenantID)
        
    case "direct_manager":
        // 查找用户在组织架构中的直接上级
        return s.resolveByDirectManager(ctx, ticket, tenantID)
        
    case "previous_approver":
        // 沿用上一级审批人
        return s.resolveByPreviousApprover(ctx, ticket)
        
    case "field_value":
        // 从工单指定字段获取审批人ID
        return s.resolveByFieldValue(ctx, assigneeValue, ticket, tenantID)
        
    case "formulator":
        // 工单创建者自己审批
        return ticket.RequesterID, "工单发起人", nil
        
    case "group":
        // 从用户组中选择一个可用用户
        return s.resolveByGroup(ctx, assigneeValue, tenantID)
        
    case "category_manager":
        // 根据工单类别查找类别经理
        return s.resolveByCategoryManager(ctx, ticket, tenantID)
        
    default:
        return 0, "", fmt.Errorf("不支持的审批人类型: %s", assigneeType)
    }
}

// resolveByRole 根据角色查找用户
func (s *ApprovalService) resolveByRole(ctx context.Context, role string, tenantID int) (int, string, error) {
    user, err := s.client.User.Query().
        Where(user.RoleEQ(user.Role(role)), user.TenantID(tenantID), user.Active(true)).
        First(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到具有角色 '%s' 的有效用户", role)
    }
    return user.ID, user.Name, nil
}

// resolveByUserID 根据用户ID查找
func (s *ApprovalService) resolveByUserID(ctx context.Context, userIDStr string, tenantID int) (int, string, error) {
    userID, err := strconv.Atoi(userIDStr)
    if err != nil {
        return 0, "", fmt.Errorf("无效的用户ID: %s", userIDStr)
    }
    user, err := s.client.User.Query().
        Where(user.ID(userID), user.TenantID(tenantID)).
        Only(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到用户ID: %d", userID)
    }
    return user.ID, user.Name, nil
}

// resolveByDepartmentHead 查找部门负责人
func (s *ApprovalService) resolveByDepartmentHead(ctx context.Context, ticket *ent.Ticket, tenantID int) (int, string, error) {
    // 获取工单创建者
    requester, err := s.client.User.Query().
        Where(user.ID(ticket.RequesterID), user.TenantID(tenantID)).
        Only(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到工单创建者")
    }
    
    // 查找该用户的部门负责人
    // 假设 DepartmentID 字段存在
    if requester.DepartmentID == nil || *requester.DepartmentID == 0 {
        return 0, "", fmt.Errorf("用户 %s 缺少部门信息", requester.Name)
    }
    
    manager, err := s.client.User.Query().
        Where(user.DepartmentID(*requester.DepartmentID), user.RoleEQ(user.Role("department_head")), user.TenantID(tenantID)).
        First(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到部门负责人")
    }
    return manager.ID, manager.Name, nil
}

// resolveByDirectManager 查找直接主管
func (s *ApprovalService) resolveByDirectManager(ctx context.Context, ticket *ent.Ticket, tenantID int) (int, string, error) {
    // 获取工单创建者
    requester, err := s.client.User.Query().
        Where(user.ID(ticket.RequesterID), user.TenantID(tenantID)).
        Only(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到工单创建者")
    }
    
    // 查找直接主管（ManagerID 字段）
    if requester.ManagerID == nil || *requester.ManagerID == 0 {
        return 0, "", fmt.Errorf("用户 %s 缺少主管信息", requester.Name)
    }
    
    manager, err := s.client.User.Query().
        Where(user.ID(*requester.ManagerID), user.TenantID(tenantID)).
        Only(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到直接主管")
    }
    return manager.ID, manager.Name, nil
}

// resolveByPreviousApprover 获取上一级审批人
func (s *ApprovalService) resolveByPreviousApprover(ctx context.Context, ticket *ent.Ticket) (int, string, error) {
    // 查询该工单最近一条已完成的审批记录
    record, err := s.client.ApprovalRecord.Query().
        Where(
            approvalrecord.TicketIDEQ(ticket.ID),
            approvalrecord.StatusEQ("approved"),
        ).
        Order(ent.Desc(approvalrecord.FieldStepOrder)).
        First(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到上一级审批人")
    }
    return record.ApproverID, record.ApproverName, nil
}

// resolveByFieldValue 从工单字段获取审批人
func (s *ApprovalService) resolveByFieldValue(ctx context.Context, fieldName, value string, ticket *ent.Ticket, tenantID int) (int, string, error) {
    var approverID int
    
    // 从工单扩展字段获取审批人ID
    if ticket.Extra != nil {
        if id, ok := ticket.Extra[fieldName].(float64); ok {
            approverID = int(id)
        } else if id, ok := ticket.Extra[fieldName].(int); ok {
            approverID = id
        } else if idStr, ok := ticket.Extra[fieldName].(string); ok {
            var err error
            approverID, err = strconv.Atoi(idStr)
            if err != nil {
                return 0, "", fmt.Errorf("字段 %s 值无效: %s", fieldName, idStr)
            }
        }
    }
    
    if approverID == 0 {
        return 0, "", fmt.Errorf("工单缺少字段 %s 或值无效", fieldName)
    }
    
    approver, err := s.client.User.Query().
        Where(user.ID(approverID), user.TenantID(tenantID)).
        Only(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到审批人: %d", approverID)
    }
    return approver.ID, approver.Name, nil
}

// resolveByGroup 从用户组选择
func (s *ApprovalService) resolveByGroup(ctx context.Context, groupIDStr, tenantID string) (int, string, error) {
    groupID, err := strconv.Atoi(groupIDStr)
    if err != nil {
        return 0, "", fmt.Errorf("无效的用户组ID: %s", groupIDStr)
    }
    
    // 查询用户组中的第一个可用用户
    user, err := s.client.User.Query().
        Where(user.TenantIDEQ(tenantID), user.Active(true)).
        First(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("用户组 %d 中没有可用用户", groupID)
    }
    return user.ID, user.Name, nil
}

// resolveByCategoryManager 查找类别经理
func (s *ApprovalService) resolveByCategoryManager(ctx context.Context, ticket *ent.Ticket, tenantID int) (int, string, error) {
    // 根据工单类别查找对应的经理
    // 需要从类别配置中获取经理信息
    category := ticket.Category
    if category == "" {
        return 0, "", fmt.Errorf("工单缺少类别信息")
    }
    
    // 查询类别经理（假设有 CategoryManager 关联）
    user, err := s.client.User.Query().
        Where(user.TenantID(tenantID), user.RoleEQ(user.Role("category_manager")), user.Active(true)).
        First(ctx)
    if err != nil {
        return 0, "", fmt.Errorf("未找到类别 %s 的经理", category)
    }
    return user.ID, user.Name, nil
}
```

#### 5.2.2 User 实体扩展

```go
// 文件：itsm-backend/ent/user.go
// 在 User 结构体中添加新字段

type User struct {
    config ent.Config
    // ... 现有字段 ...
    
    // 新增字段
    DepartmentID *int   `json:"department_id"`   // 部门ID
    ManagerID    *int   `json:"manager_id"`      // 直接主管ID
}
```

---

### 5.3 审批模式增强（quota 模式）

#### 5.3.1 修改审批记录计数逻辑

```go
// 文件：itsm-backend/service/approval_service.go
// 修改 handleApprovalApproved 方法

// handleApprovalApproved 处理审批通过
func (s *ApprovalService) handleApprovalApproved(ctx context.Context, record *ent.ApprovalRecord) error {
    // 获取工作流配置
    workflow, err := s.client.ApprovalWorkflow.Query().
        Where(approvalworkflow.IDEQ(record.WorkflowID)).
        Only(ctx)
    if err != nil {
        return fmt.Errorf("failed to get workflow: %w", err)
    }
    
    // 解析当前节点配置
    currentNode := s.getCurrentNodeConfig(workflow, record.CurrentLevel)
    approvalMode := "any"
    minimumApprovals := 1
    if currentNode != nil {
        approvalMode = getStringValue(currentNode["approval_mode"])
        if minApprovals, ok := currentNode["minimum_approvals"].(float64); ok {
            minimumApprovals = int(minApprovals)
        }
    }
    
    switch approvalMode {
    case "any":
        // any 模式：任意一人通过即完成该级
        return s.completeCurrentLevel(ctx, record)
        
    case "all":
        // all 模式：所有人都必须通过
        return s.checkAllApproved(ctx, record, workflow)
        
    case "quota":
        // quota 模式：N/M 投票制
        return s.checkQuotaApproved(ctx, record, workflow, minimumApprovals)
        
    default:
        return s.completeCurrentLevel(ctx, record)
    }
}

// checkAllApproved 检查是否所有人都已通过
func (s *ApprovalService) checkAllApproved(ctx context.Context, record *ent.ApprovalRecord, workflow *ent.ApprovalWorkflow) error {
    // 统计同级别所有审批人数量
    totalApprovers := len(s.getApproverIDsForLevel(ctx, workflow, record))
    
    // 统计已通过数量
    approvedCount, err := s.countApprovedAtLevel(ctx, record)
    if err != nil {
        return err
    }
    
    if approvedCount >= totalApprovers {
        return s.completeCurrentLevel(ctx, record)
    }
    return nil
}

// checkQuotaApproved 检查配额是否满足
func (s *ApprovalService) checkQuotaApproved(ctx context.Context, record *ent.ApprovalRecord, workflow *ent.ApprovalWorkflow, minimumApprovals int) error {
    approvedCount, err := s.countApprovedAtLevel(ctx, record)
    if err != nil {
        return err
    }
    
    if approvedCount >= minimumApprovals {
        // 配额满足，完成当前级别
        // 取消其他待审批的同级别记录
        err = s.cancelRemainingAtLevel(ctx, record)
        if err != nil {
            return err
        }
        return s.completeCurrentLevel(ctx, record)
    }
    return nil
}

// countApprovedAtLevel 统计同级别已通过数量
func (s *ApprovalService) countApprovedAtLevel(ctx context.Context, record *ent.ApprovalRecord) (int, error) {
    return s.client.ApprovalRecord.Query().
        Where(
            approvalrecord.WorkflowIDEQ(record.WorkflowID),
            approvalrecord.TicketIDEQ(record.TicketID),
            approvalrecord.CurrentLevel(record.CurrentLevel),
            approvalrecord.StatusEQ("approved"),
        ).
        Count(ctx)
}

// cancelRemainingAtLevel 取消同级别剩余待审批
func (s *ApprovalService) cancelRemainingAtLevel(ctx context.Context, record *ent.ApprovalRecord) error {
    _, err := s.client.ApprovalRecord.Update().
        Where(
            approvalrecord.WorkflowIDEQ(record.WorkflowID),
            approvalrecord.TicketIDEQ(record.TicketID),
            approvalrecord.CurrentLevel(record.CurrentLevel),
            approvalrecord.StatusEQ("pending"),
        ).
        SetStatus("cancelled").
        Save(ctx)
    return err
}

// getApproverIDsForLevel 获取指定级别的所有审批人ID
func (s *ApprovalService) getApproverIDsForLevel(ctx context.Context, workflow *ent.ApprovalWorkflow, record *ent.ApprovalRecord) []int {
    nodes := s.parseWorkflowNodes(workflow.Nodes)
    for _, node := range nodes {
        if node.Level == record.CurrentLevel {
            return node.ApproverIDs
        }
    }
    return nil
}

// completeCurrentLevel 完成当前级别，进入下一级
func (s *ApprovalService) completeCurrentLevel(ctx context.Context, record *ent.ApprovalRecord) error {
    // 检查是否还有下一级
    nextLevel := record.CurrentLevel + 1
    hasNext, err := s.hasNextLevel(ctx, record, nextLevel)
    if err != nil {
        return err
    }
    
    if !hasNext {
        // 没有下一级，标记工单为已审批
        _, err := s.client.Ticket.UpdateOneID(record.TicketID).
            SetStatus("approved").
            Save(ctx)
        return err
    }
    
    // 有下一级，激活下一级审批
    return s.activateNextLevel(ctx, record, nextLevel)
}

// hasNextLevel 检查是否有下一级
func (s *ApprovalService) hasNextLevel(ctx context.Context, record *ent.ApprovalRecord, nextLevel int) (bool, error) {
    count, err := s.client.ApprovalRecord.Query().
        Where(
            approvalrecord.WorkflowIDEQ(record.WorkflowID),
            approvalrecord.TicketIDEQ(record.TicketID),
            approvalrecord.CurrentLevel(nextLevel),
        ).
        Count(ctx)
    return count > 0, err
}

// activateNextLevel 激活下一级审批
func (s *ApprovalService) activateNextLevel(ctx context.Context, record *ent.ApprovalRecord, nextLevel int) error {
    // 更新下一级记录状态为 pending（如果有被取消的）
    _, err := s.client.ApprovalRecord.Update().
        Where(
            approvalrecord.WorkflowIDEQ(record.WorkflowID),
            approvalrecord.TicketIDEQ(record.TicketID),
            approvalrecord.CurrentLevel(nextLevel),
            approvalrecord.StatusEQ("pending"),
        ).
        SetStatus("pending").
        Save(ctx)
    return err
}
```

---

### 5.4 超时处理

#### 5.4.1 超时检查服务

```go
// 文件：itsm-backend/service/approval_timeout.go

package service

import (
    "context"
    "fmt"
    "time"
    
    "itsm-backend/ent"
    "itsm-backend/ent/approvalrecord"
)

type ApprovalTimeoutService struct {
    client *ent.Client
    logger interface{} // zap.SugaredLogger
}

func NewApprovalTimeoutService(client *ent.Client, logger interface{}) *ApprovalTimeoutService {
    return &ApprovalTimeoutService{
        client: client,
        logger: logger,
    }
}

// CheckTimeouts 检查超时并自动处理
func (s *ApprovalTimeoutService) CheckTimeouts(ctx context.Context) error {
    // 查找所有待审批且已超时的记录
    now := time.Now()
    records, err := s.client.ApprovalRecord.Query().
        Where(
            approvalrecord.StatusEQ("pending"),
            approvalrecord.DueDateLT(now),
        ).
        All(ctx)
    if err != nil {
        return fmt.Errorf("failed to query timeout records: %w", err)
    }
    
    for _, record := range records {
        if err := s.handleTimeout(ctx, record); err != nil {
            s.logger.Errorf("failed to handle timeout for record %d: %v", record.ID, err)
            continue
        }
    }
    
    return nil
}

// handleTimeout 处理单条超时记录
func (s *ApprovalTimeoutService) handleTimeout(ctx context.Context, record *ent.ApprovalRecord) error {
    // 获取超时配置
    timeoutConfig, err := s.getTimeoutConfig(ctx, record)
    if err != nil || timeoutConfig == nil {
        // 无超时配置，只记录日志
        return s.logTimeout(ctx, record)
    }
    
    // 处理每个超时动作
    for _, action := range timeoutConfig.Actions {
        switch action {
        case "notify":
            s.sendReminder(ctx, record, timeoutConfig.ReminderHours)
        case "escalate":
            return s.escalateToNextLevel(ctx, record, timeoutConfig)
        case "auto_approve":
            return s.autoApproveIfAllowed(ctx, record, timeoutConfig)
        case "auto_reject":
            return s.autoReject(ctx, record)
        }
    }
    
    return nil
}

// getTimeoutConfig 获取超时配置
func (s *ApprovalTimeoutService) getTimeoutConfig(ctx context.Context, record *ent.ApprovalRecord) (*TimeoutConfig, error) {
    workflow, err := s.client.ApprovalWorkflow.Query().
        Where(approvalworkflow.IDEQ(record.WorkflowID)).
        Only(ctx)
    if err != nil {
        return nil, err
    }
    
    nodes := s.parseWorkflowNodes(workflow.Nodes)
    for _, node := range nodes {
        if node.Level == record.CurrentLevel {
            return node.TimeoutConfig, nil
        }
    }
    
    return nil, nil
}

// escalateToNextLevel 升级到下一级
func (s *ApprovalTimeoutService) escalateToNextLevel(ctx context.Context, record *ent.ApprovalRecord, config *TimeoutConfig) error {
    var escalateTo int
    if config.EscalateTo != nil {
        escalateTo = *config.EscalateTo
    } else {
        // 查找上级审批人或管理员
        escalateTo = s.findEscalationTarget(ctx, record, config.EscalateLevel)
    }
    
    if escalateTo == 0 {
        return fmt.Errorf("无法找到升级目标")
    }
    
    // 查找被委托人
    user, err := s.client.User.Query().
        Where(user.IDEQ(escalateTo)).
        Only(ctx)
    if err != nil {
        return fmt.Errorf("未找到升级目标用户: %d", escalateTo)
    }
    
    // 创建新的审批记录给升级目标
    _, err = s.client.ApprovalRecord.Create().
        SetWorkflowID(record.WorkflowID).
        SetWorkflowName(record.WorkflowName).
        SetTicketID(record.TicketID).
        SetTicketNumber(record.TicketNumber).
        SetTicketTitle(record.TicketTitle).
        SetCurrentLevel(record.CurrentLevel).
        SetTotalLevels(record.TotalLevels).
        SetApproverID(escalateTo).
        SetApproverName(user.Name).
        SetStepOrder(record.StepOrder).
        SetStatus("pending").
        SetDueDate(time.Now().Add(24 * time.Hour)). // 新记录24小时后超时
        SetTenantID(record.TenantID).
        Save(ctx)
    if err != nil {
        return fmt.Errorf("failed to create escalated record: %w", err)
    }
    
    // 取消原记录
    _, err = s.client.ApprovalRecord.UpdateOneID(record.ID).
        SetStatus("escalated").
        Save(ctx)
    
    // 更新 SLA 记录
    s.updateSLARecord(ctx, record.ID, true, escalateTo)
    
    return err
}

// findEscalationTarget 查找升级目标
func (s *ApprovalTimeoutService) findEscalationTarget(ctx context.Context, record *ent.ApprovalRecord, level *int) int {
    // 优先使用配置的目标
    // 否则查找当前审批人的上级
    escalationLevel := 1
    if level != nil {
        escalationLevel = *level
    }
    
    // 简化：返回 tenant 管理员
    admin, err := s.client.User.Query().
        Where(user.TenantIDEQ(record.TenantID), user.RoleEQ(user.Role("admin"))).
        First(ctx)
    if err == nil && admin != nil {
        return admin.ID
    }
    
    return 0
}

// sendReminder 发送提醒
func (s *ApprovalTimeoutService) sendReminder(ctx context.Context, record *ent.ApprovalRecord, reminderHours []int) error {
    // 检查是否已发送过提醒
    sla, err := s.getSLARecord(ctx, record.ID)
    if err != nil {
        return err
    }
    
    hoursSinceStart := time.Since(record.CreatedAt).Hours()
    shouldRemind := false
    
    for _, hours := range reminderHours {
        if hoursSinceStart >= float64(hours) {
            shouldRemind = true
            break
        }
    }
    
    if shouldRemind && (sla == nil || sla.ReminderCount < len(reminderHours)) {
        // 发送提醒通知（集成通知服务）
        // notification.SendApprovalReminder(record)
        
        // 更新 SLA 记录
        s.incrementReminderCount(ctx, record.ID)
        
        s.logger.Info("Sent reminder for approval record", "record_id", record.ID)
    }
    
    return nil
}

// autoApproveIfAllowed 满足条件时自动通过
func (s *ApprovalTimeoutService) autoApproveIfAllowed(ctx context.Context, record *ent.ApprovalRecord, config *TimeoutConfig) error {
    if config.AutoApproveConditions == nil || len(config.AutoApproveConditions) == 0 {
        return nil
    }
    
    // 获取工单
    ticket, err := s.client.Ticket.Query().
        Where(ticket.IDEQ(record.TicketID)).
        Only(ctx)
    if err != nil {
        return fmt.Errorf("failed to get ticket: %w", err)
    }
    
    // 评估条件
    evaluator := NewApprovalConditionEvaluator(ticket)
    allowed, err := evaluator.EvaluateSimpleConditions(conditionsToMaps(config.AutoApproveConditions))
    if err != nil {
        return err
    }
    
    if allowed {
        // 自动通过
        _, err = s.client.ApprovalRecord.UpdateOneID(record.ID).
            SetStatus("approved").
            SetAction("auto_approve").
            SetComment("自动审批：超时满足条件").
            SetProcessedAt(time.Now()).
            Save(ctx)
        return err
    }
    
    return nil
}

// autoReject 自动拒绝
func (s *ApprovalTimeoutService) autoReject(ctx context.Context, record *ent.ApprovalRecord) error {
    _, err := s.client.ApprovalRecord.UpdateOneID(record.ID).
        SetStatus("rejected").
        SetAction("auto_reject").
        SetComment("自动拒绝：审批超时").
        SetProcessedAt(time.Now()).
        Save(ctx)
    
    // 取消其他待审批记录
    _, err = s.client.ApprovalRecord.Update().
        Where(
            approvalrecord.TicketIDEQ(record.TicketID),
            approvalrecord.StatusEQ("pending"),
        ).
        SetStatus("cancelled").
        Save(ctx)
    
    // 更新工单状态
    _, err = s.client.Ticket.UpdateOneID(record.TicketID).
        SetStatus("rejected").
        Save(ctx)
    
    return err
}

// 辅助方法
func (s *ApprovalTimeoutService) logTimeout(ctx context.Context, record *ent.ApprovalRecord) error {
    // 记录操作日志
    return s.createActionLog(ctx, record, "timeout", "system", "", "pending", "timeout")
}

func (s *ApprovalTimeoutService) updateSLARecord(ctx context.Context, recordID int, escalated bool, escalatedTo int) error {
    sla, err := s.getSLARecord(ctx, recordID)
    if err != nil || sla == nil {
        return nil
    }
    
    update := s.client.ApprovalSLARecord.UpdateOneID(sla.ID).
        SetEscalated(escalated)
    
    if escalated {
        update = update.SetEscalatedAt(time.Now()).
            SetEscalatedTo(escalatedTo)
    }
    
    _, err = update.Save(ctx)
    return err
}

func (s *ApprovalTimeoutService) incrementReminderCount(ctx context.Context, recordID int) error {
    sla, err := s.getSLARecord(ctx, recordID)
    if err != nil || sla == nil {
        return nil
    }
    
    _, err = s.client.ApprovalSLARecord.UpdateOneID(sla.ID).
        SetReminderCount(sla.ReminderCount + 1).
        SetFirstReminderTime(time.Now()).
        Save(ctx)
    return err
}

func (s *ApprovalTimeoutService) getSLARecord(ctx context.Context, recordID int) (*ent.ApprovalSLARecord, error) {
    return s.client.ApprovalSLARecord.Query().
        Where(approvalslarecord.RecordIDEQ(recordID)).
        First(ctx)
}

func (s *ApprovalTimeoutService) createActionLog(ctx context.Context, record *ent.ApprovalRecord, action, operatorName, comment, fromStatus, toStatus string) error {
    _, err := s.client.ApprovalActionLog.Create().
        SetRecordID(record.ID).
        SetAction(action).
        SetOperatorID(0). // system
        SetOperatorName(operatorName).
        SetComment(comment).
        SetFromStatus(fromStatus).
        SetToStatus(toStatus).
        Save(ctx)
    return err
}

func conditionsToMaps(conditions []ApprovalCondition) []map[string]interface{} {
    result := make([]map[string]interface{}, len(conditions))
    for i, c := range conditions {
        result[i] = map[string]interface{}{
            "field":    c.Field,
            "operator": c.Operator,
            "value":    c.Value,
        }
    }
    return result
}
```

#### 5.4.2 定时任务注册

```go
// 文件：itsm-backend/main.go 或单独的服务启动文件

func initCronJobs() {
    // 每5分钟检查一次超时
    cron.Every(5).Minutes().Do(func() {
        ctx := context.Background()
        timeoutService := service.NewApprovalTimeoutService(db, logger)
        if err := timeoutService.CheckTimeouts(ctx); err != nil {
            logger.Error("Failed to check approval timeouts", zap.Error(err))
        }
    })
}
```

---

### 5.5 委托规则管理

#### 5.5.1 委托规则服务

```go
// 文件：itsm-backend/service/delegation_service.go

package service

import (
    "context"
    "fmt"
    "time"
    
    "itsm-backend/ent"
    "itsm-backend/ent/delegationrule"
)

type DelegationService struct {
    client *ent.Client
    logger interface{}
}

func NewDelegationService(client *ent.Client, logger interface{}) *DelegationService {
    return &DelegationService{
        client: client,
        logger: logger,
    }
}

// CreateRule 创建委托规则
func (s *DelegationService) CreateRule(ctx context.Context, req *CreateDelegationRuleRequest, tenantID int) (*ent.DelegationRule, error) {
    rule, err := s.client.DelegationRule.Create().
        SetUserID(req.UserID).
        SetDelegateTo(req.DelegateTo).
        SetTicketTypes(req.TicketTypes).
        SetPriorityRange(req.PriorityRange).
        SetValidFrom(req.ValidFrom).
        SetNillableValidTo(req.ValidTo).
        SetActive(true).
        Save(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create delegation rule: %w", err)
    }
    return rule, nil
}

// ListRules 列出用户的委托规则
func (s *DelegationService) ListRules(ctx context.Context, userID int, tenantID int) ([]*ent.DelegationRule, error) {
    return s.client.DelegationRule.Query().
        Where(
            delegationrule.UserIDEQ(userID),
            delegationrule.Active(true),
            delegationrule.ValidFromLTE(time.Now()),
        ).
        All(ctx)
}

// CheckDelegation 检查当前用户是否有有效的委托
func (s *DelegationService) CheckDelegation(ctx context.Context, userID int, ticketType string, priority string) (*ent.DelegationRule, error) {
    now := time.Now()
    
    rules, err := s.client.DelegationRule.Query().
        Where(
            delegationrule.UserIDEQ(userID),
            delegationrule.Active(true),
            delegationrule.ValidFromLTE(now),
            delegationrule.Or(
                delegationrule.ValidToIsNil(),
                delegationrule.ValidToGTE(now),
            ),
        ).
        All(ctx)
    if err != nil {
        return nil, err
    }
    
    for _, rule := range rules {
        // 检查工单类型是否匹配
        if len(rule.TicketTypes) > 0 {
            if !contains(rule.TicketTypes, ticketType) {
                continue
            }
        }
        
        // 检查优先级是否匹配
        if len(rule.PriorityRange) > 0 {
            if !contains(rule.PriorityRange, priority) {
                continue
            }
        }
        
        return rule, nil
    }
    
    return nil, nil
}

// GetEffectiveApprover 获取实际审批人（考虑委托）
func (s *DelegationService) GetEffectiveApprover(ctx context.Context, approverID int, ticketType string, priority string) (int, error) {
    rule, err := s.CheckDelegation(ctx, approverID, ticketType, priority)
    if err != nil || rule == nil {
        return approverID, nil // 无委托，返回原审批人
    }
    
    // 验证被委托人是否可用
    delegate, err := s.client.User.Query().
        Where(user.IDEQ(rule.DelegateTo), user.Active(true)).
        Only(ctx)
    if err != nil {
        return approverID, nil // 被委托人不可用，返回原审批人
    }
    
    return delegate.ID, nil
}

// DeactivateRule 停用委托规则
func (s *DelegationService) DeactivateRule(ctx context.Context, ruleID int, userID int) error {
    _, err := s.client.DelegationRule.Update().
        Where(
            delegationrule.IDEQ(ruleID),
            delegationrule.UserIDEQ(userID),
        ).
        SetActive(false).
        Save(ctx)
    return err
}

func contains(arr []string, item string) bool {
    for _, v := range arr {
        if v == item {
            return true
        }
    }
    return false
}

// CreateDelegationRuleRequest DTO
type CreateDelegationRuleRequest struct {
    UserID        int        `json:"user_id"`
    DelegateTo    int        `json:"delegate_to"`
    TicketTypes   []string   `json:"ticket_types"`
    PriorityRange []string   `json:"priority_range"`
    ValidFrom     time.Time  `json:"valid_from"`
    ValidTo       *time.Time `json:"valid_to"`
}
```

#### 5.5.2 控制器

```go
// 文件：itsm-backend/controller/delegation_controller.go

package controller

import (
    "itsm-backend/common"
    "itsm-backend/service"
    
    "github.com/gin-gonic/gin"
)

type DelegationController struct {
    delegationService *service.DelegationService
}

func NewDelegationController(ds *service.DelegationService) *DelegationController {
    return &DelegationController{delegationService: ds}
}

// CreateRule 创建委托规则
// @Summary 创建委托规则
// @Router /api/v1/delegation-rules [post]
func (c *DelegationController) CreateRule(ctx *gin.Context) {
    var req service.CreateDelegationRuleRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        common.Fail(ctx, common.ParamErrorCode, err.Error())
        return
    }
    
    req.UserID = ctx.GetInt("user_id")
    
    rule, err := c.delegationService.CreateRule(ctx.Request.Context(), &req, ctx.GetInt("tenant_id"))
    if err != nil {
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }
    
    common.Success(ctx, rule)
}

// ListRules 列出委托规则
// @Summary 列出我的委托规则
// @Router /api/v1/delegation-rules [get]
func (c *DelegationController) ListRules(ctx *gin.Context) {
    userID := ctx.GetInt("user_id")
    
    rules, err := c.delegationService.ListRules(ctx.Request.Context(), userID, ctx.GetInt("tenant_id"))
    if err != nil {
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }
    
    common.Success(ctx, rules)
}

// DeleteRule 删除委托规则
// @Summary 删除委托规则
// @Router /api/v1/delegation-rules/:id [delete]
func (c *DelegationController) DeleteRule(ctx *gin.Context) {
    ruleID := ctx.GetInt("id")
    userID := ctx.GetInt("user_id")
    
    if err := c.delegationService.DeactivateRule(ctx.Request.Context(), ruleID, userID); err != nil {
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }
    
    common.Success(ctx, nil)
}
```

---

### 5.6 审批日志记录

#### 5.6.1 日志记录服务

```go
// 文件：itsm-backend/service/approval_log_service.go

package service

import (
    "context"
    "time"
    
    "itsm-backend/ent"
    "itsm-backend/ent/approvalactionlog"
)

type ApprovalLogService struct {
    client *ent.Client
}

func NewApprovalLogService(client *ent.Client) *ApprovalLogService {
    return &ApprovalLogService{client: client}
}

// LogAction 记录审批操作
func (s *ApprovalLogService) LogAction(ctx context.Context, recordID int, action string, operatorID int, operatorName string, operatorIP string, comment string, fromStatus string, toStatus string, relatedUsers []int) error {
    _, err := s.client.ApprovalActionLog.Create().
        SetRecordID(recordID).
        SetAction(action).
        SetOperatorID(operatorID).
        SetOperatorName(operatorName).
        SetOperatorIP(operatorIP).
        SetComment(comment).
        SetFromStatus(fromStatus).
        SetToStatus(toStatus).
        SetRelatedUsers(relatedUsers).
        SetCreatedAt(time.Now()).
        Save(ctx)
    return err
}

// GetLogsByRecord 获取审批记录的日志
func (s *ApprovalLogService) GetLogsByRecord(ctx context.Context, recordID int) ([]*ent.ApprovalActionLog, error) {
    return s.client.ApprovalActionLog.Query().
        Where(approvalactionlog.RecordIDEQ(recordID)).
        Order(ent.Asc(approvalactionlog.FieldCreatedAt)).
        All(ctx)
}

// GetLogsByOperator 获取操作人的日志
func (s *ApprovalLogService) GetLogsByOperator(ctx context.Context, operatorID int, tenantID int, page, pageSize int) ([]*ent.ApprovalActionLog, int, error) {
    query := s.client.ApprovalActionLog.Query().
        Where(approvalactionlog.OperatorIDEQ(operatorID))
    
    total, err := query.Count(ctx)
    if err != nil {
        return nil, 0, err
    }
    
    offset := (page - 1) * pageSize
    logs, err := query.
        Order(ent.Desc(approvalactionlog.FieldCreatedAt)).
        Offset(offset).
        Limit(pageSize).
        All(ctx)
    
    return logs, total, err
}
```

#### 5.6.2 修改审批提交流程

```go
// 文件：itsm-backend/service/approval_service.go
// 在 SubmitApproval 方法中添加日志记录

func (s *ApprovalService) SubmitApproval(ctx context.Context, recordID int, userID int, action string, comment string, delegateToUserID *int, tenantID int) error {
    // ... 现有逻辑 ...
    
    // 获取客户端IP
    clientIP := "" // 需要从调用方传入
    
    // 记录操作日志
    logService := NewApprovalLogService(s.client)
    err = logService.LogAction(
        ctx,
        recordID,
        action,
        userID,
        approver.Name, // 需要获取用户名称
        clientIP,
        comment,
        "pending",
        newStatus,
        relatedUsers,
    )
    if err != nil {
        s.logger.Warnw("Failed to log approval action", "error", err)
    }
    
    return nil
}
```

---

## 六、API 接口设计

### 6.1 接口列表

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| GET | `/api/v1/approval-workflows/:id/nodes` | 获取工作流节点配置 | - |
| POST | `/api/v1/approval-workflows/:id/nodes` | 添加工作流节点 | ApprovalNodeConfig |
| GET | `/api/v1/approval-records/:id/logs` | 获取审批日志 | - |
| GET | `/api/v1/delegation-rules` | 获取委托规则列表 | - |
| POST | `/api/v1/delegation-rules` | 创建委托规则 | CreateDelegationRuleRequest |
| PUT | `/api/v1/delegation-rules/:id` | 更新委托规则 | CreateDelegationRuleRequest |
| DELETE | `/api/v1/delegation-rules/:id` | 删除委托规则 | - |
| GET | `/api/v1/approval-sla/:record_id` | 获取SLA状态 | - |

### 6.2 请求/响应示例

#### 创建委托规则

**请求：**

```json
POST /api/v1/delegation-rules
{
    "delegate_to": 102,
    "ticket_types": ["incident", "change"],
    "priority_range": ["low", "medium"],
    "valid_from": "2026-05-01T00:00:00Z",
    "valid_to": "2026-05-31T23:59:59Z"
}
```

**响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 1,
        "user_id": 101,
        "delegate_to": 102,
        "ticket_types": ["incident", "change"],
        "priority_range": ["low", "medium"],
        "valid_from": "2026-05-01T00:00:00Z",
        "valid_to": "2026-05-31T23:59:59Z",
        "active": true,
        "created_at": "2026-05-02T10:00:00Z"
    }
}
```

#### 获取审批日志

**响应：**

```json
{
    "code": 0,
    "message": "success",
    "data": [
        {
            "id": 1,
            "record_id": 100,
            "action": "submit_approval",
            "operator_id": 101,
            "operator_name": "张三",
            "operator_ip": "192.168.1.100",
            "comment": "请领导审批",
            "from_status": "pending",
            "to_status": "pending",
            "created_at": "2026-05-01T09:00:00Z"
        },
        {
            "id": 2,
            "record_id": 100,
            "action": "approve",
            "operator_id": 102,
            "operator_name": "李四",
            "operator_ip": "192.168.1.101",
            "comment": "同意",
            "from_status": "pending",
            "to_status": "approved",
            "created_at": "2026-05-01T10:00:00Z"
        }
    ]
}
```

#### 增强的工作流节点配置

**请求：**

```json
POST /api/v1/approval-workflows/1/nodes
{
    "level": 2,
    "name": "部门经理审批",
    "approver_type": "department_head",
    "approval_mode": "quota",
    "minimum_approvals": 2,
    "allow_reject": true,
    "allow_delegate": true,
    "timeout_hours": 48,
    "timeout_config": {
        "actions": ["notify", "escalate"],
        "escalate_to": 999,
        "escalate_level": 1,
        "reminder_hours": [24, 48],
        "auto_approve_conditions": [
            {"field": "priority", "operator": "eq", "value": "low"}
        ]
    },
    "conditions": [
        {"field": "amount", "operator": "gt", "value": 10000}
    ],
    "condition_groups": [
        {
            "group_id": 1,
            "conditions": [
                {"field": "category", "operator": "eq", "value": "finance"},
                {"field": "amount", "operator": "gt", "value": 50000}
            ]
        },
        {
            "group_id": 2,
            "conditions": [
                {"field": "category", "operator": "eq", "value": "it"}
            ],
            "logical_operator": "OR"
        }
    ]
}
```

---

## 七、路由注册

```go
// 文件：itsm-backend/router/approval.go

func registerApprovalRoutes(r *gin.RouterGroup) {
    approval := r.Group("/approval")
    {
        // 工作流管理
        approval.GET("/workflows", workflowController.ListWorkflows)
        approval.POST("/workflows", workflowController.CreateWorkflow)
        approval.GET("/workflows/:id", workflowController.GetWorkflow)
        approval.PUT("/workflows/:id", workflowController.UpdateWorkflow)
        approval.DELETE("/workflows/:id", workflowController.DeleteWorkflow)
        approval.POST("/workflows/:id/nodes", workflowController.AddNode)        // 新增
        approval.GET("/workflows/:id/nodes", workflowController.GetNodes)        // 新增
        
        // 审批记录
        approval.GET("/records", approvalController.ListRecords)
        approval.GET("/records/:id", approvalController.GetRecord)
        approval.GET("/records/:id/logs", approvalController.GetLogs)           // 新增
        approval.POST("/records/:id/submit", approvalController.SubmitApproval)
        
        // SLA
        approval.GET("/sla/:record_id", approvalController.GetSLA)               // 新增
        
        // 委托规则
        approval.GET("/delegation-rules", delegationController.ListRules)         // 新增
        approval.POST("/delegation-rules", delegationController.CreateRule)      // 新增
        approval.PUT("/delegation-rules/:id", delegationController.UpdateRule)   // 新增
        approval.DELETE("/delegation-rules/:id", delegationController.DeleteRule) // 新增
    }
}
```

---

## 八、测试用例

### 8.1 复合条件测试

```go
// 文件：itsm-backend/service/approval_condition_test.go

func TestEvaluateConditionGroups(t *testing.T) {
    ticket := &ent.Ticket{
        Priority:   "high",
        Category:   "finance",
        Extra:      map[string]interface{}{"amount": 80000},
    }
    evaluator := NewApprovalConditionEvaluator(ticket)
    
    // 测试 OR 条件组
    groups := []map[string]interface{}{
        {
            "field":    "category",
            "operator": "eq",
            "value":    "finance",
            "group_id": float64(1),
        },
        {
            "field":            "amount",
            "operator":         "gt",
            "value":            50000,
            "group_id":         float64(1),
            "logical_operator": "OR",
        },
    }
    
    // category=finance OR amount>50000，结果应为 true
    result, err := evaluator.EvaluateConditionGroups(groups)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if !result {
        t.Errorf("expected true, got false")
    }
}
```

### 8.2 动态审批人测试

```go
// 文件：itsm-backend/service/approval_service_test.go

func TestResolveApprover(t *testing.T) {
    // 测试场景省略，需要准备测试数据
    tests := []struct {
        name          string
        assigneeType  string
        assigneeValue string
        expectError   bool
    }{
        {
            name:          "按角色查找",
            assigneeType:  "role",
            assigneeValue: "manager",
            expectError:   false,
        },
        {
            name:          "按用户ID查找",
            assigneeType:  "user",
            assigneeValue: "101",
            expectError:   false,
        },
        {
            name:          "无效类型",
            assigneeType:  "invalid",
            assigneeValue: "",
            expectError:   true,
        },
    }
    
    // ... 实现测试逻辑 ...
}
```

---

## 九、部署说明

### 9.1 数据库迁移

```bash
# 运行迁移脚本
cd itsm-backend
go run migrate_fresh.go
```

或手动执行 SQL：

```bash
psql -U postgres -d itsm -f migrations/2026-05-02_approval_enhancement.sql
```

### 9.2 启动检查

1. 检查新增表是否创建成功
2. 验证 cron 定时任务是否启动
3. 测试各新增 API 是否正常

---

## 十、验收标准

| 功能 | 验收条件 | 测试方法 |
|------|----------|----------|
| 复合条件 | `category=finance AND amount>50000 OR priority=critical` 正确解析 | 单元测试 |
| 动态审批人 | `department_head` 类型能正确解析并找到审批人 | 集成测试 |
| quota 模式 | 3/5 投票，3人同意后通过 | 单元测试 |
| 超时升级 | 48小时超时时自动升级给配置用户 | 手动测试 |
| 委托规则 | 委托期间审批自动转发给被委托人 | 集成测试 |
| 操作日志 | 每次操作都有完整日志记录 | 查看数据库 |

---

## 十一、参考文件

- [approval_service.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/approval_service.go) - 审批服务核心逻辑
- [workflow_controller.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/controller/workflow_controller.go) - 控制器
- [dto/approval.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/dto/) - 数据传输对象
