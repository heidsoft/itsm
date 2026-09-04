# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Changed

- **BREAKING: API response fields are now camelCase across controllers and incident handlers** — Ticket, incident, SLA, and BPMN responses no longer expose snake_case fields such as `deleted_count`, `assigned_count`, `page_size`, `workflow_steps`, or `avg_resolution_time`. The new contract uses `deletedCount`, `assignedCount`, `pageSize`, `workflowSteps`, and `avgResolutionTime` to match the existing `dto.IncidentStats` / `dto.IncidentMetrics` types and the OpenAPI schema. External integrations that read snake_case fields must migrate in this release; the frontend `toCamelCase` compatibility layer in `src/lib/api/http-client.ts` is retained for one release as defense-in-depth and will be removed in the next minor.

- **BREAKING: SLA and BPMN monitoring query parameters are now camelCase** — `GET /api/v1/sla-policies/match` expects `ticketType` and `customerTier` instead of `ticket_type` and `customer_tier`; `GET /api/v1/sla-policies/compliance-rate` expects `startDate` / `endDate`; BPMN monitoring endpoints expect `timeRange` / `startTime` / `endTime`. The frontend `src/lib/api/sla-api.ts` sends camelCase directly. Existing callers using snake_case query strings must update their requests.

- **BREAKING: SLA templates and BPMN monitoring endpoints now return the standard envelope** — `controller/sla_template_controller.go` and `controller/bpmn_monitoring_controller.go` return `{code, message, data}` via `common.Success` / `common.Fail`. The previous ad-hoc `{"message": ..., "data": ...}` envelope is gone. HTTP status-code semantics are preserved (400 → `ParamErrorCode`, 404 → `NotFoundCode`, 500 → `InternalErrorCode`); only the body shape changed.

- **BREAKING: Ant Design `direction` prop removed from `Space` / `Steps` components** — Five pages (`admin/cab`, `ai/audit`, `email-intake/on-call`, `admin/config-inheritance`, `workflow/ticket-approval`) now use `orientation="vertical"` instead of `direction="vertical"`, matching the Ant Design v6 API. A new `itsm-frontend/tools/check-antd-direction.sh` CI guard fails the build if the legacy prop reappears in `src/`.

### Fixed

- **Incident priority keyword dedup** — Removed the duplicate `"critical"` entry from the `inferIncidentPriority` keyword list in `handlers/incident/handler.go` so that `autoPriorityByKeyword("critical incident")` returns `"urgent"` exactly once instead of double-matching.
- **Handler layering violation in `GET /api/v1/incidents/stats`** — The handler no longer reaches into `ent.Client` from a Gin request context. It delegates to `service.GetStats` → `repository.GetStats`, where a single `COUNT(*) FILTER` aggregation replaces the previous seven concurrent goroutine queries and reduces database roundtrips from 7 to 1 (verified via `pprof`).
- **`avgResolutionTime` is now a real aggregated value** — `GET /api/v1/incidents/stats` computes `AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60)` for incidents with a non-null `resolved_at` (per tenant), instead of returning a hard-coded `0` for every tenant.
- **super_admin navigation restored** — `hasPermission` on the frontend now honors the `*` and `resource:*` wildcards that the backend issues to super_admin, so the admin section (users, roles, permissions, menus, tenants, workflow) renders again. `/auth/me` now returns the `permissions` list and `AuthGuard` forwards it on session restore, fixing menus disappearing after a page refresh.
- **Change status CAS** — Change status transitions use a single conditional `UPDATE` (`UpdateStatusCAS`) instead of read-validate-write, eliminating lost-update races between concurrent approvers.
- **Problem lifecycle error semantics** — Invalid problem state transitions (e.g. closing from `investigating`) now return `409/4090` business-conflict errors instead of `500` internal errors, matching the incident module's lifecycle error pattern.
- **Change approval chain cleanup SQL** — The terminal-state cleanup statement no longer references the non-existent `updated_at` column of `change_approval_chains`, removing a per-transition error log.
- **Command bus permanent failures** — Commands whose aggregate no longer exists (e.g. workflow/notification commands left behind by a deleted change) are dead-lettered immediately instead of retrying eight times; the shared in-memory test database is now isolated per test case.
- **Frontend dev server CSS parsing** — Removed 34 invalid `:global()` usages from `globals.css` that broke compilation under Lightning CSS (Next.js Turbopack); dev containers now persist `node_modules` and the `.next` cache in named volumes, cutting container restart recovery from ~170s to ~38s.
- **Production volume naming** — `docker-compose.prod.yml` pins the existing `postgres`/`redis` data volumes as `external` so switching the compose project name no longer mounts fresh empty volumes ("disappearing data"); MinIO stays auto-created since it is an optional `--profile storage` service with no historical volume.
- **Registration API contract** — Fixed the `fullName`/`displayName` mismatch between the register endpoint contract and consumers.

