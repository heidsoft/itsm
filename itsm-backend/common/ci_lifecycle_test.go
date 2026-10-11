package common

import (
	"errors"
	"testing"
)

func TestCanTransitionCILifecycle_Matrix(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{CILifecycleStatusDraft, CILifecycleStatusOnline, true},
		{CILifecycleStatusDraft, CILifecycleStatusMaintenance, false},
		{CILifecycleStatusDraft, CILifecycleStatusOffline, false},
		{CILifecycleStatusDraft, CILifecycleStatusScrapped, false},
		{CILifecycleStatusOnline, CILifecycleStatusMaintenance, true},
		{CILifecycleStatusOnline, CILifecycleStatusOffline, true},
		{CILifecycleStatusOnline, CILifecycleStatusScrapped, true},
		{CILifecycleStatusOnline, CILifecycleStatusDraft, false},
		{CILifecycleStatusMaintenance, CILifecycleStatusOnline, true},
		{CILifecycleStatusMaintenance, CILifecycleStatusOffline, true},
		{CILifecycleStatusMaintenance, CILifecycleStatusScrapped, true},
		{CILifecycleStatusMaintenance, CILifecycleStatusDraft, false},
		{CILifecycleStatusOffline, CILifecycleStatusOnline, true},
		{CILifecycleStatusOffline, CILifecycleStatusMaintenance, true},
		{CILifecycleStatusOffline, CILifecycleStatusScrapped, true},
		{CILifecycleStatusOffline, CILifecycleStatusDraft, false},
		{CILifecycleStatusScrapped, CILifecycleStatusOnline, false},
		{CILifecycleStatusScrapped, CILifecycleStatusDraft, false},
		{"ghost", CILifecycleStatusOnline, false},
		{CILifecycleStatusOnline, "ghost", false},
	}
	for _, tc := range cases {
		if got := CanTransitionCILifecycle(tc.from, tc.to); got != tc.want {
			t.Errorf("CanTransitionCILifecycle(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestIsValidCILifecycleStatus(t *testing.T) {
	for _, s := range []string{CILifecycleStatusDraft, CILifecycleStatusOnline, CILifecycleStatusMaintenance, CILifecycleStatusOffline, CILifecycleStatusScrapped} {
		if !IsValidCILifecycleStatus(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range []string{"retired", "retiring", "", "ghost"} {
		if IsValidCILifecycleStatus(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestCanRetireCILifecycle(t *testing.T) {
	for _, from := range []string{CILifecycleStatusOnline, CILifecycleStatusMaintenance, CILifecycleStatusOffline} {
		for _, reason := range []string{"manual", "decommissioned", "replaced", "discovery_missing", "end_of_life"} {
			if err := CanRetireCILifecycle(from, reason); err != nil {
				t.Fatalf("%s + %s should retire, got %v", from, reason, err)
			}
		}
	}
	if err := CanRetireCILifecycle(CILifecycleStatusOnline, "free_text"); !errors.Is(err, ErrUnknownCIRetirementReason) {
		t.Fatalf("expected ErrUnknownCIRetirementReason, got %v", err)
	}
	if err := CanRetireCILifecycle(CILifecycleStatusOnline, ""); !errors.Is(err, ErrUnknownCIRetirementReason) {
		t.Fatalf("empty reason must reject, got %v", err)
	}
	if err := CanRetireCILifecycle(CILifecycleStatusDraft, "manual"); !errors.Is(err, ErrInvalidCILifecycleTransition) {
		t.Fatalf("draft must not retire directly, got %v", err)
	}
	if err := CanRetireCILifecycle(CILifecycleStatusScrapped, "manual"); !errors.Is(err, ErrInvalidCILifecycleTransition) {
		t.Fatalf("scrapped is terminal, got %v", err)
	}
}
