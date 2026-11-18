package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/cloud"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Service struct {
	repo             Repository
	productionSvc    *ProductionService
	logger           *zap.SugaredLogger
	discoveryRuntime DiscoveryRuntime
	// ciRequirements 是治理看板计算完整率时的必填属性来源（生产实现是
	// *ConfigurationItemService）。为 nil 表示未装配：治理看板仍可打开，
	// 但所有 CI 都不可评估（Assessable=0、完整率 0），而不是假装满分。
	ciRequirements CIRequirementReader
}

// DiscoveryAdapterInspector 云发现适配器注册表在 CMDB 域内的窄接口。
//
// 之所以要有 SupportedProviders：只靠 HasAdapter，域内无法回答「本产品一共认哪些
// 厂商」，而这个问题决定前端下拉里该出现哪些选项。两者合成一个接口而不是拆成可选
// 接口——可选项一旦缺失就会静默降级成「没有厂商」，那正是要防的失效方式。
type DiscoveryAdapterInspector interface {
	HasAdapter(provider, serviceCode string) bool
	SupportedProviders() []cloud.ProviderSupport
}

type DiscoveryRuntime struct {
	Adapters                DiscoveryAdapterInspector
	CredentialResolverReady bool
	WorkerReady             bool
	// CIProjector 为 nil 表示 CI 域未装配；此时差异只写 CloudResource，
	// 而不是假装配置项已经跟上。
	CIProjector *DiscoveryCIProjector
}

// hasAnyDiscoveryAdapter 问注册表「装了几个适配器」，而不是问某个具体厂商。
// 此前这里硬编码 HasAdapter("aliyun","ecs")，厂商从一家扩到五家后，只配了腾讯云
// 或 AWS 的租户会被判成 disabled。
func (s *Service) hasAnyDiscoveryAdapter() bool {
	if s.discoveryRuntime.Adapters == nil {
		return false
	}
	for _, support := range s.discoveryRuntime.Adapters.SupportedProviders() {
		if !support.NotImplemented {
			return true
		}
	}
	return false
}

// providerHasDiscoveryAdapter 报告该厂商是否真的能自动发现。
// 未知或空 provider 一律判否：把「配了账号但不会自动发现」算成就绪，就会让能力
// 状态说假话，而这类账号的真实结局是任务跑起来才抛 no adapter。
func (s *Service) providerHasDiscoveryAdapter(provider string) bool {
	if s.discoveryRuntime.Adapters == nil {
		return false
	}
	normalized := cloud.NormalizeProvider(provider)
	if normalized == "" {
		return false
	}
	for _, support := range s.discoveryRuntime.Adapters.SupportedProviders() {
		if support.Provider == normalized {
			return !support.NotImplemented
		}
	}
	return false
}

// discoveryDeploymentReadiness 是「这条链路在当前部署里装齐了没有」的唯一判定。
//
// 四个维度逐一给出缺失项：适配器注册表、租户凭据解析器、发现 Worker、部署级申报
// 开关。此前 /capabilities 与 /discovery/health 各写一份同义判断，结果是同一个
// 部署事实在两个端点上有两种说法——加一个维度就要记得改两处，漏改一处就自相矛盾。
// 现在两处都从这里取，维度只增一次。
//
// 它不看租户数据（不查云账号），因此结论对任何租户都成立；租户维度由
// GetDiscoveryCapability 在其之上叠加。
func (s *Service) discoveryDeploymentReadiness() (bool, []string) {
	adapterReady := s.hasAnyDiscoveryAdapter()
	missing := make([]string, 0, 4)
	if !adapterReady {
		missing = append(missing, "discoveryAdapter")
	}
	if !s.discoveryRuntime.CredentialResolverReady {
		missing = append(missing, "tenantSecretResolver")
	}
	if !s.discoveryRuntime.WorkerReady {
		missing = append(missing, "discoveryWorker")
	}
	// 部署级申报与能力矩阵同源：未申报时即使适配器全装齐也不得对外声称就绪。
	envReady := common.EnvEnabled(discoveryReadinessEnv)
	if !envReady {
		missing = append(missing, discoveryReadinessEnv)
	}
	return envReady && adapterReady &&
		s.discoveryRuntime.CredentialResolverReady && s.discoveryRuntime.WorkerReady, missing
}

// GetDiscoveryHealth 报部署级健康事实。
//
// 刻意不再复用 CapabilityStatus：那个结构带 tenantReadiness 字段，而本端点不查
// 租户账号，根本算不出这个值。此前它把该字段留成 Go 零值下发，于是响应里同时出现
// "state":"ready"、"missingRequirements":[]、"tenantReadiness":false —— 前两项说
// 就绪、第三项说没配好。一个端点内部自相矛盾，比两个端点打架更难排查：运维无从
// 判断该按哪一条去修。
//
// 所以这里换成不含租户维度的独立结构：能算的就报，算不了的不报字段，而不是报一个
// 恒假的布尔值让人误以为是「租户没配账号」。租户维度请查 /cmdb/capabilities。
func (s *Service) GetDiscoveryHealth() DiscoveryHealthStatus {
	ready, missing := s.discoveryDeploymentReadiness()
	state := "ready"
	if !ready {
		state = "unready"
	}
	return DiscoveryHealthStatus{
		Key: "cmdbDiscovery", State: state, BuildCapability: true,
		DeploymentReadiness: ready, ActorPermission: true, MissingRequirements: missing,
	}
}

