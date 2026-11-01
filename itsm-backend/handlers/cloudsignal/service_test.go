package cloudsignal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceListAlertsNormalizesPagingAndPassesFilter(t *testing.T) {
	repo := &fakeAlertRepository{listRecords: []*AlertRecord{{ID: 1}}, listTotal: 1}
	svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

	records, total, err := svc.ListAlerts(context.Background(), 3, AlertFilter{Statuses: []string{AlertStatusFiring}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, defaultPageSize, repo.listFilter.PageSize)
	assert.Equal(t, 1, repo.listFilter.Page)
}

func TestServiceListAlertsRejectsUnknownFilterValues(t *testing.T) {
	repo := &fakeAlertRepository{}
	svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

	cases := map[string]AlertFilter{
		"unknown status":   {Statuses: []string{"exploded"}},
		"unknown severity": {Severities: []string{"P9"}},
		"negative account": {CloudAccountID: -1},
		"long search":      {Search: strings.Repeat("a", maxSearchRunes+1)},
	}
	for name, filter := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := svc.ListAlerts(context.Background(), 1, filter)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrSignalInvalidFilter), "want invalid filter, got %v", err)
		})
	}
	assert.Nil(t, repo.listRecords)
}

func TestServiceAcknowledgeAlert(t *testing.T) {
	base := &AlertRecord{ID: 5, Status: AlertStatusFiring}

	t.Run("firing becomes acknowledged", func(t *testing.T) {
		repo := &fakeAlertRepository{record: cloneAlert(base), ackOK: true}
		svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

		record, err := svc.AcknowledgeAlert(context.Background(), 1, 5)
		require.NoError(t, err)
		assert.Equal(t, AlertStatusAcknowledged, record.Status)
		assert.NotNil(t, record.AcknowledgedAt)
		assert.Equal(t, 1, repo.ackCalls)
	})

	t.Run("non-firing is a conflict and never touches the write", func(t *testing.T) {
		repo := &fakeAlertRepository{record: &AlertRecord{ID: 5, Status: AlertStatusAcknowledged}, ackOK: true}
		svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

		_, err := svc.AcknowledgeAlert(context.Background(), 1, 5)
		assert.True(t, errors.Is(err, ErrSignalStateConflict))
		assert.Zero(t, repo.ackCalls)
	})

	t.Run("missing alert is not found", func(t *testing.T) {
		svc := NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)
		_, err := svc.AcknowledgeAlert(context.Background(), 1, 5)
		assert.True(t, errors.Is(err, ErrSignalNotFound))
	})

	t.Run("lost race on conditional update is a conflict", func(t *testing.T) {
		repo := &fakeAlertRepository{record: cloneAlert(base), ackOK: false}
		svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

		_, err := svc.AcknowledgeAlert(context.Background(), 1, 5)
		assert.True(t, errors.Is(err, ErrSignalStateConflict))
	})

	t.Run("repository error is propagated", func(t *testing.T) {
		want := errors.New("db down")
		svc := NewService(&fakeAlertRepository{getErr: want}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)
		_, err := svc.AcknowledgeAlert(context.Background(), 1, 5)
		assert.ErrorIs(t, err, want)
	})

	t.Run("non-positive id is an invalid filter", func(t *testing.T) {
		svc := NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)
		_, err := svc.AcknowledgeAlert(context.Background(), 1, 0)
		assert.True(t, errors.Is(err, ErrSignalInvalidFilter))
	})
}

func TestServiceResolveAlert(t *testing.T) {
	t.Run("acknowledged can be resolved", func(t *testing.T) {
		repo := &fakeAlertRepository{record: &AlertRecord{ID: 7, Status: AlertStatusAcknowledged}, resolveOK: true}
		svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

		record, err := svc.ResolveAlert(context.Background(), 1, 7)
		require.NoError(t, err)
		assert.Equal(t, AlertStatusResolved, record.Status)
		assert.NotNil(t, record.ResolvedAt)
	})

	t.Run("already resolved is a conflict", func(t *testing.T) {
		repo := &fakeAlertRepository{record: &AlertRecord{ID: 7, Status: AlertStatusResolved}, resolveOK: true}
		svc := NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)

		_, err := svc.ResolveAlert(context.Background(), 1, 7)
		assert.True(t, errors.Is(err, ErrSignalStateConflict))
		assert.Zero(t, repo.resolveCalls)
	})
}

