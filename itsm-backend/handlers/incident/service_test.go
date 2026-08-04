package incident

import "testing"

func TestInferIncidentPriority(t *testing.T) {
	tests := []struct {
		name, title, description, want string
	}{
		{"urgent title", "Production outage", "", "urgent"},
		{"urgent description", "Website unavailable", "The primary service is DOWN", "urgent"},
		{"medium", "Login issue", "Users see an error", "medium"},
		{"low fallback", "Password reset", "User needs access", "low"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferIncidentPriority(tt.title, tt.description); got != tt.want {
				t.Fatalf("inferIncidentPriority() = %q, want %q", got, tt.want)
			}
		})
	}
}
