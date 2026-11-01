package cloudsignal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/connector/alert"
	"itsm-backend/dto"
)

// ErrBridgeReporterUnconfigured 表示没有配置自动建单的 reporter 用户。
// 建单桥在此状态下不认领任何告警，避免白白消耗重试次数。
var ErrBridgeReporterUnconfigured = errors.New("cloud alert bridge: automation reporter is not configured")

const (
	bridgeClaimLimit  = 20
	bridgeMaxAttempts = 3
	// bridgeStaleClaim 之后未完成的认领可被回收（进程崩溃后的恢复路径）。
	bridgeStaleClaim          = 10 * time.Minute
	bridgeTitleMaxRunes       = 200
	bridgeDescriptionMaxRunes = 4000
)

// IncidentCreator 是建单桥对事件域的最小依赖。生产实现是
// *service.IncidentService；它按 reporterID 重新校验用户属于本租户且启用。
type IncidentCreator interface {
	CreateAutomatedIncident(ctx context.Context, req *dto.CreateIncidentRequest, tenantID, reporterID int) (*dto.IncidentResponse, error)
}

// BridgeRepository 提供建单桥需要的两个只读投影。
type BridgeRepository interface {
	// FindIncidentByCloudAlertID 返回已由该告警创建的事件 ID；不存在返回 0, nil。
	FindIncidentByCloudAlertID(ctx context.Context, tenantID int, alertKey string) (int64, error)
	// FindCIByCloudResource 返回匹配的配置项 ID；不存在返回 0, nil。
	FindCIByCloudResource(ctx context.Context, tenantID int, provider, resourceID string) (int64, error)
}

// BridgeResult 是一轮建单的计量结果。
type BridgeResult struct {
	Claimed int
	Created int
	Reused  int
	Failed  int
}

// IncidentBridge 把已落库的 P1/P2 云监控告警提升为事件工单。
//
// 幂等有两层：报警认领是条件更新（同一告警同一时刻只有一个 worker 处理）；
// 创建前再按事件 metadata.cloudAlertId 反查，覆盖「建单成功后进程崩溃、
// 未及回写 bridged」的窗口，让重试识别出已创建而不重复建单。
type IncidentBridge struct {
	alerts     AlertRepository
	repo       BridgeRepository
	creator    IncidentCreator
	reporterID int
	logger     *zap.SugaredLogger
	now        func() time.Time
}

func NewIncidentBridge(alerts AlertRepository, repo BridgeRepository, creator IncidentCreator, reporterID int, logger *zap.SugaredLogger) *IncidentBridge {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &IncidentBridge{
		alerts:     alerts,
		repo:       repo,
		creator:    creator,
		reporterID: reporterID,
		logger:     logger,
		now:        time.Now,
	}
}

// RunTenant 认领并处理一批告警。单条告警失败不阻断同批其他告警。
func (b *IncidentBridge) RunTenant(ctx context.Context, tenantID int) (BridgeResult, error) {
	var result BridgeResult
	if b == nil || b.alerts == nil || b.repo == nil || b.creator == nil {
		return result, errors.New("cloud alert bridge is not configured")
	}
	if tenantID <= 0 {
		return result, errors.New("cloud alert bridge requires a positive tenant ID")
	}
	if b.reporterID <= 0 {
		return result, ErrBridgeReporterUnconfigured
	}

	now := b.now()
	candidates, err := b.alerts.ClaimBridgeCandidates(ctx, tenantID,
		[]string{string(alert.SeverityP1), string(alert.SeverityP2)},
		bridgeClaimLimit, bridgeMaxAttempts, now.Add(-bridgeStaleClaim), now)
	if err != nil {
		return result, fmt.Errorf("claim cloud alerts for bridging: %w", err)
	}
	result.Claimed = len(candidates)

	var errs []error
	for _, candidate := range candidates {
		created, err := b.bridgeOne(ctx, tenantID, candidate)
		switch {
		case err != nil:
			result.Failed++
			errs = append(errs, fmt.Errorf("alert %s: %w", candidate.AlertID, err))
		case created:
			result.Created++
		default:
			result.Reused++
		}
	}
	return result, errors.Join(errs...)
}

