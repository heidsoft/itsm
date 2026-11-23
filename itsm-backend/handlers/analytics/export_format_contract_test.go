package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type exportStub struct {
	called int
	format string
}

func (s *exportStub) ExportAnalytics(_ context.Context, _ *dto.DeepAnalyticsRequest, format string, _ int, _ dto.DashboardViewer) ([]byte, string, error) {
	s.called++
	s.format = format
	return []byte("ok"), "report." + format, nil
}

func (s *exportStub) GetTicketStats(context.Context, int) (*dto.AnalyticsTicketStatsResponse, error) {
	return &dto.AnalyticsTicketStatsResponse{}, nil
}

func TestExportAnalyticsRejectsUnsupportedPDF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &exportStub{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", 1); c.Set("user_id", 1); c.Set("role", "admin") })
	r.POST("/api/v1/analytics/export", NewHandler(stub).ExportAnalytics)

	for _, tc := range []struct {
		format string
		status int
		code   int
	}{
		{"pdf", http.StatusBadRequest, common.ParamErrorCode},
		{"txt", http.StatusBadRequest, common.ParamErrorCode},
		{"excel", http.StatusOK, 0},
		{"csv", http.StatusOK, 0},
	} {
		t.Run(tc.format, func(t *testing.T) {
			before := stub.called
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
				"/api/v1/analytics/export?format="+tc.format,
				strings.NewReader(`{"timeRange":["2026-01-01","2026-01-31"]}`)))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.status == http.StatusOK {
				require.Equal(t, "ok", w.Body.String())
				require.Equal(t, before+1, stub.called)
				require.Equal(t, tc.format, stub.format)
			} else {
				var envelope struct {
					Code int `json:"code"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
				require.Equal(t, tc.code, envelope.Code, w.Body.String())
				require.Equal(t, before, stub.called, "不支持的格式不得触达导出服务")
				require.NotContains(t, w.Body.String(), "application/pdf")
			}
		})
	}
}
