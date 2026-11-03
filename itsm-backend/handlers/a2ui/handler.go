// Package a2ui — A2UI 工单表单 handler.
// 迁移自 controller/a2ui_ticket_controller.go，保持原有 API 契约不变。
package a2ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/handlers/ticket"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ticketCreator 是 A2UI submit 所需的工单领域创建能力。
// 声明为窄接口而不是直接持有 *ticket.Service：
//   - 生产装配仍注入 ticket.NewService 的真实实现；
//   - handler 回归测试可注入 fake，断言请求参数与租户/用户来源。
//
// 工单创建的规则所有者仍是 handlers/ticket，这里不复制任何业务规则。
type ticketCreator interface {
	Create(ctx context.Context, tenantID int, params *ticket.CreateParams) (*ticket.Ticket, error)
}

type Handler struct {
	a2uiService   *service.A2UITicketService
	ticketCreator ticketCreator
	logger        *zap.SugaredLogger
}

func NewHandler(a2uiService *service.A2UITicketService, creator ticketCreator, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		a2uiService:   a2uiService,
		ticketCreator: creator,
		logger:        logger,
	}
}

type GenerateFormRequest struct {
	Intent    string `json:"intent" binding:"required"`
	SurfaceID string `json:"surfaceId"`
}

type GenerateFormResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
}

type HandleActionRequest struct {
	Action    string                 `json:"action" binding:"required"`
	SurfaceID string                 `json:"surfaceId" binding:"required"`
	Context   map[string]interface{} `json:"context"`
}

type HandleActionResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
	Success  bool     `json:"success"`
}

type ParseTicketIntentRequest struct {
	Description string `json:"description" binding:"required"`
}

// RegisterRoutes 注册 /api/v1/a2ui 路由组.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}
	// A2UI 属 AI 生成面：写操作按 ai:write 授权
	a2ai := rg.Group("/a2ui")
	{
		a2ai.POST("/ticket/form", middleware.RequirePermission("ai", "write"), h.GenerateForm)
		a2ai.POST("/ticket/action", middleware.RequirePermission("ai", "write"), h.HandleAction)
		// B9: 前端实际调用的别名 POST /api/v1/a2ui/tickets
		a2ai.POST("/tickets", middleware.RequirePermission("ai", "write"), h.ParseTicketIntent)
	}
}

// GenerateForm POST /api/v1/a2ui/ticket/form
func (h *Handler) GenerateForm(ctx *gin.Context) {
	var req GenerateFormRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	surfaceID := req.SurfaceID
	if surfaceID == "" {
		surfaceID = "ticket-create"
	}

	messages, err := h.a2uiService.GenerateFormMessages(ctx.Request.Context(), req.Intent, surfaceID)
	if err != nil {
		// B11（UPGRADE §1.48）：错误响应不透传内部 err.Error()。
		h.logger.Errorw("a2ui generate form failed", "intent", req.Intent, "error", err)
		common.Fail(ctx, common.InternalErrorCode, "生成表单失败")
		return
	}

	ctx.JSON(http.StatusOK, GenerateFormResponse{
		Code:     0,
		Message:  "success",
		Messages: messages,
	})
}

// HandleAction POST /api/v1/a2ui/ticket/action
//
// submit 走真实工单创建链路：tenant/user 只来自认证中间件上下文，
// 创建规则由 handlers/ticket 领域服务拥有（工作流、SLA、通知、审计随之生效）。
// 不再返回伪造的 TKT 编号或 501 未实现，也不把未知 action 伪装成成功。
func (h *Handler) HandleAction(ctx *gin.Context) {
	var req HandleActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	switch req.Action {
	case "submit":
		h.handleSubmit(ctx, &req)
	default:
		common.Fail(ctx, common.ParamErrorCode, "不支持的 A2UI 操作")
	}
}

// handleSubmit 解析表单数据并调用工单领域创建用例。
func (h *Handler) handleSubmit(ctx *gin.Context, req *HandleActionRequest) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}
	if tenantID <= 0 {
		common.Fail(ctx, common.AuthErrorCode, "Tenant ID missing")
		return
	}
	userID := ctx.GetInt("user_id")
	if userID <= 0 {
		common.Fail(ctx, common.AuthErrorCode, "User ID missing")
		return
	}
	if h.ticketCreator == nil {
		common.Fail(ctx, common.ServiceUnavailableCode, "工单创建服务未接入")
		return
	}

	form, err := extractTicketForm(req.Context)
	if err != nil {
		common.ParamErrorWithErr(ctx, err, a2uiPublicFormError(err))
		return
	}

	created, err := h.ticketCreator.Create(ctx.Request.Context(), tenantID, &ticket.CreateParams{
		Title:       form.title,
		Description: form.description,
		Type:        form.ticketType,
		Priority:    form.priority,
		RequesterID: userID,
		FormFields:  form.formFields,
	})
	if err != nil {
		// 统一错误出口：validation/forbidden/unavailable 按业务语义分流，不泄漏内部错误。
		common.RespondError(ctx, err, "创建工单失败")
		return
	}

	messages, err := buildSubmitSuccessMessages(req.SurfaceID, created)
	if err != nil {
		h.logger.Errorw("a2ui build submit response failed", "surface_id", req.SurfaceID, "ticket_id", created.ID, "error", err)
		common.Fail(ctx, common.InternalErrorCode, "工单已创建，但响应生成失败")
		return
	}

	ctx.JSON(http.StatusOK, HandleActionResponse{
		Code:     0,
		Message:  "success",
		Messages: messages,
		Success:  true,
	})
}

