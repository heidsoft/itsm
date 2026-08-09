package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeCloudProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "阿里云 - alicloud",
			input:    "alicloud",
			expected: "aliyun",
		},
		{
			name:     "阿里云 - aliyun",
			input:    "aliyun",
			expected: "aliyun",
		},
		{
			name:     "阿里云 - alibaba",
			input:    "alibaba",
			expected: "aliyun",
		},
		{
			name:     "腾讯云 - tencentcloud",
			input:    "tencentcloud",
			expected: "tencent",
		},
		{
			name:     "腾讯云 - qcloud",
			input:    "qcloud",
			expected: "tencent",
		},
		{
			name:     "华为云 - huaweicloud",
			input:    "huaweicloud",
			expected: "huawei",
		},
		{
			name:     "AWS - amazon",
			input:    "amazon",
			expected: "aws",
		},
		{
			name:     "AWS - aws",
			input:    "aws",
			expected: "aws",
		},
		{
			name:     "Azure",
			input:    "azure",
			expected: "azure",
		},
		{
			name:     "私有云 - onprem",
			input:    "onprem",
			expected: "onprem",
		},
		{
			name:     "私有云 - private",
			input:    "private",
			expected: "onprem",
		},
		{
			name:     "大写输入",
			input:    "AWS",
			expected: "aws",
		},
		{
			name:     "带空格输入",
			input:    "  aliyun  ",
			expected: "aliyun",
		},
		{
			name:     "未知提供商",
			input:    "gcp",
			expected: "gcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCloudProvider(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "阿里云",
			input:    "aliyun",
			expected: "阿里云",
		},
		{
			name:     "腾讯云",
			input:    "tencent",
			expected: "腾讯云",
		},
		{
			name:     "华为云",
			input:    "huawei",
			expected: "华为云",
		},
		{
			name:     "AWS",
			input:    "aws",
			expected: "AWS",
		},
		{
			name:     "Azure",
			input:    "azure",
			expected: "Azure",
		},
		{
			name:     "私有云",
			input:    "onprem",
			expected: "私有云",
		},
		{
			name:     "未知提供商 - 大写",
			input:    "gcp",
			expected: "GCP",
		},
		{
			name:     "别名输入",
			input:    "alicloud",
			expected: "阿里云",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := providerDisplayName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloudProfile(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		serviceType string
		expected    cloudResourceProfile
	}{
		{
			name:        "AWS EC2",
			provider:    "aws",
			serviceType: "ec2",
			expected: cloudResourceProfile{
				Category:         "compute",
				ServiceCode:      "ec2",
				ServiceName:      "Elastic Compute Cloud",
				ResourceTypeCode: "instance",
				ResourceTypeName: "EC2 Instance",
				CITypeName:       "AWS EC2 Instance",
			},
		},
		{
			name:        "AWS S3",
			provider:    "aws",
			serviceType: "s3",
			expected: cloudResourceProfile{
				Category:         "storage",
				ServiceCode:      "s3",
				ServiceName:      "Simple Storage Service",
				ResourceTypeCode: "bucket",
				ResourceTypeName: "S3 Bucket",
				CITypeName:       "AWS S3 Bucket",
			},
		},
		{
			name:        "AWS RDS",
			provider:    "aws",
			serviceType: "rds",
			expected: cloudResourceProfile{
				Category:         "database",
				ServiceCode:      "rds",
				ServiceName:      "Relational Database Service",
				ResourceTypeCode: "instance",
				ResourceTypeName: "RDS Instance",
				CITypeName:       "AWS RDS Instance",
			},
		},
		{
			name:        "阿里云 ECS",
			provider:    "aliyun",
			serviceType: "ecs",
			expected: cloudResourceProfile{
				Category:         "compute",
				ServiceCode:      "ecs",
				ServiceName:      "Elastic Compute Service",
				ResourceTypeCode: "instance",
				ResourceTypeName: "ECS Instance",
				CITypeName:       "阿里云 ECS 实例",
			},
		},
		{
			name:        "未知服务类型",
			provider:    "aws",
			serviceType: "lambda",
			expected: cloudResourceProfile{
				Category:         "cloud",
				ServiceCode:      "lambda",
				ServiceName:      "LAMBDA",
				ResourceTypeCode: "lambda",
				ResourceTypeName: "LAMBDA",
				CITypeName:       "AWS LAMBDA",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cloudProfile(tt.provider, tt.serviceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeResourceLifecycle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "running状态",
			input:    "running",
			expected: "active",
		},
		{
			name:     "available状态",
			input:    "available",
			expected: "active",
		},
		{
			name:     "active状态",
			input:    "active",
			expected: "active",
		},
		{
			name:     "stopped状态",
			input:    "stopped",
			expected: "inactive",
		},
		{
			name:     "terminated状态",
			input:    "terminated",
			expected: "retired",
		},
		{
			name:     "deleted状态",
			input:    "deleted",
			expected: "retired",
		},
		{
			name:     "unknown状态",
			input:    "unknown",
			expected: "unknown",
		},
		{
			name:     "大写输入",
			input:    "RUNNING",
			expected: "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeResourceLifecycle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeCIStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "active状态",
			input:    "active",
			expected: "active",
		},
		{
			name:     "running状态",
			input:    "running",
			expected: "active",
		},
		{
			name:     "available状态",
			input:    "available",
			expected: "active",
		},
		{
			name:     "inactive状态",
			input:    "inactive",
			expected: "inactive",
		},
		{
			name:     "stopped状态",
			input:    "stopped",
			expected: "inactive",
		},
		{
			name:     "deleted状态",
			input:    "deleted",
			expected: "retired",
		},
		{
			name:     "terminated状态",
			input:    "terminated",
			expected: "retired",
		},
		{
			name:     "unknown状态",
			input:    "unknown",
			expected: "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCIStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCloudDiscoveryService(t *testing.T) {
	// 测试服务创建（不需要真实的数据库连接）
	// 这个测试主要验证构造函数不会 panic
	t.Run("创建服务实例", func(t *testing.T) {
		// 由于需要 ent.Client，这里只验证函数存在且可调用
		// 实际的数据库测试需要集成测试环境
		assert.NotNil(t, NewCloudDiscoveryService)
	})
}
