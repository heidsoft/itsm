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

// SystemConfigController 系统配置控制器
type SystemConfigController struct {
	configService *service.SystemConfigService
	logger        *zap.SugaredLogger
}

// NewSystemConfigController 创建系统配置控制器
func NewSystemConfigController(configService *service.SystemConfigService, logger *zap.SugaredLogger) *SystemConfigController {
	return &SystemConfigController{
		configService: configService,
		logger:        logger,
	}
}

// ListConfigs 获取配置列表
func (sc *SystemConfigController) ListConfigs(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")

	configs, total, err := sc.configService.ListSystemConfigs(c.Request.Context(), tenantID, category, page, pageSize)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "查询配置列表失败: "+err.Error())
		return
	}

	// 转换响应
	configResponses := make([]dto.SystemConfigResponse, len(configs))
	for i, cfg := range configs {
		configResponses[i] = dto.SystemConfigResponse{
			ID:          cfg.ID,
			Key:         cfg.Key,
			Value:       cfg.Value,
			ValueType:   cfg.ValueType,
			Category:    cfg.Category,
			Description: cfg.Description,
			CreatedBy:   cfg.CreatedBy,
			TenantID:    cfg.TenantID,
			CreatedAt:   cfg.CreatedAt,
			UpdatedAt:   cfg.UpdatedAt,
		}
	}

	common.Success(c, dto.SystemConfigListResponse{
		Configs: configResponses,
		Total:   total,
		Page:    page,
		Size:    pageSize,
	})
}

// GetConfig 获取单个配置
func (sc *SystemConfigController) GetConfig(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的配置ID")
		return
	}

	config, err := sc.configService.GetSystemConfig(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, err.Error())
		return
	}

	common.Success(c, dto.SystemConfigResponse{
		ID:          config.ID,
		Key:         config.Key,
		Value:       config.Value,
		ValueType:   config.ValueType,
		Category:    config.Category,
		Description: config.Description,
		CreatedBy:   config.CreatedBy,
		TenantID:    config.TenantID,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	})
}

// GetConfigByKey 根据key获取配置
func (sc *SystemConfigController) GetConfigByKey(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	key := c.Param("key")

	config, err := sc.configService.GetSystemConfigByKey(c.Request.Context(), key, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, err.Error())
		return
	}

	common.Success(c, dto.SystemConfigResponse{
		ID:          config.ID,
		Key:         config.Key,
		Value:       config.Value,
		ValueType:   config.ValueType,
		Category:    config.Category,
		Description: config.Description,
		CreatedBy:   config.CreatedBy,
		TenantID:    config.TenantID,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	})
}

// UpdateConfig 更新配置
func (sc *SystemConfigController) UpdateConfig(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的配置ID")
		return
	}

	var req dto.UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	config, err := sc.configService.UpdateSystemConfig(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "更新配置失败: "+err.Error())
		return
	}

	common.Success(c, dto.SystemConfigResponse{
		ID:          config.ID,
		Key:         config.Key,
		Value:       config.Value,
		ValueType:   config.ValueType,
		Category:    config.Category,
		Description: config.Description,
		CreatedBy:   config.CreatedBy,
		TenantID:    config.TenantID,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	})
}

// BatchUpdateConfigs 批量更新配置
func (sc *SystemConfigController) BatchUpdateConfigs(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var reqs []dto.UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	configs, err := sc.configService.BatchUpdateSystemConfigs(c.Request.Context(), reqs, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "批量更新配置失败: "+err.Error())
		return
	}

	// 转换响应
	configResponses := make([]dto.SystemConfigResponse, len(configs))
	for i, cfg := range configs {
		configResponses[i] = dto.SystemConfigResponse{
			ID:          cfg.ID,
			Key:         cfg.Key,
			Value:       cfg.Value,
			ValueType:   cfg.ValueType,
			Category:    cfg.Category,
			Description: cfg.Description,
			CreatedBy:   cfg.CreatedBy,
			TenantID:    cfg.TenantID,
			CreatedAt:   cfg.CreatedAt,
			UpdatedAt:   cfg.UpdatedAt,
		}
	}

	common.Success(c, configResponses)
}

// InitDefaultConfigs 初始化默认配置
func (sc *SystemConfigController) InitDefaultConfigs(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	err = sc.configService.InitDefaultConfigs(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "初始化默认配置失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "默认配置初始化成功"})
}