func TestServiceIgnoreFinding(t *testing.T) {
	t.Run("open finding becomes ignored", func(t *testing.T) {
		repo := &fakeFindingRepository{
			record: &SecurityFinding{ID: 9, Status: FindingStatusOpen},
			setOK:  true,
		}
		svc := NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil)

		finding, err := svc.IgnoreFinding(context.Background(), 1, 9)
		require.NoError(t, err)
		assert.Equal(t, FindingStatusIgnored, finding.Status)
		assert.NotNil(t, finding.ResolvedAt)
		assert.Equal(t, FindingStatusIgnored, repo.setStatus)
	})

	t.Run("fixed finding cannot be ignored again", func(t *testing.T) {
		repo := &fakeFindingRepository{
			record: &SecurityFinding{ID: 9, Status: FindingStatusFixed},
			setOK:  true,
		}
		svc := NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil)

		_, err := svc.IgnoreFinding(context.Background(), 1, 9)
		assert.True(t, errors.Is(err, ErrSignalStateConflict))
		assert.Zero(t, repo.setCalls)
	})
}

func TestServiceReopenFinding(t *testing.T) {
	t.Run("ignored finding returns to open", func(t *testing.T) {
		resolved := time.Now().Add(-time.Hour)
		repo := &fakeFindingRepository{
			record: &SecurityFinding{ID: 11, Status: FindingStatusIgnored, ResolvedAt: &resolved},
			setOK:  true,
		}
		svc := NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil)

		finding, err := svc.ReopenFinding(context.Background(), 1, 11)
		require.NoError(t, err)
		assert.Equal(t, FindingStatusOpen, finding.Status)
		assert.Nil(t, finding.ResolvedAt)
		assert.Equal(t, FindingStatusOpen, repo.setStatus)
	})

	t.Run("active finding cannot be reopened", func(t *testing.T) {
		repo := &fakeFindingRepository{
			record: &SecurityFinding{ID: 11, Status: FindingStatusFixing},
			setOK:  true,
		}
		svc := NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil)

		_, err := svc.ReopenFinding(context.Background(), 1, 11)
		assert.True(t, errors.Is(err, ErrSignalStateConflict))
		assert.Zero(t, repo.setCalls)
	})

	t.Run("missing finding is not found", func(t *testing.T) {
		svc := NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil)
		_, err := svc.ReopenFinding(context.Background(), 1, 11)
		assert.True(t, errors.Is(err, ErrSignalNotFound))
	})
}

func TestServiceListFindingsValidatesFilter(t *testing.T) {
	repo := &fakeFindingRepository{listRecords: []*SecurityFinding{{ID: 2}}, listTotal: 1}
	svc := NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil)

	records, total, err := svc.ListFindings(context.Background(), 1, FindingFilter{VulnType: "cve", Severities: []string{SeverityCritical}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, defaultPageSize, repo.listFilter.PageSize)

	for name, filter := range map[string]FindingFilter{
		"unknown status":   {Statuses: []string{"wontfix"}},
		"unknown type":     {VulnType: "ransomware"},
		"unknown severity": {Severities: []string{"urgent"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := svc.ListFindings(context.Background(), 1, filter)
			assert.True(t, errors.Is(err, ErrSignalInvalidFilter))
		})
	}
}

func TestServiceSyncStatusMergesBothKinds(t *testing.T) {
	states := &fakeSyncStateRepository{states: map[SignalKind][]*SyncState{
		SignalKindAlarm:         {{CloudAccountID: 1, SignalKind: SignalKindAlarm, State: SyncStateReady}},
		SignalKindVulnerability: {{CloudAccountID: 1, SignalKind: SignalKindVulnerability, State: SyncStateUnconfigured}},
	}}
	svc := NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, states, nil)

	got, err := svc.SyncStatus(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, SignalKindAlarm, got[0].SignalKind)
	assert.Equal(t, SignalKindVulnerability, got[1].SignalKind)
}

func TestServiceSyncStatusPropagatesError(t *testing.T) {
	want := errors.New("db down")
	svc := NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{listErr: want}, nil)

	_, err := svc.SyncStatus(context.Background(), 1)
	assert.ErrorIs(t, err, want)
}

func TestServiceRequiresDependencies(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	_, _, err := svc.ListAlerts(context.Background(), 1, AlertFilter{})
	assert.Error(t, err)
	_, _, err = svc.ListFindings(context.Background(), 1, FindingFilter{})
	assert.Error(t, err)
	_, err = svc.SyncStatus(context.Background(), 1)
	assert.Error(t, err)
}

func cloneAlert(record *AlertRecord) *AlertRecord {
	clone := *record
	return &clone
}
