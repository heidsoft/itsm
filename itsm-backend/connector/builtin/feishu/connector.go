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

	"go.uber.org/zap"
)

// TicketCreator 工单创建接口，避免循环依赖
// 由 service.TicketService 实现，在 connector 初始化时注入
type TicketCreator interface {
	CreateTicketFromExternal(ctx context.Context, req *ExternalTicketRequest) (*ent.Ticket, error)
}

// ExternalTicketRequest 外部系统创建工单请求
// 统一了飞书、Webhook、邮件等外部来源的工单创建参数
type ExternalTicketRequest struct {
	Title       string
	Description string
	Priority    string
	Type        string
	Status      string
	RequesterID int
	TenantID    int
	Source      string // 来源标识，如 "feishu", "webhook", "email"
	ExternalID  string // 外部系统ID，如飞书任务GUID
	FormData    map[string]interface{}
}

// Feishu 飞书连接器实现
// 复用 package 内 Client 以享受 tenant_access_token 缓存
type Feishu struct {
	client        *Client
	cfg           connector.Config
	startedAt     time.Time
	ticketCreator TicketCreator // 可选：注入工单创建服务，避免直接操作数据库
}

// SetTicketCreator 注入工单创建服务
// 在 connector 初始化后、处理消息前调用
func (f *Feishu) SetTicketCreator(creator TicketCreator) {
	f.ticketCreator = creator
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
		Tags:                []string{"im", "feishu", "lark", "china"},
		Homepage:            "https://open.feishu.cn",
		IsOfficial:          true,
		RequiredPermissions: []string{"connector:write", "ticket:write", "cmdb:write"},
	}
}

