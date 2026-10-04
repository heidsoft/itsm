package cmdb

import (
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// The methods below expose the production CMDB use cases through the domain
// handler during the controller-to-handler route migration.
func (h *Handler) SearchCI(c *gin.Context) { h.svc.SearchCI(c) }

func (h *Handler) ListCIs(c *gin.Context) { h.svc.ListCIs(c) }

func (h *Handler) GetCI(c *gin.Context) { h.svc.GetCI(c) }

func (h *Handler) CreateCI(c *gin.Context) { h.svc.CreateCI(c) }

func (h *Handler) UpdateCI(c *gin.Context) { h.svc.UpdateCI(c) }

func (h *Handler) DeleteCI(c *gin.Context) { h.svc.DeleteCI(c) }

func (h *Handler) GetCIStats(c *gin.Context) { h.svc.GetCIStats(c) }

func (h *Handler) ListCITypes(c *gin.Context) { h.svc.ListCITypes(c) }

func (h *Handler) GetCIType(c *gin.Context) { h.svc.GetCIType(c) }

func (h *Handler) CreateCIType(c *gin.Context) { h.svc.CreateCIType(c) }

func (h *Handler) UpdateCIType(c *gin.Context) { h.svc.UpdateCIType(c) }

func (h *Handler) DeleteCIType(c *gin.Context) { h.svc.DeleteCIType(c) }

func (h *Handler) ListCIAttributeDefinitions(c *gin.Context) { h.svc.ListCIAttributeDefinitions(c) }

func (h *Handler) GetCIAttributeDefinition(c *gin.Context) { h.svc.GetCIAttributeDefinition(c) }

func (h *Handler) CreateCIAttributeDefinition(c *gin.Context) { h.svc.CreateCIAttributeDefinition(c) }

func (h *Handler) UpdateCIAttributeDefinition(c *gin.Context) { h.svc.UpdateCIAttributeDefinition(c) }

func (h *Handler) DeleteCIAttributeDefinition(c *gin.Context) { h.svc.DeleteCIAttributeDefinition(c) }

func (h *Handler) ListCITags(c *gin.Context) { h.svc.ListCITags(c) }

func (h *Handler) GetCITag(c *gin.Context) { h.svc.GetCITag(c) }

func (h *Handler) CreateCITag(c *gin.Context) { h.svc.CreateCITag(c) }

func (h *Handler) UpdateCITag(c *gin.Context) { h.svc.UpdateCITag(c) }

func (h *Handler) DeleteCITag(c *gin.Context) { h.svc.DeleteCITag(c) }

func (h *Handler) ListSavedViews(c *gin.Context) { h.svc.ListSavedViews(c) }

func (h *Handler) GetSavedView(c *gin.Context) { h.svc.GetSavedView(c) }

func (h *Handler) CreateSavedView(c *gin.Context) { h.svc.CreateSavedView(c) }

func (h *Handler) UpdateSavedView(c *gin.Context) { h.svc.UpdateSavedView(c) }

func (h *Handler) DeleteSavedView(c *gin.Context) { h.svc.DeleteSavedView(c) }

func (h *Handler) ListImportTasks(c *gin.Context) { h.svc.ListImportTasks(c) }

func (h *Handler) GetImportTaskStatus(c *gin.Context) { h.svc.GetImportTaskStatus(c) }

func (h *Handler) CreateImportTask(c *gin.Context) { h.svc.CreateImportTask(c) }

func (h *Handler) ListExportTasks(c *gin.Context) { h.svc.ListExportTasks(c) }

func (h *Handler) GetExportTaskStatus(c *gin.Context) { h.svc.GetExportTaskStatus(c) }

func (h *Handler) CreateExportTask(c *gin.Context) { h.svc.CreateExportTask(c) }

func (h *Handler) ListCIRelationships(c *gin.Context) { h.svc.ListCIRelationships(c) }

func (h *Handler) GetCIRelationship(c *gin.Context) { h.svc.GetCIRelationship(c) }

func (h *Handler) CreateCIRelationship(c *gin.Context) { h.svc.CreateCIRelationship(c) }

func (h *Handler) UpdateCIRelationship(c *gin.Context) { h.svc.UpdateCIRelationship(c) }

func (h *Handler) DeleteCIRelationship(c *gin.Context) { h.svc.DeleteCIRelationship(c) }

func (h *Handler) ListCIRelationshipsByCIID(c *gin.Context) { h.svc.ListCIRelationshipsByCIID(c) }

func (h *Handler) GetCITopology(c *gin.Context) { h.svc.GetCITopology(c) }

func (h *Handler) GetCIImpactAnalysis(c *gin.Context) { h.svc.GetCIImpactAnalysis(c) }

func (h *Handler) GetCIHistory(c *gin.Context) { h.svc.GetCIHistory(c) }

func (h *Handler) RevertCIVersion(c *gin.Context) { h.svc.RevertCIVersion(c) }

func (h *Handler) GetLifecycleHistory(c *gin.Context) { h.svc.GetLifecycleHistory(c) }

func (h *Handler) UpdateLifecycleStatus(c *gin.Context) { h.svc.UpdateLifecycleStatus(c) }

func (h *Handler) BatchUpdateLifecycleStatus(c *gin.Context) { h.svc.BatchUpdateLifecycleStatus(c) }

func (h *Handler) AddTagsToCI(c *gin.Context) { h.svc.AddTagsToCI(c) }

func (h *Handler) RemoveTagsFromCI(c *gin.Context) { h.svc.RemoveTagsFromCI(c) }

func (h *Handler) BatchCreateCI(c *gin.Context) { h.svc.BatchCreateCI(c) }

func (h *Handler) BatchUpdateCI(c *gin.Context) { h.svc.BatchUpdateCI(c) }

func (h *Handler) BatchDeleteCI(c *gin.Context) { h.svc.BatchDeleteCI(c) }

func (h *Handler) ListRelationshipTypes(c *gin.Context) { h.svc.ListRelationshipTypes(c) }

func (h *Handler) GetOntology(c *gin.Context) { h.svc.GetOntology(c) }

// failCMDBError 已删除：它用 switch(appErr.Code) 复刻了 common.classifyError 已有的
// 「领域错误 → 状态码 + 公开文案」映射（E4-5 的第二个分类真相）。域内 15 条云路由与
// 发现端点现在统一走 common.RespondError，分类只剩 common 一处所有者。

func (h *Handler) GetCapabilities(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	capability, err := h.svc.GetDiscoveryCapability(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "获取 CMDB 能力状态失败")
		return
	}
	common.Success(c, &dto.CMDBCapabilitiesResponse{Items: []dto.CMDBCapabilityResponse{{
		Key: capability.Key, State: capability.State,
		BuildCapability: capability.BuildCapability, DeploymentReadiness: capability.DeploymentReadiness,
		TenantReadiness: capability.TenantReadiness, ActorPermission: capability.ActorPermission,
		MissingRequirements: capability.MissingRequirements,
	}}})
}

