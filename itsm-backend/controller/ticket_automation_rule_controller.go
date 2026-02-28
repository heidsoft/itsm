package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketAutomationRuleController struct {
	ruleService *service.TicketAutomationRuleService
	logger      *zap.SugaredLogger
}

func NewTicketAutomationRuleController(
	ruleService *service.TicketAutomationRuleService,
	logger *zap.SugaredLogger,
) *TicketAutomationRuleController {
	return &TicketAutomationRuleController{
		ruleService: ruleService,
		logger:      logger,
	}
}

// ListAutomationRules 获取自动化规则列表
// @Summary 获取自动化规则列表
// @Description 获取当前租户的自动化规则列表
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Success 200 {object} dto.ListAutomationRulesResponse
// @Router /api/v1/tickets/automation-rules [get]
func (tarc *TicketAutomationRuleController) ListAutomationRules(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	rules, err := tarc.ruleService.ListAutomationRules(c.Request.Context(), tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to list automation rules", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListAutomationRulesResponse{
		Rules: rules,
		Total: len(rules),
	})
}

// GetAutomationRule 获取自动化规则详情
// @Summary 获取自动化规则详情
// @Description 根据 ID 获取自动化规则详细信息
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} dto.AutomationRuleResponse
// @Router /api/v1/tickets/automation-rules/{id} [get]
func (tarc *TicketAutomationRuleController) GetAutomationRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	rule, err := tarc.ruleService.GetAutomationRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to get automation rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// CreateAutomationRule 创建自动化规则
// @Summary 创建自动化规则
// @Description 创建新的工单自动化规则
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Param request body dto.CreateAutomationRuleRequest true "创建自动化规则请求"
// @Success 200 {object} dto.AutomationRuleResponse
// @Router /api/v1/tickets/automation-rules [post]
func (tarc *TicketAutomationRuleController) CreateAutomationRule(c *gin.Context) {
	var req dto.CreateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	rule, err := tarc.ruleService.CreateAutomationRule(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to create automation rule", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// UpdateAutomationRule 更新自动化规则
// @Summary 更新自动化规则
// @Description 更新现有自动化规则的配置
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param request body dto.UpdateAutomationRuleRequest true "更新自动化规则请求"
// @Success 200 {object} dto.AutomationRuleResponse
// @Router /api/v1/tickets/automation-rules/{id} [put]
func (tarc *TicketAutomationRuleController) UpdateAutomationRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	var req dto.UpdateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	rule, err := tarc.ruleService.UpdateAutomationRule(c.Request.Context(), ruleID, &req, tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to update automation rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// DeleteAutomationRule 删除自动化规则
// @Summary 删除自动化规则
// @Description 根据 ID 删除自动化规则
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Success 200 {object} common.Response
// @Router /api/v1/tickets/automation-rules/{id} [delete]
func (tarc *TicketAutomationRuleController) DeleteAutomationRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = tarc.ruleService.DeleteAutomationRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to delete automation rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// TestAutomationRule 测试自动化规则
// @Summary 测试自动化规则
// @Description 使用模拟工单数据测试自动化规则的执行效果
// @Tags 自动化规则
// @Accept json
// @Produce json
// @Param id path int true "规则 ID"
// @Param request body dto.TestAutomationRuleRequest true "测试请求（包含模拟工单数据）"
// @Success 200 {object} dto.TestAutomationRuleResponse
// @Router /api/v1/tickets/automation-rules/{id}/test [post]
func (tarc *TicketAutomationRuleController) TestAutomationRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	var req dto.TestAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.RuleID = ruleID
	tenantID := c.GetInt("tenant_id")

	response, err := tarc.ruleService.TestAutomationRule(c.Request.Context(), &req, tenantID)
	if err != nil {
		tarc.logger.Errorw("Failed to test automation rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, response)
}
