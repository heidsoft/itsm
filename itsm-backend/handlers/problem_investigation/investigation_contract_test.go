package problem_investigation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// investigationSchemaDDL 是 migrations/20261018_create_problem_investigation_tables.sql
// 的 sqlite 方言副本：迁移里的 BIGSERIAL / TIMESTAMPTZ / DEFAULT NOW() 在 sqlite 无法执行，
// 但列集合、可空性与 problem_id 的 UNIQUE 必须与迁移保持一致，否则下面两条回归锁
// 会锁在一份和线上不同的 schema 上。
var investigationSchemaDDL = []string{
	`CREATE TABLE IF NOT EXISTS problem_investigations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		problem_id INTEGER NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
		investigator_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'in_progress',
		start_date TIMESTAMP NULL,
		estimated_completion_date TIMESTAMP NULL,
		actual_completion_date TIMESTAMP NULL,
		investigation_summary TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (problem_id)
	)`,
	`CREATE TABLE IF NOT EXISTS problem_investigation_steps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		investigation_id INTEGER NOT NULL REFERENCES problem_investigations (id) ON DELETE CASCADE,
		step_number INTEGER NOT NULL,
		step_title TEXT NOT NULL,
		step_description TEXT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		assigned_to INTEGER NULL,
		start_date TIMESTAMP NULL,
		completion_date TIMESTAMP NULL,
		notes TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (investigation_id, step_number)
	)`,
	`CREATE TABLE IF NOT EXISTS problem_root_cause_analyses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		problem_id INTEGER NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
		analyst_id INTEGER NOT NULL,
		analysis_method TEXT NOT NULL,
		root_cause_description TEXT NOT NULL,
		contributing_factors TEXT NULL,
		evidence TEXT NULL,
		confidence_level REAL NULL,
		analysis_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		reviewed_by INTEGER NULL,
		review_date TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (problem_id)
	)`,
	`CREATE TABLE IF NOT EXISTS problem_solutions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		problem_id INTEGER NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
		solution_type TEXT NOT NULL,
		solution_description TEXT NOT NULL,
		proposed_by INTEGER NOT NULL,
		proposed_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status TEXT NOT NULL DEFAULT 'proposed',
		priority TEXT NOT NULL DEFAULT 'medium',
		estimated_effort_hours INTEGER NULL,
		estimated_cost REAL NULL,
		risk_assessment TEXT NULL,
		approval_status TEXT NOT NULL DEFAULT 'pending',
		approved_by INTEGER NULL,
		approval_date TIMESTAMP NULL,
		implementation_plan TEXT NULL,
		planned_start_date TIMESTAMP NULL,
		planned_end_date TIMESTAMP NULL,
		actual_start_date TIMESTAMP NULL,
		actual_end_date TIMESTAMP NULL,
		implementation_status TEXT NULL,
		implementation_result TEXT NULL,
		verified_by INTEGER NULL,
		verification_date TIMESTAMP NULL,
		verification_result TEXT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
}

type investigationFixture struct {
	router   *gin.Engine
	tenantA  int
	tenantB  int
	userA    int
	problemA int
}

// newInvestigationFixture 起一套真实调用链：route -> handler -> service -> repository。
// 原 handler_test.go 的 DSN 没有 cache=shared，raw sql.DB 与 enttest 客户端是两个不同的
// 内存库，任何触达 problem_investigations 的断言都只是在断「表不存在」，所以这两类缺陷
// （NULL 扫描 500、重复创建 500）当时测不出来。
func newInvestigationFixture(t *testing.T) *investigationFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:pi-contract-%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, "sqlite3", dsn)
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range investigationSchemaDDL {
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			t.Fatalf("apply investigation DDL: %v", err)
		}
	}

	ctx := context.Background()
	tenantA := client.Tenant.Create().SetName("调查租户A").SetCode("pi-contract-a").
		SetDomain("pi-a.test").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("调查租户B").SetCode("pi-contract-b").
		SetDomain("pi-b.test").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("pi-investigator").SetEmail("inv@pi-a.test").
		SetName("调查员甲").SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)
	problemA := client.Problem.Create().SetTitle("DNS 反复解析失败").SetProblemNumber("PRB-PI-1").
		SetStatus("open").SetCreatedBy(userA.ID).SetTenantID(tenantA.ID).SaveX(ctx)

	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(logger, service.NewProblemInvestigationService(db, client, logger))

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		tid := tenantA.ID
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tid = v
			}
		}
		c.Set("tenant_id", tid)
		c.Set("user_id", userA.ID)
		c.Next()
	})

	r.POST("/api/v1/problem-investigation/investigations", h.CreateProblemInvestigation)
	r.GET("/api/v1/problem-investigation/investigations/:id", h.GetProblemInvestigation)
	r.GET("/api/v1/problem-investigation/problems/:id/summary", h.GetProblemInvestigationSummary)

	return &investigationFixture{
		router:   r,
		tenantA:  tenantA.ID,
		tenantB:  tenantB.ID,
		userA:    userA.ID,
		problemA: problemA.ID,
	}
}

func (f *investigationFixture) do(t *testing.T, method, path string, body []byte, tenantID int) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, path, reqBody)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Test-Tenant", fmt.Sprintf("%d", tenantID))

	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

// TestCreateInvestigation_ReadsBackNullStartDate 回归锁（2026-10-18 生产验证）：
// problem_investigations.start_date 允许为 NULL，创建时也不写这一列。
// 此前 DTO 用值类型 time.Time 接收，Scan 直接失败
// （unsupported Scan, storing driver.Value type <nil> into type *time.Time），
// 于是 GET /investigations/:id 与 GET /problems/:id/summary 全部 500。
// 创建响应还必须回读库里的权威行，不能在内存里伪造 startDate。
func TestCreateInvestigation_ReadsBackNullStartDate(t *testing.T) {
	f := newInvestigationFixture(t)

	createBody := []byte(fmt.Sprintf(`{"problemId":%d,"investigatorId":%d,"investigationSummary":"复现路径：DNS 抖动"}`, f.problemA, f.userA))
	w := f.do(t, http.MethodPost, "/api/v1/problem-investigation/investigations", createBody, f.tenantA)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var createResp struct {
		Code int `json:"code"`
		Data struct {
			InvestigationID int                              `json:"investigationId"`
			Investigation   dto.ProblemInvestigationResponse `json:"investigation"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	assert.Equal(t, 0, createResp.Code)
	require.Greater(t, createResp.Data.InvestigationID, 0)
	assert.Equal(t, f.problemA, createResp.Data.Investigation.ProblemID)
	assert.Nil(t, createResp.Data.Investigation.StartDate, "start_date 未写库时应返回 null，不得伪造")
	assert.Equal(t, "调查员甲", createResp.Data.Investigation.InvestigatorName)

	w2 := f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/problem-investigation/investigations/%d", createResp.Data.InvestigationID), nil, f.tenantA)
	require.Equal(t, http.StatusOK, w2.Code, "body=%s", w2.Body.String())

	var detail dto.ProblemInvestigationResponse
	require.NoError(t, json.Unmarshal(mustData(t, w2), &detail))
	assert.Nil(t, detail.StartDate)
	assert.Equal(t, dto.InvestigationStatusInProgress, detail.Status)

	// 摘要接口同样经历 start_date，是最先在线上炸的那条路径。
	w3 := f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/problem-investigation/problems/%d/summary", f.problemA), nil, f.tenantA)
	require.Equal(t, http.StatusOK, w3.Code, "body=%s", w3.Body.String())
	var summary dto.ProblemInvestigationSummaryResponse
	require.NoError(t, json.Unmarshal(mustData(t, w3), &summary))
	require.NotNil(t, summary.Investigation)
	assert.Nil(t, summary.Investigation.StartDate)
	assert.NotNil(t, summary.Steps)
	assert.NotNil(t, summary.Solutions)
}