func (h *Handler) GetDiscoveryHealth(c *gin.Context) {
	health := h.svc.GetDiscoveryHealth()
	common.Success(c, health)
}

// toCIDTO maps domain CI to DTO
func toCIDTO(ci *ConfigurationItem) *dto.CIResponse {
	if ci == nil {
		return nil
	}
	return &dto.CIResponse{
		ID:                 ci.ID,
		CINumber:           ci.CINumber,
		Name:               ci.Name,
		Type:               ci.Type,
		CITypeID:           ci.CITypeID,
		Description:        ci.Description,
		Status:             ci.Status,
		Environment:        ci.Environment,
		Criticality:        ci.Criticality,
		AssetTag:           ci.AssetTag,
		TenantID:           ci.TenantID,
		SerialNumber:       ci.SerialNumber,
		Model:              ci.Model,
		Vendor:             ci.Vendor,
		Location:           ci.Location,
		AssignedTo:         ci.AssignedTo,
		OwnedBy:            ci.OwnedBy,
		DiscoverySource:    ci.DiscoverySource,
		Source:             ci.Source,
		CloudProvider:      ci.CloudProvider,
		CloudAccountID:     ci.CloudAccountID,
		CloudRegion:        ci.CloudRegion,
		CloudZone:          ci.CloudZone,
		CloudResourceID:    ci.CloudResourceID,
		CloudResourceType:  ci.CloudResourceType,
		CloudMetadata:      ci.CloudMetadata,
		CloudTags:          ci.CloudTags,
		CloudMetrics:       ci.CloudMetrics,
		CloudSyncTime:      ci.CloudSyncTime,
		CloudSyncStatus:    ci.CloudSyncStatus,
		CloudResourceRefID: ci.CloudResourceRefID,
		CreatedAt:          ci.CreatedAt,
		UpdatedAt:          ci.UpdatedAt,
		Attributes:         ci.Attributes,
	}
}

