# ITSM Production Readiness Program

## Objective

Turn the current open-source ITSM system into a production-usable release train with measurable gates, parallel workstreams, and clear agent ownership.

This is not a feature wishlist. This program focuses on the minimum set of engineering controls required for safe production rollout:

1. deterministic build and release
2. failing CI that actually blocks bad changes
3. domain-level regression coverage for core ITSM flows
4. operational visibility, backup, rollback, and incident runbooks
5. security and multi-tenant correctness

## Current Baseline

The repository already has a good skeleton:

- frontend, backend, monitoring, Docker, release workflow
- production deploy script with backup and rollback phases
- health checks, Prometheus endpoint, Grafana provisioning
- test case documentation for major domains

The main gap is enforcement and execution discipline:

- CI contains `|| true` in backend lint/build/test stages, so failures can pass
- production readiness criteria are not encoded as release gates
- there is no single execution board for multi-agent parallel work
- domain completion is documented, but not tied to a release scorecard

## Release Standard

The project reaches "production-usable" only when all items below are true:

| Area | Required Gate |
| --- | --- |
| Backend build | `go build` passes without soft-fail bypass |
| Backend test | core packages pass with stable test command and coverage artifact |
| Frontend build | `npm run build` passes in CI and local release mode |
| Frontend type safety | `npm run type-check` passes with no ignored errors |
| Security | secret scan, dependency audit, container scan produce no untriaged high/critical issues |
| Core flows | smoke coverage for login, ticket, incident, problem, change, knowledge, approval |
| Deployability | `scripts/deploy-prod.sh deploy --dry-run` succeeds with documented env |
| Recovery | rollback and backup paths validated and documented |
| Observability | health, metrics, log access, alert route, and runbook verified |
| Tenancy and auth | tenant isolation and permission boundaries covered by automated tests |

## Workstreams

### W1. Release Engineering

Scope:

- remove soft-fail CI behavior
- normalize toolchain versions between repo and CI
- make release artifacts reproducible
- add pre-merge gating and release checklist

Deliverables:

- hardened GitHub workflows
- release checklist document
- stable version matrix for Go, Node, Docker images

### W2. Backend Reliability

Scope:

- identify failing or skipped backend tests
- fix flaky handlers and service tests
- validate DTO mapping and tenant-safe responses
- eliminate panic-prone runtime paths in application code

Deliverables:

- backend failure inventory
- prioritized fixes by package
- smoke and regression commands for critical APIs

### W3. Frontend Stability

Scope:

- fix type errors and route regressions
- standardize API contract handling
- restore skipped tests for critical screens
- validate standalone production build behavior

Deliverables:

- frontend regression matrix
- restored route smoke coverage
- production build verification notes

### W4. Security and Multi-Tenant Safety

Scope:

- audit auth, RBAC, tenant isolation, secret handling
- review dangerous defaults in `.env`, compose, and docs
- add explicit triage policy for scan results

Deliverables:

- security backlog
- tenant boundary test plan
- secrets and configuration hardening checklist

### W5. Operations and Supportability

Scope:

- validate monitoring stack and alert routing
- define backup, restore, rollback, and incident runbooks
- add service ownership and severity mapping

Deliverables:

- production runbooks
- alert catalog
- operational scorecard

## Execution Phases

### Phase 0. Baseline Audit

Exit criteria:

- all existing CI and local verification commands inventoried
- soft-fail paths listed
- dirty worktree conflicts identified

### Phase 1. Gate Hardening

Exit criteria:

- CI fails on real backend/frontend errors
- required branch checks documented
- release workflow validated against new gates

### Phase 2. Core Flow Stabilization

Exit criteria:

- login, ticket, incident, problem, change, knowledge, approval smoke paths pass
- top failing backend/frontend regressions fixed

### Phase 3. Production Controls

Exit criteria:

- documented backup, restore, rollback procedure
- metrics, logs, alerts, and health verification completed
- environment contract for production deploy complete

### Phase 4. Release Candidate

Exit criteria:

- scorecard reviewed
- no open P0/P1 production blockers
- tagged release candidate can be deployed from clean checkout

## Parallel Agent Model

Use one orchestrator and up to five workers. Avoid more unless file ownership is clearly isolated.

| Agent | Focus | Typical Files |
| --- | --- | --- |
| Orchestrator | sequencing, merge policy, scorecard, release decision | `docs/delivery/**`, `.orchestration/**`, `.github/workflows/**` |
| Agent A | backend reliability | `itsm-backend/service/**`, `itsm-backend/controller/**`, backend tests |
| Agent B | frontend stability | `itsm-frontend/src/**`, frontend tests |
| Agent C | release engineering | `.github/workflows/**`, `Makefile`, `scripts/**`, compose files |
| Agent D | security and tenancy | auth, middleware, RBAC, tenant tests, config |
| Agent E | operations | `monitoring/**`, deploy scripts, runbooks, env docs |

## Coordination Rules

1. Each worker owns one workstream at a time.
2. Workers write status into `.orchestration/production-readiness/status/`.
3. Workers do not modify the same file set without an orchestrator handoff.
4. High-risk changes require verification commands in the handoff note.
5. The orchestrator merges only after gate results are attached.

## First 5 Backlog Items

1. Remove backend CI `|| true` soft-fail behavior and make failures visible.
2. Run a backend failure inventory and classify by flaky, real bug, or environment issue.
3. Build a frontend route smoke matrix from existing test-case docs.
4. Create a production scorecard tied to deploy, rollback, and monitoring checks.
5. Add tenant/auth/security regression cases for the highest-risk APIs.

## Definition of Done Per Change

Every change merged under this program must include:

- scope statement
- files touched
- verification commands
- risk note
- rollback note if production-facing

## Program Cadence

- daily: worker handoff updates
- every merge: scorecard refresh if gate status changed
- every release candidate: full smoke, security, deploy dry-run, rollback check
