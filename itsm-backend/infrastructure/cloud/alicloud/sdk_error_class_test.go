package alicloud

import (
	"errors"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"

	"itsm-backend/infrastructure/cloud"
)

// TestClassifySDKErrorNilIsFailSafe 锁住 2026-11-03 真实账号验证打出的
// 空指针缺陷：只读校验里「故意传非法参数却返回成功」是合法观测结果，
// ClassifySDKError(nil) 必须返回空分类而不是 panic。
func TestClassifySDKErrorNilIsFailSafe(t *testing.T) {
	require.NotPanics(t, func() {
		got := ClassifySDKError(nil)
		require.Empty(t, got.Code)
		require.False(t, got.Permanent)
		require.Empty(t, got.Detail)
	})
	require.Empty(t, describeAliyunError(nil))
	require.Empty(t, aliyunErrorCode(nil))
}

func TestClassifySDKErrorRealSDKError(t *testing.T) {
	// 2026-11-03 真实账号 DryRun 实测返回的形态：HTTP 403 + Data 里的
	// RequestId，业务码 InvalidAccountStatus.NotEnoughBalance。
	permanent := &tea.SDKError{
		Code:       tea.String("InvalidAccountStatus.NotEnoughBalance"),
		StatusCode: tea.Int(403),
		Message:    tea.String("Your account does not have enough balance to order"),
		Data:       tea.String(`{"RequestId":"01A19EF2-FC52-566C-82D5-DB9D9EBF0F1D","Code":"InvalidAccountStatus.NotEnoughBalance"}`),
	}
	got := ClassifySDKError(permanent)
	require.Equal(t, "InvalidAccountStatus.NotEnoughBalance", got.Code)
	require.True(t, got.Permanent, "余额不足是永久错误，必须停止自动重投")
	require.Contains(t, got.Detail, "code=InvalidAccountStatus.NotEnoughBalance")
	require.Contains(t, got.Detail, "requestId=01A19EF2-FC52-566C-82D5-DB9D9EBF0F1D")

	// 限流/5xx 仍须判为可重试，误判会让一次抖动变成「交付永久失败」。
	for _, code := range []string{"Throttling.User", "ServiceUnavailable", "InternalError", "RequestTimeout"} {
		retryable := ClassifySDKError(&tea.SDKError{Code: tea.String(code), StatusCode: tea.Int(500)})
		require.Equal(t, code, retryable.Code)
		require.False(t, retryable.Permanent, "%s 不得判永久", code)
	}
}

func TestClassifySDKErrorPlainErrorIsNotSDKError(t *testing.T) {
	got := ClassifySDKError(errors.New("dial tcp: i/o timeout"))
	require.Empty(t, got.Code, "普通网络错误没有 SDK 业务码")
	require.False(t, got.Permanent, "网络抖动必须可重试")
	require.Contains(t, got.Detail, "dial tcp")
}

// TestClassifySDKErrorNeverLeaksRequestBody 锁住脱敏契约：Detail 里不得出现
// 请求体里的密码或完整响应。交付失败时这段文案会写进任务错误与审计。
func TestClassifySDKErrorNeverLeaksRequestBody(t *testing.T) {
	// 用纯虚构值，避免测试用例本身成为凭据泄露面。
	const fakeSecret = "EXAMPLE_SECRET_NOT_A_REAL_KEY"
	got := ClassifySDKError(&tea.SDKError{
		Code:    tea.String("InvalidParameter"),
		Message: tea.String("failed with password=" + fakeSecret + " and access_key_secret=" + fakeSecret + " and token=" + fakeSecret),
		Data:    tea.String(`{"RequestId":"REQ-1","Password":"` + fakeSecret + `"}`),
	})
	require.NotContains(t, got.Detail, fakeSecret)
	require.Contains(t, got.Detail, "code=InvalidParameter")
	require.Contains(t, got.Detail, "requestId=REQ-1")
	require.Contains(t, got.Detail, "[REDACTED]")
}

