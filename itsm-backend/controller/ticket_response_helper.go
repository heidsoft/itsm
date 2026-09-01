package controller

import (
	"itsm-backend/dto"
	"itsm-backend/repository/ticket"
)

// ticketToResponse 将工单领域对象转换为响应 DTO。
// 原先定义在 ticket_controller.go（已随 controller→handlers 迁移删除），
// msp_controller 仍在消费，故保留在此共享 helper 文件中。
func ticketToResponse(t *ticket.Ticket) *dto.TicketResponse {
	if t == nil {
		return nil
	}
	resp := &dto.TicketResponse{
		ID:             t.ID,
		TicketNumber:   t.TicketNumber,
		Title:          t.Title,
		Description:    t.Description,
		Status:         string(t.Status),
		Priority:       string(t.Priority),
		Type:           string(t.Type),
		TicketTypeID:   0,
		TicketTypeCode: t.TicketTypeCode,
		TicketTypeName: t.TicketTypeName,
		FormFields:     t.FormFields,
		RequesterID:    t.RequesterID,
		TenantID:       t.TenantID,
		Version:        t.Version,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
	if t.AssigneeID != nil {
		resp.AssigneeID = *t.AssigneeID
	}
	if t.TicketTypeID != nil {
		resp.TicketTypeID = *t.TicketTypeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = *t.CategoryID
	}
	return resp
}
