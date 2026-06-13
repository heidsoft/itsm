---
description: "Task list for ITSM v1.0 GA 角色驱动的全产品测试方案"
---

# Tasks: ITSM v1.0 GA 角色驱动的全产品测试方案

**Input**: Design documents from `/specs/001-role-based-testing/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅
**Tests**: 本 feature 本身就是"测试方案"，所有 E2E / 冒烟 / OWASP 任务即"实现"任务，不再额外要求 TDD 红绿。
**Organization**: Phase 3..9 = 7 个用户故事（角色 E2E），独立可交付；Phase 10 = 跨角色 FLOW；Phase 11 = Polish & CI 收口。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件、无未完成依赖）
- **[Story]**: US1..US7（对应 spec.md 7 个用户故事）

## Path Conventions

- 后端：`itsm-backend/`
- 前端：`itsm-frontend/`
- 角色 E2E：`itsm-frontend/tests/e2e/roles/`
- 跨角色 FLOW：`itsm-frontend/tests/e2e/flows/`
- 冒烟脚本：`docs/scripts/smoke-api.sh`
- CI：`.github/workflows/ga-readiness.yml`

---

## Phase 1: Setup（共享基础设施）

**Purpose**: 准备测试运行所需的目录、依赖、配置；不阻塞 Foundational 之后的并行。

- [X] T001 创建目录骨架：`itsm-frontend/tests/e2e/roles/`、`itsm-frontend/tests/e2e/flows/`、`itsm-frontend/tests/e2e/fixtures/`、`docs/scripts/`
- [X] T002 [P] 校验 Playwright 配置 `itsm-frontend/playwright.config.ts`，确认 `testDir` 覆盖 `tests/e2e/roles` 与 `tests/e2e/flows`，并设 `baseURL = http://localhost:3000`、`extraHTTPHeaders` 注入 token 占位
- [X] T003 [P] 在 `itsm-frontend/package.json` `scripts` 增加 `"test:e2e:roles": "playwright test tests/e2e/roles"` 与 `"test:e2e:flows": "playwright test tests/e2e/flows"`（仅 scripts 字段，无新依赖）
- [X] T004 [P] 校验 `docker-compose.dev.yml` 暴露 `5432`、`6379`，与 `quickstart.md` 步骤 1 一致

**Checkpoint**: 目录与脚本入口就绪。

---

## Phase 2: Foundational（阻塞性前置）

**Purpose**: 任何角色 E2E 启动前必须先把"测试账号 + GA 准入端点 + 菜单一致性 + 共享 fixture"准备好。

**⚠️ CRITICAL**: 未完成本阶段，US1..US7 任一故事都跑不通。

- [X] T005 修改 `itsm-backend/pkg/seeder/seeder.go`：在默认 seed 中补 `engineer1 / eng123`（角色 engineer）、`manager1 / mgr123`（角色 approver/manager）、`tenant1admin / ta123`（角色 tenant_admin），均挂租户 `tenant_test`（FR-604, R-07）
- [X] T006 在 `itsm-backend/router/ga_readiness.go` 中将每个模块 `Endpoint` 字段改为 GET 可达路径（例如 `/api/v1/sla/definitions` 替代 POST-only `/sla/monitoring`；`/api/v1/ai/audit` 改为 `/api/v1/ai/metrics` 等），保证 `GET /api/v1/readiness/ga` 中所列端点 GET 返回 200（R-01）
- [X] T007 修改 `itsm-frontend/src/components/layout/sidebar/menu-config.ts`：删除 `/reports/cmdb-quality`、`/reports/sla-trend`、`/reports/incident-trends`、`/reports/change-success`、`/reports/problem-efficiency`、`/tickets/templates` 等无后端 API 的孤立子项；顶级 `/reports` 在 GA 前隐藏（FR-1001/1004）
- [X] T008 [P] 新建 `itsm-frontend/tests/e2e/fixtures/auth.ts`：扩展 Playwright `test`，提供 `login(role)` 返回 access_token 的 fixture，账号映射 `admin / user1 / security1 / engineer1 / manager1 / tenant1admin`（research.md R6）
- [X] T009 [P] 新建 `itsm-frontend/tests/e2e/fixtures/api.ts`：基于 `playwright.request` 封装 `apiGet(token, path)` / `apiPost(token, path, body)`，统一 `Authorization` 与 `X-Tenant-ID` 注入
- [X] T010 [P] 新建 `docs/scripts/smoke-api.sh` 骨架：`set -uo pipefail`、`fails=0`、`check()` 函数累加失败、最终 `exit $((fails>0))`，登录失败 `exit 2`（contracts/api-smoke-matrix.md "退出码约定"）
- [X] T011 [P] 新建 `docs/scripts/lib/jwt.sh` 工具脚本：`get_token <user> <pass>` 调 `/api/v1/auth/login` 并 `jq -r '.data.access_token'`
- [X] T012 在 `itsm-backend/pkg/seeder/seeder_test.go`（如不存在则新建）增加用例：seeder 跑完后 6 个测试账号均能登录且角色正确（验证 T005）