func TestSanitizeMessageKeepsReconciliationFields(t *testing.T) {
	// 脱敏不能误伤对账字段：code / requestId / message 必须原样可读。
	got := sanitizeMessage("code=InvalidImageId.NotFound requestId=REQ-2 message=image not found password=EXAMPLE")
	require.Contains(t, got, "code=InvalidImageId.NotFound")
	require.Contains(t, got, "requestId=REQ-2")
	require.Contains(t, got, "message=image not found")
	require.NotContains(t, got, "password=EXAMPLE")
	require.Contains(t, got, "password=[REDACTED]")
}

func TestSanitizeMessageCoversJSONShape(t *testing.T) {
	got := sanitizeMessage(`{"Password":"EXAMPLE_SECRET","AccessKeySecret":"EXAMPLE_SECRET","RequestId":"REQ-3"}`)
	require.NotContains(t, got, "EXAMPLE_SECRET")
	require.Contains(t, got, "REQ-3")
}

func TestBuildDryRunRequestReusesProductionParams(t *testing.T) {
	// DryRun 必须复用生产的 parseECSLaunchParams + buildRunInstancesRequest：
	// 验证跑通的参数要和真正交付的参数完全一致，否则「只读验证通过」是空头支票。
	payload := map[string]any{
		"imageId":         "aliyun_3_x64_20G_pro_alibase_20260827.vhd",
		"instanceType":    "ecs.e-c1m1.large",
		"securityGroupId": "sg-2ze0uyosip126siwlspg",
		"vSwitchId":       "vsw-2zevngivqc6lt5mtqyl2c",
	}
	req := cloud.ExecuteRequest{
		TenantID: 42, TaskID: 7, ServiceRequestID: 99,
		Region: "cn-beijing", ClientToken: "itsm-provisioning-task-7",
		Payload: payload,
	}
	got, err := BuildDryRunRequest(req)
	require.NoError(t, err)
	require.NotNil(t, got.DryRun)
	require.True(t, *got.DryRun, "DryRun 未置 true 会造成真实创建与计费")
	require.Equal(t, "cn-beijing", tea.StringValue(got.RegionId))
	require.Equal(t, "itsm-provisioning-task-7", tea.StringValue(got.ClientToken))
	require.Equal(t, "PostPaid", tea.StringValue(got.InstanceChargeType))
	require.Equal(t, "itsm-sr-99", tea.StringValue(got.InstanceName))
	require.Len(t, got.Tag, 3, "谱系 Tag 必须与真实交付一致")

	// 真正交付的那份请求除 DryRun 外必须逐字段一致。
	params, err := parseECSLaunchParams(payload)
	require.NoError(t, err)
	realReq := buildRunInstancesRequest(req, params)
	require.Equal(t, tea.StringValue(realReq.ImageId), tea.StringValue(got.ImageId))
	require.Equal(t, tea.StringValue(realReq.InstanceType), tea.StringValue(got.InstanceType))
	require.Equal(t, tea.StringValue(realReq.SecurityGroupId), tea.StringValue(got.SecurityGroupId))
	require.Equal(t, tea.StringValue(realReq.VSwitchId), tea.StringValue(got.VSwitchId))
	require.Equal(t, tea.Int32Value(realReq.Amount), tea.Int32Value(got.Amount))
}

func TestBuildDryRunRequestValidatesLikeExecute(t *testing.T) {
	base := cloud.ExecuteRequest{
		Region: "cn-beijing", ClientToken: "token",
		Payload: map[string]any{
			"imageId": "img", "instanceType": "ecs.e-c1m1.large",
			"securityGroupId": "sg", "vSwitchId": "vsw",
		},
	}
	noRegion := base
	noRegion.Region = ""
	_, err := BuildDryRunRequest(noRegion)
	require.ErrorContains(t, err, "regionId")

	noToken := base
	noToken.ClientToken = ""
	_, err = BuildDryRunRequest(noToken)
	require.ErrorContains(t, err, "幂等键")

	prepaid := base
	prepaid.Payload = map[string]any{
		"imageId": "img", "instanceType": "ecs.e-c1m1.large",
		"securityGroupId": "sg", "vSwitchId": "vsw", "instanceChargeType": "PrePaid",
	}
	_, err = BuildDryRunRequest(prepaid)
	require.ErrorContains(t, err, "PostPaid")
}
