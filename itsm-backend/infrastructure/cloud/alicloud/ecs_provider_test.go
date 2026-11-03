package alicloud

import (
	"context"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"

	"itsm-backend/infrastructure/cloud"
)

func validLaunchPayload() map[string]any {
	return map[string]any{
		"imageId":         "aliyun_3_x64_20G_alibase_20240819.vhd",
		"instanceType":    "ecs.e-c1m1.large",
		"securityGroupId": "sg-test",
		"vSwitchId":       "vsw-test",
	}
}

func TestParseECSLaunchParamsRejectsPublicIP(t *testing.T) {
	payload := validLaunchPayload()
	payload["needsPublicIp"] = true

	_, err := parseECSLaunchParams(payload)
	require.Error(t, err, "没有安全组规则管理权限就不能开公网实例")
	require.Contains(t, err.Error(), "公网")
}

func TestParseECSLaunchParamsListsMissingFields(t *testing.T) {
	_, err := parseECSLaunchParams(map[string]any{"imageId": "img"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "instanceType")
	require.Contains(t, err.Error(), "securityGroupId")
	require.Contains(t, err.Error(), "vSwitchId")
	require.NotContains(t, err.Error(), "imageId")
}

func TestParseECSLaunchParamsRejectsPrepaid(t *testing.T) {
	payload := validLaunchPayload()
	payload["instanceChargeType"] = "PrePaid"

	_, err := parseECSLaunchParams(payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "PostPaid")
}

func TestParseECSLaunchParamsDefaults(t *testing.T) {
	params, err := parseECSLaunchParams(validLaunchPayload())
	require.NoError(t, err)
	require.Equal(t, "PostPaid", params.InstanceChargeType)
	require.Equal(t, int32(1), params.Amount)
	require.Equal(t, "aliyun_3_x64_20G_alibase_20240819.vhd", params.ImageID)
	require.Zero(t, params.InternetMaxBandwidthOut)

	payload := validLaunchPayload()
	payload["quantity"] = float64(3)
	payload["internetMaxBandwidthOut"] = float64(-5)
	params, err = parseECSLaunchParams(payload)
	require.NoError(t, err)
	require.Equal(t, int32(3), params.Amount)
	require.Zero(t, params.InternetMaxBandwidthOut, "负带宽必须归零而不是传给云 API")
}

func TestBuildRunInstancesRequestCarriesIdempotencyAndOwnership(t *testing.T) {
	params, err := parseECSLaunchParams(validLaunchPayload())
	require.NoError(t, err)
	req := cloud.ExecuteRequest{
		TenantID:         42,
		TaskID:           7,
		ServiceRequestID: 99,
		Region:           "cn-hangzhou",
		ClientToken:      "itsm-provisioning-task-7",
		Payload:          validLaunchPayload(),
	}

	runReq := buildRunInstancesRequest(req, params)
	require.Equal(t, "cn-hangzhou", tea.StringValue(runReq.RegionId))
	require.Equal(t, "itsm-provisioning-task-7", tea.StringValue(runReq.ClientToken))
	require.Equal(t, "itsm-sr-99", tea.StringValue(runReq.InstanceName))

	tags := map[string]string{}
	for _, tag := range runReq.Tag {
		tags[tea.StringValue(tag.Key)] = tea.StringValue(tag.Value)
	}
	require.Equal(t, "42", tags["itsm-tenant-id"])
	require.Equal(t, "99", tags["itsm-service-request-id"])
	require.Equal(t, "7", tags["itsm-provisioning-task-id"])
}

func TestExecuteValidatesBeforeCallingCloud(t *testing.T) {
	provider := NewECSProvider(nil)
	base := cloud.ExecuteRequest{
		Region:      "cn-hangzhou",
		ClientToken: "token",
		Payload:     validLaunchPayload(),
		Credential:  cloud.Credential{AccessKeyID: "ak", AccessKeySecret: "sk"},
	}

	_, err := provider.Execute(context.Background(), cloud.ExecuteRequest{Payload: base.Payload})
	require.Error(t, err)
	require.Contains(t, err.Error(), "regionId")

	noCred := base
	noCred.Credential = cloud.Credential{}
	_, err = provider.Execute(context.Background(), noCred)
	require.Error(t, err)
	require.Contains(t, err.Error(), "凭据")

	noToken := base
	noToken.ClientToken = ""
	_, err = provider.Execute(context.Background(), noToken)
	require.Error(t, err)
	require.Contains(t, err.Error(), "幂等键")
}

func TestDescribeAliyunErrorSanitizesAndKeepsRequestID(t *testing.T) {
	err := &tea.SDKError{
		Code:    tea.String("InvalidInstanceType"),
		Message: tea.String("  instance type\n\tnot supported  "),
		Data:    tea.String(`{"RequestId":"req-abc"}`),
	}

	got := describeAliyunError(err)
	require.Contains(t, got, "code=InvalidInstanceType")
	require.Contains(t, got, "requestId=req-abc")
	require.Contains(t, got, "message=instance type not supported")
	require.NotContains(t, got, "\n")
}

func TestTruncateIsRuneSafe(t *testing.T) {
	require.Equal(t, "阿里", truncate("阿里云主机", 2))
	require.Equal(t, "abc", truncate("abc", 10))
	require.Equal(t, "阿里云主机", truncate("阿里云主机", 0))
}

func TestStubProviderFailsOnMissingComplianceAck(t *testing.T) {
	_, err := NewStubProvider().Execute(context.Background(), cloud.ExecuteRequest{
		Payload: map[string]any{"compliance_ack": false},
	})
	require.Error(t, err, "示例 provider 也必须保留合规确认失败语义")

	_, err = NewStubProvider().Execute(context.Background(), cloud.ExecuteRequest{
		Payload: map[string]any{"compliance_ack": true},
	})
	require.NoError(t, err)
}

func TestProviderNamesAreStable(t *testing.T) {
	require.Equal(t, "aliyun", NewECSProvider(nil).Name())
	require.True(t, NewECSProvider(nil).RequiresCloudAccount())
	require.False(t, NewStubProvider().RequiresCloudAccount())
	require.Equal(t, "alicloud", NewStubProvider().Name())
}