### Tests

- **Business flow regression suite** — Added `output/dev_business_flow_test.py` (27 assertions: full incident lifecycle, problem lifecycle, change reject path, state-machine negative cases returning 409 not 500, forged-token negative cases, notification/dashboard linkage) alongside the 63-item multi-domain integration suite; both run green against the dev stack.

- **Bug-fix regression coverage for the seven v1.6.x hardening fixes** — Added unit tests: `handlers/incident/repository_impl_test.go` covers Bug 1 (`autoPriorityByKeyword` single-match) and Bugs 3+4 (`service.GetStats` table-driven, no direct `ent.Client` access, `avgResolutionTime > 0` after resolved events, cross-tenant isolation); `controller/sla_template_controller_test.go` covers Bug 2 (standard `{code, message, data}` envelope, status-code mapping, no raw `ctx.JSON`); `controller/sla_policy_controller_test.go` covers Bug 6 (static source check that no snake_case `ctx.Query("...")` remains and the camelCase counterparts are present); `controller/ticket_controller_test.go` covers Bug 5 (`BatchDeleteTickets` returns `deletedCount` and never `deleted_count`); `itsm-frontend/tools/check-antd-direction.sh` is the Bug 7 CI guard that fails if `direction="vertical"` returns to `src/`. All new tests pass under `go test ./...`.

### Documentation

- Reworked the open-source capability guide around user roles, executable business journeys, TicketType-driven dynamic forms, maturity boundaries, deployment acceptance, and explicit non-goals.
- Updated README authentication and ticket examples for HttpOnly cookie sessions, CSRF protection, camelCase DTOs, string priorities, and runtime TicketType selection.
- Resynchronized the roadmap with the v1.6.x hardening line and corrected obsolete BPMN goroutine claims now that ticket and incident production wiring uses persistent command/outbox execution.

---

## [1.6.9] - 2026-08-20

### Added

- **Problem Management expansion** — Added Problem trends, hotspots, SLA lookup, and per-problem comment read/write endpoints so operators can analyze recurring issues and discuss root cause without leaving the Problem record.
- **Incident comment deletion** — Added `DELETE /api/v1/incidents/:id/comments/:commentId` for tenant-scoped comment cleanup, mirroring the existing create/list semantics.
- **Knowledge recommendations** — Replaced the previous stub implementations of `/api/v1/knowledge/recommendations` and `/api/v1/knowledge/recent` with tenant-scoped queries that return real published articles with proper permission filtering.

### Fixed

- **Frontend Edge Runtime compatibility** — Replaced `Buffer.from(...)` calls in `src/middleware.ts` and `src/app/api/[...path]/route.ts` with an `atob()`-based JWT decoder so the Next.js middleware no longer raises `Code generation from strings disallowed for this context` in Edge Runtime (was breaking `itsm-frontend-prod` with HTTP 500).

### CI

- **Backend lint toolchain** — Pinned `gofumpt` to `v0.7.0` and cached `~/go/bin` between CI runs to make formatting results reproducible and shave install time.

### Tests

- **CMDB Service layer coverage** — Added 9 table-driven test functions covering CloudService / CloudAccount / CloudResource CRUD, Reconciliation (bound / unbound / orphan / unlinked / mixed), and Discovery operations via a `mockRepository` that isolates the service from the database.

---

## [1.6.8] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.12 to include the latest TLS, X.509, and MIME-header security fixes required by the production vulnerability gate.

---

## [1.6.7] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.10 as an intermediate production security baseline.

### Fixed

- **SLA calendar enforcement** — SLA deadlines now apply each definition's configured business calendar consistently across creation, lookup, and overdue detection; the prior unused implementation was removed.
- **Backend quality gate** — Removed obsolete change-status validation code so the release static-analysis gate has no dead-code findings.

---

## [1.6.6] - 2026-08-03

### Fixed

- **Cookie-only authentication test contract** — Updated API integration and HTTP client tests to assert credentialed browser requests without exposing HttpOnly session cookies through JavaScript `Authorization` headers.

---

## [1.6.5] - 2026-08-03

### Security

- **Production dependency remediation** — Updated frontend production dependencies and lockfile overrides for known Axios, DOMPurify, lodash, PostCSS, Sharp, and UUID advisories; `npm audit --omit=dev --audit-level=high` now reports zero vulnerabilities.
- **Package-manager integrity** — Removed the obsolete `pnpm-lock.yaml`; the frontend is governed solely by the committed npm lockfile, matching the release CI configuration.

---

## [1.6.4] - 2026-08-03

### Fixed

- **CI formatting gate** — Applied the repository's `gofumpt` formatting standard to backend source and tests, restoring the required backend release pipeline gate.

---

## [1.6.3] - 2026-08-03

### Fixed

