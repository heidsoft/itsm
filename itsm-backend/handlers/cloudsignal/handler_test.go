package cloudsignal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/middleware"
)

type testEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type testListData struct {
	Items      []json.RawMessage `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

func newTestRouter(t *testing.T, svc *Service) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 7})
		c.Set("tenant_id", 7)
		c.Next()
	})
	r.GET("/cloud-signals/alerts", h.ListAlerts)
	r.POST("/cloud-signals/alerts/:id/acknowledge", h.AcknowledgeAlert)
	r.POST("/cloud-signals/alerts/:id/resolve", h.ResolveAlert)
	r.GET("/cloud-signals/vulnerabilities", h.ListFindings)
	r.GET("/cloud-signals/vulnerabilities/summary", h.FindingSummary)
	r.POST("/cloud-signals/vulnerabilities/:id/ignore", h.IgnoreFinding)
	r.POST("/cloud-signals/vulnerabilities/:id/reopen", h.ReopenFinding)
	r.GET("/cloud-signals/sync-status", h.SyncStatus)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path string) (*httptest.ResponseRecorder, testEnvelope) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(rec, req)
	var envelope testEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "body: %s", rec.Body.String())
	return rec, envelope
}

func TestHandlerListAlertsReturnsCanonicalEnvelope(t *testing.T) {
	now := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	repo := &fakeAlertRepository{
		listRecords: []*AlertRecord{{
			ID: 13, AlertID: "cm-alert-1", Name: "ECS CPU 持续高于 90%",
			Severity: "P1", Status: AlertStatusFiring, FiredAt: now,
			Labels: map[string]string{"cloudAccountId": "3"},
		}},
		listTotal: 25,
	}
	r := newTestRouter(t, NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))

	rec, envelope := doRequest(t, r, http.MethodGet, "/cloud-signals/alerts?page=2&pageSize=10")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, common.SuccessCode, envelope.Code)

	var data testListData
	require.NoError(t, json.Unmarshal(envelope.Data, &data))
	assert.Equal(t, int64(25), data.Total)
	assert.Equal(t, 2, data.Page)
	assert.Equal(t, 10, data.PageSize)
	assert.Equal(t, 3, data.TotalPages)
	require.Len(t, data.Items, 1)

	var item map[string]any
	require.NoError(t, json.Unmarshal(data.Items[0], &item))
	assert.Equal(t, "cm-alert-1", item["alertId"])
	assert.NotContains(t, item, "alert_id")
	assert.Equal(t, "firing", item["status"])
	assert.Equal(t, 2, repo.listFilter.Page)
	assert.Equal(t, 10, repo.listFilter.PageSize)
}

func TestHandlerListAlertsRejectsUnknownStatus(t *testing.T) {
	r := newTestRouter(t, NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))

	rec, envelope := doRequest(t, r, http.MethodGet, "/cloud-signals/alerts?status=exploded")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, common.ParamErrorCode, envelope.Code)
}

func TestHandlerAcknowledgeAlertMapsState(t *testing.T) {
	t.Run("missing alert is 404", func(t *testing.T) {
		r := newTestRouter(t, NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))
		rec, envelope := doRequest(t, r, http.MethodPost, "/cloud-signals/alerts/5/acknowledge")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, common.NotFoundCode, envelope.Code)
	})

	t.Run("stale state is 409", func(t *testing.T) {
		repo := &fakeAlertRepository{record: &AlertRecord{ID: 5, Status: AlertStatusResolved}}
		r := newTestRouter(t, NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))
		rec, envelope := doRequest(t, r, http.MethodPost, "/cloud-signals/alerts/5/acknowledge")
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, common.ConflictCode, envelope.Code)
	})

	t.Run("invalid id is 400", func(t *testing.T) {
		r := newTestRouter(t, NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))
		rec, envelope := doRequest(t, r, http.MethodPost, "/cloud-signals/alerts/abc/acknowledge")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, common.ParamErrorCode, envelope.Code)
	})

	t.Run("firing alert returns the updated record", func(t *testing.T) {
		repo := &fakeAlertRepository{record: &AlertRecord{ID: 5, Status: AlertStatusFiring}, ackOK: true}
		r := newTestRouter(t, NewService(repo, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))
		rec, envelope := doRequest(t, r, http.MethodPost, "/cloud-signals/alerts/5/acknowledge")
		require.Equal(t, http.StatusOK, rec.Code)
		var payload AlertResponse
		require.NoError(t, json.Unmarshal(envelope.Data, &payload))
		assert.Equal(t, AlertStatusAcknowledged, payload.Status)
		assert.NotNil(t, payload.AcknowledgedAt)
	})
}

func TestHandlerListFindingsUsesCamelCase(t *testing.T) {
	now := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	score := 9.8
	repo := &fakeFindingRepository{
		listRecords: []*SecurityFinding{{
			ID: 21, CloudAccountID: 3, Provider: "aliyun", VulnType: "cve",
			FindingKey: "cve-2026-1", Name: "OpenSSL 高危漏洞", Severity: SeverityCritical,
			Status: FindingStatusOpen, InstanceID: "i-1", CVSSScore: &score, LastSyncedAt: now,
		}},
		listTotal: 1,
	}
	r := newTestRouter(t, NewService(&fakeAlertRepository{}, repo, &fakeSyncStateRepository{}, nil))

	rec, envelope := doRequest(t, r, http.MethodGet, "/cloud-signals/vulnerabilities?vulnType=cve&severity=critical")
	require.Equal(t, http.StatusOK, rec.Code)

	var data testListData
	require.NoError(t, json.Unmarshal(envelope.Data, &data))
	require.Len(t, data.Items, 1)

	var item map[string]any
	require.NoError(t, json.Unmarshal(data.Items[0], &item))
	assert.Equal(t, "cve", item["vulnType"])
	assert.Equal(t, float64(3), item["cloudAccountId"])
	assert.Equal(t, float64(9.8), item["cvssScore"])
	assert.Equal(t, "i-1", item["instanceId"])
	for _, key := range []string{"vuln_type", "cloud_account_id", "cvss_score", "instance_id"} {
		assert.NotContains(t, item, key)
	}
	assert.Equal(t, "cve", repo.listFilter.VulnType)
	assert.Equal(t, []string{SeverityCritical}, repo.listFilter.Severities)
}

func TestHandlerFindingSummaryDefaultsEmptyMaps(t *testing.T) {
	r := newTestRouter(t, NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))

	rec, envelope := doRequest(t, r, http.MethodGet, "/cloud-signals/vulnerabilities/summary")
	require.Equal(t, http.StatusOK, rec.Code)

	var summary FindingSummary
	require.NoError(t, json.Unmarshal(envelope.Data, &summary))
	assert.Zero(t, summary.Total)
	assert.NotNil(t, summary.BySeverity)
	assert.NotNil(t, summary.ByStatus)
}

func TestHandlerSyncStatusListsBothKinds(t *testing.T) {
	states := &fakeSyncStateRepository{states: map[SignalKind][]*SyncState{
		SignalKindAlarm: {{CloudAccountID: 3, AccountName: "生产账号", Provider: "aliyun",
			SignalKind: SignalKindAlarm, State: SyncStateReady, ItemsSynced: 12}},
		SignalKindVulnerability: {{CloudAccountID: 3, AccountName: "生产账号", Provider: "aliyun",
			SignalKind: SignalKindVulnerability, State: SyncStateUnconfigured}},
	}}
	r := newTestRouter(t, NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, states, nil))

	rec, envelope := doRequest(t, r, http.MethodGet, "/cloud-signals/sync-status")
	require.Equal(t, http.StatusOK, rec.Code)

	var data testListData
	require.NoError(t, json.Unmarshal(envelope.Data, &data))
	assert.Equal(t, int64(2), data.Total)
	require.Len(t, data.Items, 2)

	var first SyncStateResponse
	require.NoError(t, json.Unmarshal(data.Items[0], &first))
	assert.Equal(t, "alarm", first.SignalKind)
	assert.Equal(t, SyncStateReady, first.State)
	assert.Equal(t, "生产账号", first.AccountName)
	assert.Equal(t, int64(3), first.CloudAccountID)
}

func TestHandlerRequiresTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(&fakeAlertRepository{}, &fakeFindingRepository{}, &fakeSyncStateRepository{}, nil))
	r := gin.New()
	r.GET("/cloud-signals/alerts", h.ListAlerts)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cloud-signals/alerts", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
