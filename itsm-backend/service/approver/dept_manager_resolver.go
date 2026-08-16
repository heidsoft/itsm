package approver

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/user"
)

// DeptManagerResolver resolves approvers based on department manager
type DeptManagerResolver struct{}

// NewDeptManagerResolver creates a new DeptManagerResolver
func NewDeptManagerResolver() *DeptManagerResolver {
	return &DeptManagerResolver{}
}

// GetType returns the resolver type
func (r *DeptManagerResolver) GetType() string {
	return "dept_manager"
}

// Resolve resolves department manager as approver
// It supports fallback to parent departments when:
// - The current department has no manager
// - The current department's manager is inactive
// It also detects circular parent references to prevent infinite recursion
func (r *DeptManagerResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	if appCtx.DepartmentID == 0 {
		return nil, fmt.Errorf("department_id is required for dept_manager resolver")
	}

	// Track visited departments to detect circular references
	visited := make(map[int]bool)

	return r.resolveWithVisited(ctx, client, appCtx, visited)
}

// resolveWithVisited internal method that tracks visited departments for cycle detection
func (r *DeptManagerResolver) resolveWithVisited(ctx context.Context, client *ent.Client, appCtx *ApproverContext, visited map[int]bool) ([]*ApproverInfo, error) {
	// Check for circular reference
	if visited[appCtx.DepartmentID] {
		return nil, fmt.Errorf("circular parent reference detected for department %d", appCtx.DepartmentID)
	}
	visited[appCtx.DepartmentID] = true

	// Get department with manager
	dept, err := client.Department.Query().
		Where(
			department.IDEQ(appCtx.DepartmentID),
			department.TenantIDEQ(appCtx.TenantID),
			department.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("department not found: %d", appCtx.DepartmentID)
		}
		return nil, fmt.Errorf("failed to query department: %w", err)
	}

	// If no manager, try parent department (fallback)
	if dept.ManagerID == 0 {
		if dept.ParentID > 0 {
			parentCtx := &ApproverContext{
				TenantID:     appCtx.TenantID,
				DepartmentID: dept.ParentID,
			}
			return r.resolveWithVisited(ctx, client, parentCtx, visited)
		}
		return nil, fmt.Errorf("no manager found for department %d or its ancestors", appCtx.DepartmentID)
	}

	// Get manager user info
	manager, err := client.User.Query().
		Where(
			user.IDEQ(dept.ManagerID),
			user.TenantIDEQ(appCtx.TenantID),
			user.Active(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// Manager is inactive or not found - try parent department (fallback)
			if dept.ParentID > 0 {
				parentCtx := &ApproverContext{
					TenantID:     appCtx.TenantID,
					DepartmentID: dept.ParentID,
				}
				return r.resolveWithVisited(ctx, client, parentCtx, visited)
			}
			return nil, fmt.Errorf("no active manager found for department %d or its ancestors", appCtx.DepartmentID)
		}
		return nil, fmt.Errorf("failed to query manager: %w", err)
	}

	return []*ApproverInfo{
		{
			UserID:    manager.ID,
			UserName:  manager.Name,
			UserEmail: manager.Email,
			Role:      "department_manager",
			Source:    fmt.Sprintf("department:%d", appCtx.DepartmentID),
		},
	}, nil
}
