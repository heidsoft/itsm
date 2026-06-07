package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIncidentListResponseUsesCamelCaseJSON(t *testing.T) {
	resp := IncidentListResponse{
		Incidents:  []*IncidentResponse{{ID: 1, Title: "CPU alert"}},
		Items:      []*IncidentResponse{{ID: 1, Title: "CPU alert"}},
		Total:      1,
		Page:       2,
		PageSize:   20,
		TotalPages: 5,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pageSize":20`)
	assert.Contains(t, jsonStr, `"totalPages":5`)
	assert.NotContains(t, jsonStr, `"page_size"`)
	assert.NotContains(t, jsonStr, `"total_pages"`)
}

func TestWorkflowResponseUsesCamelCaseJSON(t *testing.T) {
	departmentID := 8
	resp := WorkflowResponse{
		ID:           1,
		Name:         "Approval",
		Type:         "approval",
		IsActive:     true,
		TenantID:     2,
		DepartmentID: &departmentID,
		CreatedAt:    time.Unix(0, 0).UTC(),
		UpdatedAt:    time.Unix(0, 0).UTC(),
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"isActive":true`)
	assert.Contains(t, jsonStr, `"tenantId":2`)
	assert.Contains(t, jsonStr, `"departmentId":8`)
	assert.Contains(t, jsonStr, `"createdAt"`)
	assert.Contains(t, jsonStr, `"updatedAt"`)
	assert.NotContains(t, jsonStr, `"is_active"`)
	assert.NotContains(t, jsonStr, `"tenant_id"`)
	assert.NotContains(t, jsonStr, `"department_id"`)
	assert.NotContains(t, jsonStr, `"created_at"`)
	assert.NotContains(t, jsonStr, `"updated_at"`)
}

func TestWorkflowListResponseUsesPageSizeJSON(t *testing.T) {
	resp := WorkflowListResponse{
		Workflows: []*WorkflowResponse{{ID: 1, Name: "Approval"}},
		Total:     1,
		Page:      1,
		PageSize:  10,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pageSize":10`)
	assert.NotContains(t, jsonStr, `"page_size"`)
}
