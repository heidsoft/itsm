# GA Readiness Checklist: 角色驱动的全产品测试

**Purpose**: ITSM v1.0 GA 准入"可上线"质量门禁，按角色 × 流程 × 风险三维实施。

**Created**: 2026-06-13

**Feature**: [spec.md](../spec.md) · [docs/ITSM-基于角色视角的产品测试方案.md](../../../docs/ITSM-基于角色视角的产品测试方案.md)

## P1 角色（必须 100% 通过）

- [ ] CHK001 end_user：`EU-01..EU-10` 全部通过（提单/查询自有/知识库/评分/越权拒绝）
- [ ] CHK002 engineer：`EN-01..EN-10` 全部通过（接单 → 解决，事件→问题→变更，AI 采纳）
- [ ] CHK003 super_admin：`SA-01..SA-10` 全部通过（菜单 ≥80、租户 CRUD、角色/权限、连接器生命周期、GA 12/12 ready）
- [ ] CHK004 跨角色 `FLOW-1` 标准事件闭环 100% 通过
- [ ] CHK005 跨角色 `FLOW-3` 事件→问题→变更链路 100% 通过
- [ ] CHK006 跨角色 `FLOW-9` 多租户隔离零跨租户泄露

## P2 角色（≥90% 通过）

- [ ] CHK010 approver：`AP-01..AP-06` ≥90%
- [ ] CHK011 sd_manager：`SD-01..SD-07` ≥90%
- [ ] CHK012 security：`SE-01..SE-06` ≥90%
- [ ] CHK013 tenant_admin：`TA-01..TA-06` ≥90%

## 自动化基线

- [ ] CHK020 后端 `go test ./...` 100% 通过（17 包）
- [ ] CHK021 前端 `npm run test:unit` 100% 通过（≥526 用例）
- [ ] CHK022 前端 `lib/api/*` `lib/services/*` 覆盖率 ≥60%
- [ ] CHK023 TypeScript type-check 0 错误
- [ ] CHK024 ESLint 0 errors，warnings ≤500
- [ ] CHK025 后端 `go build ./...` 0 错误
- [ ] CHK026 前端 `npm run build` 0 错误
- [ ] CHK027 `docs/scripts/smoke-api.sh` 落地并 CI 跑通（≥25 端点 `code=0`）
- [ ] CHK028 Playwright 角色 E2E `tests/e2e/roles/*.spec.ts` 全部通过

## 安全（OWASP Top 10）

- [ ] CHK030 A01 越权：end_user 调用管理 API 返回 403
- [ ] CHK031 A02 加密：JWT 任意位篡改 → 401
- [ ] CHK032 A03 注入：title 含 `' OR 1=1 --` / `<script>` 持久化但渲染转义
- [ ] CHK033 A05 配置：生产 `SERVER_MODE=release`、CORS 白名单显式、`REDIS_PASSWORD` 必填或 fallback 透明
- [ ] CHK034 A07 认证：暴力登录 5 次 → 限流 + 审计
- [ ] CHK035 A08 完整性：篡改请求 `tenant_id` 不生效
- [ ] CHK036 A09 日志：登录/状态变更/审批/角色/租户/连接器全留痕
- [ ] CHK037 A10 SSRF：连接器 webhook 内网 URL 被拒绝

## 性能基线

- [ ] CHK040 工单列表 P95 < 300ms @ 50 RPS
- [ ] CHK041 工单创建 P95 < 500ms @ 20 RPS
- [ ] CHK042 dashboard overview P95 < 600ms
- [ ] CHK043 AI Triage P95 < 3s（独立基线）
- [ ] CHK044 24h soak：错误率 < 0.1%，无内存泄漏

## 准入与可观测

- [ ] CHK050 `/api/v1/health` 200ms 内 200
- [ ] CHK051 `/api/v1/readiness/ga` 12/12 ready
- [ ] CHK052 P0 缺陷 = 0
- [ ] CHK053 P1 缺陷 ≤ 2 且具备 workaround
- [ ] CHK054 风险登记 R-01..R-08 状态全部 `closed` 或具备 mitigation

## 部署与镜像

- [ ] CHK060 重新打包并 push GA 版前端镜像（替换 6/9 的旧产物）
- [ ] CHK061 `itsm-nginx-prod` 容器不再 Restarting
- [ ] CHK062 `docker-compose.prod.yml` 启动时显式 `--env-file .env.prod`
- [ ] CHK063 默认 seed 补 `engineer1 / manager1 / tenant1admin` 测试账号

## 菜单一致性（NC-001 / NC-002 决议）

- [ ] CHK080 前端 `menu-config.ts` 移除 `/reports/cmdb-quality`、`/reports/sla-trend`、`/reports/incident-trends`、`/reports/change-success`、`/reports/problem-efficiency`、`/tickets/templates` 等无后端 API 的孤立子项
- [ ] CHK081 顶级菜单 `/reports`（DB id=10）同步下架或在前端隐藏，post-GA 路线图记录补建报表
- [ ] CHK082 菜单加载以 DB `menus` 表为权威源；前端 `menu-config.ts` 与 DB diff 收敛为零
- [ ] CHK083 §六冒烟矩阵中 `/api/v1/process-instances` 修正为 `/api/v1/bpmn/process-instances` 与 `/api/v1/workflow/instances`
- [ ] CHK084 `/api/v1/templates/tickets`、`/api/v1/incident-rules` 从 GA 测试矩阵移除并写入 post-GA backlog

## 模板化与可复跑

- [ ] CHK070 CI 工作流：`go test` + `npm test` + 启后端跑冒烟
- [ ] CHK071 缺陷追踪表按 P0/P1/P2/P3 分级归档
- [ ] CHK072 性能基线 k6/wrk 报告归档
- [ ] CHK073 安全 OWASP 扫描报告归档

## Notes

- 编号 `CHK0XX` 与 spec.md 的 `FR-XXX` / `SC-XXX` 一一对应：FR=能力存在性，SC=可度量阈值，CHK=GA 实施动作。
- `EU- / EN- / SA- / AP- / SD- / SE- / TA-` 测试 ID 与跨角色 `FLOW-1..10` 详见源文档第四、五章。
- 实施顺序建议：自动化基线（CHK020-028）→ P1 角色（CHK001-006）→ P2 角色（CHK010-013）→ 安全（CHK030-037）→ 性能（CHK040-044）→ 准入（CHK050-054）→ 部署（CHK060-063）。