// ListCloudProviders 返回本产品识别的云厂商清单及其发现能力。
//
// 「识别得到」与「能自动发现」是两件事：厂商进了这张清单才能建云账号（DTO 枚举由
// 守卫测试与本清单对齐），notImplemented=true 表示可以手工登记资源但不会自动发现。
// 此前这两件事分属两处清单且互不知情，表现为「适配器已接入但账号建不了」。
func (s *Service) ListCloudProviders() []cloud.ProviderSupport {
	if s.discoveryRuntime.Adapters == nil {
		// 注册表未装配时不能返回空切片：那会被读成「本产品不认任何厂商」，
		// 而真实情况是「认得、但都没接适配器」。全量标未接入才是实话。
		out := make([]cloud.ProviderSupport, 0, len(cloud.KnownProviders))
		for _, p := range cloud.KnownProviders {
			out = append(out, cloud.ProviderSupport{
				Provider:       p,
				DisplayName:    cloud.ProviderDisplayName(p),
				ServiceCodes:   []string{},
				NotImplemented: true,
			})
		}
		return out
	}
	return s.discoveryRuntime.Adapters.SupportedProviders()
}

func NewService(repo Repository, productionSvc *ProductionService, logger *zap.SugaredLogger) *Service {
	return NewServiceWithDiscoveryRuntime(repo, productionSvc, logger, DiscoveryRuntime{})
}

func NewServiceWithDiscoveryRuntime(repo Repository, productionSvc *ProductionService, logger *zap.SugaredLogger, runtime DiscoveryRuntime) *Service {
	return &Service{
		repo:             repo,
		productionSvc:    productionSvc,
		logger:           logger,
		discoveryRuntime: runtime,
	}
}

// WithCIRequirementReader 注入必填属性读取器（治理看板计算属性级完整率用）。
//
// 之所以用注入而非改构造函数签名：构造函数有 20+ 处调用（含全部测试），
// 改签名会把无关改动扩散到每个用例。缺省 nil 的语义是「不可评估」，
// 明确区别于「都填齐了」——这个区分本身就是本批要修的核心。
func (s *Service) WithCIRequirementReader(reader CIRequirementReader) *Service {
	s.ciRequirements = reader
	return s
}

// Legacy CMDB use cases remain owned by the production CMDB service aggregate.
// These methods are the migration bridge used by Handler while those use cases
// are progressively moved into the domain package.
func (s *Service) SearchCI(c *gin.Context) { s.productionSvc.SearchCI(c) }

func (s *Service) ListCIs(c *gin.Context) { s.productionSvc.ListCIs(c) }

func (s *Service) GetCI(c *gin.Context) { s.productionSvc.GetCI(c) }

func (s *Service) CreateCI(c *gin.Context) { s.productionSvc.CreateCI(c) }

func (s *Service) UpdateCI(c *gin.Context) { s.productionSvc.UpdateCI(c) }

func (s *Service) DeleteCI(c *gin.Context) { s.productionSvc.DeleteCI(c) }

func (s *Service) GetCIStats(c *gin.Context) { s.productionSvc.GetCIStats(c) }

func (s *Service) ListCITypes(c *gin.Context) { s.productionSvc.ListCITypes(c) }

func (s *Service) GetCIType(c *gin.Context) { s.productionSvc.GetCIType(c) }

func (s *Service) CreateCIType(c *gin.Context) { s.productionSvc.CreateCIType(c) }

func (s *Service) UpdateCIType(c *gin.Context) { s.productionSvc.UpdateCIType(c) }

func (s *Service) DeleteCIType(c *gin.Context) { s.productionSvc.DeleteCIType(c) }

func (s *Service) ListCIAttributeDefinitions(c *gin.Context) {
	s.productionSvc.ListCIAttributeDefinitions(c)
}

func (s *Service) GetCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.GetCIAttributeDefinition(c)
}

func (s *Service) CreateCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.CreateCIAttributeDefinition(c)
}

func (s *Service) UpdateCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.UpdateCIAttributeDefinition(c)
}

func (s *Service) DeleteCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.DeleteCIAttributeDefinition(c)
}

func (s *Service) ListCITags(c *gin.Context) { s.productionSvc.ListCITags(c) }

func (s *Service) GetCITag(c *gin.Context) { s.productionSvc.GetCITag(c) }

func (s *Service) CreateCITag(c *gin.Context) { s.productionSvc.CreateCITag(c) }

func (s *Service) UpdateCITag(c *gin.Context) { s.productionSvc.UpdateCITag(c) }

func (s *Service) DeleteCITag(c *gin.Context) { s.productionSvc.DeleteCITag(c) }

func (s *Service) ListSavedViews(c *gin.Context) { s.productionSvc.ListSavedViews(c) }

func (s *Service) GetSavedView(c *gin.Context) { s.productionSvc.GetSavedView(c) }

func (s *Service) CreateSavedView(c *gin.Context) { s.productionSvc.CreateSavedView(c) }

func (s *Service) UpdateSavedView(c *gin.Context) { s.productionSvc.UpdateSavedView(c) }

func (s *Service) DeleteSavedView(c *gin.Context) { s.productionSvc.DeleteSavedView(c) }

func (s *Service) ListImportTasks(c *gin.Context) { s.productionSvc.ListImportTasks(c) }

func (s *Service) GetImportTaskStatus(c *gin.Context) { s.productionSvc.GetImportTaskStatus(c) }

func (s *Service) CreateImportTask(c *gin.Context) { s.productionSvc.CreateImportTask(c) }

func (s *Service) ListExportTasks(c *gin.Context) { s.productionSvc.ListExportTasks(c) }

func (s *Service) GetExportTaskStatus(c *gin.Context) { s.productionSvc.GetExportTaskStatus(c) }

func (s *Service) CreateExportTask(c *gin.Context) { s.productionSvc.CreateExportTask(c) }

func (s *Service) DownloadExportTask(c *gin.Context) { s.productionSvc.DownloadExportTask(c) }

func (s *Service) ListCIRelationships(c *gin.Context) { s.productionSvc.ListCIRelationships(c) }

func (s *Service) GetCIRelationship(c *gin.Context) { s.productionSvc.GetCIRelationship(c) }

