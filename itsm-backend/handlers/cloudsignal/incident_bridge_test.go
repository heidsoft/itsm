package cloudsignal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/connector/alert"
	"itsm-backend/dto"
)

type fakeBridgeRepository struct {
	incidentByAlert map[string]int64
	incidentErr     error
	ciID            int64
	ciErr           error
	gotProvider     string
	gotResourceID   string
}

func (f *fakeBridgeRepository) FindIncidentByCloudAlertID(_ context.Context, _ int, alertKey string) (int64, error) {
	if f.incidentErr != nil {
		return 0, f.incidentErr
	}
	return f.incidentByAlert[alertKey], nil
}

func (f *fakeBridgeRepository) FindCIByCloudResource(_ context.Context, _ int, provider, resourceID string) (int64, error) {
	f.gotProvider = provider
	f.gotResourceID = resourceID
	return f.ciID, f.ciErr
}

type fakeIncidentCreator struct {
	requests   []*dto.CreateIncidentRequest
	tenantID   int
	reporterID int
	err        error
	nextID     int
}

func (f *fakeIncidentCreator) CreateAutomatedIncident(_ context.Context, req *dto.CreateIncidentRequest, tenantID, reporterID int) (*dto.IncidentResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.requests = append(f.requests, req)
	f.tenantID = tenantID
	f.reporterID = reporterID
	if f.nextID == 0 {
		f.nextID = 101
	}
	return &dto.IncidentResponse{ID: f.nextID}, nil
}

func bridgeCandidate(id int64, alertID, severity string) *BridgeCandidate {
	return &BridgeCandidate{AlertRecord: AlertRecord{
		ID:          id,
		TenantID:    7,
		AlertID:     alertID,
		Name:        "CPU 打满 - web-01",
		Description: "CPU 使用率持续超过 95%",
		Severity:    severity,
		Status:      "firing",
		Labels: map[string]string{
			"instanceId":     "i-bp1",
			"instanceName":   "web-01",
			"provider":       "aliyun",
			"ruleId":         "rule-1",
			"ruleName":       "CPU 打满",
			"metricName":     "cpu_total",
			"cloudAccountId": "11",
		},
		FiredAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
	}}
}

func newBridgeForTest(alertRepo AlertRepository, repo BridgeRepository, creator IncidentCreator, reporterID int) *IncidentBridge {
	bridge := NewIncidentBridge(alertRepo, repo, creator, reporterID, nil)
	bridge.now = func() time.Time { return time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC) }
	return bridge
}

func TestIncidentBridgeCreatesIncidentWithMetadataAndCI(t *testing.T) {
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{bridgeCandidate(5, "log-1", string(alert.SeverityP1))}}
	repo := &fakeBridgeRepository{ciID: 42}
	creator := &fakeIncidentCreator{}
	bridge := newBridgeForTest(alertRepo, repo, creator, 99)

	result, err := bridge.RunTenant(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Claimed)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, []string{string(alert.SeverityP1), string(alert.SeverityP2)}, alertRepo.claimedSevs)
	require.Len(t, creator.requests, 1)
	req := creator.requests[0]
	assert.Equal(t, "[P1] CPU 打满 - web-01", req.Title)
	assert.Equal(t, "incident", req.Type)
	assert.Equal(t, "monitoring", req.Source)
	assert.Equal(t, "critical", req.Priority)
	assert.Equal(t, []int{42}, req.ConfigurationItemIDs)
	assert.Equal(t, 7, creator.tenantID)
	assert.Equal(t, 99, creator.reporterID)
	assert.Equal(t, "log-1", req.Metadata["cloudAlertId"])
	assert.Equal(t, "aliyun", req.Metadata["cloudProvider"])
	assert.Equal(t, "i-bp1", req.Metadata["cloudInstanceId"])
	assert.Equal(t, "2026-11-01T09:00:00Z", req.Metadata["cloudAlertTime"])
	assert.Equal(t, int64(101), alertRepo.bridged[5])
	assert.Equal(t, "aliyun", repo.gotProvider)
	assert.Equal(t, "i-bp1", repo.gotResourceID)
}

func TestIncidentBridgeReusesExistingIncident(t *testing.T) {
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{bridgeCandidate(5, "log-1", string(alert.SeverityP2))}}
	repo := &fakeBridgeRepository{incidentByAlert: map[string]int64{"log-1": 77}}
	creator := &fakeIncidentCreator{}
	bridge := newBridgeForTest(alertRepo, repo, creator, 99)

	result, err := bridge.RunTenant(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Reused)
	assert.Empty(t, creator.requests)
	assert.Equal(t, int64(77), alertRepo.bridged[5])
}

func TestIncidentBridgeRecordsFailureAndContinues(t *testing.T) {
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{
		bridgeCandidate(5, "log-1", string(alert.SeverityP1)),
		bridgeCandidate(6, "log-2", string(alert.SeverityP2)),
	}}
	creator := &fakeIncidentCreator{err: errors.New("reporter user is not active")}
	bridge := newBridgeForTest(alertRepo, &fakeBridgeRepository{}, creator, 99)

	result, err := bridge.RunTenant(context.Background(), 7)
	require.Error(t, err)

	assert.Equal(t, 2, result.Failed)
	assert.Contains(t, alertRepo.failed[5], "not active")
	assert.Contains(t, alertRepo.failed[6], "not active")
	assert.Empty(t, alertRepo.bridged)
}

func TestIncidentBridgeLookupFailureDoesNotCreate(t *testing.T) {
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{bridgeCandidate(5, "log-1", string(alert.SeverityP1))}}
	repo := &fakeBridgeRepository{incidentErr: errors.New("db down")}
	creator := &fakeIncidentCreator{}
	bridge := newBridgeForTest(alertRepo, repo, creator, 99)

	result, err := bridge.RunTenant(context.Background(), 7)
	require.Error(t, err)

	assert.Equal(t, 1, result.Failed)
	assert.Empty(t, creator.requests)
	assert.Contains(t, alertRepo.failed[5], "db down")
}

func TestIncidentBridgeRequiresReporter(t *testing.T) {
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{bridgeCandidate(5, "log-1", string(alert.SeverityP1))}}
	bridge := newBridgeForTest(alertRepo, &fakeBridgeRepository{}, &fakeIncidentCreator{}, 0)

	_, err := bridge.RunTenant(context.Background(), 7)
	assert.ErrorIs(t, err, ErrBridgeReporterUnconfigured)
	assert.Empty(t, alertRepo.claimedSevs)
}

func TestIncidentBridgeSkipsCILookupWithoutInstance(t *testing.T) {
	candidate := bridgeCandidate(5, "log-1", string(alert.SeverityP1))
	delete(candidate.Labels, "instanceId")
	alertRepo := &fakeAlertRepository{candidates: []*BridgeCandidate{candidate}}
	repo := &fakeBridgeRepository{ciID: 42}
	creator := &fakeIncidentCreator{}
	bridge := newBridgeForTest(alertRepo, repo, creator, 99)

	_, err := bridge.RunTenant(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, creator.requests, 1)
	assert.Empty(t, creator.requests[0].ConfigurationItemIDs)
	assert.Empty(t, repo.gotResourceID)
}

func TestBuildIncidentRequestTruncatesTitle(t *testing.T) {
	record := AlertRecord{AlertID: "log-1", Name: strings.Repeat("长", 400), Severity: string(alert.SeverityP2)}
	req := buildIncidentRequest(&record, nil)
	assert.LessOrEqual(t, len([]rune(req.Title)), bridgeTitleMaxRunes)
	assert.Contains(t, req.Metadata, "cloudAlertId")
}
