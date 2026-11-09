package problem_investigation

// 域内窄契约：问题调查、根因分析、解决方案与调查步骤，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

type ProblemInvestigationService interface {
	CreateProblemInvestigation(ctx context.Context, req *dto.CreateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error)
	UpdateProblemInvestigation(ctx context.Context, investigationID int, req *dto.UpdateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error)
	GetProblemInvestigation(ctx context.Context, investigationID, tenantID int) (*dto.ProblemInvestigationResponse, error)
	GetProblemInvestigationSummary(ctx context.Context, problemID, tenantID int) (*dto.ProblemInvestigationSummaryResponse, error)
	CreateInvestigationStep(ctx context.Context, req *dto.CreateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error)
	UpdateInvestigationStep(ctx context.Context, stepID int, req *dto.UpdateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error)
	CreateRootCauseAnalysis(ctx context.Context, req *dto.CreateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error)
	UpdateRootCauseAnalysis(ctx context.Context, id int, req *dto.UpdateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error)
	CreateProblemSolution(ctx context.Context, req *dto.CreateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error)
	UpdateProblemSolution(ctx context.Context, id int, req *dto.UpdateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error)
	CreateKnowledgeArticle(ctx context.Context, tenantID, authorID int, req *dto.CreateProblemKnowledgeArticleRequest) (*ent.KnowledgeArticle, error)
}
