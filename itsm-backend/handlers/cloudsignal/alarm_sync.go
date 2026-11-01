package cloudsignal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/connector/alert"
	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
)

// ErrSignalUnconfigured 表示账号没有凭据，本轮同步被跳过。
// 它是能力状态 unconfigured，不是故障：调用方不应把它计入失败告警。
var ErrSignalUnconfigured = errors.New("cloud signal sync: credential is not configured")

const (
	// alarmPageSize 是 CloudMonitor 报警历史单页上限。
	alarmPageSize = 100
	// alarmMaxPages 防止异常放大时单轮扫描无限翻页；命中上限时游标不前进，
	// 下一轮从同一窗口重扫，依赖 alerts 的唯一键去重。
	alarmMaxPages = 100
	// alarmWindowOverlap 让相邻窗口重叠，覆盖报警落库延迟。
	alarmWindowOverlap = 15 * time.Minute
	// alarmFirstLookback 是账号首次同步的回看窗口。
	alarmFirstLookback = 24 * time.Hour
	// alarmMaxWindow 是 CloudMonitor 单次查询上限（15 天）留出的余量。
	alarmMaxWindow = 14 * 24 * time.Hour
)

// CloudAccountRef 是同步所需的最小账号投影。
// 它刻意不依赖 CMDB 的完整 CloudAccount 模型，装配层负责转换。
type CloudAccountRef struct {
	ID            int64
	TenantID      int
	Provider      string
	AccountName   string
	CredentialRef string
}

// CloudAccountSource 枚举租户下启用的云账号。实现放在装配层，
// 避免 cloudsignal 直接依赖其他领域的仓储。
type CloudAccountSource interface {
	ListActiveCloudAccounts(ctx context.Context, tenantID int) ([]CloudAccountRef, error)
}

// AlarmLogSource 是 CloudMonitor 报警历史的最小读接口，生产实现是
// *aliyun.CloudMonitorClient，测试用固定分页响应替身。
type AlarmLogSource interface {
	DescribeAlertLogs(ctx context.Context, start, end time.Time, page, pageSize int) ([]aliyun.CloudMonitorAlertLog, int, error)
}

// AlarmClientFactory 由已解析的凭据构造报警历史客户端。
type AlarmClientFactory func(cred *cloud.ResolvedCredential) (AlarmLogSource, error)

// AlarmSyncResult 是一轮账号同步的计量结果。
type AlarmSyncResult struct {
	AccountID int64
	Pages     int
	Fetched   int
	Persisted int
	Created   int
	Skipped   int
	// Truncated 表示本轮撞到翻页上限，游标没有推进。
	Truncated bool
}

// AlarmSyncSummary 是一轮租户同步的汇总。
type AlarmSyncSummary struct {
	TenantID  int
	Accounts  int
	Fetched   int
	Persisted int
	Created   int
	Failed    int
}

// AlarmSyncService 轮询 CloudMonitor 报警历史并写入 alerts。
//
// 报警历史只包含已经触发的报警、没有 OK 状态，所以这里只落 firing，
// 告警的确认/解决由操作员在本地完成，不随轮询被覆盖。
type AlarmSyncService struct {
	states    SyncStateRepository
	alerts    AlertRepository
	accounts  CloudAccountSource
	resolver  cloud.TenantSecretResolver
	newClient AlarmClientFactory
	logger    *zap.SugaredLogger
	now       func() time.Time
}

func NewAlarmSyncService(states SyncStateRepository, alerts AlertRepository, accounts CloudAccountSource, resolver cloud.TenantSecretResolver, logger *zap.SugaredLogger) *AlarmSyncService {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &AlarmSyncService{
		states:    states,
		alerts:    alerts,
		accounts:  accounts,
		resolver:  resolver,
		newClient: defaultAlarmClientFactory,
		logger:    logger,
		now:       time.Now,
	}
}

func defaultAlarmClientFactory(cred *cloud.ResolvedCredential) (AlarmLogSource, error) {
	if cred == nil || cred.AccessKeyID == "" || cred.AccessKeySecret == "" {
		return nil, fmt.Errorf("aliyun credential is incomplete for cloud monitor")
	}
	return aliyun.NewCloudMonitorClient(cred.AccessKeyID, cred.AccessKeySecret, cred.SessionToken), nil
}

