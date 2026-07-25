---
name: ticket-module-dev
description: Implement, fix, and refactor the ITSM ticket module across Go services/controllers/DTOs, Next.js pages/API clients, workflow, SLA, assignment, comments, attachments, templates, permissions, tenant isolation, and tests. Use for any ticket-management code change.
---

# Ticket Module Development

## Trace before editing

Locate the active route and implementation with `rg`; the repository contains legacy and
specialized ticket surfaces. Trace backend route/controller/service/DTO/schema together with
the frontend page/client/types before deciding where to change behavior.

## Contract rules

- HTTP request, response, and query fields use `camelCase`.
- Ent/database fields may use `snake_case`.
- Controllers return ticket DTOs through existing mappers, never Ent models.
- The frontend calls through `itsm-frontend/src/lib/api/`; do not add direct `fetch` calls
  inside components.
- Keep ID types aligned with the backend contract.

## Business rules

- Validate lifecycle transitions in the service, not only in the UI.
- Derive requester/actor/tenant from authenticated context.
- Scope ticket and related user/category/team queries to the tenant.
- Preserve SLA timestamps, workflow execution, comments, attachments, CCs, and audit history.
- Route assignment automation through existing assignment rules and engineer skill matching.
- Keep AI triage/summarization auditable and provide deterministic fallback.
- Do not silently delete enterprise records when closure/archive is the intended lifecycle.

## Implementation loop

1. Add or update regression tests.
2. Change DTO/mapper and frontend types together for contract changes.
3. Put transactions and side effects in the service.
4. Expose loading, empty, validation, permission, success, and failure states in the UI.
5. Verify refresh/revisit persistence.

## Verification

```bash
cd itsm-backend
go test ./service ./controller ./tests/contract ./tests/rbac

cd ../itsm-frontend
npm run type-check
npm run lint:check
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/tickets.spec.ts \
  tests/e2e/business-flows/ticket-lifecycle.spec.ts \
  --project=chromium --workers=1
```
