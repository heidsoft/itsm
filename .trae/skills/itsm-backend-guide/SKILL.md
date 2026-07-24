---
name: itsm-backend-guide
description: Implement and review enterprise ITSM backend changes in Go, Gin, and Ent across DTOs, services, controllers, routes, workflows, CMDB, SLA, RBAC, tenants, connectors, and AI features. Use for backend features, bug fixes, refactors, or architecture decisions in itsm-backend.
---

# ITSM Backend Guide

## Preserve layer ownership

- Controller: bind/validate, read authenticated context, call a service, map errors, return the
  standard envelope.
- Service/domain: enforce lifecycle rules, permission/tenant checks, transactions, audit, and
  orchestration.
- DTO/mapper: define the public `camelCase` contract and convert Ent models.
- Ent/repository: persist/query with explicit tenant predicates.
- Router/bootstrap: register handlers and dependencies without duplicating business logic.

Never return an Ent model from a controller.

```go
common.Success(c, dto.ToTicketResponse(ticket))
```

## Change workflow

1. Trace the current route → controller → service → schema/DTO path.
2. Define request and response DTOs using `camelCase` JSON tags.
3. Add tenant/account ownership to new persistent domain data.
4. Put multi-record mutations in a service transaction.
5. Add audit records for lifecycle, workflow, approval, AI, connector, and bulk actions.
6. Register the route and update frontend types/client when the contract changes.
7. Add regression tests before broad verification.

After changing Ent schemas, use the repository's current generation/migration mechanism; do
not hand-edit generated `ent/` files.

## Domain constraints

- Keep ticket, incident, problem, change, release, and service request lifecycle rules distinct.
- Reuse BPMN/process records instead of introducing a second approval engine.
- Preserve SLA timestamps and policy bindings as authoritative state.
- Treat CMDB CI schema, CI instance, relationship, discovery, reconciliation, topology, and
  impact analysis as separate concepts.
- Route LLM calls through the gateway; retain model/provider, confidence, version, decision,
  operator feedback, and fallback behavior.
- Keep connector secrets masked and actions permission-checked/audited.

Fail closed on cross-tenant or permission ambiguity.

## Verification

```bash
cd itsm-backend
go test ./<owning-package> -count=1
go test ./tests/contract ./tests/rbac -count=1
go test ./...
```

For contract changes, also run frontend type checks and the focused Playwright spec.
