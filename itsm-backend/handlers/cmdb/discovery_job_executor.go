package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// DiscoveryAdapterProvider 是执行器对适配器注册表的窄依赖（生产实现 *cloud.Registry）。
type DiscoveryAdapterProvider interface {
	Get(provider, serviceCode string) (cloud.CloudDiscoveryAdapter, bool)
}

var _ DiscoveryAdapterProvider = (*cloud.Registry)(nil)

// DiscoveryJobExecutor 是 job 级云发现执行器，消费出箱命令 `cmdb.cloud_discovery.run`。
//
// 链路：payload 解析 → 认领 job（租约 + fencing token，双写命令租约由 commandbus
// 负责，job 租约权威在本执行器）→ 解析 source/account/凭据 → 按
// serviceCodes × regions 逐 scope 拉取 → 与 CloudResource 按 identity_hash 做
// diff（create/update/unchanged）→ DiscoveryResult 落库 → 资源写回并维护
// first/last_seen、missing_count（超阈值置 stale）→ 进度/心跳/取消检查 →
// 终态 + summary + source.last_success_at。
//
// 与旧 Runner.RunAll 的区别：RunAll 直接 upsert CI、绕过 discovery_jobs/results/
// CloudResource 三张表，且忽略命令 payload；本执行器是唯一生产路径。
type DiscoveryJobExecutor struct {
	repo        Repository
	adapters    DiscoveryAdapterProvider
	resolver    cloud.TenantSecretResolver
	ciProjector *DiscoveryCIProjector
	logger      *zap.SugaredLogger
	workerID    string
	leaseTTL    time.Duration
}

// ciProjector 允许为 nil（CI 域未装配），此时每条资源记一次 ciUnavailable 而不是静默不回写。
func NewDiscoveryJobExecutor(repo Repository, adapters DiscoveryAdapterProvider, resolver cloud.TenantSecretResolver, ciProjector *DiscoveryCIProjector, logger *zap.SugaredLogger) *DiscoveryJobExecutor {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	ttl := 5 * time.Minute
	return &DiscoveryJobExecutor{
		repo:        repo,
		adapters:    adapters,
		resolver:    resolver,
		ciProjector: ciProjector,
		logger:      logger,
		workerID:    fmt.Sprintf("discovery-%s-%d", hostnameShort(), os.Getpid()),
		leaseTTL:    ttl,
	}
}

// Handle 实现 commandbus.Handler。
func (e *DiscoveryJobExecutor) Handle(ctx context.Context, command *ent.OperationalCommand) error {
	if e == nil || e.repo == nil {
		return fmt.Errorf("discovery job executor is not configured")
	}
	if command == nil || command.TenantID <= 0 {
		return fmt.Errorf("tenant-scoped cloud discovery command is required")
	}
	var payloadJobID int
	if command.Payload != nil {
		switch v := command.Payload["discoveryJobId"].(type) {
		case float64:
			payloadJobID = int(v)
		case int:
			payloadJobID = v
		case json.Number:
			n, _ := v.Int64()
			payloadJobID = int(n)
		}
	}
	if payloadJobID <= 0 {
		return fmt.Errorf("discovery command payload missing discoveryJobId (command=%d)", command.ID)
	}
	return e.Execute(ctx, command.TenantID, payloadJobID)
}

