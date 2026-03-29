package controller

import (
	"fmt"
	"itsm-backend/common"
	"itsm-backend/dto"
	dddcmdb "itsm-backend/handlers/cmdb"
	"itsm-backend/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CMDBController struct {
	cmdbService             *service.CMDBService
	ciRelationshipService   *service.CIRelationshipService
	auditLogService         *service.AuditLogService
	dddSvc                  *dddcmdb.Service
	cloudDiscoveryService   *service.CloudDiscoveryService
}

func NewCMDBController(
	cmdbService *service.CMDBService,
	ciRelationshipService *service.CIRelationshipService,
	auditLogService *service.AuditLogService,
	dddSvc *dddcmdb.Service,
) *CMDBController {
	return &CMDBController{
		cmdbService:           cmdbService,
		ciRelationshipService: ciRelationshipService,
		auditLogService:       auditLogService,
		dddSvc:                dddSvc,
	}
}

// SetCloudDiscoveryService 设置云发现服务
func (c *CMDBController) SetCloudDiscoveryService(svc *service.CloudDiscoveryService) {
	c.cloudDiscoveryService = svc
}

// CreateCI 创建配置项（统一走 ent 层，携带租户 ID）
func (c *CMDBController) CreateCI(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户ID无效")
		ctx.Abort()
		return
	}

	// 使用 dto.CreateCIRequest 解析，支持 ci_type_id 字段
	var dtoReq dto.CreateCIRequest
	if err := ctx.ShouldBindJSON(&dtoReq); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	// 将 ci_type_id 解析为 ci_type 字符串
	ciType := dtoReq.Type
	if ciType == "" && dtoReq.CITypeID > 0 {
		if ct, err := c.dddSvc.GetType(ctx.Request.Context(), dtoReq.CITypeID, tenantID); err == nil {
			ciType = ct.Name
		}
	}

	// 验证 ciType 不为空
	if ciType == "" {
		common.Fail(ctx, common.ParamErrorCode, "CI类型不能为空，请提供有效的 type 或 ci_type_id")
		return
	}

	assetTag := dtoReq.AssetTag
	serialNumber := dtoReq.SerialNumber
	location := dtoReq.Location
	assignedTo := dtoReq.AssignedTo
	ownedBy := dtoReq.OwnedBy
	discoverySource := dtoReq.DiscoverySource

	req := &service.CreateCIRequest{
		Name:        dtoReq.Name,
		CiType:      ciType,
		CiTypeID:    dtoReq.CITypeID,
		Status:      dtoReq.Status,
		Environment: dtoReq.Environment,
		Criticality: dtoReq.Criticality,
		TenantID:    tenantID,
	}
	if assetTag != "" {
		req.AssetTag = &assetTag
	}
	if serialNumber != "" {
		req.SerialNumber = &serialNumber
	}
	if location != "" {
		req.Location = &location
	}
	if assignedTo != "" {
		req.AssignedTo = &assignedTo
	}
	if ownedBy != "" {
		req.OwnedBy = &ownedBy
	}
	if discoverySource != "" {
		req.DiscoverySource = &discoverySource
	}
	if dtoReq.Attributes != nil {
		req.Attributes = &dtoReq.Attributes
	}

	ci, err := c.cmdbService.CreateCI(ctx.Request.Context(), req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, ci)
}

// GetCI 获取配置项
func (c *CMDBController) GetCI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户ID无效")
		ctx.Abort()
		return
	}

	ci, err := c.cmdbService.GetCI(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "CI not found")
		return
	}

	common.Success(ctx, ci)
}

// ListCIs 列出配置项
func (c *CMDBController) ListCIs(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	req := service.ListCIsRequest{
		TenantID:    tenantID,
		CiType:      ctx.Query("ci_type"),
		Status:      ctx.Query("status"),
		Environment: ctx.Query("environment"),
		Limit:       20,
	}

	if offset := ctx.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			req.Offset = o
		}
	}
	if limit := ctx.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			req.Limit = l
		}
	}

	items, total, err := c.cmdbService.ListCIs(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, gin.H{"items": items, "total": total})
}

// UpdateCI 更新配置项 (via DDD service)
func (c *CMDBController) UpdateCI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	existing, err := c.dddSvc.GetCI(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "Configuration Item not found")
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
	if req.Attributes != nil {
		existing.Attributes = req.Attributes
	}

	res, err := c.dddSvc.UpdateCI(ctx.Request.Context(), existing)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, toCMDBCIDTO(res))
}

