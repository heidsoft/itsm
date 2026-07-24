---
name: product-roadmap-manager
description: Maintain the ITSM roadmap, release scope, priorities, milestones, dependencies, risks, and evidence-based progress. Use for roadmap updates, release planning, feature prioritization, milestone reviews, or aligning work with the ServiceNow-class AI-native product direction.
---

# Product Roadmap Manager

## Source of truth

Read `AGENTS.md`, `docs/roadmap.md`, relevant review reports, and current code/tests before
changing roadmap status. Use dates from the current environment. Do not preserve stale quarter
labels, fabricated team capacity, or unsupported completion percentages.

## Planning horizons

Organize work by product stage rather than invented calendar promises:

- Current hardening: coverage, controller/service boundaries, RBAC, tenant isolation, audit,
  connector lifecycle, and reliable deployment.
- Next capability: connector marketplace, skill registry, measurable AI evaluation, production
  Feishu/WeCom/DingTalk integrations.
- Longer horizon: ServiceNow-class extensibility, MSP/SaaS scale, CLI operations, and ecosystem
  marketplaces.

## Prioritize

Score each item with evidence:

```text
Customer/business impact: 1-5
Security/compliance risk: 1-5
ITIL/process criticality: 1-5
Strategic alignment: 1-5
Effort/uncertainty: 1-5
Dependencies:
Acceptance evidence:
```

Security, cross-tenant, audit, and primary workflow defects outrank cosmetic additions.

## Roadmap item format

Every item needs:

- outcome and target persona;
- scope and explicit non-goals;
- owning module/extension point;
- dependencies and migration risk;
- measurable acceptance criteria;
- test/observability evidence;
- status: proposed, planned, in progress, blocked, or done.

Mark done only when code, tests, docs, and operational verification exist. Link to files,
issues, or reports rather than copying volatile detail into the Skill.

## Product constraints

Extend existing BPMN, connector, skill, plugin, and CLI extension points. Preserve ITIL
semantics, auditability, tenant isolation, private deployment, SaaS, and SaaS+MSP compatibility.
AI remains controlled decision support unless an audited permission model explicitly allows
automation.
