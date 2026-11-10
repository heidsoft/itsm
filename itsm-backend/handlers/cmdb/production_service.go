package cmdb

import (
	"context"
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/schema"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProductionService CMDB控制器
type ProductionService struct {
	logger                       *zap.SugaredLogger
	ciTypeService                *service.CITypeService
	ciAttributeDefinitionService *service.CIAttributeDefinitionService
	ciService                    *service.ConfigurationItemService
	ciRelationshipService        *service.CIRelationshipService
	ciHistoryService             *service.CIHistoryService
	ciTagService                 *service.CITagService
	importExportService          *service.CMDBImportExportService
	savedViewService             *service.CMDBSavedViewService
	toolRegistry                 *service.ToolRegistry // 可选：本体自描述端点暴露 AI 工具面
}

// NewProductionService 创建 CMDB 生产服务（自 controller 迁入）
func NewProductionService(
	logger *zap.SugaredLogger,
	ciTypeService *service.CITypeService,
	ciAttributeDefinitionService *service.CIAttributeDefinitionService,
	ciService *service.ConfigurationItemService,
	ciRelationshipService *service.CIRelationshipService,
	ciHistoryService *service.CIHistoryService,
	ciTagService *service.CITagService,
	importExportService *service.CMDBImportExportService,
	savedViewService *service.CMDBSavedViewService,
) *ProductionService {
	return &ProductionService{
		logger:                       logger,
		ciTypeService:                ciTypeService,
		ciAttributeDefinitionService: ciAttributeDefinitionService,
		ciService:                    ciService,
		ciRelationshipService:        ciRelationshipService,
		ciHistoryService:             ciHistoryService,
		ciTagService:                 ciTagService,
		importExportService:          importExportService,
		savedViewService:             savedViewService,
	}
}

// ------------------------------ CI类型相关接口 ------------------------------

// ListCITypes 获取CI类型列表
// @Summary 获取CI类型列表
// @Description 获取所有CI类型的列表，支持分页和搜索
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Param search query string false "搜索关键词（名称）"
// @Success 200 {object} common.Response{data=dto.CITypeListResponse}
// @Router /api/v1/cmdb/ci-types [get]

// operatorContext 将当前登录用户信息下传到请求 context，供服务层记录变更历史
func (c *ProductionService) operatorContext(ctx *gin.Context) context.Context {
	return service.WithOperator(ctx.Request.Context(), ctx.GetInt("user_id"), authenticatedOperatorName(ctx))
}

// authenticatedOperatorName 读取当前登录用户名。键名单一事实源是 middleware/auth.go 写入的
// "username"；本文件此前误读 "user_name"（没有任何中间件写过该键），
// 使 CI 变更历史与导入/导出任务的 operator_name 全部落成空串。
func authenticatedOperatorName(ctx *gin.Context) string {
	return ctx.GetString("username")
}

// cmdbPagination 取列表分页参数。唯一所有者是 common.GetPaginationFromQuery（缺省 1/20，
// 只采纳落在 (0,100] 的查询值，越界回落默认页长）。CMDB 的列表端点不得再用裸
// strconv.Atoi 读 `size`：那既没有夹紧也认别名，越界值会原样下传给 Ent。
func cmdbPagination(ctx *gin.Context) (page, pageSize int) {
	pagination := common.GetPaginationFromQuery(ctx)
	return pagination.Page, pagination.PageSize
}

func (c *ProductionService) ListCITypes(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	page, pageSize := cmdbPagination(ctx)
	search := ctx.Query("search")

	result, err := c.ciTypeService.ListCITypes(ctx.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		c.logger.Errorw("List CI types failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI type list")
		return
	}

	common.Success(ctx, result)
}

// GetCIType 获取CI类型详情
// @Summary 获取CI类型详情
// @Description 根据ID获取CI类型的详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI类型ID"
// @Success 200 {object} common.Response{data=dto.CITypeResponse}
// @Router /api/v1/cmdb/ci-types/{id} [get]
func (c *ProductionService) GetCIType(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	result, err := c.ciTypeService.GetCITypeByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI type")
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI type does not exist")
		return
	}

	common.Success(ctx, result)
}

// CreateCIType 创建CI类型
// @Summary 创建CI类型
// @Description 创建新的CI类型
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCITypeRequest true "创建CI类型请求"
// @Success 200 {object} common.Response{data=dto.CITypeResponse}
// @Router /api/v1/cmdb/ci-types [post]
func (c *ProductionService) CreateCIType(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CreateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciTypeService.CreateCIType(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI type failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.RespondError(ctx, err, "Failed to create CI type")
		return
	}

	common.Success(ctx, result)
}

// UpdateCIType 更新CI类型
// @Summary 更新CI类型
// @Description 根据ID更新CI类型的信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI类型ID"
// @Param request body dto.UpdateCITypeRequest true "更新CI类型请求"
// @Success 200 {object} common.Response{data=dto.CITypeResponse}
// @Router /api/v1/cmdb/ci-types/{id} [put]
func (c *ProductionService) UpdateCIType(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	var req dto.UpdateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciTypeService.UpdateCIType(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update CI type")
		return
	}

	common.Success(ctx, result)
}

