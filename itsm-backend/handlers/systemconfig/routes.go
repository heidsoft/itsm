package systemconfig

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// tenantID 提取租户上下文；沿用旧 controller 的 middleware.GetTenantID + 401 语义
// （UnauthorizedCode + "未授权访问"，与 handlerctx.RequireTenantID 的 2001 等价，
// 但为保持行为契约原样保留旧实现）。
func tenantID(c *gin.Context) (int, bool) {
	tid, err := middleware.GetTenantID(c)
	if err != nil || tid == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	return tid, true
}

// ListConfigs 获取配置列表
func (h *Handler) ListConfigs(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")

	configs, total, err := h.configService.ListSystemConfigs(c.Request.Context(), tid, category, page, pageSize)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

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
func (h *Handler) GetConfig(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的配置ID")
		return
	}

	config, err := h.configService.GetSystemConfig(c.Request.Context(), id, tid)
	if err != nil {
		common.NotFoundWithErr(c, err, "resource not found")
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
func (h *Handler) GetConfigByKey(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	key := c.Param("key")

	config, err := h.configService.GetSystemConfigByKey(c.Request.Context(), key, tid)
	if err != nil {
		common.NotFoundWithErr(c, err, "resource not found")
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
func (h *Handler) UpdateConfig(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
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

	config, err := h.configService.UpdateSystemConfig(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
func (h *Handler) BatchUpdateConfigs(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	var reqs []dto.UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	configs, err := h.configService.BatchUpdateSystemConfigs(c.Request.Context(), reqs, tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

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
func (h *Handler) InitDefaultConfigs(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	err := h.configService.InitDefaultConfigs(c.Request.Context(), tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "默认配置初始化成功"})
}
