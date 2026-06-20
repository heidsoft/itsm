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
func (r *DeptManagerResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	if appCtx.DepartmentID == 0 {
		return nil, fmt.Errorf("department_id is required for dept_manager resolver")
	}

	// Get department with manager
	dept, err := client.Department.Query().
		Where(
			department.IDEQ(appCtx.DepartmentID),
			department.TenantIDEQ(appCtx.TenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("department not found: %d", appCtx.DepartmentID)
		}
		return nil, fmt.Errorf("failed to query department: %w", err)
	}

	// If no manager, try parent department
	if dept.ManagerID == 0 {
		if dept.ParentID > 0 {
			// Recursive call with parent department
			parentCtx := *appCtx
			parentCtx.DepartmentID = dept.ParentID
			return r.Resolve(ctx, client, &parentCtx)
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
			return nil, fmt.Errorf("manager user not found or inactive: %d", dept.ManagerID)
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
