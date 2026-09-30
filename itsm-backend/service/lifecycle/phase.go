// Package lifecycle provides unified phase definitions and status mapping
// for ITIL work items (ticket, incident, problem, change, release, service_request).
//
// Phase is a cross-domain abstraction that groups domain-specific statuses
// into a common lifecycle model. This enables unified queries, dashboards,
// and workbench views across different ITIL domains.
package lifecycle

// Phase represents a stage in the work item lifecycle.
// Unlike domain-specific status values, Phase provides a consistent
// vocabulary across ticket, incident, problem, change, release, and service_request.
type Phase string

const (
	// PhaseDraft means the work item is being prepared and has not been submitted.
	PhaseDraft Phase = "draft"

	// PhaseSubmitted means the work item has been submitted and is waiting for review/assignment.
	PhaseSubmitted Phase = "submitted"

	// PhaseActive means the work item is being actively worked on.
	PhaseActive Phase = "active"

	// PhaseResolved means the work item has been resolved and is pending closure/verification.
	PhaseResolved Phase = "resolved"

	// PhaseClosed means the work item has been closed, cancelled, or rejected.
	PhaseClosed Phase = "closed"

	// PhaseUnknown is returned when the status cannot be mapped to a known phase.
	PhaseUnknown Phase = "unknown"
)

// Domain represents an ITIL work item domain.
type Domain string

const (
	DomainTicket         Domain = "ticket"
	DomainIncident       Domain = "incident"
	DomainProblem        Domain = "problem"
	DomainChange         Domain = "change"
	DomainRelease        Domain = "release"
	DomainServiceRequest Domain = "service_request"
)

// statusPhaseMap maps (domain, status) pairs to phases.
// This is the single source of truth for status→phase translation.
var statusPhaseMap = map[Domain]map[string]Phase{
	DomainTicket: {
		"open":        PhaseSubmitted,
		"assigned":    PhaseActive,
		"in_progress": PhaseActive,
		"resolved":    PhaseResolved,
		"closed":      PhaseClosed,
		"cancelled":   PhaseClosed,
	},
	DomainIncident: {
		"new":         PhaseSubmitted,
		"open":        PhaseSubmitted,
		"assigned":    PhaseActive,
		"in_progress": PhaseActive,
		"resolved":    PhaseResolved,
		"closed":      PhaseClosed,
		"cancelled":   PhaseClosed,
	},
	DomainProblem: {
		"open":        PhaseSubmitted,
		"assigned":    PhaseActive,
		"in_progress": PhaseActive,
		"resolved":    PhaseResolved,
		"closed":      PhaseClosed,
		"cancelled":   PhaseClosed,
	},
	DomainChange: {
		"draft":       PhaseDraft,
		"submitted":   PhaseSubmitted,
		"approved":    PhaseActive,
		"in_progress": PhaseActive,
		"completed":   PhaseResolved,
		"closed":      PhaseClosed,
		"cancelled":   PhaseClosed,
		"rejected":    PhaseClosed,
	},
	DomainRelease: {
		"draft":       PhaseDraft,
		"scheduled":   PhaseSubmitted,
		"in_progress": PhaseActive,
		"completed":   PhaseResolved,
		"cancelled":   PhaseClosed,
	},
	DomainServiceRequest: {
		"submitted":   PhaseSubmitted,
		"approved":    PhaseActive,
		"rejected":    PhaseClosed,
		"in_progress": PhaseActive,
		"completed":   PhaseClosed,
		"cancelled":   PhaseClosed,
	},
}

// StatusToPhase translates a domain-specific status to a lifecycle phase.
// Returns PhaseUnknown if the domain or status is not recognized.
//
// This function is the canonical mapping used by all DTO mappers and service layers.
// Do not duplicate this logic in handlers or frontend code.
func StatusToPhase(domain Domain, status string) Phase {
	domainMap, ok := statusPhaseMap[domain]
	if !ok {
		return PhaseUnknown
	}
	phase, ok := domainMap[status]
	if !ok {
		return PhaseUnknown
	}
	return phase
}

// IsValidPhase checks if the given string is a valid Phase value.
func IsValidPhase(phase Phase) bool {
	switch phase {
	case PhaseDraft, PhaseSubmitted, PhaseActive, PhaseResolved, PhaseClosed:
		return true
	default:
		return false
	}
}

// IsValidDomain checks if the given string is a valid Domain value.
func IsValidDomain(domain Domain) bool {
	switch domain {
	case DomainTicket, DomainIncident, DomainProblem, DomainChange, DomainRelease, DomainServiceRequest:
		return true
	default:
		return false
	}
}

// AllPhases returns all valid phase values in lifecycle order.
func AllPhases() []Phase {
	return []Phase{
		PhaseDraft,
		PhaseSubmitted,
		PhaseActive,
		PhaseResolved,
		PhaseClosed,
	}
}

// AllDomains returns all valid domain values.
func AllDomains() []Domain {
	return []Domain{
		DomainTicket,
		DomainIncident,
		DomainProblem,
		DomainChange,
		DomainRelease,
		DomainServiceRequest,
	}
}
