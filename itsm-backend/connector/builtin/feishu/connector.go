package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/feishuticketsync"
	"itsm-backend/ent/user"
	"itsm-backend/ent/ticket"
)

// Feishu 飞书连接器实现
// 复用 package 内 Client 以享受 tenant_access_token 缓存
type Feishu struct {
	client    *Client
	cfg       connector.Config
	startedAt time.Time
}

// ActionHandler 卡片按钮/回调事件
type ActionHandler func(ctx context.Context, event *CardActionEvent) (map[string]interface{}, error)

// CardActionEvent 飞书卡片交互事件（精简版）
type CardActionEvent struct {
	OpenID    string                 `json:"open_id"`
	UserID    string                 `json:"user_id"`
	ChatID    string                 `json:"chat_id"`
	Action    map[string]interface{} `json:"action"`
	Raw       map[string]interface{} `json:"raw"`
	Token     string                 `json:"token"`
	Timestamp string                 `json:"ts"`
}

func init() {
	connector.MustRegister(func() connector.Connector { return &Feishu{} })
}

func New() *Feishu { return &Feishu{} }

func (f *Feishu) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "feishu",
		Version:     "1.0.0",
		Title:       "飞书 / Lark",
		Provider:    "feishu",
		Type:        connector.TypeIM,
		Description: "飞书/Lark 开放平台连接器：发送/接收消息、卡片回调、签名校验。覆盖中国大陆及海外版本。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapReceiveMessage,
			connector.CapSendCard,
			connector.CapReplyMessage,
			connector.CapCreateTicket,
			connector.CapUpdateTicket,
			connector.CapSyncAssets,
		},
		Tags:     []string{"im", "feishu", "lark", "china"},
		Homepage: "https://open.feishu.cn",
	}
}

func (f *Feishu) Init(_ context.Context, cfg connector.Config) error {
	appID := cfg.Credentials["app_id"]
	appSecret := cfg.Credentials["app_secret"]
	if appID == "" || appSecret == "" {
		return fmt.Errorf("feishu: credentials.app_id and app_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	if baseURL == "" {
		// 海外版判定
		if region, _ := cfg.Settings["region"].(string); region == "intl" {
			baseURL = BaseURLIntl
		}
	}
	f.client = NewClient(baseURL, appID, appSecret, cfg.Credentials["verification_token"], cfg.Credentials["encrypt_key"])
	f.cfg = cfg
	f.startedAt = time.Now()
	return nil
}

func (f *Feishu) Send(ctx context.Context, msg *connector.Message) error {
	if f.client == nil {
		return fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.Send(ctx, msg)
}

func (f *Feishu) HealthCheck(ctx context.Context) connector.HealthStatus {
	if f.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := f.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "tenant_access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": f.startedAt, "uptime_s": int(time.Since(f.startedAt).Seconds())},
	}
}

func (f *Feishu) Close() error { return nil }

// VerifySignature 满足 connector.Receiver 接口
func (f *Feishu) VerifySignature(headers map[string]string, body []byte) error {
	if f.client == nil {
		return fmt.Errorf("feishu: not initialized")
	}
	ts := headers["X-Lark-Request-Timestamp"]
	nonce := headers["X-Lark-Request-Nonce"]
	sig := headers["X-Lark-Signature"]
	if ts == "" || sig == "" {
		return fmt.Errorf("feishu: missing signature headers")
	}
	if !f.client.VerifyEventSignature(ts, nonce, sig, body) {
		return fmt.Errorf("feishu: signature mismatch")
	}
	return nil
}

