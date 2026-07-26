# CMDB 按钮动作专项审计（Button Actions）

- **日期**：2026-07-26
- **范围**：`itsm-frontend` CMDB 模块全部按钮 / 动作触发器（工具栏、表格行内、详情页、关系管理、拓扑、云资源 CRUD、弹窗 footer）
- **方法**：逐文件枚举所有按钮 + 关键项源码逐行核实（cloud-accounts Modal、edit 页取消守卫、CIDetail 返回导航）
- **证据可信度**：✅ 已读源码核实（标注处）；🔍 由组件结构推断（少量）

---

## 0. 复核与整改状态（2026-07-26）

审计列出的两个 P0 均已确认并修复：

- ✅ 云账号、云服务创建/编辑四处 Modal 增加独立 `confirmLoading` 和同步提交锁，重复点击不会再次发起写请求。
- ✅ 编辑 CI 与创建 CI 共用离开守卫：取消、刷新/关闭、站内链接与浏览器后退均触发未保存确认，保存成功后才清除 dirty 状态。

同时完成：

- ✅ 关系、云账号、云服务删除增加行级 loading，并阻止并行重复删除。
- ✅ 云账号编辑/删除、关系删除、拓扑刷新补充 `aria-label`。
- ✅ 详情页 404、面包屑和“返回列表”统一指向 `/cmdb/ci`。
- ✅ 详情拓扑刷新增加 loading/disabled。
- ✅ 编辑成功返回真实配置项列表，而非 CMDB Hub。
- ✅ 修复编辑页切换类型时误写 `custom_attributes` 的字段名错误，统一为 `customAttributes`。

未在本轮强行统一“页面创建 vs Modal 创建”范式；这是信息架构选择，不是按钮缺陷，应结合云账号密钥配置复杂度单独决策。

## 1. 总体结论

**正面**：所有「删除类」破坏性操作**都已有确认守卫**（`Modal.confirm` 或 `Popconfirm`），未发现死按钮 / TODO / 空处理器 / 会抛错的函数。

**主要问题（按严重度）**：
1. 🔴 **云账号/云服务的创建·编辑 Modal 的 OK 按钮缺 `confirmLoading`** → 用户连点「确定」会**重复提交、创建重复记录**（双提交 bug）。
2. 🔴 **编辑 CI 页缺失「脏数据离开确认」**：取消直接 `router.back()`，改动静默丢失；而「新建」页有守卫 → 同一表单两处行为不一致。
3. 🟠 **部分写操作按钮无 loading 态**：关系删除、云账号删除、云服务删除确认后按钮不转圈（虽有 Popconfirm 确认，但请求在途无反馈）。
4. 🟠 **图标按钮缺 `aria-label`**：仅 Tooltip，无障碍实现与 `CIList` 不一致。
5. 🟠 **「返回列表 / 配置项列表」导航目标错误**：详情页/云服务的「返回列表」跳到 hub(`/cmdb`) 而非真实列表(`/cmdb/ci`)，需再点一次。
6. 🟠 **详情页拓扑「刷新」按钮无 loading/disabled/aria-label**，与独立拓扑页（有 loading+disabled）不一致。
7. 🟡 **「新建配置项」CTA 命名不一致**：`录入资产` / `录入配置项` / `新建CI` 三种称呼，且「配置项 vs 资产」术语混用；hub 落地页无显著主新建按钮。
8. 🟡 **「创建」交互范式分裂**：CI 用独立页面，云账号/云服务用内联 Modal。

---

## 2. 已核实的高危证据（✅ 源码）

### 2.1 🔴 云账号创建 Modal 无 `confirmLoading`（双提交风险）
`src/app/(main)/cmdb/cloud-accounts/page.tsx:303-345`：
```tsx
<Modal title="新增云账号" open={createOpen}
  onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
  onOk={handleCreate}   // ← 无 confirmLoading
  width={500}>
```
编辑 Modal（`:348-356`）`onOk={handleUpdate}` 同样无 `confirmLoading`。`handleCreate/handleUpdate` 为异步写请求，连点「确定」会重复创建云账号/云服务。**对照**：同模块 `cloud-resources/page.tsx:382` 的绑定 Modal 已用 `confirmLoading={bindSubmitting}`（正确写法），应照搬。

### 2.2 🔴 编辑 CI 页取消无脏数据守卫（与新建页不一致）
`src/app/(main)/cmdb/cis/[id]/edit/page.tsx:270`：
```tsx
<CIEditorForm ... onCancel={() => router.back()} ... />
```
而新建页 `create/page.tsx:49-65` 的 `handleCancel` 在 `dirtyRef` 为真时弹 `Modal.confirm` 并注册 `beforeunload`（`:38-47`）。编辑页**改动会静默丢失**。

### 2.3 🟠 详情页「返回列表」跳到 hub 而非列表（导航指向错误）
`src/components/cmdb/ci-detail/CIDetail.tsx:140-146`：
```tsx
<Button icon={<ArrowLeft />}
  onClick={() => router.push('/cmdb')}   // ← 跳 hub，不是真实列表 /cmdb/ci
>返回列表</Button>
```
同文件面包屑「配置项列表」(:133)、404 页「返回列表」(:53) 均跳 `/cmdb`。用户从 `/cmdb/ci` 进详情后「返回列表」回到 hub，需再点一次。

---

## 3. 按钮动作全量清单（节选关键项）

> 完整 17 个文件的按钮表见审计过程；以下为按风险分级的关键动作。

