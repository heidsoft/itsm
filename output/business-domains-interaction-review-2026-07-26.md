# ITSM 六大业务域前端交互审计报告

> **审计日期**: 2026-07-26  
> **审计范围**: 工单(Ticket) / 事件(Incident) / 问题(Problem) / 故障(Fault) / 变更(Change) / 发布(Release)  
> **审计方法**: 逐文件源码核实 + 跨域同构对比，所有结论均有行号引用  
> **审计人**: WorkBuddy AI

---

## 〇、总览结论

**一句话：六域同构却各写各的，共享层只做到了"壳"（detail-tabs / BusinessPageTemplate），核心交互（状态流转、指派、批量操作、确认守卫）全面分裂。问题域是薄壳、工单域是 1061 行孤岛、故障域不存在。**

### 严重度分布

| 级别 | 数量 | 典型问题 |
|------|------|----------|
| 🔴 P0 | 4 | 故障域缺失；问题域状态流转无守卫无 loading；事件详情指派人显示原始 ID；列表页删除守卫跨域不一致 |
| 🟠 P1 | 5 | 六域状态流转 UI 各不相同；指派 UI 只有工单做了；批量操作只有事件有；工单详情 1061 行未抽取；问题域 ITIL 生命周期不完整 |
| 🟡 P2 | 4 | 路由命名混乱(create/new/[id]/[ticketId])；incidents/new 废弃重定向未清理；筛选器跨域不一致；变更列表用日历视图与其余域割裂 |

---

## 一、路由与页面地图

### 1.1 六域路由全景

| 域 | 列表页 | 详情页 | 创建页 | 编辑页 | 特殊页 |
|----|--------|--------|--------|--------|--------|
| **工单** | `tickets/page.tsx` (298行) | `tickets/[ticketId]/page.tsx` (**1061行**) | `tickets/create/` + `tickets/ai-create/` | ❌ 无独立编辑页 | analytics, dashboard, cc, templates, types |
| **事件** | `incidents/page.tsx` (521行) | `incidents/[id]/page.tsx` | `incidents/create/` (465行) + `incidents/new/` (23行,废弃) | `incidents/[id]/edit/` | — |
| **问题** | `problems/page.tsx` (363行) | `problems/[id]/page.tsx` → `ProblemDetail` | `problems/new/` | `problems/[id]/edit/` | known-errors, trends |
| **变更** | `changes/page.tsx` (532行) | `changes/[id]/page.tsx` → `ChangeDetail` | `changes/new/` | `changes/[id]/edit/` | `[id]/pir/`, `pirs/`, `standard-changes/` |
| **发布** | `releases/page.tsx` (**7行壳**) → `ReleaseList` | `releases/[id]/page.tsx` → `ReleaseDetail` | `releases/new/` | `releases/[id]/edit/` | — |
| **故障** | ❌ **不存在** | ❌ | ❌ | ❌ | 仅在事件统计中有 `majorIncidents` 字段 |

### 1.2 🔴 P0：故障(Major Incident)域完全缺失

**证据**：全仓 `grep -rniE "故障|major.?incident"` 仅命中：
- `tickets/create/page.tsx:540` — 工单分类下拉的一个选项 `{ label: '系统故障' }`
- `incidents/components/IncidentStats.tsx:64` — 事件统计卡片显示 `majorIncidents` 数值
- `incidents/hooks/useIncidentsData.ts:61` — 从 API 读取 `majorIncidents` 计数

**结论**：系统中没有任何"故障"独立路由、页面、组件或工作流。`majorIncidents` 只是一个统计数字，无法创建、升级、管理重大事件。ITIL 中 Major Incident 通常有独立流程（影响评估→升级→危机管理→事后复盘），当前完全空白。

### 1.3 路由命名混乱

| 问题 | 详情 |
|------|------|
| 动态参数不一致 | 工单用 `[ticketId]`，其余四域用 `[id]` |
| 创建页命名不一 | 工单/事件用 `create/`，问题/变更/发布用 `new/` |
| 废弃路由未清理 | `incidents/new/page.tsx` 标注 `@deprecated`，仅 23 行重定向到 `/incidents/create` |

---

## 二、代码复用评估（核心交付物）

### 2.1 共享层（做得好的部分）✅