// SyncTenant 逐个账号同步；单账号失败只记录，不影响其他账号。
func (s *AlarmSyncService) SyncTenant(ctx context.Context, tenantID int) (AlarmSyncSummary, error) {
	summary := AlarmSyncSummary{TenantID: tenantID}
	if s == nil || s.states == nil || s.alerts == nil || s.accounts == nil {
		return summary, errors.New("alarm sync service is not configured")
	}
	if tenantID <= 0 {
		return summary, errors.New("alarm sync requires a positive tenant ID")
	}
	accounts, err := s.accounts.ListActiveCloudAccounts(ctx, tenantID)
	if err != nil {
		return summary, fmt.Errorf("list cloud accounts: %w", err)
	}
	var errs []error
	for _, account := range accounts {
		if account.TenantID <= 0 {
			account.TenantID = tenantID
		}
		result, syncErr := s.SyncAccount(ctx, account)
		summary.Accounts++
		summary.Fetched += result.Fetched
		summary.Persisted += result.Persisted
		summary.Created += result.Created
		if syncErr != nil && !errors.Is(syncErr, ErrSignalUnconfigured) {
			summary.Failed++
			errs = append(errs, fmt.Errorf("account %d: %w", account.ID, syncErr))
		}
	}
	return summary, errors.Join(errs...)
}

// SyncAccount 拉取一个账号的报警历史并 upsert 到 alerts。
func (s *AlarmSyncService) SyncAccount(ctx context.Context, account CloudAccountRef) (AlarmSyncResult, error) {
	result := AlarmSyncResult{AccountID: account.ID}
	if s == nil || s.states == nil || s.alerts == nil {
		return result, errors.New("alarm sync service is not configured")
	}
	if account.ID <= 0 || account.TenantID <= 0 {
		return result, errors.New("alarm sync requires a tenant-scoped cloud account")
	}
	if strings.TrimSpace(account.CredentialRef) == "" {
		markSignalUnconfigured(ctx, s.states, s.logger, account.TenantID, account.ID, SignalKindAlarm, s.now())
		return result, ErrSignalUnconfigured
	}

	now := s.now()
	state, err := s.states.Get(ctx, account.TenantID, account.ID, SignalKindAlarm)
	if err != nil {
		return result, fmt.Errorf("load alarm sync state: %w", err)
	}
	cred, err := resolveAccountCredential(ctx, s.resolver, account)
	if err != nil {
		recordSignalFailure(ctx, s.states, s.logger, account.TenantID, account.ID, SignalKindAlarm, "credential_resolve_failed", err, now)
		return result, err
	}
	client, err := s.newAlarmClient(cred)
	if err != nil {
		recordSignalFailure(ctx, s.states, s.logger, account.TenantID, account.ID, SignalKindAlarm, "client_init_failed", err, now)
		return result, err
	}

	anchor, start, end := alarmWindow(state, now)
	watermark := end
	scanned := 0
	for page := 1; page <= alarmMaxPages; page++ {
		logs, total, err := client.DescribeAlertLogs(ctx, start, end, page, alarmPageSize)
		if err != nil {
			recordSignalFailure(ctx, s.states, s.logger, account.TenantID, account.ID, SignalKindAlarm, "describe_alert_log_failed", err, now)
			return result, fmt.Errorf("describe cloud monitor alerts: %w", err)
		}
		result.Pages = page
		result.Fetched += len(logs)
		scanned += len(logs)
		for i := range logs {
			normalized, ok := standardAlertFromCMS(account, logs[i], now)
			if !ok {
				result.Skipped++
				continue
			}
			_, created, err := s.alerts.Upsert(ctx, account.TenantID, normalized, now)
			if err != nil {
				recordSignalFailure(ctx, s.states, s.logger, account.TenantID, account.ID, SignalKindAlarm, "alert_persist_failed", err, now)
				return result, fmt.Errorf("persist cloud monitor alert: %w", err)
			}
			result.Persisted++
			if created {
				result.Created++
			}
		}
		if len(logs) == 0 || (total > 0 && scanned >= total) {
			break
		}
		if page == alarmMaxPages {
			result.Truncated = true
			watermark = anchor
			s.logger.Warnw("cloud monitor alarm window truncated; watermark not advanced",
				"tenant_id", account.TenantID, "account_id", account.ID,
				"scan_start", start, "scan_end", end, "total", total)
		}
	}

	if err := s.states.MarkSuccess(ctx, account.TenantID, account.ID, SignalKindAlarm, watermark, now, int64(result.Fetched)); err != nil {
		return result, fmt.Errorf("record alarm sync success: %w", err)
	}
	return result, nil
}

