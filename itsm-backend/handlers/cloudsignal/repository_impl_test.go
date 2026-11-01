package cloudsignal

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"itsm-backend/connector/alert"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock, func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

var t0 = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func TestUpsertFindingReturnsInsertedFlag(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewFindingRepository(db)
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	lastSeen := now.Add(-time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO security_findings")).
		WithArgs(int(7), int64(11), "aliyun", "cve", "AVD-1", "openssl", "", "high", "open", "asap",
			"i-bp1", "web-1", "uuid-1", "1.2.3.4", "10.0.0.1", "cn-hangzhou",
			nil, `{"cvss_score":7.5}`, lastSeen, lastSeen, now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "inserted"}).AddRow(int64(42), true))

	id, created, err := repo.Upsert(context.Background(), 7, &SecurityFinding{
		CloudAccountID: 11, Provider: "aliyun", VulnType: "cve", FindingKey: "AVD-1",
		Name: "openssl", Severity: "high", Status: "open", Necessity: "asap",
		InstanceID: "i-bp1", InstanceName: "web-1", UUID: "uuid-1",
		InternetIP: "1.2.3.4", IntranetIP: "10.0.0.1", Region: "cn-hangzhou",
		RawPayload:  json.RawMessage(`{"cvss_score":7.5}`),
		FirstSeenAt: &lastSeen, LastSeenAt: &lastSeen, LastSyncedAt: now,
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
	require.True(t, created)
}

func TestListFindingsKeepsTenantPredicateAndPagination(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewFindingRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*)`)).
		WithArgs(int(9), sqlmock.AnyArg(), sqlmock.AnyArg(), "%web%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("LIMIT $5 OFFSET $6")).
		WithArgs(int(9), sqlmock.AnyArg(), sqlmock.AnyArg(), "%web%", 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "cloud_account_id", "provider", "vuln_type", "finding_key",
			"name", "alias_name", "severity", "status", "necessity",
			"instance_id", "instance_name", "uuid", "internet_ip", "intranet_ip", "region",
			"ci_id", "ci_number", "ci_name", "cvss_score", "raw_payload",
			"first_seen_at", "last_seen_at", "last_synced_at", "resolved_at", "created_at", "updated_at",
		}).AddRow(int64(1), int(9), int64(11), "aliyun", "cve", "AVD-1",
			"openssl", "", "high", "open", "asap",
			"i-bp1", "web-1", "uuid-1", "1.2.3.4", "10.0.0.1", "cn-hangzhou",
			int64(3), "CI-0003", "web-1", 7.5, []byte(`{}`),
			t0, t0, t0, nil, t0, t0))

	items, total, err := repo.List(context.Background(), 9, FindingFilter{
		Statuses: []string{"open"}, Severities: []string{"high"}, Search: "web", PageSize: 50,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "web-1", items[0].CIName)
	require.NotNil(t, items[0].CIID)
}

func TestBuildFindingWhereReusesSearchPlaceholder(t *testing.T) {
	where, args := buildFindingWhere(3, FindingFilter{Statuses: []string{"open"}, Search: "100%"})
	require.NotContains(t, where, "%!d")
	require.Len(t, args, 3)
	require.Equal(t, int(3), args[0])
	require.Equal(t, "%100\\%%", args[2])
	require.Contains(t, where, "f.name ILIKE $3")
	require.Contains(t, where, "f.instance_id ILIKE $3")
}

func TestMarkSyncFailureIncrementsConsecutiveFailures(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewSyncStateRepository(db)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta("ON CONFLICT (tenant_id, cloud_account_id, signal_kind) DO UPDATE")).
		WithArgs(int(5), int64(12), "alarm", SyncStateUnavailable, now, "AccessDenied", "ak disabled").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.MarkFailure(context.Background(), 5, 12, SignalKindAlarm,
		SyncStateUnavailable, "AccessDenied", "ak disabled", now))
}

func TestClaimBridgeCandidatesOnlyClaimsP1P2Firing(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewAlertRepository(db)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	stale := now.Add(-10 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta("FOR UPDATE SKIP LOCKED")).
		WithArgs(int(5), AlertSourceCloudMonitor, sqlmock.AnyArg(), stale, now, 3, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "external_alert_id", "name", "description", "severity", "status",
			"labels", "annotations", "source_ip", "service", "fired_at", "acknowledged_at",
			"resolved_at", "created_at", "updated_at",
		}).AddRow(int64(1), int(5), "log-1", "cpu high", "cpu above 90%", "P1", "firing",
			[]byte(`{"instanceId":"i-bp1"}`), []byte(`{"bridgeStatus":"claimed","bridgeAttempts":"1"}`),
			"", "", now, nil, nil, now, now))

	candidates, err := repo.ClaimBridgeCandidates(context.Background(), 5,
		[]string{"P1", "P2"}, 10, 3, stale, now)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "claimed", candidates[0].Annotations["bridgeStatus"])
	require.Equal(t, "i-bp1", candidates[0].Labels["instanceId"])
}

func TestMarkBridgedWritesIncidentID(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewAlertRepository(db)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta("'bridgeStatus', 'bridged'")).
		WithArgs(int(5), int64(1), int64(88), now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.MarkBridged(context.Background(), 5, 1, 88, now))
}

func TestUpsertAlertNormalizesEmptyCollections(t *testing.T) {
	db, mock, done := newMock(t)
	defer done()

	repo := NewAlertRepository(db)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	fired := now.Add(-time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO alerts")).
		WithArgs(int(5), AlertSourceCloudMonitor, "log-1", "", "cpu high", "", "P1", "firing",
			"{}", "{}", "", "", "[]", fired, "{}", now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "inserted"}).AddRow(int64(9), false))

	id, created, err := repo.Upsert(context.Background(), 5, &alert.StandardAlert{
		AlertID: "log-1", Name: "cpu high", Severity: alert.SeverityP1, Status: "firing", FiredAt: fired,
	}, now)
	require.NoError(t, err)
	require.Equal(t, int64(9), id)
	require.False(t, created)
}

func TestDecodeStringMapToleratesNonStringValues(t *testing.T) {
	got := decodeStringMap([]byte(`{"bridgeAttempts":2,"region":"cn-hangzhou"}`))
	require.Equal(t, "2", got["bridgeAttempts"])
	require.Equal(t, "cn-hangzhou", got["region"])
	require.Empty(t, decodeStringMap(nil))
	require.Empty(t, decodeStringMap([]byte("not-json")))
}

func TestTruncateRunesDoesNotSplitUTF8(t *testing.T) {
	value := strings.Repeat("云", 5)
	require.Equal(t, "云云云", truncateRunes(value, 3))
	require.Equal(t, "abc", truncateRunes("abc", 3))
}
