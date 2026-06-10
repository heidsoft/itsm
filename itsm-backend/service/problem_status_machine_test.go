package service

import "testing"

func TestIsValidProblemStatusTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		want    bool
	}{
		// Valid: same status
		{"same_new", "new", "new", true},

		// Valid forward transitions
		{"new_to_investigating", "new", "investigating", true},
		{"new_to_identified", "new", "identified", true},
		{"investigating_to_identified", "investigating", "identified", true},
		{"investigating_to_resolved", "investigating", "resolved", true},
		{"identified_to_resolved", "identified", "resolved", true},
		{"resolved_to_closed", "resolved", "closed", true},

		// Valid: reopen
		{"resolved_to_investigating", "resolved", "investigating", true},
		{"identified_to_investigating", "identified", "investigating", true},

		// INVALID: cannot leave terminal closed
		{"closed_to_new", "closed", "new", false},
		{"closed_to_investigating", "closed", "investigating", false},
		{"closed_to_resolved", "closed", "resolved", false},
		{"closed_to_identified", "closed", "identified", false},

		// Unknown: permissive for legacy data
		{"unknown_legacy", "legacy", "resolved", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidProblemStatusTransition(tc.current, tc.next)
			if got != tc.want {
				t.Errorf("isValidProblemStatusTransition(%q, %q) = %v, want %v",
					tc.current, tc.next, got, tc.want)
			}
		})
	}
}
