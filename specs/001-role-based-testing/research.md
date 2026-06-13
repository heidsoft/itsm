# Phase 0 — Research: ITSM v1.0 GA 角色驱动的全产品测试方案

**Feature**: 001-role-based-testing
**Date**: 2026-06-13
**Status**: Complete

## 目的

把 spec.md 与源文档 `docs/ITSM-基于角色视角的产品测试方案.md` 中尚未明确的"实现策略"决议落地，给 Phase 1 / Phase 2 提供事实基线。

研究问题来自 plan.md / spec.md 的隐式技术决策点：

- R1：角色账号补建走 seeder 还是脚本？
- R2：菜单一致性以哪一侧为权威源？
- R3：GA Readiness 端点字段如何对齐真实 GET 路由？
- R4：CI 选型是单 job 串行还是多 job 并行？
- R5：OWASP A01-A10 测试落到哪一层？
- R6：Playwright 角色 spec 的最小可运行结构是什么？
- R7：API 冒烟脚本退出码与失败汇总策略？

---

## R1 — 角色账号补建策略

**Decision**：seeder 补丁 + 文档化前置脚本"双轨"。

**Rationale**：

- spec FR-604 要求 `engineer / manager / tenant_admin` 业务角色可"通过 seed 或脚本快速创建"。
- `pkg/seeder/seeder.go` 已有 admin/user1/security1 默认账号，是最低摩擦的扩展点。
- E2E 在 CI 上需要可重复登录的账号，写死在 seeder 比脚本更稳。
- 但脚本仍保留，便于已部署环境补账号而无需重启。

**Implementation**：

- 在 `pkg/seeder/seeder.go` 中新增三个测试账号：
  - `engineer1 / eng123` 挂 `engineer` 角色
  - `manager1 / mgr123` 挂 `manager`（approver）角色
  - `tenant1admin / ta123` 挂 `tenant_admin` 角色
- 脚本路径 `docs/scripts/seed-test-users.sh`（可选）通过 `/api/v1/users` + `/api/v1/roles/:id/users` 创建。

**Alternatives Considered**：

- 仅脚本：CI 必须每次执行额外步骤；且与 seeder 的默认账号体系不一致。
- 仅 seeder：已部署的旧库无法再追加账号；放弃。

---

## R2 — 菜单一致性权威源

**Decision**：DB `menus` 表为唯一权威源；前端 `menu-config.ts` 仅作开发兜底，diff 收敛为零。

**Rationale**：

- spec Clarifications session 2026-06-13 中两次明确"以 DB 为权威源"。
- 实际查询：DB `menus` 表 25 条记录，顶级 `/reports` 无子项；前端静态 fallback 多出 `/reports/cmdb-quality` 等孤立子项。
- 以 DB 为权威源符合 RBAC 多租户隔离要求（每个 tenant 可定制菜单）。

**Implementation**：

- GA 前清理 `itsm-frontend/src/components/layout/sidebar/menu-config.ts` 中以下子项：
  - `/reports/cmdb-quality`
  - `/reports/sla-trend`
  - `/reports/incident-trends`
  - `/reports/change-success`
  - `/reports/problem-efficiency`
  - `/tickets/templates`
- 顶级 `/reports`（DB id=10）无子菜单时在前端隐藏（不下架 DB 记录），post-GA 路线图补建报表后再启用。
- E2E 增加一项断言：`/api/v1/auth/menus` 返回项数 == 当前租户在 DB 中可见菜单数。

**Alternatives Considered**：

- 前端为权威源：违反多租户配置，已被用户明确否决。
- 双向校验脚本：增加维护负担且不解决"哪边对"的根本问题。

---

## R3 — GA Readiness 端点字段对齐

**Decision**：修订 `router/ga_readiness.go` 中 `Endpoint` 字段为可达 GET 路径；POST-only 端点改填 README/文档链接。

**Rationale**：

- 风险登记 R-01：当前若干 readiness 模块的 `Endpoint` 字段填的是 POST-only 路径（如 `/api/v1/sla/monitoring`、`/api/v1/ai/audit`），运维 / SRE 直接 GET 探活会得到 405，造成误报。
- Readiness 端点本身应给"可探活的最小 URI"，不必把功能 API 全部塞进去。

**Implementation**：

- SLA 模块 `Endpoint` 改为 `/api/v1/sla/definitions`（GET 可达）。
- AI 模块 `Endpoint` 改为 `/api/v1/ai/metrics`（GET 可达）。
- 其余 10 个模块逐一核对，必须满足 `curl -fsS $base$endpoint` 返回 200/2xx。
- 若模块无 GET 接口，则填模块 health 子路由如 `/api/v1/health`，并在 `Notes` 字段注明。

**Alternatives Considered**：

- 升级 readiness 端点支持 POST 探活：扩散面太大，违反 Simplicity gate。

---

## R4 — CI 选型与拓扑

**Decision**：GitHub Actions 单 job 串行；PG + Redis 走 services 容器；后端二进制 + 前端 dev server 后台启动。

**Rationale**：

- 仓库现有 `.github/workflows/` 与 docker-compose；新增工作流不引入新运行时。
- 串行 < 多 job 复杂度，"启后端 → 冒烟 → 启前端 → Playwright" 的依赖链在单 job 内最易表达。
- E2E 阶段对启动顺序敏感，多 job 并行需要 artifact passing，得不偿失。

**Implementation**：

