package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
)

// ==================== 存量 pending 待办迁移（管理面写工具） ====================
//
// BuildPendingWorkflowInventory 只读盘点里明确写了「本端点刻意不提供自动
// 修复通道」，所以 preview 期一直缺一个可执行、可审计、可回滚判断的迁移
// 工具。本文件把「只读建议」升级为「计划 + 执行」两段：
//   - BuildPendingMigrationPlan：纯只读，产出带幂等键的动作清单；
//   - ApplyPendingMigration：默认 dry-run，显式 -apply 才在单个 ent.Tx 里
//     执行；每个动作都做状态幂等复核，重复运行全部 skipped；
//   - 全部查询/更新按租户过滤，孤儿判定复用只读盘点的同一套终态白名单，
//     不另造一套口径；业务对象不存在时不猜状态、不判孤儿。
//
// 保守边界：只自动处理「业务已终态但流程未结束」的孤儿实例（取消其待办 +
// 终结实例），以及经运维显式指定目标用户的未指派待办；陈旧待办只列 review，
// 绝不自动改派或终结正常在途实例。

const (
	MigrationActionCancelTask        = "cancel_task"
	MigrationActionTerminateInstance = "terminate_instance"
	MigrationActionAssignTask        = "assign_task"
	MigrationActionReview            = "review"
	defaultPendingMigrationMaxItems  = 500
	maxPendingMigrationMaxItems      = 2000
	pendingMigrationCancelledStatus  = "cancelled"
	pendingMigrationTerminatedStatus = "terminated"
)

// PendingMigrationOptions 迁移参数。DryRun=true 时只输出计划，不产生写操作。
type PendingMigrationOptions struct {
	TenantID            int
	ActorUserID         int
	Reason              string
	AssigneeID          int // 未指派待办的改派目标；0 表示只列 review，不改派
	StaleThresholdHours int
	MaxItems            int
	DryRun              bool
}

// PendingMigrationItem 单个可审计的迁移动作。
type PendingMigrationItem struct {
	Kind                 string `json:"kind"`
	IdempotencyKey       string `json:"idempotencyKey"`
	Actionable           bool   `json:"actionable"`
	InstanceID           int    `json:"instanceId"`
	InstanceKey          string `json:"instanceKey"`
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	ProcessDefinitionID  int    `json:"processDefinitionId"`
	TaskID               string `json:"taskId,omitempty"`
	TargetEntityID       int    `json:"targetEntityId,omitempty"`
	TargetAssigneeID     int    `json:"targetAssigneeId,omitempty"`
	BusinessKey          string `json:"businessKey,omitempty"`
	Reason               string `json:"reason"`
}

// PendingMigrationPlan 只读动作清单。
type PendingMigrationPlan struct {
	GeneratedAt time.Time               `json:"generatedAt"`
	TenantID    int                     `json:"tenantId"`
	Items       []PendingMigrationItem  `json:"items"`
	Summary     PendingMigrationSummary `json:"summary"`
}

// PendingMigrationSummary 计划汇总。
type PendingMigrationSummary struct {
	OrphanInstances    int  `json:"orphanInstances"`
	CancelTasks        int  `json:"cancelTasks"`
	TerminateInstances int  `json:"terminateInstances"`
	AssignTasks        int  `json:"assignTasks"`
	ReviewOnly         int  `json:"reviewOnly"`
	Truncated          bool `json:"truncated"`
}

// PendingMigrationOutcome 单个动作的执行结果。
type PendingMigrationOutcome struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Kind           string `json:"kind"`
	Status         string `json:"status"` // applied | skipped
	Detail         string `json:"detail,omitempty"`
}

// PendingMigrationResult apply 结果。
type PendingMigrationResult struct {
	DryRun   bool                      `json:"dryRun"`
	TenantID int                       `json:"tenantId"`
	Applied  int                       `json:"applied"`
	Skipped  int                       `json:"skipped"`
	Items    []PendingMigrationOutcome `json:"items"`
}

func pendingMigrationKey(tenantID int, kind string, entityID int) string {
	return fmt.Sprintf("pending-migration:%d:%s:%d", tenantID, kind, entityID)
}

