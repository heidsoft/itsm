package dto

import (
	"time"

	"itsm-backend/ent"
)

type DomainConfigResponse struct {
	ID           int                    `json:"id"`
	ConfigKey    string                 `json:"config_key"`
	ConfigType   string                 `json:"config_type"`
	ConfigValue  map[string]interface{} `json:"config_value"`
	InheritMode  string                 `json:"inherit_mode"`
	TenantID     int                    `json:"tenant_id"`
	DepartmentID int                    `json:"department_id"`
	TeamID       int                    `json:"team_id"`
	Version      int                    `json:"version"`
	IsActive     bool                   `json:"is_active"`
	Description  string                 `json:"description,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func ToDomainConfigResponse(config *ent.DomainConfig) *DomainConfigResponse {
	if config == nil {
		return nil
	}
	return &DomainConfigResponse{
		ID:           config.ID,
		ConfigKey:    config.ConfigKey,
		ConfigType:   config.ConfigType,
		ConfigValue:  config.ConfigValue,
		InheritMode:  config.InheritMode,
		TenantID:     config.TenantID,
		DepartmentID: config.DepartmentID,
		TeamID:       config.TeamID,
		Version:      config.Version,
		IsActive:     config.IsActive,
		Description:  config.Description,
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	}
}

func ToDomainConfigResponseList(configs []*ent.DomainConfig) []*DomainConfigResponse {
	responses := make([]*DomainConfigResponse, 0, len(configs))
	for _, config := range configs {
		responses = append(responses, ToDomainConfigResponse(config))
	}
	return responses
}
