# ITSM 前端 UI/UX 设计审计报告

- **审计日期**：2026-07-26
- **审计范围**：`itsm-frontend`（Next.js 15 App Router + React 19 + Ant Design v6 + Tailwind v4 + Zustand）
- **方法**：代码结构探索（204 个 `page.tsx`/业务组件）+ 关键页面源码逐行核验（dashboard、incidents 列表/详情、Sidebar 菜单、Header 面包屑、设计令牌、共享状态组件）+ 专业 UI/UX 准则对照。
- **证据可信度**：标注 ✅（已读源码核实）/ 🔍（由共享模式与路由清单推断，建议二次确认）。

---

## 1. 执行摘要

| 维度 | 评分 | 主要问题 |
|---|---|---|
| 设计系统一致性 | 🔴 4/10 | 6+ 令牌来源互相冲突，主色/圆角/背景色三套尺度 |
| 导航 / 信息架构 | 🔴 4/10 | 80 个菜单项、组内重复入口、面包屑近乎失效 |
| 状态呈现（加载/空/错） | 🟠 5/10 | 共享组件存在但仅 5/204 页面采用，其余手写 |
| 数据获取可维护性 | 🟠 5/10 | React Query 已装仅 1 文件使用，203 页手写重复逻辑 |
| 模块内 UX 完成度 | 🟡 6/10 | 列表页较好，详情/编辑/二级页偏手工作坊式 |
| 可访问性 / 响应式 | 🟡 6/10 | 有 skip-link/aria 雏形，但暗色与对比度未系统校验 |

**Top 5 阻断性问题（P0）**
1. **设计令牌碎片化**：`DESIGN.colors.primary = '#0f172a'`（深海军蓝）与其他来源实际主色 `#3b82f6`（蓝）矛盾；圆角三套尺度（8/12/16 vs 2/4/8 vs 8/12/6）；存在死代码 `antd-theme.ts`。这是全站视觉不一致的根因。
2. **导航信息架构过载**：80 个菜单项平铺，且存在重复入口（如「服务目录 ▸ 服务请求」与顶层「服务请求」重复；SLA 拆成概览/仪表盘/监控 3 项）。
3. **面包屑功能失效**：`Header.pathToBreadcrumb` 仅硬编码 9 条路由，且 `MainLayout` 未传入 `breadcrumb` → 绝大多数深层页面（如 `/incidents/[id]/edit`）只显示「首页」，用户易迷失。
4. **错误态被「伪装成空态」**：列表页 `catch` 中 `setIncidents([])` + `message.error`，错误与无数据无法区分，运维/用户难以判断是「没数据」还是「系统挂了」。
5. **看板列颜色硬编码遗留蓝**（`'#1890ff'`，旧 antd 蓝），与全站新主色 `#3b82f6` 割裂，且未纳入令牌。

---

## 2. 跨模块基础问题（根因层）

### 2.1 设计令牌碎片化 ✅ `P0`
- 令牌来源 ≥ 6 处：`tailwind.config.mjs`、`src/lib/design-system/colors.ts`、`src/design-system/tokens/index.ts`、`src/lib/design-system/spacing.ts`、`src/styles/theme-variables.css` + 多份 `*.module.css`、`antd-theme.ts`（死代码）。
- **主色冲突**：`DESIGN.colors.primary = '#0f172a'`（海军蓝，语义错位），而 `tailwind primary`、根布局 `meta theme-color=#1890ff`、全站实主色均为蓝色系 →「primary」命名与视觉主色不一致，易引发开发误用。
- **圆角三套尺度**：DESIGN tokens `8/12/16`；`design-system/spacing.ts` `2/4/8`；`antdTheme` `8/12/6`。
- **背景色冲突**：`(main)/layout.tsx` 内联 `bg-[#f5f7fb]`，antd `colorBgLayout='#f8fafc'`，DESIGN `bgSubtle='#f8fafc'`。
- **字体声明失效**：令牌写 `Inter, Noto Sans SC`，但根布局注释「禁用 Google Fonts、改用系统字体」→ Inter 实际回退，标题层级视觉与设计意图不符。
- **样式机制混用**：同代码库并存 内联 `style`、Tailwind 类、`CSS Modules`、antd 令牌覆盖，无统一约束。
- **建议**：收敛到单一 token 源（推荐 `src/design-system/tokens` + antd `theme.algorithm`），删除 `antd-theme.ts` 死代码；用一次 lint/CI 规则禁止内联硬编码颜色，强制走令牌变量。