// BuildPendingMigrationPlan 生成只读迁移动作清单。
func BuildPendingMigrationPlan(ctx context.Context, client *ent.Client, opts PendingMigrationOptions) (*PendingMigrationPlan, error) {
	if client == nil {
		return nil, fmt.Errorf("迁移计划需要数据库客户端")
	}
	if opts.TenantID <= 0 {
		return nil, fmt.Errorf("迁移计划需要有效的租户")
	}
	maxItems := opts.MaxItems
	if maxItems <= 0 {
		maxItems = defaultPendingMigrationMaxItems
	}
	if maxItems > maxPendingMigrationMaxItems {
		maxItems = maxPendingMigrationMaxItems
	}

	plan := &PendingMigrationPlan{
		GeneratedAt: time.Now(),
		TenantID:    opts.TenantID,
		Items:       []PendingMigrationItem{},
	}
	add := func(item PendingMigrationItem) {
		if len(plan.Items) >= maxItems {
			plan.Summary.Truncated = true
			return
		}
		plan.Items = append(plan.Items, item)
	}

	instances, err := client.ProcessInstance.Query().Where(
		processinstance.TenantIDEQ(opts.TenantID),
		processinstance.StatusIn(openInstanceStatuses...),
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("列出未结束流程实例失败: %w", err)
	}

	orphanIDs := map[int]bool{}
	for _, instance := range instances {
		businessType, businessID, ok := parseBusinessKey(instance.BusinessKey)
		if !ok {
			continue
		}
		businessStatus, found, err := loadBusinessStatus(ctx, client, businessType, businessID, opts.TenantID)
		if err != nil {
			return nil, fmt.Errorf("读取业务状态失败（%s）: %w", instance.BusinessKey, err)
		}
		if !found || !isTerminalBusinessStatus(businessType, businessStatus) {
			continue
		}
		orphanIDs[instance.ID] = true
		plan.Summary.OrphanInstances++

		tasks, err := client.ProcessTask.Query().Where(
			processtask.TenantIDEQ(opts.TenantID),
			processtask.ProcessInstanceIDEQ(instance.ID),
			processtask.StatusIn(pendingTaskStatuses...),
		).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("列出孤儿实例待办失败: %w", err)
		}
		for _, task := range tasks {
			plan.Summary.CancelTasks++
			add(PendingMigrationItem{
				Kind: MigrationActionCancelTask, Actionable: true,
				IdempotencyKey: pendingMigrationKey(opts.TenantID, MigrationActionCancelTask, task.ID),
				InstanceID:     instance.ID, InstanceKey: instance.ProcessInstanceID,
				ProcessDefinitionKey: instance.ProcessDefinitionKey, ProcessDefinitionID: instance.ProcessDefinitionID,
				TaskID: task.TaskID, TargetEntityID: task.ID, BusinessKey: instance.BusinessKey,
				Reason: fmt.Sprintf("业务 %s 已终态（%s）但流程实例仍未结束，取消遗留待办", instance.BusinessKey, businessStatus),
			})
		}
		plan.Summary.TerminateInstances++
		add(PendingMigrationItem{
			Kind: MigrationActionTerminateInstance, Actionable: true,
			IdempotencyKey: pendingMigrationKey(opts.TenantID, MigrationActionTerminateInstance, instance.ID),
			InstanceID:     instance.ID, InstanceKey: instance.ProcessInstanceID,
			ProcessDefinitionKey: instance.ProcessDefinitionKey, ProcessDefinitionID: instance.ProcessDefinitionID,
			BusinessKey: instance.BusinessKey,
			Reason:      fmt.Sprintf("业务 %s 已终态（%s），终结悬挂的流程实例", instance.BusinessKey, businessStatus),
		})
	}

	// 未指派 / 陈旧的正常在途待办：只处理显式指定目标用户的改派，其余只列 review。
	pendingTasks, err := client.ProcessTask.Query().Where(
		processtask.TenantIDEQ(opts.TenantID),
		processtask.StatusIn(pendingTaskStatuses...),
	).Order(ent.Asc(processtask.FieldCreatedTime)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("列出待办失败: %w", err)
	}
	threshold := opts.StaleThresholdHours
	if threshold <= 0 {
		threshold = DefaultStaleThresholdHours
	}
	if threshold > 720 {
		threshold = 720
	}
	instanceKeys := map[int]*ent.ProcessInstance{}
	for _, instance := range instances {
		instanceKeys[instance.ID] = instance
	}
	for _, task := range pendingTasks {
		if orphanIDs[task.ProcessInstanceID] {
			continue // 已由 cancel_task 覆盖，避免同一待办出现两个冲突动作
		}
		instance := instanceKeys[task.ProcessInstanceID]
		instanceKey, defKey, defID, businessKey := "", "", 0, ""
		if instance != nil {
			instanceKey, defKey, defID, businessKey = instance.ProcessInstanceID, instance.ProcessDefinitionKey, instance.ProcessDefinitionID, instance.BusinessKey
		}
		unassigned := strings.TrimSpace(task.Assignee) == ""
		stale := time.Since(task.CreatedTime).Hours() >= float64(threshold)
		if unassigned && opts.AssigneeID > 0 {
			plan.Summary.AssignTasks++
			add(PendingMigrationItem{
				Kind: MigrationActionAssignTask, Actionable: true,
				IdempotencyKey: pendingMigrationKey(opts.TenantID, MigrationActionAssignTask, task.ID),
				InstanceID:     task.ProcessInstanceID, InstanceKey: instanceKey,
				ProcessDefinitionKey: defKey, ProcessDefinitionID: defID,
				TaskID: task.TaskID, TargetEntityID: task.ID, TargetAssigneeID: opts.AssigneeID,
				BusinessKey: businessKey, Reason: "待办没有处理人，按管理面指定目标改派",
			})
			continue
		}
		if unassigned || stale {
			detail := "待办没有处理人"
			if stale {
				detail = fmt.Sprintf("待办停留超过 %d 小时", threshold)
			}
			plan.Summary.ReviewOnly++
			add(PendingMigrationItem{
				Kind: MigrationActionReview, Actionable: false,
				IdempotencyKey: pendingMigrationKey(opts.TenantID, MigrationActionReview, task.ID),
				InstanceID:     task.ProcessInstanceID, InstanceKey: instanceKey,
				ProcessDefinitionKey: defKey, ProcessDefinitionID: defID,
				TaskID: task.TaskID, TargetEntityID: task.ID, BusinessKey: businessKey,
				Reason: detail + "：需人工复核，工具不自动改派或终结正常在途流程",
			})
		}
	}

	sort.Slice(plan.Items, func(i, j int) bool {
		if plan.Items[i].InstanceID != plan.Items[j].InstanceID {
			return plan.Items[i].InstanceID < plan.Items[j].InstanceID
		}
		if plan.Items[i].Kind != plan.Items[j].Kind {
			return plan.Items[i].Kind < plan.Items[j].Kind
		}
		return plan.Items[i].IdempotencyKey < plan.Items[j].IdempotencyKey
	})
	return plan, nil
}

