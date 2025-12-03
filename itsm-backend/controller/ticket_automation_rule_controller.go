package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

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
// GET /api/v1/tickets/automation-rules
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
// GET /api/v1/tickets/automation-rules/:id
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
// POST /api/v1/tickets/automation-rules
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
// PUT /api/v1/tickets/automation-rules/:id
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
// DELETE /api/v1/tickets/automation-rules/:id
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
// POST /api/v1/tickets/automation-rules/:id/test
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