// bridgeOne 返回 created=false 表示复用了已存在的工单。
func (b *IncidentBridge) bridgeOne(ctx context.Context, tenantID int, candidate *BridgeCandidate) (bool, error) {
	existing, err := b.repo.FindIncidentByCloudAlertID(ctx, tenantID, candidate.AlertID)
	if err != nil {
		reason := fmt.Sprintf("lookup existing incident: %v", err)
		b.markFailed(ctx, tenantID, candidate.ID, reason)
		return false, errors.New(reason)
	}
	if existing > 0 {
		if err := b.alerts.MarkBridged(ctx, tenantID, candidate.ID, existing, b.now()); err != nil {
			return false, fmt.Errorf("mark alert bridged to existing incident: %w", err)
		}
		return false, nil
	}

	ciIDs := b.relatedCIIDs(ctx, tenantID, candidate.AlertRecord)
	response, err := b.creator.CreateAutomatedIncident(ctx, buildIncidentRequest(&candidate.AlertRecord, ciIDs), tenantID, b.reporterID)
	if err != nil {
		b.markFailed(ctx, tenantID, candidate.ID, err.Error())
		return false, fmt.Errorf("create incident: %w", err)
	}
	if response == nil || response.ID <= 0 {
		b.markFailed(ctx, tenantID, candidate.ID, "incident service returned an empty response")
		return false, errors.New("incident service returned an empty response")
	}
	if err := b.alerts.MarkBridged(ctx, tenantID, candidate.ID, int64(response.ID), b.now()); err != nil {
		// 工单已存在，下一轮会通过 cloudAlertId 反查收口；这里只报告回写失败。
		return true, fmt.Errorf("mark alert bridged: %w", err)
	}
	return true, nil
}

// relatedCIIDs 尽力关联配置项。查不到或查询失败都不阻断建单——
// 告警本身的价值远高于 CI 关联的完整性。
func (b *IncidentBridge) relatedCIIDs(ctx context.Context, tenantID int, record AlertRecord) []int {
	instanceID := strings.TrimSpace(record.Labels["instanceId"])
	if instanceID == "" {
		return nil
	}
	provider := strings.TrimSpace(record.Labels["provider"])
	ciID, err := b.repo.FindCIByCloudResource(ctx, tenantID, provider, instanceID)
	if err != nil {
		b.logger.Warnw("lookup CI for cloud alert failed", "tenant_id", tenantID, "alert_id", record.AlertID, "error", err)
		return nil
	}
	if ciID <= 0 {
		return nil
	}
	return []int{int(ciID)}
}

func (b *IncidentBridge) markFailed(ctx context.Context, tenantID int, alertID int64, reason string) {
	if err := b.alerts.MarkBridgeFailed(ctx, tenantID, alertID, reason, b.now()); err != nil {
		b.logger.Warnw("mark cloud alert bridge failure failed", "tenant_id", tenantID, "alert_id", alertID, "error", err)
	}
}

func buildIncidentRequest(record *AlertRecord, ciIDs []int) *dto.CreateIncidentRequest {
	title := strings.TrimSpace(fmt.Sprintf("[%s] %s", record.Severity, record.Name))
	description := strings.TrimSpace(record.Description)
	context := make([]string, 0, 4)
	if instance := record.Labels["instanceName"]; instance != "" {
		context = append(context, "实例: "+instance)
	}
	if rule := record.Labels["ruleName"]; rule != "" {
		context = append(context, "报警规则: "+rule)
	}
	if metric := record.Labels["metricName"]; metric != "" {
		context = append(context, "监控指标: "+metric)
	}
	if len(context) > 0 {
		description = strings.TrimSpace(description + "\n\n" + strings.Join(context, "\n"))
	}
	firedAt := record.FiredAt
	metadata := map[string]interface{}{
		"cloudAlertId":     record.AlertID,
		"cloudAlertSource": AlertSourceCloudMonitor,
		"cloudAlertLevel":  record.Severity,
	}
	putMeta := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			metadata[key] = value
		}
	}
	putMeta("cloudAlertRuleId", record.Labels["ruleId"])
	putMeta("cloudAlertRuleName", record.Labels["ruleName"])
	putMeta("cloudAccountId", record.Labels["cloudAccountId"])
	putMeta("cloudInstanceId", record.Labels["instanceId"])
	putMeta("cloudProvider", record.Labels["provider"])
	if !firedAt.IsZero() {
		putMeta("cloudAlertTime", firedAt.UTC().Format(time.RFC3339))
	}

	return &dto.CreateIncidentRequest{
		Title:                truncateRunes(title, bridgeTitleMaxRunes),
		Description:          truncateRunes(description, bridgeDescriptionMaxRunes),
		Type:                 "incident",
		Priority:             priorityFromSeverity(record.Severity),
		Source:               "monitoring",
		ConfigurationItemIDs: ciIDs,
		Metadata:             metadata,
		DetectedAt:           &firedAt,
	}
}

func priorityFromSeverity(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case string(alert.SeverityP1):
		return "critical"
	case string(alert.SeverityP2):
		return "high"
	case string(alert.SeverityP3):
		return "medium"
	default:
		return "low"
	}
}
