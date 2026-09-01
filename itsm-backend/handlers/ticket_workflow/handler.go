// Package ticketworkflow 是工单流转域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_workflow_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketWorkflowService 承载，本包只做参数解析与响应封装。
// GetTicketWorkflowHistory 沿用旧实现的裸 SQL 查询（含 P1-08 的 NullString 修复），
// 后续演进应下沉至 service 层。
package ticketworkflow

import (
	"database/sql"

	"itsm-backend/service"

	"go.uber.org/zap"
)

// Handler 工单流转 HTTP handler
type Handler struct {
	workflowService *service.TicketWorkflowService
	db              *sql.DB
	logger          *zap.SugaredLogger
}

// NewHandler 创建工单流转 handler 实例
func NewHandler(workflowService *service.TicketWorkflowService, db *sql.DB, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		workflowService: workflowService,
		db:              db,
		logger:          logger,
	}
}
