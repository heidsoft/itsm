---
name: ticket-lifecycle-check
description: Verify the real ITSM ticket lifecycle across creation, classification, assignment, comments, SLA, status transitions, resolution, closure, permissions, tenant isolation, audit, and persistence. Use for ticket workflow testing, CRUD/lifecycle regressions, or release acceptance.
---

# Ticket Lifecycle Check

## Use the repository Playwright flow

The canonical browser flow is:

```bash
cd itsm-frontend
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/business-flows/ticket-lifecycle.spec.ts \
  --project=chromium --workers=1
```

Backend must be available on `8090`; frontend uses `PLAYWRIGHT_BASE_URL` or port `3000`.
Use existing auth fixtures and unique test data. Do not use mock backend data.

## Validate the lifecycle

1. End user creates a ticket with required classification.
2. Ticket appears in requester and service-desk views as permitted.
3. Authorized agent assigns/updates it; unauthorized roles cannot.
4. Comments/attachments and workflow/SLA side effects are visible.
5. Status follows valid service-layer transitions.
6. Resolution and closure retain authoritative timestamps and audit history.
7. Refresh/revisit preserves the result.
8. A different tenant cannot access the record.

Do not assume delete is part of the business lifecycle; many enterprise records should be
retained and audited.

## Failure triage

Observe the expected API response for each mutation. If behavior fails, trace:

`ticket page → ticket API client → router → controller → service → DTO/mapper → Ent`.

Add or update a focused test for the broken visible outcome, then run:

```bash
cd itsm-backend && go test ./service ./tests/contract ./tests/rbac
cd ../itsm-frontend && npm run type-check && npm run lint:check
```
