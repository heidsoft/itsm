package ticket

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	repoTicket "itsm-backend/repository/ticket"
)

// 工单模板 CRUD、导出与 SLA 读面由组合根注入实现。方法集 = handlers/ticket
// 下所有生产文件对 productionSvc / slaMonitor 字段的真实调用并集。
//
// 注意共享层工单服务的 GetTicketSLAInfo 与 CalculateSLAMetrics 返回值
// 已下沉为 dto.TicketSLAInfo / dto.SLAMonitorMetrics（service 侧仅保留别名），
// 因此契约无需引用共享层即可保持签名一致。

type TicketProductionService interface {
	AssignTicket(ctx context.Context, ticketID, assigneeID, tenantID int) (*repoTicket.Ticket, error)
	AssignTicketWithWorkflowAdvance(ctx context.Context, ticketID, assigneeID, tenantID, actorUserID int) (*repoTicket.Ticket, error)
	CopyTicketTemplate(ctx context.Context, tenantID, templateID int, newName string) (*dto.TicketTemplate, error)
	CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*repoTicket.Ticket, error)
	CreateTicketTemplate(ctx context.Context, tenantID int, req interface{}) (interface{}, error)
	DeleteTicketTemplate(ctx context.Context, tenantID, templateID int) error
	ExportTickets(ctx context.Context, tenantID int, filters map[string]interface{}, format string, currentUserID int, currentRole string) ([]byte, error)
	GetTicketActivity(ctx context.Context, ticketID, tenantID int) ([]map[string]interface{}, error)
	GetTicketSLAInfo(ctx context.Context, ticketID, tenantID int) (*dto.TicketSLAInfo, error)
	GetTicketTemplate(ctx context.Context, tenantID, templateID int) (*dto.TicketTemplate, error)
	GetTicketTemplateCategories(ctx context.Context, tenantID int) ([]string, error)
	GetTicketTemplates(ctx context.Context, tenantID int) ([]interface{}, error)
	UpdateTicketTemplate(ctx context.Context, tenantID, templateID int, req interface{}) (interface{}, error)
	UpdateTicketTemplateStatus(ctx context.Context, tenantID, templateID int, isActive bool) (*dto.TicketTemplate, error)
	ValidateAssignee(ctx context.Context, assigneeID, tenantID int) error
}

type TicketSLAMonitor interface {
	CalculateSLAMetrics(ctx context.Context, tenantID int, startTime, endTime time.Time) (*dto.SLAMonitorMetrics, error)
	GetSLAState(ctx context.Context, tenantID int, aggregateType string, aggregateID int) (*ent.SLAState, error)
	PauseSLA(ctx context.Context, tenantID int, entityType string, entityID int, reason string) error
	ResumeSLA(ctx context.Context, tenantID int, entityType string, entityID int) error
}
