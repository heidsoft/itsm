// Package provisioning — 交付任务 handler.
// 迁移自 controller/provisioning_controller.go，保持原有 API 契约不变。
package provisioning

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 ProvisioningService 依赖.
type Handler struct {
	provisioningService *service.ProvisioningService
	logger             *zap.SugaredLogger
}

// NewHandler 构造 provisioning Handler.
func NewHandler(provisioningService *service.ProvisioningService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		provisioningService: provisioningService,
		logger:             logger,
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
		common.Fail(c, common.BadRequestCode, "创建交付任务失败")
		return
	}

	resp := &dto.ProvisioningTaskResponse{
		ID:               task.ID,
		ServiceRequestID: task.ServiceRequestID,
		Provider:         task.Provider,
		ResourceType:     task.ResourceType,
		Status:           task.Status,
		Payload:          task.Payload,
		Result:           task.Result,
		ErrorMessage:     task.ErrorMessage,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	}
	common.Success(c, dto.StartProvisioningResponse{Task: resp})
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
		common.Fail(c, common.InternalErrorCode, "获取交付任务列表失败")
		return
	}
	out := make([]dto.ProvisioningTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, dto.ProvisioningTaskResponse{
			ID:               t.ID,
			ServiceRequestID: t.ServiceRequestID,
			Provider:         t.Provider,
			ResourceType:     t.ResourceType,
			Status:           t.Status,
			Payload:          t.Payload,
			Result:           t.Result,
			ErrorMessage:     t.ErrorMessage,
			CreatedAt:        t.CreatedAt,
			UpdatedAt:        t.UpdatedAt,
		})
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
		common.Fail(c, common.InternalErrorCode, "执行交付任务失败")
		return
	}
	common.Success(c, dto.ProvisioningTaskResponse{
		ID:               task.ID,
		ServiceRequestID: task.ServiceRequestID,
		Provider:         task.Provider,
		ResourceType:     task.ResourceType,
		Status:           task.Status,
		Payload:          task.Payload,
		Result:           task.Result,
		ErrorMessage:     task.ErrorMessage,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	})
}
