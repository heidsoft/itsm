package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/service"
)

type EscalationMatrixController struct {
	logger                  *zap.SugaredLogger
	escalationMatrixService *service.EscalationMatrixService
}

func NewEscalationMatrixController(logger *zap.SugaredLogger, escalationMatrixService *service.EscalationMatrixService) *EscalationMatrixController {
	return &EscalationMatrixController{
		logger:                  logger,
		escalationMatrixService: escalationMatrixService,
	}
}

// GetMatrix 获取当前租户的升级矩阵
// @Summary 获取升级矩阵
// @Tags 升级矩阵
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=service.EscalationMatrix}
// @Router /api/v1/escalation-matrices [get]
func (c *EscalationMatrixController) GetMatrix(ctx *gin.Context) {
	tenantID, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}
	matrix := c.escalationMatrixService.GetMatrix(tenantID.(int))
	common.Success(ctx, matrix)
}

// SetMatrix 设置当前租户的升级矩阵
// @Summary 设置升级矩阵
// @Tags 升级矩阵
// @Accept json
// @Produce json
// @Param matrix body service.EscalationMatrix true "升级矩阵配置"
// @Success 200 {object} common.Response
// @Router /api/v1/escalation-matrices [put]
func (c *EscalationMatrixController) SetMatrix(ctx *gin.Context) {
	tenantID, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}
	var matrix service.EscalationMatrix
	if err := ctx.ShouldBindJSON(&matrix); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误："+err.Error())
		return
	}
	c.escalationMatrixService.SetMatrix(tenantID.(int), matrix)
	// 清除缓存
	c.escalationMatrixService.InvalidateCache(tenantID.(int))
	common.Success(ctx, nil)
}

// InvalidateCache 清除升级矩阵缓存
// @Summary 清除缓存
// @Tags 升级矩阵
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/escalation-matrices/invalidate-cache [post]
func (c *EscalationMatrixController) InvalidateCache(ctx *gin.Context) {
	tenantID, ok := ctx.Get("tenant_id")
	if !ok {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问")
		return
	}
	c.escalationMatrixService.InvalidateCache(tenantID.(int))
	common.Success(ctx, nil)
}

// RegisterRoutes 注册路由
func (c *EscalationMatrixController) RegisterRoutes(group *gin.RouterGroup) {
	matrixGrp := group.Group("/escalation-matrices")
	{
		matrixGrp.GET("", c.GetMatrix)
		matrixGrp.PUT("", c.SetMatrix)
		matrixGrp.POST("/invalidate-cache", c.InvalidateCache)
	}
}
