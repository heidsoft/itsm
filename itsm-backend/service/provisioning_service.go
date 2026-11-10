package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/domain/provisioning"
	domainServiceRequest "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/provisioningtask"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/infrastructure/cloud"
	cloudAlicloud "itsm-backend/infrastructure/cloud/alicloud"
	"itsm-backend/internal/commandbus"
	cloudRuntime "itsm-backend/service/cloud"

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
	ErrProvisioningTaskSucceeded = errors.New("The delivery task has been completed and repeated execution is refused.")
	// ErrProvisioningTaskInProgress 任务正在执行中（或刚被其他 worker 领取）。
	ErrProvisioningTaskInProgress = errors.New("Delivery task is being executed")
	// ErrProvisioningTaskAttemptsExhausted 重试次数已达上限，必须转人工判定。
	ErrProvisioningTaskAttemptsExhausted = errors.New("The number of retries of the delivery task has reached the upper limit and manual takeover is required.")
	// ErrProvisioningTaskNotRetryable 当前状态不具备重试语义（如尚未失败的 pending）。
	ErrProvisioningTaskNotRetryable = errors.New("The current status of the delivery task does not allow retry")
	// ErrProvisioningTaskNotResolvable 当前状态不允许人工判定（running/succeeded）。
	ErrProvisioningTaskNotResolvable = errors.New("The current status of the delivery task does not allow manual judgment")
	// ErrProvisioningNotExecutable 交付参数或云账号配置不完整，自动重试不可能成功。
	// 这类失败落库后不再重投，由运维修正配置后走人工重试。
	ErrProvisioningNotExecutable = errors.New("Delivery configuration is incomplete, automatic retry will not succeed")
	// ErrProvisioningPermanentlyFailed 云侧业务拒绝（余额不足/配额封禁/参数不存在），
	// 重试期望收益为零：停止自动重投，保留人工重试入口供运维修复后重新触发。
	ErrProvisioningPermanentlyFailed = errors.New("Rejected by the cloud side and automatic retry will not succeed.")
)

// staleTaskLease 判定「running 但已无人执行」的租约时长。
//
// 没有这个租约，worker 在 provider 调用中途崩溃（进程被杀、节点回收）会让任务
// 永久停在 running：既不能执行（状态守卫），也不能人工接管（running 不可判定），
// 交付链路就此卡死。租约到期即视为可重新领取——provider 侧必须靠调用方的
// 幂等键去重，这一点在 payload 中已经传递。
const staleTaskLease = 10 * time.Minute

// ProvisioningService（应用层）：服务请求 -> 交付任务 -> 执行 -> 状态回写
type ProvisioningService struct {
	client           *ent.Client
	logger           *zap.SugaredLogger
	provider         cloud.Provider
	secretResolver   cloudRuntime.TenantSecretResolver
	deliveryListener ProvisioningDeliveryListener
	ciProjector      ProvisioningCIProjector
}

// supportedProvisioningResourceTypes 是当前 Provider 真正能交付的资源类型。
// 表外类型在「建任务」和「执行」两个入口都失败关闭：建一个注定无法执行的任务，
// 只会让运维在交付队列里看到一条永远失败的记录。
var supportedProvisioningResourceTypes = map[string]struct{}{
	string(provisioning.ResourceECS): {},
}

// normalizeProvisioningResourceType 空值按 ECS 处理（历史表单未传 resource_type），
// 其余值统一小写去空白后落库。
func normalizeProvisioningResourceType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return string(provisioning.ResourceECS)
	}
	return normalized
}

func NewProvisioningService(client *ent.Client, logger *zap.SugaredLogger) *ProvisioningService {
	return &ProvisioningService{
		client:   client,
		logger:   logger,
		provider: cloudAlicloud.NewStubProvider(),
	}
}

// SetProvider 替换交付实现。生产装配使用阿里云 ECSProvider；测试保留 Stub。
func (s *ProvisioningService) SetProvider(provider cloud.Provider) {
	if provider != nil {
		s.provider = provider
	}
}

