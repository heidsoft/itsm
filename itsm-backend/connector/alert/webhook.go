// Package alert provides AlertSource pluggable integration for ITSM.
package alert

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// WebhookAlertSource is a configuration-driven alert source that ingests
// normalized alerts from any HTTP POST webhook endpoint.
// It uses a declarative YAML/JSON config to map any JSON payload structure
// to the ITSM StandardAlert format, without requiring code changes.
type WebhookAlertSource struct {
	config *AlertSourceConfig
	norm   *Normalizer
}

var _ AlertSource = (*WebhookAlertSource)(nil)

// NewWebhookAlertSource creates a new WebhookAlertSource from a YAML config.
func NewWebhookAlertSource(cfg *AlertSourceConfig) *WebhookAlertSource {
	return &WebhookAlertSource{
		config: cfg,
		norm:   NewNormalizer(cfg),
	}
}

// Manifest returns the self-describing manifest for this alert source.
func (w *WebhookAlertSource) Manifest() AlertSourceManifest {
	return AlertSourceManifest{
		Name:                 w.config.Source,
		Version:              "1.0.0",
		Title:                w.config.Title,
		Description:          w.config.Description,
		Provider:             "generic",
		Capabilities:         []string{"ingest"},
		RequiredPermissions:   []string{"alerts:write"},
		IsOfficial:           false,
		Category:             "alerting",
		Tags:                 []string{"webhook", "configurable", w.config.Source},
	}
}

// ValidatePayload performs a basic structural validation of the raw payload.
// Returns true if the payload appears to be a valid JSON object.
// For configuration-driven sources this is a shallow check — Normalize does the real work.
func (w *WebhookAlertSource) ValidatePayload(rawPayload map[string]interface{}) bool {
	if rawPayload == nil {
		return false
	}
	return true
}

// Normalize converts a raw webhook payload into a StandardAlert using the YAML config.
// tenantID must be > 0; returns error if not (fail closed).
func (w *WebhookAlertSource) Normalize(ctx context.Context, tenantID int, rawPayload map[string]interface{}) (*StandardAlert, error) {
	if tenantID <= 0 {
		return nil, errors.New("webhook alert source: tenant_id must be > 0")
	}
	if !w.ValidatePayload(rawPayload) {
		return nil, fmt.Errorf("webhook alert source %q: invalid payload", w.config.Source)
	}
	return w.norm.Normalize(ctx, tenantID, rawPayload)
}

// Acknowledge acknowledges an alert in the source system.
// WebhookAlertSource is pull-only; it does not support outbound callbacks.
func (w *WebhookAlertSource) Acknowledge(ctx context.Context, alertID string) error {
	return ErrNotSupportedInstance
}

// Resolve resolves an alert in the source system.
// WebhookAlertSource is pull-only; it does not support outbound callbacks.
func (w *WebhookAlertSource) Resolve(ctx context.Context, alertID string) error {
	return ErrNotSupportedInstance
}

// Close releases resources. WebhookAlertSource has no persistent connections.
func (w *WebhookAlertSource) Close() error {
	return nil
}

// VerifyWebhookSignature verifies an HMAC-SHA256 webhook signature.
// It compares the provided signature against the expected HMAC of the body
// using constant-time comparison to prevent timing attacks.
//
// Security behavior:
//   - If secret is empty and EnvIsDevelopment is true, verification is skipped (dev mode).
//   - In production (EnvIsDevelopment=false), a missing secret causes immediate rejection.
//   - On mismatch, a generic error is returned — the computed HMAC is never leaked.
func VerifyWebhookSignature(body []byte, signatureHeader, secret, providedSignature string, envIsDevelopment bool) error {
	if secret == "" {
		if envIsDevelopment {
			return nil // dev mode: no secret configured
		}
		return errors.New("webhook: signature secret is not configured")
	}
	if providedSignature == "" {
		return errors.New("webhook: signature header is missing or empty")
	}

	// Strip any "sha256=" prefix (GitHub/GitLab compatibility)
	sig := providedSignature
	if len(sig) > 7 && sig[:7] == "sha256=" {
		sig = sig[7:]
	}

	// Compute expected HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	// Constant-time comparison — no HMAC leakage in error message
	if subtle.ConstantTimeCompare([]byte(sig), []byte(hex.EncodeToString(expectedMAC))) != 1 {
		return errors.New("webhook: signature mismatch")
	}
	return nil
}
