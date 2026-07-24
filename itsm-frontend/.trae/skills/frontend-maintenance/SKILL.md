---
name: frontend-maintenance
description: Maintain and harden the ITSM Next.js frontend by resolving Ant Design deprecations, console errors, async races, stale state, API failures, route issues, performance regressions, and dependency drift. Use for routine maintenance, upgrade work, warnings, crashes, or resilience defects.
---

# Frontend Maintenance

## Triage

Reproduce the warning or failure and capture its route, role, console stack, network request,
and visible state. Trace the owning component and API client before changing shared code.

## Maintenance rules

- Follow installed Ant Design types and deprecation messages; do not assume a future API.
- Use `App.useApp()` for contextual message/notification/modal calls.
- Guard async state with cancellation or request sequencing when rerenders can race.
- Show actionable loading, empty, retry, permission, and error states.
- Never substitute mock data for a failed production API.
- Do not convert errors into silent empty results.
- Preserve authentication redirect behavior and prevent refresh/login loops.
- Keep route and dynamic menu changes aligned with backend permissions.
- Prefer small dependency upgrades with lockfile review and focused regression tests.

Useful checks:

```bash
rg -n 'valueStyle|bodyStyle|headStyle|Tabs\\.TabPane|Space direction' src
rg -n \"import \\{[^}]*message[^}]*\\} from 'antd'\" src
npm outdated
```

Do not run broad automated replacements without reviewing generated JSX and types.

## API resilience

When a refresh fails:

- retain already loaded data when safe;
- mark it as stale or show a warning;
- offer retry;
- keep permission and authentication errors distinct;
- log through the project's observability path without secrets.

## Verification

```bash
npm run type-check
npm run lint:check
npm run test:ci
npm run build
```

Use a focused Playwright test to verify visible maintenance fixes and console/network health.