### 2.2 状态呈现不一致 ✅ `P0/P1`
- 存在精心设计的共享组件 `src/components/ui/LoadingEmptyError.tsx`（状态机 `loading|empty|error|success`，并按模块预置文案：`Tickets/Incidents/Problems/Changes/CMDB/Users/WorkflowsLoadingEmptyError`）。
- **采用率极低**：仅 **5/204** 个页面文件引入它（`workflow/*`、`templates`、`TicketTemplate`、`WorkflowEngine`）。其余页面用原始 `Spin` + `message.error` + 错误时清空数组。
- 个别详情页（如 `incidents/[id]/page.tsx`）**完全没有统一 loading/empty 态**，依赖子组件各自处理，体验参差。
- **建议**：将 `LoadingEmptyError` 强制定为列表/详情页标准骨架（可封装进 `BusinessPageTemplate`）；为错误态提供「重试」动作与可复制的错误码，区分「无数据」与「加载失败」。

### 2.3 数据获取模式分散 🔍 `P1`
- React Query / SWR 已装，但 `src/app` 下仅 **1** 文件引用 `useQuery/useSWR`，根布局虽挂 `QueryProvider` 实为摆设。
- 203 个页面走 `'use client'` + 手动 `useState/useEffect/try-catch` 调自建 `HttpClient` 类。导致：缓存缺失（切换页签重复请求）、失效逻辑重复、错误恢复脆弱、loading 态写法不统一。
- **建议**：在 `BusinessPageTemplate` 与共享 hooks 中接入 React Query，统一 `useList/useDetail` 封装；保留自写 `http-client` 仅作传输层。

### 2.4 信息架构 / 导航过载 ✅ `P0`
- `menu-config.ts` 共 **80 个 `key:` 菜单项**，顶层分组十余个（服务台/服务请求/我的请求/事件/问题/变更/知识库/服务目录/CMDB/资产/SLA/…），横向扫描认知负担重。
- **重复与歧义入口**：
  - 「服务目录 ▸ 服务请求」与顶层「服务请求」指向同一业务，命名重叠易混淆。
  - SLA 拆成「SLA概览 / SLA仪表盘 / SLA监控」三项，用户难辨差别。
  - 「服务请求列表」实际 `path` 指回 `/service-requests`，与子项「服务请求」重复。
- 菜单优先取 API `getUserMenus()`，失败回退静态 `getMenuConfig()`——两套来源可能不一致。
- **建议**：按 ITIL 4 实践域收敛为 ≤ 8 个一级分组；合并 SLA 三入口为「SLA 管理（含概览/监控 Tab）」；删除服务目录下的「服务请求」冗余子项；对菜单项做用户角色/使用频度驱动的「常用/全部」分级。

### 2.5 面包屑失效 ✅ `P1`
- `Header.pathToBreadcrumb` 仅硬编码 9 条一级路由（dashboard/tickets/incidents/problems/changes/knowledge/service-catalog/profile/notifications）。
- `MainLayout` 未向 `Header` 传 `breadcrumb`，`showBreadcrumb` 默认 `false` → 深层页（详情、编辑、二级列表）面包屑缺失或仅「首页」。
- **建议**：用 `usePathname()` 自动生成面包屑（基于路由段→菜单 label 映射），移除硬编码表；确保 `MainLayout` 始终开启 `showBreadcrumb`。

