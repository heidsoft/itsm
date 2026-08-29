package cmdb

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func failCMDBError(c *gin.Context, err error, publicMessage string) {
	var appErr *common.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case common.ErrCodeForbidden:
			common.Forbidden(c, appErr.Message)
		case common.ErrCodeNotFound:
			common.NotFound(c, appErr.Message)
		case common.ErrCodeConflict:
			common.Fail(c, common.ConflictCode, appErr.Message)
		case common.ErrCodeValidation, common.ErrCodeBadRequest:
			common.ParamError(c, appErr.Message)
		default:
			common.FailWithErr(c, err, publicMessage)
		}
		return
	}
	common.FailWithErr(c, err, publicMessage)
}

func (h *Handler) GetCapabilities(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.UnauthorizedCode, "缺少租户认证上下文")
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

// toCIDTO maps domain CI to DTO
func toCIDTO(ci *ConfigurationItem) *dto.CIResponse {
	if ci == nil {
		return nil
	}
	return &dto.CIResponse{
		ID:                 ci.ID,
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

// GetReconciliation 之外的 CI CRUD / CIType CRUD / 关系类型接口未注册路由，属死代码，已删除；
// CI/CIType/关系的线上路径见 controller/cmdb_controller.go + service 层。

// GetReconciliation handles GET /api/v1/cmdb/reconciliation
func (h *Handler) GetReconciliation(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	result, err := h.svc.GetReconciliation(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取对账信息失败: "+err.Error())
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

// Cloud services
func (h *Handler) ListCloudServices(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	provider := c.Query("provider")

	list, err := h.svc.ListCloudServices(c.Request.Context(), tenantID, provider)
	if err != nil {
		common.InternalError(c, "查询云服务列表失败: "+err.Error())
		return
	}

	resp := make([]*dto.CloudServiceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.CloudServiceResponse{
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
		})
	}

	common.Success(c, resp)
}

func (h *Handler) CreateCloudService(c *gin.Context) {
	var req dto.CloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}
	if err := validateAttributeSchema(req.AttributeSchema); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID := c.GetInt("tenant_id")
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
		common.InternalError(c, "创建云服务失败: "+err.Error())
		return
	}
	common.Success(c, &dto.CloudServiceResponse{
		ID:               res.ID,
		ParentID:         res.ParentID,
		Provider:         res.Provider,
		Category:         res.Category,
		ServiceCode:      res.ServiceCode,
		ServiceName:      res.ServiceName,
		ResourceTypeCode: res.ResourceTypeCode,
		ResourceTypeName: res.ResourceTypeName,
		APIVersion:       res.APIVersion,
		AttributeSchema:  res.AttributeSchema,
		IsSystem:         res.IsSystem,
		IsActive:         res.IsActive,
		TenantID:         res.TenantID,
		CreatedAt:        res.CreatedAt,
		UpdatedAt:        res.UpdatedAt,
	})
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
func (h *Handler) ListCloudAccounts(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	provider := c.Query("provider")

	list, err := h.svc.ListCloudAccounts(c.Request.Context(), tenantID, provider)
	if err != nil {
		common.InternalError(c, "查询云账号列表失败: "+err.Error())
		return
	}
	resp := make([]*dto.CloudAccountResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.CloudAccountResponse{
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
		})
	}
	common.Success(c, resp)
}

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
	tenantID := c.GetInt("tenant_id")
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
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, &dto.CloudAccountResponse{
		ID:              res.ID,
		Provider:        res.Provider,
		AccountID:       res.AccountID,
		AccountName:     res.AccountName,
		HasCredential:   res.CredentialRef != "",
		RegionWhitelist: res.RegionWhitelist,
		IsActive:        res.IsActive,
		TenantID:        res.TenantID,
		CreatedAt:       res.CreatedAt,
		UpdatedAt:       res.UpdatedAt,
	})
}

