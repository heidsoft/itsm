# CMDB 操作交互专项评审（前端 UX 流）

- **日期**：2026-07-26
- **范围**：`itsm-frontend` CMDB 模块全部交互流（列表/表单/详情/关系/拓扑/批量/导入导出/云资源）
- **方法**：组件源码探索 + 关键文件逐行核实（CIEditorForm、CIBasicInfo、CIList、CIRelationshipManager、TopologyGraph、cloud-* 页、cmdb-advanced-api）
- **证据可信度**：✅ 已读源码核实（标注处）；🔍 由组件结构推断（少量）

---

## 0. 复核与整改状态（2026-07-26）

本轮复核确认文档对“模型驱动交互未闭环”的判断成立，但字段名需校正：HTTP/前端配置项响应使用 `attributes`，`customAttributes` 只是表单内部字段，不应写成详情接口契约。

已完成：

- ✅ `CIEditorForm` 统一按类型渲染：数字使用 `InputNumber`、布尔使用 `Switch`、日期/时间使用原生日期输入、枚举使用 `Select`、长文本使用 `TextArea`。
- ✅ 云资源动态属性和 CI 类型动态属性复用同一控件映射，避免两套行为。
- ✅ `CIBasicInfo` 展示 `ci.attributes`；有类型模板时显示业务标签，无模板时安全回退属性 key，并格式化布尔/对象值。
- ✅ CI 创建与编辑页复用有效类型模板继承合并逻辑。
- ✅ 创建/编辑 CI 共用脏数据离开守卫，覆盖取消、`beforeunload`、站内链接和浏览器后退。
- ✅ `datetime-local` 在控件边界转换为 RFC3339，符合后端属性校验契约。
- ✅ 云账号/云服务创建与编辑增加提交锁；关系、云账号、云服务删除增加行级 loading。
- ✅ 修正详情页返回目标为真实列表 `/cmdb/ci`，补充关键图标按钮 `aria-label`。
- ✅ 内嵌拓扑刷新按钮增加 loading、disabled 和无障碍标签。
- ✅ CI 编辑器和云服务补齐 AWS 枚举。
- ✅ 已核查 CMDB 生产组件，不存在 mock 数据、随机数据或空按钮处理器。

仍需独立迭代：

- ⏳ 服务端导入/导出虽然已有任务 API，但完整向导涉及模板、上传、字段映射、异步任务进度、错误文件下载和权限审计，不应以一个“上传按钮”冒充完成。
- ⏳ 独立拓扑页与详情拓扑仍需产品级合并；本轮先收敛刷新反馈，未删除任何一套现有能力。
- ⏳ 自定义属性数据库检索和排序依赖建模层类型化检索投影，前端不能用全量内存过滤替代。

## 1. 结论速览

CMDB 的交互「骨架尚可、细节割裂」：**关系管理、破坏性守卫是亮点**；但**动态属性的「模型驱动」在交互层只做了一半**——创建表单能动态渲染，详情页却不显示，且控件类型严重退化（int/bool/date 全变文本框）。更大的缺口是**导入/导出服务已具备但前端零向导**，以及**拓扑两套实现不一致**。

| 交互流 | 评价 | 关键证据 |
|---|---|---|
| CI 列表过滤/搜索/批量 | 🟡 仅固定列过滤，无自定义属性过滤；批量栏可用 | CIList.tsx:358-385, 397-414 |
| CI 创建/编辑表单 | 🔴 模型驱动仅半套；控件类型退化；详情不显示动态属性 | CIEditorForm.tsx:254-278, CIBasicInfo.tsx |
| CI 详情页 | 🟠 属性面板不渲染 `customAttributes`；无内联编辑 | CIDetail/CIBasicInfo |
| 关系管理 | 🟢 Modal+方向+强度/影响+环检测+Popconfirm | CIRelationshipManager.tsx |
| 拓扑视图 | 🟠 两套 reactflow 实现不一致（demo vs 完整） | topology/page.tsx vs TopologyGraph.tsx |
| 批量导入/导出 | 🔴 服务层齐全，前端零向导/零选项 UI | cmdb-advanced-api.ts:270-312（无调用方） |
| 状态反馈 | 🟠 未用 `LoadingEmptyError`（含 `CMDBLoadingEmptyError` 变体） | 各页手搓 Spin/Empty/Skeleton |
| 破坏性守卫 | 🟢 Modal.confirm/Popconfirm 一致；创建页脏数据守卫 | CIList.tsx:157-200, create page:39-65 |
| 云资源流 | 🟠 独立 CRUD、枚举与 CI 表单不一致；仅「新建CI」复用 | cloud-*.tsx, ci-editor-shared.ts:25-31 |

