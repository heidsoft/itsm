package dto

import (
	"time"

	"itsm-backend/ent"
)

type DomainConfigResponse struct {
	ID           int                    `json:"id"`
	ConfigKey    string                 `json:"configKey"`
	ConfigType   string                 `json:"configType"`
	ConfigValue  map[string]interface{} `json:"configValue"`
	InheritMode  string                 `json:"inheritMode"`
	TenantID     int                    `json:"tenantId"`
	DepartmentID int                    `json:"departmentId"`
	TeamID       int                    `json:"teamId"`
	Version      int                    `json:"version"`
	IsActive     bool                   `json:"isActive"`
	Description  string                 `json:"description,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
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
