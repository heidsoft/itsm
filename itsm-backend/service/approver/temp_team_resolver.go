package approver

import (
	"context"

	"itsm-backend/ent"
)

// TempTeamResolver resolves temporary team leaders. It currently uses the team
// manager field as the temporary owner source until a dedicated temp-team model exists.
type TempTeamResolver struct {
	teamLeader *TeamLeaderResolver
}

// NewTempTeamResolver creates a new TempTeamResolver.
func NewTempTeamResolver() *TempTeamResolver {
	return &TempTeamResolver{teamLeader: NewTeamLeaderResolver()}
}

// GetType returns the resolver type.
func (r *TempTeamResolver) GetType() string {
	return "temp_team_leader"
}

// Resolve resolves the temporary team leader as approver.
func (r *TempTeamResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	approvers, err := r.teamLeader.Resolve(ctx, client, appCtx)
	if err != nil {
		return nil, err
	}
	for _, approver := range approvers {
		approver.Role = "temp_team_leader"
	}
	return approvers, nil
}
