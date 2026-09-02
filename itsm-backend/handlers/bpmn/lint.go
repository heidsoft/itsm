package bpmn

import (
	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
)

// LintHandler BPMN Lint 控制器
//
// 提供后端流程校验真源：前端设计器「校验流程」与 BPMN AI 生成器
// 共用同一 Lint 规则（结构/事件/任务配置/连通性/网关语义）。
type LintHandler struct {
	lintService *service.BPMNLintService
}

// NewLintHandler 创建控制器实例
func NewLintHandler() *LintHandler {
	return &LintHandler{
		lintService: service.NewBPMNLintService(),
	}
}

// LintBPMN 校验 BPMN XML
// @Summary 校验 BPMN 流程定义
// @Description 对 BPMN XML 做结构与语义校验（起止事件/任务配置/连通性/网关语义），返回结构化问题列表。error 级问题不应部署。
// @Tags BPMN
// @Accept json
// @Produce json
// @Param request body dto.BPMNLintRequest true "Lint 请求"
// @Success 200 {object} common.Response{data=dto.BPMNLintResult}
// @Failure 400 {object} common.Response
// @Router /api/v1/bpmn/lint [post]
func (c *LintHandler) LintBPMN(ctx *gin.Context) {
	var req dto.BPMNLintRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误: "+err.Error())
		return
	}

	result, err := c.lintService.LintBPMNXML([]byte(req.BPMNXML))
	if err != nil {
		// XML 无法解析/缺命名空间/缺起止事件等结构性问题按参数错误返回
		common.Fail(ctx, 1001, "BPMN 校验失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}

// RegisterRoutes 注册路由
func (c *LintHandler) RegisterRoutes(r *gin.RouterGroup) {
	bpmn := r.Group("/bpmn")
	{
		// 校验 BPMN XML
		bpmn.POST("/lint", c.LintBPMN)
	}
}
