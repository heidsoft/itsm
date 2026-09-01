// Package probleminvestigation 是问题调查域的 HTTP handler 层（域切片架构）。
// 自 controller/problem_investigation_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.ProblemInvestigationService 承载，本包只做参数解析与响应封装。
// 知识沉淀两个端点（CreateKnowledgeArticle/GetProblemKnowledgeArticles）沿用旧实现，
// 直接持有 ent.Client（与迁移前行为一致，后续演进应下沉至 service 层）。
package probleminvestigation

import (
	"itsm-backend/ent"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// Handler 问题调查 HTTP handler
type Handler struct {
	logger     *zap.SugaredLogger
	invService *service.ProblemInvestigationService
	entClient  *ent.Client
}

// NewHandler 创建问题调查 handler 实例
func NewHandler(logger *zap.SugaredLogger, invService *service.ProblemInvestigationService, entClient *ent.Client) *Handler {
	return &Handler{
		logger:     logger,
		invService: invService,
		entClient:  entClient,
	}
}
