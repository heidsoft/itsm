package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/release"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/ticket"
)

// ==================== 存量待办盘点（管理面只读） ====================
//
// 背景：BPMN 预览收口过程中发现三类「业务已推进、流程未跟进」的存量分叉
// （事件/问题域接桥前创建的实例、未启用 outbox 时期创建的实例）。
// 本工具把它们变成可查询、可决策的事实清单：
//   - 待办分布：按流程/节点聚合的未处理任务与停留时长；
//   - 孤儿实例：业务对象已终态但流程实例仍未结束；
//   - 迁移建议：只读文本，不执行任何自动迁移（终结/改派都是写操作，
//     必须由人经既有写端点执行，本端点刻意不提供自动修复通道）。
//
// 安全口径：所有查询按租户过滤；业务对象不存在（已物理删）时不猜状态，
// 只标注 unknown，不判孤儿。

// PendingInventorySummary 盘点汇总计数。
type PendingInventorySummary struct {
	RunningInstances int `json:"runningInstances"` // 未结束实例（running+suspended）
	PendingTasks     int `json:"pendingTasks"`     // 待办任务总数（不受列表截断影响）
	StaleTasks       int `json:"staleTasks"`       // 停留超过阈值的待办数
	UnassignedTasks  int `json:"unassignedTasks"`  // 无处理人的待办数
	OrphanInstances  int `json:"orphanInstances"`  // 业务已终态但流程未结束的实例数
}

// PendingTaskItem 单条待办任务。
type PendingTaskItem struct {
	TaskID           string  `json:"taskId"`
	TaskName         string  `json:"taskName"`
	TaskType         string  `json:"taskType"`
	Status           string  `json:"status"`
	Assignee         string  `json:"assignee"`
	ProcessKey       string  `json:"processKey"`
	NodeKey          string  `json:"nodeKey"`
	InstanceID       int     `json:"instanceId"`
	InstanceKey      string  `json:"instanceKey"`
	BusinessKey      string  `json:"businessKey"`
	CreatedAt        string  `json:"createdAt"`
	DwellHours       float64 `json:"dwellHours"`
	Stale            bool    `json:"stale"`
}

// PendingProcessGroup 按流程定义聚合的待办分布。
type PendingProcessGroup struct {
	ProcessKey        string  `json:"processKey"`
	PendingTasks      int     `json:"pendingTasks"`
	StaleTasks        int     `json:"staleTasks"`
	OldestDwellHours  float64 `json:"oldestDwellHours"`
}

// OrphanInstanceItem 孤儿实例：业务对象已终态、流程实例仍未结束。
type OrphanInstanceItem struct {
	InstanceID      int     `json:"instanceId"`
	InstanceKey     string  `json:"instanceKey"`
	BusinessKey     string  `json:"businessKey"`
	BusinessType    string  `json:"businessType"`
	BusinessID      int     `json:"businessId"`
	BusinessStatus  string  `json:"businessStatus"`
	InstanceStatus  string  `json:"instanceStatus"`
	ProcessKey      string  `json:"processKey"`
	StartedAt       string  `json:"startedAt"`
	DwellHours      float64 `json:"dwellHours"`
	Reason          string  `json:"reason"`
}

// PendingWorkflowInventory 盘点结果。
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

// pendingTaskStatuses 待办状态集合：created/assigned/started/delegated
// （completed/cancelled/timeout/escalated 均非待办；escalated 是通知升级态，
// 任务本身仍可能在原状态队列里，不单独计入）。
var pendingTaskStatuses = []string{
	common.ProcessTaskStatusCreated,
	common.ProcessTaskStatusAssigned,
	common.ProcessTaskStatusStarted,
	common.ProcessTaskStatusDelegated,
}

// openInstanceStatuses 未结束的实例状态集合。
var openInstanceStatuses = []string{"running", "suspended"}

// maxInventoryTasks 待办列表截断上限：盘点是管理面诊断，不是分页浏览入口，
// 全量铺开会把响应体撑爆；被截断的数量在 summary.pendingTasks 里如实呈现，
// 明细可用 /bpmn/tasks/all 分页查看。
const maxInventoryTasks = 200

// DefaultStaleThresholdHours 默认陈旧阈值（与 GA 就绪检查的 24h 口径一致）。
// 导出：handler 与测试共用同一默认值，避免两处漂移。
const DefaultStaleThresholdHours = 24

// terminalBusinessStatuses 各业务域终态集合（按 businessKey 前缀索引）。
// 来源为各域状态机白名单的「无出边」状态：
//   - ticket/incident：closed、cancelled（constants.go 状态机终态）；
//   - problem：closed（resolved 仍可回退 investigating，不算终态）；
//   - change：closed、cancelled、rejected（completed/rolled_back 仍可归档为 closed）；
//   - release：completed、cancelled、rolled_back（failed 可重新排期，不算终态）；
//   - service_request：delivered、rejected、cancelled。
var terminalBusinessStatuses = map[string]map[string]struct{}{
	"ticket": {
		common.TicketStatusClosed:    {},
		common.TicketStatusCancelled: {},
	},
	"incident": {
		common.IncidentStatusClosed:    {},
		common.IncidentStatusCancelled: {},
	},
	"problem": {
		common.ProblemStatusClosed: {},
	},
	"change": {
		common.ChangeStatusClosed:     {},
		common.ChangeStatusCancelled:  {},
		common.ChangeStatusRejected:   {},
	},
	"release": {
		"completed":   {},
		"cancelled":   {},
		"rolled_back": {},
	},
	"service_request": {
		"delivered": {},
		"rejected":  {},
		"cancelled": {},
	},
}