- **Production authentication hardening** — Browser sessions now use HttpOnly, SameSite=Lax cookies exclusively. Access and refresh tokens are no longer exposed in JSON responses or read by frontend JavaScript; refresh token rotation is performed through the cookie transport.
- **Secure proxy cookie handling** — Authentication cookies now receive the `Secure` attribute when the request is HTTPS or terminated by a trusted HTTPS reverse proxy.
- **CSRF refresh coverage** — Both supported refresh routes are explicitly covered by the CSRF session-refresh exception.
- **Initialization readiness** — `/api/v1/readyz` now requires the latest registered post-schema migration rather than a stale, hard-coded migration version.
- **Workflow validation contract** — Removed the frontend call to an unimplemented validation endpoint; workflow designer preflight validation is deterministic and local, while create/update remain server-validated.

### Changed

- **Release metadata** — Frontend package, backend version endpoint, system configuration endpoint, and GA-readiness report now identify the release as `1.6.3`.

---

## [1.6.0] - 2026-08-01

### Fixed

- **Critical: Form Boundary Issue** - Fixed TicketTypeFormModal where approval/SLA tab fields were outside `<Form>` component, causing form validation failures.
- **Critical: Auth State Persistence** - Removed `isAuthenticated` from Zustand persist partialize to prevent false login state after browser refresh.
- **High: Timer Leak in Export/Import** - Fixed memory leaks in TicketCategoryExport and TicketCategoryImport with proper useRef cleanup on unmount and error paths.
- **High: ITIL Status Guard** - Added readonly protection on change/edit and release forms to prevent editing approved/completed records.
- **High: Suspense Boundary** - Added Suspense wrapper for useSearchParams in tickets page to fix CSR bailout.
- **High: Query Key Normalization** - Fixed useTicketsQuery to only include request params in queryKey, not response data.
- **High: NaN Route Guard** - Added Number.isFinite check for marketplace/[id] route parameter.
- **High: Error Handling** - Fixed workflow-api to throw errors instead of silently returning empty arrays.
- **Medium: useFormMemory** - Added debounce (500ms), enabled flag for edit mode, and dayjs serialization.
- **Medium: CIEditorForm** - Added min/max/precision props to InputNumber for numeric fields.
- **Medium: ChangeDetail Null Guard** - Added null check for createdAt before dayjs conversion.

### Changed

- **Ant Design destroyOnHidden** - Added to ProblemInvestigationTab modals.
- **SchemaField Type** - Added validation property with minValue/maxValue/precision/pattern.

---

## [1.5.2] - 2026-07-31

### Fixed

- **SLA Compliance Statistics** - Fixed negative compliance numbers caused by mismatched time scopes between total ticket count (30 days) and violated ticket count (all time). Backend now uses `HasTicketWith` edge predicate for consistent scope.
- **Timezone Inconsistency** - Fixed dashboard activity timestamps using `time.RFC3339` format instead of `Format("2006-01-02 15:04:05")` which browsers parse as UTC.
- **TicketDetail Duplicate Request** - Fixed double API call on ticket detail page by removing `fetchTicket`/`fetchSLAInfo` from useEffect dependency arrays.
- **Login Error Message** - Login page now shows actual backend error messages (e.g., "invalid credentials") instead of generic "登录失败".

### Changed

- **Ant Design TabPane Deprecation Migrated** - Converted all `Tabs.TabPane` / `<TabPane>` usage to `items` prop pattern across 8 files: analytics, applications, NotificationCenter, TicketTypeFormModal, IncidentManagement, FieldDesigner, profile, dashboard.
- **Space direction → orientation** - Migrated 6 instances of deprecated `Space direction="vertical"` to `orientation="vertical"`.
- **destroyOnClose → destroyOnHidden** - Migrated 1 instance in ApprovalTimeline component.
- **alert()/confirm() → antd message/modal** - Replaced native browser dialogs in marketplace and installations pages with antd `App.useApp()` message/modal.
- **console.log Cleanup** - Removed debug console.log from TicketDetail and BPMNDesigner.

---

## [1.5.0] - 2026-07-30

### Added

- **Connector/Skill Manifest Hardening** - All official connector manifests now declare `version`, `requiredPermissions`, and a deterministic SHA-256 `checksum`; registration is fail-closed (incomplete manifests are rejected at startup). Skill manifests share the same validation and checksum convention. Market API exposes `isOfficial` / `requiredPermissions` / `checksum`.
- **Post-Schema Migrations 008-010** - Initialization ledger (008), PostgreSQL RLS tenant isolation (009), ticket types in `itil-core` transaction (010).
- **Bootstrap Token** - One-time hashed bootstrap token with TTL, concurrent-consumption protection, replay defense, and break-glass flow for first-admin creation.
- **Endpoint ACL Manifest** - Versioned ACL manifest with 100% static coverage gate over protected routes (route-ACL-permission-menu).
- **Fencing Token Hardening** - Owner/token/lease re-verified inside the committing transaction; stale-writer prevention proven via PostgreSQL fault-injection tests.
- **Audit Routes** - New audit trail API endpoints with tenant isolation support
- **CI Attribute Validator** - Moved from handlers to service layer for better separation of concerns
- **Operator Context** - Enhanced audit trail with operator context tracking

