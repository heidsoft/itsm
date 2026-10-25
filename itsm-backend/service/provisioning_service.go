package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/domain/provisioning"
	domainServiceRequest "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/provisioningtask"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/infrastructure/cloud"
	cloudAlicloud "itsm-backend/infrastructure/cloud/alicloud"
	"itsm-backend/internal/commandbus"

	"go.uber.org/zap"
)

// 交付任务的执行权冲突（2026-10-21）。
//
// 这组哨兵错误的存在理由：交付是「对真实基础设施产生副作用」的动作，
// 重复执行的代价不是多一次日志，而是多一台机器。调用方必须能区分
// 「已经做完了」「别人正在做」「重试次数用完了」「这个状态不能重试」，
// 而不是统一压成 500 让前端去猜。
var (
	// ErrProvisioningTaskSucceeded 任务已交付成功，拒绝再次执行（幂等空操作）。
	ErrProvisioningTaskSucceeded = errors.New("交付任务已完成，拒绝重复执行")
	// ErrProvisioningTaskInProgress 任务正在执行中（或刚被其他 worker 领取）。
	ErrProvisioningTaskInProgress = errors.New("交付任务正在执行中")
	// ErrProvisioningTaskAttemptsExhausted 重试次数已达上限，必须转人工判定。
	ErrProvisioningTaskAttemptsExhausted = errors.New("交付任务重试次数已达上限，需人工接管")
	// ErrProvisioningTaskNotRetryable 当前状态不具备重试语义（如尚未失败的 pending）。
	ErrProvisioningTaskNotRetryable = errors.New("交付任务当前状态不允许重试")
	// ErrProvisioningTaskNotResolvable 当前状态不允许人工判定（running/succeeded）。
	ErrProvisioningTaskNotResolvable = errors.New("交付任务当前状态不允许人工判定")
)

// staleTaskLease 判定「running 但已无人执行」的租约时长。
//
// 没有这个租约，worker 在 provider 调用中途崩溃（进程被杀、节点回收）会让任务
// 永久停在 running：既不能执行（状态守卫），也不能人工接管（running 不可判定），
// 交付链路就此卡死。租约到期即视为可重新领取——provider 侧必须靠调用方的
// 幂等键去重，这一点在 payload 中已经传递。
const staleTaskLease = 10 * time.Minute

// ProvisioningService（应用层）：服务请求 -> 交付任务 -> 执行 -> 状态回写
// M2：先实现可运行骨架（Stub），后续接入阿里云真实交付。
type ProvisioningService struct {
	client   *ent.Client
	logger   *zap.SugaredLogger
	provider cloud.Provider
}

func NewProvisioningService(client *ent.Client, logger *zap.SugaredLogger) *ProvisioningService {
	return &ProvisioningService{
		client:   client,
		logger:   logger,
		provider: cloudAlicloud.NewStubProvider(),
	}
}

