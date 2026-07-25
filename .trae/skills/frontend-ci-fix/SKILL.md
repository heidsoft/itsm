---
name: frontend-ci-fix
description: Diagnose and fix ITSM frontend CI failures involving npm, lockfiles, ESLint, TypeScript, Jest, Playwright, Next.js builds, artifacts, or GitHub Actions. Use when CI is red, a frontend check differs locally, or a workflow needs hardening.
---

# Frontend CI Fix

## Reproduce the exact job

Read the failing workflow and `itsm-frontend/package.json` before changing code. Match its
working directory, Node version, package manager, environment, and command.

Run locally in increasing cost order:

```bash
cd itsm-frontend
npm ci
npm run type-check
npm run lint:check
npm run test:ci
npm run build
```

For browser failures, start or provide the backend on `8090`, then run the exact Playwright
project/spec with `--workers=1` for shared authenticated state.

## Diagnose before editing

- Lockfile failure: verify `package-lock.json` matches `package.json`; do not bypass with
  `--force` or `--legacy-peer-deps`.
- Type/Lint failure: fix the narrow source error; do not weaken compiler or lint rules.
- Jest failure: reproduce the individual test and distinguish missing browser polyfills from
  product regressions. A focused run can pass every selected assertion yet exit non-zero
  because global coverage thresholds apply to the partial suite; inspect both results.
- Playwright failure: inspect trace, screenshot, console, network response, selector, and
  first-compile timing.
- Build-only failure: check server/client boundaries, dynamic route usage, environment access,
  and standalone output.
- Cache failure: correct the path relative to the workflow's working directory; do not remove
  caching unless it is genuinely unsafe.

Do not skip E2E merely because the backend is missing. Start the required service or make the
job dependency explicit.

For focused Jest diagnosis, use `npx jest --runInBand --coverage=false <test-files>` when the
purpose is regression behavior rather than repository-wide coverage. Never change global
coverage thresholds merely to make a partial run green.

## Keep changes minimal

Avoid blanket timeout increases, `passWithNoTests` additions, test deletion, or dependency
upgrades unrelated to the failure. Preserve artifacts on failure and use stable role/test-id
selectors.

## Completion

Re-run the exact failed command and the adjacent gate. Report the original cause, changed
files, commands run, and any external service still required.