// Cloud resources
func (h *Handler) ListCloudResources(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	provider := c.Query("provider")
	// Bug 修复：使用 camelCase serviceId 保持与 API 字段命名约定一致（AGENTS.md）。
	// 同时兼容旧的 snake_case service_id 查询参数，避免破坏现有调用方。
	serviceID, _ := strconv.Atoi(c.Query("serviceId"))
	if serviceID == 0 {
		serviceID, _ = strconv.Atoi(c.Query("service_id"))
	}
	region := c.Query("region")

	list, err := h.svc.ListCloudResources(c.Request.Context(), tenantID, provider, serviceID, region)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	resp := make([]*dto.CloudResourceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, toCloudResourceDTO(item))
	}
	common.Success(c, resp)
}

// GetCloudService handles GET /api/v1/cmdb/cloud-services/:id
func (h *Handler) GetCloudService(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.GetCloudService(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) UpdateCloudService(c *gin.Context) {
	var req dto.CloudServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID := c.GetInt("tenant_id")
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
		common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) DeleteCloudService(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	err := h.svc.DeleteCloudService(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetCloudAccount handles GET /api/v1/cmdb/cloud-accounts/:id
func (h *Handler) GetCloudAccount(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) UpdateCloudAccount(c *gin.Context) {
	var req dto.CMDBCloudAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID := c.GetInt("tenant_id")
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	existing, err := h.svc.GetCloudAccount(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
		common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) DeleteCloudAccount(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID := c.GetInt("tenant_id")

	err := h.svc.DeleteCloudAccount(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetCloudResource handles GET /api/v1/cmdb/cloud-resources/:id
func (h *Handler) GetCloudResource(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID := c.GetInt("tenant_id")

	result, err := h.svc.GetCloudResource(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// CreateCloudResource handles POST /api/v1/cmdb/cloud-resources
func (h *Handler) CreateCloudResource(c *gin.Context) {
	var req struct {
		CloudAccountID int                    `json:"cloudAccountId" binding:"required"`
		ServiceID      int                    `json:"serviceId" binding:"required"`
		ResourceID     string                 `json:"resourceId" binding:"required"`
		ResourceName   string                 `json:"resourceName"`
		Region         string                 `json:"region"`
		Zone           string                 `json:"zone"`
		Status         string                 `json:"status"`
		Tags           map[string]string      `json:"tags"`
		Metadata       map[string]interface{} `json:"metadata"`
		LifecycleState string                 `json:"lifecycleState"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	tenantID := c.GetInt("tenant_id")
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
		failCMDBError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// UpdateCloudResource handles PUT /api/v1/cmdb/cloud-resources/:id
func (h *Handler) UpdateCloudResource(c *gin.Context) {
	var req struct {
		CloudAccountID int                    `json:"cloudAccountId"`
		ServiceID      int                    `json:"serviceId"`
		ResourceID     string                 `json:"resourceId"`
		ResourceName   string                 `json:"resourceName"`
		Region         string                 `json:"region"`
		Zone           string                 `json:"zone"`
		Status         string                 `json:"status"`
		Tags           map[string]string      `json:"tags"`
		Metadata       map[string]interface{} `json:"metadata"`
		LifecycleState string                 `json:"lifecycleState"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID := c.GetInt("tenant_id")

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
		failCMDBError(c, err, "操作失败")
		return
	}
	common.Success(c, toCloudResourceDTO(result))
}

// DeleteCloudResource handles DELETE /api/v1/cmdb/cloud-resources/:id
func (h *Handler) DeleteCloudResource(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID := c.GetInt("tenant_id")

	err := h.svc.DeleteCloudResource(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// Discovery sources
func (h *Handler) ListDiscoverySources(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	list, err := h.svc.ListDiscoverySources(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
	tenantID := c.GetInt("tenant_id")
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
		failCMDBError(c, err, "操作失败")
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
	tenantID := c.GetInt("tenant_id")
	jobID, _ := common.ParsePositiveIDFromQuery(c, "job_id")
	list, err := h.svc.ListDiscoveryResults(c.Request.Context(), tenantID, jobID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
