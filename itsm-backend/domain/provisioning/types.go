package provisioning

// TaskStatus 交付任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
)

// Provider 云厂商
type Provider string

const (
	ProviderAlicloud Provider = "alicloud"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceECS ResourceType = "ecs"
	ResourceRDS ResourceType = "rds"
	ResourceOSS ResourceType = "oss"
	ResourceVPC ResourceType = "vpc"
	ResourceIAM ResourceType = "iam"
)


