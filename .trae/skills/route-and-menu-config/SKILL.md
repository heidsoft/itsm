---
name: route-and-menu-config
description: Add, repair, and validate ITSM backend routes, Next.js App Router pages, dynamic menus, static menu fallback, icons, permissions, and tenant-aware menu initialization. Use for new modules, 404/405 errors, missing menu items, incorrect navigation, or route/menu permission drift.
---

# Route and Menu Configuration

## Trace all surfaces

1. Backend routes: `itsm-backend/router/` and `router/router.go`.
2. Dependency wiring: `itsm-backend/internal/bootstrap/app.go`.
3. Frontend route: `itsm-frontend/src/app/(main)/.../page.tsx`.
4. Dynamic menu API/service: `/api/v1/auth/menus`, menu controller/service.
5. Static fallback/icons: `itsm-frontend/src/components/layout/sidebar/`.
6. Default initialization: `itsm-backend/pkg/seeder/` or the menu initialization endpoint.

Do not patch only the sidebar. Menu visibility is not authorization.

## Add or repair a module

- register the controller/service in bootstrap;
- register authenticated and tenant-scoped routes in the correct group;
- use stable REST paths and the standard response envelope;
- create the exact matching App Router page;
- assign a real permission code and enforce it at the endpoint;
- add dynamic default menu data idempotently for applicable deployment modes;
- add the icon to the existing icon resolver;
- keep static fallback and dynamic menu paths consistent;
- verify admin and restricted roles.

Avoid direct customer/business sample rows in default seeding.

## Diagnose

```bash
rg -n 'resource-name|/resource-path' itsm-backend/router itsm-backend/internal/bootstrap
rg --files 'itsm-frontend/src/app/(main)' | rg 'page\\.tsx$'
rg -n 'resource-name|/resource-path' \
  itsm-frontend/src/components/layout/sidebar itsm-backend/pkg/seeder
```

For a 404/405, compare the frontend client's method/path with Gin registration. For a missing
menu, inspect `/api/v1/auth/menus` before changing presentation code.

For an App Router page that exists in source but fails at runtime, distinguish an old running
image from a source/build defect. Run a production build and confirm the page appears in:

- `.next/server/app-paths-manifest.json`;
- `.next/routes-manifest.json`.

For dynamic routes, verify both the parameter route and any more-specific static sibling such
as `/changes/new`; do not infer route health from type-check alone.

## Verification

```bash
cd itsm-backend
go test ./router ./service ./pkg/seeder

cd ../itsm-frontend
npm run type-check
npm run build
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/module-access.spec.ts --project=chromium --workers=1
```

Include a role-denied or hidden-menu regression when permissions change.
