package cmdb

import (
	"time"
)

// ConfigurationItem representing a CI in the CMDB
type ConfigurationItem struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"` // Legacy type field from schema
	Status       string                 `json:"status"`
	Location     string                 `json:"location"`
	SerialNumber string                 `json:"serial_number"`
	Model        string                 `json:"model"`
	Vendor       string                 `json:"vendor"`
	CITypeID     int                    `json:"ci_type_id"`
	TenantID     int                    `json:"tenant_id"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CIType represents a classification of CIs
type CIType struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	AttributeSchema string    `json:"attribute_schema"`
	IsActive        bool      `json:"is_active"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CIRelationship represents a link between two CIs
type CIRelationship struct {
	ID                 int       `json:"id"`
	SourceCIID         int       `json:"source_ci_id"`
	TargetCIID         int       `json:"target_ci_id"`
	RelationshipTypeID int       `json:"relationship_type_id"`
	Description        string    `json:"description"`
	TenantID           int       `json:"tenant_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Stats represents CMDB statistics
type Stats struct {
	TotalCount       int            `json:"total_count"`
	ActiveCount      int            `json:"active_count"`
	InactiveCount    int            `json:"inactive_count"`
	MaintenanceCount int            `json:"maintenance_count"`
	TypeDistribution map[string]int `json:"type_distribution"`
}