// DeleteCIType 删除CI类型
// @Summary 删除CI类型
// @Description 根据ID删除CI类型
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI类型ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/ci-types/{id} [delete]
func (c *ProductionService) DeleteCIType(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	err := c.ciTypeService.DeleteCIType(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete CI type")
		return
	}

	common.Success(ctx, nil)
}

// ------------------------------ CI属性定义相关接口 ------------------------------

// ListCIAttributeDefinitions 获取CI属性定义列表
// @Summary 获取CI属性定义列表
// @Description 根据CI类型ID获取对应的属性定义列表
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI类型ID"
// @Success 200 {object} common.Response{data=[]dto.CIAttributeDefinitionResponse}
// @Router /api/v1/cmdb/ci-types/{id}/attributes [get]
func (c *ProductionService) ListCIAttributeDefinitions(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	ciTypeIDStr := ctx.Param("id")
	ciTypeID, err := strconv.Atoi(ciTypeIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI type ID parameter")
		return
	}

	result, err := c.ciAttributeDefinitionService.ListCIAttributeDefinitionsByCITypeID(ctx.Request.Context(), ciTypeID, tenantID)
	if err != nil {
		c.logger.Errorw("List CI attribute definitions failed", "error", err, "ci_type_id", ciTypeID, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI attribute definition list")
		return
	}

	common.Success(ctx, result)
}

// GetCIAttributeDefinition 获取CI属性定义详情
// @Summary 获取CI属性定义详情
// @Description 根据ID获取CI属性定义的详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI属性定义ID"
// @Success 200 {object} common.Response{data=dto.CIAttributeDefinitionResponse}
// @Router /api/v1/cmdb/attributes/{id} [get]
func (c *ProductionService) GetCIAttributeDefinition(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	result, err := c.ciAttributeDefinitionService.GetCIAttributeDefinitionByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI attribute definition")
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI attribute definition does not exist")
		return
	}

	common.Success(ctx, result)
}

// CreateCIAttributeDefinition 创建CI属性定义
// @Summary 创建CI属性定义
// @Description 创建新的CI属性定义
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCIAttributeDefinitionRequest true "创建CI属性定义请求"
// @Success 200 {object} common.Response{data=dto.CIAttributeDefinitionResponse}
// @Router /api/v1/cmdb/attributes [post]
func (c *ProductionService) CreateCIAttributeDefinition(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CreateCIAttributeDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciAttributeDefinitionService.CreateCIAttributeDefinition(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI attribute definition failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.RespondError(ctx, err, "Failed to create CI attribute definition")
		return
	}

	common.Success(ctx, result)
}

// UpdateCIAttributeDefinition 更新CI属性定义
// @Summary 更新CI属性定义
// @Description 根据ID更新CI属性定义的信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI属性定义ID"
// @Param request body dto.UpdateCIAttributeDefinitionRequest true "更新CI属性定义请求"
// @Success 200 {object} common.Response{data=dto.CIAttributeDefinitionResponse}
// @Router /api/v1/cmdb/attributes/{id} [put]
func (c *ProductionService) UpdateCIAttributeDefinition(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	var req dto.UpdateCIAttributeDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciAttributeDefinitionService.UpdateCIAttributeDefinition(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update CI attribute definition")
		return
	}

	common.Success(ctx, result)
}

// DeleteCIAttributeDefinition 删除CI属性定义
// @Summary 删除CI属性定义
// @Description 根据ID删除CI属性定义
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI属性定义ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/attributes/{id} [delete]
func (c *ProductionService) DeleteCIAttributeDefinition(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	err := c.ciAttributeDefinitionService.DeleteCIAttributeDefinition(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete CI attribute definition")
		return
	}

	common.Success(ctx, nil)
}

// ------------------------------ 配置项相关接口 ------------------------------

// ListCIs 获取配置项列表
// @Summary 获取配置项列表
// @Description 单一入口：覆盖原 List + Search 双接口。支持分页、过滤、关键词宽模糊、排序、标签过滤、时间范围与关系预加载。
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码（默认1）"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Param ciTypeId query int false "CI类型ID（精确匹配）"
// @Param ciType query string false "CI类型字符串（精确匹配）"
// @Param status query string false "状态（模糊匹配）"
// @Param environment query string false "环境"
// @Param criticality query string false "重要性"
// @Param cloudProvider query string false "云厂商"
// @Param cloudAccountId query string false "云账号ID"
// @Param cloudRegion query string false "云区域"
// @Param assignedTo query string false "负责人（模糊匹配）"
// @Param ownedBy query string false "拥有者（模糊匹配）"
// @Param search query string false "关键词宽模糊（名称/资产标签/序列号/型号/厂商/云资源ID/位置/负责人/归属人）"
// @Param ciNumber query string false "CI唯一业务编号精确匹配（AI Agent 稳定定位实体用自然键）"
// @Param sortBy query string false "排序字段（id/name/status/environment/criticality/created_at/updated_at）"
// @Param sortOrder query string false "排序方向（asc/desc）"
// @Param dateFrom query string false "创建时间起始（RFC3339）"
// @Param dateTo query string false "创建时间截止（RFC3339）"
// @Param tagIds query []int false "标签ID列表（OR）"
// @Param withRelations query bool false "是否预加载关系（默认false）"
// @Success 200 {object} common.Response{data=dto.CIListResponse}
// @Router /api/v1/cmdb/cis [get]
func (c *ProductionService) ListCIs(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.ListCIRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}
	// 分页不来自 ShouldBindQuery：DTO 的分页字段不参与绑定，生效值只由 HTTP 所有者给出。
	req.Page, req.PageSize = cmdbPagination(ctx)

	result, err := c.ciService.ListCIs(ctx.Request.Context(), tenantID, &req)
	if err != nil {
		c.logger.Errorw("List CIs failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain configuration item list")
		return
	}

	common.Success(ctx, result)
}