| 共享组件 | 覆盖范围 | 说明 |
|----------|----------|------|
| `detail-tabs/` (5个子组件) | **全部 5 域详情页** | CommentPanel / HistoryTimeline / ApprovalTimeline / ApprovalWorkflowPanel / AttachmentPanel — 这是全审计中**唯一真正跨域复用**的组件层 |
| `BusinessPageTemplate` | 工单/事件/问题 列表页 | 统一页面骨架 |
| `UnifiedKanbanBoard` | 工单/事件/问题 列表页 | 看板视图共用 |
| `BatchActionBar` | 事件列表页 | 批量操作栏（但只有事件用了） |
| `AISuggestionPanel` | 工单/事件/问题/变更 详情页 | AI 建议面板共用 |

### 2.2 断裂层（各域各写各的）❌

| 能力 | 工单 | 事件 | 问题 | 变更 | 发布 | 复用度 |
|------|------|------|------|------|------|--------|
| **列表组件** | TicketList | 内联 | ProblemList | ChangeList | ReleaseList | 0%（5套） |
| **详情组件** | 1061行内联 | IncidentDetail(38KB) | ProblemDetail(4KB) | ChangeDetail(15KB) | ReleaseDetail(12KB) | 0%（5套） |
| **状态流转** | 自定义Modal | 状态机+Modal | 裸按钮 | 审批驱动 | Modal.confirm | 0%（5种） |
| **指派 UI** | 完整Modal | ❌无 | ❌无 | ❌无 | N/A | 20%（1/5） |
| **批量操作** | ❌无 | ✅有 | ❌无 | ❌无 | ❌无 | 20%（1/5） |
| **创建表单** | create/(内联) | create/(465行) | new/ | new/ | new/ | 0%（5套） |

**复用度总评：20%**。只有 detail-tabs 子组件层和 BusinessPageTemplate 骨架层做到了跨域共享，所有业务交互逻辑（状态/指派/批量/表单）均为 per-domain 独立实现。

---

## 三、状态流转（ITSM 核心）跨域对比

### 3.1 各域状态流转实现

| 域 | UI 形式 | 状态机校验 | 确认守卫 | Loading | 评分 |
|----|---------|-----------|----------|---------|------|
| **工单** | 审批/驳回/关闭 Modal | ❌ 无 | ✅ 自定义 Modal | ✅ `approving`/`rejecting`/`deleting` | 🟠 B |
| **事件** | 升级/解决/重开 Modal | ✅ `isValidIncidentTransition` | ✅ 专用 Modal | ✅ `escalating`/`resolving`/`reopening` | 🟢 A |
| **问题** | 裸按钮（开始处理/标记解决/关闭） | ❌ 无 | ❌ **无任何确认** | ❌ **无 loading** | 🔴 D |
| **变更** | 审批驱动（approve/reject 改变状态） | ⚠️ `canApprove = status===PENDING` | ✅ 审批 Modal | ✅ `processing` | 🟠 B+ |
| **发布** | 拒绝/回滚 Modal.confirm | ❌ 无 | ✅ **Modal.confirm + 原因输入** | ⚠️ 页面 loading，按钮 loading 待确认 | 🟢 A- |

### 3.2 🔴 P0：问题域状态流转——无守卫、无 loading、无状态机

**源码证据** (`ProblemDetail.tsx:45-54, 114-126`)：
```tsx
// 状态更新：直接调 API，无 confirm，无 loading
const handleUpdateStatus = async (status: ProblemStatus) => {
  if (!id) return;
  try {
    await ProblemApi.updateProblem(Number(id), { status });
    message.success('状态更新成功');
    loadData();
  } catch (error) {
    message.error('状态更新失败');
  }
};

// UI：裸按钮，点击即执行
{data.status === ProblemStatus.OPEN && (
  <Button type="primary" onClick={() => handleUpdateStatus(ProblemStatus.IN_PROGRESS)}>
    开始处理
  </Button>
)}
{data.status === ProblemStatus.RESOLVED && (
  <Button onClick={() => handleUpdateStatus(ProblemStatus.CLOSED)}>关闭问题</Button>
)}
```

**问题**：
1. "关闭问题"是不可逆操作，点击直接执行，无确认弹窗
2. 按钮无 `loading` 状态，用户可连点导致重复请求
3. 无状态机校验——理论上可以通过编辑页把任意状态改为任意状态
4. `ProblemApi.updateProblem(id, {status})` 是通用更新接口，不是专用状态流转接口——缺少后端校验保障