func (s *Service) CreateCIRelationship(c *gin.Context) { s.productionSvc.CreateCIRelationship(c) }

func (s *Service) UpdateCIRelationship(c *gin.Context) { s.productionSvc.UpdateCIRelationship(c) }

func (s *Service) DeleteCIRelationship(c *gin.Context) { s.productionSvc.DeleteCIRelationship(c) }

func (s *Service) ListCIRelationshipsByCIID(c *gin.Context) {
	s.productionSvc.ListCIRelationshipsByCIID(c)
}

func (s *Service) GetCITopology(c *gin.Context) { s.productionSvc.GetCITopology(c) }

func (s *Service) GetCIImpactAnalysis(c *gin.Context) { s.productionSvc.GetCIImpactAnalysis(c) }

func (s *Service) GetCIHistory(c *gin.Context) { s.productionSvc.GetCIHistory(c) }

func (s *Service) RevertCIVersion(c *gin.Context) { s.productionSvc.RevertCIVersion(c) }

func (s *Service) GetLifecycleHistory(c *gin.Context) { s.productionSvc.GetLifecycleHistory(c) }

func (s *Service) UpdateLifecycleStatus(c *gin.Context) { s.productionSvc.UpdateLifecycleStatus(c) }

func (s *Service) BatchUpdateLifecycleStatus(c *gin.Context) {
	s.productionSvc.BatchUpdateLifecycleStatus(c)
}

func (s *Service) AddTagsToCI(c *gin.Context) { s.productionSvc.AddTagsToCI(c) }

func (s *Service) RemoveTagsFromCI(c *gin.Context) { s.productionSvc.RemoveTagsFromCI(c) }

func (s *Service) BatchCreateCI(c *gin.Context) { s.productionSvc.BatchCreateCI(c) }

func (s *Service) BatchUpdateCI(c *gin.Context) { s.productionSvc.BatchUpdateCI(c) }

func (s *Service) BatchDeleteCI(c *gin.Context) { s.productionSvc.BatchDeleteCI(c) }

func (s *Service) ListRelationshipTypes(c *gin.Context) { s.productionSvc.ListRelationshipTypes(c) }

func (s *Service) GetOntology(c *gin.Context) { s.productionSvc.GetOntology(c) }

type CapabilityStatus struct {
	Key                 string   `json:"key"`
	State               string   `json:"state"`
	BuildCapability     bool     `json:"buildCapability"`
	DeploymentReadiness bool     `json:"deploymentReadiness"`
	TenantReadiness     bool     `json:"tenantReadiness"`
	ActorPermission     bool     `json:"actorPermission"`
	MissingRequirements []string `json:"missingRequirements"`
}

// DiscoveryHealthStatus 是部署级健康面的响应结构，刻意**不含** tenantReadiness。
//
// 不带这个字段不是遗漏：健康端点不查租户账号，无从判断租户是否配好了。给它留一个
// 布尔位就意味着必须填一个值，而任何填出来的值都是猜的（此前留零值，恒 false，
// 与同响应的 state=ready 直接冲突）。不存在的维度就不下发，让调用方去看
// /cmdb/capabilities 拿租户结论。
type DiscoveryHealthStatus struct {
	Key                 string   `json:"key"`
	State               string   `json:"state"`
	BuildCapability     bool     `json:"buildCapability"`
	DeploymentReadiness bool     `json:"deploymentReadiness"`
	ActorPermission     bool     `json:"actorPermission"`
	MissingRequirements []string `json:"missingRequirements"`
}

// discoveryReadinessEnv 是云发现能力的部署级「就绪申报」开关，取值语义与能力矩阵
// (handlers/capability) 的 readinessEnv 完全一致，两者必须得出同一结论。
//
// 注意这是声明层而不是执行闸：未申报为真时，云发现的 job 仍可被创建与执行
// （与 ITSM_AI_READY 同款语义——未申报不影响 triage 实际可用）。它决定的是
// 「对外是否声称已就绪」，因此只影响能力面状态与前端门控。
//
// 此前域内面只判「注册表装了适配器 + 凭据解析器 + Worker」，完全不看这个开关：
// 运维在能力矩阵看到 cmdbDiscovery deploymentReady=false / "deployment dependency
// is not ready"，转到 CMDB 注册页却看到 state=unconfigured、缺失项只有
// "cloudAccountWithDiscoveryAdapter"——于是按提示配好云账号，矩阵仍然说不就绪，
// 缺的那条信息（该开关没申报）在两个接口里都看不到。同一条能力给出两种部署
// 事实，是比「没配好」更难排查的故障。
const discoveryReadinessEnv = "ITSM_CMDB_DISCOVERY_READY"

func (s *Service) GetDiscoveryCapability(ctx context.Context, tenantID int) (*CapabilityStatus, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant ID is required")
	}
	// 部署维度走与健康端点同一份判定：两处若各写一遍，加维度时漏改一处就是
	// 「health 说就绪、capabilities 说缺东西」这类无法从响应本身发现的矛盾。
	adapterReady := s.hasAnyDiscoveryAdapter()
	deploymentReady, missing := s.discoveryDeploymentReadiness()
	accounts, err := s.repo.ListCloudAccounts(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}
	tenantReady := false
	for _, account := range accounts {
		if account.IsActive && account.CredentialRef != "" && s.providerHasDiscoveryAdapter(account.Provider) {
			tenantReady = true
			break
		}
	}
	if !tenantReady {
		missing = append(missing, "cloudAccountWithDiscoveryAdapter")
	}
	state := "ready"
	switch {
	case !adapterReady:
		state = "disabled"
	case !deploymentReady:
		state = "unready"
	case !tenantReady:
		state = "unconfigured"
	}
	return &CapabilityStatus{
		Key: "cmdbDiscovery", State: state, BuildCapability: true,
		DeploymentReadiness: deploymentReady, TenantReadiness: tenantReady,
		ActorPermission: true, MissingRequirements: missing,
	}, nil
}

