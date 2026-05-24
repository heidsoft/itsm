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
	tenantID, _ := ctx.Get("tenant_id")
	res, err := c.svc.CreateVendor(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, res)
}

func (c *VendorController) ListVendors(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))
	tenantID, _ := ctx.Get("tenant_id")
	list, total, err := c.svc.ListVendors(ctx.Request.Context(), tenantID.(int), page, size)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{"list": list, "total": total, "page": page})
}

func (c *VendorController) GetVendor(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	res, err := c.svc.GetVendor(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 404, "Vendor not found")
		return
	}
	common.Success(ctx, res)
}

func (c *VendorController) DeleteVendor(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	tenantID, _ := ctx.Get("tenant_id")
	if err := c.svc.DeleteVendor(ctx.Request.Context(), id, tenantID.(int)); err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "deleted"})
}
