// Package wecom 企业微信连接器的 HTTP handler：仅承接入站回调。
// 发送路径由 connector.Manager.Send 统一管理。
package wecom

import (
	"fmt"

	"itsm-backend/common"
	"itsm-backend/connector"

	connectorDomain "itsm-backend/handlers/connector"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	manager  *connector.Manager
	dedup    *connector.InboundDedup
	logger   *zap.SugaredLogger
	auditRec connectorDomain.AuditRecorder
}

func NewHandler(mgr *connector.Manager, dedup *connector.InboundDedup, logger *zap.SugaredLogger) *Handler {
	return &Handler{manager: mgr, dedup: dedup, logger: logger}
}

// SetAuditRecorder 注入审计写入器，用于验签失败时写 audit_log。
func (h *Handler) SetAuditRecorder(rec connectorDomain.AuditRecorder) { h.auditRec = rec }

// RegisterRoutes 公开回调路由（无需 JWT）：
//
//	POST /api/v1/wecom/webhook/:instance_id
func (h *Handler) RegisterRoutes(public *gin.RouterGroup) {
	public.POST("/wecom/webhook/:instance_id", h.Webhook)
}

// Webhook 企业微信入站：验签（msg_signature + timestamp/nonce）→
// 持久化去重 → audit_log 失败落库。
func (h *Handler) Webhook(c *gin.Context) {
	instanceID := c.Param("instance_id")
	conn, tenantID, ok := h.manager.GetByCallbackInstanceID("wecom", instanceID)
	if !ok {
		common.Fail(c, common.NotFoundCode, "未找到企业微信连接器实例")
		return
	}
	rcv, ok := conn.(connector.Receiver)
	if !ok {
		common.Fail(c, common.InternalErrorCode, "wecom 渠道未实现 Receiver 接口")
		return
	}
	rawData, _ := c.GetRawData()
	headers := map[string]string{}
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	if err := rcv.VerifySignature(headers, rawData); err != nil {
		h.recordAuditFailure(tenantID, "wecom.webhook.verify_signature", err.Error(), headers)
		common.Fail(c, common.ForbiddenCode, "签名校验失败")
		return
	}
	msg, err := rcv.ParseInbound(rawData)
	if err != nil {
		h.recordAuditFailure(tenantID, "wecom.webhook.parse_inbound", err.Error(), headers)
		common.Fail(c, common.ParamErrorCode, "事件 payload 解析失败")
		return
	}
	if msg.Type == "url_verification" {
		c.String(200, msg.Content)
		return
	}
	if msg.MessageID != "" && h.dedup != nil {
		seen, err := h.dedup.HandleBeforeProcess(c.Request.Context(), tenantID, "wecom", msg.MessageID, rawData)
		if err != nil {
			h.logger.Errorw("wecom inbound dedup error", "tenant_id", tenantID, "error", err)
			common.Fail(c, common.InternalErrorCode, "持久化去重失败")
			return
		}
		if seen {
			h.logger.Infow("wecom duplicate inbound", "tenant_id", tenantID, "event_id", msg.MessageID)
			c.JSON(200, gin.H{"errcode": 0, "errmsg": "duplicate"})
			return
		}
		_ = h.dedup.MarkProcessed(c.Request.Context(), tenantID, "wecom", msg.MessageID, "received")
	}
	h.logger.Infow("wecom inbound",
		"tenant_id", tenantID, "type", msg.Type, "content", msg.Content)
	c.JSON(200, gin.H{"errcode": 0, "errmsg": "ok"})
}

func (h *Handler) recordAuditFailure(tenantID int, action, reason string, headers map[string]string) {
	// headers 在 connector.RecordInboundFailure 内经 RedactHeaders 屏蔽
	// signature / msg_signature / token / aes_key，避免 audit_log 泄露未
	// 通过认证的 caller 提交的 secret material。
	connectorDomain.RecordInboundFailure(callerContext(), h.auditRec, tenantID, action,
		fmt.Sprintf("/wecom/webhook/%d", tenantID), reason, headers, h.logger)
}
