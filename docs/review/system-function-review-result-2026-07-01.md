# ITSM 系统功能 Review 执行结果

**执行时间**: 2026-07-01 21:48 CST  
**执行范围**: P0 基线验收、P0 历史问题复测准备、P1 API 契约静态初筛  
**分支**: `main`  
**HEAD**: `720e892a`  
**结论**: 基础编译和前端构建通过，但后端全量测试存在 P0 阻塞；当前本机缺少 Postgres/Docker 环境，无法完成运行时 API 复测。

**业务可用性补修**: 2026-07-01 至 2026-07-02 后续按“业务优先、测试后补”原则，已修复事件创建默认值、工单标签关联路由/payload 兼容、工单分类导入路由与基础导入能力，补齐 dashboard 配置/布局/部件/统计等前端兼容端点，修复工单模板管理与智能分派入口的后端契约，并补齐服务目录申请/取消/完成的字段与状态兼容。

---

## 1. 工作区基线

### 1.1 未提交变更

执行命令:

```bash
git status --short
```

结果:

```text
 M itsm-backend/router/cmdb_routes.go
 M itsm-backend/router/router.go
 M itsm-frontend/src/lib/api/incident-api.ts
?? docs/review/system-function-review-checklist-2026-07-01.md
```

说明:

- 前 3 个文件为本轮执行前已经存在的未提交变更，本轮未修改。
- 本轮新增 checklist 文档，并在本报告生成后新增本结果文档。

---

## 2. P0 后端基线

| 检查项 | 命令 | 结果 | 说明 |
|--------|------|------|------|
| 后端编译 | `cd itsm-backend && go build ./...` | 通过 | 退出码 0 |
| 静态检查 | `cd itsm-backend && go vet ./...` | 通过 | 退出码 0 |
| 格式检查工具 | `gofumpt -l .` | 未执行 | 本机未安装 `gofumpt` |
| 后端全量测试 | `cd itsm-backend && go test ./...` | 失败 | `itsm-backend/service` 包失败 |
| 覆盖率基线 | `go test ./... -coverprofile=coverage.out` | 未执行 | 全量测试已失败，覆盖率结果不可信 |

### 2.1 后端测试失败明细

失败包:

```text
itsm-backend/service
```

失败用例:

| 用例 | 失败原因 |
|------|----------|
| `TestAnalyticsService_GetDimensionValue` | `ent: constraint failed: FOREIGN KEY constraint failed` |
| `TestIncidentService_CreateIncident_Success` | `Incident.impact` 字段校验失败，值为空 |
| `TestIncidentService_CreateIncident_WithOptionalFields` | `Incident.impact` 字段校验失败，值为空 |
| `TestTicketLifecycleService_CancelWorkflow` | `ent: constraint failed: FOREIGN KEY constraint failed` |

判断:

- 这是当前 P0 阻塞。后端能编译，但不能作为 GA 基线测试通过。
- `Incident.impact` 失败更像测试 fixture 或默认值与 schema validator 不一致。
- analytics/ticket lifecycle 的外键失败需要检查测试数据创建顺序和必要关联对象。

建议修复:

- 优先修 `service/incident_service_test.go` 中 incident 创建 fixture，补合法 `impact` 默认值。
- 检查 `service/analytics_service_test.go` 和 `service/ticket_lifecycle_service_test.go` 的测试数据外键依赖。
- 修复后重跑 `cd itsm-backend && go test ./...`，再生成覆盖率基线。

---

## 3. P0 前端基线

| 检查项 | 命令 | 结果 | 说明 |
|--------|------|------|------|
| 依赖可用性 | `test -d node_modules` | 通过 | `node_modules-present` |
| 脚本确认 | `npm pkg get scripts` | 通过 | 脚本齐全 |
| 类型检查 | `npm run type-check` | 通过 | `tsc --noEmit` 退出码 0 |
| 单元测试 | `npm run test:unit -- --runInBand` | 功能通过，退出异常 | 36 suites / 544 tests 通过，但 Jest 未退出，被手动中断 |
| 生产构建 | `npm run build` | 通过 | 123 个 app routes 生成成功 |

### 3.1 前端单测退出异常

Jest 输出:

```text
Test Suites: 36 passed, 36 total
Tests:       13 skipped, 544 passed, 557 total
Jest did not exit one second after the test run has completed.
```

处理:

- 测试功能结果为通过。
- 进程未退出，手动 `Ctrl-C` 后退出码为 130。

判断:

- 这不是业务功能阻塞，但属于测试治理问题。
- 建议后续用 `--detectOpenHandles` 定位未关闭的 websocket、timer、message/notification 或 mock server。

