package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
)

type VendorController struct {
	svc *service.VendorService
}

func NewVendorController(svc *service.VendorService) *VendorController {
	return &VendorController{svc: svc}
}

func (c *VendorController) CreateVendor(ctx *gin.Context) {
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
	res, err := c.svc.CreateVendor(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建供应商失败")
		return
	}
	common.Success(ctx, res)
}

func (c *VendorController) ListVendors(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
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
	list, total, err := c.svc.ListVendors(ctx.Request.Context(), tenantID, page, size)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取供应商列表失败")
		return
	}
	common.Success(ctx, gin.H{"list": list, "total": total, "page": page})
}

func (c *VendorController) GetVendor(ctx *gin.Context) {
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
	res, err := c.svc.GetVendor(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, 404, "Vendor not found")
		return
	}
	common.Success(ctx, res)
}

func (c *VendorController) DeleteVendor(ctx *gin.Context) {
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
	if err := c.svc.DeleteVendor(ctx.Request.Context(), id, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除供应商失败")
		return
	}
	common.Success(ctx, gin.H{"message": "deleted"})
}