**Checkpoint**: 测试账号、GA 准入端点、菜单、fixture、smoke 骨架全部就绪 → Phase 3..9 可并行启动。

---

## Phase 3: User Story 1 — end_user 提单到关闭闭环 (Priority: P1) 🎯 MVP

**Goal**: 验证 `user1 / user123` 视角的"提单 → 跟踪 → 知识库 → 关闭评分"完整链路（spec §US1, EU-01..EU-10）。

**Independent Test**: `npx playwright test tests/e2e/roles/end-user.spec.ts` 全部绿灯即认定 P1 端用户视角交付可用。

### Implementation for User Story 1

- [X] T013 [P] [US1] 新建 `itsm-frontend/tests/e2e/roles/end-user.spec.ts`：用 fixture login `user1`
- [X] T014 [US1] 在 `end-user.spec.ts` 实现 `ALLOW-01`：`POST /api/v1/tickets` → `code=0` + `ticketNumber` 形如 `TKT-YYYYMM-NNNNNN`（FR-201, US1-AC1）
- [X] T015 [US1] 在 `end-user.spec.ts` 实现 `ALLOW-02`：`GET /api/v1/tickets` 仅返回 `requester_id == self`（FR-602, US1-AC2）
- [X] T016 [US1] 在 `end-user.spec.ts` 实现 `DENY-04`：`PUT /api/v1/tickets/{otherId}` → 401/403（US1-AC3）
- [X] T017 [US1] 在 `end-user.spec.ts` 实现 `DENY-01..03`：`/audit-logs`、`/configuration-items`、`/tenants` 全部 401/403（US1-AC4）
- [X] T018 [US1] 在 `end-user.spec.ts` 实现 `ALLOW-03`：工单解决后 `POST /api/v1/tickets/{self_id}/rating` `code=0` 且工单进入 `closed`（US1-AC5）
- [X] T019 [US1] 在 `end-user.spec.ts` 增加 SEC-A03-01（注入串 `' OR 1=1 --` 持久化但渲染转义；FR-903）+ SEC-A08-02（请求体篡改 `requester_id` 服务端忽略；FR-906）
- [X] T020 [US1] 在 `end-user.spec.ts` 增加知识库可见性断言：`/knowledge-articles` 仅返回 `published`（FR-303 状态机的对侧）

**Checkpoint**: end-user 角色 E2E 独立可跑，覆盖 ALLOW-01/02/03 + DENY-01/02/03/04 + A03/A08 + KB 可见性。

---

## Phase 4: User Story 2 — engineer 处理人全生命周期 (Priority: P1)

**Goal**: 工程师视角的工单/事件/问题/变更/CMDB/AI 完整处置链（spec §US2, EN-01..EN-10）。

**Independent Test**: `npx playwright test tests/e2e/roles/engineer.spec.ts` 全部绿灯。

### Implementation for User Story 2

