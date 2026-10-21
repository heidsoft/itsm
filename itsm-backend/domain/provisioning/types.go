package provisioning

// TaskStatus 交付任务状态
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
)

// MaxTaskAttempts 单个交付任务的最大执行次数（含首次）。
//
// 交付是「对真实基础设施产生副作用」的动作，重试不是越多越好：云厂商的幂等
// 能力参差不齐（创建 ECS 重复调用可能真的开出两台机器），因此重试必须封顶，
// 超出后转人工判定。5 次与 commandbus 入箱时的 MaxAttempts 同值，避免出现
// 「队列还在重投、任务层已经拒绝」两套口径。
const MaxTaskAttempts = 5

// CanExecute 报告任务当前是否允许（重新）执行。
//
// succeeded 不允许再执行：重复执行等于重复交付，是这套链路里代价最高的一种错误。
// running 不允许：执行权由原子领取（pending|failed → running）独占，
// 并发执行同样会造成重复交付。
func CanExecute(s TaskStatus) bool {
	return s == TaskPending || s == TaskFailed
}

// CanRetry 报告任务是否满足重试的前提：上次失败且未达上限。
//
// pending 不在其中——它已经排在被执行的位置上，重试没有语义（只会多入一条命令）。
func CanRetry(s TaskStatus, attemptCount int) bool {
	return s == TaskFailed && attemptCount < MaxTaskAttempts
}

// CanResolveManually 报告任务是否允许人工接管判定。
//
// running 不允许：执行中的任务即将回写自己的结果，人工判定会被覆盖。
// succeeded 不允许：已经交付完成，再判定一次只会制造互相矛盾的记录。
func CanResolveManually(s TaskStatus) bool {
	return s == TaskFailed || s == TaskPending
}

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
