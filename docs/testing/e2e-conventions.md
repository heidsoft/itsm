# E2E Conventions (golden / multi-tenant / business-flows)

> Companion to **PR-0.4** of the Test Improvement Plan. See plan section
> "阶段 0 / PR-0.4" for originating rationale.
>
> Reader: anyone adding or editing a Playwright spec under `tests/e2e/`.

## 1. Why three projects?

| Project | Directory | Default local | CI | Purpose |
|---|---|---|---|---|
| (default) `chromium` / `firefox` / `webkit` / `chrome` / `edge` | `tests/e2e/**` (any) | ✅ runs | ✅ | General regression |
| `business-flows` | `tests/e2e/business-flows/` | ✅ runs | ✅ | Heavier 404-line flows |
| `golden` | `tests/e2e/golden/` | ❌ hidden | ✅ (`PLAYWRIGHT_ENABLE_GOLDEN=1`) | The 25 critical "must not break" flows wired into GA gate (PR-3.4) |
| `multi-tenant` | `tests/e2e/multi-tenant/` | ❌ hidden | ✅ (`PLAYWRIGHT_ENABLE_MULTI_TENANT=1`) | Cross-tenant isolation regression (PR-3.5) |

By default the slowest projects are opt-in so day-to-day `npm run e2e` stays
fast; CI flips the env vars to enable them on the relevant jobs.

## 2. Authoring rules for a golden spec

```ts
// tests/e2e/golden/<priority>-<feature>.spec.ts
import { test, expect } from '@playwright/test';

test.describe.configure({ mode: 'serial' }); // ordered user state

test('@golden ticket-create-assign-resolve-close', async ({ page }) => {
  // MUST end with @golden tag for grep filter to pick it up
  // MUST run ≤ 60s (CI timeout); break up longer journeys
  // MUST NOT use raw fetch — go through the BaseApi helper
});
```

Hard rules:

1. Tag every test with `@golden` (within the description string), not in
   the `tags` array — `playwright.config.ts` matches by `grep`.
2. Keep each test ≤ 60 seconds wall-clock.
3. Tests must be **idempotent** — re-running the same spec must not leave
   the seed in a corrupt state. Use `tests/e2e/_lib/seedReset.ts`.
4. Never call `page.evaluate` to mutate `localStorage` directly — use
   the documented `loginAs(page, role)` helper.
5. Assertions must use `expect(...).toBeVisible()` over arbitrary text
   searches — frontend i18n changes should never break golden tests.

## 3. Multi-tenant spec rules

```ts
// tests/e2e/multi-tenant/<scenario>.spec.ts
test('@multi-tenant tenant-a cannot read tenant-b tickets', async ({ page, request }) => {
  await loginAs(request, { tenant: 'a', role: 'agent' });
  const res = await request.get('/api/v1/tickets', {
    headers: { 'X-Tenant-Code': 'tenant-b' },
  });
  expect(res.status()).toBe(200); // MUST NOT 200 with body
  const body = await res.json();
  expect(body.data).toEqual([]);   // MUST be empty
});
```

Hard rules:

1. Tag `@multi-tenant` for `grep` matching.
2. Always pair backend HTTP assertions with at least one frontend assertion
   to catch "API is restricted but UI shows the wrong value" regressions.
3. Spin up the three-tenant seed (`scripts/seed-multi-tenant.sh`) in
   `globalSetup` (PR-3.5 will wire this).

## 4. Recording / replay

```bash
# Record a new golden flow
PLAYWRIGHT_BASE_URL=https://staging.example.com \
  npx playwright codegen --project=golden

# Replay a saved trace on failure
npx playwright show-trace test-results/<...>/trace.zip
```

Traces and videos for the `golden` project are retained even on success,
because the GA gate (PR-0.5) needs them as evidence.

## 5. Filename convention

`<priority>-<feature>.spec.ts` where `<priority>` ∈ `{p0, p1, p2}`,
mirroring the backend priority scheme in `docs/testing/controller-failing-list.md`.

Examples committed in PR-0.4:
- `golden/dashboard.spec.ts` (placeholder)
- `multi-tenant/tenant-isolation.spec.ts` (placeholder)

PR-3.3 / PR-3.5 will replace the placeholders with the actual flows.

## 6. CI wiring (PR-3.4 deliverable)

```yaml
# .github/workflows/ga-gate.yml (new job added by PR-3.4)
playwright-golden:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v7
    - uses: actions/setup-node@v6
      with: { node-version: '22' }
    - run: npm ci --prefix itsm-frontend
    - run: npx playwright install --with-deps chromium
    - name: backend up
      run: ./scripts/deploy-dev.sh up --local &
    - name: golden e2e
      run: |
        cd itsm-frontend
        PLAYWRIGHT_ENABLE_GOLDEN=1 \
        PLAYWRIGHT_ENABLE_MULTI_TENANT=1 \
          npx playwright test --project=golden --project=multi-tenant
    - name: publish report
      if: always()
      uses: actions/upload-artifact@v7
      with:
        name: playwright-golden
        path: |
          itsm-frontend/playwright-report
          itsm-frontend/test-results
        retention-days: 14
```

PR-3.4 will edit `ga-gate.yml` to add this job.

## 7. Trend pipeline (PR-0.5 deliverable)

The GA gate's status comment reads the `playwright-report/results.json`
artifact uploaded above and posts:

```
Playwright golden (n=25)
  pass: 23
  fail: 1
  flake: 1
  runtime: 4m 12s
```

PR-0.5 ships the script that turns the JSON into Markdown.

## 8. k6 baseline (cross-reference)

The backend load-test counterpart lives at `itsm-backend/tests/load/`.
See `itsm-backend/tests/load/README.md` for how to add a new endpoint
script and how PR-4.1 / PR-4.2 will plug it into `perf-budget.yml`.
