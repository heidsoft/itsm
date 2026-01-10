package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/service"
)

type CMDBController struct {
	cmdbService *service.CMDBService
}

func NewCMDBController(cmdbService *service.CMDBService) *CMDBController {
	return &CMDBController{
		cmdbService: cmdbService,
	}
}

// CreateCI 创建配置项
func (c *CMDBController) CreateCI(ctx *gin.Context) {
	var req service.CreateCIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	ci, err := c.cmdbService.CreateCI(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, ci)
}

// GetCI 获取配置项
func (c *CMDBController) GetCI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	ci, err := c.cmdbService.GetCI(ctx.Request.Context(), id)
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "CI not found")
		return
	}

	common.Success(ctx, ci)
}

// ListCIs 列出配置项
func (c *CMDBController) ListCIs(ctx *gin.Context) {
	var req service.ListCIsRequest
	
	req.CiType = ctx.Query("ci_type")
	req.Status = ctx.Query("status")
	req.Environment = ctx.Query("environment")
	
	if offset := ctx.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			req.Offset = o
		}
	}
	
	if limit := ctx.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			req.Limit = l
		}
	} else {
		req.Limit = 20
	}

	cis, err := c.cmdbService.ListCIs(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, gin.H{"data": cis})
}

// CreateRelationship 创建CI关系
func (c *CMDBController) CreateRelationship(ctx *gin.Context) {
	var req service.CreateRelationshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamError(ctx, "Invalid request body")
		return
	}

	rel, err := c.cmdbService.CreateRelationship(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, rel)
}

// GetCITopology 获取CI拓扑
func (c *CMDBController) GetCITopology(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	depth := 3 // 默认深度
	if depthStr := ctx.Query("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil {
			depth = d
		}
	}

	topology, err := c.cmdbService.GetCITopology(ctx.Request.Context(), id, depth)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, topology)
}