// DeleteCI 删除配置项 (via DDD service)
func (c *CMDBController) DeleteCI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := c.dddSvc.DeleteCI(ctx.Request.Context(), id, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, nil)
}

// CreateRelationship 创建CI关系（走 CIRelationshipService，支持富字段）
func (c *CMDBController) CreateRelationship(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	var dtoReq dto.CreateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&dtoReq); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	rel, err := c.ciRelationshipService.CreateCIRelationship(ctx.Request.Context(), &dtoReq, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, rel)
}

// ListRelationships 获取CI关系列表（走 CIRelationshipService，返回富对象）
func (c *CMDBController) ListRelationships(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	ciIDStr := ctx.Query("ci_id")
	if ciIDStr == "" {
		common.ParamError(ctx, "ci_id is required")
		return
	}
	ciID, err := strconv.Atoi(ciIDStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	req := &dto.GetCIRelationshipsRequest{
		CIID:            ciID,
		IncludeOutgoing: true,
		IncludeIncoming: true,
	}
	if v := ctx.Query("relationship_type"); v != "" {
		req.RelationshipType = dto.CIRelationshipType(v)
	}
	if ctx.Query("active_only") == "true" {
		req.ActiveOnly = true
	}

	resp, err := c.ciRelationshipService.GetCIRelationships(ctx.Request.Context(), req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	// 返回平铺列表，前端按 source_ci_id/target_ci_id 自行分组
	all := make([]dto.CIRelationshipResponse, 0, len(resp.OutgoingRelations)+len(resp.IncomingRelations))
	all = append(all, resp.OutgoingRelations...)
	for _, r := range resp.IncomingRelations {
		// 去重：避免双向关系重复
		dup := false
		for _, existing := range resp.OutgoingRelations {
			if existing.ID == r.ID {
				dup = true
				break
			}
		}
		if !dup {
			all = append(all, r)
		}
	}

	common.Success(ctx, all)
}

// DeleteRelationship 删除CI关系
func (c *CMDBController) DeleteRelationship(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid relationship ID")
		return
	}

	if err := c.ciRelationshipService.DeleteCIRelationship(ctx.Request.Context(), id, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, nil)
}

// UpdateRelationship 更新CI关系
func (c *CMDBController) UpdateRelationship(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid relationship ID")
		return
	}

	var req dto.UpdateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	rel, err := c.ciRelationshipService.UpdateCIRelationship(ctx.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, rel)
}

// GetCITopology 获取CI拓扑
func (c *CMDBController) GetCITopology(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	depth := 3
	if depthStr := ctx.Query("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil {
			depth = d
		}
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户ID无效")
		ctx.Abort()
		return
	}

	topology, err := c.cmdbService.GetCITopology(ctx.Request.Context(), id, tenantID, depth)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, topology)
}

// GetCIImpactAnalysis 获取CI影响分析
func (c *CMDBController) GetCIImpactAnalysis(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
		return
	}

	impact, err := c.ciRelationshipService.AnalyzeImpact(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, impact)
}

// GetCIChangeHistory 获取CI变更历史
func (c *CMDBController) GetCIChangeHistory(ctx *gin.Context) {
	idStr := ctx.Param("id")
	ciID, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
		return
	}

	page := 1
	pageSize := 20

	if p := ctx.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	if ps := ctx.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	logs, err := c.auditLogService.GetCIAuditLogs(ctx.Request.Context(), tenantID, ciID, page, pageSize)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, logs)
}

// GetStats 获取CMDB统计 (via DDD service)
func (c *CMDBController) GetStats(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := c.dddSvc.GetStats(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, res)
}

// ListTypes 获取CI类型列表
func (c *CMDBController) ListTypes(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, err := c.dddSvc.ListTypes(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, list)
}

// CreateType 创建CI类型
func (c *CMDBController) CreateType(ctx *gin.Context) {
	var req dto.CreateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ct := &dddcmdb.CIType{
		Name:            req.Name,
		Description:     req.Description,
		Icon:            req.Icon,
		Color:           req.Color,
		AttributeSchema: req.AttributeSchema,
		IsActive:        isActive,
		TenantID:        tenantID,
	}

	res, err := c.dddSvc.CreateType(ctx.Request.Context(), ct)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, toCMDBCITypeDTO(res))
}

