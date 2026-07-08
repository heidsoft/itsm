package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketcc"

	"go.uber.org/zap"
)

// CCTaskHandler 抄送服务任务处理器
type CCTaskHandler struct {
	HandlerBase
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCCTaskHandler 创建抄送处理器
func NewCCTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *CCTaskHandler {
	return &CCTaskHandler{
		client: client,
		logger: logger,
	}
}

// GetTaskType 返回任务类型
func (h *CCTaskHandler) GetTaskType() string {
	return "cc_task"
}

// GetHandlerID 返回处理器标识
func (h *CCTaskHandler) GetHandlerID() string {
	return "cc_handler"
}

// Execute 执行抄送服务任务
func (h *CCTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取参数
	ticketID := GetIntFromVars(variables, "ticket_id")
	ccUserIDs := GetIntSliceFromVars(variables, "cc_user_ids")
	addedBy := GetIntFromVars(variables, "added_by")
	tenantID := GetIntFromVars(variables, "tenant_id")
	

	if ticketID == 0 {
		return nil, fmt.Errorf("工单ID不能为空")
	}
	if len(ccUserIDs) == 0 {
		return nil, fmt.Errorf("抄送人不能为空")
	}
	if tenantID == 0 {
		return nil, fmt.Errorf("租户ID不能为空")
	}

	h.logger.Infow(
		"Executing CC task via BPMN",
		"ticket_id", ticketID,
		"cc_user_ids", ccUserIDs,
		"added_by", addedBy,
		"tenant_id", tenantID,
	)

	// 添加抄送人
	var addedUsers []int
	for _, ccUserID := range ccUserIDs {
		// 检查是否已存在抄送记录
		exists, err := h.client.TicketCC.Query().
			Where(ticketcc.TicketID(ticketID),
				ticketcc.UserID(ccUserID),
				ticketcc.TenantID(tenantID)).
			Exist(ctx)
		if err != nil {
			h.logger.Warnw("Failed to check CC existence", "error", err, "user_id", ccUserID)
			continue
		}
		if !exists {
			err = h.client.TicketCC.Create().
				SetTicketID(ticketID).
				SetUserID(ccUserID).
				SetAddedBy(addedBy).
				SetTenantID(tenantID).
				SetAddedAt(time.Now()).
				SetIsActive(true).
				Exec(ctx)
			if err != nil {
				h.logger.Warnw("Failed to add CC user", "error", err, "user_id", ccUserID)
				continue
			}
			addedUsers = append(addedUsers, ccUserID)
		}
	}

	// 发送通知给抄送人
	// 这里可以调用通知服务发送站内信、邮件等
	for _, userID := range addedUsers {
		h.logger.Infow("Sending CC notification to user", "user_id", userID, "ticket_id", ticketID)
		// 实际项目中这里调用通知服务
	}

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("已成功添加 %d 位抄送人", len(addedUsers)),
		OutputVars: map[string]interface{}{
			"added_cc_users": addedUsers,
		},
	}, nil
}

// Validate 验证配置
func (h *CCTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// 确保 CCTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*CCTaskHandler)(nil)