// SetCloudRuntime 注入租户凭据解析器（cloud_discovery.tenant_secrets）。
// 真实 Provider 缺少它时必须 fail closed，不能退回 Stub 造假成功。
func (s *ProvisioningService) SetCloudRuntime(resolver cloudRuntime.TenantSecretResolver) {
	s.secretResolver = resolver
}

// SetDeliveryListener 注入交付生命周期监听（SLA 首次响应/解决、BPMN 作业推进）。
// 通知在事务提交后发出，监听方失败不会回滚交付。
func (s *ProvisioningService) SetDeliveryListener(listener ProvisioningDeliveryListener) {
	s.deliveryListener = listener
}

// SetCIProjector 注入 CMDB 投影（交付成功的资源落成配置项）。
// 未装配时任务结果会如实记录 unavailable，而不是假装已经建账。
func (s *ProvisioningService) SetCIProjector(projector ProvisioningCIProjector) {
	s.ciProjector = projector
}

// CreateTaskFromServiceRequest 仅创建交付任务、把 ServiceRequest 置为 provisioning，
// 并把"执行任务"作为 CommandExecuteProvisioningTask 入箱。
// worker 重载任务后调 ExecuteTask；这样请求结束 / 进程退出后仍可重试到 max_attempts，
// 避免同步执行在网络抖动时丢交付任务。幂等键按 task ID 锁定，replay 不会重复执行。
func (s *ProvisioningService) CreateTaskFromServiceRequest(ctx context.Context, serviceRequestID, tenantID, actorUserID int) (*ent.ProvisioningTask, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to open transaction: %w", err)
	}
	defer tx.Rollback()

	sr, err := tx.ServiceRequest.Query().
		Where(servicerequest.ID(serviceRequestID), servicerequest.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		// %w 保留 ent.NotFoundError：handler 的 classifyError 需要靠它把
		// 跨租户/不存在统一呈现为 404，而不是兜底成 500。
		return nil, fmt.Errorf("Service request does not exist: %w", err)
	}
	if string(sr.Status) != domainServiceRequest.StatusSecurityApproved {
		// 非法状态迁移按仓库语义映射 Conflict(4090)，不得兜底成 400/500。
		return nil, common.NewBusinessError(common.ConflictCode, "Current status does not allow delivery to be initiated (requires security_approved)", "")
	}

	// payload：优先使用 form_data；若为空则降级为空 map
	payload := map[string]any{}
	if sr.FormData != nil {
		payload = sr.FormData
	}

	// 最小推断资源类型（后续基于 catalog/category 或专用字段）。
	// 类型必须在建任务时就校验：否则会留下一批 provider 永远不认的交付任务。
	rawResourceType := provisioningStringValue(payload, "resource_type")
	resourceType := normalizeProvisioningResourceType(rawResourceType)
	if _, ok := supportedProvisioningResourceTypes[resourceType]; !ok {
		return nil, common.NewBadRequestError("Delivery resource type is not supported yet"+resourceType+"(currently only supports ecs)", nil)
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
		return nil, fmt.Errorf("Failed to update service request status: %w", err)
	}
	if affected == 0 {
		return nil, common.NewBusinessError(common.ConflictCode, "The service request has been advanced by other operations, please refresh and try again", "")
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
		return nil, fmt.Errorf("Failed to create delivery task: %w", err)
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
		return nil, fmt.Errorf("Inbox delivery task execution command failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction: %w", err)
	}
	s.notifyDeliveryStarted(ctx, tenantID, serviceRequestID, actorUserID)
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
		return nil, fmt.Errorf("Delivery task does not exist: %w", err)
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
		return nil, fmt.Errorf("Failed to collect delivery task: %w", err)
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
	s.notifyDeliveryStarted(ctx, tenantID, task.ServiceRequestID, actorUserID)

	// 执行前先解析云账号与凭据。真实 Provider 缺账号/凭据时直接落 failed，
	// 不把一次注定失败的云 API 调用留给重试队列。
	execReq, buildErr := s.buildExecuteRequest(ctx, tenantID, task)
	if buildErr != nil {
		s.failTask(ctx, tenantID, task, actorUserID, buildErr)
		return nil, buildErr
	}

	res, execErr := s.provider.Execute(ctx, execReq)
	if execErr != nil {
		// 云调用失败：请求已到达（或可能已到达）云侧，补记账号/地域证据，
		// 供人工按 ClientToken 去云控制台核对是否残留资源。
		evidence := map[string]any{"possibleCloudResource": true}
		if execReq.Region != "" {
			evidence["region"] = execReq.Region
		}
		if execReq.AccountID != "" {
			evidence["cloudAccountId"] = execReq.AccountID
		}
		s.failTask(ctx, tenantID, task, actorUserID, execErr, evidence)
		// 云侧业务拒绝（余额/配额/参数）重试无意义：翻译成「永久失败」哨兵，
		// 让命令总线停止自动重投，把处置权交回运维（充值/改参数后人工重试）。
		if errors.Is(execErr, cloud.ErrPermanent) {
			return nil, fmt.Errorf("%w: %v", ErrProvisioningPermanentlyFailed, execErr)
		}
		return nil, execErr
	}

	return s.completeTask(ctx, tenantID, task, actorUserID, res)
}

