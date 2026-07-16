package service

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/user"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type IncidentAlertingService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewIncidentAlertingService(client *ent.Client, logger *zap.SugaredLogger) *IncidentAlertingService {
	return &IncidentAlertingService{
		client: client,
		logger: logger,
	}
}

// AlertChannel 告警渠道接口
type AlertChannel interface {
	Send(ctx context.Context, alert *dto.IncidentAlertResponse) error
	GetName() string
	IsEnabled() bool
}

// EmailChannel 邮件告警渠道
type EmailChannel struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromEmail    string
	logger       *zap.SugaredLogger
}

func (c *EmailChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
	c.logger.Infow("Sending email alert", "alert_id", alert.ID, "recipients", alert.Recipients)

	if len(alert.Recipients) == 0 {
		return nil
	}

	// 真实邮件发送逻辑
	auth := smtp.PlainAuth("", c.smtpUsername, c.smtpPassword, c.smtpHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: [ITSM Alert] %s\r\n"+
		"\r\n"+
		"%s\r\n", alert.Recipients[0], alert.AlertName, alert.Message))

	addr := fmt.Sprintf("%s:%d", c.smtpHost, c.smtpPort)

	// 在非测试/开发环境中尝试发送
	if os.Getenv("GIN_MODE") == "release" || os.Getenv("ENABLE_EMAIL_SENDING") == "true" {
		if err := smtp.SendMail(addr, auth, c.fromEmail, alert.Recipients, msg); err != nil {
			c.logger.Errorw("Failed to send email via SMTP", "error", err)
			return fmt.Errorf("failed to send email via SMTP: %w", err)
		} else {
			c.logger.Infow("Email sent via SMTP successfully")
		}
	} else {
		// 模拟邮件发送
		time.Sleep(100 * time.Millisecond)
		c.logger.Infow("Email sending simulated (dev mode)")
	}

	c.logger.Infow("Email alert processing completed", "alert_id", alert.ID)
	return nil
}

func (c *EmailChannel) GetName() string {
	return "email"
}

func (c *EmailChannel) IsEnabled() bool {
	return c.smtpHost != "" && c.smtpPort > 0
}

// SMSChannel 短信告警渠道
type SMSChannel struct {
	apiKey    string
	apiSecret string
	signName  string
	logger    *zap.SugaredLogger
}

func (c *SMSChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
	c.logger.Infow("Sending SMS alert", "alert_id", alert.ID, "recipients", alert.Recipients)
	if os.Getenv("GIN_MODE") == "release" {
		return fmt.Errorf("SMS alert provider is not implemented; use a managed connector")
	}
	time.Sleep(200 * time.Millisecond)
	c.logger.Infow("SMS alert simulated in non-production environment", "alert_id", alert.ID)
	return nil
}

func (c *SMSChannel) GetName() string {
	return "sms"
}

func (c *SMSChannel) IsEnabled() bool {
	return c.apiKey != "" && c.apiSecret != ""
}

// SlackChannel Slack告警渠道
type SlackChannel struct {
	webhookURL string
	channel    string
	logger     *zap.SugaredLogger
}

func (c *SlackChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
	c.logger.Infow("Sending Slack alert", "alert_id", alert.ID, "channel", c.channel)
	if os.Getenv("GIN_MODE") == "release" {
		return fmt.Errorf("Slack alert delivery must use the connector lifecycle")
	}
	time.Sleep(150 * time.Millisecond)
	c.logger.Infow("Slack alert simulated in non-production environment", "alert_id", alert.ID)
	return nil
}

func (c *SlackChannel) GetName() string {
	return "slack"
}

func (c *SlackChannel) IsEnabled() bool {
	return c.webhookURL != ""
}

// WebhookChannel Webhook告警渠道
type WebhookChannel struct {
	url     string
	method  string
	headers map[string]string
	logger  *zap.SugaredLogger
}

func (c *WebhookChannel) Send(ctx context.Context, alert *dto.IncidentAlertResponse) error {
	c.logger.Infow("Sending webhook alert", "alert_id", alert.ID, "url", c.url)
	if os.Getenv("GIN_MODE") == "release" {
		return fmt.Errorf("webhook alert delivery must use the connector lifecycle")
	}
	time.Sleep(100 * time.Millisecond)
	c.logger.Infow("Webhook alert simulated in non-production environment", "alert_id", alert.ID)
	return nil
}

