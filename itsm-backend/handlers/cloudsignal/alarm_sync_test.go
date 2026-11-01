package cloudsignal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/connector/alert"
	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
)

type fakeSyncStateRepository struct {
	state        *SyncState
	getErr       error
	successCalls int
	watermark    time.Time
	items        int64
	successErr   error
	failureCode  string
	failureMsg   string
	failureState string
	failureCalls int
	unconfigured int
	states       map[SignalKind][]*SyncState
	listErr      error
}

func (f *fakeSyncStateRepository) List(_ context.Context, _ int, kind SignalKind) ([]*SyncState, error) {
	return f.states[kind], f.listErr
}

func (f *fakeSyncStateRepository) Get(context.Context, int, int64, SignalKind) (*SyncState, error) {
	return f.state, f.getErr
}

func (f *fakeSyncStateRepository) MarkSuccess(_ context.Context, _ int, _ int64, _ SignalKind, watermark, _ time.Time, items int64) error {
	f.successCalls++
	f.watermark = watermark
	f.items = items
	return f.successErr
}

func (f *fakeSyncStateRepository) MarkFailure(_ context.Context, _ int, _ int64, kind SignalKind, state, code, message string, _ time.Time) error {
	f.failureCalls++
	f.failureState = state
	f.failureCode = code
	f.failureMsg = message
	return nil
}

func (f *fakeSyncStateRepository) MarkUnconfigured(context.Context, int, int64, SignalKind, time.Time) error {
	f.unconfigured++
	return nil
}

type fakeAlertRepository struct {
	upserted    []*alert.StandardAlert
	upsertErr   error
	candidates  []*BridgeCandidate
	claimErr    error
	claimedSevs []string
	bridged     map[int64]int64
	failed      map[int64]string
	markErr     error

	listRecords  []*AlertRecord
	listTotal    int
	listErr      error
	listFilter   AlertFilter
	record       *AlertRecord
	getErr       error
	ackOK        bool
	ackErr       error
	resolveOK    bool
	resolveErr   error
	ackCalls     int
	resolveCalls int
}

func (f *fakeAlertRepository) Upsert(_ context.Context, _ int, a *alert.StandardAlert, _ time.Time) (int64, bool, error) {
	if f.upsertErr != nil {
		return 0, false, f.upsertErr
	}
	f.upserted = append(f.upserted, a)
	return int64(len(f.upserted)), true, nil
}

func (f *fakeAlertRepository) List(_ context.Context, _ int, filter AlertFilter) ([]*AlertRecord, int, error) {
	f.listFilter = filter
	return f.listRecords, f.listTotal, f.listErr
}

func (f *fakeAlertRepository) Get(context.Context, int, int64) (*AlertRecord, error) {
	return f.record, f.getErr
}

func (f *fakeAlertRepository) Acknowledge(context.Context, int, int64, time.Time) (bool, error) {
	f.ackCalls++
	return f.ackOK, f.ackErr
}

func (f *fakeAlertRepository) Resolve(context.Context, int, int64, time.Time) (bool, error) {
	f.resolveCalls++
	return f.resolveOK, f.resolveErr
}

func (f *fakeAlertRepository) ClaimBridgeCandidates(_ context.Context, _ int, severities []string, _, _ int, _, _ time.Time) ([]*BridgeCandidate, error) {
	f.claimedSevs = severities
	return f.candidates, f.claimErr
}

func (f *fakeAlertRepository) MarkBridged(_ context.Context, _ int, id, incidentID int64, _ time.Time) error {
	if f.markErr != nil {
		return f.markErr
	}
	if f.bridged == nil {
		f.bridged = map[int64]int64{}
	}
	f.bridged[id] = incidentID
	return nil
}

func (f *fakeAlertRepository) MarkBridgeFailed(_ context.Context, _ int, id int64, reason string, _ time.Time) error {
	if f.failed == nil {
		f.failed = map[int64]string{}
	}
	f.failed[id] = reason
	return nil
}

type fakeAccountSource struct {
	accounts []CloudAccountRef
	err      error
}

func (f fakeAccountSource) ListActiveCloudAccounts(context.Context, int) ([]CloudAccountRef, error) {
	return f.accounts, f.err
}

type fakeResolver struct {
	ref string
	err error
}

func (f fakeResolver) Resolve(context.Context, int, string) (string, error) {
	return f.ref, f.err
}

type fakeAlarmSource struct {
	pages   map[int][]aliyun.CloudMonitorAlertLog
	total   int
	err     error
	windows [][2]time.Time
}

func (f *fakeAlarmSource) DescribeAlertLogs(_ context.Context, start, end time.Time, page, _ int) ([]aliyun.CloudMonitorAlertLog, int, error) {
	f.windows = append(f.windows, [2]time.Time{start, end})
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.pages[page], f.total, nil
}

const testAliyunCredential = `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`

