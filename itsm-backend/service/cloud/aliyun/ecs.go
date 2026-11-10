package aliyun

import (
	"context"
	"fmt"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// Provider / ServiceCodeECS 供装配层（service/cloud/adapters）与能力清单引用，
// 避免注册处再写一遍字符串字面量。
const (
	Provider       = "aliyun"
	ServiceCodeECS = "ecs"
	// ecsMaxPages 防止云侧回显不变的 NextToken 让内部翻页永不结束。
	// 命中上限必须返回错误而不是静默截断：上层缺失扫描会把没扫到的
	// 存量资源当作「本轮未出现」推进 missing_count，进而误置 stale。
	ecsMaxPages = 100
)

// AliyunECSAdapter 阿里云 ECS 发现适配器
type AliyunECSAdapter struct {
	logger *zap.SugaredLogger
}

// NewAliyunECSAdapter 构造器
func NewAliyunECSAdapter(logger *zap.SugaredLogger) *AliyunECSAdapter {
	if logger == nil {
		l, _ := zap.NewDevelopment()
		logger = l.Sugar()
	}
	return &AliyunECSAdapter{logger: logger}
}

func (*AliyunECSAdapter) Provider() string    { return Provider }
func (*AliyunECSAdapter) ServiceCode() string { return ServiceCodeECS }
func (a *AliyunECSAdapter) Close()            {}

func (a *AliyunECSAdapter) InitClients(ctx context.Context, account *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	cred, err := cloud.ResolveAliyunCredential(ctx, account.CredentialRef)
	if err != nil {
		return nil, err
	}

	clients := make(map[string]cloud.Client, len(regions))
	for _, region := range regions {
		c, err := newECSClient(region, cred)
		if err != nil {
			a.logger.Warnw("创建 ECS 客户端失败", "region", region, "error", err)
			continue
		}
		clients[region] = &ecsClientWrapper{ecs: c}
	}
	return clients, nil
}

func newECSClient(region string, cred *cloud.ResolvedCredential) (*ecs.Client, error) {
	cfg := &openapiutil.Config{}
	cfg.RegionId = tea.String(region)
	cfg.AccessKeyId = tea.String(cred.AccessKeyID)
	cfg.AccessKeySecret = tea.String(cred.AccessKeySecret)
	cfg.ConnectTimeout = tea.Int(5_000)
	cfg.ReadTimeout = tea.Int(15_000)
	if cred.SessionToken != "" {
		cfg.SecurityToken = tea.String(cred.SessionToken)
	}
	return ecs.NewClient(cfg)
}

func runtimeOptions() *dara.RuntimeOptions {
	return &dara.RuntimeOptions{
		ConnectTimeout: tea.Int(5_000),
		ReadTimeout:    tea.Int(15_000),
		Autoretry:      tea.Bool(false),
		MaxAttempts:    tea.Int(1),
	}
}

func (*AliyunECSAdapter) ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error) {
	cred, err := cloud.ResolveAliyunCredential(ctx, account.CredentialRef)
	if err != nil {
		return nil, err
	}
	client, err := newECSClient("cn-hangzhou", cred)
	if err != nil {
		return nil, err
	}
	response, err := client.DescribeRegionsWithOptions(&ecs.DescribeRegionsRequest{}, runtimeOptions())
	if err != nil {
		return nil, err
	}
	if response.Body == nil || response.Body.Regions == nil {
		return nil, nil
	}
	regions := make([]string, 0, len(response.Body.Regions.Region))
	for _, region := range response.Body.Regions.Region {
		if region == nil || region.RegionId == nil || tea.StringValue(region.RegionId) == "" {
			continue
		}
		regions = append(regions, tea.StringValue(region.RegionId))
	}
	return regions, nil
}

