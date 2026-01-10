package cmdb

import (
	"fmt"
	"net/http"
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
		ID:             resource.ID,
		CloudAccountID: resource.CloudAccountID,
		ServiceID:      resource.ServiceID,
		ResourceID:     resource.ResourceID,
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

// CreateCI handles POST /api/v1/cmdb/cis
func (h *Handler) CreateCI(c *gin.Context) {
	var req dto.CreateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ci := &ConfigurationItem{
		Name:               req.Name,
		CITypeID:           req.CITypeID,
		Description:        req.Description,
		Status:             req.Status,
		TenantID:           tenantID,
		Environment:        req.Environment,
		Criticality:        req.Criticality,
		AssetTag:           req.AssetTag,
		SerialNumber:       req.SerialNumber,
		Model:              req.Model,
		Vendor:             req.Vendor,
		Location:           req.Location,
		AssignedTo:         req.AssignedTo,
		OwnedBy:            req.OwnedBy,
		DiscoverySource:    req.DiscoverySource,
		Source:             req.Source,
		CloudProvider:      req.CloudProvider,
		CloudAccountID:     req.CloudAccountID,
		CloudRegion:        req.CloudRegion,
		CloudZone:          req.CloudZone,
		CloudResourceID:    req.CloudResourceID,
		CloudResourceType:  req.CloudResourceType,
		CloudMetadata:      req.CloudMetadata,
		CloudTags:          req.CloudTags,
		CloudMetrics:       req.CloudMetrics,
		CloudSyncTime:      req.CloudSyncTime,
		CloudSyncStatus:    req.CloudSyncStatus,
		CloudResourceRefID: req.CloudResourceRefID,
		Attributes:         req.Attributes,
	}

	res, err := h.svc.CreateCI(c.Request.Context(), ci)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toCIDTO(res))
}

// GetCI handles GET /api/v1/cmdb/cis/:id
func (h *Handler) GetCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetCI(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Configuration Item not found")
		return
	}

	common.Success(c, toCIDTO(res))
}

