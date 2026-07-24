---
name: backend-testing-guide
description: Write, repair, and review Go tests for the ITSM backend, including services, controllers, middleware, repositories, RBAC, tenant isolation, contracts, workflows, CMDB, connectors, and AI audit behavior. Use for backend test failures, regression coverage, or test design.
---

# Backend Testing Guide

## Choose the test boundary

- Pure helper or validation: table-driven unit test beside the source file.
- Service or repository behavior: use `enttest.NewClient` or the repository's existing fixture.
- Controller behavior: bind realistic Gin requests and assert the standard response envelope.
- Route or DTO compatibility: add coverage under `tests/contract`.
- Authorization or tenant boundaries: add explicit deny and cross-tenant cases under
  `tests/rbac` or the owning package.

Copy the nearest current test fixture; generated Ent fields and service constructors change
over time.

## Enterprise invariants

Always test relevant invariants:

- every tenant query is scoped and cross-tenant IDs fail closed;
- lifecycle transitions are explicit and invalid transitions fail;
- controllers return DTOs, never Ent models;
- workflow, approval, AI, connector, and bulk actions create audit evidence;
- retryable initialization/import/discovery is idempotent;
- secrets and private content do not leak into responses or logs.

For AI behavior, test deterministic fallback, low confidence, disabled provider, and audit
metadata. For CMDB traversal, test cycle and depth/size limits.

## Execution

```bash
cd itsm-backend
go test ./service -run 'TestName' -count=1
go test ./controller ./middleware ./repository/... -count=1
go test ./tests/contract ./tests/rbac -count=1
go test ./...
```

Use `require` for setup/preconditions and `assert` for independent outcome checks. Generate
unique values without relying on test order. Do not connect unit tests to a developer or
production database.

## Failure triage

1. Re-run the narrow test with `-count=1 -v`.
2. Identify whether the failure is fixture drift, contract drift, or product behavior.
3. Fix product behavior when the test protects a documented invariant.
4. Change the test only when the contract or intended behavior changed.
5. Run the owning package, then `go test ./...`.
