// Package vendor — 供应商管理 handler.
// 迁移自 controller/vendor_controller.go，保持原有 API 契约不变。
package vendor

import (
	"errors"
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

// tenantID 从认证上下文取租户，缺失或类型不符一律 fail closed.
func tenantID(ctx *gin.Context) (int, bool) {
	tv, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息缺失")
		return 0, false
	}
	tid, ok := tv.(int)
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "租户信息格式错误")
		return 0, false
	}
	return tid, true
}

// pathVendorID 解析路径中的供应商 ID，非数字或非正整数都是参数错误.
func pathVendorID(ctx *gin.Context) (int, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(ctx, common.ParamErrorCode, "供应商ID格式错误")
		return 0, false
	}
	return id, true
}

// failVendorError 把服务层失败映射为稳定语义。改动前 GetVendor 把所有错误（包括数据
// 库故障）回成 code 404，而 404 不在 common.Fail 的 status 映射表里，结果是 HTTP 200
// + 业务码 404；编码冲突和删除不存在的记录也统一回 500。原始错误只进日志。
func (h *Handler) failVendorError(ctx *gin.Context, rawErr error, opMsg string, vendorID, tid int) {
	switch {
	case errors.Is(rawErr, service.ErrVendorCodeExists):
		h.logger.Warnw("Vendor code conflict", "error", rawErr, "tenant_id", tid)
		common.Conflict(ctx, "供应商编码已存在", nil)
	case errors.Is(rawErr, service.ErrVendorNotFound):
		// 查询带租户谓词，跨租户与真实不存在同样表现为 not found。
		h.logger.Warnw("Vendor not found", "error", rawErr, "vendor_id", vendorID, "tenant_id", tid)
		common.Fail(ctx, common.NotFoundCode, "供应商不存在或无权访问")
	default:
		h.logger.Errorw("Vendor operation failed",
			"error", rawErr, "operation", opMsg, "vendor_id", vendorID, "tenant_id", tid)
		common.Fail(ctx, common.InternalErrorCode, opMsg)
	}
}

// CreateVendor POST /api/v1/vendors
func (h *Handler) CreateVendor(ctx *gin.Context) {
	var req dto.CreateVendorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误")
		return
	}
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	res, err := h.svc.CreateVendor(ctx.Request.Context(), &req, tid)
	if err != nil {
		h.failVendorError(ctx, err, "创建供应商失败", 0, tid)
		return
	}
	common.Success(ctx, res)
}

// ListVendors GET /api/v1/vendors
func (h *Handler) ListVendors(ctx *gin.Context) {
	pagination := common.GetPaginationFromQuery(ctx)
	page, size := pagination.Page, pagination.PageSize
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	list, total, err := h.svc.ListVendors(ctx.Request.Context(), tid, page, size)
	if err != nil {
		h.failVendorError(ctx, err, "获取供应商列表失败", 0, tid)
		return
	}
	// 信封统一为 items/total/page/pageSize/totalPages，不再返回历史 list 键。
	common.SuccessWithList(ctx, list, total, page, size)
}

// GetVendor GET /api/v1/vendors/:id
func (h *Handler) GetVendor(ctx *gin.Context) {
	id, ok := pathVendorID(ctx)
	if !ok {
		return
	}
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	res, err := h.svc.GetVendor(ctx.Request.Context(), id, tid)
	if err != nil {
		h.failVendorError(ctx, err, "获取供应商失败", id, tid)
		return
	}
	common.Success(ctx, res)
}

// DeleteVendor DELETE /api/v1/vendors/:id
func (h *Handler) DeleteVendor(ctx *gin.Context) {
	id, ok := pathVendorID(ctx)
	if !ok {
		return
	}
	tid, ok := tenantID(ctx)
	if !ok {
		return
	}
	if err := h.svc.DeleteVendor(ctx.Request.Context(), id, tid); err != nil {
		h.failVendorError(ctx, err, "删除供应商失败", id, tid)
		return
	}
	common.Success(ctx, gin.H{"message": "deleted"})
}
