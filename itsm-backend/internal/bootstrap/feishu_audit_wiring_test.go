package bootstrap

import (
	"context"
	"os"
	"regexp"
	"testing"

	domainCommon "itsm-backend/handlers/common"
	connectorDomain "itsm-backend/handlers/connector"
	feishuHandler "itsm-backend/handlers/feishu"

	"github.com/stretchr/testify/require"
)

// 生产装配必须把审计写入器接给飞书 handler。
//
// 为什么这里不能只做源码文本锁（本测试首版就是那么写的，已实测被击穿）：
// `os.ReadFile("app.go") + strings.Contains` 检查的是**字符串是否还在文件里**，
// 不是**装配是否执行**。把那一行注释掉后目标字符串依然在源码里，Contains
// 仍然成立 → 文本锁照绿，而生产上飞书验签失败重新变成零审计。
//
// 改成两步，各有独立判别力：
//  1. 行为断言：`SetAuditRecorder` 之后 `HasAuditRecorder()` 必须为真。
//     它证明「注入点确实把 rec 存进了 handler」，不是「某行文字存在」。
//  2. 装配语句必须**未被注释**：行首非注释锚定的正则，而不是裸 Contains。
//     这一条仍是静态检查，故注释里如实说明它只覆盖「被注释掉」这一种破坏。
func TestFeishuAuditRecorderProductionWiring(t *testing.T) {
	// 1. 行为断言：注入后 handler 自报「已装配」。
	h := feishuHandler.NewHandler(nil, nil, nil, nil)
	require.False(t, h.HasAuditRecorder(), "未注入时必须自报未装配，否则探针恒真、失去判别力")
	h.SetAuditRecorder(stubAuditRecorder{})
	require.True(t, h.HasAuditRecorder(), "SetAuditRecorder 必须真正把 rec 存进 handler")

	// 2. 装配行必须存在且不在注释里。
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)

	wired := regexp.MustCompile(`(?m)^\s*feishuHTTPHandler\.SetAuditRecorder\(`)
	require.True(t, wired.Match(source),
		"app.go 必须以未注释的语句装配飞书审计写入器：注释掉这一行会让验签失败重新变成零审计，"+
			"而第 1 步抓不到它（那测的是 handler 自身，不是组合根）")
}

// stubAuditRecorder 满足 AuditRecorder 接口，本用例不触发写入。
type stubAuditRecorder struct{}

func (stubAuditRecorder) CreateAuditLog(_ context.Context, _ *domainCommon.AuditLog) error {
	return nil
}

var _ connectorDomain.AuditRecorder = stubAuditRecorder{}
