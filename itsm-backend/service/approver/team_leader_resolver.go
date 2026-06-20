package approver

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/team"
	"itsm-backend/ent/user"
)

// TeamLeaderResolver resolves approvers based on team leader
type TeamLeaderResolver struct{}

// NewTeamLeaderResolver creates a new TeamLeaderResolver
func NewTeamLeaderResolver() *TeamLeaderResolver {
	return &TeamLeaderResolver{}
}

// GetType returns the resolver type
func (r *TeamLeaderResolver) GetType() string {
	return "team_leader"
}

// Resolve resolves team leader as approver
func (r *TeamLeaderResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	if appCtx.TeamID == 0 {
		return nil, fmt.Errorf("team_id is required for team_leader resolver")
	}

	// Get team with leader
	teamEntity, err := client.Team.Query().
		Where(
			team.IDEQ(appCtx.TeamID),
			team.TenantIDEQ(appCtx.TenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("team not found: %d", appCtx.TeamID)
		}
		return nil, fmt.Errorf("failed to query team: %w", err)
	}

	if teamEntity.ManagerID == 0 {
		return nil, fmt.Errorf("no leader found for team %d", appCtx.TeamID)
	}

	// Get leader user info
	leader, err := client.User.Query().
		Where(
			user.IDEQ(teamEntity.ManagerID),
			user.TenantIDEQ(appCtx.TenantID),
			user.Active(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("team leader user not found or inactive: %d", teamEntity.ManagerID)
		}
		return nil, fmt.Errorf("failed to query team leader: %w", err)
	}

	return []*ApproverInfo{
		{
			UserID:    leader.ID,
			UserName:  leader.Name,
			UserEmail: leader.Email,
			Role:      "team_leader",
			Source:    fmt.Sprintf("team:%d", appCtx.TeamID),
		},
	}, nil
}