### 3.2 前端 build 警告

构建通过，但存在大量 ESLint warning，主要类型:

- `no-console`
- `react-hooks/exhaustive-deps`
- unused eslint-disable directive
- Next.js ESLint plugin 未检测到

判断:

- 不阻塞 P0。
- 应归入 P4 工程治理，逐模块清理。

---

## 4. P0 历史问题运行时复测状态

### 4.1 本机服务状态

| 服务 | 探测 | 结果 |
|------|------|------|
| 后端 8090 | `curl http://localhost:8090/api/v1/health` | 未运行 |
| 前端 3000 | `curl http://localhost:3000/login` | 未运行 |
| Docker | `docker ps` | Docker daemon 未运行 |
| Postgres 5432 | `nc -z localhost 5432` | 端口关闭 |
| Redis 6379 | `nc -z localhost 6379` | 端口开启 |

结论:

- 当前本机不能完成登录、CMDB、服务目录、SLA、BPMN、workflow API 的运行时复测。
- 阻塞原因不是已确认功能失败，而是缺少可用 Postgres/Docker 全栈环境。

待复测项:

- 登录成功后前端 token/cookie 状态与跳转。
- CMDB API 是否仍有 404。
- 服务目录 API 是否仍有 404。
- SLA API 是否仍有 404。
- BPMN API 是否仍有 404。
- 工单 workflow 操作路径是否与前端一致。

建议环境恢复:

```bash
docker compose -f docker-compose.dev.yml up -d postgres redis
cd itsm-backend && go run -tags migrate main.go
cd itsm-backend && go run main.go
cd itsm-frontend && npm run dev
```

---

## 5. P1 API 契约静态初筛

### 5.1 已确认对齐的风险点

| 风险点 | 初筛结果 | 证据 |
|--------|----------|------|
| CMDB 前端 `BASE = /api/v1/configuration-items` | 后端存在兼容路由 | `itsm-backend/router/cmdb_routes.go` 注册 `/configuration-items` |
| SLA template 前端 `/api/v1/sla/templates` | 后端存在路由 | `controller/sla_template_controller.go` 注册 `/sla/templates` |
| ticket `assignee/:id`、`activity`、`batch-assign` | 后端存在路由 | `itsm-backend/router/ticket_routes.go` |

### 5.2 需要继续核查的候选契约问题

| 前端调用 | 后端发现 | 风险 |
|----------|----------|------|
| 前端部分 workflow/bpmn dashboard 路径 | 后端分散在 controller register 方法，需生成完整路由表确认 | 可能遗漏注册或前缀不一致 |

说明:

- 以上是静态初筛，不等于已确认 bug。
- 需要在后端可运行后用真实 HTTP smoke 确认。

### 5.3 已按业务优先完成的补修