### 2.6 代码卫生 🔍 `P2`
- 仓库提交了 `.bak` 文件：`incidents/page.tsx.bak`、`changes/page.tsx.bak`、`problems/page.tsx.bak`、`workflow/instances/page.tsx.bak` → 应清理。
- `Sidebar.tsx`/`Header.tsx` 仅是对 `./sidebar`、`./header` 的再导出，且 `layout/Sidebar.module.css` 与 `layout/sidebar/Sidebar.module.css` 并存 → 重复与维护混乱。
- 这些不影响运行时 UX，但拖慢迭代、易引发「改了不生效」的隐性 bug。

---

## 3. 模块逐项审计

> 严重度：🔴 P0（阻断/高可见）、🟠 P1（明显可用性问题）、🟡 P2（优化项）

### 3.1 服务台 / Dashboard ✅
- **路由**：`/dashboard`，含 9 个图表组件（KPI 卡 + TicketTrend / IncidentDistribution / SLACompliance / UserSatisfaction / ResponseTime / TeamWorkload / PeakHours / ChartsSection / QuickActions）。
- **优点**：图表 `dynamic(ssr:false)` 懒加载（性能）、有自动刷新开关 + 实时连接指示（`isConnected`）+ 刷新间隔设置（SLA 监控友好）。
- **问题**：
  - 🔴 **信息过载**：9 张图 + KPI + 快捷操作同屏，长滚动、无「角色视角/个性化」切换（运维 vs 经理看到同一屏）。
  - 🟠 缺少空态/首访引导：新租户无任何数据时图表区可能一片空白无引导。
  - 🟠 无「可拖拽/可折叠」面板，重要指标无法置顶。
- **建议**：提供视角切换（个人待办 / 团队 / 全局）；支持卡片折叠与布局记忆；首访空态引导创建第一个工单/配置 SLA。

### 3.2 工单 Tickets 🔍
- **路由**：`/tickets`、`/tickets/create`、`/tickets/dashboard`、`/tickets/analytics`、`/tickets/cc`、`/tickets/types`、`/tickets/templates`、详情页。
- **问题**：
  - 🟠 入口分散：`dashboard`、`analytics`、`cc`（抄送？）、`types`、`templates` 与列表并列，认知成本高；`/tickets/cc` 命名语义不清（用户难猜「cc」含义）。
  - 🟠 列表/详情是否统一用 `BusinessPageTemplate` 与共享状态组件未核实（推断不一致）。
  - 🟡 模板与类型管理藏在工单域下，权限/分类逻辑应与 Admin 配置打通。
- **建议**：将 `analytics` 收口到「报表中心」；`cc` 改名「抄送我的」并在菜单显示中文；统一列表/详情骨架。

### 3.3 事件 Incidents ✅
- **路由**：列表/新建/创建/详情 `/[id]`/编辑 `/[id]/edit`，含 `IncidentList`、`IncidentFilters`、`IncidentStats`、`UnifiedKanbanBoard`、`BatchActionBar`。
- **列表页（较好）**：用 `BusinessPageTemplate` + 统计 + 筛选 + 列表/看板切换 + 批量操作 + 分页。
- **问题**：
  - 🔴 **错误态伪装空态**：`catch` 中 `setIncidents([])` + `message.error`，见 2.2。
  - 🔴 **看板列颜色硬编码** `'#1890ff'`（遗留蓝），与全站 `#3b82f6` 不符，且未做租户/优先级语义映射（见 2.1）。
  - 🟠 **详情页手工作坊式**：`incidents/[id]/page.tsx` 用内联 `style={{ padding: 24 }}`、「返回列表」链接 `color:'#666'` 硬编码、外层 `Card` 才包 Tabs，无统一 loading/empty。
  - 🟠 编辑页校验仅必填（`rules required`），缺业务规则（如「已解决」必须填解决方案/根因）。
