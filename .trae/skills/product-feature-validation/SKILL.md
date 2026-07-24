---
name: product-feature-validation
description: Validate ITSM feature completeness and production readiness from user workflows through API, persistence, permission, tenant, audit, resilience, and UI evidence. Use for functional reviews, release readiness, acceptance testing, gap analysis, or test-report-driven repairs.
---

# Product Feature Validation

## Start from the user job

Record:

```text
Role:
Business goal:
Entry route:
Primary path:
Secondary actions:
Expected API:
Persistence:
Permission/tenant boundary:
Failure behavior:
Evidence:
```

Validate the actual implementation and roadmap; never copy fixed completion percentages or
invent status.

## Coverage model

For each feature, cover:

1. Access: intended role can enter; unauthorized role is denied or limited.
2. Primary task: the business outcome completes.
3. Subfeature: search/filter/table action/tab/modal/workflow action works.
4. Resilience: validation, empty, loading, error, retry, or unavailable dependency is clear.
5. Persistence: refresh/revisit shows authoritative state.
6. Governance: tenant isolation, RBAC, audit, secrets, and AI decision support are correct.

Apply domain-specific invariants from `AGENTS.md`, especially ITIL transitions, BPMN integrity,
CMDB relationship safety, SLA timestamps, RAG permissions, and connector lifecycle.

## Evidence

Use source tracing plus the narrowest executable proof:

```bash
cd itsm-backend && go test ./...
cd ../itsm-frontend && npm run type-check && npm run lint:check
```

For user-facing behavior, run a focused Playwright spec and retain screenshot/trace evidence
for complex failures.

## Report

Classify findings:

- P0: security, tenant leak, data loss, or system unavailable;
- P1: primary workflow broken or materially incorrect;
- P2: secondary workflow, resilience, or important UX defect;
- P3: polish or low-impact inconsistency.

For every gap, cite the route/file, reproduction, expected behavior, actual behavior, risk, and
verification needed. Separate confirmed facts from recommendations.