// ParseInbound 解析飞书事件回调
// 支持：url_verification / event_callback / card.action.trigger
func (f *Feishu) ParseInbound(body []byte) (*connector.InboundMessage, error) {
	var base struct {
		UUID      string `json:"uuid"`
		Token     string `json:"token"`
		Type      string `json:"type"`
		TS        string `json:"ts"`
		Challenge string `json:"challenge"`
		Header    *struct {
			AppID     string `json:"app_id"`
			TenantKey string `json:"tenant_key"`
			EventType string `json:"event_type"`
		} `json:"header"`
		Event map[string]interface{} `json:"event"`
	}
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("feishu: parse inbound: %w", err)
	}
	// URL Verification：原样返回 challenge
	if base.Type == "url_verification" {
		return &connector.InboundMessage{
			Type:       "url_verification",
			Content:    base.Challenge,
			ReceivedAt: time.Now(),
			Extras:     map[string]interface{}{"raw": base},
		}, nil
	}
	// 事件订阅
	if base.Header != nil {
		evType := base.Header.EventType
		msg := &connector.InboundMessage{
			ConnectorType: connector.TypeIM,
			ConnectorName: "feishu",
			MessageID:     base.UUID,
			Type:          evType,
			Raw:           body,
			ReceivedAt:    time.Now(),
			Extras:        map[string]interface{}{"ts": base.TS, "event": base.Event},
		}
		// 常见字段提取（im.message.receive_v1）
		if sender, ok := base.Event["sender"].(map[string]interface{}); ok {
			if sID, ok := sender["sender_id"].(map[string]interface{}); ok {
				if open, ok := sID["open_id"].(string); ok {
					msg.UserID = open
				}
			}
		}
		if msg2, ok := base.Event["message"].(map[string]interface{}); ok {
			if c, ok := msg2["content"].(map[string]interface{}); ok {
				if txt, ok := c["text"].(string); ok {
					msg.Content = txt
				}
			}
			if chID, ok := msg2["chat_id"].(string); ok {
				msg.ChatID = chID
			}
			if ct, ok := msg2["chat_type"].(string); ok {
				msg.ChatType = ct
			}
			if mID, ok := msg2["message_id"].(string); ok {
				msg.MessageID = mID
			}
		}
		// 卡片回调
		if evType == "card.action.trigger" {
			if act, ok := base.Event["action"].(map[string]interface{}); ok {
				msg.Type = "card_action"
				msg.Content = fmt.Sprintf("%v", act)
			}
		}
		// 应用机器人被加入/移除
		// Task events
		if evType == "task.created" || evType == "task.updated" || evType == "task.deleted" {
			msg.Type = "task_event"
			msg.Content = evType
			msg.Extras["task_data"] = base.Event
		}
		if evType == "im.chat.member.bot.added_v1" || evType == "im.chat.member.bot.deleted_v1" {
			msg.Type = evType
		}
		return msg, nil
	}
	return nil, fmt.Errorf("feishu: unknown event type=%s", base.Type)
}

// SyncTicketToFeishu syncs an ITSM ticket to Feishu as a task (creates or updates)
func (f *Feishu) SyncTicketToFeishu(ctx context.Context, tx *ent.Tx, ticket *ent.Ticket) (*FeishuTask, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}

	// Check if there's an existing sync mapping
	syncRecord, err := tx.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(ticket.TenantID), feishuticketsync.TicketID(ticket.ID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("feishu: failed to query sync record: %w", err)
	}

	// Map ticket fields to Feishu task
	feishuTask := &FeishuTask{
		Name:        ticket.Title,
		Description: ticket.Description,
		StartTime:   ticket.CreatedAt.Unix(),
		// Map priority: ITSM low/medium/high/critical -> Feishu low/medium/high/urgent
		Priority: mapPriorityToFeishu(ticket.Priority),
		// Map status: ITSM open/in_progress/resolved/closed -> Feishu not_started/in_progress/completed/completed
		Status: mapStatusToFeishu(ticket.Status),
	}

	var task *FeishuTask
	if syncRecord != nil {
		// Update existing task
		task, err = f.client.UpdateTask(ctx, syncRecord.FeishuTaskGUID, feishuTask)
		if err != nil {
			// Update sync record with error
			_, _ = syncRecord.Update().
				SetSyncStatus("failed").
				SetErrorMessage(err.Error()).
				Save(ctx)
			return nil, fmt.Errorf("feishu: failed to update task: %w", err)
		}
		// Update sync record
		_, err = syncRecord.Update().
			SetSyncStatus("synced").
			SetLastSyncDirection("itsm_to_feishu").
			SetLastSyncedAt(time.Now()).
			ClearErrorMessage().
			Save(ctx)
	} else {
		// Create new task
		task, err = f.client.CreateTask(ctx, feishuTask)
		if err != nil {
			return nil, fmt.Errorf("feishu: failed to create task: %w", err)
		}
		// Create sync record
		_, err = tx.FeishuTicketSync.Create().
			SetTenantID(ticket.TenantID).
			SetTicketID(ticket.ID).
			SetFeishuTaskID(task.GUID). // Wait, is GUID the same as ID? Let's check Feishu API: yes, task GUID is the unique ID
			SetFeishuTaskGUID(task.GUID).
			SetSyncStatus("synced").
			SetLastSyncDirection("itsm_to_feishu").
			SetLastSyncedAt(time.Now()).
			Save(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("feishu: failed to save sync record: %w", err)
	}

	return task, nil
}