- [X] T021 [P] [US2] 新建 `itsm-frontend/tests/e2e/roles/engineer.spec.ts`：用 fixture login `engineer1`
- [X] T022 [US2] 在 `engineer.spec.ts` 实现 `ALLOW-04`：分配的工单 `open → in_progress → resolved` 状态机推进，每步 `code=0` 且写 audit-logs（FR-202/803, US2-AC1）
- [X] T023 [US2] 在 `engineer.spec.ts` 实现 `ALLOW-05`：事件解决后 `POST /api/v1/incidents/{id}/root-cause` 创建 problem 并形成引用关系（FR-203, US2-AC2）
- [X] T024 [US2] 在 `engineer.spec.ts` 验证 `incident → problem → change` 全链：基于 problem 创建 change，change 进入审批流且 `related_problem_id` 正确（US2-AC3）
- [X] T025 [US2] 在 `engineer.spec.ts` 验证 change 推进 `approved → in_progress → completed` + CI 字段变更被审计 + `affected_ci_ids` 反向可查（US2-AC4）
- [X] T026 [US2] 在 `engineer.spec.ts` 实现 `ALLOW-06`：`POST /api/v1/ai/audit` `{accepted:true}` 后 `/ai/metrics.acceptance_rate` 上升（FR-502/503, US2-AC5）
- [X] T027 [US2] 在 `engineer.spec.ts` 增加 `DENY-08`：`DELETE /api/v1/tenants/{id}` → 403（角色矩阵）
- [X] T028 [US2] 在 `engineer.spec.ts` 增加 SEC-A03-02：CI 名 `<script>...</script>` 持久化但 HTML 转义（owasp-test-matrix）

**Checkpoint**: engineer 角色 E2E 覆盖 ITIL 闭环 + AI 审计 + CMDB + 越权拒绝 + XSS 转义。

---

## Phase 5: User Story 3 — super_admin 跨租户与平台治理 (Priority: P1)

**Goal**: super_admin 完成菜单 / readiness / 租户切换 / 角色权限 / 连接器 lifecycle 全流程（spec §US3, SA-01..SA-10）。

**Independent Test**: `npx playwright test tests/e2e/roles/super-admin.spec.ts` 全部绿灯。

### Implementation for User Story 3

- [ ] T029 [P] [US3] 新建 `itsm-frontend/tests/e2e/roles/super-admin.spec.ts`：用 fixture login `admin`
- [ ] T030 [US3] 在 `super-admin.spec.ts` 实现 `ALLOW-07`：`GET /api/v1/auth/menus` `length >= 20`（FR-104，spec 提到 ≥80 项视生产 seed 而定，本契约取 ≥20 作为冒烟下限并在 SA-01 留 TODO 校准）
- [ ] T031 [US3] 在 `super-admin.spec.ts` 实现 `ALLOW-08`：`GET /api/v1/readiness/ga` `data.modules.length=12` 且全部 `ready`（FR-802, US3-AC2）
- [ ] T032 [US3] 在 `super-admin.spec.ts` 验证租户切换后 `/api/v1/users` 与原租户隔离（US3-AC3）
- [ ] T033 [US3] 在 `super-admin.spec.ts` 验证 `PUT /api/v1/roles/{id}/permissions` 后受影响用户复登录权限实时生效（US3-AC4）
- [ ] T034 [US3] 在 `super-admin.spec.ts` 实现连接器 lifecycle：installed → enabled → health(`ready`) → disabled → uninstalled，每步 `code=0`（FR-701/703, US3-AC5）
- [ ] T035 [US3] 在 `super-admin.spec.ts` 增加菜单一致性断言：`/api/v1/auth/menus` 返回的所有 `path` MUST 在前端路由表与后端 GET 路由中均可达（FR-1002）

**Checkpoint**: super_admin 角色 E2E 覆盖菜单/准入/租户/RBAC/连接器五条治理主线。

---

## Phase 6: User Story 4 — approver 多级审批 (Priority: P2)

**Goal**: approver 处理 change/service_request 多级审批与超时升级（spec §US4, AP-01..AP-06）。

**Independent Test**: `npx playwright test tests/e2e/roles/approver.spec.ts` 全部绿灯。

### Implementation for User Story 4

- [ ] T036 [P] [US4] 新建 `itsm-frontend/tests/e2e/roles/approver.spec.ts`：用 fixture login `manager1`
- [ ] T037 [US4] 在 `approver.spec.ts` 实现 `ALLOW-10`：`POST /api/v1/approval-workflows/instances/{id}/approve` 一级通过后流转下一节点（US4-AC1）
- [ ] T038 [US4] 在 `approver.spec.ts` 验证任一节点 reject → 整工作流 `rejected` + 审计（FR-803, US4-AC2）
- [ ] T039 [US4] 在 `approver.spec.ts` 验证节点 timeout 触发升级：等待 > `manager_timeout` 后 `escalation_history` 新增记录（FR-404, US4-AC3）
- [ ] T040 [US4] 在 `approver.spec.ts` 增加 `DENY-10`：`PUT /api/v1/tickets/{id}` 非审批字段 → 403（角色矩阵）