// GetCI 获取配置项详情
// @Summary 获取配置项详情
// @Description 根据ID获取配置项的详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Param withRelations query bool false "是否包含关系信息"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id} [get]
func (c *ProductionService) GetCI(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	withRelations, _ := strconv.ParseBool(ctx.DefaultQuery("withRelations", "false"))

	result, err := c.ciService.GetCIByID(ctx.Request.Context(), id, tenantID, withRelations)
	if err != nil {
		c.logger.Errorw("Get CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain configuration items")
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "Configuration item does not exist")
		return
	}

	common.Success(ctx, result)
}

// CreateCI 创建配置项
// @Summary 创建配置项
// @Description 创建新的配置项
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCIRequest true "创建配置项请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis [post]
func (c *ProductionService) CreateCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CreateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	if req.CITypeID == 0 {
		common.Fail(ctx, common.ParamErrorCode, "CI type ID cannot be empty")
		return
	}

	// 自动设置租户ID
	req.TenantID = tenantID

	result, err := c.ciService.CreateCI(c.operatorContext(ctx), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.RespondError(ctx, err, "Failed to create configuration item")
		return
	}

	common.Success(ctx, result)
}

// UpdateCI 更新配置项
// @Summary 更新配置项
// @Description 根据ID更新配置项的信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Param request body dto.UpdateCIRequest true "更新配置项请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id} [put]
func (c *ProductionService) UpdateCI(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	var req dto.UpdateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.UpdateCI(c.operatorContext(ctx), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update configuration items")
		return
	}

	common.Success(ctx, result)
}

// DeleteCI 删除配置项
// @Summary 删除配置项
// @Description 根据ID删除配置项
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/cis/{id} [delete]
func (c *ProductionService) DeleteCI(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	err := c.ciService.DeleteCI(c.operatorContext(ctx), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete configuration item")
		return
	}

	common.Success(ctx, nil)
}

