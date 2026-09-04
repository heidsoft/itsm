# 🛣️ ITSM Roadmap

> **Source of truth for what is shipping, what is shipping next, and what
> is parked.** Updated as part of every release. Last synced: 2026-09-04.
>
> Cross-references:
> - PRD library: [docs/prd/](./docs/prd)
> - v1.0 GA readiness: [docs/v1-ga-readiness.md](./docs/v1-ga-readiness.md)
> - Architecture: [docs/architecture/](./docs/architecture)
> - Open issues & milestones: GitHub [Issues](https://github.com/heidsoft/itsm/issues) and [Projects](https://github.com/heidsoft/itsm/projects)

---

## 🎯 North Star

**Become the de-facto open-source AI-Native ITSM for enterprises that need
ServiceNow-class workflows without the lock-in or the footprint.**

Concretely that means:
1. **Process completeness** across the ITIL core, measured by executable business journeys rather than menu count.
2. **AI that earns its seat** — classification, summarization, RAG, and
   impact analysis that are measurable, not vibes.
3. **Native integration surface** — Feishu / DingTalk / WeCom / Webhook
   ship as first-class connectors, not bolt-ons.
4. **Operational discipline** — coverage, observability, security, and
   release hygiene as defaults, not afterthoughts.

---

## 📅 Release Timeline

| Version | Target | Theme | Status |
|:---|:---|:---|:---|
| **v1.0 GA** | 2026-Q2 | ITIL core + AI-Native scaffolding + private deploy | ✅ Shipped |
| **v1.6.x**   | 2026-Q3 | TicketType platform + reliability + RBAC/tenant hardening | 🟡 In progress |
| **v1.7**     | 2026-Q4 | Connector productionization + AI evaluator + business E2E | 🟢 Planned |
| **v2.0**     | 2027-Q2 | Coverage 70% + AI auto-triage GA + MSP billing + multi-region | 🔵 Roadmap |
| **v3.0**     | 2027-Q4 | Self-hostable AI inference + Plugin marketplace v2 + agent ecosystem | ⚪ Parked |

---

## 🟢 v1.0 GA — Shipped (2026-Q2)

**Theme:** Get the foundation right.

### Capability

- [x] **ITIL core flows** — ticket / incident / problem / change / release / service request
- [x] **Service catalog** — request templates, approval routing, SLA binding
- [x] **BPMN workflow engine** — process definitions, instances, user tasks,
      variable persistence, candidateGroups-driven approval (replaces the
      old dual-track approval system)
- [x] **CMDB v1** — CI types, configurations items, relationships, impact
      analysis, cloud discovery scaffold
- [x] **Knowledge base** — articles, versioning, RAG retrieval
- [x] **SLA** — multi-level policies, escalation matrix, alert rules
- [x] **AI capabilities (scaffold)** — Guidance-Harness-Skill framework,
      LLM Gateway, Triage / Summarize / KB skills
- [x] **RBAC + multi-tenant** — roles, permissions, menu gating, MSP mode
- [x] **Deployment** — Docker Compose (private / saas / saas_msp), GHCR
      images, multi-platform Release zip

### Quality

- [x] GA gate (4 checks): backend tests, frontend build, compose health,
      E2E smoke (11 core APIs)
- [x] Staticcheck + gofumpt + ESLint + tsc
- [x] Dependabot weekly scans
- [x] Security policy + Code of Conduct

### 后续持续治理项

- 🟡 关键业务旅程的服务层、集成与 E2E 覆盖继续提升。
- 🟡 超大 Controller 按现有领域边界渐进拆分，避免形成第二套接口。
- 🟡 连接器从生命周期框架推进到真实渠道生产验收。

---

## 🟡 v1.6.x — In Progress (2026-Q3)

**Theme:** Cover the seams and harden the foundation.

### 已落地

- [x] **TicketType 平台化** — 类型持久化、动态字段、创建快照、Preset Library、归档恢复和管理 UI。
- [x] **统一绑定解析** — Ticket 创建从已解析 TicketType 执行 Workflow、SLA 与 Assignment。
- [x] **权限与审计** — TicketType 独立管理/归档/Preset 安装权限，ACL manifest 覆盖；Preset 安装、归档恢复和绑定变更独立审计。
- [x] **可靠异步执行** — 工单与事件的流程启动进入持久化 command/outbox；关键通知具备 outbox、租约、重试和死信基础。
- [x] **租户与输入防线** — 覆盖跨租户、禁用类型、非法动态字段与非法绑定引用的回归测试。
- [x] **发布与安全加固** — HttpOnly cookie、初始化 migration ledger、PostgreSQL RLS、Endpoint ACL、依赖与运行时安全基线。
- [x] **分层迁移收尾** — legacy controller 全部退役，新代码统一进入 `handlers/<domain>` 垂直分层；swagger 路由冲突、菜单/认证契约断裂等收敛问题清零。
- [x] **状态机与错误语义加固** — 变更状态推进 CAS 并发防护；问题/事件状态机违规返回 409 业务语义而非 500；`super_admin` 通配权限链路（登录 / `/auth/me` / 前端判定）对齐。
- [x] **可靠执行补强** — commandbus 对聚合已删除的命令立即死信；审批链收口 SQL 缺列修复；业务流程回归套件（63 项集成 + 27 项生命周期深度）全绿。
- [x] **开源体验补强** — `make dev-seed-demo` 一键演示数据集（事件/问题/变更/知识库，幂等），README 快速开始接入；产品定位明示 **v1.6.x 界面中文优先**，完整界面 i18n 规划至 v1.7。

### 当前收敛项

- [ ] **业务旅程 E2E** — 固化 TicketType 安装与绑定、工单创建、Workflow 实例/任务/历史、SLA、Assignment、审计的完整断言。
- [ ] **可靠执行统一** — 将剩余 ITIL 域从非可靠触发路径迁移到 command/outbox，并提供积压、重放和死信运维。
- [ ] **生产数据升级门禁** — 对每次 Schema 变化执行脱敏 PostgreSQL 副本迁移、兼容、回滚与耗时验证。
- [ ] **Connector marketplace 生产化** — Feishu、DingTalk、WeCom、Webhook 的真实渠道健康检查、验签、重放与密钥治理。
- [ ] **AI Audit/Evaluator** — 对建议保留接受/拒绝反馈，并形成可重复的质量基线。
- [ ] **CMDB 数据治理** — 发现 Job、Diff、调和、退役、质量指标与规模测试。

### 发布门禁

- [x] 后端全量测试、静态分析、前端类型检查/构建、API 契约和 Endpoint ACL 均有自动化入口。
- [ ] 每个候选版本保留 Git SHA、镜像 digest、数据库版本、迁移结果、E2E 与恢复演练证据。
- [ ] 生产放行继续按部署环境验收，不能由仓库中的历史“全部通过”报告替代。

---

## 🟢 v1.7 — Planned (2026-Q4)

**Theme:** AI earns its seat, integrations go live.

### Engineering

- [ ] **AI Evaluator v1** — classification accuracy ≥85%, summarization
      ROUGE ≥0.6, RAG hit-rate ≥70%. Regression suite in CI.
- [ ] **AI telemetry** — capture prompt/response/cost/latency for every
      skill invocation; dashboard at `/api/v1/ai/audit`.
- [ ] **Knowledge base RAG v2** — chunking strategy improvements,
      re-ranking, hybrid search (BM25 + vector).
- [ ] **Skill registry v1** — declarative skill manifests, hot-pluggable
      pipeline, registry UI.

### Product

- [ ] **Feishu / DingTalk / WeCom native connectors** — end-to-end:
      account / approval / IM notification / webhook relay.
- [ ] **Auto-triage (human-in-the-loop)** — AI suggests category,
      assignee, SLA tier; engineer accepts with one click.
- [ ] **SLA forecast skill** — predict SLA breach risk per ticket,
      surface on dashboards.
- [ ] **Full UI i18n** — the UI is Chinese-first through v1.6.x; extract
      the remaining hardcoded strings (currently ~83% of pages) into
      `src/lib/i18n` message catalogs and ship an en-US locale with a
      per-user language switch.

### Quality

- [ ] **Backend coverage** 40% → **55%** overall.
- [ ] **Performance budgets** — k6 baselines for top 10 endpoints,
      enforced in CI.
- [ ] **Trivy + govulncheck** — daily scans, high-severity blockers.

---

## 🔵 v2.0 — Roadmap (2027-Q2)

**Theme:** MSP-friendly, AI-assisted, multi-region.

### Engineering

- [ ] **Coverage 55% → 70%**.
- [ ] **Service decomposition** — split monolithic `itsm-backend` into
      `core` + `workflow` + `ai` + `cmdb` services along bounded contexts.
- [ ] **Event-driven architecture** — Watermill is already in deps;
      promote to first-class pub/sub for incident events.
- [ ] **Multi-region active-active** — Redis Streams + region-aware
      routing.

### Product

- [ ] **MSP billing** — usage metering, invoicing, allocation reports.
- [ ] **AI auto-triage (full)** — replaces the human-in-the-loop step
      from v1.7 with confidence-based auto-accept.
- [ ] **Impact analysis skill** — given a change, predict affected CIs,
      tickets, and downstream SLAs.
- [ ] **Plugin marketplace v2** — signed plugins, sandboxed execution,
      revenue share for authors.

### Quality

- [ ] **SOC 2 Type II readiness** — control mapping, evidence collection,
      audit-ready logging.
- [ ] **Customer-managed keys (BYOK)** for LLM Gateway.

---

## ⚪ v3.0 — Parked (2027-Q4)

**Theme:** Self-hostable AI, agent ecosystem.

- Self-hostable LLM inference (Ollama, vLLM, llama.cpp) — drop the
  external OpenAI dependency for privacy-sensitive deployments.
- Agent marketplace — third-party agents that can act on the ITSM
  data model under strict RBAC.
- Mobile PWA with offline-first ticket intake.
- Multilingual UI (zh-CN baseline; en-US, ja-JP, ko-KR planned).

---

## 🛠️ Always-On Tracks

These don't belong to a single release; they ship incrementally:

### Testing & Quality

- Incremental coverage gate (60% on new code) — landed
- End-to-end smoke on every PR — landed v1.0
- Frontend visual regression — planned v1.7
- Property-based tests for critical parsers (BPMN XML, RAG chunking)
  — planned v1.7

### Security

- CodeQL + Trivy + govulncheck — landed
- Quarterly threat-model review
- Annual pen-test

### Open-Source Governance

- Issue triage SLA (48h first response, 14d close-or-fix) — ongoing governance target
- Monthly community digest
- Quarterly maintainer rotation review

### Developer Experience

- `make dev-*` unified dev environment (already landed v1.0)
- `itsm-cli` for ops (deploy/seed/inspect) — landed v1.0
- `itsm-skill` for OpenClaw / Codex agents — landed v1.0
- Container image size reduction (distroless base) — planned v1.7

---

## 📊 Key Metrics

We track these on every release. Numbers below are post-v1.0 GA baseline
and the **target** for the next major release.

| Metric | Historical v1.0 baseline | v1.7 target | v2.0 target |
|:---|---:|---:|---:|
| Backend coverage | ~2% | 55% | 70% |
| Frontend coverage | ~10% (UI only) | 30% | 60% |
| E2E smoke coverage | 11 APIs | 25 APIs | 50 APIs |
| Mean PR → first review | TBD | < 48h | < 24h |
| Mean issue → first response | TBD | < 48h | < 24h |
| AI triage accuracy | — | 85% | 92% |
| Open stale issues | varies | < 30 | < 15 |

---

## 🤝 How to Influence the Roadmap

1. **File an issue** with the `feature-request` template and link to
   the milestone you think it belongs in.
2. **Vote** on issues with 👍 — we sort milestone backlogs by reactions.
3. **Propose a major change** via the RFC process:
   `docs/rfcs/0000-template.md`.
4. **Pick up a "good first issue"** — every track has at least one.

---

## 📜 Changelog

Major releases are tracked in [CHANGELOG.md](./CHANGELOG.md) and via
GitHub [Releases](https://github.com/heidsoft/itsm/releases).
