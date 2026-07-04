package controller

import (
	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
)

// BPMNAIGeneratorController AI BPMN生成控制器
type BPMNAIGeneratorController struct {
	aiGeneratorService *service.BPMNAIGeneratorService
}

// NewBPMNAIGeneratorController 创建控制器实例
func NewBPMNAIGeneratorController(aiGeneratorService *service.BPMNAIGeneratorService) *BPMNAIGeneratorController {
	return &BPMNAIGeneratorController{
		aiGeneratorService: aiGeneratorService,
	}
}

// GenerateBPMN 生成BPMN流程
// @Summary 生成BPMN流程
// @Description 根据用户输入的业务需求自动生成BPMN工作流定义
// @Tags BPMN AI
// @Accept json
// @Produce json
// @Param request body dto.GenerateBPMNRequest true "生成请求"
// @Param auto_deploy query bool false "是否自动部署生成的流程"
// @Success 200 {object} common.Response{data=dto.GenerateBPMNResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/bpmn/ai/generate [post]
func (c *BPMNAIGeneratorController) GenerateBPMN(ctx *gin.Context) {
	var req dto.GenerateBPMNRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误: "+err.Error())
		return
	}

	autoDeploy := ctx.Query("auto_deploy") == "true"

	resp, err := c.aiGeneratorService.GenerateBPMN(ctx, &req, autoDeploy)
	if err != nil {
		common.Fail(ctx, 5001, "生成BPMN失败: "+err.Error())
		return
	}

	common.Success(ctx, resp)
}

// PreviewBPMN 预览流程结构
// @Summary 预览流程结构
// @Description 根据用户输入的业务需求预览流程结构，不生成完整XML
// @Tags BPMN AI
// @Accept json
// @Produce json
// @Param request body dto.PreviewBPMNRequest true "预览请求"
// @Success 200 {object} common.Response{data=dto.PreviewBPMNResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/bpmn/ai/preview [post]
func (c *BPMNAIGeneratorController) PreviewBPMN(ctx *gin.Context) {
	var req dto.PreviewBPMNRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, "参数错误: "+err.Error())
		return
	}

	resp, err := c.aiGeneratorService.PreviewBPMN(ctx, &req)
	if err != nil {
		common.Fail(ctx, 5001, "预览流程失败: "+err.Error())
		return
	}

	common.Success(ctx, resp)
}

// GetTemplateSuggestions 获取流程模板建议
// @Summary 获取流程模板建议
// @Description 根据用户输入的关键词推荐相关的流程模板
// @Tags BPMN AI
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param process_type query string false "流程类型过滤"
// @Success 200 {object} common.Response{data=[]dto.BPMNTemplateSuggestion}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/bpmn/ai/templates/suggestions [get]
func (c *BPMNAIGeneratorController) GetTemplateSuggestions(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
	if keyword == "" {
		common.Fail(ctx, 1001, "关键词不能为空")
		return
	}

	_ = ctx.Query("process_type")

	// 这里可以实现基于AI的模板推荐逻辑
	// 暂时返回空结果
	suggestions := []interface{}{}

	common.Success(ctx, suggestions)
}

// RegisterRoutes 注册路由
func (c *BPMNAIGeneratorController) RegisterRoutes(r *gin.RouterGroup) {
	bpmnAI := r.Group("/bpmn/ai")
	{
		// 生成BPMN流程
		bpmnAI.POST("/generate", c.GenerateBPMN)
		// 预览流程结构
		bpmnAI.POST("/preview", c.PreviewBPMN)
		// 获取模板建议
		bpmnAI.GET("/templates/suggestions", c.GetTemplateSuggestions)
	}
}