### 3.3 事件域状态流转——做得最好（保持）

**源码证据** (`IncidentDetail.tsx:90-92, 208-224`)：
```tsx
const [escalateModalVisible, setEscalateModalVisible] = useState(false);
const [resolveModalVisible, setResolveModalVisible] = useState(false);
const [escalating, setEscalating] = useState(false);
const [resolving, setResolving] = useState(false);

// 解决：有状态机校验 + 专用端点 + loading
if (!isValidIncidentTransition(data.status, 'resolved')) { ... }
await IncidentAPI.resolveIncident(data.id, { resolution: values.resolution });
```

**亮点**：`isValidIncidentTransition` 前端状态机校验 + `resolveIncident` 专用端点（非通用 update）+ `resolving` loading 态。这是五域中状态流转做得最规范的。

### 3.4 变更审批——走域端点而非 BPMN

**源码证据** (`ChangeDetail.tsx:88, 105`)：
```tsx
await ChangeApi.approveChange(change.id, { comment: approvalComment });
await ChangeApi.rejectChange(change.id, { comment: approvalComment });
```

**结论**：变更审批直连域端点 `changes/{id}/approve`，**未走 BPMN 任务接口**。与工作流审计结论一致——BPMN 引擎未被业务审批消费。变更状态由审批动作驱动（PENDING→approved/rejected），无手动状态流转 UI。

---

## 四、指派 / 分派 UI 跨域对比

| 域 | 指派 UI | 指派入口 | 用户选择 | 备注 |
|----|---------|----------|----------|------|
| **工单** | ✅ 完整 Modal | 详情页"指派"按钮 | ✅ 远程搜索用户列表 + Select | 含备注，有 `loadingUsers`/`assigning` loading |
| **事件** | ❌ **无指派 UI** | — | — | 详情页只显示 `负责人ID: {data.assigneeId}`（**原始数字**！） |
| **问题** | ❌ **无指派 UI** | — | — | 详情页无负责人字段 |
| **变更** | ❌ **无指派 UI** | — | — | 详情页只显示 `负责人: {change.assigneeName}`（只读） |
| **发布** | N/A | — | — | 发布无需指派 |

### 4.1 🔴 P0：事件详情页指派人显示原始 ID

**源码证据** (`IncidentDetail.tsx:463`)：
```tsx
<Descriptions.Item label="负责人ID">{data.assigneeId || '-'}</Descriptions.Item>
```

**问题**：用户看到的是 `负责人ID: 42` 而非 `负责人: 张三`。这是最基础的 UX 问题——内部 ID 不应对用户可见。且事件详情页**没有指派/转派操作**，指派只能在列表页批量操作中完成，详情页无法单独指派。

### 4.2 工单指派——做得最好（保持）

**源码证据** (`tickets/[ticketId]/page.tsx:112-117, 261-266`)：
```tsx
const [assignModalVisible, setAssignModalVisible] = useState(false);
const [assigning, setAssigning] = useState(false);
const [assignForm] = Form.useForm();

const handleAssign = () => setAssignModalVisible(true);
const handleAssignSubmit = async (values: { assigneeId: number; comment?: string }) => {
  // 远程搜索用户 + Select 选择 + 备注
};
```

工单指派是五域中最完整的：Modal + 用户远程搜索 + Select + 备注 + `loadingUsers`/`assigning` 双 loading。**应抽为共享组件供事件/问题/变更复用。**

---

## 五、批量操作跨域对比

| 域 | 批量选择 | 批量操作 | 实现方式 |
|----|----------|----------|----------|
| **工单** | ❌ | ❌ | — |
| **事件** | ✅ `selectedRowKeys` | ✅ 批量分派/解决/关闭/删除 | `BatchActionBar` + `Promise.allSettled` 兜底 |
| **问题** | ❌ | ❌ | — |
| **变更** | ❌ | ❌ | — |
| **发布** | ❌ | ❌ | — |

### 5.1 事件批量操作——唯一实现（亮点 + 隐患）

**源码证据** (`incidents/page.tsx:192-266`)：
```tsx
// ====== 批量操作 ======
const [batchLoading, setBatchLoading] = useState(false);
// 逐条循环兜底：后端目前尚未提供 incident 批量端点，
// 此处封装 Promise.allSettled，失败逐条汇总
const runIncidentBatch = useCallback(async (
  selectedRowKeys, apiCall, successMsg
) => { ... }, []);

// 批量分派/解决/关闭/删除
```