| 问题 | 修复结果 | 最小验证 |
|------|----------|----------|
| 不传 `impact/urgency/severity/source` 创建事件会写入空值并触发 Ent validator | `CreateIncident` 在保存前统一使用默认值 `medium/manual` | `go test ./service -run 'TestIncidentService_CreateIncident'` 通过 |
| `POST /api/v1/tickets/workflow/cc` 未注册 | 主路由补挂 `TicketWorkflowController.CCTicket` | `go build ./...`、`go test ./controller ./router` 通过 |
| `POST/DELETE /api/v1/tickets/:id/tags` 未注册 | 主路由和备用 ticket 路由补挂工单标签关联接口 | `go build ./...`、`go test ./controller ./router` 通过 |
| 前端传 `{ tags: string[] }`，后端只接收 `tag_ids` | 后端同时兼容 `tag_ids` 和 `tags`；新增标签名会自动创建租户内标签后绑定 | `go build ./...`、`go test ./controller ./router` 通过 |
| `ticket-categories/tree/import` 固定路径可能被 `/:id` 吞掉 | 调整路由顺序，将 `/tree`、`/import/preview`、`/import` 放到 `/:id` 前 | `go build ./...`、`go test ./controller ./router` 通过 |
| 工单分类导入前端调用 404 | 增加 CSV/JSON 预览和执行导入基础实现 | `go build ./...`、`go test ./controller ./router` 通过 |
| dashboard 配置、布局、部件、模板、报表导出等前端声明端点未注册 | 增加只读默认配置和无副作用保存兜底；`/dashboard/stats/tickets` 复用已有工单统计 | `go build ./...`、`go test ./router` 通过 |
| 工单模板管理页期望分页包装和 snake_case 字段，后端只返回数组/camelCase | 模板列表返回 `{templates,total,page,page_size}`，模板对象同时兼容 `isActive/is_active`、`formFields/form_fields`、`fields` | `go build ./...`、`go test ./controller ./router` 通过 |
| 工单模板详情、启停、复制、分类接口未注册 | 增加 `/tickets/templates/:id`、`/:id/status`、`/:id/copy`、`/categories`，并返回真实模板数据 | `go build ./...`、`go test ./controller ./router` 通过 |
| 工单模板路由参数名与控制器不一致 | 将模板路由统一为 `:id`，避免更新/删除读不到模板 ID | `go build ./...`、`go test ./controller ./router` 通过 |
| 空系统创建工单模板依赖预置分类，容易直接失败 | 模板分类按字符串保存，不再强依赖分类表预置数据；未传优先级默认 `medium` | `go build ./...`、`go test ./controller ./router` 通过 |
| 智能分派弹窗调用自动分派/推荐/规则测试接口未挂到主路由 | 补挂 `/tickets/:id/auto-assign`、`/tickets/assign-recommendations/:id`、`/tickets/assignment-rules/test` | `go build ./...`、`go test ./controller ./router` 通过 |
| 服务目录创建/申请前端使用 snake_case 字段，后端只接 camelCase | 服务目录和服务请求 DTO 兼容 `delivery_time/ci_type_id/catalog_id/form_data/compliance_ack` 等字段，并在 handler 层归一化 | `go build ./...`、`go test ./handlers/service_catalog ./handlers/service_request ./router`、`npm run type-check` 通过 |
| 服务申请页可选过期时间为空时后端硬拒绝 | 后端为空时默认使用 30 天过期时间，保留页面“按默认策略”的行为 | `go build ./...`、`npm run type-check` 通过 |
| 服务请求完成接口 `/service-requests/:id/status` 未挂主 handler 路由 | 增加租户隔离的状态更新 service/handler，并注册 `PUT /api/v1/service-requests/:id/status` | `go build ./...`、`go test ./router` 通过 |
| 服务请求取消前端直接抛“未实现” | 前端 `cancelServiceRequest` 改为调用状态更新接口，后端兼容 `cancelled` 状态 | `npm run type-check` 通过 |
| 服务审批页使用 `pending_approval`，后端筛选状态不兼容 | 后端将 `pending_approval/pending` 归一为 `submitted`，兼容审批待办查询 | `go build ./...`、`go test ./handlers/service_request ./router` 通过 |
| 服务申请页面提供 `restricted` 数据分级，后端 oneof 不允许 | 后端 DTO 允许 `restricted`，避免前端合法选项提交失败 | `go build ./...` 通过 |

---

## 6. 本轮阻塞清单

| 优先级 | 阻塞项 | Owner 建议 | 下一步 |
|--------|--------|------------|--------|
| P0 | `go test ./...` 失败 | 后端 | 修复 service 包 4 个失败用例 |
| P0 | 本机 Postgres/Docker 不可用，无法运行时复测 | 环境/DevOps | 启动 Docker 或本地 Postgres |
| P1 | Jest 测试通过但进程不退出 | 前端 | 用 `--detectOpenHandles` 定位 open handle |
| P1 | API 契约候选 mismatch 未运行时验证 | 前后端 | 全栈启动后跑 HTTP smoke |
| P4 | 前端 lint warning 数量较多 | 前端 | 分模块清理 no-console 和 hooks deps |

---

## 7. 下一步执行建议

1. 修复后端 `service` 包测试失败，这是当前最硬的 P0。
2. 恢复 Docker/Postgres 环境，重跑历史 API 404 复测。
3. 为候选契约问题补 HTTP smoke:
   - `/api/v1/tickets/workflow/cc`
   - `/api/v1/tickets/:id/tags`
   - `/api/v1/ticket-categories/import/preview`
   - `/api/v1/ticket-categories/import`
   - `/api/v1/dashboard/config`
   - `/api/v1/dashboard/layout`
   - `/api/v1/dashboard/stats/tickets`
   - `/api/v1/tickets/templates`
   - `/api/v1/tickets/templates/:id/status`
   - `/api/v1/tickets/templates/:id/copy`
   - `/api/v1/tickets/:id/auto-assign`
   - `/api/v1/tickets/assign-recommendations/:id`
   - `/api/v1/service-catalogs`
   - `/api/v1/service-requests`
   - `/api/v1/service-requests/:id/status`
4. 前端单测加一轮:

```bash
cd itsm-frontend && npm run test:unit -- --runInBand --detectOpenHandles
```

5. 后端修复后重跑:

```bash
cd itsm-backend && go test ./...
cd itsm-backend && go test ./... -coverprofile=coverage.out
```
