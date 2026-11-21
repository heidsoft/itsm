package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/knowledgeaccess"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListTools handles GET /api/v1/agent/tools
// 按 ToolDefinition.Resource/Action 过滤，仅返回当前角色有权限的工具
func (h *Handler) ListTools(c *gin.Context) {
	role := c.GetString("role")
	tenantID := c.GetInt("tenant_id")

	if tenantID <= 0 || c.GetInt("user_id") <= 0 || role == "" {
		common.AuthFailed(c, "缺少有效身份上下文")
		return
	}
	if h.svc.entClient == nil || h.svc.tools == nil {
		common.Fail(c, common.ServiceUnavailableCode, "AI 工具权限服务未就绪")
		return
	}

	// 按租户动态化工具参数（list_cis 的 ci_type 枚举来自该租户 CIType 表）
	allTools := h.svc.ListToolsForTenant(c.Request.Context(), tenantID)

	visible := make([]dto.ToolDefinition, 0, len(allTools))
	for _, t := range allTools {
		if middleware.HasResourcePermission(c.Request.Context(), h.svc.entClient, role, t.Resource, t.Action, tenantID) {
			visible = append(visible, t)
		}
	}
	common.Success(c, gin.H{"items": visible})
}

// ExecuteTool handles POST /api/v1/agent/tools/execute
func (h *Handler) ExecuteTool(c *gin.Context) {
	var req struct {
		Name string                 `json:"name" binding:"required"`
		Args map[string]interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 校验工具存在性；写工具（!ReadOnly）交由 ExecuteTool 统一走审批流
	// （创建 pending invocation + 入队等待人工审批），与聊天路径行为一致。
	if h.svc.tools == nil {
		common.Fail(c, common.ServiceUnavailableCode, "AI 工具注册表未就绪")
		return
	}
	toolDef := h.svc.tools.GetTool(req.Name)
	if toolDef == nil {
		common.Fail(c, common.UnknownToolCode, "unknown tool: "+req.Name)
		return
	}

	if req.Args == nil {
		req.Args = map[string]interface{}{}
	}

	userID := c.GetInt("user_id")
	if userID <= 0 {
		common.AuthFailed(c, "用户信息缺失")
		return
	}
	role := c.GetString("role")

	res, invocationID, err := h.svc.ExecuteTool(c.Request.Context(), userID, tenantID, role, req.Name, req.Args)
	if err != nil {
		if errors.Is(err, ErrToolUnavailable) {
			common.Fail(c, common.ServiceUnavailableCode, "AI 工具权限服务未就绪")
			return
		}
		// 区分权限拒绝与未知工具的错误码
		if errors.Is(err, ErrToolPermissionDenied) {
			common.Fail(c, common.ToolPermissionDeniedCode, "无权执行该工具")
			return
		}
		if errors.Is(err, ErrUnknownTool) {
			common.Fail(c, common.UnknownToolCode, "工具不存在")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	if invocationID > 0 {
		// 回执必须如实反映落库的审批态，不能一律写 "pending"：
		// AI/服务主体发起的写工具会落 pending_agent / pending_agent_two，
		// 硬编码 "pending" 会让调用方（前端审批页、MCP 客户端、Agent 编排器）
		// 误判「一人批完即放行」，而实际还卡在 awaiting_second 或需两名不同审批人。
		approvalState, nextActions := h.pendingApprovalReceipt(c.Request.Context(), invocationID, tenantID)
		common.Success(c, dto.ToolExecutionResponse{
			Status: "pending", Summary: pendingApprovalSummary(approvalState), InvocationID: invocationID,
			ApprovalState: approvalState, NextActions: nextActions, Artifacts: []string{},
		})
		return
	}
	common.Success(c, gin.H{
		"status":      "success",
		"summary":     "tool executed",
		"nextActions": []string{"If the result is incomplete, refine args and retry."},
		"artifacts":   []string{},
		"data":        res,
	})
}

// DryRunTool handles POST /api/v1/agent/tools/dry-run
//
// 预演：现在调用这个工具会发生什么。跑完真实判定链（Gate 2 角色 RBAC +
// Gate 2.5 主体策略 + 审批人数），但**不执行工具、不建待审批行**。
//
// 与 POST /tools/execute 的关系：判定只有一份（service.evaluateToolCall），
// 所以两者对「是否允许 / 是否需审批 / 需要几人」的回答必然一致；
// 差别只在副作用。调用方可以用本接口在真正调用前确认可行性，
// 而不必为此制造一条真实的待审批待办。
//
// 权限用 ai:read 而不是 ai:write：预演不产生任何业务副作用，
// 只读权限者预演自己的可调工具是合理需求；若用 ai:write 会出现
// 「有读权限但无写权限的人无法判断自己能不能写」的悖论。
func (h *Handler) DryRunTool(c *gin.Context) {
	var req struct {
		Name string                 `json:"name" binding:"required"`
		Args map[string]interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if h.svc.tools == nil {
		common.Fail(c, common.ServiceUnavailableCode, "AI 工具注册表未就绪")
		return
	}
	if req.Args == nil {
		req.Args = map[string]interface{}{}
	}

	userID := c.GetInt("user_id")
	// service 主体 userID 恒为 0 属正常（与 ExecuteTool 同一身份守卫口径）。
	// 但预演是「替某个用户问会发生什么」，service 主体没有可归属的用户，
	// 且预演本身不需要真实副作用，故此处要求真实用户身份。
	if userID <= 0 {
		common.AuthFailed(c, "用户信息缺失")
		return
	}
	role := c.GetString("role")

	resp, err := h.svc.DryRunTool(c.Request.Context(), userID, tenantID, role, req.Name, req.Args)
	if err != nil {
		if errors.Is(err, ErrToolUnavailable) {
			common.Fail(c, common.ServiceUnavailableCode, "AI 工具权限服务未就绪")
			return
		}
		if errors.Is(err, ErrUnknownTool) {
			common.Fail(c, common.UnknownToolCode, "工具不存在")
			return
		}
		if errors.Is(err, ErrToolPermissionDenied) {
			// 定义漏配属于服务端配置缺陷，不是调用方越权：用 500 而非 403，
			// 否则运维会去查权限表，而真因在工具注册里。
			common.FailWithErr(c, err, "工具定义不完整")
			return
		}
		common.FailWithErr(c, err, "预演失败")
		return
	}
	common.Success(c, resp)
}

// ReviewToolInvocation handles POST /api/v1/agent/tools/invocations/:id/review
//
// 复核一条既有工具调用：按它**当时的发起主体**与**当前策略**重新判定，
// 并报出两次判定的策略指纹比对结论。零副作用，不执行工具、不推进审批状态。
//
// 存在的业务理由：审批队列里的记录可能排队数小时。审批人批一条几小时前的
// AI 写操作前必须知道「策略改了没有、现在这条还合规吗」。落库的 policyVersion
// 只能回答「当时依据哪版策略」，不能回答「现在是否仍然放行」——而审批的
// 效力取决于**批准那一刻**的判定，不是排队时的。
//
// 权限用 ai:read：复核是只读判定，不产生任何副作用。审批动作本身仍由
// POST /tools/:id/approve（ai:write）门控，两者不可混用。
func (h *Handler) ReviewToolInvocation(c *gin.Context) {
	idStr := c.Param("id")
	invocationID, err := strconv.Atoi(idStr)
	if err != nil || invocationID <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid invocation id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if h.svc.tools == nil {
		common.Fail(c, common.ServiceUnavailableCode, "AI 工具注册表未就绪")
		return
	}
	userID := c.GetInt("user_id")
	if userID <= 0 {
		common.AuthFailed(c, "用户信息缺失")
		return
	}

	resp, reviewErr := h.svc.ReviewInvocation(c.Request.Context(), invocationID, tenantID, userID, c.GetString("role"))
	if reviewErr != nil {
		switch {
		case errors.Is(reviewErr, ErrToolUnavailable):
			common.Fail(c, common.ServiceUnavailableCode, "AI 工具权限服务未就绪")
		case errors.Is(reviewErr, ErrUnknownTool):
			// 工具已下线或记录参数损坏：不是「被拒」，也不该伪装成 404
			// （记录本身可能存在）。用 409 表达「当前不可复核」。
			common.FailWithErr(c, reviewErr, "该调用当前不可复核")
		case errors.Is(reviewErr, ErrToolPermissionDenied):
			common.Fail(c, common.ParamErrorCode, "缺少调用记录标识")
		case ent.IsNotFound(reviewErr):
			common.Fail(c, common.NotFoundCode, "invocation not found")
		default:
			h.svc.logger.Warnw("AI tool invocation review failed",
				"tenant_id", tenantID, "invocation_id", invocationID, "error", reviewErr)
			common.FailWithErr(c, reviewErr, "复核失败")
		}
		return
	}
	common.Success(c, resp)
}

// pendingApprovalReceipt 读取刚创建的 invocation 的**落库**审批态，供执行回执如实返回。
//
// 为什么要回读而不是由 Service 返回值携带：ExecuteTool 的返回契约是
// (result, invocationID, error) 三元组，被 4 条通道复用（HTTP、MCP、聊天工具循环、
// Skill）。给它加第四个返回值会波及所有调用点与契约测试；回读一次本身也有价值——
// 若落库状态与预期不符（并发审批已把它推到 awaiting_second/approved），
// 回执必须以库为准，不能凭调用时的推断。
//
// 读取失败不阻断回执：此时退回 "pending" 并附告警，因为「待审批」这个事实
// 已经由 invocationID>0 确定，状态细分读不到只是信息缺失，不是执行失败。
func (h *Handler) pendingApprovalReceipt(ctx context.Context, invocationID, tenantID int) (string, []string) {
	inv, err := h.svc.repo.GetToolInvocation(ctx, invocationID, tenantID)
	if err != nil || inv == nil || inv.ApprovalState == "" {
		return "pending", []string{"等待审批后执行"}
	}
	return inv.ApprovalState, pendingApprovalNextActions(inv.ApprovalState)
}

// pendingApprovalNextActions 给出该审批态下调用方应有的下一步，避免调用方
// 把「已放入审批队列」误读为「已执行」或「一人批完即放行」。
func pendingApprovalNextActions(state string) []string {
	switch state {
	case "pending_agent_two":
		// AI 发起的双人审批：首票只是转入 awaiting_second，不放行。
		return []string{"需要两名不同审批人依次批准（首票后进入 awaiting_second）", "发起人不得自批", "审批通过后由执行器落地副作用"}
	case "awaiting_second":
		return []string{"已有一名审批人批准，仍需第二名不同审批人复审", "发起人不得自批"}
	case "pending_agent":
		return []string{"需要一名非发起人的审批人批准", "发起人不得自批", "审批通过后由执行器落地副作用"}
	case "approved":
		return []string{"已通过审批，等待执行器落地副作用", "可查询 invocation 状态确认最终结果"}
	case "rejected":
		return []string{"审批已驳回，工具不会执行", "如需重试请重新发起调用"}
	default:
		return []string{"等待审批后执行"}
	}
}

// pendingApprovalSummary 返回与真实审批态一致的摘要文案。
// 双人审批与单人审批的摘要必须可区分，否则前端/Agent 无法据此提示审批工作量。
func pendingApprovalSummary(state string) string {
	switch state {
	case "pending_agent_two":
		return "AI 发起的工具操作已提交，需两名不同审批人批准"
	case "awaiting_second":
		return "AI 发起的工具操作已获首票，仍需第二名审批人复审"
	case "pending_agent":
		return "AI 发起的工具操作已提交，需一名非发起人审批人批准"
	case "approved":
		return "工具操作已通过审批，等待执行"
	case "rejected":
		return "工具操作已被驳回"
	default:
		return "工具操作已提交审批"
	}
}

// Chat handles POST /api/v1/ai/chat
func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		Query          string `json:"query" binding:"required"`
		PageSize       int    `json:"pageSize"`
		ConversationID int    `json:"conversationId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	role := c.GetString("role")
	if tenantID <= 0 || userID <= 0 || role == "" {
		common.AuthFailed(c, "缺少有效身份上下文")
		return
	}

	// 注入知识访问者身份：RAG 检索据此做分类级可见性过滤（L0 权限边界）。
	// 不注入则按匿名处理，已纳管的受限分类一律不可见（fail-closed）。
	chatCtx := knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role})

	// ChatWithTrace 在 Chat 之上补检索溯源（口径版本 / 检索模式 / 命中分）
	// 并落服务端决策底账。答案片段来自知识库检索，model 记的是检索模式而不是
	// 生成模型——写成模型名就是伪造归因。
	result, err := h.svc.ChatWithTrace(chatCtx, tenantID, userID, req.Query, req.PageSize, req.ConversationID)
	if err != nil {
		// RAG 失败时降级处理：返回空结果而非 500 错误，避免前端崩溃
		h.svc.logger.Warnw("AI Chat RAG 检索失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"answers":        []interface{}{},
			"conversationId": 0,
			"degraded":       true,
			"message":        "AI 服务暂时不可用，请稍后重试",
		})
		return
	}

	common.Success(c, gin.H{
		"answers":        result.Answers,
		"conversationId": result.ConversationID,
		"promptVersion":  result.PromptVersion,
		"model":          result.Model,
		"confidence":     result.Confidence,
		"auditId":        result.AuditID,
	})
}

// ChatStream handles POST /api/v1/ai/chat/stream and emits Server-Sent Events.
// Events:
//   - event: sources        data: [{objectType,id,title,snippet,score,...}]
//   - event: delta          data: {"content": "..."}
//   - event: done           data: {"conversationId": <id>}
//   - event: error          data: {"message": "..."}
func (h *Handler) ChatStream(c *gin.Context) {
	var req struct {
		Query          string `json:"query" binding:"required"`
		PageSize       int    `json:"pageSize"`
		ConversationID int    `json:"conversationId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	role := c.GetString("role")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// SSE headers. Nginx-friendly: X-Accel-Buffering:no disables proxy buffering.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// Streaming not supported: fall back to a normal chat response so the
		// client still gets an answer.
		answers, convID, err := h.svc.Chat(
			knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
			tenantID, userID, req.Query, req.PageSize, req.ConversationID)
		if err != nil {
			common.FailWithErr(c, err, "操作失败")
			return
		}
		common.Success(c, gin.H{"answers": answers, "conversationId": convID})
		return
	}

	writeEvent := func(event string, payload interface{}) {
		buf, err := json.Marshal(payload)
		if err != nil {
			return
		}
		// Multi-line data payloads must be prefixed on each line, but json.Marshal
		// produces a single line, so a single data: prefix is sufficient.
		_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(buf))
		flusher.Flush()
	}

	onSources := func(items []map[string]any) {
		writeEvent("sources", items)
	}
	onDelta := func(delta string) {
		writeEvent("delta", map[string]string{"content": delta})
	}

	// 注入访问者身份：AI 助手主链路，RAG 据此做知识分类可见性过滤（L0 权限边界）
	convID, _, err := h.svc.ChatStream(
		knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
		tenantID, userID, role, req.Query, req.PageSize, req.ConversationID, onSources, onDelta)
	if err != nil {
		h.svc.logger.Warnw("AI ChatStream 失败", "error", err, "tenantID", tenantID)
		// 配额/限流拒绝携带可行动文案：SSE 已 200 开头，客户端只能靠 event 里
		// 的语义字段区分「稍后再试（预算）」与「服务故障」。
		payload := map[string]string{"message": err.Error()}
		if errors.Is(err, dto.ErrQuotaExceeded) {
			payload["reason"] = "quota_exceeded"
			payload["message"] = "本月 AI token 配额已用完，请联系管理员调整配额或下月再试"
		} else if errors.Is(err, dto.ErrRateLimited) {
			payload["reason"] = "rate_limited"
		}
		writeEvent("error", payload)
		return
	}
	writeEvent("done", map[string]int{"conversationId": convID})
}

// ListConversations handles GET /api/v1/ai/conversations
func (h *Handler) ListConversations(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if c.GetInt("user_id") <= 0 || c.GetString("role") == "" {
		common.AuthFailed(c, "缺少有效身份上下文")
		return
	}
	userID := c.GetInt("user_id")
	convs, err := h.svc.ListConversations(c.Request.Context(), tenantID, userID)
	if err != nil {
		common.FailWithErr(c, err, "获取会话列表失败")
		return
	}
	common.Success(c, gin.H{"items": convs})
}

// GetConversation handles GET /api/v1/ai/conversations/:id
func (h *Handler) GetConversation(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID <= 0 || c.GetString("role") == "" {
		common.AuthFailed(c, "缺少有效身份上下文")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的会话ID")
		return
	}
	// 会话是用户私有的：校验租户 + 属主。他人会话与不存在都走同一条 404 路径，
	// 不通过错误差异暴露「这个 id 是否存在」。DB 故障必须走 500，不能折叠成 404。
	messages, err := h.svc.GetConversationMessages(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		respondConversationError(c, err)
		return
	}
	common.Success(c, gin.H{"items": messages})
}

// DeleteConversation handles DELETE /api/v1/ai/conversations/:id
func (h *Handler) DeleteConversation(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID <= 0 || c.GetString("role") == "" {
		common.AuthFailed(c, "缺少有效身份上下文")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的会话ID")
		return
	}
	if err := h.svc.DeleteConversation(c.Request.Context(), id, tenantID, userID); err != nil {
		respondConversationError(c, err)
		return
	}
	common.Success(c, nil)
}

// respondConversationError 会话读/删的错误分类单一事实源：
// ent.NotFoundError（真不存在，以及跨租户/跨用户经谓词表现为 not found）统一
// 404「会话不存在」——保留防枚举语义，不泄露资源存在性；其余（DB 故障等）
// 交 RespondError 归 500，原始驱动错误只进日志、不进响应。
func respondConversationError(c *gin.Context, err error) {
	var notFound *ent.NotFoundError
	if errors.As(err, &notFound) {
		common.NotFound(c, "会话不存在")
		return
	}
	common.RespondError(c, err, "会话操作失败")
}

// ListAIAnalysisResults handles GET /api/v1/ai/analysis-results
func (h *Handler) ListAIAnalysisResults(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	analysisType := c.Query("type")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	results, err := h.svc.ListAIAnalysisResults(c.Request.Context(), tenantID, analysisType, limit)
	if err != nil {
		common.FailWithErr(c, err, "获取分析历史失败")
		return
	}
	common.Success(c, gin.H{"items": results})
}

// GetAIAnalysisResult handles GET /api/v1/ai/analysis-results/:id
func (h *Handler) GetAIAnalysisResult(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的记录ID")
		return
	}
	result, err := h.svc.GetAIAnalysisResult(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "获取分析结果失败")
		return
	}
	common.Success(c, result)
}

