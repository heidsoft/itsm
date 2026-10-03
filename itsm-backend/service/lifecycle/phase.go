// Package lifecycle provides unified phase definitions and status mapping
// for ITIL work items (ticket, incident, problem, change, release, service_request).
//
// Phase is a cross-domain abstraction that groups domain-specific statuses
// into a common lifecycle model. This enables unified queries, dashboards,
// and workbench views across different ITIL domains.
package lifecycle

import "sort"

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

// StatusesForPhases returns the statuses of a domain that map into any of the
// given phases, so cross-domain phase filters can be pushed down into SQL
// instead of filtering an already-paginated page.
//
// Empty phases means "no phase restriction" and returns (nil, true).
// An unknown domain returns ok=false so callers skip that domain rather than
// silently treating the filter as unrestricted.
// Results are sorted for deterministic SQL.
func StatusesForPhases(domain Domain, phases []string) ([]string, bool) {
	domainMap, ok := statusPhaseMap[domain]
	if !ok {
		return nil, false
	}
	if len(phases) == 0 {
		return nil, true
	}

	wanted := make(map[Phase]struct{}, len(phases))
	for _, p := range phases {
		wanted[Phase(p)] = struct{}{}
	}

	statuses := make([]string, 0, len(domainMap))
	for status, phase := range domainMap {
		if _, hit := wanted[phase]; hit {
			statuses = append(statuses, status)
		}
	}
	sort.Strings(statuses)
	return statuses, true
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