func newAlarmSyncForTest(stateRepo SyncStateRepository, alertRepo AlertRepository, source AlarmLogSource) (*AlarmSyncService, time.Time) {
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	svc := NewAlarmSyncService(stateRepo, alertRepo, nil, fakeResolver{ref: testAliyunCredential}, nil)
	svc.now = func() time.Time { return now }
	svc.newClient = func(cred *cloud.ResolvedCredential) (AlarmLogSource, error) {
		if cred.AccessKeyID != "ak-test" {
			return nil, errors.New("unexpected credential")
		}
		return source, nil
	}
	return svc, now
}

func testAccount() CloudAccountRef {
	return CloudAccountRef{ID: 11, TenantID: 7, Provider: "aliyun", AccountName: "生产账号", CredentialRef: "secret://tenant-7/aliyun"}
}

func TestAlarmSyncFirstRunScansLookbackAndPersists(t *testing.T) {
	stateRepo := &fakeSyncStateRepository{}
	alertRepo := &fakeAlertRepository{}
	source := &fakeAlarmSource{
		total: 3,
		pages: map[int][]aliyun.CloudMonitorAlertLog{
			1: {
				{LogID: "log-1", AlertTime: "2026-11-01 09:00:00", Level: "CRITICAL", RuleName: "CPU 打满",
					InstanceID: "i-bp1", InstanceName: "web-01", Product: "ECS", MetricName: "cpu_total"},
				{LogID: "log-2", AlertTime: "2026-11-01 09:10:00", Level: "WARN", RuleName: "磁盘使用率高"},
			},
			2: {
				{LogID: "log-3", AlertTime: "2026-11-01 09:20:00", Level: "INFO", RuleName: "变更通知"},
			},
		},
	}
	svc, now := newAlarmSyncForTest(stateRepo, alertRepo, source)

	result, err := svc.SyncAccount(context.Background(), testAccount())
	require.NoError(t, err)

	assert.Equal(t, 3, result.Fetched)
	assert.Equal(t, 3, result.Persisted)
	assert.Equal(t, 3, result.Created)
	assert.Equal(t, 2, result.Pages)
	require.Len(t, source.windows, 2)
	assert.Equal(t, now.Add(-alarmFirstLookback-alarmWindowOverlap), source.windows[0][0])
	assert.Equal(t, now, source.windows[0][1])

	require.Len(t, alertRepo.upserted, 3)
	assert.Equal(t, AlertSourceCloudMonitor, alertRepo.upserted[0].Source)
	assert.Equal(t, "firing", alertRepo.upserted[0].Status)
	assert.Equal(t, alert.SeverityP1, alertRepo.upserted[0].Severity)
	assert.Equal(t, alert.SeverityP2, alertRepo.upserted[1].Severity)
	assert.Equal(t, alert.SeverityP3, alertRepo.upserted[2].Severity)
	assert.Equal(t, "11", alertRepo.upserted[0].Labels["cloudAccountId"])
	assert.Equal(t, "i-bp1", alertRepo.upserted[0].Labels["instanceId"])
	assert.Equal(t, "web-01", alertRepo.upserted[0].Labels["instanceName"])
	assert.Equal(t, 7, alertRepo.upserted[0].TenantID)

	assert.Equal(t, 1, stateRepo.successCalls)
	assert.Equal(t, now, stateRepo.watermark)
	assert.Equal(t, int64(3), stateRepo.items)
}

func TestAlarmSyncUsesWatermarkWithOverlap(t *testing.T) {
	watermark := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	stateRepo := &fakeSyncStateRepository{state: &SyncState{Watermark: &watermark}}
	source := &fakeAlarmSource{total: 0}
	svc, now := newAlarmSyncForTest(stateRepo, &fakeAlertRepository{}, source)

	_, err := svc.SyncAccount(context.Background(), testAccount())
	require.NoError(t, err)

	require.NotEmpty(t, source.windows)
	assert.Equal(t, watermark.Add(-alarmWindowOverlap), source.windows[0][0])
	assert.Equal(t, now, source.windows[0][1])
	assert.Equal(t, now, stateRepo.watermark)
}

func TestAlarmSyncClampsAncientWatermarkToServiceLimit(t *testing.T) {
	watermark := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	stateRepo := &fakeSyncStateRepository{state: &SyncState{Watermark: &watermark}}
	source := &fakeAlarmSource{total: 0}
	svc, now := newAlarmSyncForTest(stateRepo, &fakeAlertRepository{}, source)

	_, err := svc.SyncAccount(context.Background(), testAccount())
	require.NoError(t, err)

	assert.Equal(t, now.Add(-alarmMaxWindow), source.windows[0][0])
}

func TestAlarmSyncSkipsLogsWithoutID(t *testing.T) {
	source := &fakeAlarmSource{total: 2, pages: map[int][]aliyun.CloudMonitorAlertLog{
		1: {{LogID: "", Level: "CRITICAL"}, {LogID: "log-2", Level: "WARN"}},
	}}
	alertRepo := &fakeAlertRepository{}
	svc, _ := newAlarmSyncForTest(&fakeSyncStateRepository{}, alertRepo, source)

	result, err := svc.SyncAccount(context.Background(), testAccount())
	require.NoError(t, err)

	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 1, result.Persisted)
	require.Len(t, alertRepo.upserted, 1)
	assert.Equal(t, "log-2", alertRepo.upserted[0].AlertID)
}

