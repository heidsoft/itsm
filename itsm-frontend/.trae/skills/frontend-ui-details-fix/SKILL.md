---
name: frontend-ui-details-fix
description: Review and fix ITSM UI details including responsive layout, overflow, tables, forms, modals, tooltips, accessible names, keyboard behavior, loading/empty/error states, visual hierarchy, and interaction consistency. Use when core behavior works but UX polish or page-level UI quality is incomplete.
---

# Frontend UI Details Fix

## Inspect in a real browser

Check desktop and at least a 390px viewport. Start from the user's role and task, then inspect:

- page title, description, primary action, and information hierarchy;
- search/filter/action wrapping;
- table horizontal scroll and fixed action accessibility;
- long text truncation with access to the full value;
- modal/drawer height and focus behavior;
- loading, empty, validation, permission, error, retry, and success states;
- keyboard focus and accessible names for icon-only controls;
- body-level horizontal overflow and sidebar/header behavior;
- dark/theme-aware contrast and destructive-action confirmation.

Do not judge from source alone; capture screenshots or Playwright evidence.

## Fix patterns

- Reuse `BusinessPageTemplate`, `ManagementPageHeader`, and existing domain components.
- Use responsive spacing (`px-3 sm:px-6`) and stack toolbars on narrow screens.
- Use `scroll={{ x: 'max-content' }}` inside table containers; avoid hiding body overflow to
  conceal broken layout.
- Use vertical forms and full-width controls on small screens.
- Keep primary actions visible; move secondary actions into an accessible overflow menu.
- Add `aria-label` and tooltip to icon-only buttons.
- Use semantic links for navigation and buttons for actions.
- Preserve visible stale-data warnings rather than false fatal errors.

## Verification

Add a focused Playwright assertion for the defect. For responsive changes, assert:

```typescript
expect(await page.evaluate(() => document.body.scrollWidth))
  .toBeLessThanOrEqual(viewportWidth);
```

Then run:

```bash
npm run type-check
npm run lint:check
PLAYWRIGHT_SKIP_CHANNELS=1 \
npx playwright test tests/e2e/<module>.spec.ts --project=chromium --workers=1
npm run build
```

Report the viewport, before/after behavior, and evidence.
