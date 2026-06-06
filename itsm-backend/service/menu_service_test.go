package service

import (
	"testing"

	"itsm-backend/ent"
)

func TestShouldRestrictMenuForRole(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		roleCodes map[string]bool
		want      bool
	}{
		{
			name:      "end user cannot see workflow menu",
			path:      "/workflow/dashboard",
			roleCodes: map[string]bool{"end_user": true},
			want:      true,
		},
		{
			name:      "security cannot see admin menu",
			path:      "/admin/users",
			roleCodes: map[string]bool{"security": true},
			want:      true,
		},
		{
			name:      "admin can see workflow menu",
			path:      "/workflow/dashboard",
			roleCodes: map[string]bool{"admin": true},
			want:      false,
		},
		{
			name:      "main menu stays visible for end user",
			path:      "/incidents",
			roleCodes: map[string]bool{"end_user": true},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRestrictMenuForRole(tt.path, tt.roleCodes)
			if got != tt.want {
				t.Fatalf("shouldRestrictMenuForRole(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFilterMenusByPermissionRestrictsLowPrivilegeAdminMenus(t *testing.T) {
	svc := &MenuService{}
	menus := []*ent.Menu{
		{Name: "事件管理", Path: "/incidents", PermissionCode: "incident:read"},
		{Name: "工作流", Path: "/workflow", PermissionCode: "workflow:read"},
		{Name: "用户管理", Path: "/admin/users", PermissionCode: "user:read"},
	}

	filtered := svc.filterMenusByPermission(
		menus,
		map[string]bool{
			"incident:read": true,
			"workflow:read": true,
			"user:read":     true,
		},
		map[string]bool{"end_user": true},
	)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 visible menu, got %d", len(filtered))
	}

	if filtered[0].Path != "/incidents" {
		t.Fatalf("expected incidents menu to remain visible, got %s", filtered[0].Path)
	}
}
