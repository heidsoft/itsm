package approver

import (
	"context"
	"fmt"

	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ApproverInfo contains resolved approver information
type ApproverInfo struct {
	UserID    int    `json:"userId"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail,omitempty"`
	Role      string `json:"role,omitempty"`
	Source    string `json:"source"` // department, team, project, role, user, amount
}

// ApproverContext provides context for approver resolution
type ApproverContext struct {
	TenantID     int                    `json:"tenantId"`
	TicketID     int                    `json:"ticketId,omitempty"`
	RequesterID  int                    `json:"requesterId,omitempty"`
	DepartmentID int                    `json:"departmentId,omitempty"`
	TeamID       int                    `json:"teamId,omitempty"`
	ProjectID    int                    `json:"projectId,omitempty"`
	Amount       float64                `json:"amount,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
}

// ApproverResolver interface for different approver resolution strategies
type ApproverResolver interface {
	// GetType returns the resolver type identifier
	GetType() string

	// Resolve returns approvers based on context
	Resolve(ctx context.Context, client *ent.Client, appCtx *ApproverContext) ([]*ApproverInfo, error)
}

// ResolverRegistry manages approver resolvers
type ResolverRegistry struct {
	resolvers map[string]ApproverResolver
	logger    *zap.SugaredLogger
}

// NewResolverRegistry creates a new resolver registry
func NewResolverRegistry(logger *zap.SugaredLogger) *ResolverRegistry {
	return &ResolverRegistry{
		resolvers: make(map[string]ApproverResolver),
		logger:    logger,
	}
}

// Register registers a resolver
func (r *ResolverRegistry) Register(resolver ApproverResolver) {
	r.resolvers[resolver.GetType()] = resolver
	r.logger.Infow("Registered approver resolver", "type", resolver.GetType())
}

// Resolve resolves approvers using the specified resolver type
func (r *ResolverRegistry) Resolve(ctx context.Context, client *ent.Client, resolverType string, appCtx *ApproverContext) ([]*ApproverInfo, error) {
	resolver, ok := r.resolvers[resolverType]
	if !ok {
		return nil, fmt.Errorf("unknown approver resolver type: %s", resolverType)
	}

	approvers, err := resolver.Resolve(ctx, client, appCtx)
	if err != nil {
		r.logger.Warnw(
			"Failed to resolve approvers",
			"type", resolverType,
			"error", err,
			"department_id", appCtx.DepartmentID,
			"team_id", appCtx.TeamID,
		)
		return nil, err
	}

	r.logger.Infow(
		"Resolved approvers",
		"type", resolverType,
		"count", len(approvers),
	)

	return approvers, nil
}

// GetAllTypes returns all registered resolver types
func (r *ResolverRegistry) GetAllTypes() []string {
	types := make([]string, 0, len(r.resolvers))
	for t := range r.resolvers {
		types = append(types, t)
	}
	return types
}
