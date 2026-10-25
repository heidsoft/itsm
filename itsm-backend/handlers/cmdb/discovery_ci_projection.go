package cmdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"
)

// 发现结果回写配置项。CI 的写入规则（发号、历史、属性校验）在
// service.ConfigurationItemService，这里只决定落成哪一类、带哪些字段。
// 两条产品口径：动态属性只从云标签推断，凑不齐 CI 类型的必填项就记 deferred 交人补录，
// 不臆造值；资源连续缺失只把已链接的配置项标成 cloud_sync_status=stale，
// 不碰 lifecycle_status——退役是人的决定。

const (
	CIProjectionCreated     = "created"
	CIProjectionUpdated     = "updated"
	CIProjectionDeferred    = "deferred"
	CIProjectionSkipped     = "skipped"
	CIProjectionFailed      = "failed"
	CIProjectionUnavailable = "unavailable"
)

const (
	CIDiscoverySourceValue = "discovery"

	CISyncStatusSuccess = "success"
	CISyncStatusStale   = "stale"

	CIProjectionReasonMarkedStale  = "marked_stale"
	CIProjectionReasonAlreadyStale = "already_stale"
	CIProjectionReasonNoLinkedCI   = "no_linked_ci"
)

// discoveryCITypeByResource 只登记产品基线里确实存在的 CI 类型。没有命中的资源
// 一律跳过并给原因：宁可少一条配置项，也不要凭空造类型或把云主机塞进 server。
var discoveryCITypeByResource = map[string]string{
	"ecs|instance": "cloud_vm",
	"cvm|instance": "cloud_vm",
	"ec2|instance": "cloud_vm",
	"pod|pod":      "kubernetes",
}

// CITypeForCloudResource 返回云资源应落成的 CI 类型名（serviceCode|resourceType 复合键）。
func CITypeForCloudResource(serviceCode, resourceType string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(serviceCode)) + "|" + strings.ToLower(strings.TrimSpace(resourceType))
	ciType, ok := discoveryCITypeByResource[key]
	return ciType, ok
}

// DiscoveryCITypeMappingKeys 暴露映射表键，供守卫测试比对已注册适配器。
func DiscoveryCITypeMappingKeys() []string {
	keys := make([]string, 0, len(discoveryCITypeByResource))
	for key := range discoveryCITypeByResource {
		keys = append(keys, key)
	}
	return keys
}

// DiscoveryCIWriter 是回写面用到的 CI 域能力（生产实现 *service.ConfigurationItemService）。
type DiscoveryCIWriter interface {
	CreateCI(ctx context.Context, req *dto.CreateCIRequest, tenantID int) (*dto.CIResponse, error)
	UpdateCI(ctx context.Context, id, tenantID int, req *dto.UpdateCIRequest) (*dto.CIResponse, error)
}

// CIProjectionOutcome 是一次回写的结果；Reason 面向日志与任务摘要，不面向用户文案。
type CIProjectionOutcome struct {
	Status string
	Reason string
	CIID   int
}

type DiscoveryCIProjector struct {
	repo   Repository
	writer DiscoveryCIWriter
	logger *zap.SugaredLogger

	mu        sync.Mutex
	ciTypeIDs map[string]int
}

func NewDiscoveryCIProjector(repo Repository, writer DiscoveryCIWriter, logger *zap.SugaredLogger) *DiscoveryCIProjector {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &DiscoveryCIProjector{repo: repo, writer: writer, logger: logger, ciTypeIDs: map[string]int{}}
}

// Project 把一条已落库的云资源写成配置项（没有则建、有则更新发现字段）。
func (p *DiscoveryCIProjector) Project(ctx context.Context, res *CloudResource) CIProjectionOutcome {
	if p == nil || p.writer == nil || p.repo == nil {
		return CIProjectionOutcome{Status: CIProjectionUnavailable, Reason: "projector_not_wired"}
	}
	if res == nil || res.TenantID <= 0 {
		return CIProjectionOutcome{Status: CIProjectionFailed, Reason: "resource_without_tenant"}
	}

	ciTypeName, ok := CITypeForCloudResource(res.ServiceCode, res.ResourceType)
	if !ok {
		return CIProjectionOutcome{
			Status: CIProjectionSkipped,
			Reason: fmt.Sprintf("unmapped_resource_type=%s|%s", res.ServiceCode, res.ResourceType),
		}
	}
	ciTypeID, found := p.resolveCITypeID(ctx, res.TenantID, ciTypeName)
	if !found {
		return CIProjectionOutcome{
			Status: CIProjectionSkipped,
			Reason: fmt.Sprintf("ci_type_missing=%s(tenant=%d)", ciTypeName, res.TenantID),
		}
	}

	linked := p.findLinkedCI(ctx, res)
	if linked == nil {
		return p.createCI(ctx, res, ciTypeID)
	}
	return p.updateCI(ctx, res, linked.ID, ciTypeID)
}

