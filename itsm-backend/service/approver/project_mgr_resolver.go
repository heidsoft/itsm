package approver

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/project"
	"itsm-backend/ent/user"
)

// ProjectMgrResolver resolves approvers based on project manager
type ProjectMgrResolver struct{}

// NewProjectMgrResolver creates a new ProjectMgrResolver
func NewProjectMgrResolver() *ProjectMgrResolver {
	return &ProjectMgrResolver{}
}

// GetType returns the resolver type
func (r *ProjectMgrResolver) GetType() string {
	return "project_manager"
}

// Resolve resolves project manager as approver
func (r *ProjectMgrResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	if appCtx.ProjectID == 0 {
		return nil, fmt.Errorf("project_id is required for project_manager resolver")
	}

	// Get project with manager
	proj, err := client.Project.Query().
		Where(
			project.IDEQ(appCtx.ProjectID),
			project.TenantIDEQ(appCtx.TenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("project not found: %d", appCtx.ProjectID)
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	// Check if project is active
	if proj.Status != "active" {
		return nil, fmt.Errorf("project is not active: %s", proj.Status)
	}

	if proj.ManagerID == 0 {
		return nil, fmt.Errorf("no manager found for project %d", appCtx.ProjectID)
	}

	// Get manager user info
	manager, err := client.User.Query().
		Where(
			user.IDEQ(proj.ManagerID),
			user.TenantIDEQ(appCtx.TenantID),
			user.Active(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("project manager user not found or inactive: %d", proj.ManagerID)
		}
		return nil, fmt.Errorf("failed to query project manager: %w", err)
	}

	return []*ApproverInfo{
		{
			UserID:    manager.ID,
			UserName:  manager.Name,
			UserEmail: manager.Email,
			Role:      "project_manager",
			Source:    fmt.Sprintf("project:%d", appCtx.ProjectID),
		},
	}, nil
}
