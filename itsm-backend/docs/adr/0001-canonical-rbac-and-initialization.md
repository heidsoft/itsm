# ADR 0001: Canonical RBAC and initialization authority

Status: Accepted

## Decision

Database RBAC is the production authorization authority. `roles`,
`permissions`, and `role_permissions`, scoped by tenant, determine resource
authorization. Endpoint ACL maps an HTTP operation to the same permission code;
menu visibility consumes that code but never grants access.

Production uses `PermissionConfigModeDBOnly`. An empty result, revoked binding,
database error, uninitialized tenant, or cache miss followed by an empty reload
denies access. Compiled `RolePermissions` are development/test fixtures only.

`users.role` remains a one-release compatibility projection. Where it conflicts
with role relationships/database grants, database grants win. New authorization
logic must not add privileges based only on `users.role`.

`super_admin` bypass is limited to the platform tenant and must not imply access
to a customer tenant. Break-glass use is offline, time-bound, and audited.

## Initialization ownership

- Permission codes and Endpoint ACL entries are product-managed P0 records.
- System roles, exact role grants, and menus are T0 records installed per tenant.
- Product-managed role grants are reconciled exactly, including revocation.
- Customer-created roles and menus are not removed or overwritten.
- Permission/cache invalidation must occur after a successful committed
  reconcile; failed initialization does not publish invalidations.
- The final active platform administrator cannot be revoked by an automated
  reconcile.

## Migration route

1. Continue writing compatibility tenant-scoped `permissions`.
2. Populate and verify the canonical permission catalog and endpoint mapping.
3. Compare dual-read results and route/ACL/menu coverage in CI.
4. Switch all authorization reads to canonical DB grants.
5. Stop writing `users.role`; remove it only after a full release of read-only
   compatibility.

Unknown or conflicting identities fail closed and require an explicit repair
plan. No initializer silently restores compiled permissions.

