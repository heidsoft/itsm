// Package vendor — 供应商管理 handler.
// 迁移自 controller/vendor_controller.go，保持原有 API 契约不变。
package vendor

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 VendorService 依赖.
type Handler struct {
	svc    *service.VendorService
	logger *zap.SugaredLogger
}

// NewHandler 构造 vendor Handler.
func NewHandler(svc *service.VendorService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{svc: svc, logger: logger}
}

// CreateVendor POST /api/v1/vendors
func (h *Handler) CreateVendor(ctx *gin.Context) {
	var req dto.CreateVendorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误")
		return
	}
	tv, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	tenantID, ok := tv.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息格式错误")
		return
	}
	res, err := h.svc.CreateVendor(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建供应商失败")
		return
	}
	common.Success(ctx, res)
}

// ListVendors GET /api/v1/vendors
func (h *Handler) ListVendors(ctx *gin.Context) {
	pagination := common.GetPaginationFromQuery(ctx)
	page, size := pagination.Page, pagination.PageSize
	tv, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	tenantID, ok := tv.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息格式错误")
		return
	}
	list, total, err := h.svc.ListVendors(ctx.Request.Context(), tenantID, page, size)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取供应商列表失败")
		return
	}
	common.Success(ctx, gin.H{"list": list, "total": total, "page": page})
}

// GetVendor GET /api/v1/vendors/:id
func (h *Handler) GetVendor(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "供应商ID格式错误")
		return
	}
	tv, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	tenantID, ok := tv.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息格式错误")
		return
	}
	res, err := h.svc.GetVendor(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, 404, "Vendor not found")
		return
	}
	common.Success(ctx, res)
}

// DeleteVendor DELETE /api/v1/vendors/:id
func (h *Handler) DeleteVendor(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(ctx, 1001, "供应商ID格式错误")
		return
	}
	tv, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	tenantID, ok := tv.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息格式错误")
		return
	}
	if err := h.svc.DeleteVendor(ctx.Request.Context(), id, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除供应商失败")
		return
	}
	common.Success(ctx, gin.H{"message": "deleted"})
}