func (c *WebhookChannel) GetName() string {
	return "webhook"
}

func (c *WebhookChannel) IsEnabled() bool {
	return c.url != ""
}

// CreateIncidentAlert 创建事件告警
func (s *IncidentAlertingService) CreateIncidentAlert(ctx context.Context, req *dto.CreateIncidentAlertRequest, tenantID int) (*dto.IncidentAlertResponse, error) {
	s.logger.Infow("Creating incident alert", "incident_id", req.IncidentID, "type", req.AlertType)
	if err := s.validateAlertRequest(ctx, req, tenantID); err != nil {
		return nil, err
	}
	triggeredAt := time.Now()
	if req.TriggeredAt != nil {
		triggeredAt = *req.TriggeredAt
	}

	// 创建告警记录
	alert, err := s.client.IncidentAlert.Create().
		SetIncidentID(req.IncidentID).
		SetAlertType(req.AlertType).
		SetAlertName(req.AlertName).
		SetMessage(req.Message).
		SetSeverity(req.Severity).
		SetStatus("active").
		SetChannels(req.Channels).
		SetRecipients(req.Recipients).
		SetTriggeredAt(triggeredAt).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident alert", "error", err)
		return nil, fmt.Errorf("failed to create incident alert: %w", err)
	}

	// 发送告警通知
	go s.sendAlertNotifications(context.Background(), alert, tenantID)

	s.logger.Infow("Incident alert created successfully", "id", alert.ID)
	return s.toIncidentAlertResponse(alert), nil
}

func (s *IncidentAlertingService) validateAlertRequest(ctx context.Context, req *dto.CreateIncidentAlertRequest, tenantID int) error {
	if req.IncidentID <= 0 {
		return fmt.Errorf("incident id is required")
	}
	exists, err := s.client.Incident.Query().
		Where(incident.IDEQ(req.IncidentID), incident.TenantIDEQ(tenantID), incident.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate incident: %w", err)
	}
	if !exists {
		return fmt.Errorf("incident not found")
	}
	if strings.TrimSpace(req.AlertType) == "" || strings.TrimSpace(req.AlertName) == "" || strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("alert type, name, and message are required")
	}
	switch req.Severity {
	case "", "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("invalid alert severity: %s", req.Severity)
	}
	allowedChannels := map[string]struct{}{"email": {}, "sms": {}, "slack": {}, "webhook": {}, "in_app": {}}
	for _, channel := range req.Channels {
		if _, ok := allowedChannels[channel]; !ok {
			return fmt.Errorf("unsupported alert channel: %s", channel)
		}
	}
	return nil
}

// sendAlertNotifications 发送告警通知
func (s *IncidentAlertingService) sendAlertNotifications(ctx context.Context, alert *ent.IncidentAlert, tenantID int) {
	s.logger.Infow("Sending alert notifications", "alert_id", alert.ID)

	// 获取告警渠道
	channels := s.getAlertChannels(alert.Channels)

	// 转换为响应格式
	alertResponse := s.toIncidentAlertResponse(alert)

	// 发送到各个渠道
	for _, channel := range channels {
		if !channel.IsEnabled() {
			s.logger.Warnw("Alert channel is disabled", "channel", channel.GetName())
			continue
		}

		err := channel.Send(ctx, alertResponse)
		if err != nil {
			s.logger.Errorw("Failed to send alert notification",
				"error", err,
				"channel", channel.GetName(),
				"alert_id", alert.ID)
		}
	}

	// 创建系统通知记录
	s.createSystemNotification(ctx, alert, tenantID)
}

