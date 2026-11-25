package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/systemconfig"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"

	"itsm-backend/connector"
)

// 入站消息转工单的稳定错误类。handler 按 errors.Is 映射到 403 / 409 / 503，
// 禁止把所有失败压成 500 —— 操作者必须能区分「没授权」「重复事件」「未配置」。
var (
	errInboundRequesterForbidden = errors.New("feishu inbound: sender is not authorized in tenant")
	errInboundUnconfigured       = errors.New("feishu inbound: automation reporter is not configured")
	errInboundDuplicate          = errors.New("feishu inbound: duplicate message")
	errInboundInvalidPayload     = errors.New("feishu inbound: empty message text")
)

const (
	inboundModeConfigKey     = "feishu_inbound.mode"
	inboundReporterConfigKey = "feishu_inbound.automation_reporter_user_id"
	inboundModeObserveOnly   = "observe_only"
	inboundModeAutoCreate    = "auto_create"

	// 与 nonce 去重使用不同的 connector 命名空间：nonce 键是 timestamp:nonce，
	// 消息键是 provider message_id，二者语义不同，不能互相覆盖。
	inboundMessageDedupConnector = "feishu_message"
)

// CreateTicketFromInboundMessage 把一条已验签、已解密的飞书消息转成工单。
//
// 不变量：
//   - tenant 只来自已验证的 connector instance（调用方传入），绝不读消息体；
//   - 消息发送者只有映射到本租户活跃用户才可作为申请人；否则回落到本租户配置的
//     自动化账号，二者都不存在时按「未授权 / 未配置」fail closed，绝不借用他租户用户；
//   - 同一 provider message_id 只允许产生一条工单（持久去重 + 租户内唯一工单号）；
//   - 建单、审计与投递命令在同一事务提交；投递命令用幂等变体，重复不会二次副作用。
func (s *FeishuSyncService) CreateTicketFromInboundMessage(ctx context.Context, tenantID int, fc *feishuConn.Feishu, msg *connector.InboundMessage) (*dto.FeishuWebhookResponse, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("feishu sync service is not configured")
	}
	if tenantID <= 0 || msg == nil || strings.TrimSpace(msg.MessageID) == "" {
		return nil, fmt.Errorf("invalid inbound message context")
	}

	mode, reporterID, err := s.inboundRuntimeConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if mode != inboundModeAutoCreate {
		// 默认 observe_only：能力未显式开启时不得凭一条外部消息自动建单。
		return &dto.FeishuWebhookResponse{EventType: msg.Type, Action: "observed"}, nil
	}

	requesterID, err := s.resolveInboundRequester(ctx, tenantID, msg.UserID, reporterID)
	if err != nil {
		return nil, err
	}

	text := inboundMessageText(msg)
	if text == "" {
		return nil, errInboundInvalidPayload
	}

	dedup := connector.NewInboundDeduper(s.client, 0)
	seen, err := dedup.HandleBeforeProcess(ctx, tenantID, inboundMessageDedupConnector, msg.MessageID, msg.Raw)
	if err != nil {
		return nil, fmt.Errorf("register inbound message dedup: %w", err)
	}
	if seen {
		return nil, errInboundDuplicate
	}

	committed := false
	defer func() {
		if !committed {
			// 处理失败必须释放去重行，否则飞书重投会被判重复而永久丢失该消息。
			// 保留请求 context 的租户/追踪值，但解除已取消的 deadline，保证失败清理能落库。
			if forgetErr := dedup.Forget(context.WithoutCancel(ctx), tenantID, inboundMessageDedupConnector, msg.MessageID); forgetErr != nil {
				s.logger.Warnw("failed to forget feishu inbound dedup row", "tenant_id", tenantID, "err", forgetErr)
			}
		}
	}()

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("open inbound ticket tx: %w", err)
	}

	ticketNumber := deterministicInboundTicketNumber(tenantID, msg.MessageID)
	exists, err := tx.Ticket.Query().Where(
		entTicket.TenantIDEQ(tenantID), entTicket.TicketNumberEQ(ticketNumber),
	).Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("check existing inbound ticket: %w", err)
	}
	if exists {
		_ = tx.Rollback()
		return nil, errInboundDuplicate
	}

	ticket, err := tx.Ticket.Create().
		SetTenantID(tenantID).
		SetTicketNumber(ticketNumber).
		SetTitle(inboundTicketTitle(text)).
		SetDescription(text).
		SetType("ticket").
		SetRequesterID(requesterID).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("create inbound ticket: %w", err)
	}

	// 审计只写不可变标识与稳定错误类；消息正文可能含敏感信息，不进审计行。
	auditBody, _ := json.Marshal(map[string]interface{}{
		"source":       "feishu_inbound",
		"messageId":    msg.MessageID,
		"eventType":    msg.Type,
		"ticketId":     ticket.ID,
		"ticketNumber": ticketNumber,
		"actorUserId":  requesterID,
		"instanceId":   fc.CallbackInstanceID(),
	})
	if _, err := tx.AuditLog.Create().
		SetTenantID(tenantID).
		SetUserID(requesterID).
		SetResource("feishu_inbound").
		SetAction("ticket.create").
		SetPath("/api/v1/feishu/webhook/" + fc.CallbackInstanceID()).
		SetMethod("POST").
		SetStatusCode(200).
		SetRequestBody(string(auditBody)).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("write inbound ticket audit: %w", err)
	}

	// 同一个事务内落 durable command：失败即回滚建单，重放即跳过。
	_, _, err = commandbus.EnqueueTxIdempotent(ctx, tx, commandbus.EnqueueRequest{
		TenantID:       tenantID,
		CommandType:    commandbus.CommandDeliverNotification,
		AggregateType:  "ticket",
		AggregateID:    ticket.ID,
		IdempotencyKey: fmt.Sprintf("feishu-inbound-ticket:%d:%s:%d", tenantID, msg.MessageID, ticket.ID),
		MaxAttempts:    8,
		Payload: map[string]interface{}{
			"resourceType": "ticket",
			"resourceId":   ticket.ID,
			"recipientId":  requesterID,
			"channel":      "in_app",
			"type":         "ticket_created",
			"content":      "飞书消息已创建工单 " + ticketNumber,
		},
	})
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("enqueue inbound ticket notification: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit inbound ticket: %w", err)
	}
	committed = true
	if err := dedup.MarkProcessed(ctx, tenantID, inboundMessageDedupConnector, msg.MessageID, "ok"); err != nil {
		s.logger.Warnw("failed to mark inbound message processed", "tenant_id", tenantID, "err", err)
	}

	return &dto.FeishuWebhookResponse{
		EventType:    msg.Type,
		Action:       "ticket_created",
		TicketID:     ticket.ID,
		TicketNumber: ticketNumber,
	}, nil
}

