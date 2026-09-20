package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncidentListResponseUsesCamelCaseJSON(t *testing.T) {
	resp := IncidentListResponse{
		Incidents:  []*IncidentResponse{{ID: 1, Title: "CPU alert"}},
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
