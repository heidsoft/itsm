package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMSPAllocationDTOUsesCamelCaseJSON(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	resp := MSPAllocationDTO{
		ID:                 1,
		MSPUserID:          2,
		MSPUsername:        "Alice",
		CustomerTenantID:   3,
		CustomerTenantName: "Customer A",
		Role:               "primary",
		AssignedAt:         now,
		DeassignedAt:       &now,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"mspUserId":2`)
	assert.Contains(t, jsonStr, `"mspUsername":"Alice"`)
	assert.Contains(t, jsonStr, `"customerTenantId":3`)
	assert.Contains(t, jsonStr, `"customerTenantName":"Customer A"`)
	assert.Contains(t, jsonStr, `"assignedAt"`)
	assert.Contains(t, jsonStr, `"deassignedAt"`)
	assert.NotContains(t, jsonStr, `"msp_user_id"`)
	assert.NotContains(t, jsonStr, `"customer_tenant_id"`)
	assert.NotContains(t, jsonStr, `"assigned_at"`)
	assert.NotContains(t, jsonStr, `"deassigned_at"`)
}

func TestMSPStatusResponseUsesCamelCaseJSON(t *testing.T) {
	resp := MSPStatusResponse{
		IsMSP:     true,
		MSPUserID: 9,
		Role:      "msp_manager",
		IsAdmin:   true,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"isMsp":true`)
	assert.Contains(t, jsonStr, `"mspUserId":9`)
	assert.Contains(t, jsonStr, `"isAdmin":true`)
	assert.NotContains(t, jsonStr, `"is_msp"`)
	assert.NotContains(t, jsonStr, `"msp_user_id"`)
	assert.NotContains(t, jsonStr, `"is_admin"`)
}