- **建议**：详情/编辑页套用统一 `DetailTemplate`；看板颜色走令牌 + 状态语义色板；错误态用共享组件区分。

### 3.4 问题 Problems 🔍
- **路由**：`/problems`、`/problems/new`、`/[id]/edit`、`/problems/known-errors`（已知错误）、`/problems/trends`。
- **问题**：
  - 🟠 与事件高度同构却各自实现列表组件（`ProblemList` vs `IncidentList`），无共享 `IssueList` 基类 → 行为/样式易漂移（已知错误的「已知错误库」与问题列表区分度不足）。
  - 🟠 `/problems/trends` 趋势页与「报表中心」可能重复。
  - 🟡 已知错误→变更/知识库的关联入口不明显（ITIL 闭环弱）。
- **建议**：抽取事件/问题/变更的共享列表/详情骨架；已知错误页强化「升变变更/沉淀知识」的一键动作。

### 3.5 变更 Changes 🔍
- **路由**：`/changes`、`/changes/new`、`/[id]`、`/[id]/edit`、`/[id]/pir`、`/changes/pirs`、`/standard-changes`。
- **问题**：
  - 🟠 PIR（实施后评审）散落 `/changes/[id]/pir` 与 `/changes/pirs` 两处，入口分裂。
  - 🟠 标准变更 `/standard-changes` 与普通变更审批流差异在 UI 上未突出引导。
  - 🟡 审批链可视化（谁审、卡在谁）在列表/详情是否呈现未核实。
- **建议**：合并 PIR 入口；标准变更提供「模板化建单」引导；详情页强化审批时间线（结合工作流）。

### 3.6 发布 / 服务请求 / 服务目录 🔍
- **发布 Releases**：`/releases`、`/releases/new`、`/[id]`、`/[id]/edit`。🟠 与变更的生命周期衔接（变更→发布）在 UI 上缺显式过渡，用户难建立因果。
- **服务请求 SR**：`/service-requests`、`/my-requests`。🟠 与「服务目录」的请求流程重叠，终端用户视角（我的请求）与坐席视角（服务请求）分两个一级菜单，可合并为「服务目录 + 我的请求」。
- **服务目录**：`/service-catalog`、`/detail/[id]`、`/edit/[id]`、`/request/[id]`、`/approvals`。🟠 子项「服务请求」与顶层「服务请求」重复（见 2.4）；目录项「可订阅/不可订阅」「需审批/免审批」状态标识不清。

### 3.7 知识库 KB 🔍
- **路由**：`/knowledge`、`/articles/[id]`、`/articles/new`、`/reviews`（审核）。
- **问题**：
  - 🟠 文章阅读页缺乏「有用/无用」反馈与「关联工单/问题」反链，RAG 检索结果到人工沉淀的闭环弱。
  - 🟠 `/reviews` 审核流与文章编辑的权限/状态机未核实一致性。
  - 🟡 富文本编辑器的 XSS 消毒（后端 `common/sanitizer.go`）前端是否预校验未核实。
- **建议**：阅读页加反馈与反链；审核态用状态徽章统一呈现。

### 3.8 CMDB 🔍
- **路由**：`/cmdb` + `ci`/`ci-types`/`cis/[id]`/`cis/create`/`cis/[id]/edit`/`cloud-accounts`/`cloud-resources`/`cloud-services`/`reconciliation`/`registry`/`relationships`/`topology`。
- **问题**：
  - 🔴 **模块过重**：CMDB 下挂 10+ 子页（含云资源/云账号/对账/拓扑），与「资产管理」「许可证」概念重叠，信息架构臃肿。
  - 🟠 拓扑图 `/topology` 通常用 `reactflow`，节点密度/性能/缩放交互未核实（大数据量下易卡顿）。
  - 🟠 CI 关系（`relationships`）与拓扑图功能重叠，用户分不清。
