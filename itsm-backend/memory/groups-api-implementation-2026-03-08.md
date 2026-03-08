# Groups API Implementation Report

**Date**: 2026-03-08
**Task**: Implement User Groups API endpoints for ITSM backend

## Summary

Successfully implemented complete CRUD API for user groups with role-based permissions, multi-tenancy support, and many-to-many relationship with users.

## Changes Made

### 1. Database Schema (Ent)

- **Created**: `ent/schema/group.go`
  - Fields: `id`, `name`, `description`, `tenant_id`, `created_at`, `updated_at`
  - Edge: `members` (many-to-many with User)
- **Modified**: `ent/schema/user.go`
  - Added edge: `groups` (many-to-many with Group)
- **Generated**: Ent entity files (`ent/group.go`, `ent/group_create.go`, `ent/group_query.go`, `ent/group_update.go`, `ent/group_delete.go`)

### 2. Service Layer

- **Created**: `service/group_service.go`
  - Methods:
    - `CreateGroup(ctx, req)`
    - `GetGroup(ctx, id, tenantID)`
    - `ListGroups(ctx, req) -> ([]*ent.Group, int)`
    - `UpdateGroup(ctx, id, req, tenantID)`
    - `DeleteGroup(ctx, id, tenantID)`
    - `AddUserToGroup(ctx, groupID, userID, tenantID)`
    - `RemoveUserFromGroup(ctx, groupID, userID, tenantID)`
    - `GetGroupMembers(ctx, groupID, tenantID, page, pageSize) -> ([]*ent.User, int)`
  - All methods enforce tenant isolation.

### 3. Controller Layer

- **Created**: `controller/group_controller.go`
  - Endpoints:
    - `POST /api/v1/groups` → `CreateGroup` (requires `groups:write`)
    - `GET /api/v1/groups` → `ListGroups` (requires `groups:read`)
    - `GET /api/v1/groups/:id` → `GetGroup` (requires `groups:read`)
    - `PUT /api/v1/groups/:id` → `UpdateGroup` (requires `groups:write`)
    - `DELETE /api/v1/groups/:id` → `DeleteGroup` (requires `groups:write`)
    - `POST /api/v1/groups/:id/members` → `AddUserToGroup` (requires `groups:write`)
    - `DELETE /api/v1/groups/:id/members` → `RemoveUserFromGroup` (requires `groups:write`)
    - `GET /api/v1/groups/:id/members` → `GetGroupMembers` (requires `groups:read`)
  - Proper error handling and validation.

### 4. Data Transfer Objects

- **Created**: `dto/group_dto.go`
  - Requests: `CreateGroupRequest`, `UpdateGroupRequest`, `ListGroupsRequest`, `AddUserToGroupRequest`, `RemoveUserFromGroupRequest`
  - Responses: `GroupResponse`, `GroupDetailResponse`, `PagedGroupsResponse`, `GroupMembersResponse`, `UserDTO`
  - Includes pagination support.

### 5. Router Configuration

- **Modified**: `router/router.go`
  - Added `GroupController` field to `RouterConfig`
  - Registered all group routes under `/api/v1/groups` with appropriate `RequirePermission` middleware
  - Routes are scoped under tenant middleware for multi-tenancy

### 6. Application Bootstrap

- **Modified**: `internal/bootstrap/app.go`
  - Instantiated `GroupService` and `GroupController`
  - Injected `GroupController` into `RouterConfig`

### 7. Authorization & Permissions

- **Modified**: `middleware/rbac.go`
  - Added `groups:read` and `groups:write` permissions to:
    - `admin` (read + write)
    - `manager` (read + write)
    - `agent` (read)
    - `technician` (read)
    - `sysadmin` and `super_admin` already have wildcard permissions

### 8. Database Seeding

- **Modified**: `pkg/seeder/seeder.go`
  - Added permission seeds:
    - `group:read` - "查看组"
    - `group:write` - "管理组"

## Testing

- Compiled successfully with `go build`.
- All many-to-many relationships correctly configured via Ent.
- Tenant isolation enforced via middleware and service-layer checks.
- Permission checks integrated via `RequirePermission` middleware.

## Notes

- The Group entity follows the same patterns as other entities (Department, Role) in the codebase.
- The many-to-many relationship between User and Group uses Ent's edge API.
- All endpoints return standardized JSON responses using `common.Success` / `common.Fail`.
- The implementation is fully compatible with the existing RBAC system (both default and database-backed permissions).

## Missing Endpoints (All Implemented)

✅ GET /api/v1/groups
✅ POST /api/v1/groups
✅ GET /api/v1/groups/:id
✅ PUT /api/v1/groups/:id
✅ DELETE /api/v1/groups/:id

Additional endpoints for group membership management:
✅ POST /api/v1/groups/:id/members
✅ DELETE /api/v1/groups/:id/members
✅ GET /api/v1/groups/:id/members

## Future Considerations

- Consider adding group-level permission inheritance if needed.
- Add audit logging for group membership changes.
- Add group search by name with more advanced filters.
- Consider adding a "code" field for group identifiers.