### 🟢 做得好的（应保持）
| 按钮 | 位置 | 处理 | 守卫/反馈 |
|---|---|---|---|
| 列表行「删除」 | CIList.tsx:315 | `handleDelete`→`Modal.confirm` | ✅ 确认含「关系受影响」提示 + `message` 反馈 |
| 批量删除 | CIList.tsx:406 | `handleBatchDelete`→`Modal.confirm`(含数量/不可恢复) | ✅ + 部分成功反馈 |
| 关系「创建」 | CIRelationshipManager:483 | `handleCreate`(含环检测) | ✅ `loading={creating}` + 环检测结果反馈 |
| 关系表行「删除」 | CIRelationshipManager:315 | `Popconfirm` | ✅ 确认；⚠️ 无按钮 loading |
| 影响分析执行 | CIImpactAnalysisTab:123 | `onAnalyze` | ✅ `loading` + 失败 `message.error` |
| 变更历史加载 | CIChangeHistoryTab:27 | `onLoad` | ✅ `loading` + 失败反馈 |
| 绑定 Modal 确定 | cloud-resources:382 | `handleBindExisting` | ✅ `confirmLoading={bindSubmitting}` |
| 独立拓扑「刷新」 | topology/page.tsx:201 | `loadTopology` | ✅ `loading` + `disabled={!selectedCI}` |
| 新建页「取消」 | create/page.tsx:49 | dirty? `Modal.confirm` | ✅ 脏数据守卫 + `beforeunload` |
| 列表行「编辑」 | CIList.tsx:307 | push edit | ✅ `Tooltip`+`aria-label` |

### 🔴 高危（必修）
| 按钮 | 位置 | 问题 |
|---|---|---|
| 新增云账号「确定」 | cloud-accounts:310 | `onOk` 无 `confirmLoading` → 双提交 |
| 编辑云账号「确定」 | cloud-accounts:356 | 同上 |
| 新增云服务「确定」 | cloud-services:314 | 同上 |
| 编辑云服务「确定」 | cloud-services:401 | 同上 |
| 编辑 CI「取消」 | edit/page.tsx:270 | 无脏数据守卫，改动静默丢 |

### 🟠 中危（应修）
| 按钮 | 位置 | 问题 |
|---|---|---|
| 关系删除 | CIRelationshipManager:315 | 确认后无按钮 loading |
| 云账号删除 | cloud-accounts:219 | 同上（Popconfirm 已有） |
| 云服务删除 | cloud-services:251 | 同上 |
| 图标按钮（删/看/编辑/刷新） | CIRelationshipManager/cloud-*/TopologyGraph | 缺 `aria-label`（仅 Tooltip） |
| 详情「返回列表」/面包屑 | CIDetail:142/133/53、cloud-services:274 | 跳 hub 非列表 |
| 详情拓扑「刷新」 | TopologyGraph:297 | 无 loading/disabled/aria-label |

### 🟡 低危（一致性收敛）
- 新建 CTA 命名：`CIList`「录入资产」/ `CSDMHub`「录入配置项」/ `cloud-resources`「新建CI」三称混用；术语「配置项 vs 资产」不统一。
- hub 落地页无 prominent 主新建按钮。
- 「创建」范式分裂：CI=页面，云=Modal。
- 云厂商枚举：`cloud-services` 缺 `'aws'`，`cloud-accounts`/`cloud-resources` 含 `'aws'`（数据源不一致）。

---

## 4. 优先修复路线图

### P0（双提交 / 数据丢失，最高优先）
1. **云账号/云服务 4 处 Modal 加 `confirmLoading`**：`cloud-accounts:310/356`、`cloud-services:314/401` 的 `onOk` 改为 `async () => { setSubmitting(true); try{...}finally{setSubmitting(false)} }` 并 `confirmLoading={submitting}`（照搬 `cloud-resources:382` 写法）。
2. **编辑页补脏数据守卫**：`edit/page.tsx` 的 `onCancel` 复用新建页的 `handleCancel`（dirty 判断 + `Modal.confirm` + `beforeunload`）；或把守卫逻辑抽到 `CIEditorForm` 内部（创建/编辑共用）。

### P1（反馈与无障碍一致性）
3. **删除类按钮加 loading**：关系删除、云账号/云服务删除在 `handleDelete` 期间置 `loading`，Popconfirm `onConfirm` 的 Promise 解析后再关闭。
4. **图标按钮统一补 `aria-label`**：`CIRelationshipManager`、各 `cloud-*`、TopologyGraph 的图标按钮加 `aria-label`（与 `CIList` 对齐）。
5. **修正导航目标**：`CIDetail` 与 `cloud-services` 的「返回列表/配置项列表」改为 `router.push('/cmdb/ci')`（真实列表）；404 页可保留跳 hub。

### P2（术语/范式收敛）
6. 统一新建 CTA 命名为「新建配置项」（全站），术语统一用「配置项(CI)」；hub 落地页补一个 `type="primary"` 主新建按钮。
7. 统一「创建」交互范式（建议 CI 与云资源新建均走页面，或均走 Modal，二选一并文档化）。
8. 统一云厂商枚举数据源（单一 `providerOptions` 含 aws）。
9. 详情页拓扑「刷新」按钮加 `loading`+`disabled`+`aria-label`，与独立拓扑页对齐。

---

## 5. 一句话总结
CMDB 的**破坏性操作确认守卫做的是对的**（无死按钮、无漏确认），但**写操作按钮的 loading/防双提交与导航指向有硬伤**：云账号/云服务 Modal 缺 `confirmLoading`（可重复创建）、编辑页取消无脏数据守卫（改动静默丢）、「返回列表」跳错页。修好 P0 两项即可堵住最主要的数据安全与一致性漏洞。
