package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoleResponseUsesCamelCaseJSON(t *testing.T) {
	resp := RoleResponse{
		ID:          1,
		Name:        "Admin",
		Code:        "admin",
		Description: "Administrator",
		IsSystem:    true,
		TenantID:    2,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"isSystem":true`)
	assert.Contains(t, jsonStr, `"tenantId":2`)
	assert.Contains(t, jsonStr, `"createdAt"`)
	assert.Contains(t, jsonStr, `"updatedAt"`)
	assert.NotContains(t, jsonStr, `"is_system"`)
	assert.NotContains(t, jsonStr, `"tenant_id"`)
	assert.NotContains(t, jsonStr, `"created_at"`)
	assert.NotContains(t, jsonStr, `"updated_at"`)
}

func TestMenuDTOUsesCamelCaseJSON(t *testing.T) {
	parentID := 10
	resp := MenuDTO{
		ID:             1,
		Name:           "Tickets",
		Path:           "/tickets",
		ParentID:       &parentID,
		SortOrder:      5,
		TenantID:       2,
		IsVisible:      true,
		IsEnabled:      true,
		PermissionCode: stringPtrForDTO("tickets:view"),
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"parentId":10`)
	assert.Contains(t, jsonStr, `"permissionCode":"tickets:view"`)
	assert.Contains(t, jsonStr, `"sortOrder":5`)
	assert.Contains(t, jsonStr, `"tenantId":2`)
	assert.Contains(t, jsonStr, `"isVisible":true`)
	assert.Contains(t, jsonStr, `"isEnabled":true`)
	assert.NotContains(t, jsonStr, `"parent_id"`)
	assert.NotContains(t, jsonStr, `"permission_code"`)
	assert.NotContains(t, jsonStr, `"sort_order"`)
	assert.NotContains(t, jsonStr, `"tenant_id"`)
	assert.NotContains(t, jsonStr, `"is_visible"`)
	assert.NotContains(t, jsonStr, `"is_enabled"`)
}

func TestTenantListResponseUsesPageSizeJSON(t *testing.T) {
	resp := TenantListResponse{
		Tenants: []TenantResponse{{ID: 1, Name: "HQ", Code: "hq"}},
		Total:   1,
		Page:    2,
		PageSize: 20,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pageSize":20`)
	assert.NotContains(t, jsonStr, `"size"`)
}

func TestKnowledgeArticleListResponseUsesPageSizeJSON(t *testing.T) {
	resp := KnowledgeArticleListResponse{
		Articles: []KnowledgeArticleResponse{{ID: 1, Title: "KB"}},
		Total:    1,
		Page:     1,
		PageSize: 10,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"pageSize":10`)
	assert.NotContains(t, jsonStr, `"size"`)
}

func stringPtrForDTO(v string) *string {
	return &v
}