- **建议**：将「云资源/云账号/对账」归到「云管理」分组；关系管理与拓扑明确分工（列表编辑 vs 可视化浏览）。

### 3.9 SLA 🔍
- **路由**：`/sla`、`/sla/definitions/[id]`、`/sla-dashboard`、`/sla-monitor`。
- **问题**：🟠 三入口（概览/仪表盘/监控）语义重叠（见 2.4）；监控页实时性依赖轮询还是 WS 未核实，若无推送则违反 SLA 秒级感知预期。
- **建议**：合并为「SLA 管理」单入口 + 内部 Tab（定义/仪表盘/实时监控）。

### 3.10 用户 / 角色 / 权限（Admin）🔍
- **路由**：`/admin/users`、`/admin/roles`、`/admin/permissions`、`/admin/groups`、`/admin/teams`、`/admin/departments`、`/admin/tenants`，外加 `/system/users`、`/enterprise/departments`、`/enterprise/teams`。
- **问题**：
  - 🔴 **路由分裂**：用户管理同时存在于 `/admin/users` 与 `/system/users`；部门/团队同时存在于 `/admin/*` 与 `/enterprise/*` → 同一概念多入口，权限与数据易不一致。
  - 🟠 角色/权限/菜单的关系（RBAC 可视化）未提供「权限矩阵」视图，运维难排查「某人能否看某菜单」。
- **建议**：统一用户/部门/团队到单一 Admin 域，删除 `system/enterprise` 重复入口；提供「权限矩阵」可视化与「模拟某用户视角」调试工具。

### 3.11 工作流 / BPMN 🔍
- **路由**：`/workflow`（总览/分析）、`/workflow/designer`、`/workflow/instances`、`/workflow/audit`、`/workflow/automation`、`/workflow/bottlenecks`、`/workflow/versions`、`/workflow/sla`、`/workflow/ticket-approval`，外加 `/workflows`、`/admin/workflows`。
- **问题**：
  - 🔴 **命名混乱**：`/workflow` 与 `/workflows` 并存；`/workflow/ticket-approval` 与「审批」模块（`/approvals`、`/admin/approval-chains`）概念交叉。
  - 🟠 BPMN 设计器（`bpmn-js`）画布在无流程时的空态引导、保存冲突提示未核实。
  - 🟢 该模块是少数**规范使用** `LoadingEmptyError` 与共享状态组件的（值得作为标杆推广）。
- **建议**：统一 `/workflow` 单复数；设计器与审批链 UI 明确边界（设计 vs 运行实例 vs 待办）。

### 3.12 报表 / 分析 🔍
- **路由**：`/reports` + 12 个细分页（change-success、changes、cmdb-quality、incident-trends、incidents、problem-efficiency、problems、service-catalog-usage、sla-performance、sla、tickets）。
- **问题**：
  - 🟠 报表页与 Dashboard 图表、各模块自带 `analytics`/`trends` 高度重叠（如 `incident-trends` 与 Dashboard 的 IncidentDistribution），用户不知去哪看。
  - 🟠 是否支持导出（PDF/Excel）、时间范围预设、图表钻取未核实。
- **建议**：以「报表中心」为唯一分析出口，各模块内 `analytics` 仅保留轻量概览并链接到报表中心。

### 3.13 系统 / 设置 🔍
- **路由**：`/admin`（overview）、`/admin/system-config`、`/admin/config-inheritance`、`/admin/connectors`、`/admin/escalation-matrices`、`/admin/escalation-rules`、`/admin/process-routing`、`/admin/department-processes`、`/admin/service-catalogs`、`/admin/sla-definitions`、`/admin/sla-templates`、`/admin/ticket-categories`、`/admin/tickets/assignment-rules`、`/admin/tickets/automation-rules`、`/admin/menus`、`/admin/cmdb-types`、`/system/organization`。
- **问题**：🟠 配置项极多且扁平，缺乏「分级/搜索/常用」；`config-inheritance`（配置继承）概念重，无可视化解释易误配。
- **建议**：设置域引入左侧分组导航 + 顶部搜索 + 「危险操作」二次确认；关键配置提供「影响预览」。

