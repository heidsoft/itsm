package router

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 审计行必须带上**这一笔请求**的 request_id —— 从全局中间件注入到落库的整条链。
//
// 为什么这条必须存在（单元测试证明不了它）：middleware 层的
// `TestAuditWriters_PreserveRequestIDAfterCancellation` 是**直接**往 ctx 里塞
// 一个 request ID 再调 Record 函数，它只能证明「值在 ctx 里时会被搬到落库
// 上下文」。它证明不了「生产装配真的注入了值」：
// `RequestIDMiddleware` 曾经长时间没被注册（见 router.go:301 注释记录的
// 「本注册从未发生」），那时审计行的 request_id 恒为空串，而所有单元测试
// 依然全绿。两层必须各有一条：单元层锁「搬运」，本用例锁「注入→落库」。
//
// 断言用「响应头 == 落库值」而不是只断言非空：非空只能证明有值，
// 相等才能证明落库的**就是**这一笔请求的 ID，而不是某一笔残留或并发请求的。
func TestSelfServiceAuditRowCarriesRequestIDEndToEnd(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	// Register 要求「恰好一个活跃租户」，否则 fail-closed 返 503。
	client.Tenant.Create().SetName("审计链路租户").SetCode("audit-rid").
		SetDomain("audit-rid.example.com").SetStatus("active").SaveX(ctx)

	w := doPublicPOST(r, "/api/v1/auth/register",
		`{"username":"riduser","email":"rid@example.com","password":"Str0ngPass2026!","passwordConfirm":"Str0ngPass2026!"}`)
	require.Equal(t, 200, w.Code, "必须真的注册成功，否则测的是失败分支：body=%s", w.Body.String())

	headerRID := strings.TrimSpace(w.Header().Get("X-Request-Id"))
	require.NotEmpty(t, headerRID, "全局 RequestIDMiddleware 必须为每个请求注入 ID")

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1, "一次注册必须恰好一条审计")

	assertRequestIDMatches(t, headerRID, logs[0].RequestID)
}

// 登录走另一条审计写入函数（RecordLoginAudit 而非 RecordSelfServiceAudit），
// 两处都有 context.Background() 的独立写入，必须各有一条端到端证据。
func TestLoginAuditRowCarriesRequestIDEndToEnd(t *testing.T) {
	r, client := newPublicAuditRouter(t)
	ctx := context.Background()

	w := doPublicPOST(r, "/api/v1/auth/login",
		`{"username":"ghost","password":"wrong-pass","tenantCode":"nope"}`)
	require.NotEqual(t, 200, w.Code, "凭据必须被拒，否则测不到失败留痕路径")

	headerRID := strings.TrimSpace(w.Header().Get("X-Request-Id"))
	require.NotEmpty(t, headerRID)

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, logs, "失败的登录尝试必须留痕")

	assertRequestIDMatches(t, headerRID, logs[0].RequestID)
}

func assertRequestIDMatches(t *testing.T, want, got string) {
	t.Helper()
	require.NotEmpty(t, strings.TrimSpace(got),
		"落库 request_id 为空：RequestIDMiddleware 未装配，或审计写作用 Background 时没搬运链路值")
	require.Equal(t, want, got,
		"落库 request_id 必须等于这一笔请求的响应头 ID，否则审计与访问日志对不上号")
}
