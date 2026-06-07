package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateTicketViewRequestNormalizeUsesLegacyFields(t *testing.T) {
	isShared := true
	req := CreateTicketViewRequest{
		SortConfigLegacy:  map[string]interface{}{"field": "priority"},
		GroupConfigLegacy: map[string]interface{}{"field": "status"},
		IsSharedLegacy:    &isShared,
	}

	req.Normalize()

	assert.Equal(t, map[string]interface{}{"field": "priority"}, req.SortConfig)
	assert.Equal(t, map[string]interface{}{"field": "status"}, req.GroupConfig)
	assert.True(t, req.IsShared)
}

func TestShareTicketViewRequestNormalizeUsesLegacyTeamIDs(t *testing.T) {
	req := ShareTicketViewRequest{
		TeamIDsLegacy: []int{1, 2, 3},
	}

	req.Normalize()

	assert.Equal(t, []int{1, 2, 3}, req.TeamIDs)
}

func TestTicketViewResponseUsesCamelCaseJSON(t *testing.T) {
	resp := TicketViewResponse{
		ID:          1,
		Name:        "My View",
		SortConfig:  map[string]interface{}{"field": "priority"},
		GroupConfig: map[string]interface{}{"field": "status"},
		IsShared:    true,
		CreatedBy:   10,
		TenantID:    20,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"sortConfig"`)
	assert.Contains(t, jsonStr, `"groupConfig"`)
	assert.Contains(t, jsonStr, `"isShared":true`)
	assert.Contains(t, jsonStr, `"createdBy":10`)
	assert.Contains(t, jsonStr, `"tenantId":20`)
	assert.Contains(t, jsonStr, `"createdAt"`)
	assert.Contains(t, jsonStr, `"updatedAt"`)
	assert.NotContains(t, jsonStr, `"sort_config"`)
	assert.NotContains(t, jsonStr, `"group_config"`)
	assert.NotContains(t, jsonStr, `"is_shared"`)
	assert.NotContains(t, jsonStr, `"created_by"`)
	assert.NotContains(t, jsonStr, `"tenant_id"`)
}
