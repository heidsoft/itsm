// Package alert provides AlertSource pluggable integration for ITSM.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	errInvalidMapping  = errors.New("alert config: invalid field mapping")
	errTemplateRender = errors.New("alert config: template render error")
	errMissingRequired = errors.New("alert config: missing required field")
)

// FieldMapping defines how to extract a single field from the raw payload.
// Supports three modes: jmespath (default), template, and literal.
type FieldMapping struct {
	// JMESPath is the JMESPath expression to extract the value.
	// If set, takes precedence over Template and Literal.
	JMESPath string `yaml:"jmespath" json:"jmespath"`
	// Template is a Go text/template to render the value.
	// Executes after JMESPath resolution if both are set.
	Template string `yaml:"template" json:"template"`
	// Literal is a fixed value returned as-is.
	Literal string `yaml:"literal" json:"literal"`
	// Default is the fallback value if extraction yields nothing.
	Default string `yaml:"default" json:"default"`
	// Required if true causes extraction failure to return an error
	// instead of falling back to Default.
	Required bool `yaml:"required" json:"required"`
	// Transform optionally applies post-extraction transforms
	// (e.g. "map{P1:P1,Disaster:P1}" or "lower" or "int").
	Transform string `yaml:"transform" json:"transform"`
}

// SeverityMap maps source-specific severity strings to StandardAlert severities.
type SeverityMap map[string]Severity

// AlertSourceConfig is the declarative YAML configuration for one alert source.
// Example YAML:
//
//	source: prometheus
//	type: webhook
//	priority: 2
//	mapping:
//	  alert_id: "{{ .body.alerts[0].labels.alertname }}"
//
//	  severity:
//	    jmespath: "body.alerts[0].labels.severity"
//	    transform: "map{P1:critical,P2:high,P3:warning,P4:info}"
//
//	  name:
//	    jmespath: "body.alerts[0].labels.alertname"
//	    default: "未命名告警"
//
//	  description:
//	    template: "{{ .body.alerts[0].annotations.description }}"
//
//	  labels:
//	    jmespath: "body.alerts[0].labels"
//	    transform: "flatten"
//
//	cfg:
//	  signature_header: X-Signature
//	  max_payload_bytes: 65536
type AlertSourceConfig struct {
	// Source is the unique identifier for this alert source within a tenant.
	Source string `yaml:"source" json:"source"`
	// Type is always "webhook" for YAML-driven sources.
	Type string `yaml:"type" json:"type"`
	// Priority is the processing priority (lower = higher priority).
	Priority int `yaml:"priority" json:"priority"`
	// Enabled controls whether this source is active.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Title is the human-readable name shown in the UI.
	Title string `yaml:"title" json:"title"`
	// Description is shown in the admin UI.
	Description string `yaml:"description" json:"description"`
	// Mapping defines how raw payload fields map to StandardAlert fields.
	Mapping AlertMapping `yaml:"mapping" json:"mapping"`
	// Cfg holds source-specific runtime settings (signature verification, timeouts, etc.).
	Cfg WebhookCfg `yaml:"cfg" json:"cfg"`
}

// AlertMapping defines field-level extraction rules for one alert source config.
type AlertMapping struct {
	AlertID     FieldMapping `yaml:"alert_id" json:"alert_id"`
	Name        FieldMapping `yaml:"name" json:"name"`
	Description FieldMapping `yaml:"description" json:"description"`
	Severity    FieldMapping `yaml:"severity" json:"severity"`
	Status      FieldMapping `yaml:"status" json:"status"`
	Labels      FieldMapping `yaml:"labels" json:"labels"`
	Annotations FieldMapping `yaml:"annotations" json:"annotations"`
	SourceIP    FieldMapping `yaml:"source_ip" json:"source_ip"`
	Service     FieldMapping `yaml:"service" json:"service"`
	Tags        FieldMapping `yaml:"tags" json:"tags"`
	FiredAt     FieldMapping `yaml:"fired_at" json:"fired_at"`
}