// MarkStale 把已链接的配置项标记为同步滞后；未链接的资源没有可标的账。
func (p *DiscoveryCIProjector) MarkStale(ctx context.Context, res *CloudResource) CIProjectionOutcome {
	if p == nil || p.writer == nil || p.repo == nil {
		return CIProjectionOutcome{Status: CIProjectionUnavailable, Reason: "projector_not_wired"}
	}
	if res == nil || res.TenantID <= 0 {
		return CIProjectionOutcome{Status: CIProjectionFailed, Reason: "resource_without_tenant"}
	}
	linked := p.findLinkedCI(ctx, res)
	if linked == nil {
		return CIProjectionOutcome{Status: CIProjectionSkipped, Reason: CIProjectionReasonNoLinkedCI}
	}
	if linked.CloudSyncStatus == CISyncStatusStale {
		return CIProjectionOutcome{Status: CIProjectionSkipped, Reason: CIProjectionReasonAlreadyStale, CIID: linked.ID}
	}
	if _, err := p.writer.UpdateCI(ctx, linked.ID, res.TenantID, &dto.UpdateCIRequest{
		CloudSyncStatus: CISyncStatusStale,
	}); err != nil {
		p.logger.Warnw("discovery stale marking failed",
			"tenant_id", res.TenantID, "ci_id", linked.ID, "resource_id", res.ResourceID, "error", err)
		return CIProjectionOutcome{Status: CIProjectionFailed, Reason: err.Error(), CIID: linked.ID}
	}
	p.logger.Infow("configuration item marked stale by discovery",
		"tenant_id", res.TenantID, "ci_id", linked.ID, "resource_id", res.ResourceID)
	return CIProjectionOutcome{Status: CIProjectionUpdated, Reason: CIProjectionReasonMarkedStale, CIID: linked.ID}
}

func (p *DiscoveryCIProjector) createCI(ctx context.Context, res *CloudResource, ciTypeID int) CIProjectionOutcome {
	req := p.newCreateCIRequest(res, ciTypeID)
	created, err := p.writer.CreateCI(ctx, req, res.TenantID)
	if err != nil {
		if errors.Is(err, service.ErrCIAttributeValidation) {
			p.logger.Infow("discovery ci projection deferred",
				"tenant_id", res.TenantID, "resource_id", res.ResourceID,
				"ci_type_id", ciTypeID, "error", err)
			return CIProjectionOutcome{Status: CIProjectionDeferred, Reason: err.Error()}
		}
		p.logger.Warnw("discovery ci projection failed",
			"tenant_id", res.TenantID, "resource_id", res.ResourceID, "error", err)
		return CIProjectionOutcome{Status: CIProjectionFailed, Reason: err.Error()}
	}
	p.logger.Infow("configuration item created by discovery",
		"tenant_id", res.TenantID, "ci_id", created.ID, "resource_id", res.ResourceID)
	return CIProjectionOutcome{Status: CIProjectionCreated, Reason: "from_tags", CIID: created.ID}
}

func (p *DiscoveryCIProjector) updateCI(ctx context.Context, res *CloudResource, ciID, ciTypeID int) CIProjectionOutcome {
	req := p.newUpdateCIRequest(res)
	if res.ID > 0 {
		// 补上资源链接，让这条账从此由 ref 定位；类型保持不动，人工归类过的
		// CI 不能被一轮发现改类别。
		req.CloudResourceRefID = res.ID
	}
	updated, err := p.writer.UpdateCI(ctx, ciID, res.TenantID, req)
	if err != nil {
		p.logger.Warnw("discovery ci update failed",
			"tenant_id", res.TenantID, "ci_id", ciID, "resource_id", res.ResourceID,
			"ci_type_id", ciTypeID, "error", err)
		return CIProjectionOutcome{Status: CIProjectionFailed, Reason: err.Error(), CIID: ciID}
	}
	return CIProjectionOutcome{Status: CIProjectionUpdated, Reason: "refreshed", CIID: updated.ID}
}

