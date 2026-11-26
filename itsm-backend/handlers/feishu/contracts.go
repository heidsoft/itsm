package feishu

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// InboundTriageAdvisor 是入站建单后生成分诊建议的窄契约。
//
// 不直接 import handlers/ai 或 internal/services：前者会形成域间强耦合，
// 后者被 Gate C.8 禁止。实现由 bootstrap 注入 *service.TriageService。
// 建议只做决策支持，绝不自动改写工单字段。
type InboundTriageAdvisor interface {
	SuggestForTenant(ctx context.Context, title, description string, tenantID int) dto.TriageResult
}

// InboundDecisionAuditor 把服务端产出的建议写入 ai_feedbacks 决策底账
// （item_type='ai_audit'，accepted=NULL），供操作者经 PATCH /ai/audit-logs/:id 人工确认。
type InboundDecisionAuditor interface {
	RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)
}

// TicketWorkflowTerminator 在「飞书侧任务被删除」导致工单进入终态时，
// 终止关联的 BPMN 流程实例。
//
// 这条接线补的是 F1 的最后一条可达路径。F1 的根因是「收敛点放错层」——
// 修复只覆盖了详情页 handler，而写 closed 的入口一共三条。前两条
// （`POST /tickets/:id/close`、`PUT /tickets/:id/status`）已在工单域 service
// 层统一收口；剩下这一条在连接器域：飞书任务被删除时，`markTaskDeleted`
// 直接 `SetStatus("closed")`，工单关了但 BPMN 实例还在 running。
//
// 残留后果与工单域完全一致：待办被完成后，内置 `ticket_general_flow` 的
// 「升级处理」是回环，引擎会继续 createUserTask，表现为「工单已关闭却
// 不断冒出新待办」。
//
// 用窄接口而不是直接 import internal/services，是因为 Gate C.8 明确禁止
// `handlers/` 依赖 `internal/services`（层次倒置，且会拖进 BPMN 执行器整条
// 依赖链）。装配由 bootstrap 注入，本域只声明它需要什么能力。
//
// 语义约定（实现方负责保证，与 handlers/ticket 的同名契约一致）：
//   - 无运行中实例 → nil（幂等，兼容未绑定流程的存量工单）；
//   - 存在未完成的审批节点待办 → 返回错误，绝不代签或绕过职责分离；
//   - 其余 → 终止实例 + 取消遗留待办。
type TicketWorkflowTerminator interface {
	// TerminateTicketWorkflowTx 在调用方已开启的事务内终止流程实例。
	// txc 必须是该事务的 ent client——传 nil 或另开 client 会失去原子性，
	// 变成「工单状态回滚了、流程实例却已终止」的反向分叉。
	TerminateTicketWorkflowTx(ctx context.Context, txc *ent.Client, tenantID, actorUserID, ticketID int, reason string) error
}

// feishuCloseReason 是写进流程终止审计的原因。
//
// 必须能让运维一眼看出「这条工单为什么被关」——飞书侧删任务属于外部系统
// 发起的状态推进，与人工关闭在审计上必须可区分，否则事后追责时无从判断
// 「谁关的」。
const feishuCloseReason = "飞书侧任务被删除，工单随之关闭"

// SetWorkflowTerminator 注入流程终止器。
//
// 未注入时 markTaskDeleted 仍会关单，但必须留 Warnw 而不是静默——
// 静默降级正是 F1 能长期存在而无人发现的成因。
func (s *FeishuSyncService) SetWorkflowTerminator(t TicketWorkflowTerminator) {
	s.terminator = t
}