// buildExecuteRequest 把任务上下文解析成 Provider 可执行的请求。
// 账号选择、地域校验和凭据解析都在这里完成；Provider 不接触数据库和配置。
func (s *ProvisioningService) buildExecuteRequest(ctx context.Context, tenantID int, task *ent.ProvisioningTask) (cloud.ExecuteRequest, error) {
	resourceType := normalizeProvisioningResourceType(task.ResourceType)
	if _, ok := supportedProvisioningResourceTypes[resourceType]; !ok {
		return cloud.ExecuteRequest{}, fmt.Errorf("%w: The delivery resource type %s is not supported yet (only ecs is currently supported)", ErrProvisioningNotExecutable, resourceType)
	}
	if provider := cloudRuntime.NormalizeProvider(task.Provider); provider != "aliyun" {
		return cloud.ExecuteRequest{}, fmt.Errorf("%w: Delivery manufacturer %s is not currently supported (currently only aliyun is supported)", ErrProvisioningNotExecutable, task.Provider)
	}

	payload := map[string]any{}
	for key, value := range task.Payload {
		payload[key] = value
	}
	req := cloud.ExecuteRequest{
		TenantID:         tenantID,
		TaskID:           task.ID,
		ServiceRequestID: task.ServiceRequestID,
		ClientToken:      provisioningClientToken(task.ID),
		Payload:          payload,
	}
	if !s.provider.RequiresCloudAccount() {
		return req, nil
	}
	if s.secretResolver == nil {
		return req, fmt.Errorf("%w: Cloud credential parser is not assembled", ErrProvisioningNotExecutable)
	}

	sr, err := s.client.ServiceRequest.Query().
		Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return req, fmt.Errorf("Read service request failed: %w", err)
	}
	catalog, err := s.client.ServiceCatalog.Query().
		Where(servicecatalog.ID(sr.CatalogID), servicecatalog.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return req, fmt.Errorf("Failed to read service directory: %w", err)
	}

	region := provisioningStringValue(payload, "regionId")
	if region == "" && len(catalog.AvailableRegions) == 1 {
		region = catalog.AvailableRegions[0]
	}
	if region == "" {
		return req, fmt.Errorf("%w: The delivery parameter is missing regionId and the service catalog is not configured with a unique available region", ErrProvisioningNotExecutable)
	}
	if len(catalog.AvailableRegions) > 0 && !containsString(catalog.AvailableRegions, region) {
		return req, fmt.Errorf("%w: regionId %s is not in the region allowed by the service catalog", ErrProvisioningNotExecutable, region)
	}

	account, err := s.selectCloudAccount(ctx, tenantID, region, payload)
	if err != nil {
		return req, err
	}
	credentialJSON, err := s.secretResolver.Resolve(ctx, tenantID, account.CredentialRef)
	if err != nil {
		return req, fmt.Errorf("%w: Failed to parse cloud credentials: %v", ErrProvisioningNotExecutable, err)
	}
	resolved, err := cloudRuntime.ResolveAliyunCredential(ctx, credentialJSON)
	if err != nil {
		return req, fmt.Errorf("%w: %v", ErrProvisioningNotExecutable, err)
	}
	if resolved.AccessKeyID == "" || resolved.AccessKeySecret == "" {
		return req, fmt.Errorf("%w: Alibaba Cloud credentials are missing AccessKeyID/AccessKeySecret", ErrProvisioningNotExecutable)
	}

	req.AccountID = account.AccountID
	req.Region = region
	req.Credential = cloud.Credential{
		AccessKeyID:     resolved.AccessKeyID,
		AccessKeySecret: resolved.AccessKeySecret,
		SessionToken:    resolved.SessionToken,
	}
	// 公网标记以服务请求落库值为准，避免表单漏传后 Provider 误开公网。
	payload["needsPublicIp"] = sr.NeedsPublicIP
	return req, nil
}