// Execute 执行单个 discovery job。
func (e *DiscoveryJobExecutor) Execute(ctx context.Context, tenantID, jobID int) error {
	job, err := e.repo.GetDiscoveryJob(ctx, tenantID, jobID)
	if err != nil {
		return fmt.Errorf("load discovery job %d: %w", jobID, err)
	}
	// 终态任务的重复投递（outbox 至少一次语义）直接幂等吞掉。
	if isTerminalDiscoveryJobStatus(job.Status) {
		e.logger.Infow("discovery job already terminal, skip", "job", jobID, "status", job.Status)
		return nil
	}

	now := time.Now()
	if _, err := ClaimDiscoveryJob(job, e.workerID, now, e.leaseTTL); err != nil {
		if err == ErrJobAttemptsExhausted {
			return e.failJob(ctx, job, "attempts_exhausted", "发现任务重试次数已耗尽")
		}
		// 租约被其他 worker 持有：按可重试错误返回，commandbus 退避后重投。
		return fmt.Errorf("claim discovery job %d: %w", jobID, err)
	}
	if err := e.repo.UpdateDiscoveryJob(ctx, job); err != nil {
		return fmt.Errorf("persist claim for job %d: %w", jobID, err)
	}

	source, err := e.repo.GetDiscoverySource(ctx, tenantID, job.SourceID)
	if err != nil {
		return e.failJob(ctx, job, "source_not_found", fmt.Sprintf("发现源不存在: %v", err))
	}
	if !source.IsActive {
		return e.failJob(ctx, job, "source_inactive", "发现源已停用")
	}
	account, err := e.repo.GetCloudAccount(ctx, tenantID, source.CloudAccountID)
	if err != nil {
		return e.failJob(ctx, job, "account_not_found", fmt.Sprintf("云账号不存在: %v", err))
	}
	if !account.IsActive {
		return e.failJob(ctx, job, "account_inactive", "云账号已停用")
	}
	if e.resolver == nil {
		return e.failJob(ctx, job, "resolver_missing", "云凭据解析器未装配")
	}
	credential, err := e.resolver.Resolve(ctx, tenantID, account.CredentialRef)
	if err != nil {
		return e.failJob(ctx, job, "credential_resolve_failed", fmt.Sprintf("解析云凭据失败: %v", err))
	}
	// Never persist resolved material: adapters receive a request-local copy.
	resolvedAccount := entCloudAccountFromDomain(account)
	resolvedAccount.CredentialRef = credential
	accountEnt := resolvedAccount

	summary := map[string]interface{}{
		"discovered": 0, "created": 0, "updated": 0, "unchanged": 0, "missing": 0,
	}
	var (
		created, updated, unchanged, discovered, missing int
		ciCounts                                         ciProjectionCounts
		failedScopes                                     []string
		completedScopes                                  []string
	)

	scopes, err := e.resolveScopes(ctx, source, accountEnt)
	if err != nil {
		return e.failJob(ctx, job, "scope_resolution_failed", err.Error())
	}
	if len(scopes) == 0 {
		return e.failJob(ctx, job, "no_scopes", "scope 解析为空（serviceCodes 与可用适配器无交集）")
	}

	scopeFailed := false
	for i, sc := range scopes {
		// 取消检查：每 scope 前重读一次（API 侧只写 cancel_requested_at）。
		fresh, err := e.repo.GetDiscoveryJob(ctx, tenantID, jobID)
		if err == nil && fresh.CancelRequestedAt != nil {
			if err := TransitionDiscoveryJob(job, DiscoveryJobCancelled, time.Now()); err == nil {
				if uerr := e.repo.UpdateDiscoveryJob(ctx, job); uerr != nil {
					return fmt.Errorf("persist cancel for job %d: %w", jobID, uerr)
				}
				e.logger.Infow("discovery job cancelled by request", "job", jobID)
				return nil
			}
		}

		nCreated, nUpdated, nUnchanged, nDiscovered, nMissing, scopeErr := e.runScope(ctx, job, source, accountEnt, sc, &ciCounts)
		discovered += nDiscovered
		created += nCreated
		updated += nUpdated
		unchanged += nUnchanged
		missing += nMissing
		if scopeErr != nil {
			scopeFailed = true
			failedScopes = append(failedScopes, fmt.Sprintf("%s:%s", sc.serviceCode, sc.region))
			e.logger.Warnw("discovery scope failed", "job", jobID, "scope", sc.serviceCode+"/"+sc.region, "error", scopeErr)
		} else {
			completedScopes = append(completedScopes, sc.serviceCode+"/"+sc.region)
		}

		// 进度 + 心跳 + 租约续期持久化。
		job.CompletedScopes = completedScopes
		job.FailedScopes = failedScopes
		job.Progress = (i + 1) * 100 / len(scopes)
		_ = HeartbeatDiscoveryJob(job, e.workerID, job.FencingToken, time.Now(), e.leaseTTL)
		if err := e.repo.UpdateDiscoveryJob(ctx, job); err != nil {
			return fmt.Errorf("persist progress for job %d: %w", jobID, err)
		}
	}

	summary["discovered"] = discovered
	summary["created"] = created
	summary["updated"] = updated
	summary["unchanged"] = unchanged
	summary["missing"] = missing
	summary["failedScopes"] = failedScopes
	ciCounts.mergeInto(summary)

	now = time.Now()
	var finalStatus string
	if scopeFailed && len(completedScopes) == 0 {
		finalStatus = DiscoveryJobFailed
	} else if scopeFailed {
		finalStatus = DiscoveryJobPartialFailed
	} else {
		finalStatus = DiscoveryJobSucceeded
	}
	if finalStatus == DiscoveryJobSucceeded {
		// 显式走 discovered → reconciling → succeeded，保持状态机语义完整。
		if err := TransitionDiscoveryJob(job, DiscoveryJobDiscovered, now); err == nil {
			if err := TransitionDiscoveryJob(job, DiscoveryJobReconciling, now); err == nil {
				_ = e.repo.UpdateDiscoveryJob(ctx, job)
			}
		}
	}
	if err := TransitionDiscoveryJob(job, finalStatus, now); err != nil {
		// 已是终态或非法迁移：记录但不让命令重试（避免重复执行）。
		e.logger.Warnw("discovery job final transition skipped", "job", jobID, "from", job.Status, "to", finalStatus, "error", err)
	}
	job.Summary = summary
	if err := e.repo.UpdateDiscoveryJob(ctx, job); err != nil {
		return fmt.Errorf("persist final state for job %d: %w", jobID, err)
	}
	if finalStatus != DiscoveryJobFailed {
		if err := e.repo.TouchDiscoverySourceSuccess(ctx, tenantID, source.ID, now); err != nil {
			e.logger.Warnw("touch discovery source success failed", "source", source.ID, "error", err)
		}
	}
	e.logger.Infow("discovery job finished", "job", jobID, "status", finalStatus, "summary", summary)
	return nil
}

