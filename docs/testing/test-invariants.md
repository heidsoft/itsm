# 测试夹具与运行时不变量

> **Status**: current. 自 v1.5 起强制。
> **迁移来源**：
> - [`docs/review/system-function-review-result-2026-07-01.md`](../review/system-function-review-result-2026-07-01.md) §2（控制器测试 fixture 修复）
> - [`docs/review/system-function-review-result-2026-07-01.md`](../review/system-function-review-result-2026-07-01.md) §3（Jest 退出异常）
> - [`docs/review/architecture-review-2026-06-14.md`](../review/architecture-review-2026-06-14.md) §3（CMDB 跨租户）
> - [`docs/test-plan/itst-test-plan-v1.md`](../archive/testing-reports/itst-test-plan-v1.md)（测试环境分层与优先级定义）

本文档沉淀"在多个历史评审中反复触发，并被当前架构仍然依赖"的测试不变量。

---

## 1. 测试优先级与范围

### 1.1 P0 / P1 / P2 分类（统一）

| 优先级 | 含义 | 入口 |
|---|---|---|
| **P0** | 阻塞 GA；必须 100% 通过，否则不可发布 | CI 阻断（`backend-ci.yml`、`frontend-ci.yml`、`ga-gate.yml`） |
| **P1** | 重要能力；CI 失败需 review 是否阻断 | CI 警告 + 人工确认 |
| **P2** | 体验优化；CI 失败仅记录 | advisory |

P0 集合必须覆盖：用户认证与权限、工单/事件/变更完整生命周期、BPMN 引擎执行、SLA 监控与告警、跨租户隔离。

### 1.2 测试环境分层

| 环境 | 数据 | 用途 | 入口 |
|---|---|---|---|
| 开发（dev） | mock + sqlite in-memory | 本地快速反馈 | `make dev-test` |
| 测试（CI） | enttest.NewClient() + sqlite in-memory | 单元 + 集成 | `backend-ci.yml` |
| 组装（gate） | `docker compose -f docker-compose.dev.yml --profile dev up` | 端到端冒烟 | `ga-gate.yml` |
| 生产等价（cert） | `docker-compose.prod.yml --env-file .env.prod` | 发布证据 | `docs/release/` |

**禁止**把生产数据脱敏导入开发环境；脱敏数据也只能用于预生产。

---

## 2. 测试夹具不变量

### 2.1 共享唯一键必须唯一化

任何 controller/service 测试夹具**禁止**硬编码共享唯一键（如 `ticket_categories.code = "incident"`）。示例反例（评审 §A F-6..F-9）：

```go
// 反例：跨测试共享硬编码 code
client.TicketCategory.Create().
    SetName("incident").
    SetCode("incident").      // ← 跨测试冲突
    SetTenantID(tenant.ID).
    Save(ctx)
```

修复模式（任选其一）：

- **方案 A（推荐）**：`SetCode("incident-" + uniqueTestID())`
- **方案 B**：使用 Ent 的 `OnConflict()` upsert
- **方案 C**：`TestMain` 增加清理 `client.TicketCategory.Delete().ExecX(ctx)`

### 2.2 Tenant / User 上下文必须真实

任何 BPMN / 业务路由测试：

1. 必须在 setup helper 中创建真实 tenant + user（参考 `ticket_controller_test.go:78 createTestTenantAndUserForTicket`）；
2. mock 中间件 `Set("tenant_id", tenantID)` 的 tenantID 必须是非零值；
3. **禁止**依赖 `X-Test-Role` 等调试路径绕过租户上下文。

反例会触发 401 (code 2001)，表现成"业务路由失败"实为上下文缺失（评审 F-1..F-5）。

### 2.3 `defer cleanup` 强制

涉及多实体写入的测试必须在 helper 中：

```go
defer func() {
    _, _ = client.Ticket.Delete().Where(...).Exec(ctx)
    _, _ = client.User.Delete().Where(...).Exec(ctx)
    _, _ = client.Tenant.Delete().Where(...).Exec(ctx)
}()
```

### 2.4 默认值断言

如果 service 层在字段为空时补默认（如 `Incident.impact/urgency/severity`），对应测试必须：

1. **不**绕过 service 直接构造 Ent 模型；
2. 断言默认值生效且写入成功（评审 §2.1）。

---

## 3. Jest / Playwright 不变量

### 3.1 禁止 `Jest did not exit`

若 `npm run test:unit` 输出 `Jest did not exit one second after the test run has completed.`：

1. 必须立即用 `--detectOpenHandles` 定位（websocket / timer / message / mock server）；
2. CI 上若退出码 130 必须视为 P1，不允许长期忽略（评审 §3.1）。

### 3.2 构建期类型错误不得被忽略

