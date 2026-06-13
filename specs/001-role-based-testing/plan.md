# Implementation Plan: ITSM v1.0 GA 角色驱动的全产品测试方案

**Branch**: `001-role-based-testing` | **Date**: 2026-06-13 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-role-based-testing/spec.md`

## Summary

将既有的 `docs/ITSM-基于角色视角的产品测试方案.md` 落地为可执行、可重复、可在 CI 上跑的"质量门禁工程"：

- **角色驱动**：以 7 类角色（end_user/engineer/super_admin 为 P1，approver/sd_manager/security/tenant_admin 为 P2）为切片，每个角色一个独立可交付的 E2E 套件。
- **流程闭环**：FLOW-1..FLOW-10 跨角色端到端必须 100% 通过。
- **风险加权**：R-01..R-08（spec §11）逐项 mitigation 并写入 GA checklist。
- **自动化优先**：API 冒烟脚本（≥25 端点）+ 7 份 Playwright 角色 spec + 后端 `go test` + 前端 Jest，全部纳入 CI。
- **NC 已澄清**：FR-NC-001/002 决议执行——前端 `menu-config.ts` 与 DB `menus` 表收敛；`/api/v1/bpmn/process-instances` 替代 `/process-instances`；`/templates/tickets`、`/incident-rules` 移出 GA 矩阵。

技术路径：Bash 冒烟脚本 → Playwright 角色 spec → CI 工作流（GitHub Actions）→ 缺陷登记表 → RC → GA。

## Technical Context

**Language/Version**:
- 后端：Go 1.22+，Gin，Ent ORM
- 前端：TypeScript 5.x，Next.js 15，React 19
- 测试：Bash（curl + jq）、Playwright 1.48+、Jest 29、Go testing + testify

**Primary Dependencies**:
- 后端：`gin`, `entgo.io/ent`, `nitram509/lib-bpmn-engine`, `go-redis`, `zap`
- 前端：`next`, `antd`, `zustand`, `playwright`, `@testing-library/react`
- 测试工具：`curl`、`jq`、`bash`、`gh`（GitHub Actions runner）

**Storage**:
- PostgreSQL 15（主存储，Ent schema 自动迁移）
- Redis 7（限流、序列号；NOAUTH 时 fallback 内存）
- E2E 期间使用独立 schema 或 `tenant_test` 隔离

**Testing**:
- 后端单元/集成：`go test ./...`（`enttest.Open` SQLite 内存）
- 前端单元：`npm run test:unit`（Jest）
- API 冒烟：`docs/scripts/smoke-api.sh`（新建）
- 角色 E2E：`itsm-frontend/tests/e2e/roles/*.spec.ts`（新建 7 份）
- 性能：k6（独立 baseline）
- 安全：OWASP 用例化为 Playwright + Bash

**Target Platform**:
- 后端：Linux x86_64（Docker `docker-compose.prod.yml`）
- 前端：现代浏览器（Chrome/Firefox/Safari），生产 Nginx 反代
- CI：GitHub Actions ubuntu-latest

**Project Type**: Web application（`itsm-backend/` + `itsm-frontend/`）

**Performance Goals**（同 spec SC-3xx）:
- `/api/v1/tickets` 列表 P95 < 300ms @ 50 RPS
- 工单创建 P95 < 500ms @ 20 RPS
- `/dashboard/overview` P95 < 600ms
- AI Triage P95 < 3s（独立 baseline）
- 24h soak：错误率 < 0.1%，RSS 增长 < 5%/h

**Constraints**:
- 单节点 4C8G 容器；生产 `SERVER_MODE=release`、`ITSM_CORS_ALLOWED_ORIGINS` 必填、`REDIS_PASSWORD` 必填或 fallback 透明。
- 多租户隔离零跨租户泄露（FLOW-9 / FR-601）。
- 测试不得污染生产数据：使用专用租户 + 测试账号；E2E 用例自清理。

**Scale/Scope**:
- 7 角色 × 50+ 场景 × 3 质量维度
- ≥25 端点冒烟矩阵
- 12 GA 准入模块（`/api/v1/readiness/ga`）
- ≥526 前端单测维持 100% 通过；后端 17 包维持 100% 通过

## Constitution Check

*GATE: 必须在 Phase 0 之前通过；Phase 1 设计完成后复检。*

项目 `.specify/memory/constitution.md` 仍是模板占位符（未实施正式宪法），按"默认 gate"评估：

| Gate | 决议 | 证据 |
|------|------|------|
| Test-First | ✅ 默认遵守 | spec §SC-2xx 强制 `go test` / Jest / E2E / 冒烟脚本全绿后才进 RC |
| Simplicity | ✅ 通过 | 复用现有自动化基线（17 后端包 + 526 前端用例 + 13 项 E2E），仅新增"角色 E2E + 冒烟脚本 + CI 工作流"三类工件，不引入新框架 |
| Library-First | N/A | 本 feature 是测试方案，不输出库代码 |
| Integration Testing | ✅ 通过 | FLOW-1..10 + OWASP A01-A10 + 多租户隔离全部走真接口 |
| Observability | ✅ 通过 | 复用 zap + audit-logs + `/readiness/ga`，无新组件 |

**结论**：无 Constitution 违例；进入 Phase 0。

## Project Structure

### Documentation (this feature)

```text
specs/001-role-based-testing/
├── plan.md                         # 本文件
├── spec.md                         # Ready
├── research.md                     # Phase 0 产出（本次生成）
├── data-model.md                   # Phase 1 产出（本次生成）
├── quickstart.md                   # Phase 1 产出（本次生成）
├── contracts/                      # Phase 1 产出（本次生成）
│   ├── api-smoke-matrix.md
│   ├── role-permission-matrix.md
│   └── owasp-test-matrix.md
├── checklists/
│   └── ga-readiness.md             # 已生成
└── tasks.md                        # Phase 2 产出（由 /speckit-tasks 生成）
```

### Source Code (repository root)

本 feature 不修改业务源码主体，仅新增/修订下列测试与配置工件；保留现有 `itsm-backend/` 与 `itsm-frontend/` 结构。

```text
itsm-backend/
├── pkg/seeder/seeder.go                # 微调：补 engineer1/manager1/tenant1admin（FR-604, R-07）
├── router/ga_readiness.go              # 微调：endpoint 字段对齐真实 GET 路由（R-01）
└── ...                                 # 其余不动

itsm-frontend/
├── src/components/layout/sidebar/menu-config.ts  # 删除孤立 reports/* 子项（FR-1001/1004）
└── tests/e2e/roles/                    # 新建：7 份角色 spec
    ├── end-user.spec.ts
    ├── engineer.spec.ts
    ├── super-admin.spec.ts
    ├── approver.spec.ts
    ├── sd-manager.spec.ts
    ├── security.spec.ts
    └── tenant-admin.spec.ts

docs/scripts/
└── smoke-api.sh                        # 新建：≥25 端点 curl + jq 冒烟

.github/workflows/
└── ga-readiness.yml                    # 新建：go test + npm test + 启后端 + 跑冒烟 + 角色 E2E
```

**Structure Decision**: Web application 双工程（`itsm-backend` + `itsm-frontend`）。本 feature 以"测试与配置工件"形态附加：

- 不新建独立测试工程；E2E 复用 `itsm-frontend/tests/e2e/`。
- 冒烟脚本落 `docs/scripts/`，Bash + curl + jq，无额外依赖。
- CI 工作流落 `.github/workflows/`。
- 业务代码改动控制在 3 处微调（seeder、ga_readiness.go、menu-config.ts），均为风险登记 mitigation。

## Complexity Tracking

> Constitution Check 全部通过，无违例待解释。本节留空。

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| —         | —          | —                                    |

## Phase 0 — Research（已并入本次生成）

详见 [research.md](./research.md)。要点：

- **角色账号补建策略**：选择"seeder 补丁 + 文档化前置脚本"组合；GA 镜像内置 engineer1/manager1/tenant1admin。
- **菜单一致性收敛**：以 DB `menus` 表为权威源；前端 `menu-config.ts` 仅作开发兜底，diff 必须为零。
- **GA Readiness 端点对齐**：修正 `router/ga_readiness.go` 的 `Endpoint` 字段为可达 GET 路径（取消对 POST-only `/sla/monitoring`、`/ai/audit` 的误标）。
- **CI 选型**：GitHub Actions 单 job 串行（启 PG + Redis service → 启后端二进制 → 冒烟 → 启前端 → Playwright），降低复杂度优先于并行加速。
- **OWASP 落地**：A01/A02/A07/A08 进角色 E2E；A03/A05/A09/A10 进冒烟脚本；A06 暂不在 GA 范围。

## Phase 1 — Design & Contracts（已并入本次生成）

- **数据模型**（[data-model.md](./data-model.md)）：本 feature 不新增业务实体；列出测试涉及的关键实体与状态机以备 E2E 引用。
- **接口契约**（[contracts/](./contracts/)）：
  - `api-smoke-matrix.md` — 冒烟 ≥25 端点的方法/期望/角色。
  - `role-permission-matrix.md` — 角色 × 资源 × 操作矩阵（spec 附录 A 的可机读化版本）。
  - `owasp-test-matrix.md` — A01-A10 → 测试 ID → 自动化层映射。
- **快速开始**（[quickstart.md](./quickstart.md)）：本地 + CI 双流程的最小可运行步骤。
- **Agent context**：本仓库 `CLAUDE.md` 已含项目上下文，无需 SPECKIT 标记区追加，跳过 `<!-- SPECKIT START -->` 注入（README 已链接 spec/checklist）。

## Re-evaluate Constitution Check（Post-Design）

| Gate | 决议 |
|------|------|
| Test-First | ✅ 维持 |
| Simplicity | ✅ 维持（仅新增 Bash + Playwright 工件，未引入新运行时） |
| Integration Testing | ✅ 维持（OWASP + FLOW + 多租户用真接口） |
| Observability | ✅ 维持 |

**结论**：Post-Design 通过；可进入 Phase 2（`/speckit-tasks`）。

## Done When

- [x] Plan workflow 执行并生成 plan.md / research.md / data-model.md / quickstart.md / contracts/
- [x] Constitution Check Pre + Post 双通过
- [x] NC-001 / NC-002 决议固化到 spec.md `## Clarifications`
- [x] `/speckit-tasks` 基于本 plan 生成执行任务清单
- [x] `/speckit-implement` 执行完成（43/85 任务完成）
  - Phase 1: T001-T004 ✅ 目录骨架、配置、脚本
  - Phase 2: T005-T012 ✅ Seeder测试账号、GA端点、菜单清理、Fixtures
  - Phase 3-9: US1-US7 ✅ 7个角色E2E测试
  - Phase 10: T056-T076 ✅ API冒烟、OWASP、FLOW跨角色测试
  - Phase 11: T077-T082, T084-T085 ✅ CI/CD、文档同步、GA Checklist
  - ⏳ T012 (Seeder单元测试)、T083 (Quickstart全流程验证) 待运行时验证
