# CloudMesh 产品审计总报告

> Status: current
> 审计基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`（`main`，2026-11-21）
> 审计工作树：`../itsm-audit`，分支 `audit/production-tenant-readiness`
> 审计方式：七个并行只读审计通道（生产隔离、菜单路由、RBAC、初始化迁移、Release、UI 静态、官网）+ 本地真实 Postgres / bootstrap 验证
> 本次审计未修改生产数据库、未修改生产 Secret、未执行破坏性迁移、未运行 Ent Codegen。

## 1. 审计范围与证据规则

每个条目都满足以下四项之一：`confirmed`（已复现或代码证据充分）、`suspected`（存在风险但未验证）、`resolved`（已修复并通过验证）、`blocked`（缺环境/依赖）。

- 代码证据均给出 `file:line` 与函数/符号名；没有代码证据的视觉判断一律标 `suspected`。
- UI 部分为**静态检查完成，运行验证待执行**：当前工作树没有 `node_modules`、没有 `.next`，未启动 dev server，未运行 Playwright；不得据此宣称视觉问题已解决。
- 官网部分基于实际 DNS 解析与 webfetch 结果；`cloudmes.top` 不存在，规范域名是 `cloudmesh.top`（`CNAME:1`、`website/CNAME:1`、`brand.ts:23`）。

## 2. 结论摘要

| 级别 | 数量 | 性质 |
|:---|:---:|:---|
| P0 | 3 | 1 项已有独立修复 PR（未合并）；1 项阻断 fresh-install 门禁；1 项破坏发布说明可信度 |
| P1 | 15 | 权限判定语义、菜单与 API 权限不一致、前端质量门禁红、发布 provenance、官网可信信息 |
| P2 | 20+ | 一致性、文档漂移、可维护性与体验细节 |

**没有发现已证实的跨租户越权漏洞。** 应用层租户解析、RBAC 每请求重载、跨租户负例测试矩阵与 RLS readiness 门禁均有实证（见 `deployment-audit.md` §4）。`RLS_MODE=off` 是已记录的分阶段设计，不是可直接断言的 P0。

## 3. P0 整改清单

| ID | 标题 | 状态 | 证据锚点 | 修复归属 |
|:---|:---|:---|:---|:---|
| AUDIT-001 | 空卷首装被 `mcp:read` 迁移打断 | `resolved`（PR #114，基准 commit 上仍 open） | `itsm-backend/migrations/20261118_mcp_read_permission.sql:80-83`；fresh-install-gate run 44871855241 | PR #114 `fix/fresh-install-mcp-seed`（commit `284d4d27`），CI 已见 `itsm-init 退出 0` |
| AUDIT-002 | `configuration_items` 重复索引阻断 fresh-install 门禁 | `confirmed` | `migrations/20261119_ci_owner_fk_and_certification.sql:76-77` vs `ent/schema/configurationitem.go:225-226`；`database/database.go:161-195` 收敛表漏两项 | 需人工审批后修复（涉及数据库索引收敛） |
| AUDIT-003 | Release 缺 CHANGELOG 条目时静默生成 git-log 发布说明，且无测试证据/制品校验 | `confirmed` | `.github/workflows/release.yml:185-202`；`scripts/docs-gate/check-release-claims.sh` 未接入 release workflow | PR-04 `audit/release-quality-gate` |

### AUDIT-001 复现与验证

复现：空库执行 `ITSM_BOOTSTRAP_ONLY=true` → `pq: mcp:read 权限码插入失败（tenant_id=1） (P0001)`，初始化链中断。
根因：`internal/bootstrap/app.go` 先跑迁移（1651）后跑 seeder（1687），空卷上 `tenants` 为空，Step 1 插入 0 行，Step 2 按硬编码 `tenant_id=1` 判定并抛异常。
验证：真实 Postgres 复现原错误；修复后 3 条回归 + 5 条 checksum 门禁测试全绿；真实 bootstrap 对空库退出码 0；CI gate 日志 `itsm-init 退出 0`。
剩余风险：既有安装每次 bootstrap 多一条 checksum supersede 告警；是否自动回写账本 checksum 需评审。

### AUDIT-002 复现与建议

复现：`scripts/fresh-install-gate.sh` 在空卷上报 `首装库存在 2 对重复索引`。
根因：版本化 SQL 与 Ent Schema 两个 DDL 来源创建同列索引，`EnsureIndexConvergence` 的收敛清单没有收录这两对；静态守卫 `ent/migrate/index_duplicate_guard_test.go` 只看 Ent 一侧。
最小修复：在 `database/database.go:161-195` 的 `duplicateIndexPairs` 增加两对（保留 Ent 生成的名字，删除手写名）。
回滚：撤销该收敛项后，已删的手写索引不会自动重建（迁移已入账），需要新的版本化迁移；Ent 索引不受影响。
**属于数据库结构改动，按规范必须人工审批后执行。**

## 4. P1 整改清单（按用户影响排序）

| ID | 标题 | 状态 | 主要证据 | 归属 |
|:---|:---|:---|:---|:---|
| AUDIT-010 | `/tickets/types` 菜单与路由权限码不一致，服务目录管理员可见但列表 403 | `confirmed` | `pkg/menubaseline/baseline.go:86`、`router/ticket_type_routes.go:12`、`rbac_precheck_gen.go:458` | PR-02 |
| AUDIT-011 | `/service-requests`、`/my-requests` 菜单用 `ticket:read`，API 用 `service_request:read` | `confirmed` | `baseline.go:28,30`、`rbac_precheck_gen.go:407,413` | PR-02 |
| AUDIT-012 | `/workflow/audit` 菜单用 `audit:read`，数据源用 `bpmn:read` | `confirmed` | `baseline.go:201`、`rbac_precheck_gen.go:141` | PR-02 |
| AUDIT-013 | 权限停用（`IsActive=false`）只在 AI 判定链生效，人工 RBAC 路径未解码信封 | `confirmed`（代码路径） | `internal/authz/policy.go:131-135`、`middleware/rbac.go:847-873` | RBAC 专项，需审批 |
| AUDIT-014 | DBOnly 下 L3 URL 推断仍回退到编译期角色默认值 | `confirmed`（代码路径）/ `suspected`（可达性） | `middleware/smart_permission.go:355-386,407-423` | RBAC 专项，需审批 |
| AUDIT-015 | Endpoint ACL 缓存无失效调用点；Policy/ACL 缓存无跨实例广播 | `confirmed` | `smart_permission.go:82-96`、`authz/registry.go:122-134` | RBAC 专项 |
| AUDIT-020 | `npm run lint:antd`、`check-antd-direction.sh`、P0 文案守卫在基准 commit 上均为红 | `confirmed`（脚本实跑 exit 1） | `tools/check-antd-legacy.sh`、`scripts/check-frontend-p0-guards.sh` | PR-05（低风险 mechanical 项可提前） |
| AUDIT-021 | Sidebar 两套 CSS Module，实际渲染菜单的暗色覆盖规则为死代码 | `confirmed`（代码）/ `suspected`（视觉） | `Sidebar.tsx:14` vs `MenuItems.tsx:10` | PR-05，需浏览器验证 |
| AUDIT-022 | 禁用态 Select 选项在亮/暗两套主题中与可用态同色 | `confirmed`（层叠优先级）/ `suspected`（视觉） | `globals.css:101-104,1945-1955,309-319` | PR-05，需浏览器验证 |
| AUDIT-030 | 最近 22 个 tag 无 CHANGELOG 版本段落，发布说明走自动生成分支 | `confirmed` | `CHANGELOG.md:469`；`git describe` = `v1.6.59` | PR-04 |
| AUDIT-031 | `workflow_dispatch` 重发混源：二进制来自分支、镜像来自 tag | `confirmed` | `release.yml:43-44,74-75,236-239` | PR-04 |
| AUDIT-032 | 至少 7 处版本真源互相矛盾 | `confirmed` | `.env.prod.example:13`、`router.go:138`、`package.json:3`、`ga_readiness.go:78` 等 | PR-04 |
| AUDIT-040 | 官网英文页无来源的安装量数字；trust 页指向其他产品 | `confirmed`（实测网页） | `index.en.html`、`trust.html`；仓库零匹配 | 官网整改（外部站点） |
| AUDIT-041 | 仓库版 `website/` 与线上站点不一致，演示按钮自链接 | `confirmed` | `website/index.html:2-6,299` | 官网整改 |
| AUDIT-042 | 隐私政策入口缺失、条款页 404、免费试用与付费 PoC 文案冲突 | `confirmed` | live `/privacy/`、`/terms.html` 404、`/contact/`、`/pricing.html` | 官网整改 |

## 5. P2 整改清单（摘要）

完整条目与证据见各专项报告。代表性条目：

- `.env.prod.example:148` `METRICS_REQUIRE_AUTH=false` 与 compose 默认 `true` 冲突（PR-01 已修）。
- `config/seed/default.json` 业务数组为空，演示数据需显式 opt-in（正例，无需整改）。
- `ent/schema/permission_definition.go` 为死模型，且唯一键不含租户；`permissions` / `roles` 缺 `(tenant_id, code)` DB 唯一约束。
- RLS 默认 `off`、策略仅覆盖两张表；必须保留 readiness 门禁，不得直接翻 `enforce`。
- 备份依赖宿主定时脚本，compose 只提供只读挂载；缺少部署后验收断言。
- UI token 三套并存（`theme-variables.css` / `globals.css` / `layout.config.ts`），z-index 三套并存；表格与弹窗 `overflow` 规则互相打架；`LoadingSkeleton` 渲染期 `Math.random()`。
- `docs/product/frontend-menu-and-features.md` 仍写 102 条菜单与已退役入口，与 101 条基线漂移。
- 官网品牌 fragmentation（CloudMesh / AI-Native ITSM / OpenClaw / ClawOps）。

## 6. 已确认的历史整改（禁止重复实施）

- 跨租户通知/附件/审计/技能/变更 CI 引用等专项负例测试已存在并通过。
- RBAC 缓存中毒、DBOnly 缺角色 fail-open、角色移除即时失效（token revocation）、RLS 假安全感门禁均已落地。
- 权限策略信封在重复初始化时跳过管理员修改；角色授权只增不减，仅显式退役清单可收缩。
- 菜单退役（工单仪表盘、AI 中心、升级矩阵、审批中心去重）、`bpmn:*` 对齐、图标一致性守卫已落地。
- AntD v6 迁移主体已完成；当前红灯是**回归**，不是未完成迁移。

## 7. 验证边界

- 本次审计没有执行 `go test ./...`（仓库体量大且存在基线红灯），执行的是各专项的定点检查与真实 bootstrap 端到端验证。
- 前端未执行 `npm install`、`tsc`、Jest、Playwright；所有 UI 结论都是静态证据， vision 验证待执行。
- 官网结论来自本次实际 DNS / webfetch，未对站点做任何修改。
- 生产数据库未被连接或修改；所有 DB 验证在隔离的 `itsm_init_test_*` 数据库中进行，测试后已清理。

## 2026-11-28 企业质量治理首轮

本节以 `021f08c0e` 为源码基准，保留上文的历史证据。下表从 App Router、菜单基线 `pkg/menubaseline/baseline.go` 和 Gin 注册入口交叉追踪。路径中的 `handlers`、`router`、`ent/schema` 均在后端目录，`src` 在前端目录。已接线表示源码有生产调用链，浏览器验收单独记录。

| 模块与页面 | API、执行层及数据所有权 | 证据与判定 |
| --- | --- | --- |
| 我的工作台 `/workbench` | `workbench_routes.go` → `handlers/workbench` SQL 投影，入口 `dashboard:read`，服务按领域权限裁剪 | `domain_permission_test.go`；已接线，用户视角和刷新待验 |
| 全局搜索 `GlobalSearch.tsx` → `useGlobalSearch` | `router.go` → `handlers/global_search`，入口 `dashboard:read`，聚合多个业务域 | `useGlobalSearch.test.ts`；已接线，搜索动态表单字段未证实 |
| 工单新建与详情 `/tickets/create`、`TicketDetail.tsx` | `POST /tickets` 为 `ticket:create`，详情为 `ticket:read` → `handlers/ticket` → `Ticket`；流转由 `handlers/ticket_workflow` 转调 `TicketWorkflowService` | `ticket_type_workflow_e2e_test.go`、`TicketDetailWorkflowActions.test.tsx`；接单、解决、关闭已有服务端动作投影，权限与状态仍需逐入口比对 |
| 类型、模板、分类 | `ticket_type_routes.go`、`ticket_routes.go`、`ticket_category_routes.go` → `TicketType`、`TicketTemplate`、`TicketCategory`；模板使用 `ticket_template:*` | `ticket_type_precheck_route_test.go`、`handler_template_test.go`；已接线，预设升级及字段搜索审计待验 |
| 分配与自动化 | `/admin/tickets/{assignment-rules,automation-rules}` → `SetupTicketRoutes` → 分配、自动化处理器 → 对应规则 schema；`assignment_rule:*`、`automation_rule:*`，单票分配为 `ticket:assign` | `ticket_rule_config_permission_route_test.go`；配置接口已接线，执行、模拟、重试及循环防护待验 |
| 服务目录与请求 | `/service-requests` → `service_catalog_routes.go` → `handlers/service_request`，数据为 `ServiceCatalog`、`ServiceRequest`、`ServiceRequestApproval` | `service_catalog_list_route_test.go`、`service_request_approver_guard_test.go`、`service_request_sla_route_test.go`；已接线，交付闭环与重复 Tab 待浏览器验证 |
| BPMN 与审批链 | `/workflow/{designer,instances}` → `handlers/bpmn`，定义、实例、任务、投票各有 schema；`approval_chain_routes.go` → 审批链配置；工单表决使用 `ticket:approve` | `task_instance_filter_test.go`、`approval_chain/list_contract_test.go`；已有持久运行时，兼容投影并存 |
| SLA | `/workflow/sla` 与工单 SLA 区块 → `sla_routes.go`、`ticket_routes.go` → SLA 领域规则；关注 `SLAPolicy`、`SLAState` 和违约记录 | 工单读权限 `ticket:read`，暂停与恢复使用 `ticket:update`；跨域关联有路由测试，实际超时通知链待验 |
| AI 助手与创建辅助 | `/ai/chat`、工单创建页 → `ai_routes.go`、AI handler；提交工单仍有 `ticket:create` 入口；Agent 审批另见 `handlers/ai` | `tickets/create/__tests__/ai-triage.test.tsx`；辅助入口已接线，真实 provider、草稿确认和失败回退待验 |
| 评论、附件、通知、历史、关联 | `TicketDetail.tsx` → `ticket_routes.go` → 评论、附件、通知、工单历史或关联服务；对应 `TicketComment`、`TicketAttachment`、`TicketNotification`、`TicketWorkflowRecord` 等 | 附件写入与关联跨租户有路由测试；关联统计的 `relationsByType` 空对象及阻塞、重复计数常量属于部分实现，通知日志占位见 `TicketWorkflowService.NotifyTicketUpdate` |
| 初始化 | `pkg/seeder/baseline_creator.go`、`pkg/menubaseline`、`internal/bpmntemplates` | 有 baseline tenant scope、菜单权限和审批流程 reconcile 测试；首装、重放、升级保护待实测 |

首轮静态复核还发现两组并存入口。工单 `/:id/resolve`、`/:id/close` 的权限为 `ticket:update`，`/workflow/resolve`、`/workflow/close` 分别为 `ticket:resolve`、`ticket:close`。权限差异已确认，资源判定及实际状态迁移是否等价待测试。流转规则读方法 `GetWorkflowRules` 在给定工单类型时仍返回全部租户活跃规则，方法的实际页面消费及影响待确认。

首轮整改队列见 [整改 Backlog](remediation-backlog.md)，状态与审批边界见 [工单领域模型](../architecture/ticket-domain-model.md)，运行验收见 [质量验收矩阵](../acceptance/e2e-acceptance-matrix.md)。

菜单与路由另有三类明确关系。菜单基线将 `/approvals` 挂在 `task:read`，旧 `/approvals/pending` 为兼容入口；审批链配置单列 `approval:write`。自动化规则保留 `/workflow/automation` 与管理页两处入口，菜单都使用 `automation_rule:write`。原一级 AI 中心已从菜单基线撤下，顶栏助手仍使用 `/ai/chat`。服务目录列表仍有多个路由映射到同一 List handler，兼容入口的消费方需要在清理前登记。流程 SLA 监控读取 BPMN 指标，和工单 SLA 计时及 `sla:read` 的 SLA 管理页分别验收。
