package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/service"
)

type CMDBController struct {
	cmdbService           *service.CMDBService
	ciRelationshipService *service.CIRelationshipService
	auditLogService       *service.AuditLogService
}

func NewCMDBController(cmdbService *service.CMDBService, ciRelationshipService *service.CIRelationshipService, auditLogService *service.AuditLogService) *CMDBController {
	return &CMDBController{
		cmdbService:           cmdbService,
		ciRelationshipService: ciRelationshipService,
		auditLogService:       auditLogService,
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

// ListRelationships 获取CI关系列表
func (c *CMDBController) ListRelationships(ctx *gin.Context) {
	ciIDStr := ctx.Query("ci_id")
	var ciID *int
	if ciIDStr != "" {
		id, err := strconv.Atoi(ciIDStr)
		if err != nil {
			common.ParamError(ctx, "Invalid CI ID")
			return
		}
		ciID = &id
	}

	rels, err := c.cmdbService.ListRelationships(ctx.Request.Context(), 0, ciID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(ctx, rels)
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

// GetCIImpactAnalysis 获取CI影响分析
func (c *CMDBController) GetCIImpactAnalysis(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
		return
	}

	impact, err := c.ciRelationshipService.AnalyzeImpact(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "Failed to analyze impact: "+err.Error())
		return
	}

	common.Success(ctx, impact)
}

// GetCIChangeHistory 获取CI变更历史
func (c *CMDBController) GetCIChangeHistory(ctx *gin.Context) {
	idStr := ctx.Param("id")
	ciID, err := strconv.Atoi(idStr)
	if err != nil {
		common.ParamError(ctx, "Invalid CI ID")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "Tenant ID not found")
		return
	}

	// 获取审计日志中与此CI相关的记录
	// 通过resource字段过滤 (格式: ci_xxx 表示CI操作)
	page := 1
	pageSize := 20

	if p := ctx.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	if ps := ctx.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// 查询包含 ci_<id> 或 configuration_item 的资源记录
	// 由于审计日志resource字段可能存储不同格式，需要灵活查询
	logs, err := c.auditLogService.GetCIAuditLogs(ctx.Request.Context(), tenantID, ciID, page, pageSize)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "Failed to get change history: "+err.Error())
		return
	}

	common.Success(ctx, logs)
}
