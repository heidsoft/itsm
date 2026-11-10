package alert

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

const defaultMaxPayloadBytes int64 = 1 << 20

// Handler exposes authenticated, tenant-scoped alert ingestion.
type Handler struct {
	registry         *Registry
	connectorManager *connector.Manager
	repository       alertRepository
	envIsDevelopment bool
}

// NewHandler creates an alert ingestion handler.
func NewHandler(registry *Registry, connectorManager *connector.Manager, client *ent.Client, envIsDevelopment bool) *Handler {
	if registry == nil {
		registry = Default()
	}
	return &Handler{registry: registry, connectorManager: connectorManager, repository: newEntAlertRepository(client), envIsDevelopment: envIsDevelopment}
}

// Ingest receives a monitoring webhook and normalizes it into StandardAlert.
func (h *Handler) Ingest(c *gin.Context) {
	if h == nil || h.registry == nil || h.connectorManager == nil || h.repository == nil {
		common.Fail(c, common.InternalErrorCode, "Alarm access service is not ready")
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		common.Fail(c, common.ParamErrorCode, "Content-Type must be application/json")
		return
	}

	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "Tenant information is missing or in the wrong format")
		return
	}

	factory, ok := h.registry.Get(strings.TrimSpace(c.Param("source")))
	if !ok {
		common.Fail(c, common.ParamErrorCode, "Unknown or unenabled alarm source")
		return
	}
	source := factory()
	if source == nil {
		common.Fail(c, common.InternalErrorCode, "Alarm source initialization failed")
		return
	}
	defer source.Close()

	maxBytes, signatureHeader, signatureSecret := sourceWebhookConfig(source)
	limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			common.Fail(c, common.ParamErrorCode, "Alarm payload exceeds size limit")
			return
		}
		common.Fail(c, common.InternalErrorCode, "Failed to read alarm payload")
		return
	}

	providedSignature := c.GetHeader(signatureHeader)
	if err := VerifyWebhookSignature(body, signatureHeader, signatureSecret, providedSignature, h.envIsDevelopment); err != nil {
		common.Fail(c, common.AuthFailedCode, "Webhook signature verification failed")
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil || payload == nil {
		common.Fail(c, common.ValidationError, "Webhook payload must be a valid JSON object")
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		common.Fail(c, common.ValidationError, "Webhook payload can only contain a JSON object")
		return
	}
	if !source.ValidatePayload(payload) {
		common.Fail(c, common.ValidationError, "Webhook payload verification failed")
		return
	}

	normalized, err := source.Normalize(c.Request.Context(), tenantID, payload)
	if err != nil {
		common.FailWithData(c, common.UnprocessableEntityCode, "Alarm standardization failed", gin.H{"detail": err.Error()})
		return
	}
	if normalized == nil {
		common.Fail(c, common.InternalErrorCode, "The alarm source returned empty results")
		return
	}
	if _, _, err := h.repository.Store(c.Request.Context(), tenantID, normalized); err != nil {
		common.Fail(c, common.InternalErrorCode, "Failed to save alarm")
		return
	}
	common.Success(c, gin.H{
		"alert_id": normalized.AlertID,
		"source":   normalized.Source,
		"severity": normalized.Severity,
		"status":   normalized.Status,
	})
}

func tenantIDFromContext(c *gin.Context) (int, bool) {
	value, exists := c.Get("tenant_id")
	if !exists {
		return 0, false
	}
	tenantID, ok := value.(int)
	return tenantID, ok && tenantID > 0
}

func sourceWebhookConfig(source AlertSource) (int64, string, string) {
	maxBytes := defaultMaxPayloadBytes
	header := "X-Signature"
	webhook, ok := source.(*WebhookAlertSource)
	if !ok || webhook.config == nil {
		return maxBytes, header, ""
	}
	if webhook.config.Cfg.MaxPayloadBytes > 0 {
		maxBytes = int64(webhook.config.Cfg.MaxPayloadBytes)
	}
	if configured := strings.TrimSpace(webhook.config.Cfg.SignatureHeader); configured != "" {
		header = configured
	}
	return maxBytes, header, webhook.config.Cfg.SignatureSecret
}
