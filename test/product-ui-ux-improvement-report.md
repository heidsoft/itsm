# 产品改进建议与深度测试分析（UI交互 / 功能设计 / 界面布局 / 完整性）

- 日期: 2025-12-07
- 范围: itsm-prototype（前端）与 itsm-backend（后端）
- 目标: 基于测试与代码审查，提出可落地的产品改进建议，优先级与验收标准明确

## 结论与优先级
- P0（立即）
  - 恢复前端 ESLint/TS 测试能力；修复测试路径与 JSX 类型错误，保证基础质量门槛
  - 对齐后端 DTO/控制器构造与 BPMN 适配器版本，保证 `go test` 编译通过
- P1（短期）
  - 统一 UI 设计系统：明确以 Tailwind 为主，逐步减少 Ant Design 或统一主题
  - 统一 API 调用与本地存储访问：移除组件内直接 `fetch`/`localStorage`，改用统一封装
- P2（中期）
  - 无障碍（A11y）与可用性完善：键盘导航、ARIA、焦点管理与屏幕阅读优化
  - 产品完整性与一致性：统一空态/加载/错误模式，统一列表过滤与分页交互

## 测试与代码依据（精选）
- Toast 自动关闭与交互：`itsm-prototype/src/components/ui/Toast.tsx:36` 定时触发 `handleClose()`；按钮关闭 `114` 行
- 组件直接调用 API 与本地存储：`itsm-prototype/src/components/business/IncidentManagement.tsx:162-168`
- 类型与测试问题：
  - JSX 解析错误：`itsm-prototype/src/lib/hooks/useAccessibility.ts:73`
  - 测试路径错误：`itsm-prototype/src/app/components/__tests__/TicketCard.test.tsx` 无法解析真实组件路径
  - 后端编译失败：`itsm-backend/pkg/bpmn/engine_adapter.go:16,43,51,57,62`
  - 控制器构造/DTO 不匹配：`itsm-backend/controller/user_controller_test.go:40,435,444,445,454,627,685`

## UI交互（Interaction）改进建议
- 表单与输入
  - 为所有输入控件提供明确标签与描述；测试选择器用 `getByRole('textbox', { name: '密码' })` 或 `getByLabelText('密码', { selector: 'input' })`
  - 聚焦管理：表单提交失败时将焦点返回至首个有错误的输入框；屏幕阅读器播报错误摘要
- Toast/通知
  - 提供“无障碍友好模式”（减少或禁用动画）与可控持续时间；支持 `aria-live` 优先级配置
  - 批量通知分组与去重，避免噪音；提供用户可配置的持久化与关闭策略
- 列表与筛选
  - 统一筛选组件（状态/优先级/搜索/分页），在所有列表页复用；支持 URL 同步（可分享链接）
  - 加载与空态统一模板：含操作建议与快速入口（创建/刷新/清空筛选）
- 键盘与快捷键
  - 支持通用快捷键：`/` 聚焦搜索、`j/k` 列表上下导航、`Enter` 打开详情、`Esc` 关闭弹窗
  - 为对话框与抽屉统一焦点陷阱与 `Esc` 关闭逻辑

## 功能设计（Feature Design）改进建议
- 工单与事件流程
  - 标准化状态机：`新建 → 待处理 → 处理中 → 已解决 → 已关闭`，明确可转移路径与必要字段
  - 审批与 SLA 集成：在工单详情统一展示审批进度、SLA 剩余时间与违规风险提示
- 智能分配与规则
  - 在列表页提供“智能推荐”操作；支持批量分配模拟与效果预估
  - 规则管理页提供调试器视图与规则变更审计
- 依赖与影响分析
  - 在工单详情展示关联 CI/工单关系图；支持影响范围预估与变更窗口建议
- 报表与可视化
  - 为每类报表统一过滤与时间范围；支持导出 CSV/PNG；提供“保存为仪表盘卡片”与分享链接

## 界面布局（Layout）改进建议
- 设计系统统一
  - 明确以 Tailwind 为主；减少 Ant Design 或统一其主题变量与间距/色彩/字体
  - 移除 CSS Modules：当前存在 `Header.module.css`, `Sidebar.module.css`, `error-handler.module.css`
- 页面结构与网格
  - 统一 `PageHeader`（标题/面包屑/操作），主体区采用响应式网格（2/3 主内容 + 1/3 侧栏）
  - 内容密度可调（舒适/紧凑）；移动端采用单列布局并使用底部导航
- 空态/加载/错误统一组件
  - 统一在所有页面使用 `LoadingEmptyError` 模板，含操作建议/重试按钮/调试信息折叠

## 产品完整性与一致性（Integrity）
- 统一 API 层与环境配置
  - 所有请求通过 `src/lib/api/http-client.ts` 与 API 模块；禁止组件内直接 `fetch` 与 `localStorage`
  - 使用环境变量 `NEXT_PUBLIC_API_URL`、版本化路径 `/api/v1/*`
- 身份与租户
  - 统一通过封装的 Auth/Tenant 服务管理用户信息与租户切换；页面守卫在 App Router 布局层实现
- 观察性与可运维
  - 前端错误捕获与上报（Sentry 可选）；后端结构化日志与 Trace
  - 统一操作审计（关键域：工单、审批、规则、SLA）

## 可落地的改造清单（验收标准）
- 质量门槛（P0）
  - 安装 ESLint TS 依赖，`next lint` 与 `tsc --noEmit` 均通过
  - 修复测试路径与 JSX 类型；`npm run test:ci` 套件通过率 ≥ 80%
- 规范一致性（P1）
  - 将含 JSX 的 `.ts` 文件改为 `.tsx`；移除 `...` 占位符
  - 移除 CSS Modules（或说明例外），统一 Tailwind
  - 移除组件内 `fetch/localStorage`，改为统一 API 层与 Auth/Tenant 服务
- 交互一致性（P1）
  - 统一筛选与分页组件在 Tickets/Incidents/Problems/Changes 页复用；支持 URL 同步
  - 统一 Loading/Empty/Error 组件在所有列表与详情页使用
- A11y 与可用性（P2）
  - 键盘导航与焦点管理覆盖主要页面与对话框；Axe 扫描通过率 ≥ 95%

## 风险与依赖
- UI 系统统一可能影响现有 Ant Design 部分组件，需要评估替换成本与主题化策略
- DTO/服务签名调整需同步前端类型，避免联调破坏；建议新增版本化路径或向后兼容字段映射

## 推荐实施顺序
1. 修复 ESLint/TS 与测试路径（确保持续集成可运行）
2. 统一 API 层与存储封装，清理组件内直接调用
3. 引入统一 Loading/Empty/Error 与筛选分页组件
4. 明确设计系统（Tailwind 主导），制定迁移策略与主题规范
5. 增强 A11y 与可用性，完善产品一致性