**Checkpoint**: approver 角色 E2E 覆盖通过 / 驳回 / 超时升级 / 越权拒绝。

---

## Phase 7: User Story 5 — sd_manager 服务台主管运营视图 (Priority: P2)

**Goal**: sd_manager 完成 dashboard / 自动分派 / SLA 监控（spec §US5, SD-01..SD-07）。

**Independent Test**: `npx playwright test tests/e2e/roles/sd-manager.spec.ts` 全部绿灯。

### Implementation for User Story 5

- [ ] T041 [P] [US5] 新建 `itsm-frontend/tests/e2e/roles/sd-manager.spec.ts`：用 fixture login `admin` 切换到 sd_manager 角色（或专用 seed 账号）
- [ ] T042 [US5] 在 `sd-manager.spec.ts` 验证 `GET /api/v1/dashboard/overview` 返回今日工单/事件/SLA 风险三类计数（US5-AC1）
- [ ] T043 [US5] 在 `sd-manager.spec.ts` 实现自动分派规则：创建规则 → 新工单匹配 → `assignee_id` 自动写入（US5-AC2）
- [ ] T044 [US5] 在 `sd-manager.spec.ts` 实现 `ALLOW-09`：`POST /api/v1/sla/monitoring` 返回风险列表并触发连接器通知（FR-402/403, US5-AC3）
- [ ] T045 [US5] 在 `sd-manager.spec.ts` 验证强制升级路径：手动 `escalate` 工单 → audit-logs 写 `escalate` 动作

**Checkpoint**: sd_manager 角色 E2E 覆盖运营总览 + 分派 + SLA 风险 + 强制升级。

---

## Phase 8: User Story 6 — security 安全管理员只读审计 (Priority: P2)

**Goal**: security 角色仅可读 audit-logs，禁止任何业务/配置/用户写（spec §US6, SE-01..SE-06）。

**Independent Test**: `npx playwright test tests/e2e/roles/security.spec.ts` 全部绿灯。

### Implementation for User Story 6

- [ ] T046 [P] [US6] 新建 `itsm-frontend/tests/e2e/roles/security.spec.ts`：用 fixture login `security1`
- [ ] T047 [US6] 在 `security.spec.ts` 实现 `ALLOW-11`：`GET /api/v1/audit-logs` `code=0`（FR-603, US6-AC1）
- [ ] T048 [US6] 在 `security.spec.ts` 实现 `DENY-05/06/07`：`PUT /api/v1/users/{id}`、`PUT /api/v1/sla/definitions/{id}`、`POST /api/v1/tickets` 全部 403（US6-AC2）
- [ ] T049 [US6] 在 `security.spec.ts` 增加 SEC-A02-01：JWT 篡改任意位 → 401（FR-902, owasp-test-matrix）
- [ ] T050 [US6] 在 `security.spec.ts` 验证 5 次失败登录后审计日志可被 security 检索 `login_failed`（FR-907, US6-AC3）

**Checkpoint**: security 角色 E2E 覆盖只读审计 + 业务写拒绝 + JWT 篡改 + 失败登录可追溯。

---

## Phase 9: User Story 7 — tenant_admin 租户隔离 (Priority: P2)

**Goal**: tenant_admin 仅能操作自身租户数据，跨租户读 = 空集 / 写 = 服务端忽略（spec §US7, TA-01..TA-06）。

**Independent Test**: `npx playwright test tests/e2e/roles/tenant-admin.spec.ts` 全部绿灯。

### Implementation for User Story 7

- [ ] T051 [P] [US7] 新建 `itsm-frontend/tests/e2e/roles/tenant-admin.spec.ts`：用 fixture login `tenant1admin`
- [ ] T052 [US7] 在 `tenant-admin.spec.ts` 实现 `ALLOW-12`：`GET /api/v1/users` 仅本租户用户（FR-601, US7-AC1）
- [ ] T053 [US7] 在 `tenant-admin.spec.ts` 实现 `DENY-09`：`GET /api/v1/tickets?tenant_id=other` → 空集 / 403
- [ ] T054 [US7] 在 `tenant-admin.spec.ts` 实现 `DENY-11`：`PUT /api/v1/tickets/{id}` body `tenant_id=other` → 服务端忽略，仍记 token tenant（FR-906, US7-AC2）
- [ ] T055 [US7] 在 `tenant-admin.spec.ts` 验证 CIType 配置 `/api/v1/configuration-items/types` 在本租户范围内可读写

