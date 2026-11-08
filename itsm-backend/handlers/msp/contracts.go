package msp

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/repository/ticket"
)

// TicketService 是 MSP 域真正消费的工单能力窄接口（依赖倒置）。
//
// 迁移前 Handler 直接持有 *service.TicketService，使本域生产文件必须 import
// 共享 service 包；迁移后组合根仍注入同一个共享层实现，但域内只依赖这 5 个方法。
// 契约边界约束：新增方法前先确认 MSP 域确实需要，禁止把共享层大接口原样搬回域包。
type TicketService interface {
	GetCustomerTicketsForMSP(ctx context.Context, userID, customerTenantID int, status *string, page, pageSize int) ([]*ticket.Ticket, int, error)
	AssignMSPTechnician(ctx context.Context, ticketID, customerTenantID, assignerID int) (*ticket.Ticket, error)
	GetMSPCustomerReports(ctx context.Context, mspTenantID int, dateFrom, dateTo time.Time) ([]*dto.MSPCustomerReportResponse, error)
	GetMSPPerformanceReports(ctx context.Context, mspTenantID int, dateFrom, dateTo time.Time) ([]*dto.MSPPerformanceReportResponse, error)
	GetTicketSLAState(ctx context.Context, tenantID, ticketID int) (*ent.SLAState, error)
}