func (s *ProvisioningService) selectCloudAccount(ctx context.Context, tenantID int, region string, payload map[string]any) (*ent.CloudAccount, error) {
	accounts, err := s.client.CloudAccount.Query().
		Where(cloudaccount.TenantID(tenantID), cloudaccount.IsActive(true)).
		Order(ent.Asc(cloudaccount.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query cloud account: %w", err)
	}

	requestedID := provisioningIntValue(payload, "cloudAccountId")
	candidates := make([]*ent.CloudAccount, 0, len(accounts))
	for _, account := range accounts {
		if cloudRuntime.NormalizeProvider(account.Provider) != "aliyun" {
			continue
		}
		if requestedID > 0 && account.ID != requestedID {
			continue
		}
		if len(account.RegionWhitelist) > 0 && !containsString(account.RegionWhitelist, region) {
			continue
		}
		candidates = append(candidates, account)
	}

	switch len(candidates) {
	case 0:
		if requestedID > 0 {
			return nil, fmt.Errorf("%w: The specified cloud account does not exist, is not enabled, is not an Alibaba Cloud account, or the region %s is not allowed", ErrProvisioningNotExecutable, region)
		}
		return nil, fmt.Errorf("%w: The tenant has not configured an available Alibaba Cloud account (provider=aliyun, isActive=true, and the region whitelist contains %s)", ErrProvisioningNotExecutable, region)
	case 1:
		return candidates[0], nil
	default:
		return nil, fmt.Errorf("%w: There are multiple available Alibaba Cloud accounts. Please specify cloudAccountId in the delivery parameters.", ErrProvisioningNotExecutable)
	}
}

func (s *ProvisioningService) completeTask(ctx context.Context, tenantID int, task *ent.ProvisioningTask, actorUserID int, res *cloud.ExecuteResult) (*ent.ProvisioningTask, error) {
	result := map[string]any{}
	if res != nil {
		result["resources"] = res.Resources
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to open transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tx.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskSucceeded)).
		SetResult(result).
		SetErrorMessage("").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("Writeback task result failed: %w", err)
	}
	if err := tx.ServiceRequest.Update().
		Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
		SetStatus(domainServiceRequest.StatusDelivered).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("Failed to update service request status: %w", err)
	}
	if err := recordProvisioningAudit(ctx, tx, tenantID, actorUserID, task, "provision.execute.succeeded", 200, result); err != nil {
		return nil, fmt.Errorf("Write delivery audit failed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction: %w", err)
	}
	// 交付主事实已落库。之后的 CMDB 投影和生命周期通知属于跨域联动，
	// 失败只记录在任务结果/日志里，不能把一次真实成功的云上交付改判为失败。
	s.projectDeliveredResource(ctx, tenantID, actorUserID, task, res)
	s.notifyDeliveryFinished(ctx, tenantID, task.ServiceRequestID, actorUserID, true, "")
	return s.client.ProvisioningTask.Get(ctx, task.ID)
}

