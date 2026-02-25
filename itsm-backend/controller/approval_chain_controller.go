package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent/schema"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ApprovalChainController 审批链控制器
type ApprovalChainController struct {
	chainService *service.ApprovalChainService
	logger       *zap.SugaredLogger
}

// NewApprovalChainController 创建审批链控制器
func NewApprovalChainController(chainService *service.ApprovalChainService, logger *zap.SugaredLogger) *ApprovalChainController {
	return &ApprovalChainController{
		chainService: chainService,
		logger:       logger,
	}
}

// convertChainSteps 将 schema.ApprovalChainStep 转换为 DTO
func convertChainSteps(steps []schema.ApprovalChainStep) []dto.ApprovalChainStepDTO {
	result := make([]dto.ApprovalChainStepDTO, len(steps))
	for i, step := range steps {
		result[i] = dto.ApprovalChainStepDTO{
			Level:      step.Level,
			ApproverID: step.ApproverID,
			Role:       step.Role,
			Name:       step.Name,
			IsRequired: step.IsRequired,
		}
	}
	return result
}

// ListChains 获取审批链列表
func (ac *ApprovalChainController) ListChains(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	entityType := c.Query("entity_type")
	status := c.Query("status")

	chains, total, err := ac.chainService.ListApprovalChains(c.Request.Context(), tenantID, entityType, status, page, pageSize)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "查询审批链列表失败: "+err.Error())
		return
	}

	// 转换响应
	chainResponses := make([]dto.ApprovalChainResponse, len(chains))
	for i, chain := range chains {
		chainResponses[i] = dto.ApprovalChainResponse{
			ID:          chain.ID,
			Name:        chain.Name,
			Description: chain.Description,
			EntityType:  chain.EntityType,
			Chain:       convertChainSteps(chain.Chain),
			Status:      chain.Status,
			CreatedBy:   chain.CreatedBy,
			TenantID:    chain.TenantID,
			CreatedAt:   chain.CreatedAt,
			UpdatedAt:   chain.UpdatedAt,
		}
	}

	common.Success(c, dto.ApprovalChainListResponse{
		Data:  chainResponses,
		Total: total,
		Page:  page,
		Size:  pageSize,
	})
}

// GetChain 获取审批链详情
func (ac *ApprovalChainController) GetChain(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的审批链ID")
		return
	}

	chain, err := ac.chainService.GetApprovalChain(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, err.Error())
		return
	}

	common.Success(c, dto.ApprovalChainResponse{
		ID:          chain.ID,
		Name:        chain.Name,
		Description: chain.Description,
		EntityType:  chain.EntityType,
		Chain:       convertChainSteps(chain.Chain),
		Status:      chain.Status,
		CreatedBy:   chain.CreatedBy,
		TenantID:    chain.TenantID,
		CreatedAt:   chain.CreatedAt,
		UpdatedAt:   chain.UpdatedAt,
	})
}

// CreateChain 创建审批链
func (ac *ApprovalChainController) CreateChain(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.ApprovalChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	req.TenantID = tenantID

	entity, err := ac.chainService.CreateApprovalChain(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "创建审批链失败: "+err.Error())
		return
	}

	common.Success(c, dto.ApprovalChainResponse{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		EntityType:  entity.EntityType,
		Chain:       convertChainSteps(entity.Chain),
		Status:      entity.Status,
		CreatedBy:   entity.CreatedBy,
		TenantID:    entity.TenantID,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	})
}

// UpdateChain 更新审批链
func (ac *ApprovalChainController) UpdateChain(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的审批链ID")
		return
	}

	var req dto.ApprovalChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	entity, err := ac.chainService.UpdateApprovalChain(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "更新审批链失败: "+err.Error())
		return
	}

	common.Success(c, dto.ApprovalChainResponse{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		EntityType:  entity.EntityType,
		Chain:       convertChainSteps(entity.Chain),
		Status:      entity.Status,
		CreatedBy:   entity.CreatedBy,
		TenantID:    entity.TenantID,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	})
}

// DeleteChain 删除审批链
func (ac *ApprovalChainController) DeleteChain(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的审批链ID")
		return
	}

	err = ac.chainService.DeleteApprovalChain(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "删除审批链失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetStats 获取审批链统计
func (ac *ApprovalChainController) GetStats(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := ac.chainService.GetApprovalChainStats(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取统计数据失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}
