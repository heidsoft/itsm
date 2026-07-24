---
name: api-integration-testing
description: Validate and repair ITSM frontend/backend API contracts, authentication, tenant isolation, DTO mapping, routes, and persistence. Use for endpoint testing, 4xx/5xx diagnosis, request or response mismatches, camelCase issues, contract regressions, and cross-module integration failures.
---

# API Integration Testing

## Establish the contract

1. Locate the backend route in `itsm-backend/router/`.
2. Trace controller → service → DTO/mapper; controllers must not expose Ent models.
3. Locate the frontend client in `itsm-frontend/src/lib/api/`.
4. Compare method, path, query parameters, request body, response envelope, and error behavior.
5. Treat HTTP JSON fields as `camelCase`; allow `snake_case` only in Ent/database internals.

The API envelope is:

```json
{"code": 0, "message": "success", "data": {}}
```

Do not infer success from HTTP 200 alone. Assert `code`, the typed `data`, and visible persistence.

## Test in layers

Run the narrowest applicable checks first:

```bash
cd itsm-backend
go test ./controller ./service ./tests/contract

cd ../itsm-frontend
npm run type-check
npm run test:integration
```

For a live contract, use backend port `8090` and authenticate through
`POST /api/v1/auth/login`. Never print tokens. Prefer a focused Playwright test when the
contract drives a user-facing action:

```bash
PLAYWRIGHT_BASE_URL=http://localhost:3000 \
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/<module>.spec.ts --project=chromium --workers=1
```

Use `page.waitForResponse` to verify the expected method and endpoint, then assert the
resulting UI state and refresh/revisit persistence.

## Required negative coverage

- missing or invalid input;
- unauthenticated and unauthorized access;
- cross-tenant resource IDs;
- not-found records;
- duplicate or idempotent operations;
- backend failure displayed as an actionable UI state.

Fail closed on tenant or permission ambiguity. Do not add frontend mock fallbacks for missing
backend routes.

## Completion

Update backend DTO/mapper, frontend types/client, and regression tests together. Finish with:

```bash
cd itsm-backend && go test ./...
cd ../itsm-frontend && npm run type-check && npm run lint:check
```