func (a *AliyunECSAdapter) DiscoverRegion(ctx context.Context, account *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	ecsClient, ok := client.(*ecsClientWrapper)
	if !ok {
		return nil, nil
	}

	var allResources []cloud.DiscoveredResource
	token := nextToken

	for pageIndex := 0; pageIndex < ecsMaxPages; pageIndex++ {
		req := buildDescribeInstancesRequest(region, token)

		resp, err := ecsClient.ecs.DescribeInstancesWithOptions(req, runtimeOptions())
		if err != nil {
			return nil, err
		}

		if resp.Body == nil || resp.Body.Instances == nil {
			return &cloud.PageResult{Resources: allResources}, nil
		}

		for _, inst := range resp.Body.Instances.Instance {
			if inst == nil {
				continue
			}
			allResources = append(allResources, transformInstance(inst, region))
		}

		newToken := tea.StringValue(resp.Body.NextToken)
		if newToken == "" {
			return &cloud.PageResult{Resources: allResources}, nil
		}
		if newToken == token {
			return nil, fmt.Errorf("aliyun ecs: DescribeInstances returned unchanged NextToken %q in region %s", newToken, region)
		}
		token = newToken
	}
	return nil, fmt.Errorf("aliyun ecs: DescribeInstances exceeded %d pages in region %s", ecsMaxPages, region)
}

// DescribeInstances 的 RegionId 是必填项，客户端配置里的 RegionId 不会自动填充请求体。
func buildDescribeInstancesRequest(region, nextToken string) *ecs.DescribeInstancesRequest {
	req := &ecs.DescribeInstancesRequest{RegionId: tea.String(region)}
	req.MaxResults = tea.Int32(100)
	if nextToken != "" {
		req.NextToken = tea.String(nextToken)
	}
	return req
}

func (*AliyunECSAdapter) ValidateCredential(ctx context.Context, account *ent.CloudAccount) error {
	cred, err := cloud.ResolveAliyunCredential(ctx, account.CredentialRef)
	if err != nil {
		return err
	}
	c, err := newECSClient("cn-hangzhou", cred)
	if err != nil {
		return err
	}
	_, err = c.DescribeRegionsWithOptions(&ecs.DescribeRegionsRequest{}, runtimeOptions())
	return err
}

// ecsClientWrapper 包装 *ecs.Client 以实现 cloud.Client 接口
type ecsClientWrapper struct{ ecs *ecs.Client }

func (c *ecsClientWrapper) Close() error { return nil }

// ===== 辅助函数 =====

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(i *int32) int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// EipAddress 与 VpcAttributes 只在对应网络形态下返回，经典网络或未绑定公网 IP 的
// 实例上为 nil，直接取内部字段会 panic。
func extractEip(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) (ip, allocationID string, bandwidth int64) {
	if inst.EipAddress == nil {
		return "", "", 0
	}
	return derefString(inst.EipAddress.IpAddress), derefString(inst.EipAddress.AllocationId), derefInt32(inst.EipAddress.Bandwidth)
}

func extractVpc(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) (vpcID, vswitchID, natIP string, privateIPs []string) {
	if inst.VpcAttributes == nil {
		return "", "", "", nil
	}
	a := inst.VpcAttributes
	if a.PrivateIpAddress != nil {
		for _, ip := range a.PrivateIpAddress.IpAddress {
			if ip != nil {
				privateIPs = append(privateIPs, *ip)
			}
		}
	}
	return derefString(a.VpcId), derefString(a.VSwitchId), derefString(a.NatIpAddress), privateIPs
}

func extractOperationLocks(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) []map[string]string {
	if inst.OperationLocks == nil {
		return nil
	}
	locks := make([]map[string]string, 0, len(inst.OperationLocks.LockReason))
	for _, l := range inst.OperationLocks.LockReason {
		if l == nil {
			continue
		}
		locks = append(locks, map[string]string{
			"reason":  derefString(l.LockReason),
			"message": derefString(l.LockMsg),
		})
	}
	return locks
}

func extractInnerIPs(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) []string {
	if inst.InnerIpAddress == nil || len(inst.InnerIpAddress.IpAddress) == 0 {
		return nil
	}
	result := make([]string, 0, len(inst.InnerIpAddress.IpAddress))
	for _, ip := range inst.InnerIpAddress.IpAddress {
		if ip != nil {
			result = append(result, *ip)
		}
	}
	return result
}

func extractPublicIPs(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) []string {
	if inst.PublicIpAddress == nil || len(inst.PublicIpAddress.IpAddress) == 0 {
		return nil
	}
	result := make([]string, 0, len(inst.PublicIpAddress.IpAddress))
	for _, ip := range inst.PublicIpAddress.IpAddress {
		if ip != nil {
			result = append(result, *ip)
		}
	}
	return result
}

