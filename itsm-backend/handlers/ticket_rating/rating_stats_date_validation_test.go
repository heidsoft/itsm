// Package ticket_rating 的评分统计日期筛选校验回归（P1-4）。
//
// 缺陷：`dto.GetRatingStatsRequest` 的 StartDate/EndDate 是 `*string` 且无
// binding 标签，`service.GetRatingStats` 里两个 `if err == nil` 都没有 else
// 分支 —— 传 `startDate=abc` 时 time.Parse 失败被静默吞掉，该筛选条件直接
// 消失，查询退化为「全历史评分」。报表使用者以为自己筛了日期，实际看到的
// 是所有历史数据，且界面上没有任何提示。
//
// 修复分两道，两道各自独立可证明：
//
//	① DTO 层 binding `omitempty,datetime=2006-01-02` → handler ShouldBindQuery
//	   直接 400/1001，请求不进 service。
//	② service 层补 else 分支返回 ParamError → 即便有一天 binding 标签被摘掉
//	   （或 service 被别的入口直接调用），非法日期也不会退化成全历史。
//
// 判别式要点：只断言「不是 200」是不够的 —— 修复前该请求**返回 200**，
// 所以「400」本身就证明缺陷被修。另需反向证明 service 那道防线独立成立，
// 因此第二组用例绕过 binding、直接调 service。
package ticket_rating_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"itsm-backend/dto"
	ratingHandler "itsm-backend/handlers/ticket_rating"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestRatingStats_InvalidDateIsRejectedAtHTTPLayer 锁 DTO binding 这道防线。
//
// 修复前三个子用例全部返回 200（筛选被静默丢弃）——这是缺陷本身的形态，
// 断言 400 即是判据。
//
// 🔴 关键：两道防线（DTO binding / service 校验）返回的状态码与 code 完全相同
// （都是 400 / 1001），只断言状态码无法区分「是谁挡下的」。若只断言 400，
// 摘掉 binding 后 service 会把请求兜住、用例仍绿 —— 那时 DTO 这道防线
// 根本不可被证明，等于只有一道。message 是唯一可辨字段：
//   - binding 失败 → handler ShouldBindQuery 分支 → "请求参数错误"
//   - service 失败 → handler FailWithErr 兜底 → "操作失败"
//
// 因此这里断言 message 必须是 "请求参数错误"，证明拦截发生在 bind 阶段。
func TestRatingStats_InvalidDateIsRejectedAtHTTPLayer(t *testing.T) {
	fx := newFixture(t)

	cases := []struct {
		name  string
		query string
	}{
		{"startDate 非日期串", "startDate=" + url.QueryEscape("abc")},
		{"endDate 非日期串", "endDate=" + url.QueryEscape("not-a-date")},
		// 边界：ISO 时间戳不是本端点的契约（契约是日期 2006-01-02，
		// 与前端 date picker 提交的格式一致）。传完整时间戳必须被挡，
		// 否则同一语义参数有两种可接受写法，而其中一种会被 time.Parse 拒掉。
		{"完整时间戳不被接受", "startDate=" + url.QueryEscape("2026-11-22T00:00:00Z")},
		// 边界：month-only / 缺日 也要挡，形近但非法
		{"缺日期分量的月份", "endDate=" + url.QueryEscape("2026-11")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := fx.do(t, http.MethodGet, "/api/v1/tickets/rating-stats?"+tc.query, fx.requesterA)
			require.Equal(t, http.StatusBadRequest, w.Code,
				"非法日期必须 400，不得静默丢弃筛选；body=%s", w.Body.String())
			env := decodeEnvelope(t, w)
			require.Equal(t, 1001, env.Code)
			// 必须是 bind 阶段挡的（不是 service 兜底）——见函数头注释。
			require.Equal(t, "请求参数错误", env.Message,
				"拦截必须发生在 bind 阶段；若是 service 兜底则 DTO binding 未生效；body=%s", w.Body.String())
		})
	}
}

// TestRatingStats_ValidDateRangeStillAccepted 正控：合法日期不被新校验误伤。
//
// 这条与上面同时存在才能证明「不是把整个日期参数一律拒掉」。
func TestRatingStats_ValidDateRangeStillAccepted(t *testing.T) {
	fx := newFixture(t)

	w := fx.do(t, http.MethodGet,
		"/api/v1/tickets/rating-stats?startDate=2026-01-01&endDate=2026-12-31", fx.requesterA)
	require.Equal(t, http.StatusOK, w.Code, "合法日期范围必须放行；body=%s", w.Body.String())
	require.Equal(t, 0, decodeEnvelope(t, w).Code)
}

