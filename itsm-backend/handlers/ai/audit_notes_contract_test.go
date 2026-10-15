package ai_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/handlers/ai"
	"itsm-backend/service"
)

// newAuditNotesDB 建 ai_feedbacks 表，写读共用同一条连接，
// 这样 POST /api/v1/ai/audit 落库的 notes 能被审计列表查询原样读回。
func newAuditNotesDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_feedbacks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			tenant_id INT NOT NULL,
			user_id INT NOT NULL,
			request_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			query TEXT,
			item_type TEXT,
			item_id INT,
			useful BOOLEAN NOT NULL,
			score INT,
			notes TEXT,
			accepted BOOLEAN
		);`)
	require.NoError(t, err)
	return db
}

// TestRecordAuditPromptVersionSurvivesRoundTrip 锁定 AI 审计元数据的写读契约。
//
// 修复前 handlers/ai 以 camelCase（promptVersion）落库，而 service/ai_evaluator.go
// 用 `json:"prompt_version"` 解析，导致 GET /api/v1/ai/audit-logs 的 promptVersion
// 恒为空串：AI 决策支持失去可追溯的版本证据。
// 两侧各自的单测都无法发现——写侧只断言响应回显，读侧 fixture 沿用了读侧的错误拼写。
// 本测试让生产 handler 写入、生产查询服务读回，唯一能证伪的就是标签漂移本身。
func TestRecordAuditPromptVersionSurvivesRoundTrip(t *testing.T) {
	db := newAuditNotesDB(t)
	telemetry := service.NewAITelemetryService(db)
	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, telemetry)
	h := ai.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/ai/audit", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 42)
		c.Set("request_id", "req-roundtrip")
		h.RecordAudit(c)
	})

	body := `{"scenario":"triage","inputRef":"ticket_7","promptVersion":"v2026.10-rca",` +
		`"model":"doubao-pro-32k","confidence":0.82,"suggestion":{"category":"network"},"accepted":true}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newJSONRequest(http.MethodPost, "/api/v1/ai/audit", body))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored string
	require.NoError(t, db.QueryRow(
		"SELECT notes FROM ai_feedbacks WHERE tenant_id = 1 AND item_type = 'ai_audit' ORDER BY id DESC LIMIT 1",
	).Scan(&stored))

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stored), &raw))
	assert.Equal(t, "v2026.10-rca", raw["promptVersion"], "落库键必须是 camelCase")
	assert.NotContains(t, raw, "prompt_version", "禁止 snake_case 双轨")

	entries, total, err := telemetry.ListAuditLogs(context.Background(), 1, 1, 20, "", 90)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, "v2026.10-rca", entries[0].PromptVersion,
		"审计列表必须回显写入时的 prompt 版本")
	assert.Equal(t, "doubao-pro-32k", entries[0].Model)
	assert.Equal(t, "network", entries[0].Suggestion["category"])
}

func newJSONRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
