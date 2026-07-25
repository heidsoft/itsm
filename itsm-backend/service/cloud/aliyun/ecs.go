package aliyun

import (
	"context"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
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

func (*AliyunECSAdapter) Provider() string    { return "aliyun" }
func (*AliyunECSAdapter) ServiceCode() string { return "ecs" }
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
	if cred.SessionToken != "" {
		cfg.SecurityToken = tea.String(cred.SessionToken)
	}
	return ecs.NewClient(cfg)
}

func (*AliyunECSAdapter) ListRegions(ctx context.Context, account *ent.CloudAccount) ([]string, error) {
	return []string{"cn-hangzhou", "cn-shanghai", "cn-beijing"}, nil
}

func (a *AliyunECSAdapter) DiscoverRegion(ctx context.Context, account *ent.CloudAccount, region string, client cloud.Client, nextToken string) (*cloud.PageResult, error) {
	ecsClient, ok := client.(*ecsClientWrapper)
	if !ok {
		return nil, nil
	}

	var allResources []cloud.DiscoveredResource
	token := nextToken

	for {
		req := &ecs.DescribeInstancesRequest{}
		req.MaxResults = tea.Int32(100)
		if token != "" {
			req.NextToken = tea.String(token)
		}

		resp, err := ecsClient.ecs.DescribeInstancesWithOptions(req, nil)
		if err != nil {
			return nil, err
		}

		if resp.Body == nil || resp.Body.Instances == nil {
			break
		}

		for _, inst := range resp.Body.Instances.Instance {
			if inst == nil {
				continue
			}
			allResources = append(allResources, transformInstance(inst, region))
		}

		token = tea.StringValue(resp.Body.NextToken)
		if token == "" {
			break
		}
	}

	return &cloud.PageResult{Resources: allResources}, nil
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
	_, err = c.DescribeRegionsWithOptions(&ecs.DescribeRegionsRequest{}, nil)
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
	extra := map[string]interface{}{
		"instance_type":              derefString(inst.InstanceType),
		"cpu":                        derefInt32(inst.Cpu),
		"memory":                     derefInt32(inst.Memory),
		"image_id":                   derefString(inst.ImageId),
		"serial_number":              derefString(inst.SerialNumber),
		"instance_charge_type":       derefString(inst.InstanceChargeType),
		"os_type":                    derefString(inst.OSType),
		"internet_max_bandwidth_in":  derefInt32(inst.InternetMaxBandwidthIn),
		"internet_max_bandwidth_out": derefInt32(inst.InternetMaxBandwidthOut),
		"eip_ip_address":             derefString(inst.EipAddress.IpAddress),
		"public_ip_address":          extractPublicIPs(inst),
		"inner_ip_address":           extractInnerIPs(inst),
		"vpc_id":                     derefString(inst.VpcAttributes.VpcId),
		"vswitch_id":                 derefString(inst.VpcAttributes.VSwitchId),
		"security_groups":            extractSecurityGroupIDs(inst),
		"instance_network_type":      derefString(inst.InstanceNetworkType),
		"description":                derefString(inst.Description),
		"expired_time":               derefString(inst.ExpiredTime),
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