// WebhookCfg holds per-source webhook runtime configuration.
type WebhookCfg struct {
	// SignatureHeader names the HTTP header containing the request signature.
	SignatureHeader string `yaml:"signature_header" json:"signature_header"`
	// SignatureSecret is the secret used to verify the signature (HMAC-SHA256).
	SignatureSecret string `yaml:"signature_secret" json:"signature_secret,omitempty"`
	// SignatureAlgorithm defaults to "hmac-sha256".
	SignatureAlgorithm string `yaml:"signature_algorithm" json:"signature_algorithm"`
	// MaxPayloadBytes limits how large a payload can be (default 1MB).
	MaxPayloadBytes int `yaml:"max_payload_bytes" json:"max_payload_bytes"`
	// Timeout is the HTTP request timeout in seconds (default 10).
	TimeoutSeconds int `yaml:"timeout_seconds" json:"timeout_seconds"`
	// TenantID is set at runtime by the Integration Hub, not from YAML.
	TenantID int `json:"-"`
}

// ConfigStore manages YAML configurations for alert sources.
type ConfigStore struct {
	configs map[string]*AlertSourceConfig // key: source
}

// NewConfigStore loads a list of YAML configurations.
// Each config must have a unique source name within the store.
func NewConfigStore(configs []*AlertSourceConfig) (*ConfigStore, error) {
	cs := &ConfigStore{configs: make(map[string]*AlertSourceConfig)}
	for _, cfg := range configs {
		if cfg.Source == "" {
			return nil, fmt.Errorf("%w: source name is empty", errInvalidMapping)
		}
		if _, ok := cs.configs[cfg.Source]; ok {
			return nil, fmt.Errorf("%w: duplicate source %q", errInvalidMapping, cfg.Source)
		}
		cs.configs[cfg.Source] = cfg
	}
	return cs, nil
}

// Get returns the config for a source, or nil if not found.
func (cs *ConfigStore) Get(source string) *AlertSourceConfig {
	return cs.configs[source]
}

// List returns all configs sorted by priority.
func (cs *ConfigStore) List() []*AlertSourceConfig {
	configs := make([]*AlertSourceConfig, 0, len(cs.configs))
	for _, cfg := range cs.configs {
		configs = append(configs, cfg)
	}
	// Sort by priority ascending (lower number = higher priority)
	for i := 0; i < len(configs)-1; i++ {
		for j := i + 1; j < len(configs); j++ {
			if configs[j].Priority < configs[i].Priority {
				configs[i], configs[j] = configs[j], configs[i]
			}
		}
	}
	return configs
}

// Normalizer transforms raw payloads into StandardAlert using an AlertSourceConfig.
type Normalizer struct {
	config *AlertSourceConfig
}

// NewNormalizer creates a Normalizer bound to a specific AlertSourceConfig.
func NewNormalizer(cfg *AlertSourceConfig) *Normalizer {
	return &Normalizer{config: cfg}
}

// Normalize converts a raw payload into a StandardAlert using the stored config.
// It returns an error only if a required field is missing or all extraction modes fail.
// The raw payload is always preserved in the returned StandardAlert.RawPayload.
func (n *Normalizer) Normalize(
	ctx context.Context,
	tenantID int,
	rawPayload map[string]interface{},
) (*StandardAlert, error) {
	alert := &StandardAlert{
		Source:     n.config.Source,
		SourceRaw:  n.config.Type,
		TenantID:   tenantID,
		Labels:     make(map[string]string),
		Annotations: make(map[string]string),
		Tags:       []string{},
		Status:     "firing",
		RawPayload: rawPayload,
	}

	// Build extraction context with helpers
	ec := &extractCtx{
		payload: rawPayload,
	}

	if err := n.extractAlertID(ec, alert); err != nil {
		return nil, err
	}
	if err := n.extractName(ec, alert); err != nil {
		return nil, err
	}
	if err := n.extractDescription(ec, alert); err != nil {
		return nil, err
	}
	if err := n.extractSeverity(ec, alert); err != nil {
		return nil, err
	}
	n.extractStatus(ec, alert)
	n.extractLabels(ec, alert)
	n.extractAnnotations(ec, alert)
	n.extractSourceIP(ec, alert)
	n.extractService(ec, alert)
	n.extractTags(ec, alert)
	n.extractFiredAt(ec, alert)

	return alert, nil
}