// 未注册路由的 CI / CIType / 关系相关方法属死代码，已删除；
// 线上 CI 能力由 service/configuration_item_service.go 提供。

// mapCloudNotFound 把 Ent 的「按 tenant+id 查不到这一行」翻成 404 语义。
//
// 不这么做时它是一起内部错误：handler 会把 500/5001 返回给一个只是 ID 写错或
// 资源属于别的租户的请求，客户端无法区分「不存在」和「服务坏了」。
// 其余错误原样透传，交给 common.classifyError 决定状态码。
func mapCloudNotFound(err error, resource string) error {
	if ent.IsNotFound(err) {
		return common.NewNotFoundError(resource)
	}
	return err
}

// Cloud services
func (s *Service) CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	s.logger.Infow("Creating cloud service", "provider", cs.Provider, "service_code", cs.ServiceCode, "tenant_id", cs.TenantID)
	result, err := s.repo.CreateCloudService(ctx, cs)
	if err != nil {
		s.logger.Errorw("Failed to create cloud service", "error", err, "provider", cs.Provider, "service_code", cs.ServiceCode)
		return nil, err
	}
	s.logger.Infow("Cloud service created successfully", "id", result.ID, "provider", result.Provider, "service_code", result.ServiceCode)
	return result, nil
}

func (s *Service) ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
	s.logger.Infow("Listing cloud services", "tenant_id", tenantID, "provider", provider)
	result, err := s.repo.ListCloudServices(ctx, tenantID, provider)
	if err != nil {
		s.logger.Errorw("Failed to list cloud services", "error", err, "tenant_id", tenantID, "provider", provider)
		return nil, err
	}
	s.logger.Infow("Listed cloud services successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error) {
	s.logger.Infow("Getting cloud service", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudService(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud service", "error", err, "id", id, "tenant_id", tenantID)
		return nil, mapCloudNotFound(err, "cloud service")
	}
	s.logger.Infow("Got cloud service successfully", "id", id, "provider", result.Provider)
	return result, nil
}

func (s *Service) UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	s.logger.Infow("Updating cloud service", "id", cs.ID, "provider", cs.Provider, "tenant_id", cs.TenantID)
	result, err := s.repo.UpdateCloudService(ctx, cs)
	if err != nil {
		s.logger.Errorw("Failed to update cloud service", "error", err, "id", cs.ID, "provider", cs.Provider)
		return nil, mapCloudNotFound(err, "cloud service")
	}
	s.logger.Infow("Cloud service updated successfully", "id", result.ID, "provider", result.Provider)
	return result, nil
}