### 3.14 AI 功能 🔍
- **路由**：`/ai/chat`（AI 对话）、`/tickets/ai-create`（AI 建单）。
- **问题**：
  - 🟠 AI 建单的「建议字段→确认」闭环、AI 对话的「引用来源/可溯源」在 UI 上是否清晰未核实（RAG 必须可见来源，否则信任危机）。
  - 🟠 生成内容是否标注「AI 生成」、是否提供「采纳/拒绝」未核实 → 合规风险。
  - 🟡 A2UI（`a2ui-api.ts`、`src/components/a2ui/`）动态 UI 的加载/错误态是否统一未核实。
- **建议**：所有 AI 输出显式标注来源与「AI 生成」标识；提供采纳/编辑/拒绝动作并落审计。

### 3.15 登录 / 认证 ✅
- **路由**：`/login`、`/forgot-password`、`/reset-password`、`/register`、`/sso/callback`、`/onboarding/wizard`。
- **问题**：
  - 🟠 登录页仍引用**死代码 `antd-theme.ts`**（与全站 `getAntdTheme` 主题可能不一致）→ 登录态视觉与后台割裂。
  - 🟡 注册/SSO/Onboarding 流程的引导步骤与错误提示一致性未核实。
- **建议**：登录与后台统一走同一主题源；Onboarding 提供进度指示与权限申请引导。

---

## 4. 优先级路线图

### P0（2–3 周，先做地基）
1. **收敛设计令牌**：删除 `antd-theme.ts`，统一到 `design-system/tokens` + antd 主题算法；全站替换硬编码颜色/圆角/背景为令牌变量（可用 codemod/正则扫描）。
2. **统一状态骨架**：将 `LoadingEmptyError` 嵌入 `BusinessPageTemplate` 与 `DetailTemplate`，强制列表/详情页使用；错误态区分「无数据 / 加载失败」并提供重试。
3. **修复导航与面包屑**：菜单收敛 ≤ 8 一级分组、合并 SLA/服务目录重复入口；面包屑改为基于 `usePathname` 自动生成并在 `MainLayout` 常开。
4. **修复错误态伪装**：消除 `setXxx([])` 的 catch 写法，统一抛给共享状态组件。

### P1（3–5 周，体验一致性）
5. 接入 React Query 统一 `useList/useDetail`，替换手写 `useEffect` 获取。
6. 抽取事件/问题/变更共享列表/详情/编辑骨架，消除同构重复实现。
7. 统一 Admin 域（删 `system/enterprise` 重复入口），增加「权限矩阵」可视化。
8. 工作流路由单复数归一；CMDB 云资源归并到「云管理」。

### P2（持续，打磨）
9. 清理 `.bak`、重复 `Sidebar/Header` 再导出与双 `*.module.css`。
10. Dashboard 视角切换 + 布局记忆 + 首访空态引导。
11. AI 输出溯源标注与采纳/拒绝闭环。
12. 报表中心整合各模块 `analytics`，消除重叠。

---

## 5. 量化基线（建议持续追踪）
| 指标 | 现状 | 目标 |
|---|---|---|
| 设计令牌来源数 | 6+ | 1（单一 token 源） |
| 共享状态组件采用率 | 5 / 204 (2.5%) | ≥ 90% |
| React Query 采用率 | 1 / 204 (0.5%) | ≥ 80% |
| 一级导航分组数 | 10+ | ≤ 8 |
| 面包屑覆盖路由 | 9 硬编码 | 全路由自动 |
| 重复/遗留文件 | 4 .bak + 双导出 | 0 |
