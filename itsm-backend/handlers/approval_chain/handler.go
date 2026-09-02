// Package approval_chain 是审批链域的 HTTP handler 层（域切片架构）。
// 自 controller/approval_chain_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.ApprovalChainService 承载，本包只做参数解析与响应封装。
package approval_chain

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 审批链 HTTP handler
type Handler struct {
	chainService *service.ApprovalChainService
	logger       *zap.SugaredLogger
}

// NewHandler 创建审批链 handler 实例
func NewHandler(chainService *service.ApprovalChainService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		chainService: chainService,
		logger:       logger,
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	tid, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	id, ok := tid.(int)
	if !ok || id == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	return id, true
}

// pathID 提取路径参数 ID
func pathID(c *gin.Context, param string) (int, bool) {
	id, err := strconv.Atoi(c.Param(param))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的ID参数")
		return 0, false
	}
	return id, true
}

// ListChains 获取审批链列表
func (h *Handler) ListChains(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	entityType := c.Query("entity_type")
	status := c.Query("status")

	chains, total, err := h.chainService.ListApprovalChains(c.Request.Context(), tid, entityType, status, page, pageSize)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	chainResponses := dto.ToApprovalChainResponseList(chains)
	common.Success(c, dto.ApprovalChainListResponse{
		Data:  chainResponses,
		Total: total,
		Page:  page,
		Size:  pageSize,
	})
}

// GetChain 获取审批链详情
func (h *Handler) GetChain(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	chain, err := h.chainService.GetApprovalChain(c.Request.Context(), id, tid)
	if err != nil {
		common.NotFoundWithErr(c, err, "resource not found")
		return
	}

	common.Success(c, dto.ToApprovalChainResponse(chain))
}

// CreateChain 创建审批链
func (h *Handler) CreateChain(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	var req dto.ApprovalChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	req.TenantID = tid

	entity, err := h.chainService.CreateApprovalChain(c.Request.Context(), &req, tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToApprovalChainResponse(entity))
}

// UpdateChain 更新审批链
func (h *Handler) UpdateChain(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.ApprovalChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	entity, err := h.chainService.UpdateApprovalChain(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToApprovalChainResponse(entity))
}

// DeleteChain 删除审批链
func (h *Handler) DeleteChain(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	id, ok := pathID(c, "id")
	if !ok {
		return
	}

	err := h.chainService.DeleteApprovalChain(c.Request.Context(), id, tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetStats 获取审批链统计
func (h *Handler) GetStats(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	stats, err := h.chainService.GetApprovalChainStats(c.Request.Context(), tid)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}
