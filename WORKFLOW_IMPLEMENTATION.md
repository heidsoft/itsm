# ITSM 工单流转与审批系统实现文档

## 概述

本文档描述了ITSM系统中工单类型管理、审批流程和工单流转功能的完整实现。

## 功能特性

### 1. 工单类型管理

#### 1.1 功能特点
- **灵活的工单类型定义**：支持自定义工单类型编码、名称、图标和颜色
- **自定义字段配置**：支持多种字段类型（文本、数字、日期、选择器等）
- **条件显示**：支持字段的条件显示逻辑
- **字段验证**：支持必填、正则表达式等验证规则

#### 1.2 核心组件
- **前端页面**：`src/app/(main)/tickets/types/page.tsx`
- **表单组件**：`src/components/business/TicketTypeFormModal.tsx`
- **类型定义**：`src/types/ticket-type.ts`

#### 1.3 API接口
```
GET    /api/v1/ticket-types          # 获取工单类型列表
POST   /api/v1/ticket-types          # 创建工单类型
GET    /api/v1/ticket-types/:id      # 获取工单类型详情
PUT    /api/v1/ticket-types/:id      # 更新工单类型
DELETE /api/v1/ticket-types/:id      # 删除工单类型
```

### 2. 审批流程管理

#### 2.1 功能特点
- **多级审批**：支持配置多级审批链
- **灵活的审批方式**：
  - 任一通过（any）：任意一个审批人通过即可
  - 全部通过（all）：所有审批人必须通过
  - 多数通过（majority）：超过半数通过即可
- **驳回处理**：支持驳回到上一级或结束流程
- **审批委派**：支持将审批委派给其他人
- **超时处理**：支持配置审批超时自动处理

#### 2.2 审批节点配置
每个审批级别可配置：
- 审批人列表（支持用户、角色、部门）
- 审批方式（any/all/majority）
- 是否允许驳回和委派
- 驳回后的处理方式
- 超时时间和超时处理方式
- 触发条件

#### 2.3 核心组件
- **审批进度组件**：`src/components/business/TicketWorkflowActions.tsx`
- **类型定义**：`src/types/ticket-type.ts`（ApprovalChainDefinition）

### 3. 工单流转功能

#### 3.1 支持的流转操作

##### 3.1.1 接单 (Accept)
- 功能：接收并开始处理工单
- 权限：分配人或有权限的处理人
- 状态变更：`new/open` → `in_progress`
- API：`POST /api/v1/tickets/workflow/accept`

##### 3.1.2 驳回 (Reject)
- 功能：拒绝工单并说明原因
- 权限：当前处理人或审批人
- 状态变更：任意状态 → `rejected`
- API：`POST /api/v1/tickets/workflow/reject`
- 必填：驳回原因、详细说明

##### 3.1.3 撤回 (Withdraw)
- 功能：撤回已提交的工单
- 权限：工单创建人
- 状态变更：非关闭状态 → `cancelled`
- API：`POST /api/v1/tickets/workflow/withdraw`
- 必填：撤回原因

##### 3.1.4 转发 (Forward)
- 功能：将工单转发给其他人处理
- 权限：当前处理人或有权限的用户
- 选项：是否转移所有权
- API：`POST /api/v1/tickets/workflow/forward`
- 必填：接收人、转发说明

##### 3.1.5 抄送 (CC)
- 功能：将工单抄送给相关人员
- 权限：创建人、处理人或有权限的用户
- 特点：抄送人可查看工单但不参与处理
- API：`POST /api/v1/tickets/workflow/cc`
- 必填：抄送人列表

##### 3.1.6 审批 (Approve/Reject)
- 功能：审批工单（通过/拒绝/委派）
- 权限：当前级别的审批人
- 操作类型：
  - `approve`：审批通过
  - `reject`：审批拒绝
  - `delegate`：委派给他人
- API：`POST /api/v1/tickets/workflow/approve`
- 必填：审批意见

##### 3.1.7 解决 (Resolve)
- 功能：标记工单为已解决
- 权限：当前处理人
- 状态变更：`in_progress` → `resolved`
- API：`POST /api/v1/tickets/workflow/resolve`
- 必填：解决方案
- 可选：解决类型、工作笔记

