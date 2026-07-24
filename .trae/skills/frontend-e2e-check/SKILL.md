---
name: frontend-e2e-check
description: Run and diagnose ITSM browser checks with the repository's TypeScript Playwright suite. Use for UI health checks, login/navigation verification, responsive checks, module smoke tests, screenshots, or browser regressions.
---

# Frontend E2E Check

## Environment

- backend: `http://localhost:8090`;
- frontend: `PLAYWRIGHT_BASE_URL` or `http://localhost:3000`;
- tests: `itsm-frontend/tests/e2e/*.spec.ts`;
- artifacts: `/tmp/itsm-playwright-results`.

Confirm services before running tests. Reuse `tests/e2e/auth-utils.ts` or the role fixtures
instead of duplicating login logic.

## Run

```bash
cd itsm-frontend
npm run test:smoke

PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/navigation.spec.ts \
  --project=chromium --workers=1
```

For a module, select its focused spec such as `tickets.spec.ts`, `incidents.spec.ts`,
`changes.spec.ts`, or a file under `business-flows/`.

## Assertions

Verify more than rendering:

1. intended role can enter;
2. heading, loading, empty, error, and permission states are understandable;
3. controls work using role/test-id selectors;
4. API-backed actions observe the expected response;
5. successful data survives refresh or revisit;
6. narrow viewport has no body-level horizontal overflow.

Use `page.waitForResponse` for mutations. Do not use arbitrary sleeps as the primary
synchronization mechanism.

## Diagnose failures

Inspect the Playwright error context, screenshot, video, and trace. Separate:

- first-time Next.js compilation latency;
- stale/brittle selector;
- missing backend or authentication;
- actual product or API regression.

Fix the product and keep a focused regression test when behavior is user-facing.