// newCreateCIRequest 自动建账：固定列取云上事实，动态属性取云标签。
func (p *DiscoveryCIProjector) newCreateCIRequest(res *CloudResource, ciTypeID int) *dto.CreateCIRequest {
	syncedAt := discoverySyncTime(res)
	req := &dto.CreateCIRequest{
		Name:               ciName(res),
		CITypeID:           ciTypeID,
		Status:             ciStatusFromCloudStatus(res.Status),
		Attributes:         ciAttributesFromTags(res.Tags),
		Source:             CIDiscoverySourceValue,
		DiscoverySource:    res.SourceID,
		LastDiscovered:     &syncedAt,
		CloudProvider:      res.Provider,
		CloudAccountID:     res.CanonicalAccountID,
		CloudRegion:        res.Region,
		CloudZone:          res.Zone,
		CloudResourceID:    res.ResourceID,
		CloudResourceType:  res.ResourceType,
		CloudMetadata:      res.Metadata,
		CloudTags:          tagsToAttributes(res.Tags),
		CloudSyncTime:      &syncedAt,
		CloudSyncStatus:    CISyncStatusSuccess,
		CloudResourceRefID: res.ID,
	}
	if env, ok := ciEnvironmentFromTags(res.Tags); ok {
		req.Environment = env
	}
	if owner, ok := tagValue(res.Tags, "owner", "assigned_to", "assignedto"); ok {
		req.AssignedTo = owner
	}
	return req
}

// newUpdateCIRequest 只回写「发现拥有」的字段。
//
// 刻意不动 Attributes/Environment/AssignedTo/OwnedBy：这些可能是人工或导入填的，
// 一轮发现没有资格覆盖；云上的原始标签一直保留在 CloudTags 里可比对。
func (p *DiscoveryCIProjector) newUpdateCIRequest(res *CloudResource) *dto.UpdateCIRequest {
	syncedAt := discoverySyncTime(res)
	return &dto.UpdateCIRequest{
		Name:              ciName(res),
		Status:            ciStatusFromCloudStatus(res.Status),
		Source:            CIDiscoverySourceValue,
		DiscoverySource:   res.SourceID,
		LastDiscovered:    &syncedAt,
		CloudProvider:     res.Provider,
		CloudAccountID:    res.CanonicalAccountID,
		CloudRegion:       res.Region,
		CloudZone:         res.Zone,
		CloudResourceID:   res.ResourceID,
		CloudResourceType: res.ResourceType,
		CloudMetadata:     res.Metadata,
		CloudTags:         tagsToAttributes(res.Tags),
		CloudSyncTime:     &syncedAt,
		CloudSyncStatus:   CISyncStatusSuccess,
	}
}

// findLinkedCI 先按 cloud_resource_ref_id 找（发现自己建的账），再按
// (provider, 云上资源ID) 找（人工/导入建的账，找到就补链接而不是重复建）。
func (p *DiscoveryCIProjector) findLinkedCI(ctx context.Context, res *CloudResource) *ConfigurationItem {
	if res.ID > 0 {
		ci, err := p.repo.GetCIByCloudResourceRefID(ctx, res.TenantID, res.ID)
		if err == nil && ci != nil {
			return ci
		}
		if !ent.IsNotFound(err) {
			p.logger.Warnw("ci lookup by cloud resource ref failed",
				"tenant_id", res.TenantID, "cloud_resource_id", res.ID, "error", err)
		}
	}
	ci, err := p.repo.FindCIByCloudResourceIdentity(ctx, res.TenantID, res.Provider, res.ResourceID)
	if err == nil {
		return ci
	}
	if !ent.IsNotFound(err) {
		p.logger.Warnw("ci lookup by cloud identity failed",
			"tenant_id", res.TenantID, "resource_id", res.ResourceID, "error", err)
	}
	return nil
}

func (p *DiscoveryCIProjector) resolveCITypeID(ctx context.Context, tenantID int, name string) (int, bool) {
	key := fmt.Sprintf("%d|%s", tenantID, name)
	p.mu.Lock()
	cached, ok := p.ciTypeIDs[key]
	p.mu.Unlock()
	if ok {
		return cached, cached > 0
	}
	id, err := p.repo.GetCITypeIDByName(ctx, tenantID, name)
	if err != nil {
		if !ent.IsNotFound(err) {
			p.logger.Warnw("ci type lookup failed", "tenant_id", tenantID, "ci_type", name, "error", err)
		}
		// 不缓存未命中：租户补上该 CI 类型后下一轮发现就该生效。
		return 0, false
	}
	p.mu.Lock()
	p.ciTypeIDs[key] = id
	p.mu.Unlock()
	return id, true
}

