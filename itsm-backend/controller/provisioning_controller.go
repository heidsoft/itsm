package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// ProvisioningController M2：交付任务接口（骨架）
type ProvisioningController struct {
	provisioningService *service.ProvisioningService
}

func NewProvisioningController(provisioningService *service.ProvisioningService) *ProvisioningController {
	return &ProvisioningController{provisioningService: provisioningService}
}

// StartProvisioning 启动交付（创建任务并把 SR 置为 provisioning）
// @Summary 启动交付
// @Tags 交付
// @Produce json
// @Param id path int true "服务请求ID"
// @Success 200 {object} common.Response{data=dto.StartProvisioningResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/provision [post]
func (pc *ProvisioningController) StartProvisioning(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的服务请求ID")
		return
	}
	tenantIDAny, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userIDAny, ok := c.Get("user_id")
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}
	tenantID := tenantIDAny.(int)
	userID := userIDAny.(int)

	task, err := pc.provisioningService.CreateTaskFromServiceRequest(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		common.Fail(c, common.BadRequestCode, err.Error())
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

// ListProvisioningTasks 列出某个 SR 的交付任务
// @Summary 获取交付任务列表
// @Tags 交付
// @Produce json
// @Param id path int true "服务请求ID"
// @Success 200 {object} common.Response{data=[]dto.ProvisioningTaskResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/service-requests/{id}/provisioning-tasks [get]
func (pc *ProvisioningController) ListProvisioningTasks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的服务请求ID")
		return
	}
	tenantIDAny, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	tenantID := tenantIDAny.(int)

	tasks, err := pc.provisioningService.ListTasksByServiceRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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

// ExecuteProvisioningTask 执行交付任务（Stub）
// @Summary 执行交付任务
// @Tags 交付
// @Produce json
// @Param id path int true "交付任务ID"
// @Success 200 {object} common.Response{data=dto.ProvisioningTaskResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Security BearerAuth
// @Router /api/v1/provisioning-tasks/{id}/execute [post]
func (pc *ProvisioningController) ExecuteProvisioningTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的交付任务ID")
		return
	}
	tenantIDAny, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userIDAny, ok := c.Get("user_id")
	if !ok {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}
	tenantID := tenantIDAny.(int)
	userID := userIDAny.(int)

	task, err := pc.provisioningService.ExecuteTask(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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


