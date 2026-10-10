package connector

import (
	"context"
	"encoding/json"

	"itsm-backend/connector"

	domainCommon "itsm-backend/handlers/common"

	"go.uber.org/zap"
)

// AuditRecorder 是 connector 域落审计行的最小依赖，
// handler 经由该接口写审计，不得直接访问 ent.Client。
type AuditRecorder interface {
	CreateAuditLog(ctx context.Context, l *domainCommon.AuditLog) error
}

// RecordInboundFailure 记录 webhook 验签/解析失败审计。
// headers 先经 RedactHeaders 屏蔽 signature/token/secret 类字段，避免
// audit_log 成为未认证 caller 的密钥归档通道。fail-closed：落库失败仅记日志，
// 不阻塞 webhook 响应。
func RecordInboundFailure(ctx context.Context, recorder AuditRecorder, tenantID int, action, path, reason string, headers map[string]string, logger *zap.SugaredLogger) {
	if recorder == nil || logger == nil {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{"reason": reason, "headers": connector.RedactHeaders(headers)})
	if err := recorder.CreateAuditLog(ctx, &domainCommon.AuditLog{
		TenantID:    tenantID,
		Resource:    "connector_inbound",
		Action:      action,
		Method:      "POST",
		Path:        path,
		StatusCode:  401,
		RequestBody: string(body),
	}); err != nil {
		logger.Warnw("connector inbound audit failure log failed", "action", action, "error", err)
	}
}