```yaml
# 节选思路
jobs:
  ga-readiness:
    runs-on: ubuntu-latest
    services:
      postgres: { image: postgres:15, env: { POSTGRES_PASSWORD: itsm_password_2026 } }
      redis:    { image: redis:7 }
    steps:
      - checkout
      - setup-go (1.22)
      - go test ./...
      - setup-node (20)
      - npm ci && npm run test:unit
      - go build && start backend (background)
      - bash docs/scripts/smoke-api.sh
      - npm run dev (background)
      - npx playwright test tests/e2e/roles
```

**Alternatives Considered**：

- 多 job + matrix：并行增益 < 维护成本。
- 自托管 runner：超出 GA 范围。

---

## R5 — OWASP A01-A10 落点

**Decision**：

| 类别 | 落点 |
|------|------|
| A01 越权 | 角色 E2E（end-user / security spec）|
| A02 加密 | 角色 E2E（任意 spec 中加一组 token 篡改）|
| A03 注入 | 冒烟脚本（提单 title 注入串）|
| A05 配置 | 冒烟脚本（启动期 env 校验，单独 case）|
| A06 已过期组件 | 暂不在 GA 范围（依赖 SBOM/扫描，post-GA）|
| A07 认证 | 冒烟脚本（暴力登录 5 次 → 限流）|
| A08 完整性 | 角色 E2E（tenant-admin spec 篡改 tenant_id）|
| A09 日志 | 冒烟脚本（关键操作后 GET `/audit-logs` 校验）|
| A10 SSRF | 冒烟脚本（连接器 webhook 内网 URL → 拒绝）|

**Rationale**：

- E2E 适合"用户视角越权 / 横向越权"这类需要 token + 行为流的场景。
- 冒烟脚本适合"一次 curl 即可"的注入 / 限流 / SSRF 等。
- A06 需要外部扫描器，与本 feature 的"测试方案"边界冲突，列入 post-GA。

**Alternatives Considered**：

- 全部塞进 E2E：单测试时间 >5min，违反"快速反馈"原则。
- 全部塞进冒烟：A01/A08 跨角色测不出。

---

## R6 — Playwright 角色 spec 最小结构

**Decision**：每份 spec 共享 `tests/e2e/fixtures/auth.ts` 注入登录 token，按角色目录分文件。

**Rationale**：

- 现有 `tests/e2e/comprehensive-e2e.spec.ts` 13 项已通过 → 复用 Playwright 配置。
- 角色 spec 之间相互独立可单跑（FR Independent Test 要求）。
- 共享 auth fixture 减少重复登录代码。

**Implementation**：

```typescript
// tests/e2e/fixtures/auth.ts
import { test as base } from '@playwright/test';
type Roles = 'admin' | 'user1' | 'security1' | 'engineer1' | 'manager1' | 'tenant1admin';
export const test = base.extend<{ login(role: Roles): Promise<string> }>({
  login: async ({ request }, use) => {
    await use(async (role) => {
      const passwords = { admin: 'admin123', user1: 'user123', security1: 'security123', engineer1: 'eng123', manager1: 'mgr123', tenant1admin: 'ta123' };
      const r = await request.post('/api/v1/auth/login', { data: { username: role, password: passwords[role] }});
      return (await r.json()).data.access_token;
    });
  },
});
```

```text
tests/e2e/roles/
├── end-user.spec.ts        // EU-01..EU-10
├── engineer.spec.ts        // EN-01..EN-10
├── super-admin.spec.ts     // SA-01..SA-10
├── approver.spec.ts        // AP-01..AP-06
├── sd-manager.spec.ts      // SD-01..SD-07
├── security.spec.ts        // SE-01..SE-06
└── tenant-admin.spec.ts    // TA-01..TA-06
```

**Alternatives Considered**：

- 单文件 7 个 describe：违反"独立可交付"原则。
- 用 storage state：与多角色切换冲突。

---

## R7 — 冒烟脚本退出码与失败汇总

**Decision**：bash `set -euo pipefail` + 累加失败计数，最后一次性返回非零。

**Rationale**：

- 单接口失败立即退出 → CI 看不到"还有几个会失败"。
- 完整跑完 → 收集所有失败 → 一次报告，方便修复。

**Implementation**：

```bash
#!/usr/bin/env bash
set -uo pipefail
fails=0
check() {
  local name=$1 method=$2 path=$3 expect=${4:-200}
  local code=$(curl -s -o /tmp/body -w "%{http_code}" -X "$method" "$BASE$path" -H "Authorization: Bearer $TOKEN")
  if [[ "$code" != "$expect" ]]; then
    echo "[FAIL] $name $method $path -> $code (expect $expect)"; fails=$((fails+1))
  else
    echo "[ OK ] $name $method $path -> $code"
  fi
}
# ... 25+ check() 调用 ...
if [[ $fails -gt 0 ]]; then echo "$fails endpoint(s) failed"; exit 1; fi
echo "All endpoints OK"
```

**Alternatives Considered**：

- 用 Newman/Postman：引入新依赖，违反 Simplicity gate。
- 用 Go test：与"≥25 端点的可读矩阵"诉求不匹配，bash + curl + jq 反而最透明。

---

## 依赖与风险确认

| 项 | 状态 |
|----|------|
| Go 1.22 / Node 20 在 ubuntu-latest 默认可用 | ✅ |
| PostgreSQL 15 / Redis 7 services 已在 GHA 标准库 | ✅ |
| Playwright 1.48+ 已在前端 devDependencies | ✅ |
| `jq` 在 ubuntu-latest 默认安装 | ✅ |
| `curl` 在 ubuntu-latest 默认安装 | ✅ |
| MiniMax LLM 在 CI 中不可用 → AI Triage 走降级 | ⚠ 已在 spec FR-501/SC-304 标注 |

无未决事项，可进入 Phase 1。
