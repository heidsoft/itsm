# 项目测试回顾与产品问题分析（全面版）

- 日期: 2025-12-07
- 范围: itsm-prototype（Next.js 前端）与 itsm-backend（Go 后端）
- 目标: 全面回顾测试现状，逐模块排查与分析产品问题，形成优先级明确的修复与测试计划

## 概览

- 基线恢复：前端 `lint/type-check/integration tests` 已可运行，API 集成测试 19/19 通过。
- 阻塞集中在后端单元测试：测试用 Ent 字段与现有 Schema 不一致导致编译失败。
- 产品一致性与规范方面仍需整治：统一设计系统、API 层封装、无障碍与交互统一、错误与空态模板复用等。

## 当前测试状态

- 前端
  - Lint：已恢复运行（简化 `.eslintrc.json`）。
  - TypeScript：已通过（修复 `validation.ts` 泛型与 `TicketDependencyManager.tsx`）。
  - 集成测试：`src/app/lib/__tests__/api-integration.test.ts` 19/19 通过；断言与错误文案对齐。
- 后端
  - `go test ./...`：编译失败，主要因为测试代码使用了不存在的 Ent setter 与 DTO 字段（例如 `SetActive`, `SetDisplayName`, `SetDescription`）。

## 模块排查汇总

- 认证与租户
  - 端点：`itsm-backend/router/router.go:76-77, 88-95`
  - 控制器：`itsm-backend/controller/auth_controller.go:21-115`
  - 前端页面：`itsm-prototype/src/app/(auth)/login/page.tsx`
  - 问题与测试关注
    - 登录失败的焦点返回与错误播报；Token 续期与租户切换头部参数；跨租户数据隔离
- RBAC 与权限
  - 中间件：`itsm-backend/router/router.go:91` 权限保护；各路由 `RequirePermission` 使用
  - 问题与测试关注
    - 越权路径与角色矩阵覆盖；批量操作权限（导入/导出/分配/审批）
- 工单全链路
  - 路由：`itsm-backend/router/router.go:133-199, 201-210, 215-256`
  - 控制器：`itsm-backend/controller/ticket_controller.go:30-79`
  - 前端：`itsm-prototype/src/app/(main)/tickets/*`；业务组件 `src/components/business/tickets/*`
  - 问题与测试关注
    - 评论/附件/通知/评分/视图/模板/分类/标签/子任务/智能分配/导入导出一致性与合同；空态/错误提示统一
- 事件/问题/变更/审批与工作流
  - 事件：`itsm-backend/router/router.go:258-268`
  - 审批与工作流：`itsm-backend/router/router.go:221-230, 407-411`
  - 问题调查：`router/problem_investigation_router.go`
  - 问题与测试关注
    - 审批状态机一致性；BPMN 部署与实例操作；问题调查步骤与根因报告口径
- SLA
  - 路由：`itsm-backend/router/router.go:270-291`
  - 问题与测试关注
    - 合规计算时区与节假日；违规状态更新；预警规则与历史正确性
- 服务目录与请求
  - 前端：`itsm-prototype/src/app/(main)/service-catalog/*` 与 `forms/*`
  - 问题与测试关注
    - 多步表单校验、撤回与重试、审批与完成链路
- CMDB
  - 前端：`itsm-prototype/src/app/(main)/cmdb/*`、`RelationGraph.tsx`
  - 问题与测试关注
    - 大数据渲染与虚拟化；循环关系与层级深度处理；影响评估
- AI 模块
  - 路由：`itsm-backend/router/router.go:318-332`
  - 问题与测试关注
    - 工具执行权限、输入安全与审计记录；RAG/Embedding 稳定性与可解释性
- 仪表盘与报表
  - 前端：`itsm-prototype/src/app/(main)/dashboard/*`、`(main)/reports/*`
  - 问题与测试关注
    - 过滤与时间范围一致性；导出的格式与内容；卡片化与分享链接

## 关键产品问题（按领域）

- 设计与一致性
  - 双系统并存（Tailwind 与 Ant Design）增加维护成本；建议统一 Tailwind 或明确主题规范
  - CSS Modules 仍存在：`itsm-prototype/src/components/layout/Header.module.css`，`Sidebar.module.css`，`error-handler.module.css`
- API 层封装与存储访问
  - 组件内存在直接 `fetch/localStorage` 用法，违反统一 API 层与封装规范
  - 代码参考：`itsm-prototype/src/components/business/IncidentManagement.tsx:162-168`
- 无障碍与交互
  - 键盘导航与焦点管理需要统一（对话框焦点陷阱与 Esc 关闭）；屏幕阅读器播报统一化
  - 代码参考：`itsm-prototype/src/lib/hooks/useAccessibility.tsx:71-82, 96-136, 149-191, 209-230`
- 错误/空态模板与提示
  - 统一使用 `LoadingEmptyError`，在各模块复用；表单错误播报与焦点返回一致
- 后端测试与合同不一致
  - 测试引用的 Ent setter 与 DTO 字段不在当前 Schema 中；需要统一字段与更新测试
  - 失败示例：`itsm-backend/service/auth_service_test.go:36, 140, 225, 232, 307, 322, 383`；`itsm-backend/controller/user_controller_test.go:40, 631, 692`

## 建议的修复与测试计划

- P0（立即）
  - 后端：统一测试与 Ent/DTO 字段，移除或替换不存在的 setter（如 `SetActive`, `SetDisplayName`, `SetDescription`）。
  - 前端：移除组件内直接 `fetch/localStorage`，统一到 `src/lib/api/http-client.ts` 与封装的 Auth/Tenant 服务。
- P1（短期）
  - 统一筛选/分页组件并在 Tickets/Incidents/Problems/Changes 复用，支持 URL 同步；统一 Loading/Empty/Error 模板
  - 明确设计系统，以 Tailwind 为主并减少 Ant Design 或统一主题变量
  - 增加 RTL 测试覆盖认证与工单链路、空态/错误态与键盘无障碍
- P2（中期）
  - SLA 时区/节假日；CMDB 大数据与循环关系；AI 工具执行权限与审计；报表卡片化与分享

## 已实施改动摘要

- 前端
  - ESLint 配置简化，`next lint` 恢复运行：`itsm-prototype/.eslintrc.json`
  - TypeScript 修复：`itsm-prototype/src/lib/validation.ts` 泛型与占位符；`TicketDependencyManager.tsx` 语法修复
  - Jest alias 与导入修复：`itsm-prototype/jest.config.js`；`src/app/lib/__tests__/api-integration.test.ts`
  - 集成测试断言对齐中文错误文案（CORS）：`src/app/lib/__tests__/api-integration.test.ts:313`
- 后端
  - 复测 `go test`，定位失败源于测试字段与 Schema 不一致（详见失败引用）

## 下一步

- 修复后端测试：从认证与工单模块入手，对齐字段并补充 Table Driven Tests 与 `enttest.NewClient()` 构造。
- 前端统一 API 层调用：重构组件中直接调用与存储访问，首先处理 `IncidentManagement.tsx`。
- 推进统一交互与模板：在 Tickets/Incidents 模块植入统一筛选/分页与 Loading/Empty/Error 模板，并增加 A11y 测试。