// discoveryScope 一个 (serviceCode, region) 执行单元。
type discoveryScope struct {
	serviceCode string
	region      string
}

// resolveScopes 计算 serviceCodes × regions 执行矩阵：
// region 取 source 声明 ∩ 账号白名单（白名单非空时），source 未声明则由
// adapter ListRegions 发现。
func (e *DiscoveryJobExecutor) resolveScopes(ctx context.Context, source *DiscoverySource, account *ent.CloudAccount) ([]discoveryScope, error) {
	codes := source.ServiceCodes
	if len(codes) == 0 {
		return nil, nil
	}
	whitelist := map[string]struct{}{}
	for _, r := range account.RegionWhitelist {
		whitelist[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	var scopes []discoveryScope
	for _, code := range codes {
		code = strings.ToLower(strings.TrimSpace(code))
		if code == "" {
			continue
		}
		regions := source.Regions
		if len(regions) == 0 {
			adapter, ok := e.adapters.Get(cloud.NormalizeProvider(source.Provider), code)
			if !ok {
				scopes = append(scopes, discoveryScope{serviceCode: code, region: "(adapter-missing)"})
				continue
			}
			rs, err := adapter.ListRegions(ctx, account)
			if err != nil {
				return nil, fmt.Errorf("ListRegions(%s): %w", code, err)
			}
			regions = rs
		}
		for _, region := range regions {
			region = strings.ToLower(strings.TrimSpace(region))
			if region == "" {
				continue
			}
			if len(whitelist) > 0 {
				if _, ok := whitelist[region]; !ok {
					continue
				}
			}
			scopes = append(scopes, discoveryScope{serviceCode: code, region: region})
		}
	}
	return scopes, nil
}

// runScope 执行一个 scope：拉取 → diff → 落库 → 缺失扫描。
func (e *DiscoveryJobExecutor) runScope(ctx context.Context, job *DiscoveryJob, source *DiscoverySource, account *ent.CloudAccount, sc discoveryScope, ciCounts *ciProjectionCounts) (created, updated, unchanged, discovered, missing int, err error) {
	provider := cloud.NormalizeProvider(source.Provider)
	adapter, ok := e.adapters.Get(provider, sc.serviceCode)
	if !ok {
		return 0, 0, 0, 0, 0, fmt.Errorf("no adapter for %s/%s", provider, sc.serviceCode)
	}
	svcDef, err := e.findCloudService(ctx, account.TenantID, provider, sc.serviceCode)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("load cloud service %s: %w", sc.serviceCode, err)
	}

	clients, err := adapter.InitClients(ctx, account, []string{sc.region})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("InitClients: %w", err)
	}
	client := clients[sc.region]
	if client == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("adapter returned no client for region %s", sc.region)
	}

	seenIdentities := map[string]struct{}{}
	nextToken := ""
	for {
		page, err := adapter.DiscoverRegion(ctx, account, sc.region, client, nextToken)
		if err != nil {
			return created, updated, unchanged, discovered, missing, fmt.Errorf("DiscoverRegion: %w", err)
		}
		for i := range page.Resources {
			res := &page.Resources[i]
			discovered++
			action, projection, derr := e.applyResource(ctx, job, source, account, svcDef, sc, res)
			if derr != nil {
				e.logger.Warnw("apply discovered resource failed", "job", job.ID, "resource", res.ResourceID, "error", derr)
				continue
			}
			switch action {
			case "create":
				created++
			case "update":
				updated++
			default:
				unchanged++
			}
			ciCounts.add(projection)
			idHash := resourceIdentityHash(account.TenantID, provider, account.AccountID, sc.region, sc.serviceCode, svcDef.ResourceTypeCode, res.ResourceID)
			seenIdentities[idHash] = struct{}{}
		}
		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}

	// 缺失扫描：本轮没见到的存量资源推进 missing_count，超阈值置 stale。
	existing, err := e.repo.ListCloudResourcesByAccountService(ctx, account.TenantID, account.ID, sc.serviceCode)
	if err != nil {
		return created, updated, unchanged, discovered, missing, fmt.Errorf("list resources for missing scan: %w", err)
	}
	missingNow := 0
	for _, ex := range existing {
		// 缺失扫描的 scope 是“账号 + 服务 + Region”。此前查询只按账号/服务，
		// 多 Region 来源会把其它 Region 的资源全部当作本轮缺失，连续几次后
		// 误置 stale。空 Region 的历史数据保留兼容；有 Region 的必须精确匹配。
		if region := strings.TrimSpace(ex.Region); region != "" && !strings.EqualFold(region, sc.region) {
			continue
		}
		if _, seen := seenIdentities[ex.IdentityHash]; seen {
			continue
		}
		ex.MissingCount++
		turnedStale := false
		if ex.LifecycleState != "stale" && source.StaleThreshold > 0 && ex.MissingCount >= source.StaleThreshold {
			ex.LifecycleState = "stale"
			turnedStale = true
		}
		if err := e.repo.UpdateDiscoveredCloudResource(ctx, ex); err != nil {
			e.logger.Warnw("missing scan update failed", "resource", ex.ID, "error", err)
			continue
		}
		missingNow++
		if turnedStale {
			ciCounts.addStale(e.ciProjector.MarkStale(ctx, ex))
		}
	}
	return created, updated, unchanged, discovered, missingNow, nil
}