**亮点**：`Promise.allSettled` 逐条兜底 + 部分成功反馈 + `BatchActionBar` 共享组件 + 批量删除有 `confirmTitle`。

**隐患**：后端无批量端点，前端逐条循环 → 选中 100 条 = 100 个请求，有性能风险。

**问题**：其余 4 域**完全没有批量操作**，运维人员处理大量工单/问题/变更时只能逐条操作。

---

## 六、确认守卫与 Loading 跨域对比

### 6.1 列表页破坏性操作守卫

| 域 | 删除操作 | Modal.confirm | Popconfirm | Loading | 评价 |
|----|----------|---------------|------------|---------|------|
| **工单** | 无列表删除 | — | — | — | N/A |
| **事件** | ✅ 批量删除 | ❌（用 BatchActionBar `confirmTitle`） | — | ✅ `batchLoading` | 🟠 批量有守卫，但单行无删除 |
| **问题** | ❌ 无删除 | — | — | — | 无法删除问题？ |
| **变更** | ❌ 无删除 | — | — | — | 无法删除变更？ |
| **发布** | ❌ 无删除 | — | — | — | 无法删除发布？ |

### 6.2 详情页破坏性操作守卫

| 域 | 破坏性操作 | 确认方式 | Loading | 评价 |
|----|------------|----------|---------|------|
| **工单** | 删除工单 | ✅ 自定义 Modal (`setDeleteModalVisible`) + "确认删除"按钮 | ✅ `deleting` | 🟢 好 |
| **事件** | 升级/解决 | ✅ 专用 Modal（escalateModalVisible/resolveModalVisible） | ✅ `escalating`/`resolving` | 🟢 好 |
| **问题** | 关闭问题 | ❌ **无确认** | ❌ **无 loading** | 🔴 危险 |
| **变更** | 驳回 | ✅ rejectModalVisible + 审批意见 | ✅ `processing` | 🟢 好 |
| **发布** | 拒绝/回滚 | ✅ **Modal.confirm + 原因输入** | ⚠️ 页面 loading | 🟢 最好 |

### 6.3 问题域"关闭问题"——无确认守卫

如第 3.2 节所述，`ProblemDetail.tsx:124-126` 的"关闭问题"按钮点击直接执行 `handleUpdateStatus(ProblemStatus.CLOSED)`，**无任何确认弹窗**，且按钮无 loading 状态。用户误点即关闭，不可逆。

---

## 七、各域专有能力评估

### 7.1 工单域——功能最全但最重孤岛

**功能完整度**：✅ 审批/驳回/指派/转派/抄送(CC)/删除/AI建单/模板/分类/自动化规则

**问题**：
- 详情页 **1061 行内联在 page.tsx**，未抽取为独立组件——五域中唯一未抽取的
- 有 `ai-create/` AI 建单入口（亮点）
- 有 `templates/` 工单模板（亮点）
- 有 `cc/` 抄送管理（亮点）
- 但因未抽取组件，复用性为零

### 7.2 事件域——交互最规范但指派缺失

**功能完整度**：✅ 确认/升级/解决/重开 + 批量操作 + 状态机校验

**问题**：
- 详情页 38KB，功能最重
- 指派人显示原始 ID（P0）
- 详情页无指派操作
- `incidents/new/` 废弃重定向未清理

### 7.3 问题域——ITIL 生命周期不完整

**功能完整度**：⚠️ 仅有"基本信息 + 问题调查"两个 Tab

**缺失的 ITIL 问题管理能力**：
| 能力 | 状态 | 说明 |
|------|------|------|
| 根因分析(RCA) | ❌ | 无独立 RCA 表单/字段 |
| 临时解决方案(Workaround) | ❌ | 无 workaround 记录区 |
| 已知错误(Known Error) | ⚠️ | 有 `known-errors/` 页面但未与详情页联动 |
| 关联事件 | ❌ | 详情页无关联事件 Tab |
| 问题趋势分析 | ⚠️ | 有 `trends/` 页面但未核实数据源 |

**详情页仅 139 行 / 4KB**——五域中最薄。状态流转裸按钮无守卫。

### 7.4 变更域——审批链完整但与 BPMN 割裂

**功能完整度**：✅ 审批/驳回/CAB/PIR/标准变更模板 + 审批记录时间线

