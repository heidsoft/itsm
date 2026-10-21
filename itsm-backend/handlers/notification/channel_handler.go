package notification

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ChannelCatalog 是渠道诊断所需的连接器视图。
//
// 这里刻意不直接依赖 *connector.Manager：诊断接口只需要「本租户装了哪些渠道」
// 和「它们现在通不通」，用窄接口让测试不必真的拨号到 SMTP/飞书，也让未来把
// 连接器目录换成 DB 快照时不必改 handler。
type ChannelCatalog interface {
	ListByTenant(tenantID int) []connector.Config
	HealthCheckTenant(ctx context.Context, tenantID int) map[string]connector.HealthStatus
}

// ChannelHandler 暴露「通知渠道为什么发不出去」的自助诊断面。
//
// 生产环境里「我没收到通知」是最高频的工单之一，原因通常只有三类：
// 渠道没装、渠道不健康、收件人没配目标。此前这三类信息分别散在连接器日志、
// 连接器健康端点和用户表里，运维无法在一次请求里得到结论。
type ChannelHandler struct {
	client  *ent.Client
	catalog ChannelCatalog
	logger  *zap.SugaredLogger
}

func NewChannelHandler(client *ent.Client, catalog ChannelCatalog, logger *zap.SugaredLogger) *ChannelHandler {
	return &ChannelHandler{client: client, catalog: catalog, logger: logger}
}

// ChannelStatus 是单个渠道对「当前登录用户」的可投递性结论。
type ChannelStatus struct {
	Channel           string     `json:"channel"`
	Provider          string     `json:"provider,omitempty"`
	Type              string     `json:"type,omitempty"`
	Healthy           bool       `json:"healthy"`
	HealthMessage     string     `json:"healthMessage,omitempty"`
	CheckedAt         *time.Time `json:"checkedAt,omitempty"`
	LastSendSuccessAt *time.Time `json:"lastSendSuccessAt,omitempty"`
	LastSendFailureAt *time.Time `json:"lastSendFailureAt,omitempty"`
	LastSendError     string     `json:"lastSendError,omitempty"`
	RecipientTarget   string     `json:"recipientTarget,omitempty"`
	Deliverable       bool       `json:"deliverable"`
	BlockedReason     string     `json:"blockedReason,omitempty"`
}

// GetChannels 返回本租户每个通知渠道的可用性，以及当前用户能否被该渠道触达。
// @Summary 通知渠道诊断
// @Description 列出本租户已装配的通知渠道、健康状态，以及当前用户在各渠道的收件目标是否已配置
// @Tags 通知管理
// @Produce json
// @Success 200 {object} common.Response{data=object{items=[]ChannelStatus}}
// @Security BearerAuth
// @Router /api/v1/notifications/channels [get]
func (h *ChannelHandler) GetChannels(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		common.Fail(c, common.AuthFailedCode, "租户上下文缺失")
		return
	}
	userID, err := currentUserID(c)
	if err != nil {
		common.Fail(c, common.AuthFailedCode, "用户上下文缺失")
		return
	}

	recipient, err := h.client.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Only(c.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.AuthFailedCode, "当前用户不存在或已停用")
			return
		}
		common.RespondError(c, err, "查询用户失败")
		return
	}

	items := make([]ChannelStatus, 0, 8)
	// 站内信不依赖连接器，只要账号有效就能收到，永远排在第一位作为兜底渠道。
	items = append(items, ChannelStatus{
		Channel:         "in_app",
		Type:            "builtin",
		Healthy:         true,
		Deliverable:     true,
		RecipientTarget: fmt.Sprintf("user:%d", recipient.ID),
	})

	if h.catalog == nil {
		common.Success(c, gin.H{"items": items})
		return
	}

	configs := h.catalog.ListByTenant(tenantID)
	health := h.catalog.HealthCheckTenant(c.Request.Context(), tenantID)

	for _, cfg := range configs {
		key := fmt.Sprintf("%d/%s/%s", cfg.TenantID, cfg.Name, cfg.Provider)
		status := ChannelStatus{
			Channel:         cfg.Name,
			Provider:        cfg.Provider,
			Type:            string(cfg.Type),
			Healthy:         false,
			RecipientTarget: maskChannelTarget(channelTarget(recipient, cfg.Name)),
		}
		if hs, ok := health[key]; ok {
			status.Healthy = hs.OK
			status.HealthMessage = hs.Message
			checkedAt := hs.CheckedAt
			status.CheckedAt = &checkedAt
			status.LastSendSuccessAt = extraTime(hs.Extra, "lastSendSuccessAt")
			status.LastSendFailureAt = extraTime(hs.Extra, "lastSendFailureAt")
			if msg, ok := hs.Extra["lastSendError"].(string); ok {
				status.LastSendError = msg
			}
		} else {
			status.HealthMessage = "连接器未运行，健康状态未知"
		}

		switch {
		case status.RecipientTarget == "":
			status.BlockedReason = "当前用户未配置该渠道的收件目标"
		case !status.Healthy:
			status.BlockedReason = "连接器不健康"
		default:
			status.Deliverable = true
		}
		items = append(items, status)
	}

	// 稳定输出：渠道名排序，便于前端对比与截图留证。
	sort.SliceStable(items[1:], func(i, j int) bool {
		return items[1+i].Channel < items[1+j].Channel
	})

	common.Success(c, gin.H{"items": items})
}

// channelTarget 与投递链路的 notificationTarget 保持同一套映射。
// 两者若分叉，诊断会声称「能发」而投递实际找不到目标，比不诊断更糟。
func channelTarget(recipient *ent.User, channel string) string {
	switch channel {
	case "feishu":
		return recipient.FeishuOpenID
	case "dingtalk", "wecom":
		return recipient.Username
	case "email":
		return recipient.Email
	case "sms":
		return recipient.Phone
	case "webhook":
		return "tenant-default-webhook"
	default:
		return ""
	}
}

func maskChannelTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if len(target) <= 4 {
		return "****"
	}
	return target[:2] + "****" + target[len(target)-2:]
}

func extraTime(extra map[string]interface{}, key string) *time.Time {
	if extra == nil {
		return nil
	}
	switch value := extra[key].(type) {
	case time.Time:
		return &value
	case *time.Time:
		return value
	case string:
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return &parsed
		}
	}
	return nil
}