`next.config.ts` 中：

- 禁止 `typescript.ignoreBuildErrors: true`
- 禁止 `eslint.ignoreDuringBuilds: true`

任何带 `ignore*` 的临时禁用必须附带 issue 链接与到期日，到期未修必须阻断构建。

### 3.3 E2E 必备状态

每个用户面页面（路由）必须具备：

| 状态 | 文件 | 触发条件 |
|---|---|---|
| loading | `loading.tsx` | Suspense fallback / 数据加载 |
| empty | `LoadingEmptyError` 中 empty 分支 | 数据为空 |
| error | `error.tsx`（路由级） + `global-error.tsx`（根级） | 渲染异常 |
| not-found | `not-found.tsx` | 路径不存在 |
| permission-denied | `AuthGuard` AccessDenied | RBAC 拒绝 |

缺失任何一项视为 P0 UX 不变量缺失。

### 3.4 错误边界跳转目标

`ErrorBoundary` / `AccessDenied` 的"返回首页"必须跳 `/dashboard` 或基于认证状态智能跳转，**禁止**跳 `/`（被 middleware 重定向到 `/login` 会让用户被登出）。

---

## 4. 后端运行时冒烟

CI 必须保持下列运行时基线（评审 §4）：

### 4.1 必须可用的端点

| 模块 | 路径 |
|---|---|
| Health | `GET /api/v1/health` |
| GA readiness | `GET /api/v1/readiness/ga` |
| CMDB | `GET /api/v1/cmdb/cis`、`/cmdb/ci-types`、`/cmdb/relationships`、`/configuration-items` |
| 服务目录 | `GET /api/v1/service-catalogs`、`/service-catalog` |
| 服务请求 | `GET /api/v1/service-requests`、`/service-requests/me`、`/service-requests/approvals/pending` |
| SLA | `GET /api/v1/sla/definitions`、`/sla/templates`、`/sla/policies`、`/sla` |
| BPMN | `GET /api/v1/bpmn/process-definitions`、`/process-instances`、`/tasks` |
| Workflow | `GET /api/v1/workflow/instances`、`/workflow/tasks` |
| Ticket workflow | `POST /api/v1/tickets/workflow/cc`、`GET /api/v1/tickets/:id/workflow/state` |

任何模块从该列表移除必须修改本文件并更新 `docs/scripts/smoke-api.sh`。

### 4.2 GA readiness 模块基线

`/api/v1/readiness/ga` 当前要求 12 modules ready。新增模块必须：

1. 在 `itsm-backend/service/readiness_service.go` 注册；
2. 提供 `/healthz` 子端点或就绪探针；
3. 在 `docs/scripts/smoke-api.sh` 增加对应路径。

---

## 5. API 契约静态初筛

PR 评审阶段（不依赖运行时）必须做的契约对齐检查：

1. 前端 `BASE_URL`（如 `/api/v1/configuration-items`）对应后端路由表必须存在；
2. 模板 / SLA / Ticket 子域（assignee / activity / batch-assign）路由必须注册；
3. Workflow / BPMN dashboard 路径必须经过 `scripts/check-api-paths.js` 验证。

`generate-acl-manifest.js` 自动生成 `docs/acl-manifest.yaml` 作为静态证据，缺失即视为 P1 阻断。

---

## 6. 报告字段约束

任何新提交的测试/部署/发布报告必须包含：

| 字段 | 含义 | 来源 |
|---|---|---|
| 日期 | 当次执行日期 | 报告 frontmatter 或 §0 |
| Git SHA | 提交哈希（短 7 位） | `git rev-parse --short HEAD` |
| 镜像 digest | 生产镜像 SHA | `docker images --digests` |
| 数据库类型 | PostgreSQL 版本 / 库名 | `SELECT version(); current_database()` |
| 执行命令 | 关键 CI 命令 | 报告 §1 |
| 未验证范围 | 哪些路径未跑 / 哪些已知跳过 | 报告末尾"已知限制"节 |

**缺失任意字段**的发布报告由 `scripts/docs-gate/check-release-claims.sh` 标为 advisory（v1.5）+ 后续升级为 hard。

---

## 7. 引用

- [`docs/architecture/workflow-cmdb-invariants.md`](../architecture/workflow-cmdb-invariants.md)
- [`docs/architecture/domain-ownership.md`](../architecture/domain-ownership.md)
- [`docs/testing/static-analysis-gates.md`](./static-analysis-gates.md)
- [`docs/testing/coverage-audit.md`](./coverage-audit.md)
- [`docs/scripts/smoke-api.sh`](../scripts/smoke-api.sh)
- [`scripts/test-coverage-guard.js`](../../scripts/test-coverage-guard.js)
