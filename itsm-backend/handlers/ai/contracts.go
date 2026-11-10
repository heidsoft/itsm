package ai

import (
	"context"

	"itsm-backend/dto"
)

// SkillRegistrar 是内置技能注册所需的窄契约。
//
// 共享层具体注册表隐式满足本接口：Register 的入参 Skill 在两边是同一个
// dto 接口（service 侧为别名），因此组合根无需改动装配代码。
type SkillRegistrar interface {
	Register(skill dto.Skill) error
}

// ToolCatalog 是 AI 域工具执行所需的窄契约：只读执行、按名取定义、
// 全量/按租户动态化工具清单。共享层具体注册表隐式满足本接口
// （GetTool 返回 *dto.ToolDefinition，与共享层同名类型是同一身份）。
type ToolCatalog interface {
	Execute(ctx context.Context, tenantID int, name string, args map[string]interface{}) (interface{}, error)
	GetTool(name string) *dto.ToolDefinition
	ListTools() []dto.ToolDefinition
	ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition
}

// DeepAnalyticsProvider 是 AI 域深度分析所需的窄契约。
type DeepAnalyticsProvider interface {
	GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int, viewer dto.DashboardViewer) (*dto.DeepAnalyticsResponse, error)
}

// TrendPredictor 是 AI 域趋势预测（legacy 兜底路）所需的窄契约。
type TrendPredictor interface {
	GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, viewer dto.DashboardViewer) (*dto.TrendPredictionResponse, error)
}

// SLAForecastRunner 是 AI-Native SLA 预测技能所需的窄契约。
type SLAForecastRunner interface {
	Execute(ctx context.Context, input *dto.ForecastInput) (*dto.ForecastOutput, error)
}

// TriageAdvisor 是 AI 域分诊所需的窄契约（Guidance → LLM → keyword 三级链）。
type TriageAdvisor interface {
	SuggestForTenant(ctx context.Context, title, description string, tenantID int) dto.TriageResult
}

// RootCauseAnalyzer 是 AI 域根因分析所需的窄契约。
type RootCauseAnalyzer interface {
	AnalyzeIncident(ctx context.Context, incidentID, tenantID int) (*dto.IncidentAIAnalysisResponse, error)
	AnalyzeTicket(ctx context.Context, ticketID, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error)
	GetAnalysisReport(ctx context.Context, ticketID, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error)
	SummarizeTicket(ctx context.Context, ticketID, tenantID int) (map[string]interface{}, error)
}

// AITelemetryFacade 是 AI 域遥测所需的窄契约：评估报告、指标聚合、
// 审计底账读面、反馈写入与采纳翻转；RecordDecisionAudit 供 NewService
// 默认装配 auditLedger（与 AuditLedgerWriter 单方法签名一致）。
type AITelemetryFacade interface {
	Evaluate(ctx context.Context, tenantID int, lookbackDays int) (*dto.AIEvaluationReport, error)
	GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (*dto.AIMetrics, error)
	ListAuditLogs(ctx context.Context, tenantID int, page, pageSize int, kind string, lookbackDays int) ([]dto.AIAuditEntry, int, error)
	SaveFeedback(ctx context.Context, tenantID, userID int, reqID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string, accepted *bool) error
	UpdateAuditAccepted(ctx context.Context, tenantID int, id int64, accepted bool) error
	RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)
}

// Summarizer 是 AI 域工单摘要所需的窄契约（带溯源元数据）。
type Summarizer interface {
	SummarizeWithMetadata(ctx context.Context, text string, maxLen int) (*dto.SummaryWithMetadata, error)
}
