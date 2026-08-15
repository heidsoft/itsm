# ITSM同类Bug测试计划 v2.0

## Bug修复状态汇总

| Bug ID | 文件 | 问题描述 | 状态 | 修复内容 |
|--------|------|----------|------|----------|
| #13 | workflow/versions/page.tsx | 工作流版本状态显示英文 | ✅ 已修复 | 添加 workflowVersionStatusMap |
| #14 | workflow/instances/page.tsx | 工作流实例状态显示英文 | ✅ 已修复 | 添加 statusTextMap |
| #15 | cmdb/topology/page.tsx | CMDB节点状态/类型显示英文 | ✅ 已修复 | 添加 ciTypeNameMap, ciStatusNameMap, criticalityNameMap |
| #16 | service-catalog/detail/[id]/page.tsx | 服务目录状态显示英文 | ✅ 已修复 | 添加 serviceStatusMap |
| #17 | approvals/page.tsx | 审批状态显示英文 | ✅ 已修复 | 添加 approvalStatusMap |
| #18 | incidents/create/page.tsx | CI状态显示英文 | ✅ 已修复 | 添加 ciStatusNameMap |
| #19 | cmdb/cloud-resources/page.tsx | 云资源状态显示英文 | ✅ 已修复 | 添加 cloudResourceStatusTextMap |
| #20 | problems/known-errors/page.tsx | 已知错误状态显示 | ✅ 已正确 | 代码已有 statusConfig, severityConfig |

---

## 修复详情

### Bug #13: workflow/versions/page.tsx

**问题**: 工作流版本状态显示英文

**修复方案**:
```typescript
// 工作流版本状态映射
const workflowVersionStatusMap: Record<string, { text: string; color: string }> = {
  active: { text: '已激活', color: 'green' },
  draft: { text: '草稿', color: 'default' },
  inactive: { text: '未激活', color: 'default' },
};

const getVersionStatusText = (status: string): string => {
  return workflowVersionStatusMap[status]?.text || status;
};
```

**修改位置**:
- 第25-36行: 添加状态映射
- 第102-105行: 表格列渲染
- 第261-264行: 详情Modal显示

---

### Bug #14: workflow/instances/page.tsx

**问题**: 工作流实例状态显示英文

**修复方案**:
```typescript
// 工作流实例状态文本映射
const statusTextMap: Record<string, string> = {
  running: '运行中',
  completed: '已完成',
  suspended: '已暂停',
  terminated: '已终止',
  failed: '失败',
};
```

**修改位置**:
- 第46-53行: 添加状态文本映射
- 第198行: 表格列渲染
- 第344-345行: 详情Modal显示

---

### Bug #15: cmdb/topology/page.tsx

**问题**: CMDB拓扑节点状态/类型/关键程度显示英文

**修复方案**:
```typescript
// CMDB类型中文映射
const ciTypeNameMap: Record<string, string> = {
  server: '服务器', database: '数据库', application: '应用程序',
  network: '网络设备', storage: '存储设备', cloud: '云资源',
};

// CMDB状态中文映射
const ciStatusNameMap: Record<string, string> = {
  active: '活跃', inactive: '未激活', maintenance: '维护中', retired: '已下线',
};

// 关键程度中文映射
const criticalityNameMap: Record<string, string> = {
  critical: '关键', high: '高', medium: '中', low: '低',
};
```

**修改位置**:
- 第21-45行: 添加所有映射
- 第55-56行: 节点类型和状态显示
- 第125-127行: 详情Drawer显示

---

### Bug #16: service-catalog/detail/[id]/page.tsx

**问题**: 服务目录状态显示英文

**修复方案**:
```typescript
// 服务目录状态映射
const serviceStatusMap: Record<string, { text: string; color: string }> = {
  published: { text: '已发布', color: 'green' },
  draft: { text: '草稿', color: 'default' },
  archived: { text: '已归档', color: 'gray' },
};
```

**修改位置**:
- 第10-15行: 添加状态映射
- 第56-58行: 状态显示