---

## 2. 逐项问题与证据

### 2.1 🔴 CI 表单：模型驱动只做了一半 + 控件类型退化（已核实）
`src/components/cmdb/CIEditorForm.tsx`：基础/归属/云资源字段是**硬编码** `Form.Item`；只有「扩展属性（来自 CI 类型模板）」与「云资源动态属性」由 `attributeSchema` 动态生成。

```tsx
// CIEditorForm.tsx:254-278  仅识别 'select'
{typeSchemaFields.map(field => (
  <Form.Item key={field.key} label={field.label||field.key}
    name={['customAttributes', field.key]}
    rules={field.required ? [{required:true, message:`请选择${field.label||field.key}`}] : undefined}>
    {field.type === 'select' ? (
      <Select allowClear options={(field.options||[]).map(o=>({label:o,value:o}))} />
    ) : (
      <Input ... />   // ← int/bool/date 全部退化成纯文本框
    )}
  </Form.Item>
))}
```
- **后果**：数字字段无 `InputNumber`（无法限制/校验数值）、布尔无 `Switch`、日期无 `DatePicker` → 录入体验差、数据质量无保证（可填任意字符串）。
- **校验仅 `required`**：无类型/格式/唯一性客户端校验；`unique` 字段（后端 `ci_attribute_definition.unique`）前端完全不提示，重复只能等后端报错。
- **逃生舱不一致**：另有一个手工 JSON 文本框直接编 `customAttributes`（CIEditorForm.tsx:283-291），与模型驱动 Select 并存，易产生「两套入口」困惑。
- 亮点：`CIEditorForm` 创建/编辑两页复用；`normalizeSchemaFields` 鲁棒归一化 string/array/object 三种 schema 形态。

### 2.2 🔴 详情页不渲染动态属性（已核实）
`src/components/cmdb/ci-detail/sections/CIBasicInfo.tsx` 用 `Descriptions` 只读展示，**只渲染约 25 个硬编码字段（资产分类/标签/序列号/型号/厂商/位置/环境/云_* 等），完全不渲染 `customAttributes`**（grep `customAttributes` 在 `ci-detail/` 下无任何命中）。
- **后果**：用户在表单里填的扩展属性（如服务器的「资产编号」「质保到期日」），在详情页**看不到**——创建与查看断层，模型驱动形同虚设。
- 编辑入口无内联编辑，必须跳 `/cmdb/cis/[id]/edit` 整页重填。

### 2.3 🟡 CI 列表：过滤仅固定列，无自定义属性过滤
`src/components/cmdb/CIList.tsx:358-385` 只有 `ciTypeId / status / name` 三个固定筛选项 + 名称防抖搜索；**无 sorter**；**不能按 CI 类型自定义属性过滤**（即使后端已能存）。
- 批量栏（选中后出现）导出/批量删除可用（397-414）；但「导出」是**客户端手拼 CSV，只含固定列，不含动态属性**（203-233）；批量删除走 `Modal.confirm`+`Promise.allSettled`（良好）。
- 入口路由 `/cmdb` 实际是概览台 `CSDMHub`，真正列表在 `/cmdb/ci`——两级入口略绕。

### 2.4 🟢 关系管理（亮点）
`src/components/cmdb/CIRelationshipManager.tsx`（同时被 `/cmdb/relationships` 复用）：
- 新增关系用 Modal：目标 CI（防抖搜索）、关系类型（显示双向/单向）、强度、影响程度、描述。
- **客户端环检测**（DFS `hasPath`，拓扑接口不可用时降级后端校验）→ 防止成环。
- 删除用 `Popconfirm` 守卫。交互设计在 CMDB 中最完整。

### 2.5 🟠 拓扑：两套 reactflow 实现不一致
1. 独立页 `cmdb/topology/page.tsx`：选根 CI + 深度 1-4；节点用 emoji；网格布局；点节点开 `Drawer`；**无 MiniMap**；加载=Spin/空=Empty（更像 demo）。
2. 内嵌组件 `TopologyGraph.tsx`（详情页「服务拓扑」tab）：径向布局；节点带状态色+重要性角标；含 **MiniMap、图例、影响分析 Drawer**（上游/下游/关键依赖/受影响工单+事件）；加载/错误/空三态齐全。
- **两者交互、节点样式、能力明显分叉** → 应合并为单一组件，复用内嵌版（更完整）。