// SyncFeishuTaskToTicket syncs a Feishu task to ITSM as a ticket (creates or updates)
func (f *Feishu) SyncFeishuTaskToTicket(ctx context.Context, tx *ent.Tx, feishuTask *FeishuTask) (*ent.Ticket, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}

	// Check if there's an existing sync mapping
	syncRecord, err := tx.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(f.cfg.TenantID), feishuticketsync.FeishuTaskID(feishuTask.GUID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("feishu: failed to query sync record: %w", err)
	}

	// Map Feishu task fields to ITSM ticket
	ticketUpdate := dto.UpdateTicketRequest{
		Title:       feishuTask.Name,
		Description: feishuTask.Description,
		Priority:    mapPriorityFromFeishu(feishuTask.Priority),
		Status:      mapStatusFromFeishu(feishuTask.Status),
		// Set requester: need to map Feishu creator ID to ITSM user ID
		// RequesterID: userID,
	}

	var ticket *ent.Ticket
	if syncRecord != nil {
		// Update existing ticket
		ticket, err = tx.Ticket.Get(ctx, syncRecord.TicketID)
		if err != nil {
			return nil, fmt.Errorf("feishu: failed to get ticket: %w", err)
		}

		update := ticket.Update()
		if ticketUpdate.Title != "" {
			update.SetTitle(ticketUpdate.Title)
		}
		if ticketUpdate.Description != "" {
			update.SetDescription(ticketUpdate.Description)
		}
		if ticketUpdate.Priority != "" {
			update.SetPriority(ticketUpdate.Priority)
		}
		if ticketUpdate.Status != "" {
			update.SetStatus(ticketUpdate.Status)
		}

		ticket, err = update.Save(ctx)
		if err != nil {
			// Update sync record with error
			_, _ = syncRecord.Update().
				SetSyncStatus("failed").
				SetErrorMessage(err.Error()).
				Save(ctx)
			return nil, fmt.Errorf("feishu: failed to update ticket: %w", err)
		}

		// Update sync record
		_, err = syncRecord.Update().
			SetSyncStatus("synced").
			SetLastSyncDirection("feishu_to_itsm").
			SetLastSyncedAt(time.Now()).
			ClearErrorMessage().
			Save(ctx)
	} else {
		// Create new ticket
		createReq := dto.CreateTicketRequest{
			Title:       feishuTask.Name,
			Description: feishuTask.Description,
			Priority:    mapPriorityFromFeishu(feishuTask.Priority),
			Type:        "ticket",
			// Set requester: need to map Feishu creator ID to ITSM user ID
			// RequesterID: userID,
		}

		// Create ticket using the same logic as ticket service
		// TODO: Inject ticket service or reuse create logic
		// For now, we'll create it directly
		create := tx.Ticket.Create().
			SetTitle(createReq.Title).
			SetDescription(createReq.Description).
			SetPriority(createReq.Priority).
			SetType(createReq.Type).
			SetStatus(mapStatusFromFeishu(feishuTask.Status)).
			SetTenantID(f.cfg.TenantID).
	// 映射飞书创建人到ITSM用户
	requesterID := 1 // 默认管理员
	if feishuTask.CreatorID != "" {
		user, err := tx.User.Query().
			Where(user.FeishuOpenID(feishuTask.CreatorID)).
			Where(user.TenantID(f.cfg.TenantID)).
			Only(ctx)
		if err == nil {
			requesterID = user.ID
		}
	}
	create := tx.Ticket.Create().
		SetTitle(createReq.Title).
		SetDescription(createReq.Description).
		SetPriority(createReq.Priority).
		SetType(createReq.Type).
		SetStatus(mapStatusFromFeishu(feishuTask.Status)).
		SetTenantID(f.cfg.TenantID).
		SetRequesterID(requesterID)

		if createReq.AssigneeID > 0 {
			create.SetAssigneeID(createReq.AssigneeID)
		}
		// Generate ticket number (same logic as ticket service)
		ticketNumber := fmt.Sprintf("TK-%d-%s", f.cfg.TenantID, time.Now().Format("20060102150405"))
		create.SetTicketNumber(ticketNumber)

		ticket, err = create.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("feishu: failed to create ticket: %w", err)
		}

		// Create sync record
		_, err = tx.FeishuTicketSync.Create().
			SetTenantID(f.cfg.TenantID).
			SetTicketID(ticket.ID).
			SetFeishuTaskID(feishuTask.GUID).
			SetFeishuTaskGUID(feishuTask.GUID).
			SetSyncStatus("synced").
			SetLastSyncDirection("feishu_to_itsm").
			SetLastSyncedAt(time.Now()).
			Save(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("feishu: failed to save sync record: %w", err)
	}

	return ticket, nil
}