##### 3.1.8 关闭 (Close)
- 功能：关闭已解决的工单
- 权限：处理人或有权限的用户
- 前置条件：工单状态为 `resolved`
- 状态变更：`resolved` → `closed`
- API：`POST /api/v1/tickets/workflow/close`

##### 3.1.9 重开 (Reopen)
- 功能：重新打开已关闭的工单
- 权限：创建人或有权限的用户
- 前置条件：工单状态为 `closed` 或 `resolved`
- 状态变更：`closed/resolved` → `open`
- API：`POST /api/v1/tickets/workflow/reopen`
- 必填：重开原因

#### 3.2 核心组件
- **流转操作组件**：`src/components/business/TicketWorkflowActions.tsx`
- **审批进度组件**：`ApprovalProgress` 组件
- **类型定义**：`src/types/ticket-workflow.ts`

#### 3.3 流转状态查询
- API：`GET /api/v1/tickets/:id/workflow/state`
- 返回信息：
  - 当前状态
  - 当前处理人
  - 审批状态和进度
  - 可执行的操作列表
  - 操作权限

### 4. 数据库设计

#### 4.1 主要数据表

##### ticket_types（工单类型表）
```sql
- id: 主键
- code: 类型编码（租户内唯一）
- name: 类型名称
- status: 状态 (active/inactive/draft)
- custom_fields: 自定义字段定义(JSONB)
- approval_enabled: 是否启用审批
- approval_chain: 审批链定义(JSONB)
- sla_enabled: 是否启用SLA
- auto_assign_enabled: 是否启用自动分配
- assignment_rules: 分配规则(JSONB)
- notification_config: 通知配置(JSONB)
- permission_config: 权限配置(JSONB)
```

##### ticket_workflow_records（流转记录表）
```sql
- id: 主键
- ticket_id: 工单ID
- action: 流转操作类型
- from_status: 原状态
- to_status: 新状态
- operator_id: 操作人ID
- from_user_id: 来源用户ID
- to_user_id: 目标用户ID
- comment: 备注
- reason: 原因
- metadata: 附加元数据(JSONB)
```

##### ticket_approvals（审批记录表）
```sql
- id: 主键
- ticket_id: 工单ID
- level: 审批级别
- level_name: 级别名称
- approver_id: 审批人ID
- status: 审批状态
- action: 审批操作 (approve/reject/delegate)
- comment: 审批意见
- processed_at: 处理时间
- delegate_to_user_id: 委派给的用户ID
```

##### ticket_cc（抄送表）
```sql
- id: 主键
- ticket_id: 工单ID
- user_id: 抄送用户ID
- added_by: 添加人ID
- added_at: 添加时间
- is_active: 是否激活
```

#### 4.2 迁移脚本
位置：`itsm-backend/migrate_ticket_workflow.sql`

## 后端实现

### 5.1 DTO定义
- `dto/ticket_type_dto.go`：工单类型相关DTO
- `dto/ticket_workflow_dto.go`：工单流转相关DTO

### 5.2 Service层
- `service/ticket_type_service.go`：工单类型服务
  - CreateTicketType：创建工单类型
  - UpdateTicketType：更新工单类型
  - GetTicketType：获取工单类型详情
  - ListTicketTypes：获取工单类型列表
  - DeleteTicketType：删除工单类型

- `service/ticket_workflow_service.go`：工单流转服务
  - AcceptTicket：接单
  - RejectTicket：驳回
  - WithdrawTicket：撤回
  - ForwardTicket：转发
  - CCTicket：抄送
  - ApproveTicket：审批
  - ResolveTicket：解决
  - CloseTicket：关闭
  - ReopenTicket：重开
  - GetTicketWorkflowState：获取流转状态

### 5.3 Controller层
- `controller/ticket_type_controller.go`：工单类型控制器
- `controller/ticket_workflow_controller.go`：工单流转控制器

## 前端实现

### 6.1 页面组件
- **工单类型管理页**：`src/app/(main)/tickets/types/page.tsx`
  - 工单类型列表展示
  - 创建/编辑/删除工单类型
  - 查看工单类型详情
  - 启用/停用工单类型

### 6.2 业务组件
- **工单类型表单**：`src/components/business/TicketTypeFormModal.tsx`
  - 基本信息配置
  - 自定义字段配置
  - 审批流程配置
  - SLA配置
  - 自动分配配置

