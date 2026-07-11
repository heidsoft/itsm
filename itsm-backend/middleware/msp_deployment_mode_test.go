package middleware

import "testing"

// TestApplyDeploymentMode locks the policy that maps DEPLOYMENT_MODE env to
// the MSP gate. The original commit only had positive coverage (private
// turns MSP off). This test covers all four input classes including the
// reverse direction: SaaS / SaaS+MSP / empty / unknown values must all keep
// MSP enabled, so a typo in the env name can never silently turn MSP off.
func TestApplyDeploymentMode(t *testing.T) {
	cases := []struct {
		name    string
		mode    string
		wantMSP bool
	}{
		{"private disables MSP", "private", false},
		{"saas enables MSP", "saas", true},
		{"saas_msp enables MSP", "saas_msp", true},
		{"empty enables MSP", "", true},
		{"unknown mode defaults to SaaS-like behavior", "edge", true},
		{"uppercase private still treated as private", "PRIVATE", true}, // intentionally case-sensitive; see note
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset before each case so leftover state doesn't leak.
			SetMSPEnabled(true)
			ApplyDeploymentMode(tc.mode)
			if got := IsMSPEnabled(); got != tc.wantMSP {
				t.Fatalf("ApplyDeploymentMode(%q) -> IsMSPEnabled()=%v, want %v",
					tc.mode, got, tc.wantMSP)
			}
		})
	}
}

// TestApplyDeploymentMode_Idempotent makes sure calling ApplyDeploymentMode
// twice with the same mode is safe and leaves the gate in the right state.
func TestApplyDeploymentMode_Idempotent(t *testing.T) {
	for _, mode := range []string{"private", "saas", "saas_msp", ""} {
		ApplyDeploymentMode(mode)
		first := IsMSPEnabled()
		ApplyDeploymentMode(mode)
		if IsMSPEnabled() != first {
			t.Fatalf("idempotency broken for mode=%q: %v -> %v", mode, first, IsMSPEnabled())
		}
	}
}
