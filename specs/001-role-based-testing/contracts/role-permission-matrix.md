# Contract: 角色 × 资源 × 操作 权限矩阵

**Feature**: 001-role-based-testing
**Authority**: spec 附录 A 的可机读化版本，供角色 E2E 自动断言。
**Notation**: R=read, W=write, D=delete, `-`=禁止, `(self)`=仅自己, `(tenant)`=仅本租户, `(assigned)`=仅分配给自己。

## 主矩阵

| 角色 \ 资源 | tickets | incidents | problems | changes | configuration-items | knowledge-articles | service-catalogs | sla | audit-logs | users | tenants | roles | menus | connectors | ai/audit | dashboard |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| super_admin | RWD | RWD | RWD | RWD | RWD | RWD | RWD | RWD | R | RWD | RWD | RWD | RWD | RWD | RW | R |
| tenant_admin | RWD(tenant) | RWD(tenant) | RWD(tenant) | RWD(tenant) | RWD(tenant) | RWD(tenant) | RWD(tenant) | RWD(tenant) | R(tenant) | RWD(tenant) | R(tenant) | R | R | R | RW(tenant) | R |
| sd_manager | RWD | RWD | R | R | R | R | R | RW | R | R | - | R | R | R | R | R |
| engineer | RW(assigned) | RW(assigned) | RW | RW(assigned) | RW | RW | R | R | - | R | - | R | R | R | RW | R |
| approver | R | R | R | RW(approve) | R | R | R | R | - | R | - | R | R | R | R | R |
| security | R | R | R | R | R | R | R | R | R | R | R | R | R | R | R | R |
| end_user | RW(self) | R(self) | - | - | - | R(published) | R(published) | - | - | R(self) | - | - | R(self menus) | - | - | - |

## 关键否决断言（E2E 必须验证）

| ID | 角色 | 操作 | 资源 | 期望 |
|----|------|------|------|------|
| DENY-01 | end_user | GET | `/api/v1/audit-logs` | 401/403 |
| DENY-02 | end_user | GET | `/api/v1/configuration-items` | 401/403 |
| DENY-03 | end_user | GET | `/api/v1/tenants` | 401/403 |
| DENY-04 | end_user | PUT | `/api/v1/tickets/{otherUserId}` | 401/403 |
| DENY-05 | security | PUT | `/api/v1/users/{id}` | 403 |
| DENY-06 | security | PUT | `/api/v1/sla/definitions/{id}` | 403 |
| DENY-07 | security | POST | `/api/v1/tickets` | 403 |
| DENY-08 | engineer | DELETE | `/api/v1/tenants/{id}` | 403 |
| DENY-09 | tenant_admin | GET | `/api/v1/tickets?tenant_id=other` | 空集 / 403 |
| DENY-10 | approver | PUT | `/api/v1/tickets/{id}` (非审批字段) | 403 |
| DENY-11 | * | PUT | `/api/v1/tickets/{id}` body.tenant_id = other | 服务端忽略，仍记 token tenant |

## 关键允许断言（E2E 必须验证）

| ID | 角色 | 操作 | 资源 | 期望 |
|----|------|------|------|------|
| ALLOW-01 | end_user | POST | `/api/v1/tickets` | `code=0` + `ticketNumber` |
| ALLOW-02 | end_user | GET | `/api/v1/tickets`（仅 self） | `requester_id == self` |
| ALLOW-03 | end_user | POST | `/api/v1/tickets/{self_id}/rating` | `code=0` |
| ALLOW-04 | engineer | PUT | `/api/v1/tickets/{assigned_id}` (open→in_progress) | `code=0` + audit |
| ALLOW-05 | engineer | POST | `/api/v1/incidents/{id}/root-cause` | `code=0` |
| ALLOW-06 | engineer | POST | `/api/v1/ai/audit` | `code=0` |
| ALLOW-07 | super_admin | GET | `/api/v1/auth/menus` | `length >= 20` |
| ALLOW-08 | super_admin | GET | `/api/v1/readiness/ga` | `12/12 ready` |
| ALLOW-09 | sd_manager | POST | `/api/v1/sla/monitoring` | `code=0` |
| ALLOW-10 | approver | POST | `/api/v1/approval-workflows/instances/{id}/approve` | `code=0` |
| ALLOW-11 | security | GET | `/api/v1/audit-logs` | `code=0` |
| ALLOW-12 | tenant_admin | GET | `/api/v1/users` | 仅本租户用户 |

## 用法

- 角色 E2E spec 通过此矩阵驱动表格化断言。
- 冒烟脚本 `docs/scripts/smoke-api.sh` 仅覆盖 `admin` 视角；DENY/ALLOW 表的多角色断言交给 Playwright 套件。