// TestCreateInvestigation_DuplicateIsConflict 回归锁：一个 problem 只允许一条调查记录。
// 重复创建属于业务冲突，必须是 409/4090 + 领域文案，不能兜底成 500「操作失败」。
func TestCreateInvestigation_DuplicateIsConflict(t *testing.T) {
	f := newInvestigationFixture(t)

	body := []byte(fmt.Sprintf(`{"problemId":%d,"investigatorId":%d}`, f.problemA, f.userA))
	first := f.do(t, http.MethodPost, "/api/v1/problem-investigation/investigations", body, f.tenantA)
	require.Equal(t, http.StatusOK, first.Code, "body=%s", first.Body.String())

	second := f.do(t, http.MethodPost, "/api/v1/problem-investigation/investigations", body, f.tenantA)
	assert.Equal(t, http.StatusConflict, second.Code, "body=%s", second.Body.String())

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &resp))
	assert.Equal(t, common.ConflictCode, resp.Code)
	assert.Equal(t, "该问题已存在调查记录", resp.Message)
	assert.NotContains(t, second.Body.String(), "pq:", "响应不得泄漏驱动层错误")
}

// TestGetInvestigationSummary_CrossTenantRejected 跨租户读取必须 fail closed：
// 租户 B 用自己的 ID 去查租户 A 的问题摘要，只能得到 404，不能拿到数据。
func TestGetInvestigationSummary_CrossTenantRejected(t *testing.T) {
	f := newInvestigationFixture(t)

	body := []byte(fmt.Sprintf(`{"problemId":%d,"investigatorId":%d}`, f.problemA, f.userA))
	require.Equal(t, http.StatusOK,
		f.do(t, http.MethodPost, "/api/v1/problem-investigation/investigations", body, f.tenantA).Code)

	w := f.do(t, http.MethodGet, fmt.Sprintf("/api/v1/problem-investigation/problems/%d/summary", f.problemA), nil, f.tenantB)
	assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code    int                                      `json:"code"`
		Message string                                   `json:"message"`
		Data    *dto.ProblemInvestigationSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.NotFoundCode, resp.Code)
	assert.Equal(t, "问题不存在", resp.Message)
	assert.Nil(t, resp.Data, "跨租户响应不得携带任何调查数据")
}

func mustData(t *testing.T, w *httptest.ResponseRecorder) []byte {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "body=%s", w.Body.String())
	return envelope.Data
}