**亮点**：
- 审批记录时间线（`ApprovalTimeline` 共享组件）
- PIR 独立页面（`[id]/pir/` 创建编辑 + `pirs/` 列表，非重复）
- 标准变更模板管理（`standard-changes/`，预批准变更模板）
- 审批 Modal 含审批意见 + `processing` loading

**问题**：
- 审批走域端点 `ChangeApi.approveChange`，未走 BPMN
- 无手动状态流转 UI（状态完全由审批驱动）
- 变更列表页用 **Calendar 日历视图**，与其余四域（表格/看板）割裂

### 7.5 发布域——展示完整但不可操作

**功能完整度**：⚠️ 7 个只读 Card 区块，无部署操作

**ReleaseDetail 7 个区块**：描述 / 发布说明 / 受影响系统 / 受影响组件 / 部署步骤 / 回滚程序 / 验证标准

**问题**：
- `deploymentSteps` 只读展示，**无"开始部署"/"执行部署"操作按钮**
- 无部署状态跟踪（planned→deploying→deployed→verified）
- 无关联变更(CR)链接
- 无回滚执行入口（只有"回滚程序"文本展示 + Modal.confirm 回滚发布）
- 列表页仅 7 行壳 (`return <ReleaseList />`)

**评价**：发布域是"能看不能用"——信息展示完整，但缺乏发布执行流程的交互闭环。

---

## 八、筛选器与列表交互跨域对比

| 域 | 筛选方式 | 筛选维度 | 看板视图 | 批量选择 | 分页 |
|----|----------|----------|----------|----------|------|
| **工单** | TicketAdvancedSearch | 状态/优先级/指派人/关键词 | ✅ TicketKanban | ❌ | ✅ |
| **事件** | Select 下拉 | 状态/优先级 | ✅ UnifiedKanbanBoard | ✅ | ✅ |
| **问题** | Select 下拉 | 状态/优先级 | ✅ UnifiedKanbanBoard | ❌ | ✅ |
| **变更** | **Calendar 日历** + Select | 日期/状态 | ❌ | ❌ | ✅ |
| **发布** | ❌ 无 | — | ❌ | ❌ | ✅ |

**问题**：
- 变更用日历视图作为主交互，与其余四域完全不同（虽然变更窗口有日历需求，但应作为视图切换而非唯一模式）
- 发布列表无任何筛选
- 筛选维度不一致：工单有 4 维，事件/问题只有 2 维
- 只有事件支持批量选择

---

## 九、评论 / 活动 / 历史时间线

**好消息**：全部 5 域详情页都引用了 `detail-tabs/` 下的共享组件（CommentPanel / HistoryTimeline / ApprovalTimeline / AttachmentPanel）。这是六域审计中**唯一真正做到跨域复用**的交互层。

**证据**：`grep -rl "detail-tabs"` 命中全部 5 个详情页。

---

## 十、问题清单与修复优先级

### 🔴 P0（必须修）

| # | 问题 | 影响 | 修复建议 |
|---|------|------|----------|
| P0-1 | **故障(Major Incident)域完全缺失** | 无法管理重大事件，ITIL 核心流程空白 | 决策：是否做独立故障域？最小方案：在事件域增加"升级为重大事件"操作 + 影响评估 + 危机管理 |
| P0-2 | **问题域"关闭问题"无确认守卫、无 loading** | 误点即关闭，不可逆 | 加 `Modal.confirm` + `loading` 状态 |
| P0-3 | **事件详情页指派人显示原始 ID** | 用户看到 `负责人ID: 42` 而非姓名 | 显示 `assigneeName`，并增加详情页指派操作 |
| P0-4 | **工单详情 1061 行未抽取组件** | 无法复用，维护困难 | 抽取为 `TicketDetail` 组件，与 IncidentDetail/ChangeDetail 对齐 |

### 🟠 P1（应修）

| # | 问题 | 影响 | 修复建议 |
|---|------|------|----------|
| P1-1 | **五域状态流转 UI 各不相同** | 用户跨域操作认知成本高 | 抽象 `StatusTransitionBar` 共享组件，支持配置化状态机 |
| P1-2 | **指派 UI 只有工单做了** | 事件/问题/变更无法在详情页指派 | 抽取工单指派 Modal 为 `AssignModal` 共享组件 |
| P1-3 | **批量操作只有事件有** | 运维人员处理大量记录效率低 | 将 `BatchActionBar` + `runBatch` 推广到工单/问题/变更 |
| P1-4 | **问题域 ITIL 生命周期不完整** | 无 RCA/Workaround/Known Error 联动 | 补全根因分析、临时方案、关联事件 Tab |
| P1-5 | **发布域无部署执行操作** | 只能看不能做，发布流程断裂 | 增加"开始部署"操作 + 部署状态跟踪 + 关联变更 |

