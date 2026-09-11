package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归：前端不带 timeRange 时 GetTrendPrediction 不得 panic（历史 bug：
// 直接索引 req.TimeRange[0] → 500），且默认窗口语义为「过去 6 个月 → 今天」，
// 与 handlers/ai/service.go 的 GetTrendPrediction 默认值保持一致。
func TestPredictionService_EmptyTimeRange_DefaultWindow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewPredictionService(client, logger)

	ctx := context.Background()

	// 最小种子数据：tenant + user + 少量工单（历史窗口内），验证查询路径可跑通
	testTenant, err := client.Tenant.Create().
		SetName("Pred Tenant").
		SetCode("pred").
		SetDomain("pred.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("preduser").
		SetEmail("pred@example.com").
		SetName("Pred User").
		SetPasswordHash("hashed").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Pred Ticket").
			SetDescription("seed").
			SetPriority("medium").
			SetType("ticket").
			SetStatus("open").
			SetTicketNumber("TKT-PRED-" + string(rune('A'+i))).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	t.Run("空timeRange_不panic_走默认窗口", func(t *testing.T) {
		req := &dto.TrendPredictionRequest{
			PredictionType: "volume",
			// TimeRange 故意不传
		}
		resp, err := svc.GetTrendPrediction(ctx, req, testTenant.ID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "arima", resp.Model) // 默认模型
		// 默认窗口 = 过去 6 个月 → 今天。预测点粒度按月，断言所有点
		// 落在默认窗口起点之后（留 1 天缓冲）即可，不假设终点对齐今天。
		windowStart := time.Now().AddDate(0, -6, 0).AddDate(0, 0, -1)
		for _, p := range resp.Predictions {
			pd, perr := time.Parse("2006-01-02", p.Date)
			require.NoError(t, perr, "预测点日期格式异常: %s", p.Date)
			assert.False(t, pd.Before(windowStart),
				"预测点 %s 早于默认窗口起点 %s", p.Date, windowStart.Format("2006-01-02"))
		}
	})

	t.Run("空timeRange_各预测类型_均不panic", func(t *testing.T) {
		for _, ptype := range []string{"volume", "type", "priority", "resource"} {
			req := &dto.TrendPredictionRequest{PredictionType: ptype}
			resp, err := svc.GetTrendPrediction(ctx, req, testTenant.ID)
			require.NoError(t, err, "predictionType=%s 不应 panic", ptype)
			assert.NotNil(t, resp)
		}
	})

	t.Run("显式timeRange_对照_仍正常", func(t *testing.T) {
		now := time.Now()
		req := &dto.TrendPredictionRequest{
			PredictionType: "volume",
			TimeRange: []string{
				now.AddDate(0, -3, 0).Format("2006-01-02"),
				now.Format("2006-01-02"),
			},
		}
		resp, err := svc.GetTrendPrediction(ctx, req, testTenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("非法timeRange格式_返回错误而非panic", func(t *testing.T) {
		req := &dto.TrendPredictionRequest{
			PredictionType: "volume",
			TimeRange:      []string{"not-a-date", "2026-09-11"},
		}
		_, err := svc.GetTrendPrediction(ctx, req, testTenant.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start time format")
	})
}
