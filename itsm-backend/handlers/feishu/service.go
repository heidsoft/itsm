package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/feishuticketsync"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type FeishuSyncService struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	terminator TicketWorkflowTerminator
}

func NewFeishuSyncService(client *ent.Client, logger *zap.SugaredLogger) *FeishuSyncService {
	return &FeishuSyncService{client: client, logger: logger}
}

func (s *FeishuSyncService) SyncTicketToFeishu(ctx context.Context, tenantID, ticketID int, fc *feishuConn.Feishu) (*dto.FeishuTicketSyncResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not configured")
	}
	ticket, err := s.client.Ticket.Query().
		Where(entTicket.ID(ticketID), entTicket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("query ticket: %w", err)
	}

	taskPayload := s.ticketToFeishuTask(ctx, ticket)
	syncRecord, err := s.client.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tenantID), feishuticketsync.TicketID(ticketID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("query feishu sync record: %w", err)
	}

	var task *feishuConn.FeishuTask
	if syncRecord != nil && syncRecord.FeishuTaskGUID != "" {
		task, err = fc.UpdateTask(ctx, syncRecord.FeishuTaskGUID, taskPayload)
		if err != nil {
			_ = s.markSyncFailed(ctx, syncRecord, err)
			return nil, fmt.Errorf("update feishu task: %w", err)
		}
		syncRecord, err = syncRecord.Update().
			SetFeishuTaskID(firstNonEmptyString(task.GUID, syncRecord.FeishuTaskID)).
			SetFeishuTaskGUID(firstNonEmptyString(task.GUID, syncRecord.FeishuTaskGUID)).
			SetSyncStatus("synced").
			SetLastSyncDirection("itsm_to_feishu").
			SetLastSyncedAt(time.Now()).
			ClearErrorMessage().
			Save(ctx)
	} else {
		task, err = fc.CreateTask(ctx, taskPayload)
		if err != nil {
			return nil, fmt.Errorf("create feishu task: %w", err)
		}
		if task == nil || task.GUID == "" {
			return nil, fmt.Errorf("create feishu task: empty task guid")
		}
		syncRecord, err = s.client.FeishuTicketSync.Create().
			SetTenantID(tenantID).
			SetTicketID(ticket.ID).
			SetFeishuTaskID(task.GUID).
			SetFeishuTaskGUID(task.GUID).
			SetSyncStatus("synced").
			SetLastSyncDirection("itsm_to_feishu").
			SetLastSyncedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("save feishu sync record: %w", err)
	}
	return toFeishuSyncResponse(syncRecord, ticket, task), nil
}

func (s *FeishuSyncService) HandleTaskEvent(ctx context.Context, tenantID int, fc *feishuConn.Feishu, eventType string, eventData map[string]interface{}) (*dto.FeishuWebhookResponse, error) {
	taskGUID := extractFeishuTaskGUID(eventData)
	if taskGUID == "" {
		return nil, fmt.Errorf("missing feishu task guid")
	}

	if eventType == "task.deleted" {
		resp, err := s.markTaskDeleted(ctx, tenantID, taskGUID)
		if err != nil {
			return nil, err
		}
		return &dto.FeishuWebhookResponse{EventType: eventType, Action: "ticket_closed", Sync: resp}, nil
	}

	task, err := fc.GetTask(ctx, taskGUID)
	if err != nil {
		return nil, fmt.Errorf("get feishu task: %w", err)
	}
	resp, action, err := s.SyncFeishuTaskToTicket(ctx, tenantID, task)
	if err != nil {
		return nil, err
	}
	return &dto.FeishuWebhookResponse{EventType: eventType, Action: action, Sync: resp}, nil
}