func (s *ProvisioningService) failTask(ctx context.Context, tenantID int, task *ent.ProvisioningTask, actorUserID int, cause error, evidence ...map[string]any) {
	message := sanitizeProvisioningError(cause)
	// 对账证据：失败 ≠ 云侧什么都没发生。RunInstances 超时/响应丢失时实例
	// 可能已经创建，ClientToken 是云侧幂等键，按它能核对到同一笔创建请求；
	// 没有这组证据，人工裁决「failed 还是 delivered」就只能瞎猜。
	body := map[string]any{
		"taskId":           task.ID,
		"serviceRequestId": task.ServiceRequestID,
		"provider":         task.Provider,
		"resourceType":     task.ResourceType,
		"attempt":          task.AttemptCount + 1,
		"error":            message,
		"clientToken":      provisioningClientToken(task.ID),
	}
	for _, partial := range evidence {
		for key, value := range partial {
			if value == nil || value == "" {
				continue
			}
			body[key] = value
		}
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Warnw("failed to open tx for provisioning failure", "taskID", task.ID, "error", err)
		s.markServiceRequestStatus(ctx, tenantID, task.ServiceRequestID, domainServiceRequest.StatusFailed, message)
		return
	}
	defer tx.Rollback()

	if err := tx.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskFailed)).
		SetErrorMessage(message).
		Exec(ctx); err != nil {
		s.logger.Warnw("failed to update task status to failed", "taskID", task.ID, "error", err)
	}
	if err := tx.ServiceRequest.Update().
		Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
		SetStatus(domainServiceRequest.StatusFailed).
		SetLastError(message).
		Exec(ctx); err != nil {
		s.logger.Warnw("failed to update service request status", "serviceRequestID", task.ServiceRequestID, "error", err)
	}
	if err := recordProvisioningAudit(ctx, tx, tenantID, actorUserID, task, "provision.execute.failed", 500, body); err != nil {
		s.logger.Warnw("failed to write provisioning audit", "taskID", task.ID, "error", err)
	}
	if err := tx.Commit(); err != nil {
		s.logger.Warnw("failed to commit provisioning failure", "taskID", task.ID, "error", err)
		return
	}
	s.notifyDeliveryFinished(ctx, tenantID, task.ServiceRequestID, actorUserID, false, message)
}

func recordProvisioningAudit(ctx context.Context, tx *ent.Tx, tenantID, actorUserID int, task *ent.ProvisioningTask, action string, statusCode int, body map[string]any) error {
	if tx == nil || task == nil {
		return nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte("{}")
	}
	_, err = tx.AuditLog.Create().
		SetTenantID(tenantID).
		SetUserID(actorUserID).
		SetPath("worker/provisioning").
		SetMethod("COMMAND").
		SetResource("provisioning_task").
		SetAction(action).
		SetStatusCode(statusCode).
		SetRequestBody(string(raw)).
		Save(ctx)
	return err
}

// provisioningClientToken 只由任务 ID 推导，重试时保持不变。
// 阿里云按 ClientToken 去重；如果每次重试都换 token，响应丢失后的重试
// 会再创建一台机器，而不是取回第一次创建的实例。
func provisioningClientToken(taskID int) string {
	return fmt.Sprintf("itsm-provisioning-task-%d", taskID)
}

