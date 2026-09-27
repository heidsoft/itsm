package lifecycle

import "testing"

func TestStatusToPhase(t *testing.T) {
	tests := []struct {
		name     string
		domain   Domain
		status   string
		expected Phase
	}{
		// Ticket domain
		{"ticket open → submitted", DomainTicket, "open", PhaseSubmitted},
		{"ticket in_progress → active", DomainTicket, "in_progress", PhaseActive},
		{"ticket resolved → resolved", DomainTicket, "resolved", PhaseResolved},
		{"ticket closed → closed", DomainTicket, "closed", PhaseClosed},

		// Incident domain
		{"incident new → submitted", DomainIncident, "new", PhaseSubmitted},
		{"incident in_progress → active", DomainIncident, "in_progress", PhaseActive},
		{"incident resolved → resolved", DomainIncident, "resolved", PhaseResolved},
		{"incident closed → closed", DomainIncident, "closed", PhaseClosed},

		// Problem domain
		{"problem open → submitted", DomainProblem, "open", PhaseSubmitted},
		{"problem in_progress → active", DomainProblem, "in_progress", PhaseActive},
		{"problem resolved → resolved", DomainProblem, "resolved", PhaseResolved},
		{"problem closed → closed", DomainProblem, "closed", PhaseClosed},

		// Change domain
		{"change draft → draft", DomainChange, "draft", PhaseDraft},
		{"change approved → active", DomainChange, "approved", PhaseActive},
		{"change completed → resolved", DomainChange, "completed", PhaseResolved},
		{"change closed → closed", DomainChange, "closed", PhaseClosed},

		// Release domain
		{"release draft → draft", DomainRelease, "draft", PhaseDraft},
		{"release in_progress → active", DomainRelease, "in_progress", PhaseActive},
		{"release completed → resolved", DomainRelease, "completed", PhaseResolved},
		{"release cancelled → closed", DomainRelease, "cancelled", PhaseClosed},

		// ServiceRequest domain
		{"servicerequest submitted → submitted", DomainServiceRequest, "submitted", PhaseSubmitted},
		{"servicerequest approved → active", DomainServiceRequest, "approved", PhaseActive},
		{"servicerequest completed → closed", DomainServiceRequest, "completed", PhaseClosed},
		{"servicerequest rejected → closed", DomainServiceRequest, "rejected", PhaseClosed},

		// Unknown cases
		{"unknown domain", Domain("unknown"), "open", PhaseUnknown},
		{"unknown status", DomainTicket, "unknown_status", PhaseUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusToPhase(tt.domain, tt.status)
			if got != tt.expected {
				t.Errorf("StatusToPhase(%q, %q) = %q, want %q", tt.domain, tt.status, got, tt.expected)
			}
		})
	}
}

func TestIsValidPhase(t *testing.T) {
	validPhases := []Phase{PhaseDraft, PhaseSubmitted, PhaseActive, PhaseResolved, PhaseClosed}
	for _, p := range validPhases {
		if !IsValidPhase(p) {
			t.Errorf("IsValidPhase(%q) = false, want true", p)
		}
	}

	if IsValidPhase(PhaseUnknown) {
		t.Error("IsValidPhase(PhaseUnknown) = true, want false")
	}
	if IsValidPhase(Phase("invalid")) {
		t.Error("IsValidPhase('invalid') = true, want false")
	}
}

func TestIsValidDomain(t *testing.T) {
	validDomains := []Domain{DomainTicket, DomainIncident, DomainProblem, DomainChange, DomainRelease, DomainServiceRequest}
	for _, d := range validDomains {
		if !IsValidDomain(d) {
			t.Errorf("IsValidDomain(%q) = false, want true", d)
		}
	}

	if IsValidDomain(Domain("unknown")) {
		t.Error("IsValidDomain('unknown') = true, want false")
	}
}

func TestAllPhases(t *testing.T) {
	phases := AllPhases()
	if len(phases) != 5 {
		t.Errorf("AllPhases() returned %d phases, want 5", len(phases))
	}

	expected := []Phase{PhaseDraft, PhaseSubmitted, PhaseActive, PhaseResolved, PhaseClosed}
	for i, p := range expected {
		if phases[i] != p {
			t.Errorf("AllPhases()[%d] = %q, want %q", i, phases[i], p)
		}
	}
}

func TestAllDomains(t *testing.T) {
	domains := AllDomains()
	if len(domains) != 6 {
		t.Errorf("AllDomains() returned %d domains, want 6", len(domains))
	}
}