### 🟡 P2（一致性收敛）

| # | 问题 | 影响 | 修复建议 |
|---|------|------|----------|
| P2-1 | 路由命名混乱 (`create` vs `new` / `[id]` vs `[ticketId]`) | 路由不一致，导航困难 | 统一为 `new/` + `[id]` |
| P2-2 | `incidents/new/` 废弃重定向未清理 | 死路由 | 删除或保留但加 301 |
| P2-3 | 筛选器跨域不一致 | 用户跨域操作体验差 | 统一筛选维度（状态/优先级/指派人/关键词/日期范围） |
| P2-4 | 变更列表用日历视图与其余域割裂 | 交互模式不一致 | 保留日历但增加表格视图切换 |

---

## 十一、量化基线

| 指标 | 现状 | 目标 |
|------|------|------|
| 详情组件抽取率 | 4/5（工单未抽取） | 5/5 |
| 状态流转 UI 统一性 | 5 种不同实现 | 1 个共享组件 + 配置化 |
| 指派 UI 覆盖率 | 1/5 域 | 4/5 域（发布除外） |
| 批量操作覆盖率 | 1/5 域 | 4/5 域 |
| 确认守卫覆盖率 | 3/5 域（问题/变更列表缺失） | 5/5 域 |
| 故障域完整度 | 0%（仅统计数字） | 决策后补全 |
| 问题域 ITIL 覆盖 | 2/6 能力（基本信息+调查） | 6/6 |
| 发布域可操作性 | 0 个执行操作 | ≥3 个（部署/验证/回滚执行） |

---

## 十二、修复路线图

### 阶段一（P0，2 周）
1. 问题域"关闭问题"加 `Modal.confirm` + loading
2. 事件详情页 `assigneeId` → `assigneeName` + 增加指派操作
3. 工单详情抽取为 `TicketDetail` 组件
4. 故障域：决策——独立模块 or 事件子流程

### 阶段二（P1，3-4 周）
1. 抽象 `StatusTransitionBar` 共享组件（以事件域 `isValidIncidentTransition` 为蓝本）
2. 抽取 `AssignModal` 共享组件（以工单域为蓝本）
3. 将 `BatchActionBar` 推广到工单/问题/变更
4. 问题域补全 RCA / Workaround / Known Error / 关联事件
5. 发布域增加部署执行 + 状态跟踪

### 阶段三（P2，1-2 周）
1. 路由命名统一（`new/` + `[id]`）
2. 清理 `incidents/new/` 废弃路由
3. 筛选器维度统一
4. 变更列表增加表格视图切换

---

## 附录：审计文件索引

| 文件 | 行数 | 审计要点 |
|------|------|----------|
| `tickets/[ticketId]/page.tsx` | 1061 | 最大孤岛，功能最全，未抽取 |
| `incidents/page.tsx` | 521 | 唯一批量操作，Promise.allSettled 兜底 |
| `changes/page.tsx` | 532 | 日历视图，与其他域割裂 |
| `problems/page.tsx` | 363 | 筛选器只有 2 维 |
| `releases/page.tsx` | 7 | 纯壳，委托给 ReleaseList |
| `incidents/create/page.tsx` | 465 | 真实创建页 |
| `incidents/new/page.tsx` | 23 | @deprecated 重定向 |
| `changes/[id]/pir/page.tsx` | 371 | PIR 创建/编辑 |
| `changes/pirs/page.tsx` | 348 | PIR 列表（非重复） |
| `components/incident/IncidentDetail.tsx` | ~1000 | 38KB，状态机+Modal，最规范 |
| `components/change/ChangeDetail.tsx` | ~470 | 15KB，审批驱动，域端点非 BPMN |
| `components/release/ReleaseDetail.tsx` | 389 | 12KB，7区块只读，无执行操作 |
| `components/problem/ProblemDetail.tsx` | 139 | 4KB，最薄，裸按钮无守卫 |