func provisioningStringValue(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func provisioningIntValue(payload map[string]any, key string) int {
	raw := provisioningStringValue(payload, key)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func sanitizeProvisioningError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.Join(strings.Fields(err.Error()), " ")
	runes := []rune(message)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return message
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
		return nil, fmt.Errorf("Failed to open transaction: %w", err)
	}
	defer tx.Rollback()

	task, err := tx.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("Delivery task does not exist: %w", err)
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
		return nil, fmt.Errorf("Failed to reset delivery task status: %w", err)
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
		return nil, fmt.Errorf("Failed to update service request status: %w", err)
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
		return nil, fmt.Errorf("Inbox delivery task retry command failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction: %w", err)
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
		return nil, common.NewBusinessError(common.ParamErrorCode, "Manual judgment must fill in the basis description", "")
	}
	var finalStatus provisioning.TaskStatus
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "delivered":
		finalStatus = provisioning.TaskSucceeded
	case "failed":
		finalStatus = provisioning.TaskFailed
	default:
		return nil, common.NewBusinessError(common.ParamErrorCode, "The judgment result can only be delivered or failed", "")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to open transaction: %w", err)
	}
	defer tx.Rollback()

	task, err := tx.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("Delivery task does not exist: %w", err)
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
		return nil, fmt.Errorf("Update delivery task failed: %w", err)
	}

	if finalStatus == provisioning.TaskSucceeded {
		if err := tx.ServiceRequest.Update().
			Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
			SetStatus(domainServiceRequest.StatusDelivered).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("Failed to update service request status: %w", err)
		}
	} else {
		if err := tx.ServiceRequest.Update().
			Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
			SetStatus(domainServiceRequest.StatusFailed).
			SetLastError(note).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("Failed to update service request status: %w", err)
		}
	}

	// 人工判定是高风险操作：谁在什么时候把任务判成 delivered/failed、依据是什么，
	// 必须落审计行而不是只留进程日志——日志会滚动，审计表不会。
	resolveBody := map[string]any{
		"taskId":           task.ID,
		"serviceRequestId": task.ServiceRequestID,
		"provider":         task.Provider,
		"resourceType":     task.ResourceType,
		"outcome":          strings.ToLower(strings.TrimSpace(outcome)),
		"note":             note,
		"manual":           true,
	}
	if err := recordProvisioningAudit(ctx, tx, tenantID, actorUserID, task, "provision.resolve.manual", 200, resolveBody); err != nil {
		return nil, fmt.Errorf("Write manual judgment audit failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to commit transaction: %w", err)
	}
	s.logger.Infow("provisioning task resolved manually",
		"taskID", task.ID, "tenantID", tenantID, "actorUserID", actorUserID, "outcome", finalStatus, "note", note)
	s.notifyDeliveryFinished(ctx, tenantID, task.ServiceRequestID, actorUserID, finalStatus == provisioning.TaskSucceeded, note)
	return updated, nil
}

func (s *ProvisioningService) ListTasksByServiceRequest(ctx context.Context, serviceRequestID, tenantID int) ([]*ent.ProvisioningTask, error) {
	return s.client.ProvisioningTask.Query().
		Where(provisioningtask.TenantID(tenantID), provisioningtask.ServiceRequestID(serviceRequestID)).
		Order(ent.Desc(provisioningtask.FieldCreatedAt)).
		All(ctx)
}

func (s *ProvisioningService) notifyDeliveryStarted(ctx context.Context, tenantID, serviceRequestID, actorUserID int) {
	if s.deliveryListener == nil || serviceRequestID <= 0 {
		return
	}
	s.deliveryListener.OnProvisioningStarted(ctx, tenantID, serviceRequestID, actorUserID)
}

func (s *ProvisioningService) notifyDeliveryFinished(ctx context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string) {
	if s.deliveryListener == nil || serviceRequestID <= 0 {
		return
	}
	s.deliveryListener.OnProvisioningFinished(ctx, tenantID, serviceRequestID, actorUserID, delivered, detail)
}

