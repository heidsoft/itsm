package aliyun

import (
	"errors"
	"fmt"
	"os"

	"github.com/aliyun/alibabacloud-sdk-go/sdk"
	"github.com/aliyun/alibabacloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibabacloud-sdk-go/services/ecs"
)

var (
	ecsClient *ecs.Client
)

// Init 初始化阿里云 SDK
func Init() error {
	accessKeyID := os.Getenv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	regionID := os.Getenv("ALIYUN_REGION_ID")

	if accessKeyID == "" || accessKeySecret == "" || regionID == "" {
		return errors.New("Aliyun credentials not configured")
	}

	var err error
	ecsClient, err = ecs.NewClientWithAccessKey(regionID, accessKeyID, accessKeySecret)
	if err != nil {
		return fmt.Errorf("Failed to create ECS client: %w", err)
	}

	return nil
}

// CreateInstance 创建 ECS 实例
func CreateInstance(args CreateInstanceArgs) (*ecs.CreateInstanceResponse, error) {
	request := ecs.CreateCreateInstanceRequest()
	request.Scheme = "https"

	// 实例规格
	request.InstanceType = args.InstanceType
	if request.InstanceType == "" {
		request.InstanceType = "ecs.n4.small" // 2 核 4GB
	}

	// 镜像
	request.ImageId = args.ImageID
	if request.ImageId == "" {
		request.ImageId = "ubuntu_20_04_x64_20G_alibase_20210120.vhd"
	}

	// 安全组
	request.SecurityGroupId = args.SecurityGroupID

	// 实例名称
	request.InstanceName = args.InstanceName

	// 网络配置
	request.InternetChargeType = "PayByBandwidth"
	request.InternetMaxBandwidthOut = requests.NewInteger(args.Bandwidth)
	if args.Bandwidth == 0 {
		request.InternetMaxBandwidthOut = requests.NewInteger(3)
	}

	// 系统盘
	request.SystemDiskCategory = "cloud_efficiency"
	request.SystemDiskSize = requests.NewInteger(args.DiskSize)
	if args.DiskSize == 0 {
		request.SystemDiskSize = requests.NewInteger(40)
	}

	// 密码
	request.Password = args.Password

	response, err := ecsClient.CreateInstance(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to create ECS instance: %w", err)
	}

	return response, nil
}

// StartInstance 启动 ECS 实例
func StartInstance(instanceID string) error {
	request := ecs.CreateStartInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = instanceID

	_, err := ecsClient.StartInstance(request)
	if err != nil {
		return fmt.Errorf("Failed to start instance: %w", err)
	}

	return nil
}

// StopInstance 停止 ECS 实例
func StopInstance(instanceID string) error {
	request := ecs.CreateStopInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = instanceID

	_, err := ecsClient.StopInstance(request)
	if err != nil {
		return fmt.Errorf("Failed to stop instance: %w", err)
	}

	return nil
}

// DeleteInstance 删除 ECS 实例
func DeleteInstance(instanceID string) error {
	request := ecs.CreateDeleteInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = instanceID
	request.Force = requests.NewBoolean(true)

	_, err := ecsClient.DeleteInstance(request)
	if err != nil {
		return fmt.Errorf("Failed to delete instance: %w", err)
	}

	return nil
}

// DescribeInstanceStatus 获取实例状态
func DescribeInstanceStatus(instanceID string) (*ecs.DescribeInstanceStatusResponse, error) {
	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = "https"
	request.InstanceId = instanceID

	response, err := ecsClient.DescribeInstanceStatus(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to describe instance status: %w", err)
	}

	return response, nil
}

// DescribeInstances 获取实例详情
func DescribeInstances(instanceIDs []string) (*ecs.DescribeInstancesResponse, error) {
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"
	request.InstanceIds = requests.NewJsonString(instanceIDs)

	response, err := ecsClient.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to describe instances: %w", err)
	}

	return response, nil
}

// CreateInstanceArgs 创建实例参数
type CreateInstanceArgs struct {
	InstanceType    string `json:"instance_type"`    // 实例规格
	ImageID         string `json:"image_id"`         // 镜像 ID
	SecurityGroupID string `json:"security_group_id"` // 安全组 ID
	InstanceName    string `json:"instance_name"`    // 实例名称
	Bandwidth       int    `json:"bandwidth"`        // 带宽 (Mbps)
	DiskSize        int    `json:"disk_size"`        // 磁盘大小 (GB)
	Password        string `json:"password"`         // 密码
}

// InstanceStatus 实例状态
type InstanceStatus struct {
	Status       string  `json:"status"`
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	NetworkIn    int64   `json:"network_in"`
	NetworkOut   int64   `json:"network_out"`
}
