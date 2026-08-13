# k6 Load Test Suite

> Added by PR-0.4 (Test Improvement Plan). PR-4.1 / PR-4.2 will replace these
> placeholders with the Top-10 endpoint catalog and the `perf-budget.yml`
> CI job. See `docs/testing/perf-budget.md` (PR-4.2 deliverable).

## Layout

```
itsm-backend/tests/load/
├── README.md              # this file
├── lib/                   # shared helpers (auth, env, threshold presets)
│   ├── auth.js
│   └── thresholds.js
├── baseline/              # 1-RPU smoke tests; safe to run in dev
│   └── login.js           # sample — see PR-4.1 for the full Top-10 list
└── scenarios/             # 10/50-RPU RPS, meant for perf-budget CI only
    └── (filled by PR-4.1)
```

## Why two tiers?

- `baseline/` runs at 1 RPS and is safe to run repeatedly in dev — it
  primarily validates that the script's auth flow still works against the
  current API.
- `scenarios/` drives 10 / 50 RPS and should only run on the perf-budget
  CI shard (`.github/workflows/perf-budget.yml`, planned in PR-4.2).

## Quick start

```bash
# Local dev (assumes docker compose is running)
K6_BASE_URL=http://localhost:8090/api/v1 \
  k6 run itsm-backend/tests/load/baseline/login.js
```

## Threshold strategy (PR-4.2 contract)

P95 latency must stay within **20%** of the value captured at the last
green main commit. If a regression trips the threshold, `perf-budget.yml`
fails the build and posts the diff to the PR thread.

## Files committed by PR-4.1 (forward-looking)

| File | Endpoint | RPS tiers | Owner |
|---|---|---|---|
| `baseline/login.js` | `POST /auth/login` | 1 / 10 / 50 | be |
| `baseline/tickets-list.js` | `GET /tickets` | 1 / 10 / 50 | be |
| `baseline/incidents-list.js` | `GET /incidents` | 1 / 10 / 50 | be |
| `baseline/sla-monitor.js` | `GET /sla/monitor` | 1 / 10 / 50 | be |
| `baseline/workflow-tasks.js` | `GET /workflow/tasks` | 1 / 10 / 50 | be |
| `baseline/cmdb-topology.js` | `GET /cmdb/topology` | 1 / 10 / 50 | be |
| `baseline/knowledge-search.js` | `GET /knowledge/search` | 1 / 10 / 50 | be |
| `baseline/ai-triage.js` | `POST /ai/triage` | 1 / 10 | ai |
| `baseline/notifications-list.js` | `GET /notifications` | 1 / 10 / 50 | be |
| `baseline/dashboard-stats.js` | `GET /dashboard/stats` | 1 / 10 / 50 | be |

The set is anchored on the audit's high-traffic endpoints; it matches the
Test Improvement Plan §PR-4.1 catalog.
