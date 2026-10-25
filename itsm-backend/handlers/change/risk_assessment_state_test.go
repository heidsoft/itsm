package change

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// 回归：GET /changes/:id/risk-assessment 在「尚未评估」时返回 data:null，
// CLI 只能报 ok:true 但拿不到任何内容 —— 调用方无法区分「未评估」「变更不存在」
// 「能力缺失」。现在恒返回对象并由 assessed 显式判别，变更不存在时明确 404。

func decodeRiskData(t *testing.T, w *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var envelope struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Zero(t, envelope.Code, w.Body.String())
	return envelope.Data
}

func TestGetRiskAssessment_UnassessedIsExplicit(t *testing.T) {
	r, _, repo := setupTestHandler(t)
	c := createTestChange(repo, 1, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/changes/"+strconv.Itoa(c.ID)+"/risk-assessment", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), `"data":null`, "未评估不得返回 data:null")

	data := decodeRiskData(t, w)
	require.JSONEq(t, "false", string(data["assessed"]))
	require.Equal(t, strconv.Itoa(c.ID), string(data["changeId"]))
	require.NotContains(t, data, "id", "未评估不得下发 id:0 的假记录")
}

func TestGetRiskAssessment_UnknownChangeIs404(t *testing.T) {
	r, _, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/changes/99999/risk-assessment", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 4004, envelope.Code)
}

func TestGetRiskAssessment_AssessedMarksTrue(t *testing.T) {
	r, _, repo := setupTestHandler(t)
	c := createTestChange(repo, 1, 1)
	repo.riskAssess[c.ID] = &RiskAssessment{
		ID:              7,
		ChangeID:        c.ID,
		TenantID:        1,
		RiskLevel:       "high",
		RiskDescription: "核心链路变更",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/changes/"+strconv.Itoa(c.ID)+"/risk-assessment", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := decodeRiskData(t, w)
	require.JSONEq(t, "true", string(data["assessed"]))
	require.JSONEq(t, "7", string(data["id"]))
	require.JSONEq(t, `"high"`, string(data["riskLevel"]))
	require.JSONEq(t, `"核心链路变更"`, string(data["riskDescription"]))
}
