package controller

import (
	"context"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CMDBController CMDB控制器
type CMDBController struct {
	logger                       *zap.SugaredLogger
	ciTypeService                *service.CITypeService
	ciAttributeDefinitionService *service.CIAttributeDefinitionService
	ciService                    *service.ConfigurationItemService
	ciRelationshipService        *service.CIRelationshipService
	ciHistoryService             *service.CIHistoryService
	ciTagService                 *service.CITagService
	importExportService          *service.CMDBImportExportService
	savedViewService             *service.CMDBSavedViewService
}

// NewCMDBController 创建CMDB控制器
func NewCMDBController(
	logger *zap.SugaredLogger,
	ciTypeService *service.CITypeService,
	ciAttributeDefinitionService *service.CIAttributeDefinitionService,
	ciService *service.ConfigurationItemService,
	ciRelationshipService *service.CIRelationshipService,
	ciHistoryService *service.CIHistoryService,
	ciTagService *service.CITagService,
	importExportService *service.CMDBImportExportService,
	savedViewService *service.CMDBSavedViewService,
) *CMDBController {
	return &CMDBController{
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
// @Param size query int false "每页数量"
// @Param search query string false "搜索关键词（名称）"
// @Success 200 {object} common.Response{data=dto.CITypeListResponse}
// @Router /api/v1/cmdb/ci-types [get]
func (c *CMDBController) ListCITypes(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	search := ctx.Query("search")

	result, err := c.ciTypeService.ListCITypes(ctx.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		c.logger.Errorw("List CI types failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI类型列表失败: "+err.Error())
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
func (c *CMDBController) GetCIType(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	result, err := c.ciTypeService.GetCITypeByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI类型失败: "+err.Error())
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI类型不存在")
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
func (c *CMDBController) CreateCIType(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciTypeService.CreateCIType(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI type failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.Fail(ctx, common.InternalErrorCode, "创建CI类型失败: "+err.Error())
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
func (c *CMDBController) UpdateCIType(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	var req dto.UpdateCITypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciTypeService.UpdateCIType(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新CI类型失败: "+err.Error())
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
func (c *CMDBController) DeleteCIType(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	err = c.ciTypeService.DeleteCIType(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI type failed", "error", err, "ci_type_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除CI类型失败: "+err.Error())
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
func (c *CMDBController) ListCIAttributeDefinitions(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciTypeIDStr := ctx.Param("id")
	ciTypeID, err := strconv.Atoi(ciTypeIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI类型ID参数")
		return
	}

	result, err := c.ciAttributeDefinitionService.ListCIAttributeDefinitionsByCITypeID(ctx.Request.Context(), ciTypeID, tenantID)
	if err != nil {
		c.logger.Errorw("List CI attribute definitions failed", "error", err, "ci_type_id", ciTypeID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI属性定义列表失败: "+err.Error())
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
func (c *CMDBController) GetCIAttributeDefinition(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	result, err := c.ciAttributeDefinitionService.GetCIAttributeDefinitionByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI属性定义失败: "+err.Error())
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI属性定义不存在")
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
func (c *CMDBController) CreateCIAttributeDefinition(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateCIAttributeDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciAttributeDefinitionService.CreateCIAttributeDefinition(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI attribute definition failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.Fail(ctx, common.InternalErrorCode, "创建CI属性定义失败: "+err.Error())
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
func (c *CMDBController) UpdateCIAttributeDefinition(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	var req dto.UpdateCIAttributeDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciAttributeDefinitionService.UpdateCIAttributeDefinition(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新CI属性定义失败: "+err.Error())
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
func (c *CMDBController) DeleteCIAttributeDefinition(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	err = c.ciAttributeDefinitionService.DeleteCIAttributeDefinition(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI attribute definition failed", "error", err, "attr_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除CI属性定义失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// ------------------------------ 配置项相关接口 ------------------------------

// ListCIs 获取配置项列表
// @Summary 获取配置项列表
// @Description 获取所有配置项的列表，支持分页和筛选
// @Tags CMDB
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param ciTypeId query int false "CI类型ID"
// @Param status query string false "状态"
// @Param environment query string false "环境"
// @Param criticality query string false "重要性"
// @Param cloudProvider query string false "云厂商"
// @Param cloudAccountId query string false "云账号ID"
// @Param cloudRegion query string false "云区域"
// @Param assignedTo query string false "负责人"
// @Param ownedBy query string false "拥有者"
// @Param search query string false "搜索关键词（名称、资产标签、序列号等）"
// @Success 200 {object} common.Response{data=dto.CIListResponse}
// @Router /api/v1/cmdb/cis [get]
func (c *CMDBController) ListCIs(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.ListCIRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciService.ListCIs(ctx.Request.Context(), tenantID, &req)
	if err != nil {
		c.logger.Errorw("List CIs failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取配置项列表失败: "+err.Error())
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
func (c *CMDBController) GetCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	withRelations, _ := strconv.ParseBool(ctx.DefaultQuery("withRelations", "false"))

	result, err := c.ciService.GetCIByID(ctx.Request.Context(), id, tenantID, withRelations)
	if err != nil {
		c.logger.Errorw("Get CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取配置项失败: "+err.Error())
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "配置项不存在")
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
func (c *CMDBController) CreateCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	if req.CITypeID == 0 {
		req.CITypeID = req.CITypeIDSnake
	}
	if req.CITypeID == 0 {
		common.Fail(ctx, common.ParamErrorCode, "CI类型ID不能为空")
		return
	}

	// 自动设置租户ID
	req.TenantID = tenantID

	result, err := c.ciService.CreateCI(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.Fail(ctx, common.InternalErrorCode, "创建配置项失败: "+err.Error())
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
func (c *CMDBController) UpdateCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	var req dto.UpdateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciService.UpdateCI(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新配置项失败: "+err.Error())
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
func (c *CMDBController) DeleteCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	err = c.ciService.DeleteCI(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除配置项失败: "+err.Error())
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
func (c *CMDBController) GetCIStats(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	result, err := c.ciService.GetCIStats(ctx.Request.Context(), tenantID)
	if err != nil {
		c.logger.Errorw("Get CI stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取配置项统计失败: "+err.Error())
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
// @Param page_size query int false "每页数量"
// @Param relationshipType query string false "关系类型"
// @Success 200 {object} common.Response{data=dto.CIRelationshipListResponse}
// @Router /api/v1/cmdb/relationships [get]
func (c *CMDBController) ListCIRelationships(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSizeQuery := ctx.Query("page_size")
	if pageSizeQuery == "" {
		pageSizeQuery = ctx.Query("pageSize")
	}
	if pageSizeQuery == "" {
		pageSizeQuery = ctx.DefaultQuery("size", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeQuery)
	relationshipType := ctx.Query("relationshipType")

	result, err := c.ciRelationshipService.ListAllCIRelationships(ctx.Request.Context(), tenantID, page, pageSize, relationshipType)
	if err != nil {
		c.logger.Errorw("List CI relationships failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI关系列表失败: "+err.Error())
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
func (c *CMDBController) GetCIRelationship(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	result, err := c.ciRelationshipService.GetCIRelationshipByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI关系失败: "+err.Error())
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "CI关系不存在")
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
func (c *CMDBController) ListCIRelationshipsByCIID(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciIDStr := ctx.Param("id")
	ciID, err := strconv.Atoi(ciIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	direction := ctx.DefaultQuery("direction", "")

	result, err := c.ciRelationshipService.ListCIRelationshipsByCIID(ctx.Request.Context(), ciID, tenantID, direction)
	if err != nil {
		c.logger.Errorw("List CI relationships by CI ID failed", "error", err, "ci_id", ciID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI关系列表失败: "+err.Error())
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
func (c *CMDBController) CreateCIRelationship(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciRelationshipService.CreateCIRelationship(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI relationship failed", "error", err, "tenant_id", tenantID, "source_ci_id", req.SourceCIID, "target_ci_id", req.TargetCIID)
		common.Fail(ctx, common.InternalErrorCode, "创建CI关系失败: "+err.Error())
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
func (c *CMDBController) UpdateCIRelationship(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	var req dto.UpdateCIRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciRelationshipService.UpdateCIRelationship(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新CI关系失败: "+err.Error())
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
func (c *CMDBController) DeleteCIRelationship(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	err = c.ciRelationshipService.DeleteCIRelationship(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI relationship failed", "error", err, "relation_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除CI关系失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// ListRelationshipTypes 获取内置CI关系类型
// @Summary 获取内置CI关系类型
// @Description 获取CMDB支持的内置关系类型枚举
// @Tags CMDB
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=dto.GetRelationshipTypesResponse}
// @Router /api/v1/cmdb/relationship-types [get]
func (c *CMDBController) ListRelationshipTypes(ctx *gin.Context) {
	common.Success(ctx, dto.GetRelationshipTypesResponse{
		Types: []dto.RelationshipTypeInfo{
			{Type: dto.DependsOn, Name: "依赖", Description: "源CI依赖目标CI", Direction: "uni-directional", Icon: "link"},
			{Type: dto.Hosts, Name: "托管", Description: "源CI托管目标CI", Direction: "uni-directional", Icon: "server"},
			{Type: dto.HostedOn, Name: "承载于", Description: "源CI运行或部署在目标CI上", Direction: "uni-directional", Icon: "hard-drive"},
			{Type: dto.ConnectsTo, Name: "连接到", Description: "源CI连接目标CI", Direction: "bi-directional", Icon: "network"},
			{Type: dto.RunsOn, Name: "运行于", Description: "源CI运行在目标CI上", Direction: "uni-directional", Icon: "play"},
			{Type: dto.Contains, Name: "包含", Description: "源CI包含目标CI", Direction: "uni-directional", Icon: "box"},
			{Type: dto.PartOf, Name: "组成部分", Description: "源CI是目标CI的一部分", Direction: "uni-directional", Icon: "component"},
			{Type: dto.Impacts, Name: "影响", Description: "源CI故障会影响目标CI", Direction: "uni-directional", Icon: "activity"},
			{Type: dto.Owns, Name: "拥有", Description: "源CI拥有目标CI", Direction: "uni-directional", Icon: "key"},
			{Type: dto.Uses, Name: "使用", Description: "源CI使用目标CI能力", Direction: "uni-directional", Icon: "plug"},
		},
	})
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
func (c *CMDBController) GetCIImpactAnalysis(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciIDStr := ctx.Param("id")
	ciID, err := strconv.Atoi(ciIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	maxDepth, _ := strconv.Atoi(ctx.DefaultQuery("maxDepth", "3"))
	if maxDepth <= 0 {
		maxDepth = 3
	}

	result, err := c.ciRelationshipService.GetCIImpactAnalysis(ctx.Request.Context(), ciID, tenantID, maxDepth)
	if err != nil {
		c.logger.Errorw("Get CI impact analysis failed", "error", err, "ci_id", ciID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取CI影响分析失败: "+err.Error())
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
// @Param size query int false "每页数量"
// @Param search query string false "搜索关键词（标签名/值）"
// @Success 200 {object} common.Response{data=dto.CITagListResponse}
// @Router /api/v1/cmdb/tags [get]
func (c *CMDBController) ListCITags(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	search := ctx.Query("search")

	result, err := c.ciTagService.ListCITags(ctx.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		c.logger.Errorw("List CI tags failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取标签列表失败: "+err.Error())
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
func (c *CMDBController) GetCITag(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	result, err := c.ciTagService.GetCITagByID(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取标签失败: "+err.Error())
		return
	}

	if result == nil {
		common.Fail(ctx, common.NotFoundCode, "标签不存在")
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
func (c *CMDBController) CreateCITag(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateCITagRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciTagService.CreateCITag(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Create CI tag failed", "error", err, "tenant_id", tenantID, "key", req.Key)
		common.Fail(ctx, common.InternalErrorCode, "创建标签失败: "+err.Error())
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
func (c *CMDBController) UpdateCITag(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	var req dto.UpdateCITagRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciTagService.UpdateCITag(ctx.Request.Context(), id, tenantID, &req)
	if err != nil {
		c.logger.Errorw("Update CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新标签失败: "+err.Error())
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
func (c *CMDBController) DeleteCITag(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	err = c.ciTagService.DeleteCITag(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Delete CI tag failed", "error", err, "tag_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除标签失败: "+err.Error())
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
func (c *CMDBController) AddTagsToCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	var req dto.AddCITagsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciService.AddTagsToCI(ctx.Request.Context(), id, tenantID, req.TagIDs)
	if err != nil {
		c.logger.Errorw("Add tags to CI failed", "error", err, "ci_id", id, "tag_ids", req.TagIDs, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "添加标签失败: "+err.Error())
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
func (c *CMDBController) RemoveTagsFromCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	var req dto.RemoveCITagsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.ciService.RemoveTagsFromCI(ctx.Request.Context(), id, tenantID, req.TagIDs)
	if err != nil {
		c.logger.Errorw("Remove tags from CI failed", "error", err, "ci_id", id, "tag_ids", req.TagIDs, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "移除标签失败: "+err.Error())
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
// @Param size query int false "每页数量"
// @Success 200 {object} common.Response{data=dto.CIHistoryListResponse}
// @Router /api/v1/cmdb/cis/{id}/history [get]
func (c *CMDBController) GetCIHistory(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))

	result, err := c.ciHistoryService.GetCIHistory(ctx.Request.Context(), id, tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("Get CI history failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取历史记录失败: "+err.Error())
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
func (c *CMDBController) RevertCIVersion(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	var req dto.RevertCIVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取操作人信息
	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")
	ctxWithUser := context.WithValue(ctx.Request.Context(), "user_id", userID)
	ctxWithUser = context.WithValue(ctxWithUser, "user_name", userName)

	result, err := c.ciHistoryService.RevertCIVersion(ctxWithUser, id, tenantID, userID.(int), userName.(string), &req)
	if err != nil {
		c.logger.Errorw("Revert CI version failed", "error", err, "ci_id", id, "version", req.Version, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "回滚版本失败: "+err.Error())
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
func (c *CMDBController) BatchCreateCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.BatchCreateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取操作人信息
	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")
	ctxWithUser := context.WithValue(ctx.Request.Context(), "user_id", userID)
	ctxWithUser = context.WithValue(ctxWithUser, "user_name", userName)

	result, err := c.ciService.BatchCreateCI(ctxWithUser, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch create CI failed", "error", err, "count", len(req.Items), "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "批量创建失败: "+err.Error())
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
func (c *CMDBController) BatchUpdateCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.BatchUpdateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取操作人信息
	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")
	ctxWithUser := context.WithValue(ctx.Request.Context(), "user_id", userID)
	ctxWithUser = context.WithValue(ctxWithUser, "user_name", userName)

	result, err := c.ciService.BatchUpdateCI(ctxWithUser, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch update CI failed", "error", err, "count", len(req.IDs), "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "批量更新失败: "+err.Error())
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
func (c *CMDBController) BatchDeleteCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.BatchDeleteCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 获取操作人信息
	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")
	ctxWithUser := context.WithValue(ctx.Request.Context(), "user_id", userID)
	ctxWithUser = context.WithValue(ctxWithUser, "user_name", userName)

	result, err := c.ciService.BatchDeleteCI(ctxWithUser, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Batch delete CI failed", "error", err, "count", len(req.IDs), "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "批量删除失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ 高级搜索接口 ------------------------------

// SearchCI 高级搜索CI
// @Summary 高级搜索CI
// @Description 多条件组合搜索CI，支持全文搜索、属性过滤、分页、排序
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.CISearchRequest true "搜索请求"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.CIResponse]}
// @Router /api/v1/cmdb/cis/search [post]
func (c *CMDBController) SearchCI(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CISearchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 分页参数默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 20
	}

	result, err := c.ciService.SearchCI(ctx.Request.Context(), tenantID, &req)
	if err != nil {
		c.logger.Errorw("Search CI failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "搜索失败: "+err.Error())
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
func (c *CMDBController) CreateSavedView(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")

	var req dto.CreateCISavedViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.savedViewService.CreateSavedView(ctx.Request.Context(), &req, tenantID, userID.(int), userName.(string))
	if err != nil {
		c.logger.Errorw("Create saved view failed", "error", err, "tenant_id", tenantID, "name", req.Name)
		common.Fail(ctx, common.InternalErrorCode, "创建视图失败: "+err.Error())
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
// @Param size query int false "每页数量"
// @Param include_public query bool false "是否包含公开视图，默认true"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.CISavedView]}
// @Router /api/v1/cmdb/views [get]
func (c *CMDBController) ListSavedViews(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, _ := ctx.Get("user_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	includePublic, _ := strconv.ParseBool(ctx.DefaultQuery("include_public", "true"))

	result, err := c.savedViewService.ListSavedViews(ctx.Request.Context(), tenantID, userID.(int), includePublic, page, pageSize)
	if err != nil {
		c.logger.Errorw("List saved views failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取视图列表失败: "+err.Error())
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
func (c *CMDBController) GetSavedView(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	result, err := c.savedViewService.GetSavedView(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取视图失败: "+err.Error())
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
func (c *CMDBController) UpdateSavedView(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	userID, _ := ctx.Get("user_id")
	var req dto.UpdateCISavedViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.savedViewService.UpdateSavedView(ctx.Request.Context(), id, tenantID, userID.(int), &req)
	if err != nil {
		c.logger.Errorw("Update saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新视图失败: "+err.Error())
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
func (c *CMDBController) DeleteSavedView(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的ID参数")
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.savedViewService.DeleteSavedView(ctx.Request.Context(), id, tenantID, userID.(int))
	if err != nil {
		c.logger.Errorw("Delete saved view failed", "error", err, "view_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "删除视图失败: "+err.Error())
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
func (c *CMDBController) CreateImportTask(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")

	var req dto.ImportCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.importExportService.CreateImportTask(ctx.Request.Context(), &req, tenantID, userID.(int), userName.(string))
	if err != nil {
		c.logger.Errorw("Create import task failed", "error", err, "tenant_id", tenantID, "file_url", req.FileURL)
		common.Fail(ctx, common.InternalErrorCode, "创建导入任务失败: "+err.Error())
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
func (c *CMDBController) GetImportTaskStatus(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		common.Fail(ctx, common.ParamErrorCode, "任务ID不能为空")
		return
	}

	result, err := c.importExportService.GetImportTaskStatus(ctx.Request.Context(), taskID, tenantID)
	if err != nil {
		c.logger.Errorw("Get import task status failed", "error", err, "task_id", taskID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取任务状态失败: "+err.Error())
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
// @Param size query int false "每页数量"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.ImportCIResult]}
// @Router /api/v1/cmdb/import [get]
func (c *CMDBController) ListImportTasks(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))

	result, err := c.importExportService.ListImportTasks(ctx.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("List import tasks failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取导入任务列表失败: "+err.Error())
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
func (c *CMDBController) CreateExportTask(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")

	var req dto.ExportCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	result, err := c.importExportService.CreateExportTask(ctx.Request.Context(), &req, tenantID, userID.(int), userName.(string))
	if err != nil {
		c.logger.Errorw("Create export task failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建导出任务失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
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
func (c *CMDBController) GetExportTaskStatus(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		common.Fail(ctx, common.ParamErrorCode, "任务ID不能为空")
		return
	}

	result, err := c.importExportService.GetExportTaskStatus(ctx.Request.Context(), taskID, tenantID)
	if err != nil {
		c.logger.Errorw("Get export task status failed", "error", err, "task_id", taskID, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取任务状态失败: "+err.Error())
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
// @Param size query int false "每页数量"
// @Success 200 {object} common.Response{data=dto.ListResponse[dto.ExportCIResult]}
// @Router /api/v1/cmdb/export [get]
func (c *CMDBController) ListExportTasks(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))

	result, err := c.importExportService.ListExportTasks(ctx.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.logger.Errorw("List export tasks failed", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取导出任务列表失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}

// ------------------------------ 生命周期管理接口 ------------------------------

// UpdateLifecycleStatus 更新CI生命周期状态
// @Summary 更新CI生命周期状态
// @Description 变更CI的生命周期状态，记录变更历史
// @Tags CMDB
// @Accept json
// @Produce json
// @Param id path int true "CI ID"
// @Param status query string true "目标状态: draft/online/maintenance/offline/scrapped"
// @Param remark query string false "变更备注"
// @Success 200 {object} common.Response{data=dto.CIResponse}
// @Router /api/v1/cmdb/cis/{id}/lifecycle [put]
func (c *CMDBController) UpdateLifecycleStatus(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	status := ctx.Query("status")
	if status == "" {
		common.Fail(ctx, common.ParamErrorCode, "状态参数不能为空")
		return
	}

	remark := ctx.Query("remark")
	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")

	result, err := c.ciService.UpdateLifecycleStatus(ctx.Request.Context(), id, tenantID, status, remark, userID.(int), userName.(string))
	if err != nil {
		c.logger.Errorw("Update CI lifecycle status failed", "error", err, "ci_id", id, "status", status, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "更新生命周期状态失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}

// BatchUpdateLifecycleStatus 批量更新CI生命周期状态
// @Summary 批量更新CI生命周期状态
// @Description 批量变更多个CI的生命周期状态
// @Tags CMDB
// @Accept json
// @Produce json
// @Param request body dto.BatchUpdateLifecycleRequest true "批量更新请求"
// @Success 200 {object} common.Response{data=dto.BatchOperationResponse}
// @Router /api/v1/cmdb/cis/batch/lifecycle [put]
func (c *CMDBController) BatchUpdateLifecycleStatus(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req struct {
		IDs    []int  `json:"ids" binding:"required,min=1"`
		Status string `json:"status" binding:"required"`
		Remark string `json:"remark,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID, _ := ctx.Get("user_id")
	userName, _ := ctx.Get("user_name")

	result, err := c.ciService.BatchUpdateLifecycleStatus(ctx.Request.Context(), req.IDs, tenantID, req.Status, req.Remark, userID.(int), userName.(string))
	if err != nil {
		c.logger.Errorw("Batch update CI lifecycle status failed", "error", err, "count", len(req.IDs), "status", req.Status, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "批量更新失败: "+err.Error())
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
func (c *CMDBController) GetLifecycleHistory(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的CI ID参数")
		return
	}

	result, err := c.ciService.GetLifecycleHistory(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Get CI lifecycle history failed", "error", err, "ci_id", id, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "获取生命周期历史失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}
