// Package provisioning — 交付任务 handler.
// 迁移自 controller/provisioning_controller.go，保持原有 API 契约不变。
package provisioning

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/domain/provisioning"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 ProvisioningService 依赖.
type Handler struct {
	provisioningService *service.ProvisioningService
	logger              *zap.SugaredLogger
}

// NewHandler 构造 provisioning Handler.
func NewHandler(provisioningService *service.ProvisioningService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		provisioningService: provisioningService,
		logger:              logger,
	}
}

// StartProvisioning POST /api/v1/service-requests/:id/provision
func (h *Handler) StartProvisioning(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的服务请求ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	task, err := h.provisioningService.CreateTaskFromServiceRequest(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// not-found → 404、状态不符 → 409，其余仍为 500；由 classifyError 统一稳定映射。
		common.FailWithErr(c, err, "创建交付任务失败")
		return
	}

	common.Success(c, dto.StartProvisioningResponse{Task: taskToResponse(task)})
}

// ListProvisioningTasks GET /api/v1/service-requests/:id/provisioning-tasks
func (h *Handler) ListProvisioningTasks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的服务请求ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}

	tasks, err := h.provisioningService.ListTasksByServiceRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "获取交付任务列表失败")
		return
	}
	out := make([]*dto.ProvisioningTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskToResponse(t))
	}
	common.Success(c, out)
}

// ExecuteProvisioningTask POST /api/v1/provisioning-tasks/:id/execute
func (h *Handler) ExecuteProvisioningTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的交付任务ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	task, err := h.provisioningService.ExecuteTask(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// 执行权冲突 → 409 且带可操作文案；provider 执行失败 → 500。
		// 由 classifyProvisioningError 统一稳定映射。
		h.respondProvisioningError(c, err, "执行交付任务失败")
		return
	}
	common.Success(c, taskToResponse(task))
}

// RetryProvisioningTask POST /api/v1/provisioning-tasks/:id/retry
//
// 重试是「把失败任务重新排队」，不是「立刻同步执行」：请求返回时任务处于
// pending，执行由队列 worker 完成，因此响应里的 status 是排队后的状态。
func (h *Handler) RetryProvisioningTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的交付任务ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	// body 可选：无 body 时按「未说明原因的重试」处理，不因缺字段拒绝一次合法重试。
	var req dto.RetryProvisioningTaskRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "请求参数错误")
			return
		}
	}

	task, err := h.provisioningService.RetryTask(c.Request.Context(), id, tenantID, userID, req.Reason)
	if err != nil {
		h.respondProvisioningError(c, err, "重试交付任务失败")
		return
	}
	common.Success(c, taskToResponse(task))
}

// ResolveProvisioningTask POST /api/v1/provisioning-tasks/:id/resolve
//
// 人工接管：自动重试不可能成功（凭据吊销/配额封禁）或资源已在平台外建好时，
// 由运维给出终态判定。note 必填，作为审计依据。
func (h *Handler) ResolveProvisioningTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的交付任务ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	var req dto.ResolveProvisioningTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "判定结果（delivered/failed）与依据说明均为必填")
		return
	}

	task, err := h.provisioningService.ResolveTaskManually(c.Request.Context(), id, tenantID, userID, req.Outcome, req.Note)
	if err != nil {
		h.respondProvisioningError(c, err, "人工判定交付任务失败")
		return
	}
	common.Success(c, taskToResponse(task))
}

// respondProvisioningError 把执行权冲突翻译成 409 + 可操作文案。
//
// 关键点：这些不是「服务器出错了」，而是「你这次操作和任务当前状态冲突」。
// 压成 500 会让前端重试、让运维以为系统故障，实际应该去看任务状态。
func (h *Handler) respondProvisioningError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrProvisioningTaskSucceeded):
		common.Fail(c, common.ConflictCode, "该交付任务已完成，不能重复执行（重复执行会造成重复交付）")
	case errors.Is(err, service.ErrProvisioningTaskInProgress):
		common.Fail(c, common.ConflictCode, "该交付任务正在执行中，请等待本次执行结束后再操作")
	case errors.Is(err, service.ErrProvisioningTaskAttemptsExhausted):
		common.Fail(c, common.ConflictCode, "该交付任务重试次数已达上限，请人工判定交付结果")
	case errors.Is(err, service.ErrProvisioningTaskNotRetryable):
		common.Fail(c, common.ConflictCode, "该交付任务当前状态不允许重试（仅失败任务可重试）")
	case errors.Is(err, service.ErrProvisioningTaskNotResolvable):
		common.Fail(c, common.ConflictCode, "该交付任务当前状态不允许人工判定（执行中或已完成）")
	default:
		common.FailWithErr(c, err, fallback)
	}
}

// taskToResponse 交付任务 → 响应 DTO（单一投影，五个读面共用）。
func taskToResponse(t *ent.ProvisioningTask) *dto.ProvisioningTaskResponse {
	return &dto.ProvisioningTaskResponse{
		ID:               t.ID,
		ServiceRequestID: t.ServiceRequestID,
		Provider:         t.Provider,
		ResourceType:     t.ResourceType,
		Status:           t.Status,
		Payload:          t.Payload,
		Result:           t.Result,
		ErrorMessage:     t.ErrorMessage,
		AttemptCount:     t.AttemptCount,
		MaxAttempts:      provisioning.MaxTaskAttempts,
		LastAttemptAt:    t.LastAttemptAt,
		ResolvedByUserID: t.ResolvedByUserID,
		ResolutionNote:   t.ResolutionNote,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}
