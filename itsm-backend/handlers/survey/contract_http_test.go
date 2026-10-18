// Package survey 的 HTTP 契约回归（B10.1）。
//
// 权限码：/surveys 组挂 survey:{read,write}（router/survey_routes.go）。
//
// 锁定语义：
//   - [SV1 已修复] POST /surveys/responses 的 surveyId/ticketId 必须属于本租户：
//     他租/不存在引用 → 400/1001（此前直接落库——跨租户 survey 引用注入，
//     ticketId 同面）；respondentId 写入认证上下文真实用户（此前硬编码 0）。
//   - [SV2 已修复] GET /surveys/{他租id}/analytics 与 /responses：先验 survey
//     归属 → 404/4004（此前 200 + 空统计/空列表——存在性探测面）。
//   - [SV3 既有正确面] GET/PUT /surveys/{他租id} → ent NotFound → 404/4004；
//     列表/分析字段 camelCase。
package survey_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/surveyresponse"
	surveyHandler "itsm-backend/handlers/survey"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	endUser *ent.User
	ticketA *ent.Ticket
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:survey_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Survey A").SetCode("survey-a").SetDomain("survey-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Survey B").SetCode("survey-b").SetDomain("survey-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("survey-a-owner").SetEmail("survey-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("survey-a-enduser").SetEmail("survey-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().SetTicketNumber("SV-A-001").SetTitle("调查工单").
		SetStatus("closed").SetPriority("medium").SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "survey-contract-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:     secret,
		Logger:        logger,
		Client:        client,
		SurveyHandler: surveyHandler.NewHandler(service.NewSurveyService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, endUser: endUser, ticketA: ticketA,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, payload ...string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if len(payload) > 0 {
		reader = strings.NewReader(payload[0])
	}
	req := httptest.NewRequest(method, path, reader)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

func TestSurveyHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/surveys"
	ctx := context.Background()

	t.Run("create正控制与camelCase", func(t *testing.T) {
		payload := `{"title":"满意度问卷","surveyType":"NPS","questions":[{"question":"overall","type":"rating","required":true}],"isActive":true}`
		w := fx.do(t, http.MethodPost, base, fx.userA, payload)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		require.Contains(t, w.Body.String(), `"surveyType"`, "字段必须是 camelCase")
		require.NotContains(t, w.Body.String(), `"survey_type"`, "不得出现 snake_case 字段")
	})

	t.Run("submit跨租户survey是400掩码", func(t *testing.T) {
		surveyB := fx.client.Survey.Create().SetTitle("他租问卷").SetTenantID(fx.tenantB.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		payload := `{"surveyId":` + fmt.Sprint(surveyB.ID) + `,"ticketId":` + fmt.Sprint(fx.ticketA.ID) +
			`,"answers":[{"questionIndex":0,"value":8}],"score":8}`
		w := fx.do(t, http.MethodPost, base+"/responses", fx.userA, payload)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
		n, err := fx.client.SurveyResponse.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, n, "跨租户 survey 引用不得落库")
	})

	t.Run("submit跨租户ticket是400掩码", func(t *testing.T) {
		surveyA := fx.client.Survey.Create().SetTitle("本租问卷").SetTenantID(fx.tenantA.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		ticketB := fx.client.Ticket.Create().SetTicketNumber("SV-B-001").SetTitle("他租工单").
			SetStatus("closed").SetPriority("medium").SetRequesterID(fx.userA.ID).
			SetTenantID(fx.tenantB.ID).SaveX(ctx)
		payload := `{"surveyId":` + fmt.Sprint(surveyA.ID) + `,"ticketId":` + fmt.Sprint(ticketB.ID) +
			`,"answers":[{"questionIndex":0,"value":8}],"score":8}`
		w := fx.do(t, http.MethodPost, base+"/responses", fx.userA, payload)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
		n, err := fx.client.SurveyResponse.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, n, "跨租户 ticket 引用不得落库")
	})

	t.Run("submit正控制落库respondent为本租真实用户", func(t *testing.T) {
		surveyA := fx.client.Survey.Create().SetTitle("正控问卷").SetTenantID(fx.tenantA.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		payload := `{"surveyId":` + fmt.Sprint(surveyA.ID) + `,"ticketId":` + fmt.Sprint(fx.ticketA.ID) +
			`,"answers":[{"questionIndex":0,"value":9}],"score":9,"comment":"回复内容"}`
		w := fx.do(t, http.MethodPost, base+"/responses", fx.userA, payload)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		resp, err := fx.client.SurveyResponse.Query().
			Where(surveyresponse.TenantIDEQ(fx.tenantA.ID)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, resp, 1, "正控响应落库")
		require.Equal(t, fx.userA.ID, resp[0].RespondentID, "respondentId 必须是认证用户，不再是硬编码 0")
	})

	t.Run("analytics与responses跨租户survey是404掩码", func(t *testing.T) {
		surveyB := fx.client.Survey.Create().SetTitle("他租统计问卷").SetTenantID(fx.tenantB.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		for _, probe := range []struct{ path string }{
			{fmt.Sprintf("%s/%d/analytics", base, surveyB.ID)},
			{fmt.Sprintf("%s/%d/responses", base, surveyB.ID)},
		} {
			w := fx.do(t, http.MethodGet, probe.path, fx.userA)
			require.Equal(t, http.StatusNotFound, w.Code, "path=%s body=%s", probe.path, w.Body.String())
			require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		}
	})

	t.Run("analytics与responses正控制", func(t *testing.T) {
		surveyA := fx.client.Survey.Create().SetTitle("统计问卷").SetTenantID(fx.tenantA.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		payload := `{"surveyId":` + fmt.Sprint(surveyA.ID) + `,"ticketId":` + fmt.Sprint(fx.ticketA.ID) +
			`,"answers":[{"questionIndex":0,"value":6}],"score":6}`
		w := fx.do(t, http.MethodPost, base+"/responses", fx.userA, payload)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		w = fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d/analytics", base, surveyA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		var analytics struct {
			SurveyID      int     `json:"surveyId"`
			ResponseCount int     `json:"responseCount"`
			AverageScore  float64 `json:"averageScore"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &analytics))
		require.Equal(t, surveyA.ID, analytics.SurveyID)
		require.Equal(t, 1, analytics.ResponseCount, "统计只聚合本租可见响应")

		w = fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d/responses", base, surveyA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"items"`)
	})

	t.Run("get跨租户404掩码与list租户隔离", func(t *testing.T) {
		surveyB := fx.client.Survey.Create().SetTitle("他租列表问卷").SetTenantID(fx.tenantB.ID).
			SetQuestions([]schema.SurveyQuestion{{Question: "overall", Type: "rating"}}).SaveX(ctx)
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, surveyB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		w = fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotContains(t, w.Body.String(), "他租列表问卷", "列表不得混入他租问卷")
	})

	t.Run("survey面权限码403禁拒", func(t *testing.T) {
		// survey:write；end_user 不持有（写面）。
		w := fx.do(t, http.MethodPost, base+"/responses", fx.endUser,
			`{"surveyId":1,"ticketId":1,"answers":[{"questionIndex":0,"value":1}]}`)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