// GetCIStats 获取配置项统计
// @Summary 获取配置项统计
// @Description 获取当前租户下配置项的统计信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=dto.CIStatsResponse}
// @Router /api/v1/cmdb/cis/stats [get]
func (c *ProductionService) GetCIStats(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	result, err := c.ciService.GetCIStats(ctx.Request.Context(), tenantID)
	if err != nil {
		c.logger.Errorw("Get CI stats failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain configuration item statistics")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ CI关系相关接口 ------------------------------

// ListCIRelationships 获取CI关系列表
// @Summary 获取CI关系列表
// @Description 获取所有CI关系的列表，支持分页和筛选
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param relationshipType query string false "关系类型"
// @Success 200 {object} common.Response{data=dto.CIRelationshipListResponse}
// @Router /api/v1/cmdb/relationships [get]
func (c *ProductionService) ListCIRelationships(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	pagination := common.GetPaginationFromQuery(ctx)
	page, pageSize := pagination.Page, pagination.PageSize
	relationshipType := ctx.Query("relationshipType")

	result, err := c.ciRelationshipService.ListAllCIRelationships(ctx.Request.Context(), tenantID, page, pageSize, relationshipType)
	if err != nil {
		c.logger.Errorw("List CI relationships failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(ctx, err, "Failed to obtain CI relationship list")
		return
	}

	common.Success(ctx, result)
}

// GetCIRelationship 获取CI关系详情
// @Summary 获取CI关系详情
// @Description 根据ID获取CI关系的详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI关系ID"
// @Success 200 {object} common.Response{data=dto.CIRelationshipResponse}
// @Router /api/v1/cmdb/relationships/{id} [get]
func (c *ProductionService) GetCIRelationship(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	result, err := c.ciRelationshipService.GetCIRelationshipByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI relationship")
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI relationship does not exist")
		return
	}

	common.Success(ctx, result)
}

// ListCIRelationshipsByCIID 根据CI ID获取关系列表
// @Summary 根据CI ID获取关系列表
// @Description 获取指定CI的所有关系列表
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Param direction query string false "关系方向（outgoing/incoming/不传为所有）"
// @Success 200 {object} common.Response{data=[]dto.CIRelationshipResponse}
// @Router /api/v1/cmdb/cis/{id}/relationships [get]
func (c *ProductionService) ListCIRelationshipsByCIID(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	ciIDStr := ctx.Param("id")
	ciID, err := strconv.Atoi(ciIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	direction := ctx.DefaultQuery("direction", "")

	result, err := c.ciRelationshipService.ListCIRelationshipsByCIID(ctx.Request.Context(), ciID, tenantID, direction)
	if err != nil {
		c.logger.Errorw("List CI relationships by CI ID failed", "error", err, "ci_id", ciID, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain CI relationship list")
		return
	}

	common.Success(ctx, result)
}

// CreateCIRelationship 创建CI关系
// @Summary 创建CI关系
// @Description 创建新的CI关系
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCIRelationshipRequest true "创建CI关系请求"
// @Success 200 {object} common.Response{data=dto.CIRelationshipResponse}
// @Router /api/v1/cmdb/relationships [post]
func (c *ProductionService) CreateCIRelationship(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CreateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciRelationshipService.CreateCIRelationship(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		// 词表收口： 词表外关系类型是客户端参数错误（400），不是服务端故障
		if errors.Is(err, service.ErrInvalidRelationshipType) {
			common.ParamError(ctx, "Wrong request parameters")
			return
		}
		c.logger.Errorw("Create CI relationship failed", "error", err, "tenant_id", tenantID, "source_ci_id", req.SourceCIID, "target_ci_id", req.TargetCIID)
		common.RespondError(ctx, err, "Failed to create CI relationship")
		return
	}

	common.Success(ctx, result)
}

// UpdateCIRelationship 更新CI关系
// @Summary 更新CI关系
// @Description 根据ID更新CI关系的信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI关系ID"
// @Param request body dto.UpdateCIRelationshipRequest true "更新CI关系请求"
// @Success 200 {object} common.Response{data=dto.CIRelationshipResponse}
// @Router /api/v1/cmdb/relationships/{id} [put]
func (c *ProductionService) UpdateCIRelationship(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	var req dto.UpdateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciRelationshipService.UpdateCIRelationship(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		// 词表收口： 词表外关系类型是客户端参数错误（400），不是服务端故障
		if errors.Is(err, service.ErrInvalidRelationshipType) {
			common.ParamError(ctx, "Wrong request parameters")
			return
		}
		c.logger.Errorw("Update CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update CI relationship")
		return
	}

	common.Success(ctx, result)
}

// DeleteCIRelationship 删除CI关系
// @Summary 删除CI关系
// @Description 根据ID删除CI关系
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI关系ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/relationships/{id} [delete]
func (c *ProductionService) DeleteCIRelationship(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	err := c.ciRelationshipService.DeleteCIRelationship(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete CI relationship")
		return
	}

	common.Success(ctx, nil)
}

// ListRelationshipTypes 获取内置CI关系类型
// @Summary 获取内置CI关系类型
// @Description 获取CMDB支持的内置关系类型枚举（受控词表单一源：ent/schema/ci_relationship.go，全部 13 种）
// @Tags CMDB
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=dto.GetRelationshipTypesResponse}
// @Router /api/v1/cmdb/relationship-types [get]
func (c *ProductionService) ListRelationshipTypes(ctx *gin.Context) {
	// 词表收口： 从 ent/schema 受控词表派生，禁止在本文件维护第二份清单。
	// schema 词表顺序即常量声明顺序，前端图标/方向语义直接消费。
	types := make([]dto.RelationshipTypeInfo, 0, len(schema.CIRelationshipTypeVocabulary))
	for _, m := range schema.CIRelationshipTypeVocabulary {
		types = append(types, dto.RelationshipTypeInfo{
			Type:        dto.CIRelationshipType(m.Type),
			Name:        m.Name,
			Description: m.Description,
			Direction:   m.Direction,
			Icon:        m.Icon,
			Reverse:     dto.CIRelationshipType(m.Reverse),
		})
	}
	common.Success(ctx, dto.GetRelationshipTypesResponse{Types: types})
}

// GetCITopology 获取统一的 CI 拓扑图
func (c *ProductionService) GetCITopology(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}
	ciID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}
	depth, _ := strconv.Atoi(ctx.DefaultQuery("depth", "3"))
	result, err := c.ciRelationshipService.GetCITopology(ctx.Request.Context(), ciID, tenantID, depth)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(ctx, "Configuration item does not exist")
			return
		}
		c.logger.Errorw("Get CI topology failed", "errorClass", "internal", "ciId", ciID, "tenantId", tenantID)
		common.InternalError(ctx, "Failed to obtain CI topology")
		return
	}
	common.Success(ctx, result)
}

