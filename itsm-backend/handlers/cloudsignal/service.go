package cloudsignal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/connector/alert"
)

// 三类可判定的失败语义：不存在回 404、状态冲突回 409、参数不在词表回 400。
var (
	ErrSignalNotFound      = errors.New("cloud signal: resource not found")
	ErrSignalStateConflict = errors.New("cloud signal: state conflict")
	ErrSignalInvalidFilter = errors.New("cloud signal: invalid filter")
)

// 列表接口对自由文本搜索设上限，避免超大 ILIKE 全表扫描。
const maxSearchRunes = 200

// vulnTypeSet 是云安全中心漏洞类型词表（app/emg 属应用与应急漏洞，首版未接入同步）。
var vulnTypeSet = map[string]struct{}{
	"cve": {}, "sys": {}, "cms": {}, "app": {}, "emg": {},
}

var alertStatusSet = map[string]struct{}{
	AlertStatusFiring: {}, AlertStatusAcknowledged: {}, AlertStatusResolved: {},
}

var alertSeveritySet = map[string]struct{}{
	string(alert.SeverityP1): {}, string(alert.SeverityP2): {},
	string(alert.SeverityP3): {}, string(alert.SeverityP4): {},
}

var findingStatusSet = map[string]struct{}{
	FindingStatusOpen: {}, FindingStatusFixing: {},
	FindingStatusFixed: {}, FindingStatusIgnored: {},
}

var findingSeveritySet = map[string]struct{}{
	SeverityCritical: {}, SeverityHigh: {}, SeverityMedium: {},
	SeverityLow: {}, SeverityUnknown: {},
}

// Service 是云信号查询与本地状态流转的唯一业务入口。
type Service struct {
	alerts   AlertRepository
	findings FindingRepository
	states   SyncStateRepository
	logger   *zap.SugaredLogger
	now      func() time.Time
}

func NewService(alerts AlertRepository, findings FindingRepository, states SyncStateRepository, logger *zap.SugaredLogger) *Service {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Service{alerts: alerts, findings: findings, states: states, logger: logger, now: time.Now}
}

func (s *Service) ListAlerts(ctx context.Context, tenantID int, filter AlertFilter) ([]*AlertRecord, int, error) {
	if s == nil || s.alerts == nil {
		return nil, 0, errors.New("cloud signal service is not configured")
	}
	if err := validateAlertFilter(&filter); err != nil {
		return nil, 0, err
	}
	return s.alerts.List(ctx, tenantID, filter)
}

