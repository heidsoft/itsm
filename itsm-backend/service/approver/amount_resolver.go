package approver

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/user"
)

// AmountThreshold defines approval threshold
type AmountThreshold struct {
	MinAmount float64 `json:"minAmount"`
	MaxAmount float64 `json:"maxAmount"` // 0 means no upper limit
	Role      string  `json:"role"`
}

// AmountResolver resolves approvers based on amount thresholds
type AmountResolver struct {
	thresholds []AmountThreshold
}

// NewAmountResolver creates a new AmountResolver
func NewAmountResolver(thresholds []AmountThreshold) *AmountResolver {
	return &AmountResolver{thresholds: thresholds}
}

// GetType returns the resolver type
func (r *AmountResolver) GetType() string {
	return "amount_based"
}

// Resolve resolves approvers based on amount thresholds
func (r *AmountResolver) Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	if appCtx.Amount <= 0 {
		return nil, fmt.Errorf("amount is required for amount_based resolver")
	}

	// Find matching threshold
	var matchedThreshold *AmountThreshold
	for _, t := range r.thresholds {
		if appCtx.Amount >= t.MinAmount && (t.MaxAmount == 0 || appCtx.Amount <= t.MaxAmount) {
			matchedThreshold = &t
			break
		}
	}

	if matchedThreshold == nil {
		return nil, fmt.Errorf("no approval threshold matched for amount %.2f", appCtx.Amount)
	}
	matchedRole := user.Role(matchedThreshold.Role)
	if err := user.RoleValidator(matchedRole); err != nil {
		return nil, fmt.Errorf("invalid approval role %q: %w", matchedThreshold.Role, err)
	}

	// Find users with the matching role
	users, err := client.User.Query().
		Where(
			user.RoleEQ(matchedRole),
			user.TenantIDEQ(appCtx.TenantID),
			user.Active(true),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query users with role %s: %w", matchedThreshold.Role, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no active users found with role %s", matchedThreshold.Role)
	}

	// Return all matching users as potential approvers
	approvers := make([]*ApproverInfo, 0, len(users))
	for _, u := range users {
		approvers = append(approvers, &ApproverInfo{
			UserID:    u.ID,
			UserName:  u.Name,
			UserEmail: u.Email,
			Role:      matchedThreshold.Role,
			Source:    fmt.Sprintf("amount_threshold:%.0f-%.0f", matchedThreshold.MinAmount, matchedThreshold.MaxAmount),
		})
	}

	return approvers, nil
}