// applyResource 对单个发现资源做 diff + 落库 + CI 回写，返回 action（create/update/unchanged）
// 与本次回写结果（未触发回写时为零值 outcome）。
func (e *DiscoveryJobExecutor) applyResource(ctx context.Context, job *DiscoveryJob, source *DiscoverySource, account *ent.CloudAccount, svcDef *CloudService, sc discoveryScope, res *cloud.DiscoveredResource) (string, CIProjectionOutcome, error) {
	provider := cloud.NormalizeProvider(source.Provider)
	identityHash := resourceIdentityHash(account.TenantID, provider, account.AccountID, sc.region, sc.serviceCode, svcDef.ResourceTypeCode, res.ResourceID)
	fingerprint := resourceFingerprint(res)
	now := time.Now()

	existing, err := e.repo.GetCloudResourceByIdentity(ctx, account.TenantID, identityHash)
	if err != nil && !ent.IsNotFound(err) {
		return "", CIProjectionOutcome{}, fmt.Errorf("load resource by identity: %w", err)
	}

	if existing == nil {
		cr := &CloudResource{
			CloudAccountID:     account.ID,
			ServiceID:          svcDef.ID,
			ResourceID:         res.ResourceID,
			IdentityVersion:    1,
			Provider:           provider,
			Partition:          "public",
			CanonicalAccountID: strings.TrimSpace(account.AccountID),
			ResourceScope:      resourceScope(res.Zone, res.Region),
			ServiceCode:        sc.serviceCode,
			ResourceType:       strings.ToLower(strings.TrimSpace(svcDef.ResourceTypeCode)),
			IdentityHash:       identityHash,
			SourceID:           source.ID,
			SourceFingerprint:  fingerprint,
			MissingCount:       0,
			ResourceName:       res.ResourceName,
			Region:             strings.ToLower(sc.region),
			Zone:               strings.ToLower(strings.TrimSpace(res.Zone)),
			Status:             cloud.NormalizeStatus(res.Status),
			Tags:               res.Tags,
			Metadata:           res.Extra,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
			LifecycleState:     "active",
			TenantID:           account.TenantID,
		}
		saved, err := e.repo.CreateCloudResource(ctx, cr)
		if err != nil {
			return "", CIProjectionOutcome{}, fmt.Errorf("create cloud resource: %w", err)
		}
		projection := e.ciProjector.Project(ctx, saved)
		if _, err := e.repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
			JobID:            job.ID,
			CIID:             projection.CIID,
			Action:           "create",
			ResourceType:     cr.ResourceType,
			ResourceID:       cr.ResourceID,
			ResourceIdentity: identityHash,
			IdentityVersion:  1,
			ResourceSnapshot: resourceSnapshot(res),
			AfterHash:        fingerprint,
			Diff:             map[string]interface{}{"op": "create", "resource": resourceSnapshot(res)},
			Status:           "applied",
			TenantID:         account.TenantID,
		}); err != nil {
			return "", projection, fmt.Errorf("create discovery result: %w", err)
		}
		return "create", projection, nil
	}

	// 已存在：指纹一致 → unchanged（仍要清零缺失计数 + 刷新 last_seen）。
	if existing.SourceFingerprint == fingerprint {
		wasStale := existing.LifecycleState == "stale"
		existing.MissingCount = 0
		existing.SourceID = source.ID
		if wasStale {
			// 资源回来了就要从 stale 回到 active，否则下一轮内容没变化时
			// 它会永远停在「缺失」，配置项也跟着被误标。
			existing.LifecycleState = "active"
		}
		if err := e.repo.UpdateDiscoveredCloudResource(ctx, existing); err != nil {
			return "", CIProjectionOutcome{}, fmt.Errorf("touch unchanged resource: %w", err)
		}
		if !wasStale {
			return "unchanged", CIProjectionOutcome{}, nil
		}
		return "unchanged", e.ciProjector.Project(ctx, existing), nil
	}

	// 有变化：按 reconcile_policy 决定是否自动应用。
	diff := map[string]interface{}{}
	if existing.ResourceName != res.ResourceName {
		diff["resourceName"] = map[string]interface{}{"before": existing.ResourceName, "after": res.ResourceName}
	}
	if existing.Status != cloud.NormalizeStatus(res.Status) {
		diff["status"] = map[string]interface{}{"before": existing.Status, "after": cloud.NormalizeStatus(res.Status)}
	}
	if existing.Region != strings.ToLower(sc.region) {
		diff["region"] = map[string]interface{}{"before": existing.Region, "after": strings.ToLower(sc.region)}
	}

	manual := source.ReconcilePolicy == string(cloud.ReconcileManual)
	resultStatus := "applied"
	var projection CIProjectionOutcome
	if manual {
		resultStatus = "pending"
	} else {
		existing.ResourceName = res.ResourceName
		existing.Status = cloud.NormalizeStatus(res.Status)
		existing.Region = strings.ToLower(sc.region)
		existing.Zone = strings.ToLower(strings.TrimSpace(res.Zone))
		existing.Tags = res.Tags
		existing.Metadata = res.Extra
		existing.SourceID = source.ID
		existing.SourceFingerprint = fingerprint
		existing.MissingCount = 0
		existing.LifecycleState = "active"
		if err := e.repo.UpdateDiscoveredCloudResource(ctx, existing); err != nil {
			return "", CIProjectionOutcome{}, fmt.Errorf("apply resource update: %w", err)
		}
		projection = e.ciProjector.Project(ctx, existing)
	}
	if _, err := e.repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
		JobID:            job.ID,
		CIID:             projection.CIID,
		Action:           "update",
		ResourceType:     existing.ResourceType,
		ResourceID:       existing.ResourceID,
		ResourceIdentity: identityHash,
		IdentityVersion:  existing.IdentityVersion,
		ResourceSnapshot: resourceSnapshot(res),
		BeforeHash:       existing.SourceFingerprint,
		AfterHash:        fingerprint,
		Diff:             diff,
		Status:           resultStatus,
		TenantID:         account.TenantID,
	}); err != nil {
		return "", projection, fmt.Errorf("create update result: %w", err)
	}
	return "update", projection, nil
}