func (s *Service) AcknowledgeAlert(ctx context.Context, tenantID int, id int64) (*AlertRecord, error) {
	record, err := s.getAlert(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if record.Status != AlertStatusFiring {
		return nil, fmt.Errorf("%w: alert %d is %s", ErrSignalStateConflict, id, record.Status)
	}
	now := s.now()
	ok, err := s.alerts.Acknowledge(ctx, tenantID, id, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: alert %d is no longer firing", ErrSignalStateConflict, id)
	}
	record.Status = AlertStatusAcknowledged
	if record.AcknowledgedAt == nil {
		record.AcknowledgedAt = &now
	}
	record.UpdatedAt = now
	return record, nil
}

func (s *Service) ResolveAlert(ctx context.Context, tenantID int, id int64) (*AlertRecord, error) {
	record, err := s.getAlert(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if record.Status != AlertStatusFiring && record.Status != AlertStatusAcknowledged {
		return nil, fmt.Errorf("%w: alert %d is %s", ErrSignalStateConflict, id, record.Status)
	}
	now := s.now()
	ok, err := s.alerts.Resolve(ctx, tenantID, id, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: alert %d was already resolved", ErrSignalStateConflict, id)
	}
	record.Status = AlertStatusResolved
	if record.ResolvedAt == nil {
		record.ResolvedAt = &now
	}
	record.UpdatedAt = now
	return record, nil
}

func (s *Service) getAlert(ctx context.Context, tenantID int, id int64) (*AlertRecord, error) {
	if s == nil || s.alerts == nil {
		return nil, errors.New("cloud signal service is not configured")
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w: alert id must be positive", ErrSignalInvalidFilter)
	}
	record, err := s.alerts.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("%w: alert %d", ErrSignalNotFound, id)
	}
	return record, nil
}

func (s *Service) ListFindings(ctx context.Context, tenantID int, filter FindingFilter) ([]*SecurityFinding, int, error) {
	if s == nil || s.findings == nil {
		return nil, 0, errors.New("cloud signal service is not configured")
	}
	if err := validateFindingFilter(&filter); err != nil {
		return nil, 0, err
	}
	return s.findings.List(ctx, tenantID, filter)
}

func (s *Service) FindingSummary(ctx context.Context, tenantID int) (*FindingSummary, error) {
	if s == nil || s.findings == nil {
		return nil, errors.New("cloud signal service is not configured")
	}
	return s.findings.Summary(ctx, tenantID)
}

// IgnoreFinding 只允许忽略未修复项；已修复/已忽略再点忽略属于状态冲突，
// 不能靠重复写入掩盖前端状态过期。
func (s *Service) IgnoreFinding(ctx context.Context, tenantID int, id int64) (*SecurityFinding, error) {
	finding, err := s.getFinding(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !IsActiveFindingStatus(finding.Status) {
		return nil, fmt.Errorf("%w: finding %d is %s", ErrSignalStateConflict, id, finding.Status)
	}
	now := s.now()
	ok, err := s.findings.SetStatus(ctx, tenantID, id, FindingStatusIgnored, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: finding %d disappeared", ErrSignalStateConflict, id)
	}
	finding.Status = FindingStatusIgnored
	finding.ResolvedAt = &now
	finding.UpdatedAt = now
	return finding, nil
}

// ReopenFinding 把误关闭（fixed）或当初忽略（ignored）的漏洞放回未修复队列。
func (s *Service) ReopenFinding(ctx context.Context, tenantID int, id int64) (*SecurityFinding, error) {
	finding, err := s.getFinding(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if IsActiveFindingStatus(finding.Status) {
		return nil, fmt.Errorf("%w: finding %d is already %s", ErrSignalStateConflict, id, finding.Status)
	}
	now := s.now()
	ok, err := s.findings.SetStatus(ctx, tenantID, id, FindingStatusOpen, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: finding %d disappeared", ErrSignalStateConflict, id)
	}
	finding.Status = FindingStatusOpen
	finding.ResolvedAt = nil
	finding.UpdatedAt = now
	return finding, nil
}

func (s *Service) getFinding(ctx context.Context, tenantID int, id int64) (*SecurityFinding, error) {
	if s == nil || s.findings == nil {
		return nil, errors.New("cloud signal service is not configured")
	}
	if id <= 0 {
		return nil, fmt.Errorf("%w: finding id must be positive", ErrSignalInvalidFilter)
	}
	finding, err := s.findings.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if finding == nil {
		return nil, fmt.Errorf("%w: finding %d", ErrSignalNotFound, id)
	}
	return finding, nil
}

// SyncStatus 两类信号一起返回：页面要展示「哪个账号哪种信号没在跑」。
func (s *Service) SyncStatus(ctx context.Context, tenantID int) ([]*SyncState, error) {
	if s == nil || s.states == nil {
		return nil, errors.New("cloud signal service is not configured")
	}
	alarms, err := s.states.List(ctx, tenantID, SignalKindAlarm)
	if err != nil {
		return nil, fmt.Errorf("list alarm sync states: %w", err)
	}
	findings, err := s.states.List(ctx, tenantID, SignalKindVulnerability)
	if err != nil {
		return nil, fmt.Errorf("list vulnerability sync states: %w", err)
	}
	return append(alarms, findings...), nil
}

func validateAlertFilter(filter *AlertFilter) error {
	if err := validateInSet("status", filter.Statuses, alertStatusSet); err != nil {
		return err
	}
	if err := validateInSet("severity", filter.Severities, alertSeveritySet); err != nil {
		return err
	}
	if filter.CloudAccountID < 0 {
		return fmt.Errorf("%w: cloudAccountId must be positive", ErrSignalInvalidFilter)
	}
	if len([]rune(filter.Search)) > maxSearchRunes {
		return fmt.Errorf("%w: search is too long", ErrSignalInvalidFilter)
	}
	normalizeListPaging(&filter.Page, &filter.PageSize)
	return nil
}

func validateFindingFilter(filter *FindingFilter) error {
	if err := validateInSet("status", filter.Statuses, findingStatusSet); err != nil {
		return err
	}
	if err := validateInSet("severity", filter.Severities, findingSeveritySet); err != nil {
		return err
	}
	if filter.VulnType != "" {
		if _, ok := vulnTypeSet[filter.VulnType]; !ok {
			return fmt.Errorf("%w: unknown vulnType %q", ErrSignalInvalidFilter, filter.VulnType)
		}
	}
	if filter.CloudAccountID < 0 {
		return fmt.Errorf("%w: cloudAccountId must be positive", ErrSignalInvalidFilter)
	}
	if len([]rune(filter.Search)) > maxSearchRunes {
		return fmt.Errorf("%w: search is too long", ErrSignalInvalidFilter)
	}
	if len([]rune(filter.InstanceID)) > maxSearchRunes {
		return fmt.Errorf("%w: instanceId is too long", ErrSignalInvalidFilter)
	}
	normalizeListPaging(&filter.Page, &filter.PageSize)
	return nil
}

func validateInSet(field string, values []string, allowed map[string]struct{}) error {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%w: empty %s", ErrSignalInvalidFilter, field)
		}
		if _, ok := allowed[trimmed]; !ok {
			return fmt.Errorf("%w: unknown %s %q", ErrSignalInvalidFilter, field, trimmed)
		}
	}
	return nil
}

func normalizeListPaging(page, pageSize *int) {
	if *page <= 0 {
		*page = 1
	}
	if *pageSize <= 0 {
		*pageSize = defaultPageSize
	}
	if *pageSize > maxPageSize {
		*pageSize = maxPageSize
	}
}