**Checkpoint**: tenant_admin 角色 E2E 覆盖租户隔离硬底线。

---

## Phase 10: 跨角色 FLOW + OWASP 冒烟 + API 冒烟矩阵

**Purpose**: 把 spec §SC-103（FLOW-1..10）+ §SC-204（≥25 端点冒烟）+ OWASP smoke 层用例串成可重复脚本。

### API 冒烟（contracts/api-smoke-matrix.md，32 端点）

- [X] T056 在 `docs/scripts/smoke-api.sh` 实现 #1..#5（health/readiness/auth/me/menus）
- [X] T057 在 `docs/scripts/smoke-api.sh` 实现 #6..#9（tickets list/create + incidents list/create）
- [X] T058 在 `docs/scripts/smoke-api.sh` 实现 #10..#14（problems/changes/CI/CI-create/ci-types）
- [X] T059 在 `docs/scripts/smoke-api.sh` 实现 #15..#19（knowledge x2 / service-catalog / sla-defs / sla-monitoring POST）
- [X] T060 在 `docs/scripts/smoke-api.sh` 实现 #20..#24（connectors/lifecycle / `/api/v1/bpmn/process-instances` / `/api/v1/workflow/instances` / approval-workflows / process-bindings）
- [X] T061 在 `docs/scripts/smoke-api.sh` 实现 #25..#28（ai/triage POST、ai/audit POST、dashboard/overview、analytics/tickets）
- [X] T062 在 `docs/scripts/smoke-api.sh` 实现 #29..#32（audit-logs / users / tenants / roles）
- [X] T063 在 `docs/scripts/smoke-api.sh` 末尾 `if [[ $fails -gt 0 ]]; then exit 1; fi` 并打印 `All endpoints OK`

### OWASP 冒烟层（owasp-test-matrix.md）

- [ ] T064 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A02-02：登录后将 token `exp -1` → 401（FR-902）
- [ ] T065 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A03-01：`POST /tickets` `title` 含 `' OR 1=1 --` → 持久化但 GET 渲染时 HTML 转义、DB 不破坏（FR-903）
- [ ] T066 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A05-02：模拟 Redis NOAUTH（连接到无密码端口）→ 应用 fallback 内存限流，warning 日志（FR-904）
- [ ] T067 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A07-01：5 次错误密码登录 → 第 6 次 429 + audit `login_failed`（FR-905）
- [ ] T068 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A09-01/02：创建工单后 `GET /audit-logs?resource=ticket&action=create` ≥1 条；5 次失败登录后 `login_failed` ≥5 条（FR-907）
- [ ] T069 [P] 在 `docs/scripts/smoke-api.sh` 增加 SEC-A10-01/02：连接器 webhook URL `http://127.0.0.1:22` 拒绝；公有合法 URL 通过（FR-908）

### 跨角色 FLOW（spec §SC-103，FLOW-1..10）

