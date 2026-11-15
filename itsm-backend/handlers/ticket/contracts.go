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

// TicketWorkflowTerminator 在「工单进入终态」时终止关联的 BPMN 流程实例。
//
// 存在的理由是一条实测出来的分叉：工单有两个可达的关闭入口——详情页走
// `POST /tickets/workflow/close`（内部服务层，同事务终止实例并取消待办），列表页
// 走 `POST /tickets/:id/close`（本包，原先只 `UpdateStatus("closed")`，**完全不碰
// 流程**）。同一条工单在列表页关闭后，实例仍是 running、待办仍然可见；用户再把待办
// 完成掉，引擎沿网关继续 createUserTask（内置 ticket_general_flow 的「升级处理」是
// 回环），于是出现「工单已关闭却不断冒出新的待办」—— 这就是浏览器 QA 的 F1。
// 只改列表查询把待办藏起来是掩耳盗铃：残留任务被完成后仍会复活。
//
// 因此终态收敛必须落在 service 层的**每一个**写 closed 的路径上，而不是某一个
// handler。用窄接口而不是直接依赖服务实现，是为了不让 handlers/ticket 反向依赖
// internal/services（那是层次倒置，且会拖进 BPMN 执行器整条依赖链）。
//
// 语义约定（实现方负责保证）：
//   - 无运行中实例 → nil（幂等，兼容未绑定流程的存量工单）；
//   - 存在未完成的审批节点待办 → 409，绝不代签或绕过职责分离；
//   - 其余 → 终止实例 + 取消遗留待办。
type TicketWorkflowTerminator interface {
	// TerminateTicketWorkflowTx 在调用方已开启的事务内终止流程实例。
	// txc 必须是该事务的 ent client——传 nil 或另开 client 都会失去原子性，
	// 变成「状态回滚了、实例却已终止」的反向分叉。
	TerminateTicketWorkflowTx(ctx context.Context, txc *ent.Client, tenantID, actorUserID, ticketID int, reason string) error
}

// TxTicketCloser 是「把工单写成 closed 且带事务钩子」的窄契约。
//
// 状态写入与流程终止必须同事务，而 ent.Tx 只在 repository 层可见，因此关闭动作本身
// 也得由 repository 承担：hook 在事务内、在状态写入之后执行（与
// repository/ticket.TransactionalUpdater.UpdateWithTxHook 语义一致）。
// 用 hook 而不是让 service 直接开事务，是因为 handlers/ticket 不该持有 *ent.Client。
//
// 不放进 Repository 主接口：那是「工单域的读写全集」，而本能力只在终态收敛时使用；
// 用类型断言按需接线，未接线的仓储（如部分测试替身）仍可编译，走无流程收口的旧行为。
type TxTicketCloser interface {
	// CloseTicketWithTxHook 以 CAS（version 条件）把工单置为 closed，并在同一事务内
	// 执行 hook。hook 返回错误则整体回滚。
	CloseTicketWithTxHook(ctx context.Context, ticketID int, version int, tenantID int, hook TxCloseHook) (*Ticket, error)
}

// TxCloseHook 在已开启的事务内被调用，用于终止关联流程实例。
// 参数是事务内的 ent client（*ent.Client 视图），不是裸 ctx：拿到它才能保证
// 流程终止与工单状态写入同生共死。
type TxCloseHook func(ctx context.Context, txc *ent.Client) error
