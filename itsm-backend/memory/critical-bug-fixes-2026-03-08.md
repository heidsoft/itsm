# Critical Bug Fixes Report
**Date:** 2026-03-08  
**Status:** Fixed  
**Fixed by:** Subagent  

---

## Summary
Fixed three critical backend bugs identified during E2E testing:

1. Knowledge Stats SQL Error (aggregate column mapping)
2. Group Creation Missing TenantID (auto-fill from context)
3. CMDB Asset Creation Invalid Request Body (DTO validation mismatch)

---

## BUG 1: Knowledge Stats SQL Error

### Problem
- **File:** `internal/domain/knowledge/repository_impl.go`
- **Error:** `sql/scan: missing struct field for column: sum (sum)`
- **Root cause:** The `GetStats` method used `Aggregate(ent.Sum(...))` which returns a column named "sum" by default, but the intermediate structs `ViewSum` and `LikeSum` expected fields named `TotalViews` and `TotalLikes` respectively (with JSON tags `total_views`, `total_likes`). The column name mismatch caused the scan to fail.

### Fix
Used `ent.As()` to alias the aggregate column names to match the struct fields:
```go
// Before
Aggregate(ent.Sum(knowledgearticle.FieldViewCount))
// After
Aggregate(ent.As(ent.Sum(knowledgearticle.FieldViewCount), "total_views"))
```

Applied the same pattern for the like count aggregate.

### Files Changed
- `internal/domain/knowledge/repository_impl.go`

### Test Curl Command
```bash
curl -X GET "http://localhost:8080/api/v1/knowledge/stats" \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "X-Tenant-ID: 1" \
  -H "Content-Type: application/json"
```
Expected: 200 OK with stats JSON containing `total_views` and `total_likes` fields.

---

## BUG 2: Group Creation Missing TenantID

### Problem
- **File:** `controller/group_controller.go` and `dto/group_dto.go`
- **Error:** `"TenantID required"` (validation failure)
- **Root cause:** The `CreateGroupRequest` DTO required `tenant_id` to be provided by the frontend. However, the frontend was not sending this field, expecting the backend to auto-fill it from the authenticated user's context. The controller also enforced that the provided tenant_id must match the authenticated tenant, which caused a validation error when the field was missing.

### Fix
1. Made `TenantID` optional in `CreateGroupRequest` by removing the `required` binding tag.
2. Modified the controller to automatically set `req.TenantID` from the authenticated user's context (extracted from JWT via middleware) instead of requiring the frontend to provide it.

### Files Changed
- `dto/group_dto.go` (changed `TenantID` binding from `required` to `omitempty`)
- `controller/group_controller.go` (replaced the validation check with auto-fill from middleware)

### Test Curl Command
```bash
curl -X POST "http://localhost:8080/api/v1/groups" \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "X-Tenant-ID: 1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Group",
    "description": "Test group created without tenant_id"
  }'
```
Expected: 200 OK with group data, `tenant_id` automatically set to 1.

---

## BUG 3: CMDB Asset Creation Invalid Request Body

### Problem
- **File:** `dto/cmdb_dto.go` (CreateCIRequest)
- **Error:** `POST /api/v1/cmdb/cis` returns 400 Bad Request
- **Root cause:** The `CreateCIRequest` DTO had two mismatches with frontend expectations:
  1. The `Type` field was marked as `binding:"required"` but the frontend does not send this field (it's derived from `ci_type_id`).
  2. The `Status` field had `binding:"required,oneof=active inactive maintenance"` which excluded valid database statuses like `"operational"` (the default in the schema).
  These validation constraints caused the binding to fail when the frontend sent a typical payload.

### Fix
1. Made `Type` optional by moving it to the end with `json:"type,omitempty"` (no required tag).
2. Removed the `oneof` restriction from `Status`, keeping only `binding:"required"` to accept any string status (including "operational").

### Files Changed
- `dto/cmdb_dto.go` (updated `CreateCIRequest` struct tags)

### Test Curl Command
```bash
curl -X POST "http://localhost:8080/api/v1/cmdb/cis" \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "X-Tenant-ID: 1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test CI",
    "ci_type_id": 1,
    "status": "operational",
    "environment": "production"
  }'
```
Expected: 200 OK with created CI data. Note: `ci_type_id` must refer to an existing CI type.

---

## Notes
- All fixes were auto-applied without requiring user confirmation.
- Test curl commands assume the server is running on `localhost:8080` and that a valid JWT token is available. Adjust host/port as needed.
- The fixes align the backend API with the frontend's expectations and ensure proper data mapping and validation.