// getAlertChannels 获取告警渠道
func (s *IncidentAlertingService) getAlertChannels(channelNames []string) []AlertChannel {
	var channels []AlertChannel

	for _, channelName := range channelNames {
		switch channelName {
		case "email":
			smtpHost := viper.GetString("alerting.smtp.host")
			smtpPort := viper.GetInt("alerting.smtp.port")
			smtpUsername := viper.GetString("alerting.smtp.username")
			smtpPassword := viper.GetString("alerting.smtp.password")
			fromEmail := viper.GetString("alerting.smtp.from_email")

			// 从环境变量覆盖配置
			if host := os.Getenv("SMTP_HOST"); host != "" {
				smtpHost = host
			}
			if port := os.Getenv("SMTP_PORT"); port != "" {
				fmt.Sscanf(port, "%d", &smtpPort)
			}
			if username := os.Getenv("SMTP_USERNAME"); username != "" {
				smtpUsername = username
			}
			if password := os.Getenv("SMTP_PASSWORD"); password != "" {
				smtpPassword = password
			}
			if from := os.Getenv("SMTP_FROM_EMAIL"); from != "" {
				fromEmail = from
			}

			channels = append(channels, &EmailChannel{
				smtpHost:     smtpHost,
				smtpPort:     smtpPort,
				smtpUsername: smtpUsername,
				smtpPassword: smtpPassword,
				fromEmail:    fromEmail,
				logger:       s.logger,
			})
		case "sms":
			apiKey := viper.GetString("alerting.sms.api_key")
			apiSecret := viper.GetString("alerting.sms.api_secret")
			signName := viper.GetString("alerting.sms.sign_name")

			// 从环境变量覆盖配置
			if key := os.Getenv("SMS_API_KEY"); key != "" {
				apiKey = key
			}
			if secret := os.Getenv("SMS_API_SECRET"); secret != "" {
				apiSecret = secret
			}
			if sign := os.Getenv("SMS_SIGN_NAME"); sign != "" {
				signName = sign
			}

			channels = append(channels, &SMSChannel{
				apiKey:    apiKey,
				apiSecret: apiSecret,
				signName:  signName,
				logger:    s.logger,
			})
		case "slack":
			webhookURL := viper.GetString("alerting.slack.webhook_url")
			channel := viper.GetString("alerting.slack.channel")

			// 从环境变量覆盖配置
			if url := os.Getenv("SLACK_WEBHOOK_URL"); url != "" {
				webhookURL = url
			}
			if ch := os.Getenv("SLACK_CHANNEL"); ch != "" {
				channel = ch
			}

			channels = append(channels, &SlackChannel{
				webhookURL: webhookURL,
				channel:    channel,
				logger:     s.logger,
			})
		case "webhook":
			url := viper.GetString("alerting.webhook.url")
			method := viper.GetString("alerting.webhook.method")

			// 从环境变量覆盖配置
			if u := os.Getenv("WEBHOOK_URL"); u != "" {
				url = u
			}
			if m := os.Getenv("WEBHOOK_METHOD"); m != "" {
				method = m
			}

			// Authorization header from env
			authHeader := os.Getenv("WEBHOOK_AUTH_HEADER")

			headers := map[string]string{
				"Content-Type": "application/json",
			}
			if authHeader != "" {
				headers["Authorization"] = authHeader
			}

			channels = append(channels, &WebhookChannel{
				url:     url,
				method:  method,
				headers: headers,
				logger:  s.logger,
			})
		}
	}

	return channels
}

// createSystemNotification 创建系统通知记录
func (s *IncidentAlertingService) createSystemNotification(ctx context.Context, alert *ent.IncidentAlert, tenantID int) {
	recipients, err := s.client.User.Query().
		Where(
			user.TenantIDEQ(tenantID),
			user.ActiveEQ(true),
			user.RoleIn(user.RoleAdmin, user.RoleSuperAdmin),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to resolve in-app alert recipients", "error", err)
		return
	}
	for _, recipient := range recipients {
		_, err := s.client.Notification.Create().
			SetTitle(alert.AlertName).
			SetMessage(alert.Message).
			SetType("incident_alert").
			SetUserID(recipient.ID).
			SetTenantID(tenantID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to create system notification", "error", err, "user_id", recipient.ID)
		}
	}
}

// AcknowledgeAlert 确认告警
func (s *IncidentAlertingService) AcknowledgeAlert(ctx context.Context, alertID int, userID int, tenantID int) error {
	s.logger.Infow("Acknowledging alert", "alert_id", alertID, "user_id", userID)
	if err := s.validateAlertActor(ctx, userID, tenantID); err != nil {
		return err
	}

	now := time.Now()
	alert, err := s.client.IncidentAlert.UpdateOneID(alertID).
		Where(incidentalert.TenantIDEQ(tenantID), incidentalert.StatusEQ("active")).
		SetStatus("acknowledged").
		SetAcknowledgedAt(now).
		SetAcknowledgedBy(userID).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("alert not found")
		}
		s.logger.Errorw("Failed to acknowledge alert", "error", err)
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	// 记录确认活动
	s.createAlertEvent(ctx, alert, "acknowledged", fmt.Sprintf("告警已被用户 %d 确认", userID), userID, tenantID)

	s.logger.Infow("Alert acknowledged successfully", "alert_id", alertID)
	return nil
}

