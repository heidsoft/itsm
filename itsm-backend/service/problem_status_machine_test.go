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
		{"same_open", "open", "open", true},

		// Valid forward transitions
		{"open_to_investigating", "open", "investigating", true},
		{"open_to_identified", "open", "identified", true},
		{"investigating_to_identified", "investigating", "identified", true},
		{"investigating_to_resolved", "investigating", "resolved", true},
		{"identified_to_resolved", "identified", "resolved", true},
		{"resolved_to_closed", "resolved", "closed", true},

		// Valid: reopen
		{"resolved_to_investigating", "resolved", "investigating", true},
		{"identified_to_investigating", "identified", "investigating", true},

		// INVALID: cannot leave terminal closed
		{"closed_to_open", "closed", "open", false},
		{"closed_to_investigating", "closed", "investigating", false},
		{"closed_to_resolved", "closed", "resolved", false},
		{"closed_to_identified", "closed", "identified", false},

		// Unknown states fail closed.
		{"unknown_legacy", "legacy", "resolved", false},
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