func TestAlarmSyncMarksFailureOnSourceError(t *testing.T) {
	stateRepo := &fakeSyncStateRepository{}
	source := &fakeAlarmSource{err: errors.New("Forbidden.RAM")}
	svc, _ := newAlarmSyncForTest(stateRepo, &fakeAlertRepository{}, source)

	_, err := svc.SyncAccount(context.Background(), testAccount())
	require.Error(t, err)

	assert.Equal(t, 1, stateRepo.failureCalls)
	assert.Equal(t, SyncStateUnavailable, stateRepo.failureState)
	assert.Equal(t, "describe_alert_log_failed", stateRepo.failureCode)
	assert.Contains(t, stateRepo.failureMsg, "Forbidden.RAM")
	assert.Equal(t, 0, stateRepo.successCalls)
}

func TestAlarmSyncUnconfiguredCredentialIsNotFailure(t *testing.T) {
	stateRepo := &fakeSyncStateRepository{}
	alertRepo := &fakeAlertRepository{}
	svc, _ := newAlarmSyncForTest(stateRepo, alertRepo, &fakeAlarmSource{})

	account := testAccount()
	account.CredentialRef = ""
	_, err := svc.SyncAccount(context.Background(), account)
	assert.ErrorIs(t, err, ErrSignalUnconfigured)
	assert.Equal(t, 1, stateRepo.unconfigured)
	assert.Equal(t, 0, stateRepo.failureCalls)
	assert.Empty(t, alertRepo.upserted)
}

func TestAlarmSyncCredentialResolveFailureMarksUnavailable(t *testing.T) {
	stateRepo := &fakeSyncStateRepository{}
	svc := NewAlarmSyncService(stateRepo, &fakeAlertRepository{}, nil, fakeResolver{err: errors.New("secret missing")}, nil)
	svc.now = func() time.Time { return time.Now() }

	_, err := svc.SyncAccount(context.Background(), testAccount())
	require.Error(t, err)
	assert.Equal(t, "credential_resolve_failed", stateRepo.failureCode)
	assert.Equal(t, SyncStateUnavailable, stateRepo.failureState)
}

func TestAlarmSyncTruncationKeepsWatermark(t *testing.T) {
	watermark := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	stateRepo := &fakeSyncStateRepository{state: &SyncState{Watermark: &watermark}}
	source := &fakeAlarmSource{total: 1_000_000, pages: map[int][]aliyun.CloudMonitorAlertLog{}}
	for page := 1; page <= alarmMaxPages; page++ {
		source.pages[page] = []aliyun.CloudMonitorAlertLog{{LogID: "log-dup"}}
	}
	svc, _ := newAlarmSyncForTest(stateRepo, &fakeAlertRepository{}, source)

	result, err := svc.SyncAccount(context.Background(), testAccount())
	require.NoError(t, err)

	assert.True(t, result.Truncated)
	assert.Equal(t, alarmMaxPages, result.Pages)
	assert.Equal(t, watermark, stateRepo.watermark)
}

func TestAlarmSyncRejectsAccountWithoutTenant(t *testing.T) {
	svc, _ := newAlarmSyncForTest(&fakeSyncStateRepository{}, &fakeAlertRepository{}, &fakeAlarmSource{})

	_, err := svc.SyncAccount(context.Background(), CloudAccountRef{ID: 3})
	require.Error(t, err)
}

func TestSyncTenantContinuesAfterAccountFailure(t *testing.T) {
	stateRepo := &fakeSyncStateRepository{}
	source := &fakeAlarmSource{err: errors.New("boom")}
	svc, _ := newAlarmSyncForTest(stateRepo, &fakeAlertRepository{}, source)
	svc.accounts = fakeAccountSource{accounts: []CloudAccountRef{
		{ID: 1, TenantID: 7, Provider: "aliyun", CredentialRef: "secret://tenant-7/a"},
		{ID: 2, TenantID: 7, Provider: "aliyun", CredentialRef: "secret://tenant-7/b"},
	}}

	summary, err := svc.SyncTenant(context.Background(), 7)
	require.Error(t, err)
	assert.Equal(t, 2, summary.Accounts)
	assert.Equal(t, 2, summary.Failed)
}

func TestSeverityFromCMSLevel(t *testing.T) {
	cases := map[string]alert.Severity{
		"CRITICAL": alert.SeverityP1,
		"critical": alert.SeverityP1,
		"WARN":     alert.SeverityP2,
		"warning":  alert.SeverityP2,
		"INFO":     alert.SeverityP3,
		"":         alert.SeverityP3,
		"P4":       alert.SeverityP3,
	}
	for level, want := range cases {
		assert.Equal(t, want, severityFromCMSLevel(level), level)
	}
}

func TestParseCMSAlertTime(t *testing.T) {
	fallback := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	parsed := parseCMSAlertTime("2026-11-01 09:30:00", fallback)
	assert.Equal(t, "2026-11-01 09:30:00", parsed.Format("2006-01-02 15:04:05"))
	assert.Equal(t, fallback, parseCMSAlertTime("not-a-time", fallback))
	assert.Equal(t, fallback, parseCMSAlertTime("", fallback))
}