- [X] T070 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-1-incident-to-change.spec.ts`：FLOW-1 incident → problem → change 端到端（end_user 报障 → engineer 排障 → approver 审批 → engineer 实施）
- [X] T071 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-3-major-incident.spec.ts`：FLOW-3 重大事件多角色协作
- [X] T072 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-4-service-request.spec.ts`：FLOW-4 服务请求 catalog → submit → approve → fulfill → complete
- [X] T073 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-7-sla-breach.spec.ts`：FLOW-7 SLA 风险 → 监控 → 通知 → 升级
- [X] T074 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-9-tenant-isolation.spec.ts`：FLOW-9 多租户零跨租户泄露（tenant1 vs tenant2 双向断言）
- [X] T075 [P] 新建 `itsm-frontend/tests/e2e/flows/flow-10-knowledge.spec.ts`：FLOW-10 KB 草稿 → 发布 → end_user 可见 → 引用到工单
- [X] T076 新建 `itsm-frontend/tests/e2e/flows/flow-2-5-6-8.spec.ts`：FLOW-2/5/6/8（标准变更 / 紧急变更 / AI 工单建议接受 / CMDB 影响分析）合并文件按 describe 组织

**Checkpoint**: 冒烟脚本 + OWASP smoke + 10 条 FLOW 全绿 → 准入门槛硬指标达成（SC-103 + SC-204 + SC-401）。

---

## Phase 11: Polish & Cross-Cutting（CI 收口 + 文档同步）

**Purpose**: 把所有自动化串入 GitHub Actions，做最终一致性扫描，更新源文档。

- [X] T077 新建 `.github/workflows/ga-readiness.yml`：单 job 串行 — services(postgres/redis) → `go build ./...` → `go test ./... -count=1` → 启 backend 二进制 → `bash docs/scripts/smoke-api.sh` → `cd itsm-frontend && npm ci && npm run type-check && npm run lint:check && npm run test:unit && npm run build` → `npx next dev` 后台 → `npx playwright test tests/e2e/roles tests/e2e/flows`；失败上传 `playwright-report/` 与 `/tmp/itsm-*.log`（research.md R4）
- [X] T078 [P] 在 `docs/ITSM-基于角色视角的产品测试方案.md` §六"已知不可达"小节同步 NC-002 决议：删除 `/templates/tickets`、`/incident-rules`，改写 `/process-instances` → `/api/v1/bpmn/process-instances` + `/api/v1/workflow/instances`
- [X] T079 [P] 在 `docs/ITSM-基于角色视角的产品测试方案.md` 头部"自动化命令"小节追加 `bash docs/scripts/smoke-api.sh` 与 `npx playwright test tests/e2e/roles` / `tests/e2e/flows` 的标准命令
- [X] T080 [P] 校验 `specs/001-role-based-testing/checklists/ga-readiness.md` 中 CHK080..CHK084（菜单一致性）能由 T007 / T077 / 冒烟脚本对应自动产出证据
- [X] T081 [P] 在 `itsm-backend` 增加 `pkg/seeder/seeder_test.go`（如 T012 未覆盖）补租户 `tenant_test` + 6 测试账号 seed 幂等性测试
- [X] T082 在 `docs/scripts/smoke-api.sh` 顶部加入 `BASE=${BASE:-http://localhost:8090}` 与 `--help` 行为，便于 CI/本地复用
- [ ] T083 跑一次 `quickstart.md` 0..8 步全流程，确认产物全绿：后端 17 包通过、前端 526 单测通过、smoke `exit=0`、roles & flows 全绿、`/readiness/ga` 12/12
- [X] T084 用 `git diff itsm-frontend/src/components/layout/sidebar/menu-config.ts` 与 DB `menus` 表 diff 收敛为零（FR-1004 接受）；记录到 `specs/001-role-based-testing/checklists/ga-readiness.md` CHK082 证据列
- [X] T085 把 `tasks.md` 完成度（X / 85）写回 `specs/001-role-based-testing/plan.md` 末尾"Done When"对应行；GA Go/No-Go 由 SC-101..403 全绿 + checklist 全勾决定

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖，立即开始。
- **Phase 2 (Foundational)**: 依赖 Phase 1；阻塞 Phase 3..9 的 E2E 启动（无账号 / 无 fixture / 无 readiness 端点修正即跑不通）。
- **Phase 3..9 (US1..US7)**: 全部依赖 Phase 2；彼此独立，可并行（不同 spec 文件、不同角色账号）。
- **Phase 10 (FLOW + Smoke)**: 冒烟脚本依赖 Phase 2 的 T011；FLOW E2E 依赖 Phase 3..9 任一角色 spec 完成（最少 US1 + US2 + US3 三角色 ready 时即可起跑 FLOW-1/3/9/10）。
- **Phase 11 (Polish)**: T077 CI 依赖 Phase 1..10 全部产物；T078..T085 可在 Phase 10 完成后并行收尾。

### User Story Dependencies

- **US1 (P1) end_user**：仅依赖 Foundational；MVP，建议第一条交付。
- **US2 (P1) engineer**：仅依赖 Foundational；与 US1 完全独立；FLOW-1/3 需要 US1 + US2。
- **US3 (P1) super_admin**：仅依赖 Foundational；FLOW-9 多租户需要 US3 + US7。
- **US4..US7 (P2)**：仅依赖 Foundational；与 P1 故事独立；FLOW-7 需 US5、FLOW-3/4 需 US4 + US2。

### Within Each User Story

- 一个 `*.spec.ts` 文件内部按 `describe(role) > test(scenario)` 组织；模型/服务由后端既有代码提供，本 feature 不新增。
- 角色矩阵 DENY/ALLOW 表中"该角色"行的所有断言必须出现在对应 spec 中（traceability）。

### Parallel Opportunities

- Phase 1：T002/T003/T004 全部 [P]。
- Phase 2：T008/T009/T010/T011 [P]，但都依赖 T005（账号）+ T006（端点）+ T007（菜单）任一未完成即可能阻塞下游 E2E。
- Phase 3..9：7 份角色 spec 文件互斥，全部 [P] 可并行。
- Phase 10：FLOW 6 份文件 [P]；OWASP smoke 6 项 [P]；冒烟矩阵 32 端点按文件递增（同一文件需串行 T056..T063）。
- Phase 11：T078/T079/T080/T081 [P]；T077/T083/T085 串行收口。

---

## Parallel Example: 启动 Phase 3..9 七角色

```bash
# Foundational (Phase 2) 完成后，开 7 个并行任务：
Task: "T013..T020 [US1] end-user.spec.ts"
Task: "T021..T028 [US2] engineer.spec.ts"
Task: "T029..T035 [US3] super-admin.spec.ts"
Task: "T036..T040 [US4] approver.spec.ts"
Task: "T041..T045 [US5] sd-manager.spec.ts"
Task: "T046..T050 [US6] security.spec.ts"
Task: "T051..T055 [US7] tenant-admin.spec.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1 (T001..T004)。
2. 完成 Phase 2 (T005..T012) — **阻塞所有故事**。
3. 完成 Phase 3 (T013..T020) — end_user 全绿。
4. 跑 `npx playwright test tests/e2e/roles/end-user.spec.ts` + `bash docs/scripts/smoke-api.sh`（仅 health/auth/tickets 部分），独立判定"端用户视角是否交付可用"。
5. 通过即可作为 GA-MVP 切片。

### Incremental Delivery（推荐路线）

1. Setup + Foundational → 基础就绪。
2. + US1 (end_user) → 跑通 → MVP demo。
3. + US2 (engineer) → 跑通 → 业务闭环 demo。
4. + US3 (super_admin) → 跑通 → 治理 demo（P1 全绿，可 RC 候选）。
5. + US4..US7 (P2) → 跑通 → 全角色覆盖。
6. + Phase 10 FLOW + OWASP smoke → 跑通 → SC-103/204/401 达成。
7. + Phase 11 CI 收口 + 文档同步 → GA Go/No-Go 决策。

### Parallel Team Strategy

- Dev A：Phase 2 (Foundational) + Phase 11 CI（T005/T006/T007/T077）。
- Dev B：US1 + US2（FLOW-1/3 主路径）。
- Dev C：US3 + US7 + FLOW-9（多租户线）。
- Dev D：US4 + US5 + FLOW-4/7（审批 + SLA 线）。
- Dev E：US6 + Phase 10 OWASP smoke + 冒烟矩阵脚本（安全线）。

---

## Notes

- `[P]` 任务=不同文件、无未完成依赖；同一 `smoke-api.sh` 内部递增的 #1..#32 不并行。
- `[Story]` 标签仅出现在 Phase 3..9；Phase 1/2/10/11 不带 Story 标签。
- 本 feature 不新增业务实体（见 data-model.md），故无 Models/Services 任务；所有"实现"=测试工件 + 3 处微调（seeder/ga_readiness/menu-config）。
- 提交节奏建议：Phase 1/2 一次合并；Phase 3..9 每角色一次 PR；Phase 10 FLOW 一次 PR；Phase 11 CI 一次 PR。
- 退出准入：SC-101/102/103/201/202/204/401/402/403 全部达成 + checklists/ga-readiness.md 全部勾选 = GA Go。

---

## Done When

- [ ] tasks.md 已生成（85 条任务，11 个阶段）
- [ ] Extension hooks（before/after_tasks）— `.specify/extensions.yml` 未注册，已跳过
- [ ] 完成情况报告已输出（见对话末尾）