// ApplyPendingMigration 执行（或仅预演）迁移动作。所有写操作与审计在同一
// ent.Tx 中；任一动作失败整体回滚，不留半迁移状态。
func (e *CustomProcessEngine) ApplyPendingMigration(ctx context.Context, plan *PendingMigrationPlan, opts PendingMigrationOptions) (*PendingMigrationResult, error) {
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("迁移执行需要有效的流程引擎")
	}
	if plan == nil {
		return nil, fmt.Errorf("迁移执行需要计划")
	}
	if plan.TenantID != opts.TenantID || opts.TenantID <= 0 {
		return nil, fmt.Errorf("迁移计划与请求租户不一致，拒绝执行")
	}
	result := &PendingMigrationResult{DryRun: opts.DryRun, TenantID: opts.TenantID, Items: []PendingMigrationOutcome{}}
	if opts.DryRun {
		for _, item := range plan.Items {
			result.Skipped++
			result.Items = append(result.Items, PendingMigrationOutcome{
				IdempotencyKey: item.IdempotencyKey, Kind: item.Kind, Status: "skipped", Detail: "dry-run",
			})
		}
		return result, nil
	}
	if opts.ActorUserID <= 0 {
		return nil, fmt.Errorf("迁移执行需要有效的操作人（写入审计）")
	}
	if strings.TrimSpace(opts.Reason) == "" {
		return nil, fmt.Errorf("迁移执行必须填写原因（写入审计）")
	}

	tx, err := e.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启迁移事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()

	for _, item := range plan.Items {
		if !item.Actionable {
			result.Skipped++
			result.Items = append(result.Items, PendingMigrationOutcome{
				IdempotencyKey: item.IdempotencyKey, Kind: item.Kind, Status: "skipped", Detail: "需人工复核",
			})
			continue
		}
		applied, detail, err := e.applyOneMigrationAction(ctx, txc, item, opts)
		if err != nil {
			return nil, fmt.Errorf("迁移动作 %s 失败: %w", item.IdempotencyKey, err)
		}
		outcome := PendingMigrationOutcome{IdempotencyKey: item.IdempotencyKey, Kind: item.Kind, Detail: detail}
		if applied {
			outcome.Status = "applied"
			result.Applied++
		} else {
			outcome.Status = "skipped"
			result.Skipped++
		}
		result.Items = append(result.Items, outcome)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交迁移事务失败: %w", err)
	}
	return result, nil
}

