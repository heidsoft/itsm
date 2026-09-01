// Package approval 是审批工作流域的 HTTP handler 层（域切片架构）。
// 自 controller/approval_controller.go 迁移而来（2026-09-01），
// 业务逻辑仍由 service.ApprovalService 承载，本包只做参数解析与响应封装。
package approval

import (
	"itsm-backend/service"
)

// Handler 审批工作流 HTTP handler
type Handler struct {
	approvalService *service.ApprovalService
}

// NewHandler 创建审批工作流 handler 实例
func NewHandler(approvalService *service.ApprovalService) *Handler {
	return &Handler{approvalService: approvalService}
}
