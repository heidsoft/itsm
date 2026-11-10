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
		common.Fail(c, common.ParamErrorCode, "Invalid service request ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Tenant information is missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "User not authenticated")
		return
	}

	task, err := h.provisioningService.CreateTaskFromServiceRequest(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// not-found → 404、状态不符 → 409 且带领域文案，其余仍为 500。
		// 用 RespondError 而非 FailWithErr：状态守卫的 BusinessError 文案
		// （「当前状态不允许启动交付（需要 security_approved）」）必须透出——
		// FailWithErr 会用兜底文案「创建交付任务失败」把它整段吞掉，运维拿到
		// 409 却不知道该去补审批链还是改状态，只能翻后端日志。未分类的错误
		// （事务/DB 故障）safeMsg 为空，仍回落兜底文案，不泄内部细节。
		common.RespondError(c, err, "Failed to create delivery task")
		return
	}

	common.Success(c, dto.StartProvisioningResponse{Task: taskToResponse(task)})
}

// ListProvisioningTasks GET /api/v1/service-requests/:id/provisioning-tasks
func (h *Handler) ListProvisioningTasks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid service request ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Tenant information is missing")
		return
	}

	tasks, err := h.provisioningService.ListTasksByServiceRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Failed to obtain delivery task list")
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
		common.Fail(c, common.ParamErrorCode, "Invalid delivery task ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Tenant information is missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "User not authenticated")
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
		common.Fail(c, common.ParamErrorCode, "Invalid delivery task ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Tenant information is missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "User not authenticated")
		return
	}

	// body 可选：无 body 时按「未说明原因的重试」处理，不因缺字段拒绝一次合法重试。
	var req dto.RetryProvisioningTaskRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Wrong request parameters")
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
		common.Fail(c, common.ParamErrorCode, "Invalid delivery task ID")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Tenant information is missing")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "User not authenticated")
		return
	}

	var req dto.ResolveProvisioningTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Both the judgment result (delivered/failed) and the basis description are required.")
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
		common.Fail(c, common.ConflictCode, "The delivery task has been completed and cannot be repeated (repeated execution will cause repeated delivery)")
	case errors.Is(err, service.ErrProvisioningTaskInProgress):
		common.Fail(c, common.ConflictCode, "The delivery task is being executed, please wait until the execution is completed before proceeding.")
	case errors.Is(err, service.ErrProvisioningTaskAttemptsExhausted):
		common.Fail(c, common.ConflictCode, "The number of retries for this delivery task has reached the upper limit. Please manually determine the delivery result.")
	case errors.Is(err, service.ErrProvisioningTaskNotRetryable):
		common.Fail(c, common.ConflictCode, "The current status of this delivery task does not allow retry (only failed tasks can be retried)")
	case errors.Is(err, service.ErrProvisioningTaskNotResolvable):
		common.Fail(c, common.ConflictCode, "The current status of this delivery task does not allow manual judgment (executing or completed)")
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