// BuildPendingWorkflowInventory 生成租户级存量待办盘点（只读）。
// staleThresholdHours<=0 时取默认 24h；上限 720h（30 天），防止误传把一切标脏。
func BuildPendingWorkflowInventory(ctx context.Context, client *ent.Client, tenantID int, staleThresholdHours int) (*PendingWorkflowInventory, error) {
	if client == nil {
		return nil, fmt.Errorf("inventory requires a database client")
	}
	if tenantID <= 0 {
		return nil, fmt.Errorf("inventory requires a valid tenant")
	}
	if staleThresholdHours <= 0 {
		staleThresholdHours = DefaultStaleThresholdHours
	}
	if staleThresholdHours > 720 {
		staleThresholdHours = 720
	}
	now := time.Now()

	inventory := &PendingWorkflowInventory{
		GeneratedAt:         now,
		TenantID:            tenantID,
		StaleThresholdHours: staleThresholdHours,
		PendingTasks:        []PendingTaskItem{},
		ProcessGroups:       []PendingProcessGroup{},
		OrphanInstances:     []OrphanInstanceItem{},
	}

	// 1. 待办任务总数（不受列表截断影响）。
	totalPending, err := client.ProcessTask.Query().Where(
		processtask.TenantIDEQ(tenantID),
		processtask.StatusIn(pendingTaskStatuses...),
	).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count pending tasks: %w", err)
	}
	inventory.Summary.PendingTasks = totalPending

	// 2. 待办明细（按创建时间升序，最旧的排前面——盘点首先看的是积压）。
	tasks, err := client.ProcessTask.Query().Where(
		processtask.TenantIDEQ(tenantID),
		processtask.StatusIn(pendingTaskStatuses...),
	).Order(ent.Asc(processtask.FieldCreatedTime)).Limit(maxInventoryTasks).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending tasks: %w", err)
	}

	groupIndex := map[string]*PendingProcessGroup{}
	for _, task := range tasks {
		createdAt := task.CreatedTime
		dwell := now.Sub(createdAt).Hours()
		if dwell < 0 {
			dwell = 0
		}
		stale := dwell >= float64(staleThresholdHours)
		if stale {
			inventory.Summary.StaleTasks++
		}
		if strings.TrimSpace(task.Assignee) == "" {
			inventory.Summary.UnassignedTasks++
		}
		inventory.PendingTasks = append(inventory.PendingTasks, PendingTaskItem{
			TaskID:      task.TaskID,
			TaskName:    task.TaskName,
			TaskType:    task.TaskType,
			Status:      task.Status,
			Assignee:    task.Assignee,
			ProcessKey:  task.ProcessDefinitionKey,
			NodeKey:     task.TaskDefinitionKey,
			InstanceID:  task.ProcessInstanceID,
			CreatedAt:   createdAt.Format(time.RFC3339),
			DwellHours:  roundHours(dwell),
			Stale:       stale,
		})
		group, ok := groupIndex[task.ProcessDefinitionKey]
		if !ok {
			group = &PendingProcessGroup{ProcessKey: task.ProcessDefinitionKey}
			groupIndex[task.ProcessDefinitionKey] = group
		}
		group.PendingTasks++
		if stale {
			group.StaleTasks++
		}
		if dwell > group.OldestDwellHours {
			group.OldestDwellHours = roundHours(dwell)
		}
	}
	for _, group := range groupIndex {
		inventory.ProcessGroups = append(inventory.ProcessGroups, *group)
	}
	sort.Slice(inventory.ProcessGroups, func(i, j int) bool {
		return inventory.ProcessGroups[i].PendingTasks > inventory.ProcessGroups[j].PendingTasks
	})

	// 3. 未结束实例与孤儿识别。
	instances, err := client.ProcessInstance.Query().Where(
		processinstance.TenantIDEQ(tenantID),
		processinstance.StatusIn(openInstanceStatuses...),
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open process instances: %w", err)
	}
	inventory.Summary.RunningInstances = len(instances)

	for _, instance := range instances {
		businessType, businessID, ok := parseBusinessKey(instance.BusinessKey)
		if !ok {
			continue
		}
		businessStatus, found, err := loadBusinessStatus(ctx, client, businessType, businessID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("load business status for %s: %w", instance.BusinessKey, err)
		}
		if !found {
			// 业务对象不存在（物理删或跨租户）：不猜状态，不判孤儿。
			continue
		}
		if !isTerminalBusinessStatus(businessType, businessStatus) {
			continue
		}
		dwell := now.Sub(instance.StartTime).Hours()
		if dwell < 0 {
			dwell = 0
		}
		inventory.Summary.OrphanInstances++
		inventory.OrphanInstances = append(inventory.OrphanInstances, OrphanInstanceItem{
			InstanceID:     instance.ID,
			InstanceKey:    instance.ProcessInstanceID,
			BusinessKey:    instance.BusinessKey,
			BusinessType:   businessType,
			BusinessID:     businessID,
			BusinessStatus: businessStatus,
			InstanceStatus: instance.Status,
			ProcessKey:     instance.ProcessDefinitionKey,
			StartedAt:      instance.StartTime.Format(time.RFC3339),
			DwellHours:     roundHours(dwell),
			Reason: fmt.Sprintf("业务对象已终态（%s）但流程实例仍未结束（%s）",
				businessStatus, instance.Status),
		})
	}

	inventory.Recommendations = buildInventoryRecommendations(inventory)
	return inventory, nil
}