func (e *DiscoveryJobExecutor) findCloudService(ctx context.Context, tenantID int, provider, serviceCode string) (*CloudService, error) {
	list, err := e.repo.ListCloudServices(ctx, tenantID, provider)
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if strings.EqualFold(s.ServiceCode, serviceCode) {
			return s, nil
		}
	}
	return nil, &ent.NotFoundError{}
}

// failJob 把 job 推到 failed 终态并写错误信息；返回 nil 让命令不再重试
// （业务性失败重试无意义，死信/人工重试面已有专门通道）。
func (e *DiscoveryJobExecutor) failJob(ctx context.Context, job *DiscoveryJob, code, message string) error {
	now := time.Now()
	if job.Status == DiscoveryJobQueued {
		_, _ = ClaimDiscoveryJob(job, e.workerID, now, e.leaseTTL)
	}
	if err := TransitionDiscoveryJob(job, DiscoveryJobFailed, now); err != nil {
		if !isTerminalDiscoveryJobStatus(job.Status) {
			return fmt.Errorf("transition job %d to failed: %w", job.ID, err)
		}
	}
	job.ErrorCode = code
	job.ErrorMessage = message
	job.Summary = map[string]interface{}{"errorCode": code}
	if err := e.repo.UpdateDiscoveryJob(ctx, job); err != nil {
		return fmt.Errorf("persist failure for job %d: %w", job.ID, err)
	}
	e.logger.Warnw("discovery job failed", "job", job.ID, "code", code, "message", message)
	return nil
}

