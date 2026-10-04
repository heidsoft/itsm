package contract

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	aidomain "itsm-backend/handlers/ai"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// GET /api/v1/ai/audit-logs 的页长此前有三处真相：handler 用私有 queryInt 且不给上界、
// service 里再写一份 auditMaxPageSize=200 与缺省 20、响应又只回四键并回显未采纳的原值。
// 本文件把「单一所有者 + 五键信封 + 回显等于实际采纳值」锁在真实注册的那条路由上。

const aiAuditCreateSQL = `
	CREATE TABLE IF NOT EXISTS ai_feedbacks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP,
		tenant_id INT NOT NULL,
		user_id INT NOT NULL,
		request_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		query TEXT,
		item_type TEXT,
		item_id INT,
		useful BOOLEAN NOT NULL,
		score INT,
		notes TEXT
	);`

func newAuditLogsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	// 内存库只在至少一条连接存活期间存在，单连接同时保证读己写。
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(aiAuditCreateSQL)
	require.NoError(t, err)
	return db
}

// newAuditLogsRouter 复现 router/ai_routes.go:37 的注册（permission 中间件不在本信封契约范围内），
// 并复现 internal/bootstrap/app.go:944 的装配：本端点只用到 aiTelemetryService。
func newAuditLogsRouter(t *testing.T, db *sql.DB, tenantID int, withTenant bool) *gin.Engine {
	t.Helper()

	telemetry := service.NewAITelemetryService(db)
	svc := aidomain.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, telemetry)
	h := aidomain.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if withTenant {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
			c.Set("tenant_id", tenantID)
			c.Set("user_id", 1)
			c.Set("role", "admin")
		}
		c.Next()
	})
	r.GET("/api/v1/ai/audit-logs", h.GetAuditLogs)
	return r
}

func seedAuditFeedbacks(t *testing.T, db *sql.DB, tenantID int, prefix string, n int) {
	t.Helper()
	now := time.Now()
	for i := 0; i < n; i++ {
		_, err := db.Exec(
			`INSERT INTO ai_feedbacks (created_at, tenant_id, user_id, request_id, kind, query, item_type, useful, score, notes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			now.Add(time.Duration(i)*time.Minute), tenantID, 1,
			fmt.Sprintf("%s-%d", prefix, i), "triage", fmt.Sprintf("%s-Q%d", prefix, i),
			"ai_audit", true, 80, "",
		)
		require.NoError(t, err)
	}
}

func doAuditLogs(t *testing.T, r *gin.Engine, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ai/audit-logs?"+query, nil))
	return w
}

func envelopeKeysOf(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

func decodeAuditEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, float64(0), body["code"], body["message"])
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是列表信封，实际 %v", body)
	return data
}

func TestAIAuditLogs_EnvelopeIsExactlyFiveKeys(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)

	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), "page=1"))
	assert.ElementsMatch(t, []string{"items", "total", "page", "pageSize", "totalPages"}, envelopeKeysOf(data))
}

func TestAIAuditLogs_DefaultPageSizeIsPlatformTwenty(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)

	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), ""))
	assert.Len(t, data["items"], 20, "缺省页长必须是平台值 20")
	assert.Equal(t, float64(20), data["pageSize"])
	assert.Equal(t, float64(25), data["total"])
	assert.Equal(t, float64(2), data["totalPages"])
}

func TestAIAuditLogs_OversizedPageSizeCannotEscapePlatformBound(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 130)

	// 150 落在旧 service 的 200 上界之内 ⇒ 一次返回整段 130 条并回显 150；
	// 平台契约只采纳 1-100，越界回落缺省 20。
	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), "pageSize=150"))
	assert.Len(t, data["items"], 20)
	assert.Equal(t, float64(20), data["pageSize"], "信封声明的页长必须等于数据库实际采纳的 LIMIT")
	assert.Equal(t, float64(130), data["total"])
	assert.Equal(t, float64(7), data["totalPages"])
}

func TestAIAuditLogs_EchoUsesAdoptedValuesNotClientClaims(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)

	for _, query := range []string{"pageSize=5000", "pageSize=abc", "pageSize=0", "page=-3"} {
		data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), query))
		assert.Equal(t, float64(20), data["pageSize"], query)
		assert.Equal(t, float64(1), data["page"], query)
		assert.Len(t, data["items"], 20, query)
	}
}

func TestAIAuditLogs_SecondPageReturnsRemainder(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)

	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), "page=2&pageSize=20"))
	assert.Len(t, data["items"], 5)
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(2), data["totalPages"])
}

func TestAIAuditLogs_TenantScopeIsConverged(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)
	seedAuditFeedbacks(t, db, 2, "t2", 7)

	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), "pageSize=100"))
	assert.Equal(t, float64(25), data["total"], "total 只能是本租户条数")
	require.Len(t, data["items"], 25)
	for _, raw := range data["items"].([]interface{}) {
		item := raw.(map[string]interface{})
		assert.Equal(t, float64(1), item["tenantId"], "响应不得混入其他租户的行")
		assert.Contains(t, item["requestId"], "t1-")
	}
}

func TestAIAuditLogs_MissingTenantContextAbortsOnce(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 1, "t1", 25)

	w := doAuditLogs(t, newAuditLogsRouter(t, db, 0, false), "page=1")
	require.NotEqual(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())

	// ResolveTenantID 在缺上下文时已经写过一份 401/2001；handler 再写一份就是同一响应体里
	// 塞两个 JSON 文档（Go 记 superfluous WriteHeader，客户端拿到的是拼接字节）。
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "响应体必须是单个 JSON 文档: %s", w.Body.String())
	assert.Equal(t, float64(2001), body["code"], w.Body.String())
}

func TestAIAuditLogs_EmptyResultSetSerializesAsArray(t *testing.T) {
	db := newAuditLogsTestDB(t)
	seedAuditFeedbacks(t, db, 2, "t2", 7)

	data := decodeAuditEnvelope(t, doAuditLogs(t, newAuditLogsRouter(t, db, 1, true), ""))
	assert.Equal(t, float64(0), data["total"])
	assert.Equal(t, float64(0), data["totalPages"])
	raw, err := json.Marshal(data["items"])
	require.NoError(t, err)
	assert.Equal(t, "[]", string(raw))
}