### Fixed

- **Ant Design v6 Select Compatibility** - Replaced deprecated `<Select><Option>` child pattern with `options` prop across 100+ files. Fixes issue where clicking Select dropdowns had no response in antd v6. Affected modules: Ticket, Incident, Problem, Change, CMDB, Workflow, SLA, Service Catalog, Admin, and Reports pages.

### Changed

- **CMDB API Route Convergence** - `/api/v1/cmdb/*` is now the canonical prefix for all CMDB endpoints (CIs, CI types, relationships, relationship types, topology, impact analysis, change history, stats). Frontend API clients (`cmdb-api.ts`, `cmdb-relationship.ts`) have been switched to the canonical prefix. Added `GET /api/v1/cmdb/relationship-types` to the canonical tree. Note: change history is `GET /api/v1/cmdb/cis/:id/history` (the old `change-history` suffix only exists on the deprecated alias).
- **CMDB Multi-Tenant Isolation** - Added `tenant_id` field to CI relationships, configuration item history, and discovery sources. Added tenant-aware backfill migration.
- **CI Attribute Validation** - Migrated from `handlers/cmdb/attribute_validation.go` to `service/ci_attribute_validator.go`.

### Deprecated

- **`/api/v1/configuration-items/*` routes** - Kept as a compatibility alias for clients not yet upgraded. No new endpoints will be added under this prefix; removal will be evaluated after a regression period.

### Migration Notes

- **CMDB tenant backfill**: Run `itsm-backend/migrations/20260610_cmdb_tenant_id_backfill.sql` on existing databases to populate the new `tenant_id` fields.
- **Preset CI types**: Enable with `itsm-backend/migrations/20260611_enable_preset_ci_types.sql`.
- **Audit routes**: New endpoints are protected by existing JWT + RBAC + tenant middleware.

---

## [1.0.0] - 2026-03-07

### Added

#### Core ITIL Modules
- **Ticket Management** - Complete ticket lifecycle management with creation, assignment, tracking, and closure; built-in SLA management, priority handling, comments and attachments
- **Incident Management** - Incident discovery, logging, classification, escalation; real-time monitoring and alerts
- **Problem Management** - Root Cause Analysis (RCA), Known Error Database, problem resolution tracking
- **Change Management** - Change requests, risk assessment, multi-level approval workflows

#### Service & Knowledge Base
- **Service Catalog** - Service request templates, self-service portal, SLA management
- **Knowledge Base** - RAG intelligent search, knowledge categorization, FAQ management, vector retrieval

#### Workflow Engine
- **BPMN Workflow** - Visual process designer, approval workflow automation
- **Task Management** - Workflow task assignment and tracking

#### AI-Powered Features
- **Intelligent Classification** - Auto-identify ticket type, priority, impact scope
- **Auto-Summary** - AI-generated ticket/incident summaries
- **RAG Knowledge Base** - Vector search-based intelligent knowledge recommendation
- **Smart Suggestions** - Recommended solutions, similar tickets

#### User & Permissions
- **Multi-Tenant Architecture** - Complete tenant isolation and management
- **Role-Based Access Control** - RBAC permission system, fine-grained access control
- **User Management** - User CRUD, team and department management

#### SLA Monitoring
- **SLA Definition** - Service Level Agreement configuration
- **Real-Time Monitoring** - SLA compliance rate tracking
- **Alert Rules** - SLA violation alerts and notifications

### Technical Stack

| Category | Technology |
|----------|------------|
| Backend | Go 1.25+ / Gin / Ent ORM |
| Frontend | Next.js 15 / React 19 / TypeScript / Ant Design 6 |
| Database | PostgreSQL 17 / Redis 7 |
| Deployment | Docker / Docker Compose |

### Quick Start

```bash
# Docker Compose (Recommended)
git clone https://github.com/heidsoft/itsm.git
cd itsm
make dev-up

# Access
# Frontend: http://localhost:3000
# Backend: http://localhost:8090
# API Docs: http://localhost:8090/swagger

# Login
# Username: admin
# Password: admin123
```

### Known Limitations

- Mobile PWA features are under development
- Enterprise integration features (LDAP/SSO) planned for future releases

### Documentation

- [Development Guide](./docs/DEVELOPMENT.md)
- [Deployment Guide](./docs/DEPLOYMENT.md)
- [API Documentation](./docs/API.md)

---

*Thank you to all contributors for your support!*
