package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTicketTypeDefinitionUsesCamelCaseJSON(t *testing.T) {
	resp := TicketTypeDefinition{
		ID:                 1,
		Code:               "incident",
		Name:               "Incident",
		CustomFields:       []CustomFieldDefinition{{ID: "cf_1", DefaultValue: "p1"}},
		ApprovalEnabled:    true,
		ApprovalWorkflowID: testStringPtr("wf-1"),
		SLAEnabled:         true,
		DefaultSLAID:       intPtr(9),
		AutoAssignEnabled:  true,
		NotificationConfig: &NotificationConfig{},
		PermissionConfig:   &PermissionConfig{},
		CreatedBy:          11,
		CreatedByName:      "Admin",
		UpdatedBy:          intPtr(12),
		UpdatedByName:      testStringPtr("Ops"),
		CreatedAt:          time.Unix(0, 0).UTC(),
		UpdatedAt:          time.Unix(0, 0).UTC(),
		TenantID:           21,
		UsageCount:         3,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"customFields"`)
	assert.Contains(t, jsonStr, `"defaultValue"`)
	assert.Contains(t, jsonStr, `"approvalEnabled":true`)
	assert.Contains(t, jsonStr, `"approvalWorkflowId":"wf-1"`)
	assert.Contains(t, jsonStr, `"slaEnabled":true`)
	assert.Contains(t, jsonStr, `"defaultSlaId":9`)
	assert.Contains(t, jsonStr, `"autoAssignEnabled":true`)
	assert.Contains(t, jsonStr, `"notificationConfig"`)
	assert.Contains(t, jsonStr, `"permissionConfig"`)
	assert.Contains(t, jsonStr, `"createdBy":11`)
	assert.Contains(t, jsonStr, `"createdByName":"Admin"`)
	assert.Contains(t, jsonStr, `"updatedBy":12`)
	assert.Contains(t, jsonStr, `"updatedByName":"Ops"`)
	assert.Contains(t, jsonStr, `"tenantId":21`)
	assert.Contains(t, jsonStr, `"usageCount":3`)
	assert.NotContains(t, jsonStr, `"custom_fields"`)
	assert.NotContains(t, jsonStr, `"approval_enabled"`)
	assert.NotContains(t, jsonStr, `"approval_workflow_id"`)
	assert.NotContains(t, jsonStr, `"default_sla_id"`)
	assert.NotContains(t, jsonStr, `"auto_assign_enabled"`)
	assert.NotContains(t, jsonStr, `"notification_config"`)
	assert.NotContains(t, jsonStr, `"permission_config"`)
	assert.NotContains(t, jsonStr, `"created_by"`)
	assert.NotContains(t, jsonStr, `"created_by_name"`)
	assert.NotContains(t, jsonStr, `"updated_by"`)
	assert.NotContains(t, jsonStr, `"updated_by_name"`)
	assert.NotContains(t, jsonStr, `"tenant_id"`)
	assert.NotContains(t, jsonStr, `"usage_count"`)
}

func TestTicketTypeListResponseUsesCamelCaseJSON(t *testing.T) {
	resp := TicketTypeListResponse{
		Types:      []TicketTypeDefinition{{ID: 1, Code: "incident", Name: "Incident"}},
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

func intPtr(v int) *int {
	return &v
}

func testStringPtr(v string) *string {
	return &v
}