// projectDeliveredResource 把已交付的云资源投影成 CMDB 配置项，并把结果写回任务。
// 投影失败不影响交付结论：任务成功代表云上资源已创建，配置项缺失是可见的账务缺口，
// 由结果里的 cmdbProjection 状态和审计暴露给运维。
func (s *ProvisioningService) projectDeliveredResource(ctx context.Context, tenantID, actorUserID int, task *ent.ProvisioningTask, res *cloud.ExecuteResult) {
	if task == nil {
		return
	}
	outcome := ProvisioningCIProjectionResult{
		Status: ProvisioningCIProjectionUnavailable,
		Reason: "projector_not_wired",
	}
	if s.ciProjector != nil {
		instanceID := ""
		if res != nil {
			instanceID = strings.TrimSpace(res.Resources["instance_id"])
		}
		switch {
		case instanceID == "":
			outcome = ProvisioningCIProjectionResult{
				Status: ProvisioningCIProjectionSkipped,
				Reason: "provider_result_missing_instance_id",
			}
		default:
			resources := make(map[string]string, len(res.Resources))
			for key, value := range res.Resources {
				resources[key] = value
			}
			outcome = s.ciProjector.ProjectProvisionedResource(ctx, ProvisioningCIProjectionRequest{
				TenantID:           tenantID,
				ServiceRequestID:   task.ServiceRequestID,
				ProvisioningTaskID: task.ID,
				ResourceType:       task.ResourceType,
				Provider:           task.Provider,
				Resources:          resources,
				Payload:            task.Payload,
			})
			if strings.TrimSpace(outcome.Status) == "" {
				outcome = ProvisioningCIProjectionResult{
					Status: ProvisioningCIProjectionFailed,
					Reason: "empty_projection_result",
				}
			}
			if outcome.CIID > 0 &&
				(outcome.Status == ProvisioningCIProjectionCreated || outcome.Status == ProvisioningCIProjectionUpdated) {
				s.linkServiceRequestCI(ctx, tenantID, task.ServiceRequestID, outcome.CIID)
			}
		}
	}

	if err := s.persistCIProjectionOutcome(ctx, tenantID, actorUserID, task, outcome); err != nil {
		s.logger.Warnw("failed to persist cmdb projection outcome",
			"taskID", task.ID, "serviceRequestID", task.ServiceRequestID, "status", outcome.Status, "error", err)
	}
}

// persistCIProjectionOutcome 把投影结果并入 task.result 并留审计。
// result 是运维排障的读面，不能只写在日志里。
func (s *ProvisioningService) persistCIProjectionOutcome(ctx context.Context, tenantID, actorUserID int, task *ent.ProvisioningTask, outcome ProvisioningCIProjectionResult) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	latest, err := tx.ProvisioningTask.Query().
		Where(provisioningtask.ID(task.ID), provisioningtask.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return err
	}
	merged := make(map[string]any, len(latest.Result)+1)
	for key, value := range latest.Result {
		merged[key] = value
	}
	projection := map[string]any{
		"status": outcome.Status,
		"ciId":   outcome.CIID,
	}
	if outcome.Reason != "" {
		projection["reason"] = outcome.Reason
	}
	merged["cmdbProjection"] = projection
	if _, err := tx.ProvisioningTask.UpdateOneID(latest.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetResult(merged).
		Save(ctx); err != nil {
		return err
	}

	body := map[string]any{
		"taskId":           task.ID,
		"serviceRequestId": task.ServiceRequestID,
		"ciId":             outcome.CIID,
		"status":           outcome.Status,
		"reason":           outcome.Reason,
	}
	if action := ciProjectionAuditAction(outcome.Status); action != "" {
		if err := recordProvisioningAudit(ctx, tx, tenantID, actorUserID, latest, action, 200, body); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ciProjectionAuditAction(status string) string {
	switch status {
	case ProvisioningCIProjectionCreated, ProvisioningCIProjectionUpdated:
		return "provision.ci.projected"
	case ProvisioningCIProjectionDeferred:
		return "provision.ci.deferred"
	case ProvisioningCIProjectionFailed:
		return "provision.ci.failed"
	default:
		return ""
	}
}

// linkServiceRequestCI 把首次交付出的配置项挂到服务请求上。
// 只填空值：请求若已关联用户指定的 CI，不能被自动交付结果覆盖。
func (s *ProvisioningService) linkServiceRequestCI(ctx context.Context, tenantID, serviceRequestID, ciID int) {
	affected, err := s.client.ServiceRequest.Update().
		Where(
			servicerequest.ID(serviceRequestID),
			servicerequest.TenantID(tenantID),
			servicerequest.CiIDIsNil(),
		).
		SetCiID(ciID).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("failed to link provisioning CI to service request",
			"serviceRequestID", serviceRequestID, "ciID", ciID, "error", err)
		return
	}
	if affected > 0 {
		s.logger.Infow("service request linked to provisioned CI",
			"serviceRequestID", serviceRequestID, "ciID", ciID)
	}
}