func (s *AlarmSyncService) newAlarmClient(cred *cloud.ResolvedCredential) (AlarmLogSource, error) {
	if s.newClient == nil {
		return nil, errors.New("alarm client factory is not configured")
	}
	return s.newClient(cred)
}

// alarmWindow 计算本轮查询窗口。
//
// 第一轮从 24 小时前开始；之后从上次成功游标回退一小段重叠。窗口长度必须
// 收敛在 CloudMonitor 的 15 天上限内——同步停用很久后游标可能非常旧。
func alarmWindow(state *SyncState, now time.Time) (anchor, start, end time.Time) {
	anchor = now.Add(-alarmFirstLookback)
	if state != nil && state.Watermark != nil && !state.Watermark.IsZero() {
		anchor = *state.Watermark
	}
	if anchor.After(now) {
		anchor = now
	}
	start = anchor.Add(-alarmWindowOverlap)
	if earliest := now.Add(-alarmMaxWindow); start.Before(earliest) {
		start = earliest
	}
	if start.After(now) {
		start = now
	}
	return anchor, start, now
}

func standardAlertFromCMS(account CloudAccountRef, log aliyun.CloudMonitorAlertLog, now time.Time) (*alert.StandardAlert, bool) {
	logID := strings.TrimSpace(string(log.LogID))
	if logID == "" {
		return nil, false
	}
	firedAt := parseCMSAlertTime(log.AlertTime, now)
	labels := map[string]string{}
	putLabel := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			labels[key] = value
		}
	}
	putLabel("cloudAccountId", strconv.FormatInt(account.ID, 10))
	putLabel("provider", account.Provider)
	putLabel("namespace", log.Namespace)
	putLabel("product", log.Product)
	putLabel("metricName", log.MetricName)
	putLabel("instanceId", log.InstanceID)
	putLabel("instanceName", log.InstanceName)
	putLabel("ruleId", log.RuleID)
	putLabel("ruleName", log.RuleName)
	putLabel("level", log.Level)
	putLabel("groupId", log.GroupID)

	name := strings.TrimSpace(log.RuleName)
	if name == "" {
		name = "云监控告警"
	}
	if instance := strings.TrimSpace(log.InstanceName); instance != "" {
		name = name + " - " + instance
	}
	service := strings.TrimSpace(log.InstanceName)
	if service == "" {
		service = strings.TrimSpace(log.Product)
	}

	return &alert.StandardAlert{
		AlertID:     logID,
		Source:      AlertSourceCloudMonitor,
		SourceRaw:   "cloudmonitor",
		TenantID:    account.TenantID,
		Name:        name,
		Description: strings.TrimSpace(log.Message),
		Severity:    severityFromCMSLevel(log.Level),
		Status:      "firing",
		Labels:      labels,
		Annotations: map[string]string{
			"alertTime":  strings.TrimSpace(log.AlertTime),
			"dimensions": strings.TrimSpace(log.Dimensions),
			"rawLevel":   strings.TrimSpace(log.Level),
		},
		Service: service,
		Tags:    []string{"cloudmonitor"},
		FiredAt: firedAt,
		// 原始字段名与 CloudMonitor 返回一致，便于回溯上游。
		RawPayload: map[string]interface{}{
			"LogId":        logID,
			"AlertTime":    log.AlertTime,
			"Level":        log.Level,
			"RuleId":       log.RuleID,
			"RuleName":     log.RuleName,
			"Namespace":    log.Namespace,
			"Product":      log.Product,
			"MetricName":   log.MetricName,
			"InstanceId":   log.InstanceID,
			"InstanceName": log.InstanceName,
			"Dimensions":   log.Dimensions,
			"Message":      log.Message,
			"GroupId":      log.GroupID,
		},
	}, true
}

// severityFromCMSLevel 把 CloudMonitor 的报警级别映射到平台分级。
// 未知级别降为 P3：宁可少建单，也不把不认识的信息抬成 P1。
func severityFromCMSLevel(level string) alert.Severity {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "CRITICAL":
		return alert.SeverityP1
	case "WARN", "WARNING":
		return alert.SeverityP2
	default:
		return alert.SeverityP3
	}
}

func parseCMSAlertTime(value string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed
		}
	}
	return fallback
}
