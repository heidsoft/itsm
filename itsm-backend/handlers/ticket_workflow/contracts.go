package ticket_workflow

// 域内窄契约：工单流转动作、状态视图与抄送记录，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
)

type TicketWorkflowService interface {
	AcceptTicket(ctx context.Context, req *dto.AcceptTicketRequest, userID, tenantID int) error
	RejectTicket(ctx context.Context, req *dto.RejectTicketRequest, userID, tenantID int) error
	WithdrawTicket(ctx context.Context, req *dto.WithdrawTicketRequest, userID, tenantID int) error
	ForwardTicket(ctx context.Context, req *dto.ForwardTicketRequest, userID, tenantID int) error
	CCTicket(ctx context.Context, req *dto.CCTicketRequest, userID, tenantID int) error
	ApproveTicket(ctx context.Context, req *dto.ApproveTicketRequest, userID, tenantID int) error
	ResolveTicket(ctx context.Context, req *dto.ResolveTicketRequest, userID, tenantID int) error
	CloseTicket(ctx context.Context, req *dto.CloseTicketRequest, userID, tenantID int) error
	ReopenTicket(ctx context.Context, req *dto.ReopenTicketRequest, userID, tenantID int) error
	ListMyCCRecords(ctx context.Context, userID, tenantID int) (*dto.TicketCCListResponse, error)
	ListTicketCCRecords(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketCCListResponse, error)
	GetTicketWorkflowState(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketWorkflowState, error)
	GetTicketWorkflowStateV2(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketWorkflowState, error)
}