// extractCtx holds intermediate state during field extraction.
type extractCtx struct {
	payload map[string]interface{}
	// body is the top-level "body" key if present, as most webhook payloads nest data there.
	body interface{}
}

func (ec *extractCtx) resolve(path string) interface{} {
	// Support both flat payloads and {"body": {...}} wrapped payloads
	if ec.body == nil {
		if m, ok := ec.payload["body"].(map[string]interface{}); ok {
			ec.body = m
		} else {
			ec.body = ec.payload
		}
	}

	// JSONPath-style: "$.foo.bar" or "$.items[0].name" or "body.alerts[0].labels.severity"
	path = strings.TrimPrefix(path, "$.")
	result := ec.jsonPathGet(ec.payload, path)
	// If nothing found in payload root, try body-wrapped (body is only set if {"body": ...} exists)
	if result == nil && ec.body != nil {
		result = ec.jsonPathGet(ec.body, path)
	}
	return result
}

// jsonPathGet implements a subset of JSONPath: dot notation and array index.
func (ec *extractCtx) jsonPathGet(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}
	segments := strings.Split(path, ".")
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		// Array index: "alerts[0]"
		arrMatch := reArrayIndex.FindStringSubmatch(seg)
		var key string
		var idx int = -1
		if arrMatch != nil {
			key = arrMatch[1]
			idx, _ = strconv.Atoi(arrMatch[2])
		} else {
			key = seg
		}
		if m, ok := data.(map[string]interface{}); ok {
			data = m[key]
		} else {
			return nil
		}
		if data == nil {
			return nil
		}
		if idx >= 0 {
			if arr, ok := data.([]interface{}); ok && idx < len(arr) {
				data = arr[idx]
			} else {
				return nil
			}
		}
	}
	return data
}

var reArrayIndex = regexp.MustCompile(`^(.+)\[(\d+)\]$`)

func (n *Normalizer) extractAlertID(ec *extractCtx, alert *StandardAlert) error {
	val, err := n.extractField(ec, n.config.Mapping.AlertID)
	if err != nil {
		return fmt.Errorf("alert_id: %w", err)
	}
	if val == nil {
		if n.config.Mapping.AlertID.Required {
			return fmt.Errorf("%w: alert_id is required", errMissingRequired)
		}
		val = fmt.Sprintf("%s-%d", alert.Source, time.Now().UnixNano())
	}
	alert.AlertID = fmt.Sprintf("%v", val)
	return nil
}

func (n *Normalizer) extractName(ec *extractCtx, alert *StandardAlert) error {
	val, err := n.extractField(ec, n.config.Mapping.Name)
	if err != nil {
		return fmt.Errorf("name: %w", err)
	}
	if val == nil {
		if n.config.Mapping.Name.Required {
			return fmt.Errorf("%w: name is required", errMissingRequired)
		}
		val = "未命名告警"
	}
	alert.Name = fmt.Sprintf("%v", val)
	return nil
}

func (n *Normalizer) extractDescription(ec *extractCtx, alert *StandardAlert) error {
	val, err := n.extractField(ec, n.config.Mapping.Description)
	if err != nil {
		return fmt.Errorf("description: %w", err)
	}
	if val != nil {
		alert.Description = fmt.Sprintf("%v", val)
	}
	return nil
}