// GetCIImpactAnalysis 获取CI影响分析
// @Summary 获取CI影响分析
// @Description 分析指定CI故障时可能影响的其他CI
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "配置项ID"
// @Param maxDepth query int false "最大分析深度，默认3"
// @Success 200 {object} common.Response{data=dto.CIImpactAnalysisResponse}
// @Router /api/v1/cmdb/cis/{id}/impact-analysis [get]
func (c *ProductionService) GetCIImpactAnalysis(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	ciIDStr := ctx.Param("id")
	ciID, err := strconv.Atoi(ciIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	maxDepth, _ := strconv.Atoi(ctx.DefaultQuery("maxDepth", "3"))
	if maxDepth <= 0 {
		maxDepth = 3
	}

	result, err := c.ciRelationshipService.GetCIImpactAnalysis(ctx.Request.Context(), ciID, tenantID, maxDepth)
	if err != nil {
		if ent.IsNotFound(err) {
			common.NotFound(ctx, "Configuration item does not exist")
			return
		}
		c.logger.Errorw("Get CI impact analysis failed", "errorClass", "internal", "ciId", ciID, "tenantId", tenantID)
		common.InternalError(ctx, "Failed to obtain CI impact analysis")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ CI Tag 相关接口 ------------------------------

// ListCITags 获取CI标签列表
// @Summary 获取CI标签列表
// @Description 获取所有CI标签的列表，支持分页和搜索
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Param search query string false "搜索关键词（标签名/值）"
// @Success 200 {object} common.Response{data=dto.CITagListResponse}
// @Router /api/v1/cmdb/tags [get]
func (c *ProductionService) ListCITags(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	page, pageSize := cmdbPagination(ctx)
	search := ctx.Query("search")

	result, err := c.ciTagService.ListCITags(ctx.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		c.logger.Errorw("List CI tags failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain tag list")
		return
	}

	common.Success(ctx, result)
}

// GetCITag 获取CI标签详情
// @Summary 获取CI标签详情
// @Description 根据ID获取CI标签的详细信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Success 200 {object} common.Response{data=dto.CITagResponse}
// @Router /api/v1/cmdb/tags/{id} [get]
func (c *ProductionService) GetCITag(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	result, err := c.ciTagService.GetCITagByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to get tags")
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "Tag does not exist")
		return
	}

	common.Success(ctx, result)
}

// CreateCITag 创建CI标签
// @Summary 创建CI标签
// @Description 创建新的CI标签
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCITagRequest true "创建标签请求"
// @Success 200 {object} common.Response{data=dto.CITagResponse}
// @Router /api/v1/cmdb/tags [post]
func (c *ProductionService) CreateCITag(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CreateCITagRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciTagService.CreateCITag(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI tag failed", "error", err, "tenant_id", tenantID, "key", req.Key)
		common.RespondError(ctx, err, "Failed to create label")
		return
	}

	common.Success(ctx, result)
}

// UpdateCITag 更新CI标签
// @Summary 更新CI标签
// @Description 根据ID更新CI标签的信息
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Param request body dto.UpdateCITagRequest true "更新标签请求"
// @Success 200 {object} common.Response{data=dto.CITagResponse}
// @Router /api/v1/cmdb/tags/{id} [put]
func (c *ProductionService) UpdateCITag(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	var req dto.UpdateCITagRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciTagService.UpdateCITag(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update label")
		return
	}

	common.Success(ctx, result)
}

// DeleteCITag 删除CI标签
// @Summary 删除CI标签
// @Description 根据ID删除CI标签
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/tags/{id} [delete]
func (c *ProductionService) DeleteCITag(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	err := c.ciTagService.DeleteCITag(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete tag")
		return
	}

	common.Success(ctx, nil)
}

// AddTagsToCI 给CI添加标签
// @Summary 给CI添加标签
// @Description 给指定的CI添加多个标签
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param request body dto.AddCITagsRequest true "添加标签请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id}/tags [post]
func (c *ProductionService) AddTagsToCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	var req dto.AddCITagsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.AddTagsToCI(c.operatorContext(ctx), id, tenantID, req.TagIDs)
	if err != nil {
		c.logger.Errorw("Add tags to CI failed", "error", err, "ci_id", id, "tag_ids", req.TagIDs, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to add tag")
		return
	}

	common.Success(ctx, result)
}

// RemoveTagsFromCI 从CI移除标签
// @Summary 从CI移除标签
// @Description 从指定的CI移除多个标签
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param request body dto.RemoveCITagsRequest true "移除标签请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id}/tags [delete]
func (c *ProductionService) RemoveTagsFromCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	var req dto.RemoveCITagsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.RemoveTagsFromCI(c.operatorContext(ctx), id, tenantID, req.TagIDs)
	if err != nil {
		c.logger.Errorw("Remove tags from CI failed", "error", err, "ci_id", id, "tag_ids", req.TagIDs, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to remove tag")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ CI History 相关接口 ------------------------------

// GetCIHistory 获取CI历史记录
// @Summary 获取CI历史记录
// @Description 获取指定CI的所有变更历史记录
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Success 200 {object} common.Response{data=dto.CIHistoryListResponse}
// @Router /api/v1/cmdb/cis/{id}/history [get]
func (c *ProductionService) GetCIHistory(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	page, pageSize := cmdbPagination(ctx)

	result, err := c.ciHistoryService.GetCIHistory(ctx.Request.Context(), id, tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("Get CI history failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain history")
		return
	}

	common.Success(ctx, result)
}

// RevertCIVersion 回滚CI到指定版本
// @Summary 回滚CI到指定版本
// @Description 将CI回滚到历史中的某个版本
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param request body dto.RevertCIVersionRequest true "回滚版本请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id}/revert [post]
func (c *ProductionService) RevertCIVersion(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	var req dto.RevertCIVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	// 获取操作人信息
	operatorID := ctx.GetInt("user_id")
	operatorName := authenticatedOperatorName(ctx)

	result, err := c.ciHistoryService.RevertCIVersion(c.operatorContext(ctx), id, tenantID, operatorID, operatorName, &req)
	if err != nil {
		c.logger.Errorw("Revert CI version failed", "error", err, "ci_id", id, "version", req.Version, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Rollback version failed")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ Batch Operation 相关接口 ------------------------------

// BatchCreateCI 批量创建CI
// @Summary 批量创建CI
// @Description 批量创建多个CI，最多支持100个
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.BatchCreateCIRequest true "批量创建请求"
// @Success 200 {object} common.Response{data=dto.BatchOperationResponse}
// @Router /api/v1/cmdb/cis/batch [post]
func (c *ProductionService) BatchCreateCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.BatchCreateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.BatchCreateCI(c.operatorContext(ctx), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch create CI failed", "error", err, "count", len(req.Items), "tenant_id", tenantID)
		common.RespondError(ctx, err, "Batch creation failed")
		return
	}

	common.Success(ctx, result)
}

// BatchUpdateCI 批量更新CI
// @Summary 批量更新CI
// @Description 批量更新多个CI的相同属性，最多支持100个
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.BatchUpdateCIRequest true "批量更新请求"
// @Success 200 {object} common.Response{data=dto.BatchOperationResponse}
// @Router /api/v1/cmdb/cis/batch [put]
func (c *ProductionService) BatchUpdateCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.BatchUpdateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.BatchUpdateCI(c.operatorContext(ctx), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch update CI failed", "error", err, "count", len(req.IDs), "tenant_id", tenantID)
		common.RespondError(ctx, err, "Batch update failed")
		return
	}

	common.Success(ctx, result)
}

// BatchDeleteCI 批量删除CI
// @Summary 批量删除CI
// @Description 批量删除多个CI，最多支持100个
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.BatchDeleteCIRequest true "批量删除请求"
// @Success 200 {object} common.Response{data=dto.BatchOperationResponse}
// @Router /api/v1/cmdb/cis/batch [delete]
func (c *ProductionService) BatchDeleteCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.BatchDeleteCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.ciService.BatchDeleteCI(c.operatorContext(ctx), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch delete CI failed", "error", err, "count", len(req.IDs), "tenant_id", tenantID)
		common.RespondError(ctx, err, "Batch delete failed")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ 高级搜索接口 ------------------------------

// SearchCI 高级搜索CI
// @Summary 高级搜索CI
// @Description 多条件组合搜索CI，支持全文搜索、属性过滤、分页、排序
// @Summary [已废弃] CI高级搜索
// @Description 已废弃：合并至 GET /api/v1/cmdb/cis，search/sortBy/sortOrder/tagIds/dateFrom/dateTo/withRelations 全部支持。保留向后兼容至 v1.7 末。
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CISearchRequest true "搜索请求（已废弃）"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.CIResponse]}
// @Deprecated
// @Router /api/v1/cmdb/cis/search [post]
func (c *ProductionService) SearchCI(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.CISearchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	// 页长归一在 ListCIs 单点完成（本方法内部转 ListCIRequest）；这里再写一套
	// `>1000→20` 就成了第二个所有者，会把大批量请求静默截断。

	result, err := c.ciService.SearchCI(ctx.Request.Context(), tenantID, &req)
	if err != nil {
		c.logger.Errorw("Search CI failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Search failed")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ 保存视图接口 ------------------------------

// CreateSavedView 创建保存的搜索视图
// @Summary 创建保存的搜索视图
// @Description 保存搜索条件为视图，方便后续快速查询
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CreateCISavedViewRequest true "创建视图请求"
// @Success 200 {object} common.Response{data=dto.CISavedView}
// @Router /api/v1/cmdb/views [post]
func (c *ProductionService) CreateSavedView(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	userNameStr := authenticatedOperatorName(ctx)

	var req dto.CreateCISavedViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.savedViewService.CreateSavedView(ctx.Request.Context(), &req, tenantID, userID, userNameStr)
	if err != nil {
		c.logger.Errorw("Create saved view failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.RespondError(ctx, err, "Failed to create view")
		return
	}

	common.Success(ctx, result)
}

// ListSavedViews 获取保存的视图列表
// @Summary 获取保存的视图列表
// @Description 获取当前用户创建的和公开的搜索视图
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Param includePublic query bool false "是否包含公开视图，默认true"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.CISavedView]}
// @Router /api/v1/cmdb/views [get]
func (c *ProductionService) ListSavedViews(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	page, pageSize := cmdbPagination(ctx)
	// 查询参数按平台契约用 camelCase；include_public 只有 swagger 提到过，
	// 前端与测试都没有调用方。
	includePublic, _ := strconv.ParseBool(ctx.DefaultQuery("includePublic", "true"))

	result, err := c.savedViewService.ListSavedViews(ctx.Request.Context(), tenantID, userID, includePublic, page, pageSize)
	if err != nil {
		c.logger.Errorw("List saved views failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain view list")
		return
	}

	common.Success(ctx, result)
}

// GetSavedView 获取视图详情
// @Summary 获取视图详情
// @Description 根据ID获取保存的搜索视图
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "视图ID"
// @Success 200 {object} common.Response{data=dto.CISavedView}
// @Router /api/v1/cmdb/views/{id} [get]
func (c *ProductionService) GetSavedView(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	result, err := c.savedViewService.GetSavedView(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to get view")
		return
	}

	common.Success(ctx, result)
}

// UpdateSavedView 更新视图
// @Summary 更新视图
// @Description 更新保存的搜索视图
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "视图ID"
// @Param request body dto.UpdateCISavedViewRequest true "更新视图请求"
// @Success 200 {object} common.Response{data=dto.CISavedView}
// @Router /api/v1/cmdb/views/{id} [put]
func (c *ProductionService) UpdateSavedView(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	var req dto.UpdateCISavedViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.savedViewService.UpdateSavedView(ctx.Request.Context(), id, tenantID, userID, &req)
	if err != nil {
		c.logger.Errorw("Update saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update view")
		return
	}

	common.Success(ctx, result)
}

// DeleteSavedView 删除视图
// @Summary 删除视图
// @Description 删除保存的搜索视图
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "视图ID"
// @Success 200 {object} common.Response
// @Router /api/v1/cmdb/views/{id} [delete]
func (c *ProductionService) DeleteSavedView(ctx *gin.Context) {
	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "CI")
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	err := c.savedViewService.DeleteSavedView(ctx.Request.Context(), id, tenantID, userID)
	if err != nil {
		c.logger.Errorw("Delete saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to delete view")
		return
	}

	common.Success(ctx, nil)
}

// ------------------------------ 导入导出接口 ------------------------------

// CreateImportTask 创建CI导入任务
// @Summary 创建CI导入任务
// @Description 批量导入CI数据，支持Excel和CSV格式，异步执行
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.ImportCIRequest true "导入请求"
// @Success 200 {object} common.Response{data=dto.ImportCIResult}
// @Router /api/v1/cmdb/import [post]
func (c *ProductionService) CreateImportTask(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	userNameStr := authenticatedOperatorName(ctx)

	var req dto.ImportCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.importExportService.CreateImportTask(ctx.Request.Context(), &req, tenantID, userID, userNameStr)
	if err != nil {
		c.logger.Errorw("Create import task failed", "error", err, "tenant_id", tenantID, "file_url", req.FileURL)
		common.RespondError(ctx, err, "Failed to create import task")
		return
	}

	common.Success(ctx, result)
}

// GetImportTaskStatus 获取导入任务状态
// @Summary 获取导入任务状态
// @Description 查询导入任务的执行状态和结果
// @Tags CMDB
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} common.Response{data=dto.ImportCIResult}
// @Router /api/v1/cmdb/import/{task_id} [get]
func (c *ProductionService) GetImportTaskStatus(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		common.Fail(ctx, common.ParamErrorCode, "Task ID cannot be empty")
		return
	}

	result, err := c.importExportService.GetImportTaskStatus(ctx.Request.Context(), taskID, tenantID)
	if err != nil {
		c.logger.Errorw("Get import task status failed", "error", err, "task_id", taskID, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain task status")
		return
	}

	common.Success(ctx, result)
}

// ListImportTasks 获取导入任务列表
// @Summary 获取导入任务列表
// @Description 查询历史导入任务列表
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.ImportCIResult]}
// @Router /api/v1/cmdb/import [get]
func (c *ProductionService) ListImportTasks(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	page, pageSize := cmdbPagination(ctx)

	result, err := c.importExportService.ListImportTasks(ctx.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("List import tasks failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain import task list")
		return
	}

	common.Success(ctx, result)
}

