# Test Trend — PR Comment Pipeline

> Companion to **PR-0.5** of the Test Improvement Plan. See plan section
> "阶段 0 / PR-0.5" for the originating rationale.

## What is the trend?

Every PR run of `backend-ci`, `frontend-ci`, and `ga-gate` now produces a
single Markdown summary that captures:

- **Backend (Go):** aggregated line coverage % from `coverage.out`,
  compared against `docs/testing/coverage-baseline.json`, plus the
  bottom 5 packages.
- **Frontend (Jest):** lines / branches / functions / statements coverage
  from `itsm-frontend/coverage/coverage-summary.json`.
- **Test suites (junit.xml):** pass / fail / error / skip counts and
  total wall-clock duration.

The script that produces it is `scripts/test-trend.js`.

## How is it shipped?

1. The CI job writes the trend markdown to `output/trend/test-trend.md`
   and uploads it as the artifact `<job>-test-trend` (e.g.
   `backend-test-trend`, `frontend-test-trend`, `ga-gate-trend`).
2. The same job then calls `marocchino/sticky-pull-request-comment@v2`
   with `header: itsm-<job>-test-trend` so the trend stays attached to
   the PR thread as a sticky comment.
3. Each new push replaces the existing comment via the sticky mechanism —
   the PR has at most one trend comment per CI job at any time.

## Where does the baseline come from?

`docs/testing/coverage-baseline.json` holds the captured snapshot:

```json
{
  "go":    13.6,
  "jest":   0.47,
  "ts":   "2026-06-28T00:00:00Z"
}
```

To roll the baseline forward after a successful run:

```bash
# Read the latest coverage-summary.md
cat coverage-summary.md
# Update docs/testing/coverage-baseline.json with the new {go, jest} pair
# and bump ts to today's date.
```

PR-1.x CI updates will roll the baseline automatically once coverage is
at the 40% target.

## How to run locally

```bash
# Same args the CI uses
node scripts/test-trend.js \
  --go-coverage itsm-backend/coverage.out \
  --jest-coverage itsm-frontend/coverage/coverage-summary.json \
  --junit itsm-frontend/test-results/junit.xml \
  --baseline docs/testing/coverage-baseline.json \
  --out output/trend/test-trend.md
```

The script never fails on missing inputs — it gracefully degrades to
"no data" rows so partial CI runs still produce a useful trend.

## Permission notes

Both `backend-ci.yml` and `frontend-ci.yml` declare:

```yaml
permissions:
  contents: read
  pull-requests: write
  issues: write
```

The `pull-requests: write` + `issues: write` grants are required by
`marocchino/sticky-pull-request-comment@v2`. Without them, the
comment step fails silently and the trend only lives as a workflow
artifact.

## Permissions for `ga-gate.yml`

The `ga-gate.yml` workflow does not currently post a PR comment — its
trend artifact (`ga-gate-trend`) is intended for the eventual release
gate dashboard. PR-3.4 will revisit this when the gate job is extended
with the `playwright-golden` step.