func (n *Normalizer) extractSeverity(ec *extractCtx, alert *StandardAlert) error {
	val, err := n.extractField(ec, n.config.Mapping.Severity)
	if err != nil {
		return fmt.Errorf("severity: %w", err)
	}
	if val == nil {
		alert.Severity = SeverityP3
		return nil
	}

	rawSev := fmt.Sprintf("%v", val)

	// Apply severity transform if configured
	transform := n.config.Mapping.Severity.Transform
	if transform != "" && strings.HasPrefix(transform, "map{") {
		rawSev = applySeverityMap(transform, rawSev)
	} else {
		rawSev = strings.ToLower(rawSev)
	}

	switch rawSev {
	case "critical", "p1", "disaster", "blocker", "fatal":
		alert.Severity = SeverityP1
	case "high", "p2", "major", "error":
		alert.Severity = SeverityP2
	case "medium", "warning", "p3", "warn":
		alert.Severity = SeverityP3
	case "low", "p4", "info", "minor", "debug":
		alert.Severity = SeverityP4
	default:
		alert.Severity = SeverityP3
	}
	return nil
}

func applySeverityMap(transform, value string) string {
	// transform format: map{source:standard,...}
	inner := strings.TrimPrefix(transform, "map{")
	inner = strings.TrimSuffix(inner, "}")
	pairs := strings.Split(inner, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		stdVal := strings.TrimSpace(parts[1])
		if strings.EqualFold(value, key) {
			return stdVal
		}
	}
	return value // no match: pass through unchanged
}

func (n *Normalizer) extractStatus(ec *extractCtx, alert *StandardAlert) {
	val, err := n.extractField(ec, n.config.Mapping.Status)
	if err != nil || val == nil {
		// Status defaults to "firing" — no error for missing status
		alert.Status = "firing"
		return
	}
	raw := strings.ToLower(fmt.Sprintf("%v", val))
	switch raw {
	case "resolved", "ok", "success", "closed":
		alert.Status = "resolved"
	case "acknowledged", "ack":
		alert.Status = "acknowledged"
	default:
		alert.Status = "firing"
	}
}

func (n *Normalizer) extractLabels(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.Labels)
	if m, ok := val.(map[string]interface{}); ok {
		for k, v := range m {
			alert.Labels[k] = fmt.Sprintf("%v", v)
		}
	}
}

func (n *Normalizer) extractAnnotations(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.Annotations)
	if m, ok := val.(map[string]interface{}); ok {
		for k, v := range m {
			alert.Annotations[k] = fmt.Sprintf("%v", v)
		}
	}
}

func (n *Normalizer) extractSourceIP(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.SourceIP)
	if val != nil {
		alert.SourceIP = fmt.Sprintf("%v", val)
	}
}

func (n *Normalizer) extractService(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.Service)
	if val != nil {
		alert.Service = fmt.Sprintf("%v", val)
	}
}

func (n *Normalizer) extractTags(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.Tags)
	if arr, ok := val.([]interface{}); ok {
		for _, item := range arr {
			alert.Tags = append(alert.Tags, fmt.Sprintf("%v", item))
		}
	} else if val != nil {
		alert.Tags = strings.Split(fmt.Sprintf("%v", val), ",")
	}
}

func (n *Normalizer) extractFiredAt(ec *extractCtx, alert *StandardAlert) {
	val, _ := n.extractField(ec, n.config.Mapping.FiredAt)
	if val == nil {
		alert.FiredAt = time.Now()
		return
	}
	parsed := false
	switch v := val.(type) {
	case string:
		t, err := parseTime(v)
		if err == nil {
			alert.FiredAt = t
			parsed = true
		}
	case float64:
		ts := int64(v)
		if ts > 1e12 {
			ts /= 1000
		}
		alert.FiredAt = time.Unix(ts, 0)
		parsed = true
	case int64:
		alert.FiredAt = time.Unix(v, 0)
		parsed = true
	}
	// If parsing failed, use a sentinel zero time — do NOT silently substitute Now()
	// so that callers can detect an unparseable timestamp.
	if !parsed {
		alert.FiredAt = time.Time{}
	}
}

