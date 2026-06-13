# Phase 1 — Data Model: 测试涉及的关键实体

**Feature**: 001-role-based-testing
**Date**: 2026-06-13
**Status**: Complete

## 说明

本 feature 不新增业务实体；以下列出 7 角色测试 / FLOW-1..10 引用到的现有实体、关键字段、状态机与多租户键，便于角色 E2E 与冒烟脚本编写时定位 schema。

数据来源：`itsm-backend/ent/schema/*.go`、`docs/ITSM-基于角色视角的产品测试方案.md` 附录 B。

---

## 实体一览（按测试触达频次排序）

| 实体 | 关键字段 | 状态机 | 多租户键 | 测试触达 |
|------|----------|--------|----------|----------|
| Tenant | name, code, domain, status | active / suspended | self | TA-* / FLOW-9 |
| User | username, email, name, password_hash, role, active, tenant_id | active / disabled | tenant_id | 全角色登录 |
| Role | name, code, description, tenant_id | — | tenant_id | SA-04 / 角色矩阵 |
| Permission | resource, action | — | global | SA-04 |
| Menu | name, path, parent_id, sort_order, tenant_id, is_visible, is_enabled | visible / hidden | tenant_id | SA-01 / FR-1004 |
| Ticket | title, ticket_number, priority, status, requester_id, assignee_id, tenant_id, affected_ci_ids | open → in_progress → resolved → closed（reopen 可逆） | tenant_id | EU-* / EN-* / SD-* |
| Incident | + ticket 字段 + reporter_id, root_cause | open → investigating → resolved → closed | tenant_id | EN-05/06 / FLOW-3 |
| Problem | title, status, related_incident_ids | identified → analyzing → known_error → resolved | tenant_id | EN-06 / FLOW-3 |
| Change | title, type(standard/normal/emergency), status, created_by, related_problem_id | draft → pending_approval → approved → in_progress → completed / rejected | tenant_id | EN-07 / AP-* / FLOW-3,5 |
| ConfigurationItem | name, ci_type_id, status, attributes, relationships, tenant_id | active / retired | tenant_id | EN-04 / FLOW-8 |
| CIType | name, schema, tenant_id | — | tenant_id | TA-06 |
| KnowledgeArticle | title, content, status, tags, tenant_id | draft → published → archived | tenant_id | EU-05/06 / EN-08 / FLOW-10 |
| ServiceCatalog | name, status, tenant_id | published / archived | tenant_id | TA-05 / EU-04 |
| ServiceRequest | catalog_id, requester_id, status, requires_approval, expire_at | submitted → approved → fulfilling → completed | tenant_id | EU-04 / FLOW-4 |
| SLADefinition | name, priority, response_minutes, resolve_minutes, tenant_id | — | tenant_id | SA-09 / SD-04 |
| SLAMonitoring | ticket_id, sla_id, breach_at, status | on_track / at_risk / breached | tenant_id | SD-04 / FLOW-7 |
| SLAAlertHistory | sla_id, ticket_id, channel, sent_at | — | tenant_id | FLOW-7 |
| ApprovalWorkflow | name, nodes, manager_timeout, tenant_id | — | tenant_id | TA-03 / AP-* |
| ApprovalInstance | workflow_id, target_id, current_node, status | pending → approved / rejected / escalated | tenant_id | AP-* / FLOW-3 |
| Connector | type(feishu/dingtalk/wecom/console/webhook), config, status | installed → enabled → degraded → disabled → uninstalled | tenant_id | SA-07 / FLOW-7 |
| AIAudit | ticket_id, suggestion, accepted, latency_ms, timeout | — | tenant_id | EN-10 / FLOW-6 |
| AuditLog | actor_id, tenant_id, resource, action, before, after, created_at | — append-only — | tenant_id | SE-* / 全流程 |

---

## 关键状态机

### Ticket / Incident

```
open → in_progress → resolved → closed
                ↑                  │
                └──── reopen ──────┘
```

- `reopen` 仅允许从 `closed` 回到 `in_progress`。
- 状态变更 MUST 写 AuditLog（FR-803）。

### Change

```
draft → pending_approval → approved → in_progress → completed
                  │
                  └→ rejected
```

- `type=standard` 跳过 `pending_approval`，由系统自动 approved。
- `type=emergency` 进入快速通道但仍需事后审批。

### KnowledgeArticle

```
draft → published → archived
```

- end_user 在 `/knowledge-articles` 仅可见 `published`。

### Connector

```
installed → enabled ⇄ degraded → disabled → uninstalled
```

- `health` 检查返回 `ready / degraded / unhealthy` 三态（FR-703）。

### ApprovalInstance

```
pending → approved
        → rejected
        → escalated → pending(next_approver)
```

- 任意节点 `rejected` 整个工作流终止（AP-03）。
- 节点超时 → escalated 或 hung，记 `escalation_history`（AP-04 / FR-404）。

---

## 多租户隔离约束（FR-601 / FLOW-9）

所有写操作的 `tenant_id` 强制取 token 中值，忽略客户端传入：

```go
// 伪代码示意
ticket.TenantID = ctx.GetInt("tenant_id") // 来自 JWT，不取请求体
```

读操作必须在 query 中带 `WHERE tenant_id = $token_tenant_id`。

测试断言（FLOW-9）：

- tenant1.admin 创建工单 T1 → tenant2.admin GET 不可见。
- 客户端篡改 `tenant_id` 提交 → 服务端忽略，工单仍记 token 租户。

---

## 测试数据准备（最小集）

```sql
-- 至少 2 个租户
INSERT INTO tenants(name, code, domain, status) VALUES
  ('Tenant A', 'TENA', 'a.example.com', 'active'),
  ('Tenant B', 'TENB', 'b.example.com', 'active');

-- 每租户至少 1 engineer + 1 manager + 1 tenant_admin（由 seeder 补）
-- KB seed 已有 23 条；服务目录 22+5；SLA 6 条；标准变更 5 条 — 不需重建。
```

测试期间使用专用租户 `tenant_test`，避免污染生产数据。
