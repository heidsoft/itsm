package menubaseline_test

import (
	"testing"

	"itsm-backend/pkg/menubaseline"
)

func TestMenuBaselinePageReadPermissions(t *testing.T) {
	expected := map[string]string{
		"/dashboard":        "dashboard:read",
		"/service-requests": "service_request:read",
		"/my-requests":      "service_request:read",
		"/workflow/audit":   "bpmn:read",
	}
	for _, item := range menubaseline.MenuDefinitions() {
		code, found := expected[item.Path]
		if !found {
			continue
		}
		if item.PermissionCode != code {
			t.Errorf("%s permission = %q; want %q", item.Path, item.PermissionCode, code)
		}
		delete(expected, item.Path)
	}
	for route := range expected {
		t.Errorf("missing baseline menu %q", route)
	}
}
