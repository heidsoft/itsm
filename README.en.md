<div align="center">

# AI-Native ITSM

An open-source IT service management system for enterprises, covering ITIL core processes with BPMN workflow orchestration, CMDB, SLA, knowledge base, and multi-tenancy.

[![Go](https://img.shields.io/badge/Go-1.25.13-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

**[简体中文](./README.md)** · **English**

</div>

> **Translation status**: `README.md` (Simplified Chinese) is the single source of truth for project content; this file is its English counterpart and must stay in sync with it — section structure, the capability/maturity table, and every command/tool count. If the two disagree, the Chinese README wins and this file is the one to fix. Product UI text is Chinese-first through v1.6.x (see the language note under Quick Start). Terminology is fixed throughout this file: Preview / Available for maturity states, knowledge capture for 知识沉淀, fulfillment for 服务请求交付, extension delivery for 扩展交付, and Alibaba Cloud / Tencent Cloud / AWS / Volcengine / Kubernetes for the five discovery providers. US spelling is used consistently (for example `fulfillment`, not `fulfilment`).

## Table of Contents

- [Overview](#overview)
- [Use Cases](#use-cases)
- [Capabilities & Maturity](#capabilities--maturity)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Docker Development](#docker-development)
  - [Verify Startup](#verify-startup)
  - [Local Hot-Reload Development](#local-hot-reload-development)
  - [Optional AI & Monitoring Components](#optional-ai--monitoring-components)
- [Usage Examples](#usage-examples)
  - [API Examples](#api-examples)
  - [Common Development Commands](#common-development-commands)
  - [MCP Integration](#mcp-integration-let-an-agent-drive-itsm)
  - [Common Scenarios](#common-scenarios)
- [Core Business Flows](#core-business-flows)
- [Reliable Execution Architecture](#reliable-execution-architecture)
- [Pluggable Integrations](#pluggable-integrations)
- [Screenshots](#screenshots)
- [Tech Stack & Repository Structure](#tech-stack--repository-structure)
- [Development & Testing](#development--testing)
- [Production Deployment](#production-deployment)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Star History](#star-history)
- [License](#license)

![ITSM Dashboard](./docs/images/01-仪表盘.png)

## Overview

This project aims to provide an enterprise ITSM system that actually works. Rather than piling up feature menus, it connects tickets, incidents, problems, changes, SLA, CMDB, and knowledge base into complete business processes, with audit trails, permission control, and multi-tenant isolation.

Key design choices:

- **BPMN for workflow orchestration**: Uses BPMN 2.0 standard for approvals and workflows, no custom engine
- **CMDB is more than an asset table**: Configuration items and relationships feed into incident, change, and other processes for impact analysis
- **AI as assistant, not replacement**: Triage, summarization, and knowledge retrieval can use AI, but must degrade gracefully, have audit records, and never bypass human approval
- **Reliable async operations**: Workflow triggers, notifications, and other critical operations use transaction + outbox pattern, not goroutine fire-and-forget

> **Current version v1.6.x**: In production hardening phase. Core ITIL processes (tickets, incidents, problems, changes, SLA, CMDB) are available, but module maturity varies. Some features are still in Preview stage. Check the [open-source capability statement](./docs/product/open-source-release-capability.md) before production use.

## Use Cases

- IT service desk: unified intake, dispatch, and tracking of employee requests
- Operations governance: end-to-end management of incidents, CIs, SLA, problems, and changes
- Process automation: configure approvals and cross-system workflows via BPMN
- Multi-organization services: multi-tenant management for private, SaaS, or MSP deployments
- Custom development: extend with Go + Next.js and open APIs

## Capabilities & Maturity

| Capability | Status | Description |
|:---|:---:|:---|
| Tickets & Incidents | Available | Status transitions, dispatch, SLA, BPMN binding, tenant isolation |
| Ticket Types & Dynamic Forms | Available | Custom fields, form presets, Workflow/SLA bindings |
| Change Management | Available | Risk assessment, approval chains, rollback plans, PIR. Submission gates are closed (critical CIs / open incidents force `high` risk plus window-conflict blocking); the approval path is atomic and quorum/threshold/multi-level chains take effect on the approval endpoint (returns `pending` below threshold), with terminal transitions in the same transaction. Outcomes are recorded truthfully (a fail endpoint plus a mandatory failure/rollback reason written into `outcomeNote`; PIR only opens for completed/rolled_back/failed), and rollback is verifiable step by step (done/skipped/failed with note, confirmer and timestamp) — it cannot be marked done while steps are unconfirmed, and the checklist locks once implementation starts |
| Problems & Known Errors | Available | Root cause analysis, workarounds, linked incidents, SLA (same engine as tickets/incidents), comments, knowledge capture |
| Service Catalog & Requests | Preview | Catalog management, request approvals, service tasks. Approvals are orchestrated solely by BPMN (three-level approval gateway plus an approval-plan variable), and catalog items that need no approval are fulfilled directly. Approval actions are gated by the current node's approver set — users outside that set are not shown an approval button. Fulfillment follows "failure is terminal" — execution is claimed atomically (repeat execution returns 409 instead of double-delivering), failures are retryable (capped at 5 attempts), and manual adjudication is recorded. Fulfillment is wired to the real Alibaba Cloud ECS provider (credentials referenced via `secret://` from the tenant secret store; requests fail closed when unconfigured), but only ECS is covered and rollback compensation for partial failure is not implemented |
| CMDB | Available | CI types, configuration items, relationships, topology, impact analysis |
| CMDB Cloud Discovery | Preview | Auto-discovery across five resource types — Alibaba Cloud, Tencent Cloud, AWS, Volcengine, and Kubernetes; diffs update cloud resources then write back configuration items by cloud service type (unmapped types are skipped explicitly, CIs missing required attributes wait for manual completion, and vanished resources only mark sync stale without changing lifecycle). Each discovery round lists every page in full — there is no incremental watermark; discovery against real accounts has not been accepted yet |
| BPMN & Approvals | Preview | Process definitions, instances, tasks, variables, execution history; approvals are orchestrated solely by BPMN (there is no direct business-side approval path), driven by a three-level approval gateway plus an approval-plan variable, with quorum/threshold/multi-level chains taking effect on the approval endpoint; approval eligibility is decided server-side and returned with the response, so the UI renders approval actions only for eligible users — users outside the current level's approver set get no clickable approval button |
| SLA | Available | Policies, deadlines, warnings, violation statistics |
| Knowledge & RAG | Preview | Article management, keyword/vector search, Q&A fallback, problem→knowledge capture (`POST /problems/:id/knowledge`, drafts enter review), publish gate (cannot publish before review; passing adds to retrieval, rejection removes it) |
| AI Assistance | Preview | LLM Gateway, triage, summarization, RAG |
| Notifications & Connectors | Preview | In-app notifications (already Available), delivery audit, channel diagnostics; Feishu / WeCom / DingTalk connectors are all implemented (Preview — deployment readiness declared via READY env vars) |
| Marketplace & Extension Delivery | Available | Catalog items, version rows and publish/unpublish; connectors and skills are the deliverable types — installing a skill activates the runtime skill registered in this process via the version row's `skill://<code>` locator, and a miss reports `status=failed` with the available codes instead of a fake success; `POST /marketplace/items/:id/upgrade` covers both upgrade and rollback (target may be named, an older target rolls back, dependencies are re-checked against the target version, and a runtime that fails to start keeps the current version); the UI opens a dialog listing the item's published versions and states "changes from vX to vY" before a second confirmation, since switching versions reloads the runtime; `auto_upgrade` is an effective explicit switch executed periodically by the worker; `GET /marketplace/dependency-status` re-verifies prerequisites of every existing installation and keeps `declaration_missing` separate from "healthy"; **plugins are deliberately not delivered** — there is no host runtime, so install/upgrade returns `501` and publish returns `400`; skills ship with releases, while an artifact registry, signing and sandboxed execution are not implemented |
| RBAC / Multi-Tenancy | Available | Role permissions, endpoint ACL, tenant isolation, audit logs |
| Reporting & Operations | Preview | SLA and incident partial metrics, dashboard; a unified management-level view is still pending |

> **Maturity legend**: **Available** = core features complete, ready for production validation; **Preview** = feature is implemented but still being polished — evaluate limitations before production use. See module docs for detailed constraints and acceptance criteria.

## Quick Start

> **Language note**: through v1.6.x the product UI is **Chinese-first** (targeting China private-deployment scenarios). The English README covers onboarding; full UI localization (en-US etc.) is planned for v1.7 — see the [ROADMAP](ROADMAP.md).

### Prerequisites

- Docker Desktop or compatible Docker Engine / Compose
- Git
- Recommended: at least 4 CPU cores, 8 GB RAM

### Docker Development

The repository provides two Compose files. Do **not** run `docker compose up` without `-f` — there is no default `docker-compose.yml` at the repo root:

| File | Purpose |
|:---|:---|
| `docker-compose.dev.yml` | Local development. Core services start by default; `--profile ai` adds Ollama, `--profile monitoring` adds Prometheus + Grafana |
| `docker-compose.prod.yml` | Production deployment. Requires `.env.prod`, includes nginx and background worker |

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm

cp .env.dev.example .env
make dev-start-docker
```

After startup, visit:

| Service | URL |
|:---|:---|
| Web | <http://localhost:3000> |
| Backend API | <http://localhost:8090> |
| Swagger | <http://localhost:8090/swagger/index.html> |
| MinIO Console | <http://localhost:9001> |

Development default credentials:

```text
Username: admin
Password: admin123
```

This account is for local development only. In production, the admin password is set by `ADMIN_PASSWORD` in your `.env.prod` (applied by the one-shot `itsm-init` container on first start). Never use documentation example passwords. Any deployment accessible by others must change the admin password, `JWT_SECRET`, database, Redis, and object storage credentials.

### (Optional) One-Command Demo Data

A fresh system ships with configuration templates only — no business records. Run the following to seed a demo dataset (8 incidents, 2 problems, 3 changes, 5 knowledge articles) so you can explore each module with realistic content:

```bash
make dev-seed-demo   # idempotent, safe to re-run
```

Demo records use fixed numbers (`INC-DEMO-xxxx` / `PRB-DEMO-xxxx` / `CHG-DEMO-xxxx`) covering the incident lifecycle (new → escalated → closed). Production deployments **never** seed these fictional records.

### Verify Startup

```bash
make dev-status
make dev-health

curl http://localhost:8090/api/v1/health
curl http://localhost:3000/api/health
```

View logs and stop the environment:

```bash
make dev-logs
make dev-stop-docker
```

Clean up volumes (this deletes local database and object storage data):

```bash
make dev-clean
```

### Local Hot-Reload Development

Docker provides PostgreSQL, Redis, and MinIO while running Go and Next.js natively:

```bash
make dev-start-local
make dev-status
```

See [Local Development Commands](./docs/dev-commands-reference.md) for native mode, PostgreSQL 17/pgvector, and proxy troubleshooting.

### Optional AI & Monitoring Components

The base development stack does not force-start Ollama or monitoring:

```bash
# Ollama
docker compose --env-file .env -f docker-compose.dev.yml \
  --profile dev --profile ai up -d

# Prometheus + Grafana
docker compose --env-file .env -f docker-compose.dev.yml \
  --profile dev --profile monitoring up -d
```

When no AI model is available, core ITIL workflows must keep running; AI capabilities must degrade according to configuration.

## Usage Examples

### API Examples

All APIs return a unified format `{ code: number, message: string, data: any }`.

```bash
# 1. Login and save HttpOnly session cookie (example account for local dev only)
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -c /tmp/itsm-cookies.txt \
  -d '{"username":"admin","password":"admin123"}'

# Login response does not expose access token to JavaScript.
# Write operations also require a double-submit CSRF token; examples below need jq.
CSRF_TOKEN=$(curl -s -b /tmp/itsm-cookies.txt -c /tmp/itsm-cookies.txt \
  http://localhost:8090/api/v1/csrf-token | jq -r '.data.csrfToken')

# 2. Query ticket types available for current tenant
curl -X GET "http://localhost:8090/api/v1/ticket-types?status=active&page=1&pageSize=20" \
  -b /tmp/itsm-cookies.txt

# 3. Create a ticket using the returned type ID; formFields must match that type's field definition
TICKET_TYPE_ID=1

curl -X POST http://localhost:8090/api/v1/tickets \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b /tmp/itsm-cookies.txt \
  -d '{
    "title": "Printer not working",
    "description": "Printer in 3F meeting room is broken",
    "priority": "medium",
    "ticketTypeId": '"$TICKET_TYPE_ID"',
    "category": "hardware",
    "formFields": {}
  }'

# 4. Query ticket list
curl -X GET "http://localhost:8090/api/v1/tickets?page=1&pageSize=10" \
  -b /tmp/itsm-cookies.txt

# 5. Create a change request
curl -X POST http://localhost:8090/api/v1/changes \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b /tmp/itsm-cookies.txt \
  -d '{
    "title": "Database upgrade plan",
    "description": "PostgreSQL 14 to 16 upgrade",
    "type": "standard",
    "riskLevel": "medium",
    "implementationPlan": "Blue-green deployment"
  }'

# 6. Query SLA policies (canonical route; /sla/policies is the legacy alias)
curl -X GET "http://localhost:8090/api/v1/sla/definitions?page=1&pageSize=20" \
  -b /tmp/itsm-cookies.txt
```

### Common Development Commands

```bash
# Backend development
cd itsm-backend
go run main.go                           # Start backend server
go test ./...                            # Run tests
go build -o itsm-backend main.go         # Build binary

# Frontend development
cd itsm-frontend
npm install                              # Install dependencies
npm run dev                              # Start dev server
npm run build                            # Production build

# Database migration
cd itsm-backend
go run -tags migrate main.go             # Run migration

# View API docs
open http://localhost:8090/swagger/index.html
```

### MCP Integration (Let an Agent Drive ITSM)

`itsm-cli/bin/itsm-mcp.js` is a zero-dependency MCP server (stdio / JSON-RPC) that exposes **352 commands** to clients such as WorkBuddy or Claude Desktop: **27 curated tools** plus `itsm_discover` / `itsm_describe` / `itsm_call`, which together cover the entire command surface. High-risk writes require explicit confirmation, and list responses are trimmed to 50 items by default (`meta` carries `total` and `truncated`; page with `limit`/`pageSize` when you need more).

> The command and tool counts above are taken from live output — `node itsm-cli/bin/itsm.js --json schema | jq .data.count` and `node itsm-cli/bin/itsm-mcp.js --describe`. Both grow as routes are added, so this document does not enumerate them.

```json
{
  "mcpServers": {
    "itsm": {
      "command": "node",
      "args": ["/path/to/itsm/itsm-cli/bin/itsm-mcp.js"],
      "env": { "ITSM_BASE_URL": "http://localhost:8090" }
    }
  }
}
```

See [ITSM MCP Server](./docs/tools/itsm-mcp.md) for startup, the security model, the tool list and troubleshooting.

### Common Scenarios

| Scenario | Operation |
|:---|:---|
| Create tenant | `POST /api/v1/tenants` — then create users under that tenant |
| Configure SLA | `POST /api/v1/sla/definitions` — create SLA policy, bind via `conditions` (priority, etc.) |
| Design workflow | Use the Workflow module in the frontend to design BPMN processes, bind to business objects |
| Manage CMDB | `POST /api/v1/cmdb/cis` — create configuration items, establish CI relationships |
| Knowledge search | `POST /api/v1/knowledge/search` — search knowledge articles |
| Ingest external alerts | After configuring `ALERT_SOURCE_CONFIG`, `POST /api/v1/alerts/sources/:source/ingest` to receive alert webhooks |

Full API documentation: [API Reference](./docs/api-reference.md).

## Core Business Flows

```mermaid
flowchart LR
    A[Alert or Report] --> B[Incident]
    B --> C[Link CIs & Impact Scope]
    C --> D[SLA & BPMN]
    D --> E[Resolution & Recovery]
    E --> F[Problem / Known Error / Knowledge]
    C --> G[Controlled Change]
    G --> H[Risk & Approval]
    H --> I[Implement / Verify / Rollback / PIR]
```

Four primary acceptance journeys:

1. **Incident Management**: Report → Incident → CI association → SLA → Workflow → Recovery → Audit
2. **Problem Management**: Recurring incidents → Problem → Known Error → Knowledge publication → RAG
3. **Change Management**: Change → Impact analysis → Risk → Approval → Implement/Rollback → PIR
4. **Service Request**: Catalog → Request → Three-level approval → Fulfillment → CI creation or modification

Journey 4 deserves a closer acceptance checklist — it is the one most likely to look healthy while actually being stuck:

- **Approval actions are conditional.** Whether the request detail page shows Approve / Reject is decided server-side by checking whether the signed-in user belongs to the current level's approver set, and is returned with the response. The requester and accounts outside that set see no approval button.
- **Out-of-scope calls get an actionable answer.** A non-approver calling the approval endpoint directly receives `403` with the reason (for example "you are not the assignee or a candidate for this task; check the approvers configured on the approval node"), instead of an uninformative server error.
- **Approvers are resolved, not defaulted.** The approval plan resolves candidates within the tenant by step role (manager / IT / security) and removes the requester; when nothing resolves it falls back to a super administrator and logs a warning, so the tenant must actually have users with those roles.
- **Fulfillment failures must suggest a next step.** When a provisioning task lands in `failed`, `errorMessage` says whether a parameter is missing, credentials are unconfigured, or the cloud provider rejected the call — fix the configuration and retry, or take over manually, rather than facing a generic "operation failed".

## Reliable Execution Architecture

Production uses three process roles from the same backend image:

- `itsm-init`: runs database migrations and initialization
- `ITSM_PROCESS_MODE=api`: serves HTTP/WebSocket
- `ITSM_PROCESS_MODE=worker`: executes async tasks (workflows, SLA checks, notifications, indexing, etc.)

```mermaid
flowchart TB
    UI[Next.js Web / Open API] --> API[Go / Gin API]
    API --> DOMAIN[ITIL Domain Services]
    DOMAIN --> TX[(Business Data + Operational Command)]
    TX --> WORKER[Lease + Heartbeat Worker]
    WORKER --> BPMN[BPMN]
    WORKER --> NOTICE[In-App Notifications / Enterprise Connectors]
    WORKER --> FUTURE[AI / CMDB Sync / Indexing]
    DOMAIN --> AUDIT[(Audit & History)]
    API --> REDIS[(Redis)]
    API --> OBJECT[(MinIO / S3)]
```

Critical async operations (workflow initiation, notification delivery, SLA violations, etc.) use transaction + outbox pattern for reliability, with retry, dead-letter, and delivery audit support.

## Pluggable Integrations

The system connects to external systems through a connector framework with configuration-driven registration:

| Extension Point | Current Implementation | Configuration |
|:---|:---|:---|
| Alert sources | Generic Webhook (supports Prometheus Alertmanager, PagerDuty, etc.) | `ALERT_SOURCE_CONFIG` pointing to YAML |
| Vector store | Milvus, Qdrant, PGVector, and in-memory keyword fallback | `VECTOR_STORE_CONFIG` pointing to YAML or inline config |

> When no vector store is configured, the system defaults to in-memory keyword search. Data is lost on restart — not suitable for standalone production use.

### Alert Source Integration

Configure YAML to declare alert sources, field mappings, and webhook signature parameters. The repository provides a Prometheus Alertmanager example:

```bash
cd itsm-backend
export ALERT_SOURCE_CONFIG=etc/alert-sources/prometheus-alertmanager.yaml
```

Ingestion endpoint: `POST /api/v1/alerts/sources/:source/ingest` (requires authentication and `alert:write` permission).

### Vector Store Configuration

```bash
cd itsm-backend
cp etc/vector-store/config.yaml.example etc/vector-store/config.yaml
export VECTOR_STORE_CONFIG=etc/vector-store/config.yaml
```

Configuration supports `${ENV_VAR}` expansion. When `fallback: true` is enabled, the system automatically falls back to keyword search when the primary store is unavailable.

## Screenshots

The ticket list is the service desk's main entry point: status, priority, SLA countdown, assignee
and inline actions in a single screen, with list/board/analytics view switching and bulk assignment.

![Ticket Management](./docs/images/02-工单管理.png)

| Incidents & Problems | Changes & CMDB |
|:---:|:---:|
| ![Incident Management](./docs/images/03-事件管理.png) | ![Change Management](./docs/images/06-变更管理.png) |
| ![Problem Management](./docs/images/04-问题管理.png) | ![CMDB](./docs/images/08-cmdb.png) |

| Service Catalog & Knowledge | Workflow & Permissions |
|:---:|:---:|
| ![Service Catalog](./docs/images/09-服务目录.png) | ![Workflow](./docs/images/11-工作流.png) |
| ![Knowledge Base](./docs/images/10-知识库.png) | ![Role Management](./docs/images/12-角色管理.png) |

## Tech Stack & Repository Structure

| Layer | Technology |
|:---|:---|
| Backend | Go 1.25.13, Gin, Ent, PostgreSQL, Redis |
| Frontend | Next.js 15.5, React 19, TypeScript 6, Ant Design 6, Tailwind CSS |
| Workflow | BPMN 2.0, process definitions/instances/tasks/variables/history |
| AI/RAG | LLM Gateway, pgvector, OpenAI/compatible APIs, Ollama optional |
| Delivery | Docker Compose, GHCR, GitHub Actions, Prometheus/Grafana optional |

```text
itsm/
├── itsm-backend/     # Go API, domain services, Ent schemas, Worker
├── itsm-frontend/    # Next.js admin, service desk, and user portal
├── itsm-ai-service/  # AI/RAG assistant service
├── itsm-agent/       # Agent extensions
├── itsm-skill/       # Skill extensions
├── itsm-cli/         # CLI + agent protocol + MCP server
├── docs/             # Product, architecture, development, deployment, testing docs
├── scripts/          # Development, production, release, and diagnostic scripts
└── monitoring/       # Prometheus/Grafana configuration
```

The backend is the source of truth for business rules, permissions, tenant isolation, workflow execution, and audit. The frontend does not replicate lifecycle rules.

## Development & Testing

```bash
# Backend
cd itsm-backend
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./...

# Frontend
cd ../itsm-frontend
npm install
npm run type-check
npm test

# Root-level engineering contracts
cd ..
make check-contracts

# E2E that writes business data (creating users/tickets/permission fixtures) always
# runs against a one-off isolated stack, never the resident stack: "up" proves via a
# canary write that the frontend proxy really lands in this stack's database, and a
# Playwright-side guard refuses to run when that proof does not hold.
eval "$(scripts/e2e-isolated-stack.sh up --print-env)"   # start the stack and inject test env vars (incl. E2E_RUN_ID) into the current shell
cd itsm-frontend && npm run test:e2e:roles               # run fixtures with those env vars
cd .. && scripts/e2e-isolated-stack.sh down              # tear down (incl. this stack's volumes) and delete the proof file; in a new shell pass E2E_RUN_ID=<run id>

# Business flow regression (requires dev environment; covers incident/problem
# lifecycle, change rejection path, state machine negative cases, forged token
# negative cases, and notification/dashboard integration — 26 assertions)
python3 output/dev_business_flow_test.py
```

See [Local Development Commands](./docs/dev-commands-reference.md) (isolated stack: §2.5) and [Testing Guide](./docs/testing/README.md) for detailed layering, test strategy, and E2E usage.

## Production Deployment

Three deployment modes are supported:

- **private**: single-enterprise private deployment
- **saas**: multi-tenant SaaS, platform-operated
- **saas_msp**: SaaS + MSP, platform and service provider collaboration

```bash
make prod-init        # Generate production config
# Edit .env.prod: change passwords, JWT secret, domain, etc.
make fresh-install-gate  # Empty-volume first-install self-check before delivery: migrate+seed must pass; ledger / process templates / binding invariants must hold
make prod-deploy      # Deploy
make prod-health      # Check status
```

Pre-launch checklist:

- [ ] Change all default passwords and secrets (`ADMIN_PASSWORD`, `JWT_SECRET`, database, Redis)
- [ ] Configure TLS certificates (terminated at gateway/load balancer)
- [ ] Run database migration and backup recovery drill
- [ ] Verify tenant isolation and RBAC permissions
- [ ] Test capacity, failure recovery, and dead-letter replay
- [ ] Validate enabled modules (CMDB, AI, connectors, etc.)

> Never use development defaults in production. See [Production Readiness Program](./docs/delivery/production-readiness-program.md) and [Operations Runbook](./docs/delivery/production-initialization.md) for detailed steps.

## Documentation

| Document | Purpose |
|:---|:---|
| [Documentation Center](./docs/README.md) | Find resources by role and topic |
| [Open-Source Capability Statement](./docs/product/open-source-release-capability.md) | Roles, end-to-end process closure, maturity, limitations, and acceptance entry points |
| [Commercial Capability Contract](./docs/product/itsm-commercial-capability-contract.md) | Capability maturity, commercial MVP, and non-goals |
| [Commercial-Ready Architecture](./docs/architecture/commercial-ready-architecture.md) | Production-grade overall architecture |
| [CMDB Commercial MVP](./docs/product/cmdb-commercial-mvp.md) | CMDB Available/Preview boundaries and acceptance criteria |
| [Outbox Architecture](./docs/architecture/operational-command-outbox.md) | Reliable async execution specification |
| [API Reference](./docs/api-reference.md) | HTTP API documentation |
| [Local Development Commands](./docs/dev-commands-reference.md) | Development commands, debugging, and troubleshooting |
| [Testing Guide](./docs/testing/README.md) | Unit, integration, contract, and E2E testing |
| [Roadmap](./ROADMAP.md) | Single source of truth for current iteration direction |
| [Upgrade Guide](./UPGRADE.md) | v1.6.x → latest: breaking changes, environment variables, migration, and rollback |

## Contributing

Issues, documentation, and code contributions are welcome. Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before getting started.

### Quick Contribution Flow

```bash
# 1. Fork and clone the repository
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. Create a feature branch
git checkout -b feature/your-feature

# 3. Set up development environment
make dev-start-docker

# 4. Develop and test
# Backend: cd itsm-backend && go test ./...
# Frontend: cd itsm-frontend && npm test

# 5. Commit (use Conventional Commits)
# Stage files explicitly — do not use git add . / git add -A
git add itsm-backend/handlers/incident/service.go
git commit -m "feat: describe your change"

# 6. Push and create PR
git push origin feature/your-feature
```

### Ways to Contribute

| Method | Description |
|:---|:---|
| Report bugs | Use [GitHub Issues](https://github.com/heidsoft/itsm/issues/new) |
| Propose features | Discuss in [Discussions](https://github.com/heidsoft/itsm/discussions) |
| Improve docs | Submit documentation improvement PRs |
| Submit code | Contribute via Pull Request |
| Code review | Participate in PR reviews |

### Contribution Requirements

Code style (ESLint + gofmt), commit message conventions, test requirements, and PR process are detailed in [CONTRIBUTING.md](./CONTRIBUTING.md). PRs must pass all CI checks before merging.

Before submitting, stage files explicitly and confirm the staging area contains no passwords, keys, or tokens (`git add .` / `git add -A` may accidentally include local credential files).

- [View contributors](https://github.com/heidsoft/itsm/graphs/contributors)

## Star History

[![GitHub Star History](./docs/images/star-history.svg)](https://github.com/heidsoft/itsm/stargazers)

## License

Licensed under the [Apache License 2.0](./LICENSE). Free for commercial use, modification, and distribution; subject to the license and [NOTICE](./NOTICE) requirements.

<div align="center">

[GitHub](https://github.com/heidsoft/itsm) · [Issues](https://github.com/heidsoft/itsm/issues) · [Discussions](https://github.com/heidsoft/itsm/discussions)

If this project helps you, feel free to star it, try it out, and share your real-world feedback.

</div>
