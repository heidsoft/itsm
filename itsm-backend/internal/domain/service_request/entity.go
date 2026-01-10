package service_request

import (
	"context"
	"time"
)

// ServiceRequest represents the domain entity for a service request
type ServiceRequest struct {
	ID                 int
	TenantID           int
	CatalogID          int
	RequesterID        int
	CIID               int
	Status             string
	Title              string
	Reason             string
	FormData           map[string]interface{}
	CostCenter         string
	DataClassification string
	NeedsPublicIP      bool
	SourceIPWhitelist  []string
	ExpireAt           *time.Time
	ComplianceAck      bool
	CurrentLevel       int
	TotalLevels        int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ServiceRequestApproval represents an approval step for a request
type ServiceRequestApproval struct {
	ID               int
	TenantID         int
	ServiceRequestID int
	Level            int
	Step             string
	Status           string
	ApproverID       *int
	ApproverName     string
	Comment          string
	Action           string
	TimeoutHours     int
	DueAt            *time.Time
	ProcessedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ListFilters defines filters for listing service requests
type ListFilters struct {
	Status string
	UserID int // Requester ID
	Page   int
	Size   int
}

// Repository defines the interface for data persistence
type Repository interface {
	Create(ctx context.Context, req *ServiceRequest, approvals []*ServiceRequestApproval) (*ServiceRequest, error)
	Get(ctx context.Context, id int) (*ServiceRequest, error)
	GetWithApprovals(ctx context.Context, id int) (*ServiceRequest, []*ServiceRequestApproval, error)
	List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceRequest, int, error)
	UpdateStatus(ctx context.Context, id int, status string) error

	// Approval related
	GetApproval(ctx context.Context, requestID int, level int) (*ServiceRequestApproval, error)
	UpdateApproval(ctx context.Context, approval *ServiceRequestApproval) error
	UpdateRequestAndApproval(ctx context.Context, req *ServiceRequest, approval *ServiceRequestApproval) error

	// Pending approvals for approver
	ListPendingApprovals(ctx context.Context, tenantID int, targetLevel int, requiredStatus string, page, size int) ([]*ServiceRequest, int, error)
}