// applyOneMigrationAction 在事务内做状态幂等复核后执行单个动作并写审计。
func (e *CustomProcessEngine) applyOneMigrationAction(ctx context.Context, txc *ent.Client, item PendingMigrationItem, opts PendingMigrationOptions) (bool, string, error) {
	instance, err := txc.ProcessInstance.Query().Where(
		processinstance.IDEQ(item.InstanceID),
		processinstance.TenantIDEQ(opts.TenantID),
	).Only(ctx)
	if err != nil {
		return false, "", fmt.Errorf("加载流程实例失败: %w", err)
	}

	action := ""
	activityID := ""
	detail := ""
	switch item.Kind {
	case MigrationActionCancelTask:
		task, err := txc.ProcessTask.Query().Where(
			processtask.IDEQ(item.TargetEntityID),
			processtask.TenantIDEQ(opts.TenantID),
			processtask.ProcessInstanceIDEQ(item.InstanceID),
		).Only(ctx)
		if err != nil {
			return false, "", fmt.Errorf("加载待办失败: %w", err)
		}
		if task.Status == pendingMigrationCancelledStatus {
			return false, "待办已取消（幂等跳过）", nil
		}
		if _, err := txc.ProcessTask.UpdateOneID(task.ID).Where(
			processtask.TenantIDEQ(opts.TenantID),
		).SetStatus(pendingMigrationCancelledStatus).Save(ctx); err != nil {
			return false, "", fmt.Errorf("取消待办失败: %w", err)
		}
		action, activityID, detail = AuditActionTaskCancelled, item.TaskID, "取消孤儿实例遗留待办"
	case MigrationActionTerminateInstance:
		if instance.Status != "running" && instance.Status != "suspended" {
			return false, "实例已终结（幂等跳过）", nil
		}
		if _, err := txc.ProcessInstance.UpdateOneID(instance.ID).Where(
			processinstance.TenantIDEQ(opts.TenantID),
		).SetStatus(pendingMigrationTerminatedStatus).Save(ctx); err != nil {
			return false, "", fmt.Errorf("终结流程实例失败: %w", err)
		}
		action, activityID, detail = AuditActionProcessTerminated, "pending_migration", "终结业务已终态的悬挂实例"
	case MigrationActionAssignTask:
		task, err := txc.ProcessTask.Query().Where(
			processtask.IDEQ(item.TargetEntityID),
			processtask.TenantIDEQ(opts.TenantID),
			processtask.ProcessInstanceIDEQ(item.InstanceID),
		).Only(ctx)
		if err != nil {
			return false, "", fmt.Errorf("加载待办失败: %w", err)
		}
		if strings.TrimSpace(task.Assignee) != "" {
			return false, "待办已有处理人（幂等跳过）", nil
		}
		if opts.AssigneeID <= 0 {
			return false, "未指定改派目标（跳过）", nil
		}
		if _, err := txc.ProcessTask.UpdateOneID(task.ID).Where(
			processtask.TenantIDEQ(opts.TenantID),
		).SetAssignee(strconv.Itoa(opts.AssigneeID)).Save(ctx); err != nil {
			return false, "", fmt.Errorf("改派待办失败: %w", err)
		}
		action, activityID, detail = AuditActionTaskAssigned, item.TaskID, "改派未指派待办"
	default:
		return false, "未知动作（跳过）", nil
	}

	if err := e.auditService.RecordAuditWithClient(ctx, txc, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           activityID,
		ActivityType:         "PendingMigration",
		Action:               action,
		UserID:               opts.ActorUserID,
		VariablesBefore:      map[string]interface{}{"reason": opts.Reason},
		VariablesAfter:       map[string]interface{}{"idempotencyKey": item.IdempotencyKey},
		Comment: fmt.Sprintf("存量待办迁移（原因：%s；动作：%s；幂等键：%s）",
			opts.Reason, item.Kind, item.IdempotencyKey),
		TenantID: opts.TenantID,
	}); err != nil {
		return false, "", fmt.Errorf("写入迁移审计失败: %w", err)
	}
	return true, detail, nil
}