func toCloudResourceDTO(resource *CloudResource) *dto.CloudResourceResponse {
	if resource == nil {
		return nil
	}
	return &dto.CloudResourceResponse{
		ID:              resource.ID,
		CloudAccountID:  resource.CloudAccountID,
		ServiceID:       resource.ServiceID,
		ResourceID:      resource.ResourceID,
		IdentityVersion: resource.IdentityVersion, Provider: resource.Provider, Partition: resource.Partition,
		CanonicalAccountID: resource.CanonicalAccountID, ResourceScope: resource.ResourceScope,
		ServiceCode: resource.ServiceCode, ResourceType: resource.ResourceType, IdentityHash: resource.IdentityHash,
		SourceID: resource.SourceID, SourceFingerprint: resource.SourceFingerprint, MissingCount: resource.MissingCount,
		ResourceName:   resource.ResourceName,
		Region:         resource.Region,
		Zone:           resource.Zone,
		Status:         resource.Status,
		Tags:           resource.Tags,
		Metadata:       resource.Metadata,
		FirstSeenAt:    resource.FirstSeenAt,
		LastSeenAt:     resource.LastSeenAt,
		LifecycleState: resource.LifecycleState,
		TenantID:       resource.TenantID,
		CreatedAt:      resource.CreatedAt,
		UpdatedAt:      resource.UpdatedAt,
	}
}

// GetReconciliation 之外的 CI CRUD / CIType CRUD / 关系类型接口由本包的
// ProductionService 承接，生产路径不再依赖 legacy controller。

// GetReconciliation handles GET /api/v1/cmdb/reconciliation
func (h *Handler) GetReconciliation(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	result, err := h.svc.GetReconciliation(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "获取对账信息失败")
		return
	}

	unboundResources := make([]*dto.CloudResourceResponse, 0, len(result.UnboundResources))
	for _, item := range result.UnboundResources {
		unboundResources = append(unboundResources, toCloudResourceDTO(item))
	}

	orphanCIs := make([]*dto.CIResponse, 0, len(result.OrphanCIs))
	for _, item := range result.OrphanCIs {
		orphanCIs = append(orphanCIs, toCIDTO(item))
	}

	unlinkedCIs := make([]*dto.CIResponse, 0, len(result.UnlinkedCIs))
	for _, item := range result.UnlinkedCIs {
		unlinkedCIs = append(unlinkedCIs, toCIDTO(item))
	}

	resp := &dto.ReconciliationResponse{
		Summary: dto.ReconciliationSummary{
			ResourceTotal:        result.Summary.ResourceTotal,
			BoundResourceCount:   result.Summary.BoundResourceCount,
			UnboundResourceCount: result.Summary.UnboundResourceCount,
			OrphanCICount:        result.Summary.OrphanCICount,
			UnlinkedCICount:      result.Summary.UnlinkedCICount,
		},
		UnboundResources: unboundResources,
		OrphanCIs:        orphanCIs,
		UnlinkedCIs:      unlinkedCIs,
	}

	common.Success(c, resp)
}

// ListCloudServices 云服务类型列表。
//
// @Summary  云服务类型列表
// @Description 当前租户的云服务类型目录，是 CI 表单「厂商→服务→资源类型」级联选择的数据源。刻意不分页，因此只返回诚实的 {items,total}，没有 page/pageSize/totalPages。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param provider query string false "云厂商过滤（省略=全部）" Enums(aliyun,tencent,huawei,aws,azure,onprem)
// @Success 200 {object} common.Response{data=dto.CloudServiceListResponse}
// @Failure 400 {object} common.Response "provider 不在六个规范值内"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Router /api/v1/cmdb/cloud-services [get]
func (h *Handler) ListCloudServices(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	// provider 过滤的唯一校验者是这里的 binding，不再是「任意字符串进精确匹配」。
	var req dto.ListCloudSelectorRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	list, err := h.svc.ListCloudServices(c.Request.Context(), tenantID, req.Provider)
	if err != nil {
		common.RespondError(c, err, "查询云服务列表失败")
		return
	}

	items := make([]*dto.CloudServiceResponse, 0, len(list))
	for _, item := range list {
		items = append(items, toCloudServiceDTO(item))
	}

	// 不分页：本端点是 CI 表单的级联选择数据源，整份返回只给诚实的 {items,total}。
	common.Success(c, &dto.CloudServiceListResponse{Items: items, Total: len(items)})
}

