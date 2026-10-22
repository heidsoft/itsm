package notification

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/predicate"
	"itsm-backend/internal/commandbus"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeliveryHandler 暴露通知投递审计的读面与人工重试入口。
//
// 背景：notification_deliveries 表由 commandbus 的投递处理器写入（成功/失败/重试次数
// 全部落库），但此前没有任何 HTTP 端点可读。结果是「用户说没收到通知」时，运维只能翻
// 日志或直接查库。本 handler 只做读 + 重试入队，不改变投递主链路的语义：
//   - 读：按租户隔离，支持 channel / status / 接收人 / 工单 / 时间窗过滤；
//   - 统计：按 channel × status 聚合，给出成功率，供仪表盘与告警使用；
//   - 重试：把原命令的 payload 以新幂等键重新入队，旧投递记录保留为历史（失败可追溯），
//     新命令由 worker 执行后写入一条新的投递记录。
type DeliveryHandler struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewDeliveryHandler 创建投递审计 handler。client 为 nil 时所有端点返回 503，
// 便于在未启用命令总线的部署里安全降级。
func NewDeliveryHandler(client *ent.Client, logger *zap.SugaredLogger) *DeliveryHandler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &DeliveryHandler{client: client, logger: logger}
}

// DeliveryItem 是投递记录的对外投影。
// 不返回 target 明文，只返回 targetMasked（写入时已脱敏）。
type DeliveryItem struct {
	ID                   int        `json:"id"`
	OperationalCommandID int        `json:"operationalCommandId"`
	RecipientID          int        `json:"recipientId"`
	Channel              string     `json:"channel"`
	TargetMasked         string     `json:"targetMasked"`
	Status               string     `json:"status"`
	Attempt              int        `json:"attempt"`
	TicketID             *int       `json:"ticketId,omitempty"`
	TicketNotificationID *int       `json:"ticketNotificationId,omitempty"`
	ProviderMessageID    string     `json:"providerMessageId,omitempty"`
	ErrorCode            string     `json:"errorCode,omitempty"`
	ErrorMessage         string     `json:"errorMessage,omitempty"`
	SentAt               *time.Time `json:"sentAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

// ChannelDeliveryStat 是按渠道聚合的投递质量。
type ChannelDeliveryStat struct {
	Channel     string  `json:"channel"`
	Total       int     `json:"total"`
	Sent        int     `json:"sent"`
	Failed      int     `json:"failed"`
	Pending     int     `json:"pending"`
	SuccessRate float64 `json:"successRate"`
}

// DeliveryStatsResponse 是投递统计的响应体。
type DeliveryStatsResponse struct {
	From      time.Time             `json:"from"`
	To        time.Time             `json:"to"`
	Total     int                   `json:"total"`
	ByStatus  map[string]int        `json:"byStatus"`
	ByChannel []ChannelDeliveryStat `json:"byChannel"`
}

// ListDeliveries 查询投递记录。
//
// @Summary 查询通知投递记录
// @Tags 通知管理
// @Param channel query string false "渠道（email/in_app/feishu/wecom/dingtalk/sms/webhook）"
// @Param status query string false "状态（pending/sent/failed）"
// @Param recipientId query int false "接收人ID"
// @Param ticketId query int false "工单ID"
// @Param from query string false "起始时间（YYYY-MM-DD 或 RFC3339）"
// @Param to query string false "结束时间（YYYY-MM-DD 或 RFC3339，含当日）"
// @Success 200 {object} common.Response
// @Router /api/v1/notification-deliveries [get]
func (h *DeliveryHandler) ListDeliveries(c *gin.Context) {
	tenantID, ok := h.requireTenant(c)
	if !ok {
		return
	}
	pg := common.GetPaginationFromQuery(c)
	predicates, err := deliveryPredicates(c, tenantID)
	if err != nil {
		common.ParamErrorWithErr(c, err, "时间或过滤参数格式错误")
		return
	}

	base := h.client.NotificationDelivery.Query().Where(predicates...)
	total, err := base.Clone().Count(c.Request.Context())
	if err != nil {
		h.logger.Errorw("count notification deliveries", "tenant_id", tenantID, "error", err)
		common.InternalError(c, "查询投递记录失败")
		return
	}
	rows, err := base.
		Order(ent.Desc(notificationdelivery.FieldCreatedAt)).
		Offset((pg.Page - 1) * pg.PageSize).
		Limit(pg.PageSize).
		All(c.Request.Context())
	if err != nil {
		h.logger.Errorw("list notification deliveries", "tenant_id", tenantID, "error", err)
		common.InternalError(c, "查询投递记录失败")
		return
	}

	items := make([]DeliveryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDeliveryItem(row))
	}
	common.SuccessWithList(c, items, total, pg.Page, pg.PageSize)
}

// GetDeliveryStats 按渠道聚合投递质量。
//
// @Summary 通知投递统计
// @Tags 通知管理
// @Param days query int false "统计窗口天数（默认 7，上限 90）"
// @Success 200 {object} common.Response
// @Router /api/v1/notification-deliveries/stats [get]
func (h *DeliveryHandler) GetDeliveryStats(c *gin.Context) {
	tenantID, ok := h.requireTenant(c)
	if !ok {
		return
	}
	days := 7
	if raw := strings.TrimSpace(c.Query("days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			common.ParamError(c, "days 必须是正整数")
			return
		}
		if parsed > 90 {
			parsed = 90
		}
		days = parsed
	}

	to := time.Now()
	from := to.AddDate(0, 0, -days)
	var rows []struct {
		Channel string `json:"channel"`
		Status  string `json:"status"`
		Count   int    `json:"count"`
	}
	err := h.client.NotificationDelivery.Query().
		Where(
			notificationdelivery.TenantIDEQ(tenantID),
			notificationdelivery.CreatedAtGTE(from),
		).
		GroupBy(notificationdelivery.FieldChannel, notificationdelivery.FieldStatus).
		Aggregate(ent.Count()).
		Scan(c.Request.Context(), &rows)
	if err != nil {
		h.logger.Errorw("aggregate notification deliveries", "tenant_id", tenantID, "error", err)
		common.InternalError(c, "统计投递数据失败")
		return
	}

	response := DeliveryStatsResponse{
		From: from, To: to, Total: 0,
		ByStatus:  map[string]int{},
		ByChannel: []ChannelDeliveryStat{},
	}
	index := map[string]int{}
	for _, row := range rows {
		response.Total += row.Count
		response.ByStatus[row.Status] += row.Count
		position, exists := index[row.Channel]
		if !exists {
			position = len(response.ByChannel)
			index[row.Channel] = position
			response.ByChannel = append(response.ByChannel, ChannelDeliveryStat{Channel: row.Channel})
		}
		stat := &response.ByChannel[position]
		stat.Total += row.Count
		switch row.Status {
		case "sent":
			stat.Sent += row.Count
		case "failed":
			stat.Failed += row.Count
		default:
			stat.Pending += row.Count
		}
	}
	for i := range response.ByChannel {
		stat := &response.ByChannel[i]
		// 成功率只按「已定论」的投递计算（sent / failed），pending 不计入分母，
		// 否则队列积压会把成功率压低，误导排障方向。
		settled := stat.Sent + stat.Failed
		if settled > 0 {
			stat.SuccessRate = float64(stat.Sent) / float64(settled)
		}
	}
	common.Success(c, response)
}

// RetryDelivery 重新入队一条失败的投递。
//
// 语义：旧投递记录保持不变（保留失败证据），以原命令 payload 生成新命令，
// 新命令由 commandbus worker 执行后写入一条新的投递记录。因此同一条通知
// 可能出现多条投递记录，这是刻意的审计设计，不是重复写入。
//
// @Summary 重试通知投递
// @Tags 通知管理
// @Param id path int true "投递记录ID"
// @Success 200 {object} common.Response
// @Router /api/v1/notification-deliveries/{id}/retry [post]
func (h *DeliveryHandler) RetryDelivery(c *gin.Context) {
	tenantID, ok := h.requireTenant(c)
	if !ok {
		return
	}
	deliveryID, err := strconv.Atoi(c.Param("id"))
	if err != nil || deliveryID <= 0 {
		common.ParamError(c, "投递记录ID格式错误")
		return
	}
	ctx := c.Request.Context()
	delivery, err := h.client.NotificationDelivery.Query().
		Where(notificationdelivery.IDEQ(deliveryID), notificationdelivery.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(c, "投递记录不存在")
			return
		}
		common.InternalError(c, "读取投递记录失败")
		return
	}
	switch delivery.Status {
	case "failed":
		// 只有失败记录允许重试。
	case "sent":
		common.Conflict(c, "该通知已投递成功，无需重试", nil)
		return
	default:
		common.Conflict(c, "该通知仍在投递队列中，请等待结果", nil)
		return
	}

	command, err := h.client.OperationalCommand.Get(ctx, delivery.OperationalCommandID)
	if err != nil || command.TenantID != tenantID {
		common.NotFound(c, "原投递命令不存在或不属于当前租户")
		return
	}

	// 幂等守卫：同一条失败投递被反复点「重试」会生成多条新命令，每条都真实发送一次。
	// 运维排障时连点两下，收件人就收到两条一模一样的通知——而旧投递记录永远是 failed
	// （刻意保留失败证据），界面上「重试」按钮永远可点，天然诱导重复点击。
	// 重试命令的幂等键是 `<原键>:retry:<纳秒>`，据此判断是否已有在途/已处理的重试。
	if blocked, message := h.existingRetry(c.Request.Context(), tenantID, command.IdempotencyKey); blocked {
		common.Conflict(c, message, nil)
		return
	}

	payload := make(map[string]interface{}, len(command.Payload))
	for key, value := range command.Payload {
		payload[key] = value
	}
	retried, err := commandbus.Enqueue(ctx, h.client, commandbus.EnqueueRequest{
		TenantID:      tenantID,
		CommandType:   command.CommandType,
		AggregateType: command.AggregateType,
		AggregateID:   command.AggregateID,
		// 幂等键必须变化，否则会命中唯一索引 (tenant, type, idempotency_key)，
		// 新命令无法入队；旧记录已失败，不存在重复投递风险。
		IdempotencyKey: fmt.Sprintf("%s:retry:%d", command.IdempotencyKey, time.Now().UnixNano()),
		Payload:        payload,
		MaxAttempts:    command.MaxAttempts,
	})
	if err != nil {
		h.logger.Errorw("requeue notification delivery", "delivery_id", deliveryID,
			"tenant_id", tenantID, "error", err)
		common.InternalError(c, "重试入队失败")
		return
	}

	h.logger.Infow("notification delivery requeued by operator",
		"delivery_id", deliveryID, "tenant_id", tenantID,
		"old_command_id", command.ID, "new_command_id", retried.ID,
		"operator_id", c.GetInt("user_id"), "channel", delivery.Channel)

	common.Success(c, gin.H{
		"deliveryId": deliveryID,
		"commandId":  retried.ID,
		"status":     retried.Status,
		"retriedAt":  time.Now(),
	})
}

// existingRetry 判断该投递是否已经有过「还没定论」或「已经处理完」的人工重试。
//
// 重试命令以 `<原幂等键>:retry:<纳秒>` 入箱（见 RetryDelivery），因此用前缀关系
// 就能把它和原始命令、以及原始命令派生出的其它命令区分开。判定语义：
//   - pending / processing：已有在途重试，再点一次只会再多发一条 → 拒绝；
//   - succeeded：上一次重试已经被 worker 执行过（发送成功，或因用户偏好跳过），
//     再点一次必然重复投递 → 拒绝；
//   - dead_letter / cancelled：上一次确实没送出去，允许再试。
//
// 为什么不用状态字段另存标记：delivery 记录刻意保持 failed 不动（失败证据可追溯），
// 新增「已重试」状态会污染投递状态机；而命令表本身已完整记录重试谱系。
func (h *DeliveryHandler) existingRetry(ctx context.Context, tenantID int, originalKey string) (bool, string) {
	prefix := originalKey + ":retry:"
	attempts, err := h.client.OperationalCommand.Query().
		Where(
			operationalcommand.TenantIDEQ(tenantID),
			operationalcommand.IdempotencyKeyHasPrefix(prefix),
			operationalcommand.StatusIn(
				commandbus.StatusPending, commandbus.StatusProcessing, commandbus.StatusSucceeded,
			),
		).
		Order(ent.Desc(operationalcommand.FieldID)).
		All(ctx)
	if err != nil {
		// 查不出既有重试就不放行：宁可让运维重试失败，也不重复给用户发通知。
		h.logger.Errorw("query existing notification delivery retries", "tenant_id", tenantID, "error", err)
		return true, "无法确认该投递是否已有重试，为避免重复发送已拒绝本次操作，请稍后重试"
	}
	for _, attempt := range attempts {
		// 前缀谓词走的是 SQL LIKE，幂等键里的 `_` 是通配符，必须再用精确前缀复核一次，
		// 否则 `a_b:retry:` 会命中 `axb:retry:`，把无关投递判成「已在途」。
		if !strings.HasPrefix(attempt.IdempotencyKey, prefix) {
			continue
		}
		switch attempt.Status {
		case commandbus.StatusPending, commandbus.StatusProcessing:
			return true, fmt.Sprintf(
				"该投递已有在途重试（命令 #%d，状态 %s），请等待结果后再决定是否需要再试",
				attempt.ID, attempt.Status)
		case commandbus.StatusSucceeded:
			return true, fmt.Sprintf(
				"该投递已重试成功（命令 #%d），无需再次重试；如仍未收到，请检查收件目标是否变更",
				attempt.ID)
		}
	}
	return false, ""
}

func (h *DeliveryHandler) requireTenant(c *gin.Context) (int, bool) {
	if h.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code": 503, "message": "通知投递审计未启用（命令总线未装配）",
		})
		return 0, false
	}
	tenantID, err := currentTenantID(c)
	if err != nil {
		common.ParamError(c, "租户信息错误")
		return 0, false
	}
	return tenantID, true
}

func deliveryPredicates(c *gin.Context, tenantID int) ([]predicate.NotificationDelivery, error) {
	predicates := []predicate.NotificationDelivery{
		notificationdelivery.TenantIDEQ(tenantID),
	}
	if channel := strings.TrimSpace(c.Query("channel")); channel != "" {
		predicates = append(predicates, notificationdelivery.ChannelEQ(channel))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		predicates = append(predicates, notificationdelivery.StatusEQ(status))
	}
	if raw := strings.TrimSpace(c.Query("recipientId")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("recipientId 无效")
		}
		predicates = append(predicates, notificationdelivery.RecipientIDEQ(id))
	}
	if raw := strings.TrimSpace(c.Query("ticketId")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("ticketId 无效")
		}
		predicates = append(predicates, notificationdelivery.TicketIDEQ(id))
	}
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		from, err := parseDeliveryTime(raw, false)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, notificationdelivery.CreatedAtGTE(from))
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		to, err := parseDeliveryTime(raw, true)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, notificationdelivery.CreatedAtLT(to))
	}
	return predicates, nil
}

// parseDeliveryTime 接受 YYYY-MM-DD（按服务器本地时区）或 RFC3339。
// endOfDay 为 true 时，纯日期输入返回次日零点（用于 < 比较，含当日）。
func parseDeliveryTime(raw string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed, nil
	}
	date, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("时间格式错误: %s", raw)
	}
	if endOfDay {
		return date.AddDate(0, 0, 1), nil
	}
	return date, nil
}

func toDeliveryItem(row *ent.NotificationDelivery) DeliveryItem {
	item := DeliveryItem{
		ID:                   row.ID,
		OperationalCommandID: row.OperationalCommandID,
		RecipientID:          row.RecipientID,
		Channel:              row.Channel,
		TargetMasked:         row.TargetMasked,
		Status:               row.Status,
		Attempt:              row.Attempt,
		TicketID:             row.TicketID,
		TicketNotificationID: row.TicketNotificationID,
		ProviderMessageID:    row.ProviderMessageID,
		ErrorCode:            row.ErrorCode,
		ErrorMessage:         row.ErrorMessage,
		SentAt:               row.SentAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	return item
}