// TestRatingStats_EmptyDateIsOptional 正控：不传日期是合法的（两个字段都可选）。
//
// 若把 binding 写成 `required,datetime=...`，空查询会被拒成 400 ——
// 那会让「查全部历史」这个正当需求变成 400。本用例锁住 omitempty 语义。
func TestRatingStats_EmptyDateIsOptional(t *testing.T) {
	fx := newFixture(t)

	w := fx.do(t, http.MethodGet, "/api/v1/tickets/rating-stats", fx.requesterA)
	require.Equal(t, http.StatusOK, w.Code, "不带日期必须放行；body=%s", w.Body.String())
}

// TestService_GetRatingStats_InvalidDateFailsClosedNotWholeHistory 锁 service 第二道防线。
//
// 绕过 handler 的 binding 直接调 service —— 真实场景是「binding 标签被摘掉」
// 或「service 被别的入口（定时报表、导出任务）直接调用」。此时若 service 仍
// 静默忽略，用户拿到的是一份标注了日期范围、内容却是全历史的报表。
func TestService_GetRatingStats_InvalidDateFailsClosedNotWholeHistory(t *testing.T) {
	fx := newFixture(t)
	svc := ratingHandler.NewTicketRatingService(fx.client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	badStart := "abc"
	badEnd := "2026-13-45" // 月份/日越界，time.Parse 同样失败

	t.Run("非法 startDate 返回参数错误而不是全历史", func(t *testing.T) {
		resp, err := svc.GetRatingStats(ctx, &dto.GetRatingStatsRequest{
			TenantID:  fx.tenantA.ID,
			StartDate: &badStart,
		})
		require.Error(t, err, "非法 startDate 必须报错，不得退化为全历史查询")
		require.Nil(t, resp)
	})

	t.Run("非法 endDate 返回参数错误而不是全历史", func(t *testing.T) {
		resp, err := svc.GetRatingStats(ctx, &dto.GetRatingStatsRequest{
			TenantID: fx.tenantA.ID,
			EndDate:  &badEnd,
		})
		require.Error(t, err, "非法 endDate 必须报错，不得退化为全历史查询")
		require.Nil(t, resp)
	})

	t.Run("合法日期仍可查询（service 层不被误伤）", func(t *testing.T) {
		goodStart, goodEnd := "2026-01-01", "2026-12-31"
		resp, err := svc.GetRatingStats(ctx, &dto.GetRatingStatsRequest{
			TenantID:  fx.tenantA.ID,
			StartDate: &goodStart,
			EndDate:   &goodEnd,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestRatingStats_DateFilterActuallyNarrowsResult 证明日期筛选真的收窄了结果。
//
// 本用例与上面几条互补：上面证明「非法日期被挡」，这里证明「合法日期确实
// 生效」。若日期一直被静默丢弃（修复前的状态），这条会红 —— 它锁的是
// 「筛选语义可用」，不只是「参数被校验」。
func TestRatingStats_DateFilterActuallyNarrowsResult(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()

	// 两条已评分工单，评分时间一旧一新。
	oldTicket := fx.createTicket(t, ctx, "RT-OLD", "closed", fx.tenantA.ID, fx.requesterA.ID)
	newTicket := fx.createTicket(t, ctx, "RT-NEW", "closed", fx.tenantA.ID, fx.requesterA.ID)
	fx.client.Ticket.UpdateOneID(oldTicket.ID).SetRating(5).
		SetRatedAt(time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)).ExecX(ctx)
	fx.client.Ticket.UpdateOneID(newTicket.ID).SetRating(4).
		SetRatedAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)).ExecX(ctx)

	// 只查 2026 年：应只命中 newTicket 一条。
	w := fx.do(t, http.MethodGet,
		"/api/v1/tickets/rating-stats?startDate=2026-01-01&endDate=2026-12-31", fx.requesterA)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	require.Contains(t, w.Body.String(), `"totalRatings":1`,
		"2026 年区间应只统计 1 条，body=%s", w.Body.String())
}