// DeleteAIAnalysisResult handles DELETE /api/v1/ai/analysis-results/:id
func (h *Handler) DeleteAIAnalysisResult(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的记录ID")
		return
	}
	if err := h.svc.DeleteAIAnalysisResult(c.Request.Context(), id, tenantID); err != nil {
		common.FailWithErr(c, err, "删除分析结果失败")
		return
	}
	common.Success(c, nil)
}

// GetDeepAnalytics handles POST /api/v1/ai/analytics
func (h *Handler) GetDeepAnalytics(c *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	// R1：viewer 只从认证上下文取，报表与列表/导出共用同一行级谓词。
	userID, role := handlerctx.ReportViewer(c)
	viewer := dto.DashboardViewer{UserID: userID, Role: role}
	res, err := h.svc.GetDeepAnalytics(c.Request.Context(), &req, tenantID, viewer)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// GetTrendPrediction handles POST /api/v1/ai/predictions
func (h *Handler) GetTrendPrediction(c *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	// userID 用于把预测底账归属到具体操作者：谁依据这组预测做了决策，事后可追。
	userID := c.GetInt("user_id")
	// R1：预测底账归属用 userID；统计范围另用 viewer（角色 + 用户 ID）。
	_, role := handlerctx.ReportViewer(c)
	viewer := dto.DashboardViewer{UserID: userID, Role: role}
	res, err := h.svc.GetTrendPrediction(c.Request.Context(), &req, tenantID, userID, viewer)
	if err != nil {
		common.RespondError(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// AnalyzeTicket handles POST /api/v1/ai/tickets/:id/analyze
func (h *Handler) AnalyzeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid ticket id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")

	res, err := h.svc.AnalyzeTicketWithAudit(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// InvestigateTicket handles POST /api/v1/ai/tickets/:id/investigate.
//
// 触发一等公民的工单调查编排（triage → KB → CMDB → LLM 综合）。错误映射：
// 目标工单不存在（含跨租户，不泄漏存在性）→ 404；编排依赖未装配 → 503。
func (h *Handler) InvestigateTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid ticket id")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID, role := handlerctx.ReportViewer(c)
	if userID <= 0 {
		common.AuthFailed(c, "用户信息缺失")
		return
	}

	res, err := h.svc.InvestigateTicket(c.Request.Context(), id, tenantID, userID, role)
	if err != nil {
		if errors.Is(err, ErrInvestigationTicketNotFound) {
			common.NotFoundWithErr(c, err, "工单不存在")
			return
		}
		if errors.Is(err, ErrInvestigationUnavailable) {
			common.Fail(c, common.ServiceUnavailableCode, "AI 工单调查服务尚未就绪")
			return
		}
		common.FailWithErr(c, err, "工单调查失败")
		return
	}
	common.Success(c, res)
}

// AnalyzeIncident handles POST /api/v1/ai/incidents/:id/analyze.
func (h *Handler) AnalyzeIncident(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid incident id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	res, err := h.svc.AnalyzeIncidentWithAudit(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		if errors.Is(err, dto.ErrIncidentNotFound) {
			common.NotFoundWithErr(c, err, "事件不存在")
			return
		}
		if errors.Is(err, dto.ErrAIAnalysisUnavailable) {
			common.Fail(c, common.ServiceUnavailableCode, "AI 事件分析服务尚未就绪")
			return
		}
		common.FailWithErr(c, err, "事件分析失败")
		return
	}
	common.Success(c, res)
}

// SummarizeTicket handles GET /api/v1/ai/tickets/:id/summary
// B9: AI 工单总结 - 优先用 LLM，fallback 用字段拼接
func (h *Handler) SummarizeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid ticket id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	// userID 用于把决策底账归属到具体操作者：谁采纳/拒绝了这条 AI 建议，
	// 事后要能追到人。取不到时传 0（skill 等无用户上下文的调用同此）。
	userID := c.GetInt("user_id")

	summary, err := h.svc.SummarizeTicket(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.svc.logger.Warnw("AI摘要失败，返回降级响应", "error", err, "ticketID", id)
		common.Success(c, gin.H{
			"degraded": true,
			"message":  "AI 摘要服务暂时不可用，请稍后重试",
			"summary":  "",
		})
		return
	}
	common.Success(c, summary)
}

// SaveFeedback handles POST /api/v1/ai/feedback
func (h *Handler) SaveFeedback(c *gin.Context) {
	var req struct {
		Kind     string  `json:"kind" binding:"required"`
		Query    string  `json:"query"`
		ItemType *string `json:"itemType"`
		ItemID   *int    `json:"itemId"`
		Useful   bool    `json:"useful" binding:"required"`
		Score    *int    `json:"score"`
		Notes    *string `json:"notes"`
		Accepted *bool   `json:"accepted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	requestID := c.GetString("request_id")
	if requestID == "" {
		// 幂等/归因身份禁止拼时间戳（AGENTS.md）：随机 128-bit ID 兜底，
		// 用户归属已由 tenant_id/user_id 列承载，不再拼进 ID。
		requestID = ledgerRequestID(c.Request.Context(), "feedback")
	}

	var itemTypeVal string
	if req.ItemType != nil {
		itemTypeVal = *req.ItemType
	}

	err := h.svc.SaveFeedback(c.Request.Context(), tenantID, userID, requestID, req.Kind, req.Query, itemTypeVal, req.ItemID, req.Useful, req.Score, req.Notes, req.Accepted)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, gin.H{"message": "Feedback saved"})
}

// RecordAudit handles POST /api/v1/ai/audit.
// It records the GA AI trace contract without allowing the AI to auto-apply high-risk actions.
func (h *Handler) RecordAudit(c *gin.Context) {
	var req struct {
		Scenario      string                 `json:"scenario" binding:"required"`
		InputRef      string                 `json:"inputRef" binding:"required"`
		PromptVersion string                 `json:"promptVersion"`
		Model         string                 `json:"model"`
		Confidence    float64                `json:"confidence"`
		Suggestion    map[string]interface{} `json:"suggestion" binding:"required"`
		Accepted      bool                   `json:"accepted"`
		Notes         string                 `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	// 置信度契约统一 0-1：超出区间或 NaN/Inf 一律 400，不再静默 clamp——
	// 非法值进入 evaluator 的置信度校准分桶会把整个桶污染成"低置信"。
	if req.Confidence < 0 || req.Confidence > 1 || math.IsNaN(req.Confidence) || math.IsInf(req.Confidence, 0) {
		common.Fail(c, common.ValidationError, "confidence 必须在 0-1 之间（0 与 1 边界合法）")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	requestID := c.GetString("request_id")
	if requestID == "" {
		// 幂等/归因身份禁止拼时间戳（AGENTS.md）：随机 128-bit ID 兜底。
		requestID = ledgerRequestID(c.Request.Context(), "ai_audit")
	}

	notePayload := map[string]interface{}{
		"promptVersion": req.PromptVersion,
		"model":         req.Model,
		"confidence":    req.Confidence,
		"suggestion":    req.Suggestion,
		"notes":         req.Notes,
	}
	noteBytes, _ := json.Marshal(notePayload)
	note := string(noteBytes)
	itemType := "ai_audit"
	score := int(req.Confidence * 100)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	if err := h.svc.SaveFeedback(c.Request.Context(), tenantID, userID, requestID, req.Scenario, req.InputRef, itemType, nil, req.Accepted, &score, &note, &req.Accepted); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"requestId":     requestID,
		"scenario":      req.Scenario,
		"inputRef":      req.InputRef,
		"promptVersion": req.PromptVersion,
		"model":         req.Model,
		"confidence":    req.Confidence,
		"accepted":      req.Accepted,
	})
}

// GetEvaluation handles GET /api/v1/ai/evaluation.
// 输出 AI 评估报告：按场景的有用率、置信度校准、平台级 LLM 成功率/延迟。
func (h *Handler) GetEvaluation(c *gin.Context) {
	days := queryInt(c, "days", 30)
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	report, err := h.svc.Evaluate(c.Request.Context(), tenantID, days)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, report)
}

// GetAuditLogs handles GET /api/v1/ai/audit-logs.
// 分页查询 AI 审计记录（item_type='ai_audit'），可按场景 kind 过滤。
func (h *Handler) GetAuditLogs(c *gin.Context) {
	pg := common.GetPaginationFromQuery(c)
	days := queryInt(c, "days", 90)
	kind := c.Query("kind")
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		// ResolveTenantID 已按中间件契约写入 401/403/500，不能再写第二个响应体。
		return
	}
	entries, total, err := h.svc.ListAuditLogs(c.Request.Context(), tenantID, pg.Page, pg.PageSize, kind, days)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.SuccessWithPagination(c, entries, pg.Page, pg.PageSize, int64(total))
}

// UpdateAuditAccepted handles PATCH /api/v1/ai/audit-logs/:id.
//
// AI Audit accept/reject feedback loop 闭合：让
// 操作者在 Audit Console 详情 Drawer 里点 Accept/Revoke 翻转
// 已记录的决策。原 ai_feedbacks 表写时即终态（append-only），
// 现在允许 PATCH 翻 accepted 列；用 usecase / version 不变量校验
// 不在此层（粗粒度足够，本提交只解决数据通路）。
func (h *Handler) UpdateAuditAccepted(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		common.ParamErrorWithErr(c, err, "id 参数非法")
		return
	}
	var req struct {
		Accepted bool `json:"accepted" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	if err := h.svc.UpdateAuditAccepted(c.Request.Context(), tenantID, id, req.Accepted); err != nil {
		if errors.Is(err, dto.ErrFeedbackNotFound) {
			common.NotFoundWithErr(c, err, "audit-logs 行不存在或非 ai_audit")
			return
		}
		common.FailWithErr(c, err, "更新 accepted 失败")
		return
	}
	common.Success(c, gin.H{"id": id, "accepted": req.Accepted})
}

// queryInt 解析正整数查询参数，非法/缺失时回退到默认值。
func queryInt(c *gin.Context, key string, def int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// GetMetrics handles GET /api/v1/ai/metrics.
func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	lookbackDays := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 365 {
			lookbackDays = days
		}
	}
	metrics, err := h.svc.GetMetrics(c.Request.Context(), tenantID, lookbackDays)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, metrics)
}

// KnowledgeSearch handles POST /api/v1/ai/rag/search - RAG search over knowledge base
func (h *Handler) KnowledgeSearch(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		PageSize int    `json:"pageSize"`
		Type     string `json:"type"` // kb|incident
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	role := c.GetString("role")

	limit := req.PageSize
	if limit <= 0 {
		limit = 5
	}

	// Use the service's RAG search capability
	// 注入访问者身份，使知识分类可见性过滤生效（L0 权限边界）
	result, err := h.svc.SearchKnowledge(
		knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
		tenantID, req.Query, req.Type, limit)
	if err != nil {
		h.svc.logger.Warnw("AI知识搜索失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"items":    []interface{}{},
			"degraded": true,
			"message":  "AI 搜索服务暂时不可用，请稍后重试",
		})
		return
	}
	common.Success(c, gin.H{
		"items":    result,
		"degraded": false,
	})
}

// Triage handles POST /api/v1/ai/triage - Ticket classification and recommendation
//
// ticketId 可选：带上后按工单幂等（同一工单只分析一次，重复进入详情页复用既有结论），
// 避免每次刷新都重跑模型并让分类在多次之间漂移（prod 实测 F2）。
func (h *Handler) Triage(c *gin.Context) {
	var req struct {
		TicketID    int    `json:"ticketId"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BindValidationError(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	result, err := h.svc.TriageTicket(c.Request.Context(), tenantID, c.GetInt("user_id"), req.Title, req.Description, req.Category, req.Priority, req.TicketID)
	if err != nil {
		h.svc.logger.Warnw("AI分诊失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"title":       req.Title,
			"description": req.Description,
			"suggestions": map[string]interface{}{},
			"degraded":    true,
			"message":     "AI 分诊服务暂时不可用，请稍后重试",
		})
		return
	}
	common.Success(c, result)
}

// CreateTicketByAI handles POST /api/v1/ai/ticket/create
// 通过 AI 解析自然语言描述，智能分析描述并返回工单创建建议
func (h *Handler) CreateTicketByAI(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		TenantID    int    `json:"tenantId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 调用 AI 分析描述，返回工单创建建议
	result, err := h.svc.CreateTicketByAI(c.Request.Context(), req.Description, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "AI ticket creation failed")
		return
	}
	common.Success(c, result)
}

// GetToolInvocation handles GET /api/v1/agent/tools/:id
// 查询工具执行记录（跨租户隔离）
func (h *Handler) GetToolInvocation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid invocation id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	inv, err := h.svc.repo.GetToolInvocation(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "invocation not found")
		return
	}
	// 统一走 toolInvocationView：此前这里手写一份 gin.H，导致新增审计字段
	// （policyVersion/runId）只在列表页出现、详情页缺失。审计字段的字段级
	// 漂移比缺字段更糟——详情页是复查单条调用时唯一会看的页面。
	common.Success(c, h.toolInvocationView(c.Request.Context(), tenantID, inv))
}

// invocationPolicyMeta 返回该工具对应权限码的风险摘要，供展示面使用。
// 策略不可读时返回 unknown/0：展示面不因目录不可读整体失败，
// 安全判定仍以执行时 Gate 2.5 严格路径为准。

// toolInvocationView 把工具调用记录序列化成 API 响应体。
//
// 收敛成单一函数而不是三处各写一份 gin.H：字段集合一旦分叉，
// 必然出现「详情页有、列表页没有」的字段级漂移——审计字段尤其不能这样，
// 复查时看不到策略版本就等于没记。
//
// policyVersion 原样透出三态（空串 / unknown / 指纹），不在此处做解释性改写：
// 读面把语义翻译成「人类友好」会掩盖「读不出来」与「无策略参与」的区别。
func (h *Handler) toolInvocationView(ctx context.Context, tenantID int, inv *ToolInvocation) gin.H {
	riskLevel, minApprovals := h.invocationPolicyMeta(ctx, tenantID, inv.ToolName)
	return gin.H{
		"id":               inv.ID,
		"toolName":         inv.ToolName,
		"arguments":        inv.Arguments,
		"result":           inv.Result,
		"error":            inv.Error,
		"status":           inv.Status,
		"needsApproval":    inv.NeedsApproval,
		"approvalState":    inv.ApprovalState,
		"approvalReason":   inv.ApprovalReason,
		"permissionCheck":  inv.PermissionCheck,
		"permissionReason": inv.PermissionReason,
		"requestId":        inv.RequestID,
		"createdAt":        inv.CreatedAt,
		"conversationId":   inv.ConversationID,
		"userId":           inv.UserID,
		// Phase B6：业务单据强关联（0 表示无关联）
		"ticketId":     inv.TicketID,
		"changeId":     inv.ChangeID,
		"riskLevel":    riskLevel,
		"minApprovals": minApprovals,
		// 审计回放（2026-11-21）
		"policyVersion": inv.PolicyVersion,
		"runId":         inv.RunID,
		// 预演标记（2026-11-21）。读面必须带上它：审批队列虽然已把
		// dry_run=true 排除在外，但「按运行轮次」读面（ListRunInvocations）
		// 与详情读面会同时出现真实调用与预演记录，消费方（前端调用链视图、
		// 事后审计）只有拿到这个布尔值才能区分「这一步真的发生了」与
		// 「这一步只是在问会发生什么」。漏掉它，预演就会在回放里被当成真实动作。
		"dryRun": inv.DryRun,
	}
}

// ListToolInvocations handles GET /api/v1/agent/tools/invocations
// 列出工具调用审批记录，供审批人查看待办。默认返回 pending（待审批），
// 支持 ?state=approved|rejected|auto 查看其它状态。跨租户隔离由 tenant_id 强制。
//
// 每条记录附带该工具对应权限码的运行期策略摘要（riskLevel/minApprovals），
// 供审批页展示风险分级与审批进度。策略缺失时降级为 unknown/0：
// 展示面不因目录不可读而整体失败，安全判定仍以执行时 Gate 2.5 严格路径为准。
func (h *Handler) ListToolInvocations(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		state = "pending"
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	invs, err := h.svc.repo.ListToolInvocations(c.Request.Context(), tenantID, state)
	if err != nil {
		h.svc.logger.Warnw("查询工具调用列表失败", "error", err, "tenantID", tenantID)
		common.Fail(c, common.InternalErrorCode, "查询工具调用失败")
		return
	}

	items := make([]gin.H, 0, len(invs))
	for _, inv := range invs {
		items = append(items, h.toolInvocationView(c.Request.Context(), tenantID, inv))
	}
	common.Success(c, gin.H{"items": items, "state": state})
}

// ListRunInvocations handles GET /api/v1/agent/runs/:runId/invocations
//
// 按运行轮次返回该轮全部工具调用（含只读审计行），按时间正序。
// 这是「不可抵赖审计」的读面：调用链按发生顺序排列，每步带
// policyVersion（依据哪版策略放行）与 approvalState（是否需要及是否通过审批）。
//
// 跨租户隔离：run_id 是调用方可传的字符串，仓储层强制带 tenant_id 谓词；
// 若只按 run_id 查，任何人都能读到别人的运行记录。
//
// 空 run_id 返回 400 而不是全表：空串语义是「不属于任何运行轮次」，
// 没有可聚合的内容，静默返回空集会让调用方以为「这轮什么都没做」。
func (h *Handler) ListRunInvocations(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("runId"))
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if runID == "" {
		common.Fail(c, common.ParamErrorCode, "runId 不能为空")
		return
	}
	// 长度上限：run_id 是客户端可控字符串，超长值会让索引查询退化为扫表，
	// 也会污染审计日志。64 字符足以覆盖当前生成格式（run_ + 32 hex）。
	if len(runID) > 64 {
		common.Fail(c, common.ParamErrorCode, "runId 过长")
		return
	}

	invs, err := h.svc.repo.ListToolInvocationsByRun(c.Request.Context(), tenantID, runID)
	if err != nil {
		h.svc.logger.Warnw("按运行轮次查询工具调用失败",
			"error", err, "tenantID", tenantID, "runId", runID)
		common.Fail(c, common.InternalErrorCode, "查询运行轮次记录失败")
		return
	}

	items := make([]gin.H, 0, len(invs))
	for _, inv := range invs {
		items = append(items, h.toolInvocationView(c.Request.Context(), tenantID, inv))
	}
	common.Success(c, gin.H{"items": items, "runId": runID, "count": len(items)})
}

// invocationPolicyMeta 返回工具对应权限码的风险等级与最低审批人数（只读展示用）。
func (h *Handler) invocationPolicyMeta(ctx context.Context, tenantID int, toolName string) (string, int) {
	toolDef := h.svc.tools.GetTool(toolName)
	if toolDef == nil || toolDef.Resource == "" || toolDef.Action == "" {
		return "unknown", 0
	}
	catalog := authz.Shared()
	if catalog == nil {
		return "unknown", 0
	}
	def, err := catalog.Lookup(ctx, tenantID, toolDef.Resource+":"+toolDef.Action)
	if err != nil || def == nil {
		return "unknown", 0
	}
	return def.Policy.RiskLevel, def.Policy.MinApprovals
}

// ApproveTool handles POST /api/v1/agent/tools/:id/approve
// 审批危险工具执行请求（RBAC 由路由中间件强制）
func (h *Handler) ApproveTool(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid invocation id")
		return
	}
	var req struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	// Phase A：角色传入审批资格矩阵——审批人须持目标工具 resource:action。
	role := c.GetString("role")

	state, err := h.svc.ApproveTool(c.Request.Context(), id, tenantID, userID, role, req.Approve, req.Reason)
	if err != nil {
		// 不存在是 404，重复审批是 409，其余内部错误 500——不能一律折叠成 404。
		common.RespondError(c, err, "工具调用审批失败")
		return
	}
	common.Success(c, gin.H{"invocationId": id, "approvalState": state})
}