// a2uiTicketForm 是 A2UI 表单提交解析后的工单创建入参。
type a2uiTicketForm struct {
	title       string
	description string
	ticketType  string
	priority    string
	formFields  map[string]interface{}
}

// extractTicketForm 从 A2UI 按钮 context（{data: {title, type, priority, description}}）
// 解析出工单字段。context.data 来自前端数据模型，不可信，全部在此校验。
func extractTicketForm(contextData map[string]interface{}) (*a2uiTicketForm, error) {
	data, ok := contextData["data"].(map[string]interface{})
	if !ok {
		return nil, errInvalidA2UIForm
	}

	title := strings.TrimSpace(stringValue(data["title"]))
	if title == "" {
		return nil, errA2UITitleRequired
	}

	rawType := strings.TrimSpace(stringValue(data["type"]))
	if rawType == "" {
		rawType = "other"
	}

	priority := strings.TrimSpace(stringValue(data["priority"]))
	if priority == "" {
		priority = "medium"
	}
	if !isValidA2UIPriority(priority) {
		return nil, errA2UIPriorityInvalid
	}

	return &a2uiTicketForm{
		title:       title,
		description: stringValue(data["description"]),
		ticketType:  mapA2UITicketType(rawType),
		priority:    priority,
		// 保留 AI 表单原始类型，供后续扩展/审计使用；领域类型已按白名单归一。
		formFields: map[string]interface{}{"aiTicketType": rawType},
	}, nil
}

// mapA2UITicketType 把 AI 表单类型映射到工单领域允许的类型。
// 领域合法类型由 repository/ticket 定义：incident / problem / change / service_request。
// AI 表单的硬件/软件/网络/权限申请都属于服务请求，其余归为事件。
func mapA2UITicketType(raw string) string {
	switch raw {
	case "hardware", "software", "network", "access", "service_request":
		return "service_request"
	case "problem":
		return "problem"
	case "change":
		return "change"
	default:
		return "incident"
	}
}

func isValidA2UIPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "urgent", "critical":
		return true
	default:
		return false
	}
}

func stringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// buildSubmitSuccessMessages 构造前端可消费的 A2UI 消息序列：
// dataModelUpdate(status=created) 触发成功态，deleteSurface 清理表单。
func buildSubmitSuccessMessages(surfaceID string, created *ticket.Ticket) ([]string, error) {
	if created == nil {
		return nil, errA2UITicketCreateEmpty
	}
	ticketID := created.ID
	dataMsg := service.A2UIDataModelUpdate{
		DataModelUpdate: service.DataModelUpdate{
			SurfaceID: surfaceID,
			Path:      "/ticket",
			Contents: []service.Content{
				{Key: "status", ValueString: "created"},
				{Key: "ticketId", ValueNumber: &ticketID},
				{Key: "ticketNumber", ValueString: created.TicketNumber},
			},
		},
	}
	deleteMsg := service.A2UIDeleteSurface{
		DeleteSurface: service.DeleteSurface{SurfaceID: surfaceID},
	}

	data, err := json.Marshal(dataMsg)
	if err != nil {
		return nil, err
	}
	deleted, err := json.Marshal(deleteMsg)
	if err != nil {
		return nil, err
	}
	return []string{string(data), string(deleted)}, nil
}

var (
	errInvalidA2UIForm       = errors.New("缺少工单表单数据")
	errA2UITitleRequired     = errors.New("工单标题不能为空")
	errA2UIPriorityInvalid   = errors.New("无效的工单优先级")
	errA2UITicketCreateEmpty = errors.New("工单创建结果为空")
)

// a2uiPublicFormError 只放行上面三条自带文案的哨兵错误；其余错误原文进日志，
// 响应给固定文案，避免把内部结构透出到 A2UI 客户端。
func a2uiPublicFormError(err error) string {
	switch {
	case errors.Is(err, errInvalidA2UIForm),
		errors.Is(err, errA2UITitleRequired),
		errors.Is(err, errA2UIPriorityInvalid):
		return err.Error()
	default:
		return "工单表单数据不完整"
	}
}

// ParseTicketIntent POST /api/v1/a2ui/tickets
func (h *Handler) ParseTicketIntent(ctx *gin.Context) {
	var req ParseTicketIntentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	result, err := h.a2uiService.ParseTicketIntent(ctx.Request.Context(), req.Description)
	if err != nil {
		// B11（UPGRADE §1.48）：错误响应不透传内部 err.Error()。
		h.logger.Errorw("a2ui parse intent failed", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "解析工单意图失败")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