// buildInventoryRecommendations 生成只读建议（不包含任何自动迁移动作）。
func buildInventoryRecommendations(inv *PendingWorkflowInventory) []string {
	recommendations := []string{}
	if inv.Summary.OrphanInstances > 0 {
		recommendations = append(recommendations, fmt.Sprintf(
			"发现 %d 个孤儿实例（业务已终态、流程未结束）：确认业务无误后，由管理员经 PUT /bpmn/process-instances/:id/terminate 人工终结；若业务状态有误，先修正业务再继续流程，避免双向分叉",
			inv.Summary.OrphanInstances))
	}
	if inv.Summary.StaleTasks > 0 {
		recommendations = append(recommendations, fmt.Sprintf(
			"%d 个待办停留超过 %d 小时：可经 PUT /bpmn/tasks/:id/reassign 改派或 /bpmn/tasks/:id/claim 认领后督办",
			inv.Summary.StaleTasks, inv.StaleThresholdHours))
	}
	if inv.Summary.UnassignedTasks > 0 {
		recommendations = append(recommendations, fmt.Sprintf(
			"%d 个待办没有处理人：先经 PUT /bpmn/tasks/:id/assign 分配，否则停留计时失真",
			inv.Summary.UnassignedTasks))
	}
	if inv.Summary.PendingTasks == 0 && inv.Summary.RunningInstances == 0 {
		recommendations = append(recommendations, "当前租户没有未完成的流程实例与待办任务，无需迁移")
	}
	recommendations = append(recommendations,
		"本端点为只读盘点，不执行任何自动迁移；终结/改派均为写操作，请经既有写端点人工执行")
	return recommendations
}

// parseBusinessKey 解析 businessKey（`<type>:<id>`）。
// 无法解析（如存量 approval:<id> 键）时 ok=false，调用方跳过而非误判。
func parseBusinessKey(businessKey string) (businessType string, businessID int, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(businessKey), ":", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return strings.ToLower(strings.TrimSpace(parts[0])), id, true
}

// isTerminalBusinessStatus 判断业务状态是否属于该域终态集合。
func isTerminalBusinessStatus(businessType, status string) bool {
	terminals, ok := terminalBusinessStatuses[businessType]
	if !ok {
		return false
	}
	_, terminal := terminals[status]
	return terminal
}

// loadBusinessStatus 读取业务对象当前状态。found=false 表示对象不存在
// （物理删或跨租户），调用方不得据此推断孤儿。
func loadBusinessStatus(ctx context.Context, client *ent.Client, businessType string, businessID, tenantID int) (status string, found bool, err error) {
	switch businessType {
	case "ticket":
		entity, qErr := client.Ticket.Query().Where(ticket.IDEQ(businessID), ticket.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return entity.Status, true, nil
	case "incident":
		entity, qErr := client.Incident.Query().Where(incident.IDEQ(businessID), incident.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return entity.Status, true, nil
	case "problem":
		entity, qErr := client.Problem.Query().Where(problem.IDEQ(businessID), problem.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return entity.Status, true, nil
	case "change":
		entity, qErr := client.Change.Query().Where(change.IDEQ(businessID), change.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return string(entity.Status), true, nil
	case "release":
		entity, qErr := client.Release.Query().Where(release.IDEQ(businessID), release.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return string(entity.Status), true, nil
	case "service_request":
		entity, qErr := client.ServiceRequest.Query().Where(servicerequest.IDEQ(businessID), servicerequest.TenantIDEQ(tenantID)).Only(ctx)
		if qErr != nil {
			if ent.IsNotFound(qErr) {
				return "", false, nil
			}
			return "", false, qErr
		}
		return entity.Status, true, nil
	default:
		// 未登记的业务域（如 approval:<id> 存量键）：不猜状态。
		return "", false, nil
	}
}

// roundHours 停留时长保留一位小数，避免 JSON 里出现 23.999999999999936 这类噪声。
// 负值（时钟回拨）钳为 0：调用方虽已钳制，函数自身保持防御性。
func roundHours(hours float64) float64 {
	if hours < 0 {
		hours = 0
	}
	return float64(int64(hours*10+0.5)) / 10
}
