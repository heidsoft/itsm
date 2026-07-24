---
name: env-troubleshooting
description: Diagnose ITSM local development and CI environment failures involving Go/Node versions, npm dependencies, ports, environment files, databases, Redis, Docker networks, frontend-backend routing, or service health. Use when install, startup, build, test, or local integration fails.
---

# Environment Troubleshooting

## Diagnose read-only first

Collect evidence before changing the environment:

```bash
node --version
npm --version
go version
go env GOROOT GOTOOLCHAIN
lsof -nP -iTCP:3000 -sTCP:LISTEN
lsof -nP -iTCP:8090 -sTCP:LISTEN
curl -i http://localhost:8090/api/v1/health
```

Read `package.json`, lockfiles, `go.mod`, `.env.example`, Compose files, and the exact error.
Do not change `go.mod`, kill processes, delete `node_modules`, clear caches, or force dependency
resolution until the cause is confirmed.

## Common boundaries

- Frontend defaults to `3000`; backend defaults to `8090`.
- Browser API configuration uses `NEXT_PUBLIC_API_URL` or the project's same-origin proxy.
- Production Compose must receive an explicit `--env-file`.
- Development and production containers may use different Docker networks.
- Secrets belong in local environment files and must never be printed or committed.

## Recovery order

1. Correct the command/working directory.
2. Correct missing or invalid environment values.
3. Resolve the port owner or choose an explicit alternate port.
4. Align the toolchain with the repository without weakening version requirements.
5. Use `npm ci` when the lockfile is authoritative; repair lockfile drift deliberately.
6. Verify dependent database/Redis/network health.
7. Re-run the smallest failed command and health endpoint.

Avoid `npm --force`, `--legacy-peer-deps`, global Go environment mutation, or broad cleanup as
default fixes.

## Docker verification

```bash
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
docker logs <container> --tail 50
docker inspect <container> --format '{{json .NetworkSettings.Networks}}'
```

Report the root cause, current service endpoints, exact remediation, and any remaining external
dependency.