func (s *FeishuSyncService) inboundRuntimeConfig(ctx context.Context, tenantID int) (string, int, error) {
	items, err := s.client.SystemConfig.Query().Where(
		systemconfig.TenantIDEQ(tenantID),
		systemconfig.KeyIn(inboundModeConfigKey, inboundReporterConfigKey),
		systemconfig.DeletedAtIsNil(),
	).All(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("query feishu inbound config: %w", err)
	}
	mode := inboundModeObserveOnly
	reporterID := 0
	for _, item := range items {
		switch item.Key {
		case inboundModeConfigKey:
			if item.Value == inboundModeAutoCreate || item.Value == inboundModeObserveOnly {
				mode = item.Value
			}
		case inboundReporterConfigKey:
			if value, parseErr := strconv.Atoi(item.Value); parseErr == nil && value > 0 {
				reporterID = value
			}
		}
	}
	return mode, reporterID, nil
}

func (s *FeishuSyncService) resolveInboundRequester(ctx context.Context, tenantID int, feishuOpenID string, reporterID int) (int, error) {
	if feishuOpenID != "" {
		u, err := s.client.User.Query().Where(
			user.TenantIDEQ(tenantID), user.FeishuOpenIDEQ(feishuOpenID), user.ActiveEQ(true),
		).Only(ctx)
		if err == nil {
			return u.ID, nil
		}
		if !ent.IsNotFound(err) {
			return 0, fmt.Errorf("resolve feishu sender: %w", err)
		}
		// 发送者在本租户无对应活跃用户：若配置了自动化账号则回落，否则视为未授权。
	}
	if reporterID > 0 {
		u, err := s.client.User.Query().Where(
			user.IDEQ(reporterID), user.TenantIDEQ(tenantID), user.ActiveEQ(true),
		).Only(ctx)
		if err == nil {
			return u.ID, nil
		}
		if !ent.IsNotFound(err) {
			return 0, fmt.Errorf("resolve automation reporter: %w", err)
		}
		return 0, errInboundRequesterForbidden
	}
	if feishuOpenID != "" {
		return 0, errInboundRequesterForbidden
	}
	return 0, errInboundUnconfigured
}

func inboundMessageText(msg *connector.InboundMessage) string {
	if text := strings.TrimSpace(msg.Content); text != "" {
		return text
	}
	event, ok := msg.Extras["event"].(map[string]interface{})
	if !ok {
		return ""
	}
	message, ok := event["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, ok := message["content"].(string)
	if !ok {
		return ""
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Text)
}

func inboundTicketTitle(text string) string {
	firstLine := strings.SplitN(text, "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)
	if len(firstLine) > 200 {
		return firstLine[:200]
	}
	if firstLine == "" {
		return "飞书消息工单"
	}
	return firstLine
}

func deterministicInboundTicketNumber(tenantID int, messageID string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(messageID))
	return fmt.Sprintf("TKT-FS-%d-%d", tenantID, h.Sum32())
}