// HandleTaskEvent handles Feishu task webhook events
func (f *Feishu) HandleTaskEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error {
	if f.client == nil {
		return fmt.Errorf("feishu: connector not initialized")
	}

	// Extract task GUID from event
	taskGUID, ok := eventData["task_guid"].(string)
	if !ok {
		return fmt.Errorf("feishu: missing task_guid in event")
	}

	// Get the latest task data from Feishu
	task, err := f.client.GetTask(ctx, taskGUID)
	if err != nil {
		return fmt.Errorf("feishu: failed to get task for event: %w", err)
	}

	// Start transaction
	tx, err := ent.FromContext(ctx).Tx(ctx)
	if err != nil {
		return fmt.Errorf("feishu: failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	switch eventType {
	case "task.created":
		// Sync new task to ITSM
		_, err = f.SyncFeishuTaskToTicket(ctx, tx, task)
	case "task.updated":
		// Sync updated task to ITSM
		_, err = f.SyncFeishuTaskToTicket(ctx, tx, task)
	case "task.deleted":
		// Handle task deletion: mark ticket as closed or delete sync record?
		// For now, we'll just delete the sync record
		syncRecord, err := tx.FeishuTicketSync.Query().
			Where(feishuticketsync.TenantID(f.cfg.TenantID), feishuticketsync.FeishuTaskID(taskGUID)).
			Only(ctx)
		if err == nil {
			err = tx.FeishuTicketSync.DeleteOne(syncRecord).Exec(ctx)
		}
	}

	if err != nil {
		return fmt.Errorf("feishu: failed to handle task event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("feishu: failed to commit transaction: %w", err)
	}

	return nil
}

// mapPriorityToFeishu maps ITSM ticket priority to Feishu task priority
func mapPriorityToFeishu(itsmPriority string) string {
	switch strings.ToLower(itsmPriority) {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "critical":
		return "urgent"
	default:
		return "medium"
	}
}

// mapPriorityFromFeishu maps Feishu task priority to ITSM ticket priority
func mapPriorityFromFeishu(feishuPriority string) string {
	switch strings.ToLower(feishuPriority) {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "urgent":
		return "critical"
	default:
		return "medium"
	}
}

// mapStatusToFeishu maps ITSM ticket status to Feishu task status
func mapStatusToFeishu(itsmStatus string) string {
	switch strings.ToLower(itsmStatus) {
	case "open":
		return "not_started"
	case "in_progress":
		return "in_progress"
	case "resolved", "closed":
		return "completed"
	case "canceled":
		return "canceled"
	default:
		return "not_started"
	}
}

// mapStatusFromFeishu maps Feishu task status to ITSM ticket status
func mapStatusFromFeishu(feishuStatus string) string {
	switch strings.ToLower(feishuStatus) {
	case "not_started":
		return "open"
	case "in_progress":
		return "in_progress"
	case "completed":
		return "resolved"
	case "canceled":
		return "closed"
	default:
		return "open"
	}
}