### 2.6 🔴 导入/导出：服务齐全，前端零向导（最大功能缺口）
- 后端 `CMDBAdvancedApi.createImportTask/getImportTasks/createExportTask/getExportTasks`（`cmdb-advanced-api.ts:270-312`）支持 xlsx/csv、字段选择、错误明细。
- 全仓 grep 仅 API 定义与测试引用，**无任何页面/组件调用** → 引导式导入（模板下载/字段映射/校验反馈）与导出选项 UI **完全缺失**。
- 现有「导出」仅是列表页客户端拼 CSV（无模板、无字段映射、无服务端校验）。

### 2.7 🟠 状态反馈未统一
共享组件 `LoadingEmptyError` 已提供 `CMDBLoadingEmptyError` 变体，但 CMDB 所有页面均**未使用**，各页手搓：`CSDMHub` 用 `<Card loading>`、`CIList` 用 `<Table loading>+<Empty>`、`CIDetail` 用 `<Skeleton>+<Result 404>`、`TopologyGraph` 手搓 Spin/Alert/Empty。→ 加载/空/错体验不一致（与全局审计结论一致）。

### 2.8 🟠 云资源流与通用 CI 割裂
- `cloud-resources/accounts/services` 是**独立 CRUD 流，未集成 `CIEditorForm`**；各自硬编码表单、非模型驱动。
- **枚举不一致**：`cloud-resources.tsx` 的 `providerOptions`（含 `'aws'`）与 `ci-editor-shared.ts` 的 `cloudProviderOptions`（无 aws）对不上 → 同一概念两处维护，易漂移。
- 亮点：云资源页「新建CI / 绑定」通道设计合理，把发现数据带回 CMDB 的链路清晰（且复用了 `CIEditorForm`）。

---

## 3. 根因（与设计/建模层呼应）

1. **`ci_attribute_definition` 缺 `type→控件` 元数据贯通**：后端属性 `type` 字段（string/int/bool/...）在**前端没有完整的 type→antd 控件映射**，导致表单退化、详情不渲染。模型定义了类型，但交互层没收口。
2. **列表过滤只认固定列**：后端属性是 JSON（不可查，见建模评审），前端自然也无法做属性级过滤 UI——**建模层的「JSON 不可查」直接限制了交互层的过滤能力**，两层问题叠加。
3. **详情只读组件与表单组件各写一套字段**：未共用「按 schema 渲染属性」的逻辑，造成断层。
4. **重复实现**：拓扑两套、云 CRUD 与 CI 表单各一套——缺统一 CMDB 交互组件库。

---

## 4. 优先修复路线图

### P0（止血：让「模型驱动」真正可用）
1. **补全 type→控件映射**：在 `CIEditorForm` 按 `field.type` 渲染 `InputNumber`(int)/`Switch`(bool)/`DatePicker`(date)/`Select`(enum)/`Input`(text)/`Input.TextArea`(text long)；详情页 `CIBasicInfo` 用**同一映射**渲染 `customAttributes`（只读态）。一处定义、两处复用。
2. **详情页显示动态属性**：`CIBasicInfo` 遍历 `ci.customAttributes` 按 schema 渲染（缺失 schema 时按 key 直显）；消除「创建能填、详情看不见」。
3. **接入导入/导出向导**：基于已有 `CMDBAdvancedApi` 做引导式导入（模板下载→字段映射→上传→错误明细）与导出（格式/字段选择/进度）；这是最大功能缺口。

### P1（一致性）
4. **合并两套拓扑**为单一 `TopologyGraph` 组件（保留 MiniMap/图例/影响分析），独立页与详情 tab 共用。
5. **统一状态反馈**：CMDB 各页接入 `CMDBLoadingEmptyError`；错误态区分「无数据/加载失败」并带重试。
6. **统一云厂商枚举**：`providerOptions` 与 `cloudProviderOptions` 合并到单一常量；云资源 CRUD 复用 `CIEditorForm`（或抽公共属性表单）。

### P2（打磨）
7. 列表增加自定义属性过滤（依赖建模层「类型化属性表」落地后）；增加列排序 `sorter`。
8. 客户端补类型/格式/唯一性校验；`unique` 属性提交前查重提示。
9. 详情页加内联编辑（原地改属性，无需整页跳编辑）；关系管理加合法组合约束提示。

---

## 5. 一句话总结
CMDB 交互的**关系管理、破坏性守卫、云→CI 通道**做得不错；但「模型驱动」在交互层只兑现了一半——**表单控件退化、详情不显动态属性、导入导出无 UI、拓扑两套实现**。修好 P0 三项（控件映射贯通 + 详情渲染动态属性 + 接入导入导出向导），CMDB 才真正从「能建模」走向「好用」。