- **工单流转操作**：`src/components/business/TicketWorkflowActions.tsx`
  - 主要操作按钮（接单、审批、解决等）
  - 更多操作菜单（驳回、撤回、转发、抄送等）
  - 操作表单模态框
  - 审批进度展示

### 6.3 类型定义
- `src/types/ticket-type.ts`：工单类型相关类型
- `src/types/ticket-workflow.ts`：工单流转相关类型

## 使用示例

### 7.1 创建工单类型
```typescript
const ticketType = {
  code: 'incident',
  name: '故障工单',
  description: '用于报告和跟踪系统故障',
  customFields: [
    {
      id: 'field_1',
      name: 'impact',
      label: '影响范围',
      type: 'select',
      required: true,
      options: [
        { label: '个人', value: 'individual' },
        { label: '部门', value: 'department' },
        { label: '全公司', value: 'company' }
      ]
    }
  ],
  approvalEnabled: true,
  approvalChain: [
    {
      id: 'level_1',
      level: 1,
      name: '部门主管审批',
      approvers: [{ type: 'role', value: 'manager', name: '部门主管' }],
      approvalType: 'any',
      allowReject: true,
      allowDelegate: true,
      rejectAction: 'end'
    },
    {
      id: 'level_2',
      level: 2,
      name: 'IT总监审批',
      approvers: [{ type: 'role', value: 'it_director', name: 'IT总监' }],
      approvalType: 'all',
      allowReject: true,
      allowDelegate: false,
      rejectAction: 'return'
    }
  ]
};
```

### 7.2 工单流转示例
```typescript
// 1. 接单
await acceptTicket({ ticketId: 123, comment: '已接单，开始处理' });

// 2. 转发给其他人
await forwardTicket({
  ticketId: 123,
  toUserId: 456,
  comment: '此问题需要网络组处理',
  transferOwnership: true
});

// 3. 解决工单
await resolveTicket({
  ticketId: 123,
  resolution: '已重启服务器，问题解决',
  resolutionCategory: '配置更改',
  workNotes: '发现是服务配置问题'
});

// 4. 关闭工单
await closeTicket({
  ticketId: 123,
  closeReason: '已解决',
  closeNotes: '用户确认问题已解决'
});
```

## 部署说明

### 8.1 数据库迁移
```bash
# 执行数据库迁移脚本
psql -U postgres -d itsm -f migrate_ticket_workflow.sql
```

### 8.2 后端配置
1. 确保数据库连接配置正确
2. 注册新的服务和控制器
3. 添加路由配置

### 8.3 前端配置
1. 安装依赖：`npm install`
2. 构建项目：`npm run build`
3. 启动服务：`npm start`

## 权限控制

### 9.1 工单类型管理权限
- 创建工单类型：`ticket_type.create`
- 查看工单类型：`ticket_type.read`
- 更新工单类型：`ticket_type.update`
- 删除工单类型：`ticket_type.delete`

### 9.2 工单流转权限
- 接单：分配人或有权限的处理人
- 驳回：当前处理人或审批人
- 撤回：工单创建人
- 转发：当前处理人或有权限的用户
- 抄送：创建人、处理人或有权限的用户
- 审批：当前级别的审批人
- 解决：当前处理人
- 关闭：处理人或有权限的用户
- 重开：创建人或有权限的用户

## 扩展建议

### 10.1 功能增强
1. **工作流可视化设计器**：提供拖拽式的工作流设计界面
2. **自动化规则引擎**：基于条件自动执行操作
3. **SLA监控和告警**：实时监控SLA状态并发送告警
4. **移动端支持**：开发移动应用支持审批和流转
5. **智能分配**：基于AI的智能工单分配

### 10.2 性能优化
1. 审批流程缓存
2. 流转记录分页加载
3. 大量抄送人的批量处理
4. 工单类型配置缓存

### 10.3 监控和分析
1. 流转效率分析
2. 审批时长统计
3. 瓶颈识别
4. 用户操作行为分析

## 总结

本实现提供了完整的工单类型管理、审批流程和工单流转功能，支持：
- ✅ 灵活的工单类型定义和自定义字段
- ✅ 多级审批流程配置
- ✅ 完整的工单流转操作（接单、驳回、撤回、抄送等）
- ✅ 审批进度跟踪和可视化
- ✅ 权限控制和租户隔离
- ✅ 完整的API接口和前端界面

系统已经具备生产环境部署的基础，可以根据实际需求进行定制和扩展。

