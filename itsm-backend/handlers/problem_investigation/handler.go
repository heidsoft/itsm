// Package probleminvestigation 是问题调查域的 HTTP handler 层（域切片架构）。
// 自 controller/problem_investigation_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.ProblemInvestigationService 承载，本包只做参数解析与响应封装。
package probleminvestigation

import (
	"itsm-backend/service"

	"go.uber.org/zap"
)

// Handler 问题调查 HTTP handler
type Handler struct {
	logger     *zap.SugaredLogger
	invService *service.ProblemInvestigationService
}

// NewHandler 创建问题调查 handler 实例
func NewHandler(logger *zap.SugaredLogger, invService *service.ProblemInvestigationService) *Handler {
	return &Handler{
		logger:     logger,
		invService: invService,
	}
}
