---
name: frontend-audit-fix
description: Audit and repair the ITSM Next.js frontend for real-API usage, camelCase contracts, type safety, Ant Design correctness, user states, accessibility, tests, and production build health. Use for frontend QA, code review, mock-data removal, API/UI mismatch, or build validation.
---

# Frontend Audit and Fix

## Audit

1. Trace the page → API client → backend DTO/route.
2. Search production components for mock/sample/random fallback data.
3. Confirm HTTP fields and TypeScript types use `camelCase`.
4. Check loading, empty, error, validation, permission, success, and refresh states.
5. Check icon-button names, labels, keyboard access, responsive layout, and body overflow.
6. Inspect console/network errors in a real browser.
7. Identify whether the browser is serving the checked-out source or an older container image.

Useful searches:

```bash
rg -n 'mockData|Math\\.random\\(\\)|模拟数据' src --glob '*.tsx'
rg -n 'snake_case|_[a-z]+:' src/lib/api src/app --glob '*.{ts,tsx}'
rg -n 'valueStyle|bodyStyle|headStyle|Tabs\\.TabPane' src
```

Verify findings before mechanical replacement; legitimate internal snake_case or fixtures may
exist.

## Fix rules

- Never add mock data to production paths to hide a missing API.
- Fix backend DTO/mapper and frontend client/types together when the contract is wrong.
- Keep business rules in backend services.
- Use `App.useApp()` for contextual Ant Design message/modal APIs.
- Preserve useful prior data on refresh failure, but clearly disclose stale state.
- Avoid swallowing errors or converting permission denial into an empty list.
- Keep authentication, CSRF, RBAC, and tenant-denial errors distinct.
- Add a focused regression test for user-facing defects.

## Verification

```bash
npm run type-check
npm run lint:check
npm run test:ci
npm run build
```

For page behavior:

```bash
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/<module>.spec.ts --project=chromium --workers=1
```
