# 状态定义统一修复指南

## 问题描述
前后端工单状态定义不一致，导致状态流转错误。

## 已完成的修复

### 1. 后端状态常量补充 ✅

**文件**: `itsm-backend/common/constants.go`

**修改内容**:
```go
const (
  TicketStatusNew              = "new"
  TicketStatusOpen             = "open"
  TicketStatusInProgress       = "in_progress"
  TicketStatusPending          = "pending"
  TicketStatusResolved         = "resolved"
  TicketStatusClosed           = "closed"
  TicketStatusCancelled        = "cancelled"
  TicketStatusAssigned         = "assigned"           // ✅ 已存在
  TicketStatusPendingApproval  = "pending_approval"   // ✅ 新增
  TicketStatusRejected         = "rejected"           // ✅ 新增
)
```

### 2. 前端状态枚举补充 ✅

**文件**: `itsm-frontend/src/constants/taxonomy.ts`

**修改内容**:
```typescript
export enum TicketStatus {
  NEW = 'new',
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  PENDING_APPROVAL = 'pending_approval',
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
  REJECTED = 'rejected',
  ASSIGNED = 'assigned',  // ✅ 新增
}
```

## 待手动修复的文件

### 3. 后端状态转换规则 ⚠️

**文件**: `itsm-backend/service/ticket_lifecycle_service.go:320-332`

**当前代码**:
```go
validTransitions := map[string][]string{
  common.TicketStatusNew:        {common.TicketStatusOpen, common.TicketStatusAssigned},
  common.TicketStatusAssigned:   {common.TicketStatusInProgress, common.TicketStatusPending, common.TicketStatusClosed},
  common.TicketStatusOpen:       {common.TicketStatusInProgress, common.TicketStatusPending, common.TicketStatusClosed},
  common.TicketStatusInProgress: {common.TicketStatusResolved, common.TicketStatusPending, common.TicketStatusOpen},
  common.TicketStatusPending:     {common.TicketStatusInProgress, common.TicketStatusResolved, common.TicketStatusOpen},
  common.TicketStatusResolved:   {common.TicketStatusClosed, common.TicketStatusOpen},
  common.TicketStatusClosed:      {},
}
```

**修复后的代码**:
```go
validTransitions := map[string][]string{
  common.TicketStatusNew:              {common.TicketStatusOpen, common.TicketStatusAssigned},
  common.TicketStatusAssigned:         {common.TicketStatusInProgress, common.TicketStatusPending, common.TicketStatusClosed},
  common.TicketStatusOpen:             {common.TicketStatusInProgress, common.TicketStatusPending, common.TicketStatusClosed},
  common.TicketStatusInProgress:       {common.TicketStatusResolved, common.TicketStatusPending, common.TicketStatusOpen},
  common.TicketStatusPending:           {common.TicketStatusInProgress, common.TicketStatusResolved, common.TicketStatusOpen},
  common.TicketStatusResolved:         {common.TicketStatusClosed, common.TicketStatusOpen},
  common.TicketStatusClosed:            {},
  // 新增状态转换规则
  common.TicketStatusPendingApproval: {common.TicketStatusOpen, common.TicketStatusRejected, common.TicketStatusCancelled},
  common.TicketStatusRejected:         {common.TicketStatusOpen, common.TicketStatusCancelled},
  common.TicketStatusCancelled:        {},
}
```

**修复说明**:
- 添加 `PENDING_APPROVAL` 状态的转换规则：可转为 `OPEN`(审批通过)、`REJECTED`(审批拒绝)、`CANCELLED`(取消)
- 添加 `REJECTED` 状态的转换规则：可重新打开或取消
- 添加 `CANCELLED` 状态的转换规则：终态，不可转换

### 4. 前端状态机同步 ⚠️

**文件**: `itsm-frontend/src/lib/utils/workflow-state-machine.ts`

**需要同步更新**:
```typescript
export const VALID_TICKET_TRANSITIONS: Record<TicketStatus, TicketStatus[]> = {
  [TicketStatus.NEW]: [TicketStatus.OPEN, TicketStatus.CANCELLED],
  [TicketStatus.OPEN]: [
    TicketStatus.IN_PROGRESS,
    TicketStatus.PENDING,
    TicketStatus.PENDING_APPROVAL,  // ✅ 新增
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.IN_PROGRESS]: [
    TicketStatus.PENDING,
    TicketStatus.PENDING_APPROVAL,  // ✅ 新增
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.PENDING_APPROVAL]: [
    TicketStatus.OPEN,      // 审批通过
    TicketStatus.REJECTED,  // 审批拒绝
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.PENDING]: [
    TicketStatus.IN_PROGRESS,
    TicketStatus.PENDING_APPROVAL,  // ✅ 新增
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.RESOLVED]: [TicketStatus.CLOSED, TicketStatus.OPEN],
  [TicketStatus.CLOSED]: [],
  [TicketStatus.CANCELLED]: [],
  [TicketStatus.REJECTED]: [TicketStatus.OPEN, TicketStatus.CANCELLED],
  [TicketStatus.ASSIGNED]: [TicketStatus.IN_PROGRESS, TicketStatus.PENDING, TicketStatus.CANCELLED],  // ✅ 新增
};
```

## 验证步骤

### 1. 编译验证
```bash
# 后端
cd itsm-backend && go build

# 前端
cd itsm-frontend && npm run build
```

### 2. 状态转换测试
```go
func TestTicketStatusTransitions(t *testing.T) {
  // 测试新增状态转换
  assert.True(t, IsValidTicketStatusTransition("pending_approval", "open"))
  assert.True(t, IsValidTicketStatusTransition("pending_approval", "rejected"))
  assert.True(t, IsValidTicketStatusTransition("rejected", "open"))
  assert.False(t, IsValidTicketStatusTransition("cancelled", "open"))
}
```

## 影响范围

### 需要检查的文件
1. 工单创建逻辑 - `ticket_core_service.go`
2. 工单更新逻辑 - `ticket_core_service.go`
3. 审批流程 - `approval_service.go`
4. 前端工单列表组件
5. 前端工单详情组件

### 需要更新的数据库
- 检查现有工单是否有非法状态
- 可能需要数据迁移脚本

## 后续工作

1. ⏳ 手动修改状态转换规则
2. ⏳ 同步前端状态机
3. ⏳ 运行测试验证
4. ⏳ 更新相关文档

---

**修复状态**:
- ✅ 后端状态常量已修复
- ✅ 前端状态枚举已修复
- ⏳ 状态转换规则待手动修复

**优先级**: 高
**预计时间**: 1小时
