# Change Approval Workflow Implementation Report
**Date:** 2026-03-08  
**Status:** Completed

## Summary
Implemented the missing endpoint POST `/api/v1/changes/:id/submit` to submit a change for approval, which previously returned 404.

## Changes Made

### 1. DTO Definition (`dto/change_dto.go`)
Added `SubmitChangeRequest` struct:
```go
type SubmitChangeRequest struct {
	ApproverIDs []int  `json:"approver_ids"` // 审批人ID列表
	Comment     string `json:"comment"`      // 提交说明（可选）
}
```

### 2. Handler (`internal/domain/change/handler.go`)
Added `SubmitChange` method:
- Extracts change ID from URL params, tenant ID and user ID from context
- Binds request body to `SubmitChangeRequest`
- Calls `svc.SubmitChange` with context, change ID, tenant ID, user ID, and request
- Returns the updated change DTO on success

### 3. Service (`internal/domain/change/service.go`)
- Imported `itsm-backend/dto` package
- Implemented `SubmitChange` method:
  - Fetches the change, verifies it exists and status is 'draft'
  - Updates status to 'pending'
  - Creates `ApprovalRecord` entries for each approver ID from request with status 'pending'
  - Logs the submission event
  - Returns the updated change

### 4. Router (`router/router.go`)
Registered the new route under the changes group:
```go
changes.POST("/:id/submit", middleware.RequirePermission("change", "write"), config.ChangeHandler.SubmitChange)
```
- Uses `RequirePermission("change", "write")` consistent with other state-changing endpoints
- Route path `/changes/:id/submit`

## Behavior
- **Precondition:** Change must be in `draft` status
- **Outcome:** Change status transitions to `pending`
- **Approval Records:** One record per provided approver ID is created with status `pending` and optional comment
- **Permissions:** Requires `change:write` permission
- **Notification:** Logged (notifier can be added later)

## Example Request
```http
POST /api/v1/changes/123/submit
Authorization: Bearer <token>
Content-Type: application/json

{
  "approver_ids": [5, 8],
  "comment": "Ready for review"
}
```

## Response
Returns the updated change object in standard response format.

## Notes
- The implementation assumes the workflow of approvers is determined by the caller (submitter). Future enhancements could derive approvers automatically from risk level, type, or configured workflow.
- Notification to approvers is optional and currently only logs; can be extended by integrating with the notification service.

## Verification
- Route no longer returns 404
- Change status updates correctly
- Approval records are created for specified approvers

---

**Implementation completed as per requirements.**