func extractSecurityGroupIDs(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) []string {
	if inst.SecurityGroupIds == nil || len(inst.SecurityGroupIds.SecurityGroupId) == 0 {
		return nil
	}
	result := make([]string, 0, len(inst.SecurityGroupIds.SecurityGroupId))
	for _, id := range inst.SecurityGroupIds.SecurityGroupId {
		if id != nil {
			result = append(result, *id)
		}
	}
	return result
}

func extractTags(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) map[string]string {
	if inst.Tags == nil || inst.Tags.Tag == nil {
		return nil
	}
	tags := make(map[string]string)
	for _, tag := range inst.Tags.Tag {
		if tag == nil {
			continue
		}
		key := derefString(tag.TagKey)
		if key != "" {
			tags[key] = derefString(tag.TagValue)
		}
	}
	return tags
}

func mapStatus(status string) string {
	switch status {
	case "Creating":
		return "pending"
	case "Running":
		return "active"
	case "Stopped", "Stopping", "ShuttingDown":
		return "inactive"
	case "Released", "Expired", "Deleted", "Terminated":
		return "retired"
	default:
		return "inactive"
	}
}

func transformInstance(inst *ecs.DescribeInstancesResponseBodyInstancesInstance, region string) cloud.DiscoveredResource {
	eipIP, eipAllocationID, eipBandwidth := extractEip(inst)
	vpcID, vswitchID, natIP, privateIPs := extractVpc(inst)

	extra := map[string]interface{}{
		"instance_type":              derefString(inst.InstanceType),
		"instance_type_family":       derefString(inst.InstanceTypeFamily),
		"cpu":                        derefInt32(inst.Cpu),
		"memory":                     derefInt32(inst.Memory),
		"image_id":                   derefString(inst.ImageId),
		"serial_number":              derefString(inst.SerialNumber),
		"instance_charge_type":       derefString(inst.InstanceChargeType),
		"instance_network_type":      derefString(inst.InstanceNetworkType),
		"io_optimized":               derefBool(inst.IoOptimized),
		"host_name":                  derefString(inst.HostName),
		"os_type":                    derefString(inst.OSType),
		"os_name":                    derefString(inst.OSName),
		"key_pair_name":              derefString(inst.KeyPairName),
		"resource_group_id":          derefString(inst.ResourceGroupId),
		"spot_strategy":              derefString(inst.SpotStrategy),
		"credit_specification":       derefString(inst.CreditSpecification),
		"gpu_spec":                   derefString(inst.GPUSpec),
		"gpu_amount":                 derefInt32(inst.GPUAmount),
		"start_time":                 derefString(inst.StartTime),
		"auto_release_time":          derefString(inst.AutoReleaseTime),
		"expired_time":               derefString(inst.ExpiredTime),
		"deletion_protection":        derefBool(inst.DeletionProtection),
		"stopped_mode":               derefString(inst.StoppedMode),
		"internet_charge_type":       derefString(inst.InternetChargeType),
		"internet_max_bandwidth_in":  derefInt32(inst.InternetMaxBandwidthIn),
		"internet_max_bandwidth_out": derefInt32(inst.InternetMaxBandwidthOut),
		"eip_ip_address":             eipIP,
		"eip_allocation_id":          eipAllocationID,
		"eip_bandwidth":              eipBandwidth,
		"public_ip_address":          extractPublicIPs(inst),
		"inner_ip_address":           extractInnerIPs(inst),
		"vpc_id":                     vpcID,
		"vswitch_id":                 vswitchID,
		"nat_ip_address":             natIP,
		"private_ip_address":         privateIPs,
		"security_groups":            extractSecurityGroupIDs(inst),
		"operation_locks":            extractOperationLocks(inst),
		"description":                derefString(inst.Description),
	}

	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID:   derefString(inst.InstanceId),
			ResourceName: derefString(inst.InstanceName),
			Region:       region,
			Zone:         derefString(inst.ZoneId),
			Status:       mapStatus(derefString(inst.Status)),
			Tags:         extractTags(inst),
			CreatedTime:  derefString(inst.CreationTime),
		},
		CloudServiceCode: "ecs",
		CloudServiceName: "云服务器 ECS",
		Extra:            extra,
	}
}

// compile-time interface compliance check
var (
	_ cloud.CloudDiscoveryAdapter = (*AliyunECSAdapter)(nil)
	_ cloud.Client                = (*ecsClientWrapper)(nil)
)
