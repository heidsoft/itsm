package ticket_notification

// 域内窄契约：工单通知的发送与列表，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
)

type TicketNotificationService interface {
	SendNotification(ctx context.Context, ticketID int, req *dto.SendTicketNotificationRequest, tenantID int) error
	ListTicketNotifications(ctx context.Context, ticketID, tenantID, viewerID int, viewerRole string) ([]*dto.TicketNotificationResponse, error)
}
