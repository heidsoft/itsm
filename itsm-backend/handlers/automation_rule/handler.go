// Package automation_rule 是工单自动化规则域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_automation_rule_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketAutomationRuleService 承载，本包只做参数解析与响应封装。
package automation_rule

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 工单自动化规则 HTTP handler
type Handler struct {
	ruleService *service.TicketAutomationRuleService
	logger      *zap.SugaredLogger
}

// NewHandler 创建工单自动化规则 handler 实例
func NewHandler(ruleService *service.TicketAutomationRuleService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		ruleService: ruleService,
		logger:      logger,
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) int {
	return c.GetInt("tenant_id")
}

// pathID 提取路径参数 ID
func pathID(c *gin.Context, param string) (int, bool) {
	id, err := strconv.Atoi(c.Param(param))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的ID参数")
		return 0, false
	}
	return id, true
}

// ListAutomationRules 获取自动化规则列表
func (h *Handler) ListAutomationRules(c *gin.Context) {
	tid := tenantID(c)
	rules, err := h.ruleService.ListAutomationRules(c.Request.Context(), tid)
	if err != nil {
		h.logger.Errorw("Failed to list automation rules", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ListAutomationRulesResponse{
		Rules: rules,
		Total: len(rules),
	})
}

// GetAutomationRule 获取自动化规则详情
func (h *Handler) GetAutomationRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tid := tenantID(c)
	rule, err := h.ruleService.GetAutomationRule(c.Request.Context(), ruleID, tid)
	if err != nil {
		h.logger.Errorw("Failed to get automation rule", "error", err, "rule_id", ruleID)
		if ent.IsNotFound(err) {
			common.NotFoundWithErr(c, err, "自动化规则不存在或无权访问")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// CreateAutomationRule 创建自动化规则
func (h *Handler) CreateAutomationRule(c *gin.Context) {
	var req dto.CreateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid := tenantID(c)
	uid := c.GetInt("user_id")
	rule, err := h.ruleService.CreateAutomationRule(c.Request.Context(), &req, uid, tid)
	if err != nil {
		h.logger.Errorw("Failed to create automation rule", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// UpdateAutomationRule 更新自动化规则
func (h *Handler) UpdateAutomationRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid := tenantID(c)
	rule, err := h.ruleService.UpdateAutomationRule(c.Request.Context(), ruleID, &req, tid)
	if err != nil {
		h.logger.Errorw("Failed to update automation rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// DeleteAutomationRule 删除自动化规则
func (h *Handler) DeleteAutomationRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tid := tenantID(c)
	err := h.ruleService.DeleteAutomationRule(c.Request.Context(), ruleID, tid)
	if err != nil {
		h.logger.Errorw("Failed to delete automation rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// TestAutomationRule 测试自动化规则
func (h *Handler) TestAutomationRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.TestAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	req.RuleID = ruleID
	tid := tenantID(c)
	response, err := h.ruleService.TestAutomationRule(c.Request.Context(), &req, tid)
	if err != nil {
		h.logger.Errorw("Failed to test automation rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, response)
}