---

### Bug #17: approvals/page.tsx

**问题**: 审批状态显示英文

**修复方案**:
```typescript
// 审批状态中文映射
const approvalStatusMap: Record<string, { text: string; color: string }> = {
  pending: { text: '待审批', color: 'gold' },
  approved: { text: '已批准', color: 'green' },
  rejected: { text: '已拒绝', color: 'red' },
  cancelled: { text: '已取消', color: 'gray' },
};
```

**修改位置**:
- 第81-87行: 添加状态映射
- 第693-696行: 详情Drawer状态显示

---

### Bug #18: incidents/create/page.tsx

**问题**: 事件创建时CI状态显示英文

**修复方案**:
```typescript
// CI状态中文映射
const ciStatusNameMap: Record<string, string> = {
  active: '活跃', inactive: '未激活', maintenance: '维护中', retired: '已下线',
};
```

**修改位置**:
- 第18-24行: 添加状态映射
- 第312行: CI搜索结果状态显示

---

### Bug #19: cmdb/cloud-resources/page.tsx

**问题**: 云资源状态显示英文

**修复方案**:
```typescript
// 云资源状态中文映射
const cloudResourceStatusTextMap: Record<string, string> = {
  running: '运行中', stopped: '已停止', active: '活跃',
  inactive: '未激活', available: '可用', unavailable: '不可用',
  pending: '处理中', failed: '失败',
};
```

**修改位置**:
- 第32-41行: 添加状态映射
- 第447行: 资源详情状态显示

---

### Bug #20: problems/known-errors/page.tsx

**问题**: 已知错误状态显示英文

**状态**: ✅ 已正确处理

代码已有完整的 statusConfig 和 severityConfig:
```typescript
const statusConfig: Record<string, { color: string; text: string }> = {
  draft: { color: 'default', text: '草稿' },
  active: { color: 'processing', text: '活动中' },
  resolved: { color: 'success', text: '已解决' },
  deprecated: { color: 'error', text: '已废弃' },
};

const severityConfig: Record<string, { color: string; text: string }> = {
  critical: { color: 'red', text: '严重' },
  high: { color: 'orange', text: '高' },
  medium: { color: 'gold', text: '中' },
  low: { color: 'green', text: '低' },
};
```

---

## 测试验证清单

- [x] Bug #13: 工作流版本状态显示中文
- [x] Bug #14: 工作流实例状态显示中文
- [x] Bug #15: CMDB拓扑节点状态/类型/关键程度显示中文
- [x] Bug #16: 服务目录状态显示中文
- [x] Bug #17: 审批状态显示中文
- [x] Bug #18: CI状态显示中文
- [x] Bug #19: 云资源状态显示中文
- [x] Bug #20: 已知错误状态显示中文

---

## 通用本地化映射表

以下映射表可在多个组件中复用:

```typescript
// 状态映射
const STATUS_MAP: Record<string, string> = {
  // 工作流状态
  active: '已激活',
  draft: '草稿',
  inactive: '未激活',
  running: '运行中',
  completed: '已完成',
  suspended: '已暂停',
  terminated: '已终止',
  // 服务目录状态
  published: '已发布',
  // 审批状态
  pending: '待审批',
  approved: '已批准',
  rejected: '已拒绝',
  // CMDB状态
  active: '活跃',
  inactive: '未激活',
  // 云资源状态
  running: '运行中',
  stopped: '已停止',
  available: '可用',
  unavailable: '不可用',
};

// CI类型映射
const CI_TYPE_MAP: Record<string, string> = {
  database: '数据库',
  server: '服务器',
  network: '网络设备',
  storage: '存储设备',
  application: '应用程序',
  cloud: '云资源',
};

// 关键程度映射
const CRITICALITY_MAP: Record<string, string> = {
  critical: '关键',
  high: '高',
  medium: '中',
  low: '低',
};
```

---

**创建时间**：2026-07-06
**更新完成时间**：2026-07-06
**状态**：✅ 全部修复完成