func toCloudServiceDTO(item *CloudService) *dto.CloudServiceResponse {
	if item == nil {
		return nil
	}
	return &dto.CloudServiceResponse{
		ID:               item.ID,
		ParentID:         item.ParentID,
		Provider:         item.Provider,
		Category:         item.Category,
		ServiceCode:      item.ServiceCode,
		ServiceName:      item.ServiceName,
		ResourceTypeCode: item.ResourceTypeCode,
		ResourceTypeName: item.ResourceTypeName,
		APIVersion:       item.APIVersion,
		AttributeSchema:  item.AttributeSchema,
		IsSystem:         item.IsSystem,
		IsActive:         item.IsActive,
		TenantID:         item.TenantID,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

// CreateCloudService 新增云服务类型。
//
// @Summary  新增云服务类型
// @Description 租户级云服务类型目录。provider 只接受六个规范值，别名（alibaba/alicloud/qcloud 等）属于适配器边界输入，不可写入。attributeSchema.fields 若非空，必须每项都带 type=select 和非空 options。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CloudServiceRequest true "云服务类型"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Failure 400 {object} common.Response "provider 枚举外 / 必填缺失 / attributeSchema 结构非法"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 cmdb write 权限"
// @Router /api/v1/cmdb/cloud-services [post]
func (h *Handler) CreateCloudService(c *gin.Context) {
	var req dto.CloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Invalid request body")
		return
	}
	if err := validateAttributeSchema(req.AttributeSchema); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isSystem := false
	if req.IsSystem != nil {
		isSystem = *req.IsSystem
	}
	cs := &CloudService{
		ParentID:         req.ParentID,
		Provider:         req.Provider,
		Category:         req.Category,
		ServiceCode:      req.ServiceCode,
		ServiceName:      req.ServiceName,
		ResourceTypeCode: req.ResourceTypeCode,
		ResourceTypeName: req.ResourceTypeName,
		APIVersion:       req.APIVersion,
		AttributeSchema:  req.AttributeSchema,
		IsSystem:         isSystem,
		IsActive:         isActive,
		TenantID:         tenantID,
	}
	res, err := h.svc.CreateCloudService(c.Request.Context(), cs)
	if err != nil {
		common.RespondError(c, err, "创建云服务失败")
		return
	}
	common.Success(c, toCloudServiceDTO(res))
}

func validateAttributeSchema(schema map[string]interface{}) error {
	if schema == nil {
		return nil
	}
	rawFields, ok := schema["fields"]
	if !ok {
		return nil
	}
	fields, ok := rawFields.([]interface{})
	if !ok {
		return fmt.Errorf("attribute_schema.fields must be an array")
	}
	for index, item := range fields {
		fieldMap, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("attribute_schema.fields[%d] 必须是对象", index)
		}
		fieldType, _ := fieldMap["type"].(string)
		fieldKey, _ := fieldMap["key"].(string)
		fieldLabel := fieldKey
		if label, ok := fieldMap["label"].(string); ok && label != "" {
			fieldLabel = label
		}
		if fieldType == "" {
			return fmt.Errorf("attribute_schema.fields[%d]（%s）必须指定 type，且仅支持 select", index, fieldLabel)
		}
		if fieldType != "select" {
			return fmt.Errorf("attribute_schema.fields[%d]（%s）仅支持 type=select", index, fieldLabel)
		}
		rawOptions, ok := fieldMap["options"]
		if !ok {
			return fmt.Errorf("attribute_schema.fields[%d]（%s）必须提供 options", index, fieldLabel)
		}
		options, ok := rawOptions.([]interface{})
		if !ok || len(options) == 0 {
			return fmt.Errorf("attribute_schema.fields[%d]（%s）options 必须为非空数组", index, fieldLabel)
		}
	}
	return nil
}

// Cloud accounts

// ListCloudAccounts 云账号列表。
//
// @Summary  云账号列表
// @Description 当前租户的云账号，是云资源页与 CI 表单的选择器数据源。刻意不分页，只返回诚实的 {items,total}。credentialRef 属于凭据引用，永不外露，只返回 hasCredential 布尔。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param provider query string false "云厂商过滤（省略=全部）" Enums(aliyun,tencent,huawei,aws,azure,onprem)
// @Success 200 {object} common.Response{data=dto.CloudAccountListResponse}
// @Failure 400 {object} common.Response "provider 不在六个规范值内"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Router /api/v1/cmdb/cloud-accounts [get]
func (h *Handler) ListCloudAccounts(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	var req dto.ListCloudSelectorRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	list, err := h.svc.ListCloudAccounts(c.Request.Context(), tenantID, req.Provider)
	if err != nil {
		common.RespondError(c, err, "查询云账号列表失败")
		return
	}
	items := make([]*dto.CloudAccountResponse, 0, len(list))
	for _, item := range list {
		items = append(items, toCloudAccountDTO(item))
	}
	// 不分页：本端点是 CI 表单/云资源页的云账号选择器数据源，整份返回。
	// credentialRef 只以 hasCredential 布尔外露，凭据引用本身不出接口边界。
	common.Success(c, &dto.CloudAccountListResponse{Items: items, Total: len(items)})
}