// ListCIs handles GET /api/v1/cmdb/cis
func (h *Handler) ListCIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	typeID, _ := strconv.Atoi(c.Query("ci_type_id"))
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, total, err := h.svc.ListCIs(c.Request.Context(), tenantID, page, pageSize, typeID, status)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var dtos []*dto.CIResponse
	for _, item := range list {
		dtos = append(dtos, toCIDTO(item))
	}

	common.Success(c, gin.H{
		"items": dtos,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// UpdateCI handles PUT /api/v1/cmdb/cis/:id
func (h *Handler) UpdateCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := h.svc.GetCI(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Configuration Item not found")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.CITypeID > 0 {
		existing.CITypeID = req.CITypeID
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	if req.Environment != "" {
		existing.Environment = req.Environment
	}
	if req.Criticality != "" {
		existing.Criticality = req.Criticality
	}
	if req.AssetTag != "" {
		existing.AssetTag = req.AssetTag
	}
	if req.SerialNumber != "" {
		existing.SerialNumber = req.SerialNumber
	}
	if req.Model != "" {
		existing.Model = req.Model
	}
	if req.Vendor != "" {
		existing.Vendor = req.Vendor
	}
	if req.Location != "" {
		existing.Location = req.Location
	}
	if req.AssignedTo != "" {
		existing.AssignedTo = req.AssignedTo
	}
	if req.OwnedBy != "" {
		existing.OwnedBy = req.OwnedBy
	}
	if req.DiscoverySource != "" {
		existing.DiscoverySource = req.DiscoverySource
	}
	if req.Source != "" {
		existing.Source = req.Source
	}
	if req.CloudProvider != "" {
		existing.CloudProvider = req.CloudProvider
	}
	if req.CloudAccountID != "" {
		existing.CloudAccountID = req.CloudAccountID
	}
	if req.CloudRegion != "" {
		existing.CloudRegion = req.CloudRegion
	}
	if req.CloudZone != "" {
		existing.CloudZone = req.CloudZone
	}
	if req.CloudResourceID != "" {
		existing.CloudResourceID = req.CloudResourceID
	}
	if req.CloudResourceType != "" {
		existing.CloudResourceType = req.CloudResourceType
	}
	if req.CloudMetadata != nil {
		existing.CloudMetadata = req.CloudMetadata
	}
	if req.CloudTags != nil {
		existing.CloudTags = req.CloudTags
	}
	if req.CloudMetrics != nil {
		existing.CloudMetrics = req.CloudMetrics
	}
	if req.CloudSyncTime != nil {
		existing.CloudSyncTime = req.CloudSyncTime
	}
	if req.CloudSyncStatus != "" {
		existing.CloudSyncStatus = req.CloudSyncStatus
	}
	if req.CloudResourceRefID > 0 {
		existing.CloudResourceRefID = req.CloudResourceRefID
	}
	if req.Attributes != nil {
		existing.Attributes = req.Attributes
	}

	res, err := h.svc.UpdateCI(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toCIDTO(res))
}

// DeleteCI handles DELETE /api/v1/cmdb/cis/:id
func (h *Handler) DeleteCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := h.svc.DeleteCI(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetStats handles GET /api/v1/cmdb/stats
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, res)
}

// ListTypes handles GET /api/v1/cmdb/types
func (h *Handler) ListTypes(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, err := h.svc.ListTypes(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, list)
}

// GetReconciliation handles GET /api/v1/cmdb/reconciliation
func (h *Handler) GetReconciliation(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	result, err := h.svc.GetReconciliation(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
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
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := c.Query("provider")

	list, err := h.svc.ListCloudServices(c.Request.Context(), tenantID, provider)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
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
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := validateAttributeSchema(req.AttributeSchema); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
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
		common.Fail(c, http.StatusInternalServerError, err.Error())
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
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := c.Query("provider")

	list, err := h.svc.ListCloudAccounts(c.Request.Context(), tenantID, provider)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]*dto.CloudAccountResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.CloudAccountResponse{
			ID:              item.ID,
			Provider:        item.Provider,
			AccountID:       item.AccountID,
			AccountName:     item.AccountName,
			CredentialRef:   item.CredentialRef,
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
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
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
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, &dto.CloudAccountResponse{
		ID:              res.ID,
		Provider:        res.Provider,
		AccountID:       res.AccountID,
		AccountName:     res.AccountName,
		CredentialRef:   res.CredentialRef,
		RegionWhitelist: res.RegionWhitelist,
		IsActive:        res.IsActive,
		TenantID:        res.TenantID,
		CreatedAt:       res.CreatedAt,
		UpdatedAt:       res.UpdatedAt,
	})
}

// Cloud resources
func (h *Handler) ListCloudResources(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := c.Query("provider")
	serviceID, _ := strconv.Atoi(c.Query("service_id"))
	region := c.Query("region")

	list, err := h.svc.ListCloudResources(c.Request.Context(), tenantID, provider, serviceID, region)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]*dto.CloudResourceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.CloudResourceResponse{
			ID:             item.ID,
			CloudAccountID: item.CloudAccountID,
			ServiceID:      item.ServiceID,
			ResourceID:     item.ResourceID,
			ResourceName:   item.ResourceName,
			Region:         item.Region,
			Zone:           item.Zone,
			Status:         item.Status,
			Tags:           item.Tags,
			Metadata:       item.Metadata,
			FirstSeenAt:    item.FirstSeenAt,
			LastSeenAt:     item.LastSeenAt,
			LifecycleState: item.LifecycleState,
			TenantID:       item.TenantID,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}

// Relationship types
func (h *Handler) ListRelationshipTypes(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	list, err := h.svc.ListRelationshipTypes(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]*dto.RelationshipTypeResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.RelationshipTypeResponse{
			ID:          item.ID,
			Name:        item.Name,
			Directional: item.Directional,
			ReverseName: item.ReverseName,
			Description: item.Description,
			TenantID:    item.TenantID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}

// Discovery sources
func (h *Handler) ListDiscoverySources(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	list, err := h.svc.ListDiscoverySources(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]*dto.DiscoverySourceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.DiscoverySourceResponse{
			ID:          item.ID,
			Name:        item.Name,
			SourceType:  item.SourceType,
			Provider:    item.Provider,
			IsActive:    item.IsActive,
			Description: item.Description,
			TenantID:    item.TenantID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}

func (h *Handler) CreateDiscoverySource(c *gin.Context) {
	var req dto.DiscoverySourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	ds := &DiscoverySource{
		ID:          fmt.Sprintf("ds_%d", time.Now().UnixNano()),
		Name:        req.Name,
		SourceType:  req.SourceType,
		Provider:    req.Provider,
		IsActive:    isActive,
		Description: req.Description,
		TenantID:    tenantID,
	}
	res, err := h.svc.CreateDiscoverySource(c.Request.Context(), ds)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, &dto.DiscoverySourceResponse{
		ID:          res.ID,
		Name:        res.Name,
		SourceType:  res.SourceType,
		Provider:    res.Provider,
		IsActive:    res.IsActive,
		Description: res.Description,
		TenantID:    res.TenantID,
		CreatedAt:   res.CreatedAt,
		UpdatedAt:   res.UpdatedAt,
	})
}

// Discovery jobs
func (h *Handler) CreateDiscoveryJob(c *gin.Context) {
	var req dto.DiscoveryJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	startedAt := time.Now()
	job := &DiscoveryJob{
		SourceID:  req.SourceID,
		Status:    "pending",
		StartedAt: &startedAt,
		TenantID:  tenantID,
	}
	res, err := h.svc.CreateDiscoveryJob(c.Request.Context(), job)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, &dto.DiscoveryJobResponse{
		ID:         res.ID,
		SourceID:   res.SourceID,
		Status:     res.Status,
		StartedAt:  res.StartedAt,
		FinishedAt: res.FinishedAt,
		Summary:    res.Summary,
		TenantID:   res.TenantID,
		CreatedAt:  res.CreatedAt,
		UpdatedAt:  res.UpdatedAt,
	})
}

func (h *Handler) ListDiscoveryResults(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	jobID, _ := strconv.Atoi(c.Query("job_id"))
	list, err := h.svc.ListDiscoveryResults(c.Request.Context(), tenantID, jobID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]*dto.DiscoveryResultResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, &dto.DiscoveryResultResponse{
			ID:           item.ID,
			JobID:        item.JobID,
			CIID:         item.CIID,
			Action:       item.Action,
			ResourceType: item.ResourceType,
			ResourceID:   item.ResourceID,
			Diff:         item.Diff,
			Status:       item.Status,
			TenantID:     item.TenantID,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	common.Success(c, resp)
}
