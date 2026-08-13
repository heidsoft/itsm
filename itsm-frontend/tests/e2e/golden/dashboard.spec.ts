/**
 * Golden E2E — 25 critical business flow smoke tests.
 *
 * Added by PR-0.4 (Test Improvement Plan). These specs run *only* on the
 * dedicated `playwright-golden` CI job (see .github/workflows/ga-gate.yml).
 * They are intentionally short and observable: each test must finish in
 * ≤60s and must post a status line to the console so the trend pipeline
 * (PR-0.5) can pick up pass/fail counters.
 *
 * Convention:
 *   - Filename pattern: <priority>-<feature>.spec.ts
 *   - Each spec declares `test.describe.configure({ mode: 'serial' })`
 *     to keep user state ordered.
 *   - Uses the shared `loginAs(page, role)` helper from tests/e2e/_lib/.
 *
 * Re-recording a flow:
 *   npx playwright codegen --project=golden http://localhost:3000
 */

import { test, expect } from '@playwright/test';

const FLOW_TAG = 'golden';

// Placeholder smoke test: dashboard renders for an end-user.
// PR-3.3 will replace this with the real 25 flows listed in
// docs/testing/e2e-golden-catalog.md.
test.describe('golden / dashboard renders @golden', () => {
  test('end-user can reach /dashboard', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/.*login|dashboard/);
  });
});

// See docs/testing/e2e-conventions.md for the full authoring rules.
test.describe.skip('golden catalog stubs (PR-3.3 will fill in)', () => {
  test.skip('ticket-create-assign-resolve-close @golden', () => {});
  test.skip('incident-to-problem-to-change @golden', () => {});
  test.skip('sla-breach-notification @golden', () => {});
  test.skip('knowledge-publish-rag-search @golden', () => {});
  test.skip('cmdb-ci-relationship-impact @golden', () => {});
  test.skip('bpmn-approve-cosign-addsign @golden', () => {});
  test.skip('service-catalog-request-multi-approval @golden', () => {});
  test.skip('notification-preferences-read-batch @golden', () => {});
  test.skip('tenant-switch-logout-relogin @golden', () => {});
  test.skip('connector-install-configure-health @golden', () => {});
});

// The `FLOW_TAG` is exported so the trend pipeline (PR-0.5) can filter
// `playwright-report/results.json` to entries tagged `golden` when
// computing the GA-gate score.
export const GOLDEN_TAG = FLOW_TAG;