func (s *FeishuSyncService) SyncFeishuTaskToTicket(ctx context.Context, tenantID int, task *feishuConn.FeishuTask) (*dto.FeishuTicketSyncResponse, string, error) {
	if task == nil || task.GUID == "" {
		return nil, "", fmt.Errorf("feishu task guid is required")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("start transaction: %w", err)
	}
	defer tx.Rollback()

	syncRecord, err := tx.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tenantID), feishuticketsync.FeishuTaskID(task.GUID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, "", fmt.Errorf("query feishu sync record: %w", err)
	}

	var ticket *ent.Ticket
	action := "updated"
	if syncRecord != nil {
		ticket, err = s.updateTicketFromFeishuTask(ctx, tx, tenantID, syncRecord.TicketID, task)
		if err != nil {
			_, _ = syncRecord.Update().SetSyncStatus("failed").SetErrorMessage(err.Error()).Save(ctx)
			return nil, "", err
		}
		syncRecord, err = syncRecord.Update().
			SetSyncStatus("synced").
			SetLastSyncDirection("feishu_to_itsm").
			SetLastSyncedAt(time.Now()).
			ClearErrorMessage().
			Save(ctx)
	} else {
		action = "created"
		ticket, err = s.createTicketFromFeishuTask(ctx, tx, tenantID, task)
		if err != nil {
			return nil, "", err
		}
		syncRecord, err = tx.FeishuTicketSync.Create().
			SetTenantID(tenantID).
			SetTicketID(ticket.ID).
			SetFeishuTaskID(task.GUID).
			SetFeishuTaskGUID(task.GUID).
			SetSyncStatus("synced").
			SetLastSyncDirection("feishu_to_itsm").
			SetLastSyncedAt(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return nil, "", fmt.Errorf("save feishu sync record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("commit feishu sync transaction: %w", err)
	}
	return toFeishuSyncResponse(syncRecord, ticket, task), action, nil
}

func (s *FeishuSyncService) ticketToFeishuTask(ctx context.Context, ticket *ent.Ticket) *feishuConn.FeishuTask {
	task := &feishuConn.FeishuTask{
		Name:        fmt.Sprintf("%s %s", ticket.TicketNumber, ticket.Title),
		Description: ticket.Description,
		StartTime:   ticket.CreatedAt.Unix(),
		Status:      mapTicketStatusToFeishu(ticket.Status),
		Priority:    mapTicketPriorityToFeishu(ticket.Priority),
		Extra: map[string]interface{}{
			"itsm_ticket_id":     ticket.ID,
			"itsm_ticket_number": ticket.TicketNumber,
			"tenant_id":          ticket.TenantID,
		},
	}
	if ticket.AssigneeID > 0 {
		if assignee, err := s.client.User.Get(ctx, ticket.AssigneeID); err == nil && assignee.FeishuOpenID != "" {
			task.Assignees = []string{assignee.FeishuOpenID}
		}
	}
	return task
}

func (s *FeishuSyncService) updateTicketFromFeishuTask(ctx context.Context, tx *ent.Tx, tenantID, ticketID int, task *feishuConn.FeishuTask) (*ent.Ticket, error) {
	current, err := tx.Ticket.Query().
		Where(entTicket.ID(ticketID), entTicket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("query ticket for feishu sync: %w", err)
	}
	update := tx.Ticket.UpdateOneID(ticketID).
		Where(entTicket.TenantID(tenantID)).
		SetVersion(current.Version + 1)
	if task.Name != "" {
		update.SetTitle(stripTicketNumberPrefix(task.Name))
	}
	update.SetDescription(task.Description)
	if task.Priority != "" {
		update.SetPriority(mapFeishuPriorityToTicket(task.Priority))
	}
	if task.Status != "" || task.Completed {
		update.SetStatus(mapFeishuStatusToTicket(task.Status, task.Completed))
	}
	return update.Save(ctx)
}

func (s *FeishuSyncService) createTicketFromFeishuTask(ctx context.Context, tx *ent.Tx, tenantID int, task *feishuConn.FeishuTask) (*ent.Ticket, error) {
	requesterID, err := s.resolveRequesterID(ctx, tx, tenantID, task.CreatorID)
	if err != nil {
		return nil, err
	}
	ticketNumber := fmt.Sprintf("TKT-FS-%d-%d", tenantID, time.Now().UnixNano())
	return tx.Ticket.Create().
		SetTenantID(tenantID).
		SetTicketNumber(ticketNumber).
		SetTitle(stripTicketNumberPrefix(firstNonEmptyString(task.Name, "飞书同步工单"))).
		SetDescription(task.Description).
		SetType("ticket").
		SetPriority(mapFeishuPriorityToTicket(task.Priority)).
		SetStatus(mapFeishuStatusToTicket(task.Status, task.Completed)).
		SetRequesterID(requesterID).
		Save(ctx)
}

func (s *FeishuSyncService) resolveRequesterID(ctx context.Context, tx *ent.Tx, tenantID int, feishuOpenID string) (int, error) {
	if feishuOpenID != "" {
		u, err := tx.User.Query().
			Where(user.TenantID(tenantID), user.FeishuOpenID(feishuOpenID)).
			Only(ctx)
		if err == nil {
			return u.ID, nil
		}
		if err != nil && !ent.IsNotFound(err) {
			return 0, fmt.Errorf("query feishu requester: %w", err)
		}
	}
	u, err := tx.User.Query().
		Where(user.TenantID(tenantID), user.Active(true)).
		Order(ent.Asc(user.FieldID)).
		First(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve default requester for feishu task: %w", err)
	}
	return u.ID, nil
}

func (s *FeishuSyncService) markTaskDeleted(ctx context.Context, tenantID int, taskGUID string) (*dto.FeishuTicketSyncResponse, error) {
	syncRecord, err := s.client.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tenantID), feishuticketsync.FeishuTaskID(taskGUID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("query feishu sync record: %w", err)
	}
	// 状态写入与流程终止必须同事务：否则会出现「工单已置 closed、实例仍在 running」
	// 或反过来的「实例已终止、工单状态回滚」两种脏状态。前者正是 F1 在飞书侧的
	// 复现路径（工单关了、待办还在，待办完成后回环网关继续造新待办）。
	tx, err := s.client.Tx(ctx)
	if err != nil {
		_ = s.markSyncFailed(ctx, syncRecord, err)
		return nil, fmt.Errorf("begin feishu task deletion tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	ticket, err := tx.Ticket.UpdateOneID(syncRecord.TicketID).
		Where(entTicket.TenantID(tenantID)).
		SetStatus("closed").
		Save(ctx)
	if err != nil {
		_ = s.markSyncFailed(ctx, syncRecord, err)
		return nil, fmt.Errorf("close ticket after feishu task deletion: %w", err)
	}

	if err := s.terminateWorkflowTx(ctx, tx.Client(), tenantID, syncRecord.TicketID); err != nil {
		_ = s.markSyncFailed(ctx, syncRecord, err)
		return nil, fmt.Errorf("terminate ticket workflow after feishu task deletion: %w", err)
	}

	// 必须走 tx 而不是 syncRecord.Update()：ent 实体绑定的是创建它的 client
	// （这里是事务外的 s.client），用它的 Update() 会把同步记录写成一个独立事务，
	// 于是「工单关了、流程终止了、同步状态却没落地」——正是本函数要消灭的那种分叉。
	syncRecord, err = tx.FeishuTicketSync.UpdateOneID(syncRecord.ID).
		SetSyncStatus("deleted").
		SetLastSyncDirection("feishu_to_itsm").
		SetLastSyncedAt(time.Now()).
		ClearErrorMessage().
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update feishu sync deletion status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit feishu task deletion tx: %w", err)
	}
	committed = true
	return toFeishuSyncResponse(syncRecord, ticket, &feishuConn.FeishuTask{GUID: taskGUID}), nil
}

// terminateWorkflowTx 在已开启的事务内终止工单关联的 BPMN 实例。
//
// 未注入终止器时不阻断关单（退回旧行为），但必须 Warnw：静默降级正是
// F1 能长期存在而无人察觉的成因。operator 需要能从这个日志看出「本实例
// 没有做流程收口」，而不是等到发现「关掉的工单还在造待办」才回头查。
func (s *FeishuSyncService) terminateWorkflowTx(ctx context.Context, txc *ent.Client, tenantID, ticketID int) error {
	if s.terminator == nil {
		s.logger.Warnw("飞书任务删除关闭工单未触发流程收口：组装根未注入 TicketWorkflowTerminator，"+
			"工单已置 closed 但关联 BPMN 实例可能仍在运行（F1 飞书侧分叉）",
			"ticket_id", ticketID, "tenant_id", tenantID)
		return nil
	}
	// actorUserID 传 0：本次状态推进由飞书侧事件驱动，没有 IT 侧操作人。
	// 审计里靠 reason 区分来源（见 feishuCloseReason），不虚构一个操作人 ID。
	return s.terminator.TerminateTicketWorkflowTx(ctx, txc, tenantID, 0, ticketID, feishuCloseReason)
}

func (s *FeishuSyncService) markSyncFailed(ctx context.Context, syncRecord *ent.FeishuTicketSync, err error) error {
	_, saveErr := syncRecord.Update().
		SetSyncStatus("failed").
		SetErrorMessage(err.Error()).
		Save(ctx)
	return saveErr
}

func mapTicketPriorityToFeishu(priority string) string {
	switch strings.ToLower(priority) {
	case "low":
		return "low"
	case "high":
		return "high"
	case "critical", "urgent":
		return "urgent"
	default:
		return "medium"
	}
}

func mapFeishuPriorityToTicket(priority string) string {
	switch strings.ToLower(priority) {
	case "low":
		return "low"
	case "high":
		return "high"
	case "urgent":
		return "critical"
	default:
		return "medium"
	}
}

func mapTicketStatusToFeishu(status string) string {
	switch strings.ToLower(status) {
	case "in_progress":
		return "in_progress"
	case "resolved", "closed":
		return "completed"
	case "cancelled", "canceled":
		return "canceled"
	default:
		return "not_started"
	}
}

func mapFeishuStatusToTicket(status string, completed bool) string {
	if completed {
		return "resolved"
	}
	switch strings.ToLower(status) {
	case "in_progress":
		return "in_progress"
	case "completed":
		return "resolved"
	case "canceled", "cancelled":
		return "closed"
	default:
		return "open"
	}
}

func extractFeishuTaskGUID(event map[string]interface{}) string {
	for _, key := range []string{"task_guid", "guid", "task_id"} {
		if v, ok := event[key].(string); ok && v != "" {
			return v
		}
	}
	for _, key := range []string{"task", "task_data"} {
		if nested, ok := event[key].(map[string]interface{}); ok {
			if guid := extractFeishuTaskGUID(nested); guid != "" {
				return guid
			}
		}
	}
	return ""
}

func stripTicketNumberPrefix(name string) string {
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 2 && (strings.HasPrefix(parts[0], "TKT-") || strings.HasPrefix(parts[0], "TK-")) {
		return parts[1]
	}
	return name
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func toFeishuSyncResponse(syncRecord *ent.FeishuTicketSync, ticket *ent.Ticket, task *feishuConn.FeishuTask) *dto.FeishuTicketSyncResponse {
	if syncRecord == nil {
		return nil
	}
	resp := &dto.FeishuTicketSyncResponse{
		ID:                syncRecord.ID,
		TenantID:          syncRecord.TenantID,
		TicketID:          syncRecord.TicketID,
		FeishuTaskID:      syncRecord.FeishuTaskID,
		FeishuTaskGUID:    syncRecord.FeishuTaskGUID,
		SyncStatus:        syncRecord.SyncStatus,
		LastSyncDirection: syncRecord.LastSyncDirection,
		ErrorMessage:      syncRecord.ErrorMessage,
	}
	if !syncRecord.LastSyncedAt.IsZero() {
		resp.LastSyncedAt = &syncRecord.LastSyncedAt
	}
	if ticket != nil {
		resp.TicketNumber = ticket.TicketNumber
	}
	if task != nil {
		resp.Task = &dto.FeishuTaskResponse{
			GUID:        task.GUID,
			Name:        task.Name,
			Description: task.Description,
			Status:      task.Status,
			Priority:    task.Priority,
			Completed:   task.Completed,
			CreatorID:   task.CreatorID,
			Assignees:   task.Assignees,
			Extra:       task.Extra,
		}
	}
	return resp
}