// ciProjectionCounts 是发现任务摘要里的回写计数。回写失败不影响发现任务本身，
// 但必须留痕：只报「发现了几台」而掩盖「一条配置项都没落成」是最坏的报表。
type ciProjectionCounts struct {
	created     int
	updated     int
	deferred    int
	skipped     int
	failed      int
	unavailable int
	staleMarked int
}

func (c *ciProjectionCounts) add(o CIProjectionOutcome) {
	if c == nil {
		return
	}
	switch o.Status {
	case CIProjectionCreated:
		c.created++
	case CIProjectionUpdated:
		c.updated++
	case CIProjectionDeferred:
		c.deferred++
	case CIProjectionSkipped:
		c.skipped++
	case CIProjectionFailed:
		c.failed++
	case CIProjectionUnavailable:
		c.unavailable++
	}
}

// addStale 只累计「资源转 stale 后配置项是否跟着标了 stale」，不混进 created/updated。
func (c *ciProjectionCounts) addStale(o CIProjectionOutcome) {
	if c == nil {
		return
	}
	if o.Status == CIProjectionUpdated && o.Reason == CIProjectionReasonMarkedStale {
		c.staleMarked++
		return
	}
	if o.Status == CIProjectionFailed {
		c.failed++
	}
}

func (c *ciProjectionCounts) mergeInto(summary map[string]interface{}) {
	if c == nil || summary == nil {
		return
	}
	summary["ciCreated"] = c.created
	summary["ciUpdated"] = c.updated
	summary["ciDeferred"] = c.deferred
	summary["ciSkipped"] = c.skipped
	summary["ciFailed"] = c.failed
	summary["ciUnavailable"] = c.unavailable
	summary["ciStaleMarked"] = c.staleMarked
}

func ciName(res *CloudResource) string {
	if strings.TrimSpace(res.ResourceName) != "" {
		return strings.TrimSpace(res.ResourceName)
	}
	return strings.TrimSpace(res.ResourceID)
}

func discoverySyncTime(res *CloudResource) time.Time {
	if res.LastSeenAt != nil {
		return *res.LastSeenAt
	}
	return time.Now()
}

// ciStatusFromCloudStatus 把归一后的云上状态映射到 CI 运行状态。
// pending（创建中）按未运行处理：CI.status 表达的是这台资源当前能不能提供服务。
func ciStatusFromCloudStatus(status string) string {
	switch status {
	case "active":
		return common.CIStatusActive
	case "retired":
		return common.CIStatusRetired
	default:
		return common.CIStatusInactive
	}
}

// ciAttributesFromTags 是「标签推断」的全部来源：云标签原样带进动态属性，
// 缺的必填项由 deferred 结果暴露给人补录，代码不补默认值。
func ciAttributesFromTags(tags map[string]string) map[string]interface{} {
	attrs := tagsToAttributes(tags)
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func tagsToAttributes(tags map[string]string) map[string]interface{} {
	if len(tags) == 0 {
		return nil
	}
	attrs := make(map[string]interface{}, len(tags))
	for key, value := range tags {
		attrs[key] = value
	}
	return attrs
}

// ciEnvironmentFromTags 只认无歧义的环境写法；其它值宁可不写（留给 CI 类型默认）
// 也不猜测，因为把生产机器标成开发环境比留空更难发现。
func ciEnvironmentFromTags(tags map[string]string) (string, bool) {
	value, ok := tagValue(tags, "env", "environment")
	if !ok {
		return "", false
	}
	switch strings.ToLower(value) {
	case "prod", "production":
		return "production", true
	case "stage", "staging":
		return "staging", true
	case "dev", "develop", "development":
		return "development", true
	}
	return "", false
}

func tagValue(tags map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		for tagKey, tagValue := range tags {
			if strings.EqualFold(strings.TrimSpace(tagKey), key) {
				if trimmed := strings.TrimSpace(tagValue); trimmed != "" {
					return trimmed, true
				}
			}
		}
	}
	return "", false
}