// ResolveAlert 解决告警
func (s *IncidentAlertingService) ResolveAlert(ctx context.Context, alertID int, userID int, tenantID int) error {
	s.logger.Infow("Resolving alert", "alert_id", alertID, "user_id", userID)
	if err := s.validateAlertActor(ctx, userID, tenantID); err != nil {
		return err
	}

	now := time.Now()
	alert, err := s.client.IncidentAlert.UpdateOneID(alertID).
		Where(incidentalert.TenantIDEQ(tenantID), incidentalert.StatusIn("active", "acknowledged")).
		SetStatus("resolved").
		SetResolvedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("alert not found")
		}
		s.logger.Errorw("Failed to resolve alert", "error", err)
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	// 记录解决活动
	s.createAlertEvent(ctx, alert, "resolved", fmt.Sprintf("告警已被用户 %d 解决", userID), userID, tenantID)

	s.logger.Infow("Alert resolved successfully", "alert_id", alertID)
	return nil
}

// createAlertEvent 创建告警活动记录
func (s *IncidentAlertingService) createAlertEvent(ctx context.Context, alert *ent.IncidentAlert, eventType, description string, userID, tenantID int) {
	_, err := s.client.IncidentEvent.Create().
		SetIncidentID(alert.IncidentID).
		SetEventType("alert_" + eventType).
		SetEventName("告警" + eventType).
		SetDescription(description).
		SetStatus("active").
		SetSeverity(alert.Severity).
		SetSource("user").
		SetUserID(userID).
		SetOccurredAt(time.Now()).
		SetTenantID(tenantID).
		SetData(map[string]interface{}{"alertId": alert.ID}).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create alert audit event", "error", err, "alert_id", alert.ID)
	}
}

func (s *IncidentAlertingService) validateAlertActor(ctx context.Context, userID, tenantID int) error {
	exists, err := s.client.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate alert actor: %w", err)
	}
	if !exists {
		return fmt.Errorf("alert actor not found or inactive")
	}
	return nil
}

// GetActiveAlerts 获取活跃告警
func (s *IncidentAlertingService) GetActiveAlerts(ctx context.Context, tenantID int, page, size int) ([]*dto.IncidentAlertResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	query := s.client.IncidentAlert.Query().
		Where(
			incidentalert.TenantIDEQ(tenantID),
			incidentalert.StatusEQ("active"),
		)

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count active alerts", "error", err)
		return nil, 0, fmt.Errorf("failed to count active alerts: %w", err)
	}

	// 分页查询
	alerts, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(incidentalert.FieldTriggeredAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get active alerts", "error", err)
		return nil, 0, fmt.Errorf("failed to get active alerts: %w", err)
	}

	responses := make([]*dto.IncidentAlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = s.toIncidentAlertResponse(alert)
	}

	return responses, total, nil
}