// extractField tries each extraction mode in priority order: JMESPath -> Template -> Literal.
func (n *Normalizer) extractField(ec *extractCtx, fm FieldMapping) (interface{}, error) {
	var val interface{}
	var err error

	// 1. JMESPath
	if fm.JMESPath != "" {
		val = ec.resolve(fm.JMESPath)
		if val != nil {
			val, err = applyTransform(val, fm.Transform)
			return val, err
		}
	}

	// 2. Template
	if fm.Template != "" {
		val, err = n.executeTemplate(fm.Template, ec.payload)
		if err == nil && val != "" {
			v, _ := applyTransform(val, fm.Transform)
			return v, nil
		}
	}

	// 3. Literal
	if fm.Literal != "" {
		val, _ := applyTransform(fm.Literal, fm.Transform)
		return val, nil
	}

	// 4. Default
	if fm.Default != "" {
		val, _ := applyTransform(fm.Default, fm.Transform)
		return val, nil
	}

	return nil, nil
}

func (n *Normalizer) executeTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := jsonMarshal(v)
			return string(b)
		},
		"base64": func(s string) string {
			return base64Encode(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		"default": func(s, def string) string {
			if s == "" {
				return def
			}
			return s
		},
	}).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errTemplateRender, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", errTemplateRender, err)
	}
	return buf.String(), nil
}

// applyTransform applies a named transform to a value.
func applyTransform(val interface{}, transform string) (interface{}, error) {
	if transform == "" || val == nil {
		return val, nil
	}

	s := fmt.Sprintf("%v", val)

	switch transform {
	case "lower":
		return strings.ToLower(s), nil
	case "upper":
		return strings.ToUpper(s), nil
	case "trim":
		return strings.TrimSpace(s), nil
	case "int":
		if i, err := strconv.Atoi(s); err == nil {
			return i, nil
		}
		return 0, nil
	case "float":
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, nil
		}
		return 0.0, nil
	case "bool":
		return strings.ToLower(s) == "true" || s == "1", nil
	case "flatten":
		if m, ok := val.(map[string]interface{}); ok {
			pairs := make([]string, 0, len(m))
			for k, v := range m {
				pairs = append(pairs, k+"="+fmt.Sprintf("%v", v))
			}
			return strings.Join(pairs, ","), nil
		}
		return s, nil
	default:
		if strings.HasPrefix(transform, "map{") {
			return applySeverityMap(transform, s), nil
		}
		// regex extract: "regex(.*pattern.*)"
		if strings.HasPrefix(transform, "regex(") {
			re := extractRegexPattern(transform)
			if re != nil {
				matches := re.FindStringSubmatch(s)
				if len(matches) > 1 {
					return matches[1], nil
				}
			}
		}
		return s, nil
	}
}

var regexExtractRe = regexp.MustCompile(`^regex\((.+)\)$`)

func extractRegexPattern(transform string) *regexp.Regexp {
	m := regexExtractRe.FindStringSubmatch(transform)
	if len(m) < 2 {
		return nil
	}
	re, err := regexp.Compile(m[1])
	if err != nil {
		return nil
	}
	return re
}

func parseTime(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04:05",
		"02/Jan/2006:15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 GMT",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time: cannot parse %q", s)
}

// jsonMarshal serializes v to JSON using the standard library.
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func base64Encode(s string) string {
	return encodeBase64(s)
}

var base64Std = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func encodeBase64(s string) string {
	b := []byte(s)
	n := len(b)
	out := make([]byte, (n+2)/3*4)
	j := 0
	for i := 0; i < n; i += 3 {
		var val uint32
		switch {
		case i+2 < n:
			val = uint32(b[i])<<16 | uint32(b[i+1])<<8 | uint32(b[i+2])
		case i+1 < n:
			val = uint32(b[i])<<16 | uint32(b[i+1])<<8
		default:
			val = uint32(b[i]) << 16
		}
		out[j] = base64Std[(val>>18)&0x3F]
		out[j+1] = base64Std[(val>>12)&0x3F]
		if i+1 < n {
			out[j+2] = base64Std[(val>>6)&0x3F]
		} else {
			out[j+2] = '='
		}
		if i+2 < n {
			out[j+3] = base64Std[val&0x3F]
		} else {
			out[j+3] = '='
		}
		j += 4
	}
	return string(out)
}
