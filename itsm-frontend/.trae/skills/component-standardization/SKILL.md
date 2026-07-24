---
name: component-standardization
description: Standardize ITSM React components on existing Ant Design, Tailwind, design tokens, shared page templates, accessibility, and responsive patterns. Use when consolidating duplicate UI, refactoring legacy components, or aligning pages with the product design system.
---

# Component Standardization

## Reuse before creating

Search for an existing component or page pattern before adding another abstraction:

```bash
rg -n 'ComponentName|visible label' src/components src/app
rg --files src/components/layout src/components/ui
```

Prefer `BusinessPageTemplate`, `ManagementPageHeader`, existing domain components, Ant Design
controls, and the current sidebar/header patterns where they fit.

## Standards

- Use semantic headings, labels, buttons, and links.
- Use Ant Design for complex controls and Tailwind for layout/spacing.
- Use design tokens or theme-aware classes for colors that must support dark/tenant themes.
- Use `lucide-react` consistently with the surrounding module.
- Keep domain-specific components close to their route; promote to shared only after real reuse.
- Preserve loading, empty, validation, error, permission, and success states.
- Give icon-only controls an accessible name and tooltip.
- Make tables horizontally scrollable within their container, never at the body level.
- Use responsive spacing and breakpoint behavior; verify at 390px and desktop width.

Do not mechanically replace all `Typography`, `Space`, inline styles, or theme tokens. Change
them only when the result is clearer and behaviorally equivalent.

## Refactor loop

1. Capture the current page behavior and screenshot.
2. Identify duplicate structure and the intended shared primitive.
3. Refactor the smallest coherent area.
4. Remove dead exports only after confirming no references.
5. Verify interaction and visual behavior in a real browser.

## Verification

```bash
npm run type-check
npm run lint:check
npm run test:unit
npm run build
```

Run the focused Playwright page spec for user-facing changes.
