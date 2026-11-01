package aliyun

import (
	"testing"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"
)

func TestBuildDescribeInstancesRequestIncludesRegion(t *testing.T) {
	req := buildDescribeInstancesRequest("cn-hangzhou", "")
	require.Equal(t, "cn-hangzhou", tea.StringValue(req.RegionId))
	require.Equal(t, int32(100), tea.Int32Value(req.MaxResults))
	require.Nil(t, req.NextToken)
}

func TestBuildDescribeInstancesRequestCarriesNextToken(t *testing.T) {
	req := buildDescribeInstancesRequest("cn-beijing", "token-1")
	require.Equal(t, "cn-beijing", tea.StringValue(req.RegionId))
	require.Equal(t, "token-1", tea.StringValue(req.NextToken))
}

func TestTransformInstanceToleratesMissingNetworkAttributes(t *testing.T) {
	res := transformInstance(&ecs.DescribeInstancesResponseBodyInstancesInstance{
		InstanceId:   tea.String("i-classic"),
		InstanceName: tea.String("legacy"),
		Status:       tea.String("Running"),
	}, "cn-hangzhou")

	require.Equal(t, "i-classic", res.ResourceID)
	require.Equal(t, "active", res.Status)
	require.Equal(t, "", res.Extra["eip_ip_address"])
	require.Equal(t, "", res.Extra["vpc_id"])
	require.Nil(t, res.Extra["private_ip_address"])
	require.Nil(t, res.Extra["operation_locks"])
}

func TestTransformInstanceCarriesInstanceDetail(t *testing.T) {
	res := transformInstance(&ecs.DescribeInstancesResponseBodyInstancesInstance{
		InstanceId:          tea.String("i-detail"),
		InstanceName:        tea.String("app-1"),
		HostName:            tea.String("app-1.internal"),
		Status:              tea.String("Running"),
		ZoneId:              tea.String("cn-hangzhou-h"),
		InstanceType:        tea.String("ecs.c7.large"),
		InstanceTypeFamily:  tea.String("ecs.c7"),
		Cpu:                 tea.Int32(2),
		Memory:              tea.Int32(4096),
		OSName:              tea.String("Alibaba Cloud Linux 3"),
		OSType:              tea.String("linux"),
		IoOptimized:         tea.Bool(true),
		KeyPairName:         tea.String("ops-key"),
		ResourceGroupId:     tea.String("rg-1"),
		StartTime:           tea.String("2026-10-01T00:00Z"),
		DeletionProtection:  tea.Bool(true),
		CreditSpecification: tea.String("Standard"),
		EipAddress: &ecs.DescribeInstancesResponseBodyInstancesInstanceEipAddress{
			IpAddress:    tea.String("47.1.2.3"),
			AllocationId: tea.String("eip-1"),
			Bandwidth:    tea.Int32(5),
		},
		VpcAttributes: &ecs.DescribeInstancesResponseBodyInstancesInstanceVpcAttributes{
			VpcId:     tea.String("vpc-1"),
			VSwitchId: tea.String("vsw-1"),
			PrivateIpAddress: &ecs.DescribeInstancesResponseBodyInstancesInstanceVpcAttributesPrivateIpAddress{
				IpAddress: []*string{tea.String("10.0.0.5")},
			},
		},
		OperationLocks: &ecs.DescribeInstancesResponseBodyInstancesInstanceOperationLocks{
			LockReason: []*ecs.DescribeInstancesResponseBodyInstancesInstanceOperationLocksLockReason{
				{LockReason: tea.String("security"), LockMsg: tea.String("locked for security")},
			},
		},
	}, "cn-hangzhou")

	extra := res.Extra
	require.Equal(t, "cn-hangzhou-h", res.Zone)
	require.Equal(t, "ecs.c7", extra["instance_type_family"])
	require.Equal(t, int64(2), extra["cpu"])
	require.Equal(t, "app-1.internal", extra["host_name"])
	require.Equal(t, "Alibaba Cloud Linux 3", extra["os_name"])
	require.Equal(t, true, extra["io_optimized"])
	require.Equal(t, true, extra["deletion_protection"])
	require.Equal(t, "ops-key", extra["key_pair_name"])
	require.Equal(t, "47.1.2.3", extra["eip_ip_address"])
	require.Equal(t, "eip-1", extra["eip_allocation_id"])
	require.Equal(t, int64(5), extra["eip_bandwidth"])
	require.Equal(t, "vpc-1", extra["vpc_id"])
	require.Equal(t, []string{"10.0.0.5"}, extra["private_ip_address"])
	require.Equal(t, []map[string]string{{"reason": "security", "message": "locked for security"}}, extra["operation_locks"])
}