func toCloudAccountDTO(item *CloudAccount) *dto.CloudAccountResponse {
	if item == nil {
		return nil
	}
	return &dto.CloudAccountResponse{
		ID:              item.ID,
		Provider:        item.Provider,
		AccountID:       item.AccountID,
		AccountName:     item.AccountName,
		HasCredential:   item.CredentialRef != "",
		RegionWhitelist: item.RegionWhitelist,
		IsActive:        item.IsActive,
		TenantID:        item.TenantID,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

// CreateCloudAccount 新增云账号。
//
// @Summary  新增云账号
// @Description 租户级云账号。credentialRef 只是凭据的引用标识（不落明文密钥），写入后经校验保存，读取侧仅外露 hasCredential。provider 只接受六个规范值。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CloudAccountRequest true "云账号"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Failure 400 {object} common.Response "provider 枚举外 / 必填缺失 / credentialRef 跨租户引用"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 cmdb write 权限"
// @Router /api/v1/cmdb/cloud-accounts [post]
func (h *Handler) CreateCloudAccount(c *gin.Context) {
	var req dto.CloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	if err := validateTenantCredentialRef(req.CredentialRef); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	ca := &CloudAccount{
		Provider:        req.Provider,
		AccountID:       req.AccountID,
		AccountName:     req.AccountName,
		CredentialRef:   req.CredentialRef,
		RegionWhitelist: req.RegionWhitelist,
		IsActive:        isActive,
		TenantID:        tenantID,
	}
	res, err := h.svc.CreateCloudAccount(c.Request.Context(), ca)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudAccountDTO(res))
}

// Cloud resources

// ListCloudResources 云资源分页列表。
//
// @Summary  云资源分页列表
// @Description 三个云列表里唯一真分页的一个：total 来自仓储 Count，items 是当前页，page/pageSize/totalPages 回显生效值（缺省 1/20，pageSize 越界回落 20）。provider 走云账号边而不是资源自报表字段。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param provider query string false "云厂商过滤" Enums(aliyun,tencent,huawei,aws,azure,onprem)
// @Param cloudAccountId query int false "云账号ID过滤"
// @Param serviceId query int false "云服务类型ID过滤"
// @Param region query string false "Region 过滤"
// @Param status query string false "资源状态过滤"
// @Param search query string false "搜索关键词（云资源唯一ID前缀）"
// @Param page query int false "页码，缺省 1"
// @Param pageSize query int false "页长，缺省 20，最大 100；越界回落 20"
// @Success 200 {object} common.Response{data=dto.CloudResourceListResponse}
// @Failure 400 {object} common.Response "provider 枚举外 / 查询参数类型非法（含旧的 snake_case service_id）"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Router /api/v1/cmdb/cloud-resources [get]
func (h *Handler) ListCloudResources(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	// service_id 别名已删除：snake_case 查询参数属于旧 /api/v1/cloud/* 表面的契约，
	// 静默接受它会让两套命名继续共存。写错 serviceId 现在是 400，不再是解析失败当 0。
	var req dto.ListCloudResourcesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	page, pageSize := cmdbPagination(c)
	list, total, err := h.svc.ListCloudResources(c.Request.Context(), tenantID, CloudResourceFilter{
		Provider:       req.Provider,
		CloudAccountID: req.CloudAccountID,
		ServiceID:      req.ServiceID,
		Region:         req.Region,
		Status:         req.Status,
		Search:         req.Search,
	}, page, pageSize)
	if err != nil {
		common.RespondError(c, err, "查询云资源列表失败")
		return
	}
	items := make([]*dto.CloudResourceResponse, 0, len(list))
	for _, item := range list {
		items = append(items, toCloudResourceDTO(item))
	}
	common.Success(c, &dto.CloudResourceListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// GetCloudService handles GET /api/v1/cmdb/cloud-services/:id
//
// @Summary  云服务类型详情
// @Description 按 tenant + id 读取；行不存在或属于其他租户统一返回 404，不回露存在性差异。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云服务类型ID"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云服务不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-services/{id} [get]
func (h *Handler) GetCloudService(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.GetCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.CloudServiceResponse{
		ID:               result.ID,
		ParentID:         result.ParentID,
		Provider:         result.Provider,
		Category:         result.Category,
		ServiceCode:      result.ServiceCode,
		ServiceName:      result.ServiceName,
		ResourceTypeCode: result.ResourceTypeCode,
		ResourceTypeName: result.ResourceTypeName,
		APIVersion:       result.APIVersion,
		AttributeSchema:  result.AttributeSchema,
		IsSystem:         result.IsSystem,
		IsActive:         result.IsActive,
		TenantID:         result.TenantID,
		CreatedAt:        result.CreatedAt,
		UpdatedAt:        result.UpdatedAt,
	})
}

// UpdateCloudService handles PUT /api/v1/cmdb/cloud-services/:id
//
// @Summary  更新云服务类型
// @Description 全量覆盖语义：请求体沿用创建 DTO，缺席字段按零值写入。id 取自路径，tenant 取自认证上下文，body 无法自报归属。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云服务类型ID"
// @Param body body dto.CloudServiceRequest true "云服务类型"
// @Success 200 {object} common.Response{data=dto.CloudServiceResponse}
// @Failure 400 {object} common.Response "provider 枚举外 / 必填缺失"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云服务不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-services/{id} [put]
func (h *Handler) UpdateCloudService(c *gin.Context) {
	var req dto.CloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	cs := &CloudService{
		ID:               id,
		ParentID:         req.ParentID,
		Provider:         req.Provider,
		Category:         req.Category,
		ServiceCode:      req.ServiceCode,
		ServiceName:      req.ServiceName,
		ResourceTypeCode: req.ResourceTypeCode,
		ResourceTypeName: req.ResourceTypeName,
		APIVersion:       req.APIVersion,
		AttributeSchema:  req.AttributeSchema,
		IsActive:         isActive,
		TenantID:         tenantID,
	}
	result, err := h.svc.UpdateCloudService(c.Request.Context(), cs)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.CloudServiceResponse{
		ID:               result.ID,
		ParentID:         result.ParentID,
		Provider:         result.Provider,
		Category:         result.Category,
		ServiceCode:      result.ServiceCode,
		ServiceName:      result.ServiceName,
		ResourceTypeCode: result.ResourceTypeCode,
		ResourceTypeName: result.ResourceTypeName,
		APIVersion:       result.APIVersion,
		AttributeSchema:  result.AttributeSchema,
		IsSystem:         result.IsSystem,
		IsActive:         result.IsActive,
		TenantID:         result.TenantID,
		CreatedAt:        result.CreatedAt,
		UpdatedAt:        result.UpdatedAt,
	})
}

// DeleteCloudService handles DELETE /api/v1/cmdb/cloud-services/:id
//
// @Summary  删除云服务类型
// @Description 删除按 tenant + id 条件执行；命中 0 行不再报「删除成功」，而是 404，避免跨租户删除对外表现为成功。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云服务类型ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云服务不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-services/{id} [delete]
func (h *Handler) DeleteCloudService(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	err := h.svc.DeleteCloudService(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetCloudAccount handles GET /api/v1/cmdb/cloud-accounts/:id
//
// @Summary  云账号详情
// @Description 按 tenant + id 读取。凭据引用（credentialRef）不外露，只返回 hasCredential。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云账号不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-accounts/{id} [get]
func (h *Handler) GetCloudAccount(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.CloudAccountResponse{
		ID:              result.ID,
		Provider:        result.Provider,
		AccountID:       result.AccountID,
		AccountName:     result.AccountName,
		HasCredential:   result.CredentialRef != "",
		RegionWhitelist: result.RegionWhitelist,
		IsActive:        result.IsActive,
		TenantID:        result.TenantID,
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.UpdatedAt,
	})
}

// UpdateCloudAccount handles PUT /api/v1/cmdb/cloud-accounts/:id
//
// @Summary  更新云账号
// @Description PATCH 语义的可选字段（指针）：未传即保留原值。provider 与 accountId 不在请求体里，创建后不可改；改凭证前会先按当前租户读取账号，跨租户统一 404。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param body body dto.CMDBCloudAccountUpdateRequest true "可更新字段"
// @Success 200 {object} common.Response{data=dto.CloudAccountResponse}
// @Failure 400 {object} common.Response "字段超长 / credentialRef 跨租户引用"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云账号不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-accounts/{id} [put]
func (h *Handler) UpdateCloudAccount(c *gin.Context) {
	var req dto.CMDBCloudAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	existing, err := h.svc.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	credentialRef := existing.CredentialRef
	if req.CredentialRef != nil && *req.CredentialRef != "" {
		if err := validateTenantCredentialRef(*req.CredentialRef); err != nil {
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}
		credentialRef = *req.CredentialRef
	}
	accountName := existing.AccountName
	if req.AccountName != nil {
		accountName = *req.AccountName
	}
	regionWhitelist := existing.RegionWhitelist
	if req.RegionWhitelist != nil {
		regionWhitelist = *req.RegionWhitelist
	}
	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	ca := &CloudAccount{
		ID:              id,
		Provider:        existing.Provider,
		AccountID:       existing.AccountID,
		AccountName:     accountName,
		CredentialRef:   credentialRef,
		RegionWhitelist: regionWhitelist,
		IsActive:        isActive,
		TenantID:        tenantID,
	}
	result, err := h.svc.UpdateCloudAccount(c.Request.Context(), ca)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.CloudAccountResponse{
		ID:              result.ID,
		Provider:        result.Provider,
		AccountID:       result.AccountID,
		AccountName:     result.AccountName,
		HasCredential:   result.CredentialRef != "",
		RegionWhitelist: result.RegionWhitelist,
		IsActive:        result.IsActive,
		TenantID:        result.TenantID,
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.UpdatedAt,
	})
}

// DeleteCloudAccount handles DELETE /api/v1/cmdb/cloud-accounts/:id
//
// @Summary  删除云账号
// @Description 删除按 tenant + id 条件执行；命中 0 行返回 404，不再对跨租户删除报成功。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云账号不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-accounts/{id} [delete]
func (h *Handler) DeleteCloudAccount(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	err := h.svc.DeleteCloudAccount(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetCloudResource handles GET /api/v1/cmdb/cloud-resources/:id
//
// @Summary  云资源详情
// @Description 按 tenant + id 读取；identity 字段（provider/partition/canonicalAccountId/identityHash 等）由 service 从云账号与云服务派生后外露。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云资源ID"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云资源不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-resources/{id} [get]
func (h *Handler) GetCloudResource(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	result, err := h.svc.GetCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// CreateCloudResource handles POST /api/v1/cmdb/cloud-resources
//
// @Summary  登记云资源
// @Description cloudAccountId 与 serviceId 会先按当前租户重新加载：查不到即 404（跨租户与不存在同义，不回露差异）；两者厂商不一致是 400。identity 字段与 firstSeenAt/lastSeenAt 由 service 派生，请求体无法自报 tenantId。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateCloudResourceRequest true "云资源"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Failure 400 {object} common.Response "必填缺失 / 云账号与云服务厂商不一致"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 cmdb write 权限"
// @Failure 404 {object} common.Response "引用的云账号或云服务不存在/不属于当前租户"
// @Router /api/v1/cmdb/cloud-resources [post]
func (h *Handler) CreateCloudResource(c *gin.Context) {
	var req dto.CreateCloudResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	now := time.Now()
	cr := &CloudResource{
		CloudAccountID: req.CloudAccountID,
		ServiceID:      req.ServiceID,
		ResourceID:     req.ResourceID,
		ResourceName:   req.ResourceName,
		Region:         req.Region,
		Zone:           req.Zone,
		Status:         req.Status,
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		LifecycleState: req.LifecycleState,
		FirstSeenAt:    &now,
		LastSeenAt:     &now,
		TenantID:       tenantID,
	}
	result, err := h.svc.CreateCloudResource(c.Request.Context(), cr)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// UpdateCloudResource handles PUT /api/v1/cmdb/cloud-resources/:id
//
// @Summary  更新云资源
// @Description 全量覆盖语义：缺席字段按零值写入（区分「未传/传零值」属于 PATCH 契约，尚未拍板）。identity 字段在写入前按当前租户重新解析，跨租户引用为 404/403。
// @Tags CMDB-云资源
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "云资源ID"
// @Param body body dto.UpdateCloudResourceRequest true "云资源"
// @Success 200 {object} common.Response{data=dto.CloudResourceResponse}
// @Failure 400 {object} common.Response "字段类型非法 / 必填标识缺失 / 账号与云服务厂商不一致"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云资源不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-resources/{id} [put]
func (h *Handler) UpdateCloudResource(c *gin.Context) {
	var req dto.UpdateCloudResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	cr := &CloudResource{
		ID:             id,
		CloudAccountID: req.CloudAccountID,
		ServiceID:      req.ServiceID,
		ResourceID:     req.ResourceID,
		ResourceName:   req.ResourceName,
		Region:         req.Region,
		Zone:           req.Zone,
		Status:         req.Status,
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		LifecycleState: req.LifecycleState,
		TenantID:       tenantID,
	}
	result, err := h.svc.UpdateCloudResource(c.Request.Context(), cr)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// DeleteCloudResource handles DELETE /api/v1/cmdb/cloud-resources/:id
//
// @Summary  删除云资源
// @Description 删除按 tenant + id 条件执行；命中 0 行返回 404，不再对跨租户删除报成功。
// @Tags CMDB-云资源
// @Produce json
// @Security BearerAuth
// @Param id path int true "云资源ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "id 非正整数"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 404 {object} common.Response "云资源不存在或不属于当前租户"
// @Router /api/v1/cmdb/cloud-resources/{id} [delete]
func (h *Handler) DeleteCloudResource(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	err := h.svc.DeleteCloudResource(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// Discovery sources
func (h *Handler) ListDiscoverySources(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	list, err := h.svc.ListDiscoverySources(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	resp := make([]*dto.DiscoverySourceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.DiscoverySourceResponse{
			ID:             item.ID,
			Name:           item.Name,
			SourceType:     item.SourceType,
			Provider:       item.Provider,
			IsActive:       item.IsActive,
			Description:    item.Description,
			CloudAccountID: item.CloudAccountID, ServiceCodes: item.ServiceCodes, Regions: item.Regions,
			Schedule: item.Schedule, ReconcilePolicy: item.ReconcilePolicy, StaleThreshold: item.StaleThreshold,
			LastSuccessAt: item.LastSuccessAt,
			TenantID:      item.TenantID,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}

func (h *Handler) CreateDiscoverySource(c *gin.Context) {
	var req dto.DiscoverySourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	ds := &DiscoverySource{
		ID:             fmt.Sprintf("ds_%d", time.Now().UnixNano()),
		Name:           req.Name,
		SourceType:     req.SourceType,
		Provider:       req.Provider,
		IsActive:       isActive,
		Description:    req.Description,
		CloudAccountID: req.CloudAccountID, ServiceCodes: req.ServiceCodes, Regions: req.Regions,
		Schedule: req.Schedule, ReconcilePolicy: req.ReconcilePolicy, StaleThreshold: req.StaleThreshold,
		TenantID: tenantID,
	}
	res, err := h.svc.CreateDiscoverySource(c.Request.Context(), ds)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.DiscoverySourceResponse{
		ID:             res.ID,
		Name:           res.Name,
		SourceType:     res.SourceType,
		Provider:       res.Provider,
		IsActive:       res.IsActive,
		Description:    res.Description,
		CloudAccountID: res.CloudAccountID, ServiceCodes: res.ServiceCodes, Regions: res.Regions,
		Schedule: res.Schedule, ReconcilePolicy: res.ReconcilePolicy, StaleThreshold: res.StaleThreshold,
		LastSuccessAt: res.LastSuccessAt,
		TenantID:      res.TenantID,
		CreatedAt:     res.CreatedAt,
		UpdatedAt:     res.UpdatedAt,
	})
}

// Discovery jobs
func (h *Handler) CreateDiscoveryJob(c *gin.Context) {
	var req dto.DiscoveryJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	// 商业化收敛期间禁止创建永远停留在 pending 的假任务。真实发现必须先接入
	// 执行器、状态推进、结果落库、失败重试和审计，再重新开放此入口。
	common.Fail(c, common.ServiceUnavailableCode, "云资源自动发现尚未通过生产验收，当前服务不可用")
}

func (h *Handler) ListDiscoveryResults(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	jobID, _ := common.ParsePositiveIDFromQuery(c, "job_id")
	list, err := h.svc.ListDiscoveryResults(c.Request.Context(), tenantID, jobID)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	resp := make([]*dto.DiscoveryResultResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.DiscoveryResultResponse{
			ID:               item.ID,
			JobID:            item.JobID,
			CIID:             item.CIID,
			Action:           item.Action,
			ResourceType:     item.ResourceType,
			ResourceID:       item.ResourceID,
			ResourceIdentity: item.ResourceIdentity, IdentityVersion: item.IdentityVersion,
			ResourceSnapshot: item.ResourceSnapshot, BeforeHash: item.BeforeHash, AfterHash: item.AfterHash,
			Diff:      item.Diff,
			Status:    item.Status,
			ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
			TenantID:  item.TenantID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}
