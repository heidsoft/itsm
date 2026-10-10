// Package dingtalk 钉钉开放平台连接器的 HTTP handler：仅承接入站回调，
// 发送路径由 connector.Manager.Send 统一管理。
package dingtalk

import (
	"fmt"

	"itsm-backend/common"
	"itsm-backend/connector"

	connectorDomain "itsm-backend/handlers/connector"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 钉钉入站回调 + 验签 handler。
// 解析 / 验签由 connector.builtin.dingtalk.DingTalk 实现。
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

// RegisterRoutes 把公开回调注册到 public 路由组（IM 回调无需 JWT）：
//
//	POST /api/v1/dingtalk/webhook/:instance_id
func (h *Handler) RegisterRoutes(public *gin.RouterGroup) {
	public.POST("/dingtalk/webhook/:instance_id", h.Webhook)
}

// Webhook 钉钉 stream 入站入口：解析 instance_id → tenant → connector，
// 走统一验签 + 持久化去重 + audit_log 失败记录。
func (h *Handler) Webhook(c *gin.Context) {
	instanceID := c.Param("instance_id")
	conn, tenantID, ok := h.manager.GetByCallbackInstanceID("dingtalk", instanceID)
	if !ok {
		common.Fail(c, common.NotFoundCode, "未找到钉钉连接器实例")
		return
	}
	rcv, ok := conn.(connector.Receiver)
	if !ok {
		common.Fail(c, common.InternalErrorCode, "dingtalk 渠道未实现 Receiver 接口")
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
		h.recordAuditFailure(tenantID, "dingtalk.webhook.verify_signature", err.Error(), headers)
		common.Fail(c, common.ForbiddenCode, "签名校验失败")
		return
	}
	msg, err := rcv.ParseInbound(rawData)
	if err != nil {
		h.recordAuditFailure(tenantID, "dingtalk.webhook.parse_inbound", err.Error(), headers)
		common.Fail(c, common.ParamErrorCode, "事件 payload 解析失败")
		return
	}
	if msg.Type == "url_verification" {
		c.JSON(200, gin.H{"echostr": msg.Content})
		return
	}
	// url_verification 不需要持久化去重，但业务事件用 (tenant, dingtalk, event_id) 去重
	if msg.MessageID != "" && h.dedup != nil {
		seen, err := h.dedup.HandleBeforeProcess(c.Request.Context(), tenantID, "dingtalk", msg.MessageID, rawData)
		if err != nil {
			h.logger.Errorw("dingtalk inbound dedup error", "tenant_id", tenantID, "error", err)
			common.Fail(c, common.InternalErrorCode, "持久化去重失败")
			return
		}
		if seen {
			h.logger.Infow("dingtalk duplicate inbound", "tenant_id", tenantID, "event_id", msg.MessageID)
			c.JSON(200, gin.H{"code": 0, "msg": "duplicate"})
			return
		}
		_ = h.dedup.MarkProcessed(c.Request.Context(), tenantID, "dingtalk", msg.MessageID, "received")
	}
	h.logger.Infow("dingtalk inbound",
		"tenant_id", tenantID, "type", msg.Type, "user", msg.UserID, "chat", msg.ChatID)
	c.JSON(200, gin.H{"code": 0})
}

// recordAuditFailure 把签名 / 解析失败写 audit_log，便于运维追责。
// fail-closed：DB 出错仅写日志，绝不让审计失败阻塞响应。
// headers 在 connector.RecordInboundFailure 内经 RedactHeaders 屏蔽
// signature/token/secret 类 header，避免 audit_log 变成密钥泄露通道。
func (h *Handler) recordAuditFailure(tenantID int, action, reason string, headers map[string]string) {
	connectorDomain.RecordInboundFailure(callerContext(), h.auditRec, tenantID, action,
		fmt.Sprintf("/dingtalk/webhook/%d", tenantID), reason, headers, h.logger)
}