// CreateTaskFromServiceRequest 仅创建交付任务、把 ServiceRequest 置为 provisioning，
// 并把"执行任务"作为 CommandExecuteProvisioningTask 入箱。
// worker 重载任务后调 ExecuteTask；这样请求结束 / 进程退出后仍可重试到 max_attempts，
// 避免同步执行在网络抖动时丢交付任务。幂等键按 task ID 锁定，replay 不会重复执行。
func (s *ProvisioningService) CreateTaskFromServiceRequest(ctx context.Context, serviceRequestID, tenantID, actorUserID int) (*ent.ProvisioningTask, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	sr, err := tx.ServiceRequest.Query().
		Where(servicerequest.ID(serviceRequestID), servicerequest.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		// %w 保留 ent.NotFoundError：handler 的 classifyError 需要靠它把
		// 跨租户/不存在统一呈现为 404，而不是兜底成 500。
		return nil, fmt.Errorf("服务请求不存在: %w", err)
	}
	if string(sr.Status) != domainServiceRequest.StatusSecurityApproved {
		// 非法状态迁移按仓库语义映射 Conflict(4090)，不得兜底成 400/500。
		return nil, common.NewBusinessError(common.ConflictCode, "当前状态不允许启动交付（需要 security_approved）", "")
	}

	// payload：优先使用 form_data；若为空则降级为空 map
	payload := map[string]any{}
	if sr.FormData != nil {
		payload = sr.FormData
	}

	// 最小推断资源类型（后续基于 catalog/category 或专用字段）
	resourceType := string(provisioning.ResourceECS)
	if v, ok := payload["resource_type"]; ok {
		resourceType = strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	}

	// 原子占用：只有仍处于 security_approved 的服务请求才能被推进到 provisioning。
	// 必须先 CAS 再建任务——若按「先建 task、最后无条件改状态」的顺序，两个并发
	// 调用会各自创建一条交付任务，同一份资源被交付两次。CAS 的 RowsAffected
	// 让输的一方在写任务前就失败。
	affected, err := tx.ServiceRequest.Update().
		Where(
			servicerequest.ID(serviceRequestID),
			servicerequest.TenantID(tenantID),
			servicerequest.StatusEQ(domainServiceRequest.StatusSecurityApproved),
		).
		SetStatus("provisioning").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
	}
	if affected == 0 {
		return nil, common.NewBusinessError(common.ConflictCode, "服务请求已被其他操作推进，请刷新后重试", "")
	}

	task, err := tx.ProvisioningTask.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(serviceRequestID).
		SetProvider(string(provisioning.ProviderAlicloud)).
		SetResourceType(resourceType).
		SetPayload(payload).
		SetStatus(string(provisioning.TaskPending)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建交付任务失败: %w", err)
	}

	// 入箱：worker 重载 task 后调 ExecuteTask 实际跑 provider。同一 task ID
	// 作为 idempotency_key，replay / 重试不会创建新任务。
	if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID:       tenantID,
		CommandType:    commandbus.CommandExecuteProvisioningTask,
		AggregateType:  "provisioning_task",
		AggregateID:    task.ID,
		IdempotencyKey: provisioningExecuteKey(task.ID, 1),
		MaxAttempts:    provisioning.MaxTaskAttempts,
		Payload: map[string]interface{}{
			"serviceRequestId": serviceRequestID,
			"taskId":           task.ID,
			"actorUserId":      actorUserID,
		},
	}); err != nil {
		return nil, fmt.Errorf("入箱交付任务执行命令失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	return task, nil
}

// provisioningExecuteKey 每次尝试一个独立的幂等键。
//
// 这不是可以省掉的细节：入箱层按 idempotency_key 去重，若重试复用首次的键，
// 命令会被判为「已处理」直接跳过——接口返回 200、任务永远停在 failed，
// 是那种「看起来重试成功、其实什么都没发生」的故障。
func provisioningExecuteKey(taskID, attempt int) string {
	return fmt.Sprintf("provisioning_task:%d:execute:%d", taskID, attempt)
}

// ExecuteTask 执行交付任务（Stub），并回写 ServiceRequest 状态为 delivered/failed。
//
// 执行权靠一次原子领取获得（pending|failed|过期 running → running），因此：
//   - 并发调用只有一个能真正执行，其余拿到 ErrProvisioningTaskInProgress；
//   - 已完成任务再次调用是幂等空操作（ErrProvisioningTaskSucceeded），不会重复交付；
//   - worker 崩溃留下的 running 由 staleTaskLease 兜底回收。
func (s *ProvisioningService) ExecuteTask(ctx context.Context, taskID, tenantID, actorUserID int) (*ent.ProvisioningTask, error) {
	// 读任务
	task, err := s.client.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		// 同上：保留 ent.NotFoundError 分类，让「任务不存在/跨租户」表现为 404。
		return nil, fmt.Errorf("交付任务不存在: %w", err)
	}

	switch provisioning.TaskStatus(task.Status) {
	case provisioning.TaskSucceeded:
		return task, ErrProvisioningTaskSucceeded
	case provisioning.TaskRunning:
		// 只有租约过期的 running 才允许重新领取（见下方 claimable 条件）；
		// 这里的提前返回只是省一次写操作，真正的并发保护在原子领取上。
		if task.LastAttemptAt == nil || time.Since(*task.LastAttemptAt) < staleTaskLease {
			return task, ErrProvisioningTaskInProgress
		}
	}
	if task.AttemptCount >= provisioning.MaxTaskAttempts {
		return task, ErrProvisioningTaskAttemptsExhausted
	}
	if !provisioning.CanExecute(provisioning.TaskStatus(task.Status)) &&
		provisioning.TaskStatus(task.Status) != provisioning.TaskRunning {
		return task, ErrProvisioningTaskNotRetryable
	}

	now := time.Now()
	cutoff := now.Add(-staleTaskLease)

	// 原子领取：状态 + 尝试次数双条件，保证只有一个调用方能把任务推进到 running。
	claimed, err := s.client.ProvisioningTask.Update().
		Where(
			provisioningtask.ID(task.ID),
			provisioningtask.TenantID(tenantID),
			provisioningtask.AttemptCount(task.AttemptCount),
			provisioningtask.Or(
				provisioningtask.StatusIn(string(provisioning.TaskPending), string(provisioning.TaskFailed)),
				provisioningtask.And(
					provisioningtask.StatusEQ(string(provisioning.TaskRunning)),
					provisioningtask.Or(
						provisioningtask.LastAttemptAtIsNil(),
						provisioningtask.LastAttemptAtLT(cutoff),
					),
				),
			),
		).
		SetStatus(string(provisioning.TaskRunning)).
		SetAttemptCount(task.AttemptCount + 1).
		SetLastAttemptAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("领取交付任务失败: %w", err)
	}
	if claimed == 0 {
		// 竞争失败：重新读一次，把「已完成」和「别人在做」区分开，
		// 不要笼统报 500 让调用方以为可以再试。
		latest, getErr := s.client.ProvisioningTask.Get(ctx, task.ID)
		if getErr != nil {
			return nil, ErrProvisioningTaskInProgress
		}
		if provisioning.TaskStatus(latest.Status) == provisioning.TaskSucceeded {
			return latest, ErrProvisioningTaskSucceeded
		}
		return latest, ErrProvisioningTaskInProgress
	}

	// 领取成功即把服务请求拉回 provisioning：重试场景下它此前是 failed，
	// 若不同步，执行期间详情页会显示「已失败」而任务正在跑，自相矛盾。
	s.markServiceRequestStatus(ctx, tenantID, task.ServiceRequestID, domainServiceRequest.StatusProvisioning, "")

	// 执行（stub）
	payload := map[string]any{}
	if task.Payload != nil {
		payload = task.Payload
	}
	res, execErr := s.provider.Execute(ctx, payload)

	// 回写任务状态 + ServiceRequest
	if execErr != nil {
		if err := s.client.ProvisioningTask.UpdateOneID(task.ID).
			Where(provisioningtask.TenantID(tenantID)).
			SetStatus(string(provisioning.TaskFailed)).
			SetErrorMessage(execErr.Error()).
			Exec(ctx); err != nil {
			s.logger.Warnw("failed to update task status to failed", "taskID", task.ID, "error", err)
		}
		s.markServiceRequestStatus(ctx, tenantID, task.ServiceRequestID, domainServiceRequest.StatusFailed, execErr.Error())
		return nil, execErr
	}

	result := map[string]any{}
	if res != nil {
		result["resources"] = res.Resources
	}
	if err := s.client.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskSucceeded)).
		SetResult(result).
		SetErrorMessage("").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("回写任务结果失败: %w", err)
	}
	s.markServiceRequestStatus(ctx, tenantID, task.ServiceRequestID, domainServiceRequest.StatusDelivered, "")

	// 返回最新任务
	return s.client.ProvisioningTask.Get(ctx, task.ID)
}

