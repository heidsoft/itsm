package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestNotImplementedSemantics 锁定「能力未实现 ≠ 内部故障」这条口径。
//
// 背景：企业微信 / 钉钉渠道在 capability registry 中已声明 built:false
// （MaturityDisabled），webhook 命中"渠道未实现 Receiver"时此前返回 500。
// 这会让已知的产品边界伪装成服务端故障：监控告警被刷屏、调用方误判为自身问题。
// 语义正确的状态码是 501 Not Implemented。
func TestNotImplementedSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, NotImplementedCode, "企业微信渠道未实现 Receiver 接口")

	if w.Code != http.StatusNotImplemented {
		t.Errorf("NotImplementedCode 应映射到 HTTP 501，got %d", w.Code)
	}
	if NotImplementedCode == InternalErrorCode {
		t.Error("NotImplementedCode 必须与 InternalErrorCode 区分，否则无法表达「已知边界」")
	}

	// 反向：501 状态码必须能反查回 5010 业务码，不能被降级成 5001/500。
	if got := statusToAppCode(http.StatusNotImplemented); got != NotImplementedCode {
		t.Errorf("statusToAppCode(501) = %d，期望 %d（501 必须登记在 codeStatusPairs）", got, NotImplementedCode)
	}
}
