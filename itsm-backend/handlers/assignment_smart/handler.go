// Package assignment_smart 是工单智能分配域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_assignment_smart_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketAssignmentSmartService / service.TicketAssignmentRuleService 承载，
// 本包只做参数解析与响应封装。
package assignment_smart

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 工单智能分配 HTTP handler
type Handler struct {
	smartService *service.TicketAssignmentSmartService
	ruleService  *service.TicketAssignmentRuleService
	logger       *zap.SugaredLogger
}

// NewHandler 创建工单智能分配 handler 实例
func NewHandler(
	smartService *service.TicketAssignmentSmartService,
	ruleService *service.TicketAssignmentRuleService,
	logger *zap.SugaredLogger,
) *Handler {
	return &Handler{
		smartService: smartService,
		ruleService:  ruleService,
		logger:       logger,
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	return handlerctx.ResolveTenantID(c)
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

// AutoAssign 自动分配工单
func (h *Handler) AutoAssign(c *gin.Context) {
	ticketID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	response, err := h.smartService.AutoAssign(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to auto assign ticket", "error", err, "ticket_id", ticketID)
		// 领域错误（工单不存在/跨租户）必须带着自己的状态码出去，不能被压成 500；
		// 未分类的驱动层错误仍只进日志，客户端拿到固定文案。
		common.RespondError(c, err, "自动分配失败")
		return
	}

	common.Success(c, response)
}

// GetAssignRecommendations 获取分配推荐
func (h *Handler) GetAssignRecommendations(c *gin.Context) {
	ticketID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	recommendations, err := h.smartService.GetAssignRecommendations(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get assignment recommendations", "error", err, "ticket_id", ticketID)
		// E4-29/E4-5：工单不存在与跨租户都是 service 的 4004 领域错误，必须原样带着
		// 404 出去；此前 FailWithErr 把它们压成 500/5001，与同域附件/变更的 404 口径不一致。
		common.RespondError(c, err, "获取分配推荐失败")
		return
	}

	common.Success(c, dto.AssignRecommendationListResponse{
		Items: recommendations,
		Total: len(recommendations),
	})
}

// ListAssignmentRules 获取分配规则列表
func (h *Handler) ListAssignmentRules(c *gin.Context) {
	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	rules, err := h.ruleService.ListAssignmentRules(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Failed to list assignment rules", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ListAssignmentRulesResponse{
		Items: rules,
		Total: len(rules),
	})
}

// GetAssignmentRule 获取分配规则详情
func (h *Handler) GetAssignmentRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	rule, err := h.ruleService.GetAssignmentRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get assignment rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// CreateAssignmentRule 创建分配规则
func (h *Handler) CreateAssignmentRule(c *gin.Context) {
	var req dto.CreateAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	rule, err := h.ruleService.CreateAssignmentRule(c.Request.Context(), &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to create assignment rule", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// UpdateAssignmentRule 更新分配规则
func (h *Handler) UpdateAssignmentRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	rule, err := h.ruleService.UpdateAssignmentRule(c.Request.Context(), ruleID, &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to update assignment rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, rule)
}

// DeleteAssignmentRule 删除分配规则
func (h *Handler) DeleteAssignmentRule(c *gin.Context) {
	ruleID, ok := pathID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	err := h.ruleService.DeleteAssignmentRule(c.Request.Context(), ruleID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to delete assignment rule", "error", err, "rule_id", ruleID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// TestAssignmentRule 测试分配规则
func (h *Handler) TestAssignmentRule(c *gin.Context) {
	var req dto.TestAssignmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := tenantID(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	response, err := h.ruleService.TestAssignmentRule(c.Request.Context(), &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to test assignment rule", "error", err, "rule_id", req.RuleID, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, response)
}