// resourceIdentityHash 与 prepareCloudResourceIdentity 的身份算法逐字节一致
// （v1|tenant|provider|partition|account|scope|region|service|type|resourceID → sha256 hex）。
// 两处共用一条算法：改动必须同步并递增 identity_version。
func resourceIdentityHash(tenantID int, provider, canonicalAccount, region, serviceCode, resourceType, resourceID string) string {
	region = strings.ToLower(strings.TrimSpace(region))
	identity := strings.Join([]string{
		"v1", fmt.Sprint(tenantID), provider, "public",
		strings.TrimSpace(canonicalAccount), resourceScope("", region), region,
		serviceCode, strings.ToLower(strings.TrimSpace(resourceType)), strings.TrimSpace(resourceID),
	}, "|")
	hash := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(hash[:])
}

func resourceScope(zone, region string) string {
	if strings.TrimSpace(zone) != "" {
		return "zonal"
	}
	if strings.TrimSpace(region) != "" {
		return "regional"
	}
	return "global"
}

// resourceFingerprint 发现快照的内容指纹：身份之外的属性变化都要被 diff 捕捉。
func resourceFingerprint(res *cloud.DiscoveredResource) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"name":   res.ResourceName,
		"status": res.Status,
		"zone":   res.Zone,
		"tags":   res.Tags,
		"extra":  res.Extra,
	})
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func resourceSnapshot(res *cloud.DiscoveredResource) map[string]interface{} {
	snap := map[string]interface{}{
		"resourceId":   res.ResourceID,
		"resourceName": res.ResourceName,
		"region":       res.Region,
		"status":       res.Status,
	}
	if res.Zone != "" {
		snap["zone"] = res.Zone
	}
	if len(res.Tags) > 0 {
		snap["tags"] = res.Tags
	}
	if len(res.Extra) > 0 {
		snap["extra"] = res.Extra
	}
	return snap
}

// entCloudAccountFromDomain 把域 CloudAccount 映射为 ent 实体（适配器接口吃 ent 类型）。
func entCloudAccountFromDomain(ca *CloudAccount) *ent.CloudAccount {
	return &ent.CloudAccount{
		ID:              ca.ID,
		Provider:        ca.Provider,
		AccountID:       ca.AccountID,
		AccountName:     ca.AccountName,
		CredentialRef:   ca.CredentialRef,
		RegionWhitelist: ca.RegionWhitelist,
		IsActive:        ca.IsActive,
		TenantID:        ca.TenantID,
	}
}

func hostnameShort() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
