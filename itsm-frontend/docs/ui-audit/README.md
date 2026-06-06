# Frontend UI Audit Inventory

Generated at: 2026-06-06T23:09:18.803Z

## Coverage

- Total pages: 111
- Batch 1 pages: 43
- Batch 2 pages: 21
- Batch 3 pages: 47

## Highest Complexity Pages

| Path | Batch | Type | Lines | Issues |
| --- | --- | --- | ---: | ---: |
| `workflow/page.tsx` | batch-1 | list | 1160 | 5 |
| `tickets/templates/page.tsx` | batch-1 | list | 997 | 2 |
| `templates/page.tsx` | batch-3 | list | 975 | 3 |
| `profile/page.tsx` | batch-3 | list | 919 | 2 |
| `workflow/instances/page.tsx` | batch-1 | list | 915 | 5 |
| `admin/workflows/page.tsx` | batch-3 | admin | 847 | 4 |
| `cmdb/page.tsx` | batch-1 | list | 826 | 4 |
| `admin/service-catalogs/page.tsx` | batch-3 | admin | 799 | 2 |
| `admin/escalation-rules/page.tsx` | batch-3 | admin | 772 | 2 |
| `admin/sla-definitions/page.tsx` | batch-3 | admin | 753 | 2 |
| `tickets/[ticketId]/page.tsx` | batch-1 | list | 753 | 2 |
| `notifications/page.tsx` | batch-3 | list | 741 | 3 |
| `admin/users/page.tsx` | batch-3 | admin | 695 | 3 |
| `admin/permissions/page.tsx` | batch-3 | admin | 675 | 2 |
| `workflow/versions/page.tsx` | batch-1 | list | 655 | 3 |

## Highest Complexity workflow/cmdb Components

| Path | Domain | Lines | Issues |
| --- | --- | ---: | ---: |
| `src/components/workflow/WorkflowEngine.tsx` | workflow | 919 | 2 |
| `src/components/workflow/designer/WorkflowDesigner.tsx` | workflow | 712 | 3 |
| `src/components/cmdb/TopologyGraph.tsx` | cmdb | 509 | 1 |
| `src/components/cmdb/CIRelationshipManager.tsx` | cmdb | 430 | 0 |
| `src/components/workflow/BPMNDesigner.tsx` | workflow | 337 | 1 |
| `src/components/cmdb/CIList.tsx` | cmdb | 324 | 1 |
| `src/components/workflow/designer/WorkflowProperties.tsx` | workflow | 322 | 0 |
| `src/components/workflow/designer/WorkflowContext.tsx` | workflow | 238 | 0 |
| `src/components/cmdb/ci-detail/sections/CIImpactAnalysisTab.tsx` | cmdb | 189 | 0 |
| `src/components/workflow/designer/WorkflowNodePalette.tsx` | workflow | 187 | 0 |
| `src/components/cmdb/ci-detail/CIDetail.tsx` | cmdb | 169 | 0 |
| `src/components/workflow/designer/WorkflowToolbar.tsx` | workflow | 162 | 0 |
| `src/components/cmdb/ci-detail/__tests__/CIImpactAnalysisTab.test.tsx` | cmdb | 161 | 0 |
| `src/components/cmdb/ci-detail/__tests__/CIChangeHistoryTab.test.tsx` | cmdb | 157 | 0 |
| `src/components/cmdb/ci-detail/__tests__/CIBasicInfo.test.tsx` | cmdb | 156 | 0 |

## Notes

- Full machine-readable audit ledger lives in `docs/ui-audit/inventory.json`.
- Severity and issue categories are heuristic seeds for manual review, not final product decisions.
