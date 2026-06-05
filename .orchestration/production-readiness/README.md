# Production Readiness Orchestration

This folder is the control plane for turning the repository into a production-usable system.

## Layout

- `plan.json` — worker session definition for dmux or manual worktree execution
- `tasks/` — per-worker prompts and context
- `handoff/` — results written by workers
- `status/` — short machine-readable progress snapshots

## How To Use

1. Review `docs/delivery/production-readiness-program.md`.
2. Pick the next phase and only assign independent work.
3. Start workers with separate worktrees when file overlap is likely.
4. Require each worker to update its `status/*.md` and `handoff/*.md`.
5. Merge only after verification commands are captured.

## Worker Contract

Each worker must report:

- objective
- files changed
- commands run
- unresolved risk
- next recommended step

## Merge Policy

- backend and frontend may run in parallel
- release engineering must merge before release candidate creation
- security findings block release if severity is P0 or P1
- production scripts and monitoring changes require dry-run verification