// GetAlertStatistics 获取告警统计
func (s *IncidentAlertingService) GetAlertStatistics(ctx context.Context, tenantID int, startTime, endTime time.Time) (map[string]interface{}, error) {
	s.logger.Infow("Getting alert statistics", "tenant_id", tenantID, "start_time", startTime, "end_time", endTime)
	if endTime.Before(startTime) {
		return nil, fmt.Errorf("end time must not be before start time")
	}

	// 获取告警总数
	totalAlerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.TenantIDEQ(tenantID),
			incidentalert.TriggeredAtGTE(startTime),
			incidentalert.TriggeredAtLTE(endTime),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total alerts: %w", err)
	}

	// 获取活跃告警数
	activeAlerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.TenantIDEQ(tenantID),
			incidentalert.StatusEQ("active"),
			incidentalert.TriggeredAtGTE(startTime),
			incidentalert.TriggeredAtLTE(endTime),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active alerts: %w", err)
	}

	// 获取已确认告警数
	acknowledgedAlerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.TenantIDEQ(tenantID),
			incidentalert.StatusEQ("acknowledged"),
			incidentalert.TriggeredAtGTE(startTime),
			incidentalert.TriggeredAtLTE(endTime),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count acknowledged alerts: %w", err)
	}

	// 获取已解决告警数
	resolvedAlerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.TenantIDEQ(tenantID),
			incidentalert.StatusEQ("resolved"),
			incidentalert.TriggeredAtGTE(startTime),
			incidentalert.TriggeredAtLTE(endTime),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count resolved alerts: %w", err)
	}

	// 计算平均响应时间
	var avgResponseTime float64
	if acknowledgedAlerts > 0 {
		// 计算实际平均响应时间 (Minutes)
		// 聚合查询: AVG(EXTRACT(EPOCH FROM (acknowledged_at - triggered_at))/60)
		// 由于Ent聚合查询较复杂，这里使用简化逻辑：查询最近100条已确认告警计算平均值
		recentAlerts, err := s.client.IncidentAlert.Query().
			Where(
				incidentalert.TenantIDEQ(tenantID),
				incidentalert.TriggeredAtGTE(startTime),
				incidentalert.TriggeredAtLTE(endTime),
				incidentalert.AcknowledgedAtNotNil(),
			).
			Limit(100).
			Select(incidentalert.FieldTriggeredAt, incidentalert.FieldAcknowledgedAt).
			All(ctx)

		if err == nil && len(recentAlerts) > 0 {
			var totalDiff float64
			count := 0
			for _, a := range recentAlerts {
				if a.AcknowledgedAt.After(a.TriggeredAt) {
					totalDiff += a.AcknowledgedAt.Sub(a.TriggeredAt).Minutes()
					count++
				}
			}
			if count > 0 {
				avgResponseTime = totalDiff / float64(count)
			}
		} else {
			// Fallback if query fails or no data
			avgResponseTime = 0
		}
	}

	// 计算解决率
	var resolutionRate float64
	if totalAlerts > 0 {
		resolutionRate = float64(resolvedAlerts) / float64(totalAlerts) * 100
	}

	statistics := map[string]interface{}{
		"total_alerts":        totalAlerts,
		"active_alerts":       activeAlerts,
		"acknowledged_alerts": acknowledgedAlerts,
		"resolved_alerts":     resolvedAlerts,
		"avg_response_time":   avgResponseTime,
		"resolution_rate":     resolutionRate,
		"period": map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
		},
	}

	return statistics, nil
}

// ProcessEscalationAlerts 处理升级告警
func (s *IncidentAlertingService) ProcessEscalationAlerts(ctx context.Context, incidentID int, escalationLevel int, tenantID int) error {
	s.logger.Infow("Processing escalation alerts", "incident_id", incidentID, "level", escalationLevel)

	// 获取事件信息
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("incident not found")
		}
		return fmt.Errorf("failed to get incident: %w", err)
	}

	// 根据升级级别确定告警内容
	var alertMessage string
	var severity string
	var channels []string
	var recipients []string

	switch escalationLevel {
	case 1:
		alertMessage = fmt.Sprintf("事件 %s 已升级到级别 1，需要关注", incidentEntity.IncidentNumber)
		severity = "high"
		channels = []string{"email", "slack"}
		recipients = []string{"manager@company.com", "team@company.com"}
	case 2:
		alertMessage = fmt.Sprintf("事件 %s 已升级到级别 2，需要立即处理", incidentEntity.IncidentNumber)
		severity = "critical"
		channels = []string{"email", "sms", "slack"}
		recipients = []string{"director@company.com", "manager@company.com", "team@company.com"}
	case 3:
		alertMessage = fmt.Sprintf("事件 %s 已升级到级别 3，需要紧急处理", incidentEntity.IncidentNumber)
		severity = "critical"
		channels = []string{"email", "sms", "slack", "webhook"}
		recipients = []string{"cto@company.com", "director@company.com", "manager@company.com", "team@company.com"}
	default:
		alertMessage = fmt.Sprintf("事件 %s 已升级到级别 %d", incidentEntity.IncidentNumber, escalationLevel)
		severity = "high"
		channels = []string{"email"}
		recipients = []string{"admin@company.com"}
	}

	// 创建升级告警
	_, err = s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incidentID,
		AlertType:  "escalation",
		AlertName:  "事件升级告警",
		Message:    alertMessage,
		Severity:   severity,
		Channels:   channels,
		Recipients: recipients,
		Metadata: map[string]interface{}{
			"escalation_level":  escalationLevel,
			"incident_title":    incidentEntity.Title,
			"incident_severity": incidentEntity.Severity,
		},
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create escalation alert", "error", err)
		return fmt.Errorf("failed to create escalation alert: %w", err)
	}

	s.logger.Infow("Escalation alerts processed successfully", "incident_id", incidentID, "level", escalationLevel)
	return nil
}

