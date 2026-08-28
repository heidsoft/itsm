// Package alert provides AlertSource pluggable integration for ITSM.
// It enables configuration-driven alert ingestion from any external monitoring system
// (Prometheus Alertmanager, Zabbix, CloudWatch, PagerDuty, etc.) without code changes.
package alert

import (
	"context"
	"errors"
	"time"
)

// Severity represents the alert severity level.
type Severity string

const (
	SeverityP1 Severity = "P1" // Disaster / Critical
	SeverityP2 Severity = "P2" // High
	SeverityP3 Severity = "P3" // Medium / Warning
	SeverityP4 Severity = "P4" // Low / Info
)

// StandardAlert is the normalized alert structure all AlertSources must produce.
// It is the canonical representation inside ITSM regardless of the source system.
type StandardAlert struct {
	AlertID       string                 `json:"alert_id"`                 // External unique alert ID
	Source        string                 `json:"source"`                   // Source system name (e.g. "zabbix", "prometheus")
	SourceRaw     string                 `json:"source_raw,omitempty"`     // Raw source type from config
	TenantID      int                    `json:"tenant_id"`
	Name          string                 `json:"name"`                     // Short alert title
	Description   string                 `json:"description"`             // Detailed description
	Severity      Severity               `json:"severity"`                // P1-P4
	Status        string                 `json:"status"`                  // firing / acknowledged / resolved
	Labels        map[string]string      `json:"labels"`                  // Key-value tags
	Annotations   map[string]string      `json:"annotations"`             // Human-readable metadata
	SourceIP      string                 `json:"source_ip,omitempty"`     // Originating host/IP
	Service       string                 `json:"service,omitempty"`        // Affected service/app name
	Tags          []string               `json:"tags"`                    // Flat tag list
	FiredAt       time.Time              `json:"fired_at"`               // When the alert was first triggered
	AcknowledgedAt *time.Time            `json:"acknowledged_at,omitempty"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
	RawPayload    map[string]interface{} `json:"raw_payload,omitempty"`   // Original payload preserved
}

// AlertSource is the contract for any monitoring system that pushes alerts into ITSM.
// Implementations are configuration-driven; the same binary supports any source
// by loading a YAML/JSON config file at startup or via the admin API.
type AlertSource interface {
	// Manifest returns the connector manifest (name, version, capabilities).
	Manifest() AlertSourceManifest

	// ValidatePayload checks whether the incoming raw payload is structurally valid
	// for this source. Returns false if the payload is malformed or missing required fields.
	// A return of false does NOT mean the alert is rejected — only that it cannot be parsed.
	// Malformed payloads should still be logged and tracked in the raw_events table.
	ValidatePayload(rawPayload map[string]interface{}) bool

	// Normalize converts a raw payload from this specific monitoring system
		// into the ITSM-standard StandardAlert structure.
		// tenantID must be > 0; the implementation MUST fail closed if not.
		// Implementations MUST NOT return nil *StandardAlert on success.
		// If Normalize returns an error, the raw payload should still be preserved
		// (e.g. stored in a raw_events table) for later investigation.
		Normalize(ctx context.Context, tenantID int, rawPayload map[string]interface{}) (*StandardAlert, error)

	// Acknowledge marks an alert as acknowledged in the source system (if applicable).
	// If the source does not support acknowledgment, return ErrNotSupported.
	Acknowledge(ctx context.Context, alertID string) error

	// Resolve marks an alert as resolved in the source system (if applicable).
	// If the source does not support resolution, return ErrNotSupported.
	Resolve(ctx context.Context, alertID string) error

	// Close releases resources held by the alert source (connections, goroutines, etc.).
	Close() error
}

// AlertSourceManifest is the self-describing metadata for an AlertSource.
type AlertSourceManifest struct {
	Name                 string   `json:"name"`                  // Unique key, e.g. "prometheus", "zabbix", "webhook-generic"
	Version              string   `json:"version"`              // Semver
	Title                string   `json:"title"`                 // Display name
	Description          string   `json:"description"`          // One-liner
	Provider             string   `json:"provider"`              // e.g. "cncf", "zabbix", "custom"
	Capabilities         []string `json:"capabilities"`         // ["ingest", "acknowledge", "resolve"]
	ConfigSchema         string   `json:"config_schema,omitempty"`
	RequiredPermissions  []string `json:"required_permissions"` // e.g. ["alerts:write"]
	IsOfficial           bool     `json:"is_official"`
	Category             string   `json:"category"` // "monitoring" | "alerting" | "custom"
	Tags                 []string `json:"tags,omitempty"`
}

// ErrNotSupported signals that an AlertSource capability is not implemented.
type ErrNotSupported = interface{ IsErrNotSupported() bool }

type errNotSupported struct{}

func (e errNotSupported) IsErrNotSupported() bool { return true }
func (e errNotSupported) Error() string            { return "alert source: capability not supported" }

// ErrNotSupportedInstance is the singleton error value.
var ErrNotSupportedInstance = errNotSupported{}

// IsNotSupported returns true if err indicates an unsupported capability.
func IsNotSupported(err error) bool {
	if err == nil {
		return false
	}
	var n ErrNotSupported
	return errors.As(err, &n)
}
