package controller

import (
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
}

// NewCMDBController 创建CMDB控制器
func NewCMDBController(
	logger *zap.SugaredLogger,
	ciTypeService *service.CITypeService,
	ciAttributeDefinitionService *service.CIAttributeDefinitionService,
	ciService *service.ConfigurationItemService,
	ciRelationshipService *service.CIRelationshipService,
) *CMDBController {
	return &CMDBController{
		logger:                       logger,
		ciTypeService:                ciTypeService,
		ciAttributeDefinitionService: ciAttributeDefinitionService,
		ciService:                    ciService,
		ciRelationshipService:        ciRelationshipService,
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
// @Param ciTypeId path int true "CI类型ID"
// @Success 200 {object} common.Response{data=[]dto.CIAttributeDefinitionResponse}
// @Router /api/v1/cmdb/ci-types/{ciTypeId}/attributes [get]
func (c *CMDBController) ListCIAttributeDefinitions(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciTypeIDStr := ctx.Param("ciTypeId")
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
// @Param size query int false "每页数量"
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
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
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
// @Param ciId path int true "配置项ID"
// @Param direction query string false "关系方向（outgoing/incoming/不传为所有）"
// @Success 200 {object} common.Response{data=[]dto.CIRelationshipResponse}
// @Router /api/v1/cmdb/cis/{ciId}/relationships [get]
func (c *CMDBController) ListCIRelationshipsByCIID(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciIDStr := ctx.Param("ciId")
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
// @Param ciId path int true "配置项ID"
// @Param maxDepth query int false "最大分析深度，默认3"
// @Success 200 {object} common.Response{data=dto.CIImpactAnalysisResponse}
// @Router /api/v1/cmdb/cis/{ciId}/impact-analysis [get]
func (c *CMDBController) GetCIImpactAnalysis(ctx *gin.Context) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}

	ciIDStr := ctx.Param("ciId")
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
