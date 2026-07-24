---
name: frontend-testing-guide
description: Design, write, and repair ITSM frontend tests using Jest, React Testing Library, and TypeScript Playwright. Use for component/page tests, integration tests, E2E coverage, flaky tests, accessibility assertions, or frontend coverage improvements.
---

# Frontend Testing Guide

## Select the level

- Unit: formatting, validation, reducers/stores, pure hooks.
- Component: isolated UI behavior with React Testing Library.
- Integration: API client/store/component interaction with controlled dependencies.
- Playwright: routing, real browser behavior, API-backed workflows, roles, tenants, and
  persistence.

Do not test implementation details when a user-visible assertion is available.

## Repository commands

```bash
cd itsm-frontend
npm run test:unit
npm run test:integration
npm run test:ci
npm run test:e2e
```

Run focused tests while iterating:

```bash
npm test -- TicketList.test.tsx
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/tickets.spec.ts --project=chromium --workers=1
```

## Component rules

- Prefer `userEvent` over low-level events.
- Query by role/label; use test IDs for business-critical controls.
- Await async UI with `findBy*` or `waitFor`.
- Mock at an external boundary; never introduce mock data into production components.
- Cover loading, empty, success, validation, failure, and permission states as applicable.

## Playwright rules

Start from a role and business outcome. Assert the visible state, expected API method/path,
and persistence. Prefer existing auth helpers and page objects. Avoid CSS structure selectors
and arbitrary timeouts. Keep one visible outcome per test.

For shared state or a single development database, use `--workers=1`. Create unique records and
avoid destructive cleanup unless the test owns them.

## Completion

Run the focused test, then:

```bash
npm run type-check
npm run lint:check
npm run test:ci
```

Run `npm run build` for route, config, dependency, or server/client boundary changes.