// CreateExportTask 创建CI导出任务
// @Summary 创建CI导出任务
// @Description 批量导出CI数据，支持Excel和CSV格式，异步执行
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.ExportCIRequest true "导出请求"
// @Success 200 {object} common.Response{data=dto.ExportCIResult}
// @Router /api/v1/cmdb/export [post]
func (c *ProductionService) CreateExportTask(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	userNameStr := authenticatedOperatorName(ctx)

	var req dto.ExportCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	result, err := c.importExportService.CreateExportTask(ctx.Request.Context(), &req, tenantID, userID, userNameStr)
	if err != nil {
		c.logger.Errorw("Create export task failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to create export task")
		return
	}

	common.Success(ctx, result)
}

// DownloadExportTask 下载导出结果
// @Summary 下载CMDB导出文件
// @Description 按任务ID下载导出文件；校验租户、操作人归属与 expires_at
// @Tags CMDB
// @Produce octet-stream
// @Param task_id path string true "任务ID"
// @Success 200 {file} file
// @Router /api/v1/cmdb/export/{task_id}/download [get]
func (c *ProductionService) DownloadExportTask(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}
	userID, uidOK := middleware.UserIDOrUnauthorized(ctx)
	if !uidOK {
		return
	}
	taskID := ctx.Param("task_id")
	if taskID == "" {
		common.Fail(ctx, common.ParamErrorCode, "Task ID is required")
		return
	}

	filePath, fileName, err := c.importExportService.ResolveExportDownload(ctx.Request.Context(), taskID, tenantID, userID)
	if err != nil {
		c.logger.Warnw("CMDB export download rejected", "task_id", taskID,
			"tenant_id", tenantID, "user_id", userID, "error", err)
		// AppError 的 403/404/409 必须原样透传，不能兜底成 500。
		common.RespondError(ctx, err, "Failed to download export file")
		return
	}
	ctx.FileAttachment(filePath, fileName)
}