// markServiceRequestStatus 回写服务请求状态。best-effort：任务状态是交付链路
// 的主事实，服务请求投影失败只留 Warn，不回滚已经发生的交付。
func (s *ProvisioningService) markServiceRequestStatus(ctx context.Context, tenantID, serviceRequestID int, status, lastError string) {
	update := s.client.ServiceRequest.Update().
		Where(servicerequest.ID(serviceRequestID), servicerequest.TenantID(tenantID)).
		SetStatus(status)
	if lastError != "" {
		update = update.SetLastError(lastError)
	}
	if err := update.Exec(ctx); err != nil {
		s.logger.Warnw("failed to update service request status", "serviceRequestID", serviceRequestID, "status", status, "error", err)
	}
}

// RetryTask 重试失败的交付任务。
//
// 与「直接再调一次 execute」的区别不只是语义：重试要
//   - 只对 failed 生效（pending 已在队列里、running 有人在做、succeeded 不能重做）；
//   - 受 MaxTaskAttempts 约束，避免对永久性故障无限重投；
//   - 用「第 N 次尝试」作为新的幂等键，否则命令会被入箱层判重直接跳过；
//   - 在同一事务里把服务请求从 failed 拉回 provisioning，避免任务在跑、请求显示失败。
func (s *ProvisioningService) RetryTask(ctx context.Context, taskID, tenantID, actorUserID int, reason string) (*ent.ProvisioningTask, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	task, err := tx.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("交付任务不存在: %w", err)
	}

	switch provisioning.TaskStatus(task.Status) {
	case provisioning.TaskSucceeded:
		return task, ErrProvisioningTaskSucceeded
	case provisioning.TaskRunning:
		return task, ErrProvisioningTaskInProgress
	}
	if task.AttemptCount >= provisioning.MaxTaskAttempts {
		return task, ErrProvisioningTaskAttemptsExhausted
	}
	if !provisioning.CanRetry(provisioning.TaskStatus(task.Status), task.AttemptCount) {
		return task, ErrProvisioningTaskNotRetryable
	}

	nextAttempt := task.AttemptCount + 1

	// 重新排队：置回 pending 让状态机允许领取，同时保留上一次的 error_message
	// 作为失败证据（下一次执行成功时才清空）。
	updated, err := tx.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskPending)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("重置交付任务状态失败: %w", err)
	}

	// 服务请求 failed → provisioning（仅当仍是 failed，避免覆盖人工已推进的状态）
	if err := tx.ServiceRequest.Update().
		Where(
			servicerequest.ID(task.ServiceRequestID),
			servicerequest.TenantID(tenantID),
			servicerequest.StatusEQ(domainServiceRequest.StatusFailed),
		).
		SetStatus(domainServiceRequest.StatusProvisioning).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
	}

	payload := map[string]interface{}{
		"serviceRequestId": task.ServiceRequestID,
		"taskId":           task.ID,
		"actorUserId":      actorUserID,
		"attempt":          nextAttempt,
	}
	if trimmed := strings.TrimSpace(reason); trimmed != "" {
		payload["retryReason"] = trimmed
	}
	if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID:       tenantID,
		CommandType:    commandbus.CommandExecuteProvisioningTask,
		AggregateType:  "provisioning_task",
		AggregateID:    task.ID,
		IdempotencyKey: provisioningExecuteKey(task.ID, nextAttempt),
		MaxAttempts:    provisioning.MaxTaskAttempts - task.AttemptCount,
		Payload:        payload,
	}); err != nil {
		return nil, fmt.Errorf("入箱交付任务重试命令失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	return updated, nil
}

// ResolveTaskManually 人工接管交付任务：把 failed/pending 的任务直接判定为
// delivered 或 failed。
//
// 为什么必须有这条路径：provider 永久性故障（凭据吊销、配额封禁、地域下线）时，
// 自动重试永远不可能成功；而资源已在平台外建好的情况，系统也不该继续显示失败。
// 两种情形都需要人给出结论，且结论必须可审计——所以 note 必填、接管人必记。
func (s *ProvisioningService) ResolveTaskManually(ctx context.Context, taskID, tenantID, actorUserID int, outcome, note string) (*ent.ProvisioningTask, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return nil, common.NewBusinessError(common.ParamErrorCode, "人工判定必须填写依据说明", "")
	}
	var finalStatus provisioning.TaskStatus
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "delivered":
		finalStatus = provisioning.TaskSucceeded
	case "failed":
		finalStatus = provisioning.TaskFailed
	default:
		return nil, common.NewBusinessError(common.ParamErrorCode, "判定结果只能是 delivered 或 failed", "")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	task, err := tx.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("交付任务不存在: %w", err)
	}
	if !provisioning.CanResolveManually(provisioning.TaskStatus(task.Status)) {
		return task, ErrProvisioningTaskNotResolvable
	}

	update := tx.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(finalStatus)).
		SetResolvedByUserID(actorUserID).
		SetResolutionNote(note)
	if finalStatus == provisioning.TaskFailed {
		update = update.SetErrorMessage(note)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新交付任务失败: %w", err)
	}

	if finalStatus == provisioning.TaskSucceeded {
		if err := tx.ServiceRequest.Update().
			Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
			SetStatus(domainServiceRequest.StatusDelivered).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
		}
	} else {
		if err := tx.ServiceRequest.Update().
			Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
			SetStatus(domainServiceRequest.StatusFailed).
			SetLastError(note).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	s.logger.Infow("provisioning task resolved manually",
		"taskID", task.ID, "tenantID", tenantID, "actorUserID", actorUserID, "outcome", finalStatus, "note", note)
	return updated, nil
}

func (s *ProvisioningService) ListTasksByServiceRequest(ctx context.Context, serviceRequestID, tenantID int) ([]*ent.ProvisioningTask, error) {
	return s.client.ProvisioningTask.Query().
		Where(provisioningtask.TenantID(tenantID), provisioningtask.ServiceRequestID(serviceRequestID)).
		Order(ent.Desc(provisioningtask.FieldCreatedAt)).
		All(ctx)
}
