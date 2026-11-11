package escalation_matrix

import (
	"context"

	"itsm-backend/dto"
)

// MatrixService 是本域对升级矩阵服务的窄契约，方法集等于本域所有
// 生产文件对 escalationMatrixService 字段的真实调用面。
//
// EscalationLevel / EscalationMatrix 已下沉到 dto 包，注册表实现本体也已
// 随域归位到本包，因此 GetMatrix / SetMatrix 的签名在本文件内可完整表达
// 而无需引用共享层。
type MatrixService interface {
	GetMatrix(tenantID int) dto.EscalationMatrix
	SetMatrix(tenantID int, matrix dto.EscalationMatrix) error
	InvalidateCache(tenantID int)
}

// NotificationSender 是升级扫描器对工单通知能力的最小依赖（依赖倒置）。
// 2026-11-11 迁入本域前直接持有 *service.TicketNotificationService；
// 改为域内单方法端口后由 internal/bootstrap 注入具体实现，域包不再依赖 service/。
type NotificationSender interface {
	SendNotification(ctx context.Context, ticketID int, req *dto.SendTicketNotificationRequest, tenantID int) error
}