// GetExportTaskStatus 获取导出任务状态
// @Summary 获取导出任务状态
// @Description 查询导出任务的执行状态和结果
// @Tags CMDB
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} common.Response{data=dto.ExportCIResult}
// @Router /api/v1/cmdb/export/{task_id} [get]
func (c *ProductionService) GetExportTaskStatus(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		common.Fail(ctx, common.ParamErrorCode, "Task ID cannot be empty")
		return
	}

	result, err := c.importExportService.GetExportTaskStatus(ctx.Request.Context(), taskID, tenantID)
	if err != nil {
		c.logger.Errorw("Get export task status failed", "error", err, "task_id", taskID, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain task status")
		return
	}

	common.Success(ctx, result)
}

// ListExportTasks 获取导出任务列表
// @Summary 获取导出任务列表
// @Description 查询历史导出任务列表
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量（默认20，最大100；越界回落默认页长）"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.ExportCIResult]}
// @Router /api/v1/cmdb/export [get]
func (c *ProductionService) ListExportTasks(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	page, pageSize := cmdbPagination(ctx)

	result, err := c.importExportService.ListExportTasks(ctx.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("List export tasks failed", "error", err, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain export task list")
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ 生命周期管理接口 ------------------------------

// UpdateLifecycleStatus 更新CI生命周期状态
// @Summary 更新CI生命周期状态
// @Description 按状态机变更CI生命周期；目标 scrapped 为退役终态，必须携带受控退役理由，body 需携带当前 version 做乐观锁
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param request body dto.UpdateCILifecycleRequest true "生命周期变更请求"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id}/lifecycle [put]
func (c *ProductionService) UpdateLifecycleStatus(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	var req dto.UpdateCILifecycleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	operatorID := ctx.GetInt("user_id")
	operatorName := authenticatedOperatorName(ctx)

	params := service.LifecycleChangeParams{
		Status:           req.Status,
		Remark:           req.Remark,
		RetirementReason: req.RetirementReason,
		ExpectedVersion:  req.Version,
	}
	result, err := c.ciService.UpdateLifecycleStatus(c.operatorContext(ctx), id, tenantID, params, operatorID, operatorName)
	if err != nil {
		c.logger.Errorw("Update CI lifecycle status failed", "error", err, "ci_id", id, "status", req.Status, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to update lifecycle status")
		return
	}

	common.Success(ctx, result)
}

// BatchUpdateLifecycleStatus 批量更新CI生命周期状态
// @Summary 批量更新CI生命周期状态
// @Description 批量变更多个CI的生命周期状态（批量路径不做乐观锁，逐项返回结果）
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.BatchUpdateLifecycleRequest true "批量更新请求"
// @Success 200 {object} common.Response{data=dto.BatchOperationResponse}
// @Router /api/v1/cmdb/cis/batch/lifecycle [put]
func (c *ProductionService) BatchUpdateLifecycleStatus(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	var req dto.BatchUpdateLifecycleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Wrong request parameters")
		return
	}

	operatorID := ctx.GetInt("user_id")
	operatorName := authenticatedOperatorName(ctx)

	params := service.LifecycleChangeParams{
		Status:           req.Status,
		Remark:           req.Remark,
		RetirementReason: req.RetirementReason,
	}
	result, err := c.ciService.BatchUpdateLifecycleStatus(c.operatorContext(ctx), req.IDs, tenantID, params, operatorID, operatorName)
	if err != nil {
		c.logger.Errorw("Batch update CI lifecycle status failed", "error", err, "count", len(req.IDs), "status", req.Status, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Batch update failed")
		return
	}

	common.Success(ctx, result)
}

// GetLifecycleHistory 获取CI生命周期变更历史
// @Summary 获取CI生命周期变更历史
// @Description 查询CI的所有生命周期状态变更记录
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Success 200 {object} common.Response{data=[]map[string]interface{}}
// @Router /api/v1/cmdb/cis/{id}/lifecycle/history [get]
func (c *ProductionService) GetLifecycleHistory(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "Invalid CI ID parameter")
		return
	}

	result, err := c.ciService.GetLifecycleHistory(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI lifecycle history failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.RespondError(ctx, err, "Failed to obtain life cycle history")
		return
	}

	common.Success(ctx, result)
}
