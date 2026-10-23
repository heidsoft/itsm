package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRateLimitSemantics 锁定「限流 ≠ 无权限」这条口径。
//
// 背景：登录限流曾返回 ForbiddenCode(2003/403)，普通限流中间件直接传
// http.StatusTooManyRequests(429) 作为**业务码**。两者都表达错了：
//   - 403 让客户端以为「重试也没用」，于是不会退避；
//   - 429 不是已登记业务码，statusForCode 兜底成 HTTP 200，限流在传输层
//     看起来像成功（`itsm` 曾因此把 404/限流都归到 validation）。
func TestRateLimitSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, RateLimitCode, "请求过于频繁，请稍后再试")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("RateLimitCode 应映射到 HTTP 429，got %d", w.Code)
	}
	if got := statusToAppCode(http.StatusTooManyRequests); got != RateLimitCode {
		t.Errorf("statusToAppCode(429) = %d，期望 %d（429 必须登记在 codeStatusPairs）", got, RateLimitCode)
	}
	if RateLimitCode == ForbiddenCode {
		t.Error("限流业务码必须与权限不足区分：403 表示重试无用，429 表示稍后重试")
	}
}
