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

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 ProvisioningService 依赖.
type Handler struct {
	provisioningService ProvisioningTaskService
	logger              *zap.SugaredLogger
}

// NewHandler 构造 provisioning Handler.
func NewHandler(provisioningService ProvisioningTaskService, logger *zap.SugaredLogger) *Handler {
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
		// not-found → 404、状态不符 → 409 且带领域文案，其余仍为 500。
		// 用 RespondError 而非 FailWithErr：状态守卫的 BusinessError 文案
		// （「当前状态不允许启动交付（需要 security_approved）」）必须透出——
		// FailWithErr 会用兜底文案「创建交付任务失败」把它整段吞掉，运维拿到
		// 409 却不知道该去补审批链还是改状态，只能翻后端日志。未分类的错误
		// （事务/DB 故障）safeMsg 为空，仍回落兜底文案，不泄内部细节。
		common.RespondError(c, err, "创建交付任务失败")
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
//
// 另两类「配置不完整 / 云侧永久拒绝」此前落在 default 分支被兜底成 5001，
// 前端只能看到「API Error: 500 Internal Server Error」这类原始报文（prod 实测 O1）。
// 它们同样是**可行动的业务结果**：配置补齐或人工判定后即可继续，不该让运维
// 以为系统故障、也不该让前端把原始 500 报文直接抛给用户。
func (h *Handler) respondProvisioningError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, dto.ErrProvisioningTaskSucceeded):
		common.Fail(c, common.ConflictCode, "该交付任务已完成，不能重复执行（重复执行会造成重复交付）")
	case errors.Is(err, dto.ErrProvisioningTaskInProgress):
		common.Fail(c, common.ConflictCode, "该交付任务正在执行中，请等待本次执行结束后再操作")
	case errors.Is(err, dto.ErrProvisioningTaskAttemptsExhausted):
		common.Fail(c, common.ConflictCode, "该交付任务重试次数已达上限，请人工判定交付结果")
	case errors.Is(err, dto.ErrProvisioningTaskNotRetryable):
		common.Fail(c, common.ConflictCode, "该交付任务当前状态不允许重试（仅失败任务可重试）")
	case errors.Is(err, dto.ErrProvisioningTaskNotResolvable):
		common.Fail(c, common.ConflictCode, "该交付任务当前状态不允许人工判定（执行中或已完成）")
	case errors.Is(err, dto.ErrProvisioningNotExecutable):
		// 交付配置不完整（缺地域/规格/云账号或凭据未装配）：重试不会成功，
		// 必须由管理员补齐配置后人工重试。真实原因已写入任务 error_message，
		// 这里给可行动文案而不透出原始报文。
		common.Fail(c, common.ConflictCode, "交付配置不完整，自动重试不会成功：请检查服务目录可用地域、云账号与凭据配置后重试")
	case errors.Is(err, dto.ErrProvisioningPermanentlyFailed):
		// 云侧业务拒绝（余额不足/配额封禁/参数不存在）：重试期望收益为零，
		// 处置权交回运维（充值或改参数后人工重试）。
		common.Fail(c, common.ConflictCode, "云侧已拒绝该交付请求且自动重试不会成功，请修正账户或参数后人工重试")
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