// UpdateType 更新CI类型
func (c *CMDBController) UpdateType(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	existing, err := c.dddSvc.GetType(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "CI Type not found")
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Icon = req.Icon
	existing.Color = req.Color
	existing.AttributeSchema = req.AttributeSchema
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	res, err := c.dddSvc.UpdateType(ctx.Request.Context(), existing)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, toCMDBCITypeDTO(res))
}

// DeleteType 删除CI类型
func (c *CMDBController) DeleteType(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	count, err := c.dddSvc.CountCIsByType(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	if count > 0 {
		common.Fail(ctx, common.ParamErrorCode, fmt.Sprintf("Cannot delete CI type: %d CI(s) are using this type", count))
		return
	}

	if err := c.dddSvc.DeleteType(ctx.Request.Context(), id, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, nil)
}

// RunReconciliation 触发对账
func (c *CMDBController) RunReconciliation(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	// 直接调用 GetReconciliation 重新计算并返回最新结果
	result, err := c.dddSvc.GetReconciliation(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, gin.H{
		"message":        "reconciliation completed",
		"resource_total": result.Summary.ResourceTotal,
		"unbound_count":  result.Summary.UnboundResourceCount,
		"orphan_count":   result.Summary.OrphanCICount,
	})
}

// GetReconciliation 获取对账结果
func (c *CMDBController) GetReconciliation(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	result, err := c.dddSvc.GetReconciliation(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	unboundResources := make([]*dto.CloudResourceResponse, 0, len(result.UnboundResources))
	for _, item := range result.UnboundResources {
		unboundResources = append(unboundResources, toCloudResourceDTO(item))
	}

	orphanCIs := make([]*dto.CIResponse, 0, len(result.OrphanCIs))
	for _, item := range result.OrphanCIs {
		orphanCIs = append(orphanCIs, toCMDBCIDTO(item))
	}

	unlinkedCIs := make([]*dto.CIResponse, 0, len(result.UnlinkedCIs))
	for _, item := range result.UnlinkedCIs {
		unlinkedCIs = append(unlinkedCIs, toCMDBCIDTO(item))
	}

	common.Success(ctx, &dto.ReconciliationResponse{
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
	})
}

// ListRelationshipTypes 获取关系类型列表
func (c *CMDBController) ListRelationshipTypes(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, err := c.dddSvc.ListRelationshipTypes(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
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

	common.Success(ctx, resp)
}

// ListCloudServices 获取云服务列表
func (c *CMDBController) ListCloudServices(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := ctx.Query("provider")

	list, err := c.dddSvc.ListCloudServices(ctx.Request.Context(), tenantID, provider)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
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

	common.Success(ctx, resp)
}

// CreateCloudService 创建云服务
func (c *CMDBController) CreateCloudService(ctx *gin.Context) {
	var req dto.CloudServiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isSystem := false
	if req.IsSystem != nil {
		isSystem = *req.IsSystem
	}

	cs := &dddcmdb.CloudService{
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

	res, err := c.dddSvc.CreateCloudService(ctx.Request.Context(), cs)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, &dto.CloudServiceResponse{
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

// ListCloudAccounts 获取云账号列表
func (c *CMDBController) ListCloudAccounts(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := ctx.Query("provider")

	list, err := c.dddSvc.ListCloudAccounts(ctx.Request.Context(), tenantID, provider)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
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

	common.Success(ctx, resp)
}

// CreateCloudAccount 创建云账号
func (c *CMDBController) CreateCloudAccount(ctx *gin.Context) {
	var req dto.CloudAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ca := &dddcmdb.CloudAccount{
		Provider:        req.Provider,
		AccountID:       req.AccountID,
		AccountName:     req.AccountName,
		CredentialRef:   req.CredentialRef,
		RegionWhitelist: req.RegionWhitelist,
		IsActive:        isActive,
		TenantID:        tenantID,
	}

	res, err := c.dddSvc.CreateCloudAccount(ctx.Request.Context(), ca)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, &dto.CloudAccountResponse{
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

// ListCloudResources 获取云资源列表
func (c *CMDBController) ListCloudResources(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	provider := ctx.Query("provider")
	serviceID, _ := strconv.Atoi(ctx.Query("service_id"))
	region := ctx.Query("region")

	list, err := c.dddSvc.ListCloudResources(ctx.Request.Context(), tenantID, provider, serviceID, region)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	resp := make([]*dto.CloudResourceResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, toCloudResourceDTO(item))
	}

	common.Success(ctx, resp)
}

// ListDiscoverySources 获取发现源列表
func (c *CMDBController) ListDiscoverySources(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, err := c.dddSvc.ListDiscoverySources(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
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

	common.Success(ctx, resp)
}

// CreateDiscoverySource 创建发现源
func (c *CMDBController) CreateDiscoverySource(ctx *gin.Context) {
	var req dto.DiscoverySourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ds := &dddcmdb.DiscoverySource{
		ID:          fmt.Sprintf("ds_%d", time.Now().UnixNano()),
		Name:        req.Name,
		SourceType:  req.SourceType,
		Provider:    req.Provider,
		IsActive:    isActive,
		Description: req.Description,
		TenantID:    tenantID,
	}

	res, err := c.dddSvc.CreateDiscoverySource(ctx.Request.Context(), ds)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, &dto.DiscoverySourceResponse{
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

// CreateDiscoveryJob 创建发现任务
func (c *CMDBController) CreateDiscoveryJob(ctx *gin.Context) {
	var req dto.DiscoveryJobRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	startedAt := time.Now()
	job := &dddcmdb.DiscoveryJob{
		SourceID:  req.SourceID,
		Status:    "pending",
		StartedAt: &startedAt,
		TenantID:  tenantID,
	}

	res, err := c.dddSvc.CreateDiscoveryJob(ctx.Request.Context(), job)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
		return
	}

	common.Success(ctx, &dto.DiscoveryJobResponse{
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

// ListDiscoveryResults 获取发现结果列表
func (c *CMDBController) ListDiscoveryResults(ctx *gin.Context) {
	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	jobID, _ := strconv.Atoi(ctx.Query("job_id"))

	list, err := c.dddSvc.ListDiscoveryResults(ctx.Request.Context(), tenantID, jobID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "操作失败，请稍后重试")
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

	common.Success(ctx, resp)
}

// GetDiscoveryStatus 获取云资源发现状态
func (c *CMDBController) GetDiscoveryStatus(ctx *gin.Context) {
	if c.cloudDiscoveryService == nil {
		common.Fail(ctx, common.InternalErrorCode, "云发现服务未初始化")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	if tenantID == 0 {
		tenantID = 1 // 默认租户
	}

	status, err := c.cloudDiscoveryService.GetDiscoveryStatus(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取发现状态失败: "+err.Error())
		return
	}

	common.Success(ctx, status)
}

// RunDiscovery 运行云资源发现任务
func (c *CMDBController) RunDiscovery(ctx *gin.Context) {
	if c.cloudDiscoveryService == nil {
		common.Fail(ctx, common.InternalErrorCode, "云发现服务未初始化")
		return
	}

	tenantIDVal, _ := ctx.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	if tenantID == 0 {
		tenantID = 1 // 默认租户
	}

	// 异步执行发现任务
	go func() {
		err := c.cloudDiscoveryService.DiscoverAll(ctx.Request.Context(), tenantID)
		if err != nil {
			fmt.Printf("Cloud discovery error: %v\n", err)
		}
	}()

	common.Success(ctx, gin.H{"message": "发现任务已启动"})
}

// ==================== Helper mappers ====================

func toCMDBCIDTO(ci *dddcmdb.ConfigurationItem) *dto.CIResponse {
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

func toCMDBCITypeDTO(ct *dddcmdb.CIType) *dto.CITypeResponse {
	if ct == nil {
		return nil
	}
	return &dto.CITypeResponse{
		ID:              ct.ID,
		Name:            ct.Name,
		Description:     ct.Description,
		Icon:            ct.Icon,
		Color:           ct.Color,
		AttributeSchema: ct.AttributeSchema,
		IsActive:        ct.IsActive,
		TenantID:        ct.TenantID,
		CreatedAt:       ct.CreatedAt,
		UpdatedAt:       ct.UpdatedAt,
	}
}

func toCloudResourceDTO(resource *dddcmdb.CloudResource) *dto.CloudResourceResponse {
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