func (s *Service) DeleteCloudService(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud service", "id", id, "tenant_id", tenantID)
	affected, err := s.repo.DeleteCloudService(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud service", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	if affected == 0 {
		s.logger.Warnw("Cloud service delete matched no row", "id", id, "tenant_id", tenantID)
		return common.NewNotFoundError("cloud service")
	}
	s.logger.Infow("Cloud service deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

// Cloud accounts
func (s *Service) CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	s.logger.Infow("Creating cloud account", "provider", ca.Provider, "account_id", ca.AccountID, "tenant_id", ca.TenantID)
	result, err := s.repo.CreateCloudAccount(ctx, ca)
	if err != nil {
		s.logger.Errorw("Failed to create cloud account", "error", err, "provider", ca.Provider, "account_id", ca.AccountID)
		return nil, err
	}
	s.logger.Infow("Cloud account created successfully", "id", result.ID, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
	s.logger.Infow("Listing cloud accounts", "tenant_id", tenantID, "provider", provider)
	result, err := s.repo.ListCloudAccounts(ctx, tenantID, provider)
	if err != nil {
		s.logger.Errorw("Failed to list cloud accounts", "error", err, "tenant_id", tenantID, "provider", provider)
		return nil, err
	}
	s.logger.Infow("Listed cloud accounts successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error) {
	s.logger.Infow("Getting cloud account", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudAccount(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud account", "error", err, "id", id, "tenant_id", tenantID)
		return nil, mapCloudNotFound(err, "cloud account")
	}
	s.logger.Infow("Got cloud account successfully", "id", id, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	s.logger.Infow("Updating cloud account", "id", ca.ID, "provider", ca.Provider, "tenant_id", ca.TenantID)
	result, err := s.repo.UpdateCloudAccount(ctx, ca)
	if err != nil {
		s.logger.Errorw("Failed to update cloud account", "error", err, "id", ca.ID, "provider", ca.Provider)
		return nil, mapCloudNotFound(err, "cloud account")
	}
	s.logger.Infow("Cloud account updated successfully", "id", result.ID, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) DeleteCloudAccount(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud account", "id", id, "tenant_id", tenantID)
	affected, err := s.repo.DeleteCloudAccount(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud account", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	if affected == 0 {
		s.logger.Warnw("Cloud account delete matched no row", "id", id, "tenant_id", tenantID)
		return common.NewNotFoundError("cloud account")
	}
	s.logger.Infow("Cloud account deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

// Cloud resources

// ListCloudResources 分页读取云资源，返回 (当前页, 总数)。
//
// page/pageSize 的唯一所有者是 HTTP 入口的 common.GetPaginationFromQuery（缺省 1/20，
// 越界回落默认页长）；这里只再用 ValidatePagination 兜住非 HTTP 调用方，不新增第三套夹紧规则。
// 因此响应里的 page/pageSize 由 handler 用同一份夹紧结果回显。
func (s *Service) ListCloudResources(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error) {
	page, pageSize = common.ValidatePagination(page, pageSize)
	result, total, err := s.repo.ListCloudResourcesPage(ctx, tenantID, filter, page, pageSize)
	if err != nil {
		s.logger.Errorw("Failed to list cloud resources", "error", err, "tenant_id", tenantID, "provider", filter.Provider, "service_id", filter.ServiceID, "region", filter.Region)
		return nil, 0, err
	}
	s.logger.Infow("Listed cloud resources successfully", "count", len(result), "total", total, "tenant_id", tenantID)
	return result, total, nil
}

func (s *Service) GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error) {
	s.logger.Infow("Getting cloud resource", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudResource(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud resource", "error", err, "id", id, "tenant_id", tenantID)
		return nil, mapCloudNotFound(err, "cloud resource")
	}
	s.logger.Infow("Got cloud resource successfully", "id", id, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	s.logger.Infow("Creating cloud resource", "resource_id", cr.ResourceID, "service_id", cr.ServiceID, "tenant_id", cr.TenantID)
	if err := s.prepareCloudResourceIdentity(ctx, cr); err != nil {
		return nil, err
	}
	result, err := s.repo.CreateCloudResource(ctx, cr)
	if err != nil {
		s.logger.Errorw("Failed to create cloud resource", "error", err, "resource_id", cr.ResourceID, "service_id", cr.ServiceID)
		return nil, err
	}
	s.logger.Infow("Cloud resource created successfully", "id", result.ID, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	s.logger.Infow("Updating cloud resource", "id", cr.ID, "resource_id", cr.ResourceID, "tenant_id", cr.TenantID)
	if err := s.prepareCloudResourceIdentity(ctx, cr); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateCloudResource(ctx, cr)
	if err != nil {
		s.logger.Errorw("Failed to update cloud resource", "error", err, "id", cr.ID, "resource_id", cr.ResourceID)
		return nil, mapCloudNotFound(err, "cloud resource")
	}
	s.logger.Infow("Cloud resource updated successfully", "id", result.ID, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) prepareCloudResourceIdentity(ctx context.Context, resource *CloudResource) error {
	if resource == nil || resource.TenantID <= 0 || resource.CloudAccountID <= 0 || resource.ServiceID <= 0 || strings.TrimSpace(resource.ResourceID) == "" {
		// 缺必填标识是客户端错误，不是内部错误：必须映射 400/1001 而不是 500/5001。
		return common.NewValidationError("cloudAccountId, serviceId and resourceId are required", nil)
	}
	account, err := s.repo.GetCloudAccount(ctx, resource.TenantID, resource.CloudAccountID)
	if err != nil {
		if ent.IsNotFound(err) {
			return common.NewNotFoundError("cloud account")
		}
		return fmt.Errorf("load tenant cloud account: %w", err)
	}
	service, err := s.repo.GetCloudService(ctx, resource.TenantID, resource.ServiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return common.NewNotFoundError("cloud service")
		}
		return fmt.Errorf("load tenant cloud service: %w", err)
	}
	if account == nil || service == nil || account.TenantID != resource.TenantID || service.TenantID != resource.TenantID {
		return common.NewForbiddenError("cloud resource references must belong to the authenticated tenant")
	}
	provider := strings.ToLower(strings.TrimSpace(account.Provider))
	if provider == "" || provider != strings.ToLower(strings.TrimSpace(service.Provider)) {
		// 账号与云服务厂商不一致是请求数据本身矛盾，映射 400 而不是 500。
		return common.NewValidationError("cloud account and cloud service belong to different providers", nil)
	}
	region := strings.ToLower(strings.TrimSpace(resource.Region))
	zone := strings.ToLower(strings.TrimSpace(resource.Zone))
	scope := "global"
	if zone != "" {
		scope = "zonal"
	} else if region != "" {
		scope = "regional"
	}
	resource.IdentityVersion = 1
	resource.Provider = provider
	resource.Partition = "public"
	resource.CanonicalAccountID = strings.TrimSpace(account.AccountID)
	resource.ResourceScope = scope
	resource.Region = region
	resource.Zone = zone
	resource.ServiceCode = strings.ToLower(strings.TrimSpace(service.ServiceCode))
	resource.ResourceType = strings.ToLower(strings.TrimSpace(service.ResourceTypeCode))
	resource.ResourceID = strings.TrimSpace(resource.ResourceID)
	identity := strings.Join([]string{
		"v1", fmt.Sprint(resource.TenantID), resource.Provider, resource.Partition,
		resource.CanonicalAccountID, resource.ResourceScope, resource.Region,
		resource.ServiceCode, resource.ResourceType, resource.ResourceID,
	}, "|")
	hash := sha256.Sum256([]byte(identity))
	resource.IdentityHash = hex.EncodeToString(hash[:])
	return nil
}

func (s *Service) DeleteCloudResource(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud resource", "id", id, "tenant_id", tenantID)
	affected, err := s.repo.DeleteCloudResource(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud resource", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	if affected == 0 {
		s.logger.Warnw("Cloud resource delete matched no row", "id", id, "tenant_id", tenantID)
		return common.NewNotFoundError("cloud resource")
	}
	s.logger.Infow("Cloud resource deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetReconciliation(ctx context.Context, tenantID int) (*ReconciliationResult, error) {
	resources, err := s.repo.ListCloudResources(ctx, tenantID, CloudResourceFilter{})
	if err != nil {
		return nil, err
	}
	cis, err := s.repo.ListCIsForReconciliation(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	resourceIndex := make(map[int]*CloudResource, len(resources))
	for _, res := range resources {
		resourceIndex[res.ID] = res
	}

	usedResources := make(map[int]struct{})
	var orphanCIs []*ConfigurationItem
	var unlinkedCIs []*ConfigurationItem

	for _, ci := range cis {
		if ci.CloudResourceRefID > 0 {
			if _, ok := resourceIndex[ci.CloudResourceRefID]; ok {
				usedResources[ci.CloudResourceRefID] = struct{}{}
			} else {
				orphanCIs = append(orphanCIs, ci)
			}
		} else if ci.CloudResourceID != "" {
			unlinkedCIs = append(unlinkedCIs, ci)
		}
	}

	var unboundResources []*CloudResource
	for _, res := range resources {
		if _, ok := usedResources[res.ID]; !ok {
			unboundResources = append(unboundResources, res)
		}
	}

	result := &ReconciliationResult{
		Summary: ReconciliationSummary{
			ResourceTotal:        len(resources),
			BoundResourceCount:   len(usedResources),
			UnboundResourceCount: len(unboundResources),
			OrphanCICount:        len(orphanCIs),
			UnlinkedCICount:      len(unlinkedCIs),
		},
		UnboundResources: unboundResources,
		OrphanCIs:        orphanCIs,
		UnlinkedCIs:      unlinkedCIs,
	}
	return result, nil
}

// GetGovernanceReport 产出单租户 CMDB 治理看板。
//
// 数据面分两侧，刻意不同源：
//   - 对账侧（Diff）：云资源全集 × 有云引用的 CI 全集。比对的是「云上有、账上没有」
//     这类关系，只有带云引用的 CI 才有意义，沿用 ListCIsForReconciliation。
//   - 治理侧（Quality/Duplicate）：**全租户 CI 全集**（含手工建账、导入、无云引用）。
//     旧实现让两侧共用一个基数，导致手工 CI 永远不进入质量视野——而手工建账恰恰
//     是最容易缺必填属性的那批。
//
// 完整率口径：按该 CI 类型实际配置的必填属性定义算（事实来源
// CIAttributeValidator.RequiredAttributes，经 CIRequirementReader 取得），
// 不再用「有没有 owner/来源」这种与属性无关的代理指标。没有必填定义的类型
// 不进分母——否则全新租户（属性定义表为空）会被报成 100% 完整。
//
// staleThreshold 固定 7 天（两次周度发现的间隔上限）。租户级配置化需先有真实
// 需求与配置入口，本批不预造。
func (s *Service) GetGovernanceReport(ctx context.Context, tenantID int) (*GovernanceReport, error) {
	const staleThreshold = 7 * 24 * time.Hour
	now := time.Now()

	resources, err := s.repo.ListCloudResources(ctx, tenantID, CloudResourceFilter{})
	if err != nil {
		return nil, err
	}
	// 对账基数：只有带云引用的 CI 才有「云上是否有对应资源」这个问题。
	reconcileCIs, err := s.repo.ListCIsForReconciliation(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// 治理基数：全租户 CI。
	everything, err := s.repo.ListCIsForGovernance(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	specs, err := s.requiredAttributeSpecs(ctx, tenantID, everything)
	if err != nil {
		// 完整率读不出必填定义时失败关闭：降级成「假设都填齐了」会让治理看板
		// 与实际数据质量问题反向（这正是本批要修的 fail-open 模式）。
		s.logger.Errorw("GetGovernanceReport: 读取必填属性定义失败，拒绝输出完整率",
			"error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("读取必填属性定义失败: %w", err)
	}

	attrMetrics := ComputeAttributeQuality(everything, specs, staleThreshold, now)
	duplicateGroups := DetectDuplicateGroups(everything)

	report := &GovernanceReport{
		Quality: QualityMetrics{
			Total:                 attrMetrics.Total,
			Active:                attrMetrics.Active,
			Retired:               attrMetrics.Retired,
			Stale:                 attrMetrics.Stale,
			Orphan:                attrMetrics.Orphan,
			Incomplete:            attrMetrics.Incomplete,
			CompletenessPct:       attrMetrics.CompletenessPct,
			Assessable:            attrMetrics.Assessable,
			AttributeDefinedTypes: attrMetrics.AssessedTypes,
			MissingByAttribute:    attrMetrics.MissingByAttribute,
		},
		Diff:           CountByAction(DiffReconciliation(reconcileCIs, resources)),
		CloudResources: len(resources),
		// CIs 上报治理基数（全租户）而不是对账基数：调用方拿它当「这个租户有多少
		// 配置项」用，报成「有多少带云引用的配置项」会小得离谱。
		CIs:              len(everything),
		DuplicateGroups:  duplicateGroups,
		DuplicateCICount: CountDuplicateCIs(duplicateGroups),
		DuplicateRatePct: DuplicateRatePct(CountDuplicateCIs(duplicateGroups), attrMetrics.Active),
	}
	return report, nil
}

// requiredAttributeSpecs 按「本次 CI 集合里出现过的 ciTypeID」解析必填属性名集合。
//
// 只查出现过的类型，不扫全表：类型数量与 CI 数量是两三个数量级的差别，逐类型查
// 一次即可，不产生 N+1。ciTypeID<=0 的 CI（历史脏数据）跳过——查了也会被判
// 「类型不存在」，把它们算进分母只会让完整率无缘无故下降。
func (s *Service) requiredAttributeSpecs(ctx context.Context, tenantID int, cis []*ConfigurationItem) (map[int][]string, error) {
	if s.ciRequirements == nil {
		// 未装配读取器：返回空 map，所有 CI 都不可评估（完整率 0、Assessable 0）。
		// 不返回错误——治理看板仍应能打开，但要如实呈现「没有属性定义可比」。
		return map[int][]string{}, nil
	}
	seen := make(map[int]struct{}, len(cis))
	for _, ci := range cis {
		if ci == nil || ci.CITypeID <= 0 {
			continue
		}
		seen[ci.CITypeID] = struct{}{}
	}
	specs := make(map[int][]string, len(seen))
	for ciTypeID := range seen {
		reqs, err := s.ciRequirements.ListRequiredCIAttributes(ctx, tenantID, ciTypeID)
		if err != nil {
			return nil, fmt.Errorf("读取 CI 类型 %d 的必填属性失败: %w", ciTypeID, err)
		}
		if len(reqs) == 0 {
			continue
		}
		names := make([]string, 0, len(reqs))
		for _, req := range reqs {
			names = append(names, req.Name)
		}
		specs[ciTypeID] = names
	}
	return specs, nil
}

// ListPendingDiscoveryResources 返回「发现 deferred、需要人工补全属性」的云资源全集。
// 推导规则只在 DiscoveryCIProjector 一处；分页由 handler 负责。
func (s *Service) ListPendingDiscoveryResources(ctx context.Context, tenantID int) ([]*PendingDiscoveryResource, error) {
	if s.discoveryRuntime.CIProjector == nil {
		return nil, ErrDiscoveryProjectorNotWired
	}
	return s.discoveryRuntime.CIProjector.ListPending(ctx, tenantID)
}

// RegisterPendingDiscoveryResource 用人工补录的属性把一条待建账云资源写成配置项。
func (s *Service) RegisterPendingDiscoveryResource(ctx context.Context, tenantID, resourceID, operatorID int, req ManualCompletionRequest) (*dto.CIResponse, error) {
	if s.discoveryRuntime.CIProjector == nil {
		return nil, ErrDiscoveryProjectorNotWired
	}
	return s.discoveryRuntime.CIProjector.CompleteManually(ctx, tenantID, resourceID, operatorID, req)
}

// Discovery
func (s *Service) CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error) {
	if ds == nil || ds.TenantID <= 0 {
		return nil, fmt.Errorf("tenant-scoped discovery source is required")
	}
	s.logger.Infow("Creating discovery source", "name", ds.Name, "source_type", ds.SourceType, "tenant_id", ds.TenantID)
	if ds.ReconcilePolicy == "" {
		ds.ReconcilePolicy = "manual"
	}
	if ds.StaleThreshold == 0 {
		ds.StaleThreshold = 3
	}
	if ds.CloudAccountID > 0 {
		account, err := s.repo.GetCloudAccount(ctx, ds.TenantID, ds.CloudAccountID)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, common.NewNotFoundError("cloud account")
			}
			return nil, fmt.Errorf("load tenant cloud account: %w", err)
		}
		if account == nil || account.TenantID != ds.TenantID || !account.IsActive || account.CredentialRef == "" {
			return nil, common.NewForbiddenError("cloud account must belong to the tenant and be configured and active")
		}
		if ds.Provider != "" && !strings.EqualFold(strings.TrimSpace(ds.Provider), strings.TrimSpace(account.Provider)) {
			return nil, fmt.Errorf("discovery source provider does not match cloud account")
		}
		ds.Provider = strings.ToLower(strings.TrimSpace(account.Provider))
	}
	result, err := s.repo.CreateDiscoverySource(ctx, ds)
	if err != nil {
		s.logger.Errorw("Failed to create discovery source", "error", err, "name", ds.Name, "source_type", ds.SourceType)
		return nil, err
	}
	s.logger.Infow("Discovery source created successfully", "id", result.ID, "name", result.Name)
	return result, nil
}

func (s *Service) ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	s.logger.Infow("Listing discovery sources", "tenant_id", tenantID)
	result, err := s.repo.ListDiscoverySources(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to list discovery sources", "error", err, "tenant_id", tenantID)
		return nil, err
	}
	s.logger.Infow("Listed discovery sources successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

// CreateDiscoveryJob 创建发现任务并在同一事务内入队出箱命令。
//
// 幂等语义：请求携带 idempotencyKey 时，同 (tenant, operation, source, key) 的
// 既有任务直接 409 返回原任务（不重复入队）；不携带则每次生成唯一 key
// （指纹去重会让「重复执行同一来源」永远撞唯一索引，违背手动重跑诉求）。
//
// 手动创建按 full_discovery 语义（重置覆盖水位线、强制新窗）；
// 调度器走 CreateScheduledDiscoveryJob（增量推进水位线）。
func (s *Service) CreateDiscoveryJob(ctx context.Context, tenantID, userID int, req *dto.DiscoveryJobRequest) (*DiscoveryJob, error) {
	return s.createDiscoveryJob(ctx, tenantID, userID, req, DiscoveryJobOperationFull)
}

// CreateScheduledDiscoveryJob 调度器专用：按覆盖水位线增量推进
// （operation=incremental_discovery——未完成窗口断点续扫、已完成窗口开新窗）。
func (s *Service) CreateScheduledDiscoveryJob(ctx context.Context, tenantID int, req *dto.DiscoveryJobRequest) (*DiscoveryJob, error) {
	return s.createDiscoveryJob(ctx, tenantID, 0, req, DiscoveryJobOperationIncremental)
}

func (s *Service) createDiscoveryJob(ctx context.Context, tenantID, userID int, req *dto.DiscoveryJobRequest, operation string) (*DiscoveryJob, error) {
	if req == nil || strings.TrimSpace(req.SourceID) == "" {
		return nil, common.NewValidationError("sourceId is required", nil)
	}
	source, err := s.repo.GetDiscoverySource(ctx, tenantID, req.SourceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("discovery source")
		}
		return nil, fmt.Errorf("load discovery source: %w", err)
	}
	if !source.IsActive {
		return nil, common.NewValidationError("发现源已停用，无法创建任务", nil)
	}
	if source.CloudAccountID > 0 {
		account, err := s.repo.GetCloudAccount(ctx, tenantID, source.CloudAccountID)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, common.NewNotFoundError("cloud account")
			}
			return nil, fmt.Errorf("load tenant cloud account: %w", err)
		}
		if !account.IsActive || account.CredentialRef == "" {
			return nil, common.NewForbiddenError("cloud account must be configured and active")
		}
	}

	if req.IdempotencyKey != "" {
		existing, err := s.repo.FindDiscoveryJobByIdempotencyKey(ctx, tenantID, source.ID, req.IdempotencyKey)
		if err != nil && !ent.IsNotFound(err) {
			return nil, fmt.Errorf("check discovery job idempotency: %w", err)
		}
		if existing != nil {
			return nil, common.NewStateConflictError(
				fmt.Sprintf("相同幂等键的任务已存在（job=%d, status=%s）", existing.ID, existing.Status),
				"idempotency key reused")
		}
	}

	fingerprint := discoveryJobFingerprint(tenantID, source)
	idemKey := req.IdempotencyKey
	if idemKey == "" {
		idemKey = fmt.Sprintf("auto-%d", time.Now().UnixNano())
	}
	now := time.Now()
	job := &DiscoveryJob{
		SourceID:           source.ID,
		Status:             DiscoveryJobQueued,
		Operation:          operation,
		IdempotencyKey:     idemKey,
		RequestFingerprint: fingerprint,
		SourceSnapshot: map[string]interface{}{
			"provider":        source.Provider,
			"cloudAccountId":  source.CloudAccountID,
			"reconcilePolicy": source.ReconcilePolicy,
			"staleThreshold":  source.StaleThreshold,
		},
		ScopeSnapshot: map[string]interface{}{
			"serviceCodes": source.ServiceCodes,
			"regions":      source.Regions,
		},
		SnapshotGeneration: "v1",
		RequestedBy:        userID,
		QueuedAt:           &now,
		TenantID:           tenantID,
	}
	created, err := s.repo.CreateDiscoveryJobWithCommand(ctx, job, commandbus.CommandRunCMDBCloudDiscovery, nil)
	if err != nil {
		s.logger.Errorw("Failed to create discovery job", "error", err, "source_id", source.ID)
		return nil, err
	}
	s.logger.Infow("Discovery job created and enqueued", "id", created.ID, "source_id", source.ID)
	return created, nil
}

// ListDiscoveryJobs 分页列出发现任务；status 为空不过滤。
func (s *Service) ListDiscoveryJobs(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*DiscoveryJob, int, error) {
	return s.repo.ListDiscoveryJobs(ctx, tenantID, status, page, pageSize)
}

// GetDiscoveryJob 取单个任务详情。
func (s *Service) GetDiscoveryJob(ctx context.Context, tenantID, id int) (*DiscoveryJob, error) {
	job, err := s.repo.GetDiscoveryJob(ctx, tenantID, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("discovery job")
		}
		return nil, fmt.Errorf("load discovery job: %w", err)
	}
	return job, nil
}

// CancelDiscoveryJob 取消任务：queued 直接终态取消；执行中只记取消请求，
// 由执行器在 scope 边界检查生效；终态任务 409。
func (s *Service) CancelDiscoveryJob(ctx context.Context, tenantID, id int) (*DiscoveryJob, error) {
	job, err := s.GetDiscoveryJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := RequestDiscoveryJobCancellation(job, time.Now()); err != nil {
		return nil, common.NewStateConflictError(
			fmt.Sprintf("任务已终态（%s），无法取消", job.Status), job.Status)
	}
	if err := s.repo.UpdateDiscoveryJob(ctx, job); err != nil {
		return nil, fmt.Errorf("persist cancel for job %d: %w", id, err)
	}
	return job, nil
}

// RetryDiscoveryJob 从 failed/partial_failed 任务派生新任务并入队（parent_job_id 溯源）。
func (s *Service) RetryDiscoveryJob(ctx context.Context, tenantID, userID, id int) (*DiscoveryJob, error) {
	previous, err := s.GetDiscoveryJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	retry, err := NewDiscoveryJobRetry(previous, time.Now())
	if err != nil {
		if err == ErrJobAttemptsExhausted {
			return nil, common.NewStateConflictError("任务重试次数已耗尽，无法再次重试", "attempts exhausted")
		}
		return nil, common.NewStateConflictError(
			fmt.Sprintf("只有失败（failed/partial_failed）的任务可以重试（当前 %s）", previous.Status), previous.Status)
	}
	retry.IdempotencyKey = fmt.Sprintf("retry-%d-%d", previous.ID, time.Now().UnixNano())
	retry.RequestFingerprint = previous.RequestFingerprint
	retry.QueuedAt = timePointer(time.Now())
	created, err := s.repo.CreateDiscoveryJobWithCommand(ctx, retry, commandbus.CommandRunCMDBCloudDiscovery, nil)
	if err != nil {
		s.logger.Errorw("Failed to retry discovery job", "error", err, "parent", previous.ID)
		return nil, err
	}
	s.logger.Infow("Discovery job retried", "parent", previous.ID, "new", created.ID)
	return created, nil
}

// discoveryJobFingerprint 请求指纹：来源声明（provider/账号/服务/区域/策略）的内容哈希。
func discoveryJobFingerprint(tenantID int, source *DiscoverySource) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"tenant":         tenantID,
		"sourceId":       source.ID,
		"provider":       source.Provider,
		"cloudAccountId": source.CloudAccountID,
		"serviceCodes":   source.ServiceCodes,
		"regions":        source.Regions,
		"policy":         source.ReconcilePolicy,
	})
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func (s *Service) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	s.logger.Infow("Listing discovery results", "tenant_id", tenantID, "job_id", jobID)
	result, err := s.repo.ListDiscoveryResults(ctx, tenantID, jobID)
	if err != nil {
		s.logger.Errorw("Failed to list discovery results", "error", err, "tenant_id", tenantID, "job_id", jobID)
		return nil, err
	}
	s.logger.Infow("Listed discovery results successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}
