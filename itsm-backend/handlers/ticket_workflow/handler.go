// Package ticketworkflow 是工单流转域的 HTTP handler 层（域切片架构）。
// 业务逻辑仍由 service.TicketWorkflowService 承载，本包只做参数解析与响应封装。
// GetTicketWorkflowHistory 沿用旧实现的裸 SQL 查询，
// 后续演进应下沉至 service 层。
package ticket_workflow

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
