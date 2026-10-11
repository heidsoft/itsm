# Coverage Report Conventions

> Status: current

> Companion to `scripts/coverage-summarize.js` and `scripts/coverage-report.sh`.
> See plan section "阶段 0" / PR-0.1 for context.

## Quick start

```bash
bash scripts/coverage-report.sh                    # run everything; emit coverage-summary.md
SKIP_BACKEND=1 bash scripts/coverage-report.sh     # frontend-only
SKIP_FRONTEND=1 bash scripts/coverage-report.sh    # backend-only
BASELINE_JSON=path/to.json bash scripts/coverage-report.sh  # override baseline (default docs/testing/coverage-baseline.json)
```

> 没有 `make coverage-report` 目标；根 Makefile 只有 `test-backend` / `test-frontend` / `lint-backend` / `build-backend`。该脚本目前是本地入口，CI 里只调 `node scripts/test-trend.js`。

Outputs:

| File | Source | Purpose |
|---|---|---|
| `itsm-backend/coverage.out` | `go test -coverprofile=...` | Per-statement coverage, machine-readable |
| `itsm-backend/coverage.html` | `go tool cover -html` | Human-browsable HTML |
| `itsm-frontend/coverage/coverage-summary.json` | Jest | Pre-existing Jest summary |
| `itsm-frontend/coverage/lcov.info` | Jest | Pre-existing LCOV |
| `coverage-summary.md` | `scripts/coverage-summarize.js` | Unified Markdown |
| `output/coverage/*.log` | wrapper | Per-stage logs |

## Baseline

The first run writes `docs/testing/coverage-baseline.json` with placeholder zeros.
After a green CI run, copy `{go, jest}` from `coverage-summary.md` into the JSON
so subsequent runs show deltas:

```bash
# After a successful coverage-report run:
# 没有 make coverage-baseline 目标；直接改 docs/testing/coverage-baseline.json
#   或把 coverage-summary.md 的 {go, jest} 数字写回该文件
```

`scripts/coverage-summarize.js` reads the baseline JSON via `--baseline` and
annotates the combined-totals table with `↑/↓/±` per delta.

## Why two surfaces?

The repo mixes Go (backend) and TypeScript (frontend). Go coverage reports
files-by-package; Jest coverage reports files-by-source. The summarizer keeps
both representations intact and only merges the totals — no double-counting.

## When does CI run this?

`.github/workflows/coverage-report.yml` is added in PR-0.5; see that PR for
artifact upload and PR-comment wiring.