func (f *Feishu) Init(_ context.Context, cfg connector.Config) error {
	appID := cfg.Credentials["app_id"]
	appSecret := cfg.Credentials["app_secret"]
	if appID == "" || appSecret == "" {
		return fmt.Errorf("feishu: credentials.app_id and app_secret are required")
	}
	baseURL, _ := connector.SettingString(cfg.Settings, "base_url")
	if baseURL == "" {
		// 海外版判定
		if region, _ := connector.SettingString(cfg.Settings, "region"); region == "intl" {
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

// GetOAuthAuthURL returns the Feishu OAuth authorization URL
func (f *Feishu) GetOAuthAuthURL(redirectURI, state string) string {
	if f.client == nil {
		return ""
	}
	return f.client.GetOAuthAuthURL(redirectURI, state)
}

func (f *Feishu) CallbackInstanceID() string {
	id, _ := connector.SettingString(f.cfg.Settings, "callbackInstanceId")
	return id
}

// ExchangeOAuthCode exchanges an authorization code for an access token
func (f *Feishu) ExchangeOAuthCode(ctx context.Context, code string) (*OAuthTokenResponse, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.ExchangeOAuthCode(ctx, code)
}

// VerifyURLToken validates the token sent by Feishu URL verification events.
func (f *Feishu) VerifyURLToken(token string) bool {
	return f.client != nil && f.client.VerifyURLToken(token)
}

func (f *Feishu) CreateTask(ctx context.Context, task *FeishuTask) (*FeishuTask, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.CreateTask(ctx, task)
}

func (f *Feishu) UpdateTask(ctx context.Context, taskGUID string, task *FeishuTask) (*FeishuTask, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.UpdateTask(ctx, taskGUID, task)
}

func (f *Feishu) GetTask(ctx context.Context, taskGUID string) (*FeishuTask, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.GetTask(ctx, taskGUID)
}

// VerifySignature 满足 connector.Receiver 接口
func (f *Feishu) VerifySignature(headers map[string]string, body []byte) error {
	if f.client == nil {
		return fmt.Errorf("feishu: not initialized")
	}
	ts := headers["X-Lark-Request-Timestamp"]
	nonce := headers["X-Lark-Request-Nonce"]
	sig := headers["X-Lark-Signature"]
	if ts == "" || nonce == "" || sig == "" {
		return fmt.Errorf("feishu: missing signature headers")
	}
	_, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("feishu: stale request timestamp")
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
// SyncTicketToFeishu syncs an ITSM ticket to Feishu as a task (creates or updates).
//
// 分相执行（合规命令纪律：数据库事务内禁止连接器/HTTP 外部调用）：
//   - phase A（短事务）：读取/抢占同步映射。mapping 唯一键 (tenant_id, ticket_id)
//     兜住并发双发送：抢占失败即重读既有记录，按其 GUID 走 create/update。
//   - phase B（无事务）：飞书 HTTP create/update task，失败同时把 phase C 的
//     状态落为 failed（可观察），重试按 mapping 处于无 GUID 状态重新走 create。
//     状态落为 failed（可观察），重试按 mapping 处于无 GUID 状态重新走 create。
//   - phase C（短事务）：落最终 feishu_task_guid/状态（成功 synced / 失败 failed）。
//
// 崩溃窗口：phase B 成功后、phase C 提交前进程崩溃，重试会重新 create 一条
// 任务（外部孤儿任务），与旧实现事务中断留下的窗口等价；飞书 create task
// 不提供幂等 token，无法进一步收敛，由映射状态可观察性兜底。
func (f *Feishu) SyncTicketToFeishu(ctx context.Context, client *ent.Client, ticket *ent.Ticket) (*FeishuTask, error) {
	if f.client == nil {
		return nil, fmt.Errorf("feishu: connector not initialized")
	}
	if client == nil {
		return nil, fmt.Errorf("feishu: ent client is required")
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

	// ---- phase A：读取/抢占同步映射（短事务，无外部调用）----
	syncRecord, err := claimTicketSyncRecord(ctx, client, ticket.TenantID, ticket.ID)
	if err != nil {
		return nil, fmt.Errorf("feishu: failed to claim sync record: %w", err)
	}

	// ---- phase B：飞书外呼（无 DB 事务）----
	var task *FeishuTask
	if syncRecord.FeishuTaskGUID != "" {
		task, err = f.client.UpdateTask(ctx, syncRecord.FeishuTaskGUID, feishuTask)
	} else {
		task, err = f.client.CreateTask(ctx, feishuTask)
	}
	if err != nil {
		// 失败落库（best-effort）：不回滚业务，状态可观察，命令重试按
		// failed + 无 GUID 重新走 create/update。
		if persistErr := persistTicketSyncOutcome(ctx, client, syncRecord, "failed", "", err); persistErr != nil {
			zap.S().Warnw("feishu: failed to persist sync failure state",
				"error", persistErr.Error(), "ticket_id", ticket.ID, "tenant_id", ticket.TenantID)
		}
		return nil, fmt.Errorf("feishu: failed to sync task: %w", err)
	}

	// ---- phase C：同步结果落库（短事务）----
	if err := persistTicketSyncOutcome(ctx, client, syncRecord, "synced", task.GUID, nil); err != nil {
		return nil, fmt.Errorf("feishu: failed to save sync record: %w", err)
	}

	return task, nil
}

// claimTicketSyncRecord 读取或抢占式创建 (tenant, ticket) 的同步映射行。
// feishu_task_id 非空必填且参与唯一索引：创建真实任务前尚无真实 ID，用
// 确定性占位符占用唯一键（tenant+ticket 唯一 → 占位符也唯一，真实 GUID
// 回填时覆盖）。占位行 sync_status=pending，phase C 落最终状态。
func claimTicketSyncRecord(ctx context.Context, client *ent.Client, tenantID, ticketID int) (*ent.FeishuTicketSync, error) {
	rec, err := client.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tenantID), feishuticketsync.TicketID(ticketID)).
		Only(ctx)
	if err == nil {
		return rec, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	created, cerr := client.FeishuTicketSync.Create().
		SetTenantID(tenantID).
		SetTicketID(ticketID).
		SetFeishuTaskID(fmt.Sprintf("pending:%d:%d", tenantID, ticketID)).
		SetSyncStatus("pending").
		Save(ctx)
	if cerr == nil {
		return created, nil
	}
	// 并发抢占失败（唯一键冲突）→ 重读竞争对手创建的行。
	rec, rerr := client.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tenantID), feishuticketsync.TicketID(ticketID)).
		Only(ctx)
	if rerr != nil {
		return nil, fmt.Errorf("create then reload failed: %v / %v", cerr, rerr)
	}
	return rec, nil
}

// persistTicketSyncOutcome 落 phase C 的最终映射状态（短事务语义，单条更新）。
func persistTicketSyncOutcome(ctx context.Context, client *ent.Client, rec *ent.FeishuTicketSync, status, guid string, syncErr error) error {
	now := time.Now()
	upd := rec.Update().
		SetSyncStatus(status).
		SetLastSyncDirection("itsm_to_feishu").
		SetLastSyncedAt(now)
	if status == "synced" && guid != "" {
		upd.SetFeishuTaskID(guid).SetFeishuTaskGUID(guid)
	}
	if syncErr != nil {
		upd.SetErrorMessage(syncErr.Error())
	} else {
		upd.ClearErrorMessage()
	}
	_, err := upd.Save(ctx)
	return err
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
		// Create new ticket via injected TicketCreator service (preferred)
		// Falls back to direct database access when TicketCreator is not injected
		if f.ticketCreator != nil {
			// 使用注入的 TicketCreator 服务创建工单
			// 该服务负责业务规则校验、审批链触发、SLA 计算等
			ticket, err = f.ticketCreator.CreateTicketFromExternal(ctx, &ExternalTicketRequest{
				Title:       feishuTask.Name,
				Description: feishuTask.Description,
				Priority:    mapPriorityFromFeishu(feishuTask.Priority),
				Type:        "ticket",
				Status:      mapStatusFromFeishu(feishuTask.Status),
				TenantID:    f.cfg.TenantID,
				Source:      "feishu",
				ExternalID:  feishuTask.GUID,
			})
		} else {
			// Fallback: direct database access when TicketCreator not injected
			// 映射飞书创建人到ITSM用户
			requesterID := 1 // 默认管理员
			if feishuTask.CreatorID != "" {
				u, qerr := tx.User.Query().
					Where(user.FeishuOpenID(feishuTask.CreatorID)).
					Where(user.TenantID(f.cfg.TenantID)).
					Only(ctx)
				if qerr == nil {
					requesterID = u.ID
				}
			}

			ticketNumber := fmt.Sprintf("TK-%d-%s", f.cfg.TenantID, time.Now().Format("20060102150405"))
			create := tx.Ticket.Create().
				SetTitle(feishuTask.Name).
				SetDescription(feishuTask.Description).
				SetPriority(mapPriorityFromFeishu(feishuTask.Priority)).
				SetType("ticket").
				SetStatus(mapStatusFromFeishu(feishuTask.Status)).
				SetTenantID(f.cfg.TenantID).
				SetRequesterID(requesterID).
				SetTicketNumber(ticketNumber)
			ticket, err = create.Save(ctx)
		}
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
	return fmt.Errorf("feishu: task event handling requires service.FeishuSyncService")
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
