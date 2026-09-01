package ticketworkflow

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

// getAuthContext extracts user_id and tenant_id from gin context with zero-value guards.
// Returns (userID, tenantID, ok). If either is missing, responds with auth error and returns ok=false.
// （自旧 controller 原样迁移：RequireTenantID 已含 401 响应，userID 零值兜底 2001）
func getAuthContext(c *gin.Context) (int, int, bool) {
	userID := c.GetInt("user_id")
	tenantID, ok := handlerctx.RequireTenantID(c)
	if !ok {
		return 0, 0, false
	}
	if userID == 0 || tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return 0, 0, false
	}
	return userID, tenantID, true
}

// AcceptTicket 接单
func (h *Handler) AcceptTicket(c *gin.Context) {
	var req dto.AcceptTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.AcceptTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to accept ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "接单成功"})
}

// RejectTicket 驳回工单
func (h *Handler) RejectTicket(c *gin.Context) {
	var req dto.RejectTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.RejectTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to reject ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "驳回成功"})
}

// WithdrawTicket 撤回工单
func (h *Handler) WithdrawTicket(c *gin.Context) {
	var req dto.WithdrawTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.WithdrawTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to withdraw ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "撤回成功"})
}

// ForwardTicket 转发工单
func (h *Handler) ForwardTicket(c *gin.Context) {
	var req dto.ForwardTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.ForwardTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to forward ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "转发成功"})
}

// CCTicket 抄送工单
func (h *Handler) CCTicket(c *gin.Context) {
	var req dto.CCTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.CCTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to cc ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "抄送成功"})
}

// ListMyCCRecords 获取当前用户收到的抄送记录
func (h *Handler) ListMyCCRecords(c *gin.Context) {
	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	resp, err := h.workflowService.ListMyCCRecords(c.Request.Context(), userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to list my CC records", "error", err, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, resp)
}

// ListTicketCCRecords 获取工单抄送记录
func (h *Handler) ListTicketCCRecords(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	resp, err := h.workflowService.ListTicketCCRecords(c.Request.Context(), ticketID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket CC records", "error", err, "ticket_id", ticketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, resp)
}

// ApproveTicket 审批工单
func (h *Handler) ApproveTicket(c *gin.Context) {
	var req dto.ApproveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.ApproveTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to approve ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	var message string
	switch req.Action {
	case "approve":
		message = "审批通过"
	case "reject":
		message = "审批拒绝"
	case "delegate":
		message = "已委派"
	default:
		message = "操作成功"
	}

	common.Success(c, gin.H{"message": message})
}

// ResolveTicket 解决工单
func (h *Handler) ResolveTicket(c *gin.Context) {
	var req dto.ResolveTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	// 兼容 ticket_id 和 ticketId 两种字段名（ResolveTicketRequest 的 JSON tag 是反向的）

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.ResolveTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to resolve ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单已解决"})
}

// CloseTicket 关闭工单
func (h *Handler) CloseTicket(c *gin.Context) {
	var req dto.CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	// 兼容 ticket_id 和 ticketId 两种字段名（CloseTicketRequest 的 JSON tag 是反向的）

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.CloseTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to close ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单已关闭"})
}

// ReopenTicket 重开工单
func (h *Handler) ReopenTicket(c *gin.Context) {
	var req dto.ReopenTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	err := h.workflowService.ReopenTicket(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to reopen ticket", "error", err, "ticket_id", req.TicketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "工单已重开"})
}

// GetTicketWorkflowState 获取工单流转状态
func (h *Handler) GetTicketWorkflowState(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	state, err := h.workflowService.GetTicketWorkflowState(c.Request.Context(), ticketID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get ticket workflow state", "error", err, "ticket_id", ticketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, state)
}

// GetTicketWorkflowStateV2 获取工单流转状态（V2：含 BPMN 真实节点详情）
func (h *Handler) GetTicketWorkflowStateV2(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	userID, tenantID, ok := getAuthContext(c)
	if !ok {
		return
	}

	state, err := h.workflowService.GetTicketWorkflowStateV2(c.Request.Context(), ticketID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to get ticket workflow state v2", "error", err, "ticket_id", ticketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, state)
}

// GetTicketWorkflowHistory 获取工单流转历史（原样迁移的裸 SQL 实现，含 P1-08 NullString 修复）
func (h *Handler) GetTicketWorkflowHistory(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	query := `
		SELECT id, ticket_id, action, from_status, to_status,
		       operator_id, from_user_id, to_user_id,
		       comment, reason, metadata, created_at
		FROM ticket_workflow_records
		WHERE ticket_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC, id ASC
	`
	rows, err := h.db.QueryContext(c.Request.Context(), query, ticketID, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	defer rows.Close()

	type workflowRec struct {
		ID         int64                  `json:"id"`
		TicketID   int64                  `json:"ticketId"`
		Action     string                 `json:"action"`
		FromStatus *string                `json:"fromStatus,omitempty"`
		ToStatus   *string                `json:"toStatus,omitempty"`
		OperatorID int64                  `json:"operatorId"`
		FromUserID *int64                 `json:"fromUserId,omitempty"`
		ToUserID   *int64                 `json:"toUserId,omitempty"`
		Comment    *string                `json:"comment"`
		Reason     *string                `json:"reason"`
		Metadata   map[string]interface{} `json:"metadata"`
		CreatedAt  time.Time              `json:"createdAt"`
	}
	var records []workflowRec
	for rows.Next() {
		var r workflowRec
		var metaJSON []byte
		// P1-08 修复：comment / reason 在表中可空，使用 sql.NullString 避免 Scan NULL 报错
		var commentNS, reasonNS sql.NullString
		if err := rows.Scan(&r.ID, &r.TicketID, &r.Action, &r.FromStatus, &r.ToStatus,
			&r.OperatorID, &r.FromUserID, &r.ToUserID, &commentNS, &reasonNS, &metaJSON, &r.CreatedAt); err != nil {
			common.FailWithErr(c, err, "操作失败")
			return
		}
		if commentNS.Valid {
			s := commentNS.String
			r.Comment = &s
		}
		if reasonNS.Valid {
			s := reasonNS.String
			r.Reason = &s
		}
		if len(metaJSON) > 0 {
			if err := json.Unmarshal(metaJSON, &r.Metadata); err != nil {
				h.logger.Warnw("Failed to unmarshal workflow metadata", "error", err, "ticket_id", ticketID)
			}
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if records == nil {
		records = []workflowRec{}
	}
	common.Success(c, records)
}
