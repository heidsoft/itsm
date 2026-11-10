// Package dto 的存量待办盘点值类型：批次 3 第 5 步从共享层下沉。
// 定义唯一来源在此；共享层保留同名别名，handlers/bpmn 只读本包。
package dto

import "time"

// PendingInventorySummary 存量待办盘点响应值类型（与共享层同名别名共享类型身份）。
type PendingInventorySummary struct {
	RunningInstances int `json:"runningInstances"` // 未结束实例（running+suspended）
	PendingTasks     int `json:"pendingTasks"`     // 待办任务总数（不受列表截断影响）
	StaleTasks       int `json:"staleTasks"`       // 停留超过阈值的待办数
	UnassignedTasks  int `json:"unassignedTasks"`  // 无处理人的待办数
	OrphanInstances  int `json:"orphanInstances"`  // 业务已终态但流程未结束的实例数
}

// PendingTaskItem 存量待办盘点响应值类型（与共享层同名别名共享类型身份）。
type PendingTaskItem struct {
	TaskID      string  `json:"taskId"`
	TaskName    string  `json:"taskName"`
	TaskType    string  `json:"taskType"`
	Status      string  `json:"status"`
	Assignee    string  `json:"assignee"`
	ProcessKey  string  `json:"processKey"`
	NodeKey     string  `json:"nodeKey"`
	InstanceID  int     `json:"instanceId"`
	InstanceKey string  `json:"instanceKey"`
	BusinessKey string  `json:"businessKey"`
	CreatedAt   string  `json:"createdAt"`
	DwellHours  float64 `json:"dwellHours"`
	Stale       bool    `json:"stale"`
}

// PendingProcessGroup 存量待办盘点响应值类型（与共享层同名别名共享类型身份）。
type PendingProcessGroup struct {
	ProcessKey       string  `json:"processKey"`
	PendingTasks     int     `json:"pendingTasks"`
	StaleTasks       int     `json:"staleTasks"`
	OldestDwellHours float64 `json:"oldestDwellHours"`
}

// OrphanInstanceItem 存量待办盘点响应值类型（与共享层同名别名共享类型身份）。
type OrphanInstanceItem struct {
	InstanceID     int     `json:"instanceId"`
	InstanceKey    string  `json:"instanceKey"`
	BusinessKey    string  `json:"businessKey"`
	BusinessType   string  `json:"businessType"`
	BusinessID     int     `json:"businessId"`
	BusinessStatus string  `json:"businessStatus"`
	InstanceStatus string  `json:"instanceStatus"`
	ProcessKey     string  `json:"processKey"`
	StartedAt      string  `json:"startedAt"`
	DwellHours     float64 `json:"dwellHours"`
	Reason         string  `json:"reason"`
}

// PendingWorkflowInventory 存量待办盘点响应值类型（与共享层同名别名共享类型身份）。
type PendingWorkflowInventory struct {
	GeneratedAt         time.Time               `json:"generatedAt"`
	TenantID            int                     `json:"tenantId"`
	StaleThresholdHours int                     `json:"staleThresholdHours"`
	Summary             PendingInventorySummary `json:"summary"`
	PendingTasks        []PendingTaskItem       `json:"pendingTasks"`
	ProcessGroups       []PendingProcessGroup   `json:"processGroups"`
	OrphanInstances     []OrphanInstanceItem    `json:"orphanInstances"`
	Recommendations     []string                `json:"recommendations"`
}
