package main

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"itsm-backend/service/cloud/aliyun"
)

// TestProbeRejectsWriteActions 是本命令最关键的安全守卫：安全侧探测只允许
// 只读动作。写动作必须在**发起请求之前**被拒——不是「调了但别用」，
// 而是连请求都不发。
//
// 这里用「动作名未在白名单 → 返回 ErrNotReadOnlyAction」实现，验证它确实
// 在发请求前就返回（用不存在的凭据调用：若守卫失效，错误会是签名/鉴权
// 失败而不是 ErrNotReadOnlyAction）。
func TestProbeRejectsWriteActions(t *testing.T) {
	writes := []string{
		"OpenVulScan",
		"CreateVulScanTask",
		"ModifyVulStatus",
		"DeleteInstance",
		"HandleRiskCheckResult",
		"SetConfig",
		"OpenSasService",
	}
	for _, action := range writes {
		_, err := aliyun.ProbeSecurityCenter(context.Background(), "ak-invalid", "sk-invalid", "", action, nil)
		require.Error(t, err, "%s 必须被拒", action)
		require.IsType(t, &aliyun.ErrNotReadOnlyAction{}, err, "%s 必须在发请求前被白名单拒绝", action)
		require.Contains(t, err.Error(), action)
	}

	for _, action := range writes {
		_, err := aliyun.ProbeCloudMonitor(context.Background(), "ak-invalid", "sk-invalid", "", action, nil)
		require.Error(t, err, "%s 必须被拒", action)
		require.IsType(t, &aliyun.ErrNotReadOnlyAction{}, err, "%s 必须在发请求前被白名单拒绝", action)
	}
}

// TestProbeReadOnlyAllowlistShape 锁住白名单本身的形状：收录的必须是
// Describe 开头的只读动作，且产品实际调用的两个动作必须在册。
func TestProbeReadOnlyAllowlistShape(t *testing.T) {
	for _, action := range aliyun.ReadOnlySecurityActions() {
		require.True(t, strings.HasPrefix(action, "Describe"),
			"云安全中心白名单只收 Describe* 只读动作，发现 %q", action)
	}
	for _, action := range aliyun.ReadOnlyMonitorActions() {
		require.True(t, strings.HasPrefix(action, "Describe"),
			"云监控白名单只收 Describe* 只读动作，发现 %q", action)
	}
	require.Contains(t, aliyun.ReadOnlySecurityActions(), "DescribeVulList",
		"产品漏洞同步在用的动作必须在白名单内")
	require.Contains(t, aliyun.ReadOnlyMonitorActions(), "DescribeAlertLogList",
		"产品告警同步在用的动作必须在白名单内")
}

// TestProbeAllowlistIsSortedAndNonEmpty 防止白名单被清空或顺序漂移导致
// 守卫「零命中即通过」。
func TestProbeAllowlistIsSortedAndNonEmpty(t *testing.T) {
	for _, list := range [][]string{aliyun.ReadOnlySecurityActions(), aliyun.ReadOnlyMonitorActions()} {
		require.NotEmpty(t, list)
		for i := 1; i < len(list); i++ {
			require.Less(t, list[i-1], list[i], "白名单应保持有序")
		}
	}
}

// actionPattern 扫描源码里出现的云 API 动作名。
var actionPattern = regexp.MustCompile(`action:\s*"([A-Za-z]+)"`)

// TestVerifyCommandOnlyUsesAllowlistedActions 扫描验证命令源码，确认它
// 只使用白名单内的动作。扫描器零命中必须失败（避免命令写法一变守卫假绿）。
func TestVerifyCommandOnlyUsesAllowlistedActions(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)

	matches := actionPattern.FindAllStringSubmatch(string(src), -1)
	require.NotEmpty(t, matches, "扫描器零命中：动作写法已漂移，守卫失效")

	allowed := map[string]bool{}
	for _, a := range aliyun.ReadOnlySecurityActions() {
		allowed[a] = true
	}
	for _, a := range aliyun.ReadOnlyMonitorActions() {
		allowed[a] = true
	}
	for _, m := range matches {
		require.True(t, allowed[m[1]],
			"命令使用了不在只读白名单内的动作 %q", m[1])
	}
}

// TestVerifyCommandNeverCallsWriteActions 反向清单：源码里不得出现任何
// 写动作名，防止有人绕过 Probe* 的白名单直接用底层 client。
func TestVerifyCommandNeverCallsWriteActions(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)

	forbidden := []string{
		"OpenVulScan", "CreateVulScanTask", "ModifyVulStatus", "HandleRiskCheckResult",
		"OpenSasService", "SetConfig", "DeleteInstance", "CreateInstance",
	}
	for _, name := range forbidden {
		require.NotContains(t, string(src), name, "命令不得引用写动作 %q", name)
	}
}

// TestClassifyRecognizesPermanentVsTransient 锁住结论分类：服务未开通与
// 权限不足是「要人处理」的永久结论，限流是「等一会就好」的临时结论。
// 分类错了会让人在临时故障上做永久改动。
func TestClassifyRecognizesPermanentVsTransient(t *testing.T) {
	cases := []struct {
		code    string
		verdict string
	}{
		{"Forbidden.RAM", "no-permission"},
		{"NoPermission", "no-permission"},
		{"Unauthorized", "no-permission"},
		{"AuthFailure.ServiceNotOpened", "service-not-open"},
		{"InvalidParameter.Type", "invalid-parameter"},
		{"Throttling.User", "throttled"},
		{"ServiceUnavailable", "throttled"},
	}
	for _, c := range cases {
		got := classify(result{Code: c.code}, "product-path")
		require.Equal(t, c.verdict, got, "错误码 %q 的分类", c.code)
	}

	// 无业务码但 HTTP 非 2xx → http-error；有业务码但不识别的 → cloud-error。
	require.Equal(t, "http-error", classify(result{HTTPStatus: 500}, "product-path"))
	require.Equal(t, "cloud-error", classify(result{HTTPStatus: 400, Code: "Something.New"}, "product-path"))
	require.Equal(t, "transport-error", classify(result{HTTPStatus: 0}, "product-path"))
}
