package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncidentListResponseUsesCamelCaseJSON(t *testing.T) {
	resp := IncidentListResponse{
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
	// 列表集合只用 items，不再返回领域名别名 incidents。
	assert.Contains(t, jsonStr, `"items"`)
	assert.NotContains(t, jsonStr, `"incidents"`)
	assert.NotContains(t, jsonStr, `"page_size"`)
	assert.NotContains(t, jsonStr, `"total_pages"`)
}
