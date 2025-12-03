package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketAssignmentSmartController struct {
	smartService *service.TicketAssignmentSmartService
	ruleService  *service.TicketAssignmentRuleService
	logger       *zap.SugaredLogger
}

func NewTicketAssignmentSmartController(
	smartService *service.TicketAssignmentSmartService,
	ruleService *service.TicketAssignmentRuleService,
	logger *zap.SugaredLogger,
) *TicketAssignmentSmartController {
	return &TicketAssignmentSmartController{
		smartService: smartService,
		ruleService:  ruleService,
		logger:       logger,
	}
}

// AutoAssign 自动分配工单
// POST /api/v1/tickets/:id/auto-assign
func (tasc *TicketAssignmentSmartController) AutoAssign(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	response, err := tasc.smartService.AutoAssign(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to auto assign ticket", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, response)
}

// GetAssignRecommendations 获取分配推荐
// GET /api/v1/tickets/assign-recommendations/:id
func (tasc *TicketAssignmentSmartController) GetAssignRecommendations(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	recommendations, err := tasc.smartService.GetAssignRecommendations(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to get assignment recommendations", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.GetAssignRecommendationsResponse{
		Recommendations: recommendations,
		Total:           len(recommendations),
	})
}

// ListAssignmentRules 获取分配规则列表
// GET /api/v1/tickets/assignment-rules
func (tasc *TicketAssignmentSmartController) ListAssignmentRules(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	rules, err := tasc.ruleService.ListAssignmentRules(c.Request.Context(), tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to list assignment rules", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListAssignmentRulesResponse{
		Rules: rules,
		Total: len(rules),
	})
}

// GetAssignmentRule 获取分配规则详情
// GET /api/v1/tickets/assignment-rules/:id
func (tasc *TicketAssignmentSmartController) GetAssignmentRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	rule, err := tasc.ruleService.GetAssignmentRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to get assignment rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// CreateAssignmentRule 创建分配规则
// POST /api/v1/tickets/assignment-rules
func (tasc *TicketAssignmentSmartController) CreateAssignmentRule(c *gin.Context) {
	var req dto.CreateAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	rule, err := tasc.ruleService.CreateAssignmentRule(c.Request.Context(), &req, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to create assignment rule", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// UpdateAssignmentRule 更新分配规则
// PUT /api/v1/tickets/assignment-rules/:id
func (tasc *TicketAssignmentSmartController) UpdateAssignmentRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	var req dto.UpdateAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	rule, err := tasc.ruleService.UpdateAssignmentRule(c.Request.Context(), ruleID, &req, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to update assignment rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, rule)
}

// DeleteAssignmentRule 删除分配规则
// DELETE /api/v1/tickets/assignment-rules/:id
func (tasc *TicketAssignmentSmartController) DeleteAssignmentRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的规则ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = tasc.ruleService.DeleteAssignmentRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to delete assignment rule", "error", err, "rule_id", ruleID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// TestAssignmentRule 测试分配规则
// POST /api/v1/tickets/assignment-rules/test
func (tasc *TicketAssignmentSmartController) TestAssignmentRule(c *gin.Context) {
	var req dto.TestAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	response, err := tasc.ruleService.TestAssignmentRule(c.Request.Context(), &req, tenantID)
	if err != nil {
		tasc.logger.Errorw("Failed to test assignment rule", "error", err, "rule_id", req.RuleID, "ticket_id", req.TicketID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, response)
}