// ProcessThresholdAlerts 处理阈值告警
func (s *IncidentAlertingService) ProcessThresholdAlerts(ctx context.Context, incidentID int, metricType string, metricValue float64, threshold float64, tenantID int) error {
	s.logger.Infow("Processing threshold alerts", "metric_type", metricType, "value", metricValue, "threshold", threshold)

	// 检查是否超过阈值
	if metricValue <= threshold {
		return nil
	}

	// 创建阈值告警
	alertMessage := fmt.Sprintf("指标 %s 当前值 %.2f 超过阈值 %.2f", metricType, metricValue, threshold)

	_, err := s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incidentID,
		AlertType:  "threshold",
		AlertName:  "阈值告警",
		Message:    alertMessage,
		Severity:   "medium",
		Channels:   []string{"email", "slack"},
		Recipients: []string{"monitoring@company.com", "team@company.com"},
		Metadata: map[string]interface{}{
			"metric_type":  metricType,
			"metric_value": metricValue,
			"threshold":    threshold,
			"exceeded_by":  metricValue - threshold,
		},
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create threshold alert", "error", err)
		return fmt.Errorf("failed to create threshold alert: %w", err)
	}

	s.logger.Infow("Threshold alert processed successfully", "metric_type", metricType)
	return nil
}

// ProcessSLAViolationAlerts 处理SLA违规告警
func (s *IncidentAlertingService) ProcessSLAViolationAlerts(ctx context.Context, incidentID int, violationType string, tenantID int) error {
	s.logger.Infow("Processing SLA violation alerts", "incident_id", incidentID, "violation_type", violationType)

	// 获取事件信息
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
			incident.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("incident not found")
		}
		return fmt.Errorf("failed to get incident: %w", err)
	}

	// 根据违规类型确定告警内容
	var alertMessage string
	var severity string
	var channels []string
	var recipients []string

	switch violationType {
	case "response_time":
		alertMessage = fmt.Sprintf("事件 %s 响应时间超时，违反SLA", incidentEntity.IncidentNumber)
		severity = "high"
		channels = []string{"email", "sms", "slack"}
		recipients = []string{"manager@company.com", "team@company.com"}
	case "resolution_time":
		alertMessage = fmt.Sprintf("事件 %s 解决时间超时，违反SLA", incidentEntity.IncidentNumber)
		severity = "critical"
		channels = []string{"email", "sms", "slack", "webhook"}
		recipients = []string{"director@company.com", "manager@company.com", "team@company.com"}
	default:
		alertMessage = fmt.Sprintf("事件 %s 违反SLA: %s", incidentEntity.IncidentNumber, violationType)
		severity = "medium"
		channels = []string{"email"}
		recipients = []string{"admin@company.com"}
	}

	// 创建SLA违规告警
	_, err = s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incidentID,
		AlertType:  "sla_violation",
		AlertName:  "SLA违规告警",
		Message:    alertMessage,
		Severity:   severity,
		Channels:   channels,
		Recipients: recipients,
		Metadata: map[string]interface{}{
			"violation_type":    violationType,
			"incident_title":    incidentEntity.Title,
			"incident_priority": incidentEntity.Priority,
		},
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create SLA violation alert", "error", err)
		return fmt.Errorf("failed to create SLA violation alert: %w", err)
	}

	s.logger.Infow("SLA violation alert processed successfully", "incident_id", incidentID, "violation_type", violationType)
	return nil
}

// 转换为响应DTO
func (s *IncidentAlertingService) toIncidentAlertResponse(alert *ent.IncidentAlert) *dto.IncidentAlertResponse {
	return &dto.IncidentAlertResponse{
		ID:             alert.ID,
		IncidentID:     alert.IncidentID,
		AlertType:      alert.AlertType,
		AlertName:      alert.AlertName,
		Message:        alert.Message,
		Severity:       alert.Severity,
		Status:         alert.Status,
		Channels:       alert.Channels,
		Recipients:     alert.Recipients,
		TriggeredAt:    alert.TriggeredAt,
		AcknowledgedAt: &alert.AcknowledgedAt,
		ResolvedAt:     &alert.ResolvedAt,
		AcknowledgedBy: &alert.AcknowledgedBy,
		Metadata:       alert.Metadata,
		TenantID:       alert.TenantID,
		CreatedAt:      alert.CreatedAt,
		UpdatedAt:      alert.UpdatedAt,
	}
}
