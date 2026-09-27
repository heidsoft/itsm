package workbench

import "context"

// Repository defines the data access interface for workbench queries.
type Repository interface {
	// Query returns paginated workbench items across multiple ITIL domains.
	Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error)
}
