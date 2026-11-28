# Approval Runtime Contract

> Status: current

## Decision

BPMN process execution is the target runtime source of truth for approval state:

- process definition/version describes the deployed approval flow;
- process instance and task describe current execution state;
- process variables describe execution inputs;
- process execution history and `ProcessApprovalDecision` provide immutable runtime evidence;
- the owning domain service performs the final incident/problem/change/service-request lifecycle transition.

An HTTP success, `ApprovalChain.status`, `ApprovalWorkflow`, `ChangeApproval`, or
`ServiceRequestApproval` row alone must not be interpreted as completion of a BPMN-backed
approval.

## Current Write Classes

| Class | Role during migration | New runtime instances |
| --- | --- | --- |
| `ProcessApprovalDecision` | authoritative BPMN decision evidence | allowed |
| process task/instance/history | authoritative BPMN execution | allowed |
| `ApprovalWorkflow` | legacy definition and migration input | compatibility only |
| `ApprovalChain` | configurable template/compatibility view | template only |
| `ChangeApproval` and `change_approval_chains` | domain compatibility projection/history | grandfather existing paths |
| `ServiceRequestApproval` | domain compatibility projection/history | grandfather existing paths |

The machine-readable inventory is
`itsm-backend/internal/contracts/approval_write_paths.json`. Every source file that writes one
of these models must be registered before merge.

## Invariants

1. A BPMN-backed business object has at most one active approval process instance for the same
   binding/version and occurrence.
2. An approve/reject action writes `ProcessApprovalDecision`, completes the matching process
   task, appends execution/audit history, and updates the domain state in an auditable order.
3. Legacy projections cannot independently advance a BPMN-backed domain lifecycle.
4. Tenant ID is derived from the process instance and business aggregate. Request payload,
   legacy approval row, or command payload cannot override it.
5. Direct object access, list/search, delegation, countersign, migration, and background actions
   enforce the same tenant and permission boundary.
6. Historical decisions are append-only. Corrections are new decisions/events, not destructive
   updates.

## Migration Protocol

1. **Inventory**: keep all writers registered with owner, model, mode, and intended target.
2. **Observe**: add correlation keys and shadow comparison between legacy state and BPMN state.
3. **Freeze**: behind a feature flag, prevent creation of new legacy runtime instances while
   retaining template reads and existing-instance writes.
4. **Drain/grandfather**: existing legacy instances finish on their original engine; do not
   translate an in-flight vote into a new task without an audited migration record.
5. **Switch reads**: use BPMN state for new instances; expose legacy state only as a labelled
   compatibility view.
6. **Contract**: remove legacy runtime write endpoints after the supported rollback window and
   after zero active legacy instances is proven.

## Rollback Window

Before read switching, rollback disables the freeze flag. After new BPMN instances exist,
rollback must keep them on BPMN and may only restore legacy creation for new occurrences; it
must never make one occurrence writable by both engines. Contract migration is allowed only
after evidence is archived and rollback no longer requires legacy runtime writers.

## PR-0 Exit Evidence

- inventory drift test passes;
- every registered writer has an owner and migration mode;
- new unregistered approval writer fails CI;
- no runtime writer is deleted and no in-flight instance is migrated in PR-0;
- PR-0.5 and PR-1 reference this contract for template ownership and service-request binding.

## 治理补充

`internal/bpmntemplates/bpmn` 包含变更、紧急事件、问题、发布和服务请求五个 `_cn` 变体。按租户核对技术编码、版本、部署、节点与策略差异、默认绑定以及活跃和历史引用，再决定推荐范围。展示名可本地化，内部编码及历史实例链接保持稳定。

每个审批节点以部署版本的配置确定参与人和会签、或签、拒绝策略，任务及投票记录保留运行时证据。审批链模板用于配置和显式绑定。工单原始申请人身份在业务数据上做职责分离判定。暂停、恢复、终止、重试、超时与任务完成分别核对流程状态、业务状态、审计及重复执行结果。
