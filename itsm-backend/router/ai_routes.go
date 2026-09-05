package router

import (
	aiHandler "itsm-backend/handlers/ai"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAIRoutes 注册 AI & Analytics 相关路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
func SetupAIRoutes(tenant *gin.RouterGroup, h *aiHandler.Handler) {
	aiGrp := tenant.Group("/ai")
	{
		aiGrp.POST("/chat", middleware.RequirePermission("ai", "read"), h.Chat)
		aiGrp.POST("/chat/stream", middleware.RequirePermission("ai", "read"), h.ChatStream)
		aiGrp.GET("/conversations", middleware.RequirePermission("ai", "read"), h.ListConversations)
		aiGrp.GET("/conversations/:id", middleware.RequirePermission("ai", "read"), h.GetConversation)
		aiGrp.DELETE("/conversations/:id", middleware.RequirePermission("ai", "write"), h.DeleteConversation)
		aiGrp.GET("/analysis-results", middleware.RequirePermission("ai", "read"), h.ListAIAnalysisResults)
		aiGrp.GET("/analysis-results/:id", middleware.RequirePermission("ai", "read"), h.GetAIAnalysisResult)
		aiGrp.DELETE("/analysis-results/:id", middleware.RequirePermission("ai", "write"), h.DeleteAIAnalysisResult)
		aiGrp.POST("/analytics", middleware.RequirePermission("ai", "read"), h.GetDeepAnalytics)
		aiGrp.POST("/predictions", middleware.RequirePermission("ai", "read"), h.GetTrendPrediction)
		aiGrp.POST("/tickets/:id/analyze", middleware.RequirePermission("ai", "read"), h.AnalyzeTicket)
		aiGrp.POST("/incidents/:id/analyze",
			middleware.RequirePermission("ai", "read"),
			middleware.RequirePermission("incident", "read"),
			h.AnalyzeIncident,
		)
		aiGrp.POST("/feedback", middleware.RequirePermission("ai", "write"), h.SaveFeedback)
		aiGrp.POST("/audit", middleware.RequirePermission("ai", "write"), h.RecordAudit)
		aiGrp.GET("/metrics", middleware.RequirePermission("ai", "read"), h.GetMetrics)
		// AI 评估报告（按场景有用率 / 置信度校准 / 平台 LLM 统计）
		aiGrp.GET("/evaluation", middleware.RequirePermission("ai", "read"), h.GetEvaluation)
		// AI 审计日志（ai_audit 记录分页查询）
		aiGrp.GET("/audit-logs", middleware.RequirePermission("ai", "read"), h.GetAuditLogs)
		aiGrp.POST("/triage", middleware.RequirePermission("ai", "read"), h.Triage)
		// RAG endpoints
		// Bug fix (2026-08-15): handler KnowledgeSearch uses ShouldBindJSON
		// to read {query,limit,type} from a request body, but the route was
		// registered as GET. Gin would refuse to parse a body on GET, so the
		// endpoint always returned ParamError. Promote to POST to match the
		// handler's binding contract.
		aiGrp.POST("/rag/search", middleware.RequirePermission("ai", "read"), h.KnowledgeSearch)
		// AI 工单智能创建
		aiGrp.POST("/ticket/create", middleware.RequirePermission("ticket", "create"), h.CreateTicketByAI)
	}

	agentGrp := tenant.Group("/agent")
	{
		agentGrp.GET("/tools", middleware.RequirePermission("ai", "read"), h.ListTools)
		agentGrp.POST("/tools/execute", middleware.RequirePermission("ai", "read"), h.ExecuteTool)
		agentGrp.GET("/tools/:id", middleware.RequirePermission("ai", "read"), h.GetToolInvocation)
		// 审批人待办列表：GET /api/v1/agent/tools/invocations?state=pending
		agentGrp.GET("/tools/invocations", middleware.RequirePermission("ai", "read"), h.ListToolInvocations)
		agentGrp.POST("/tools/:id/approve", middleware.RequirePermission("ai", "write"), h.ApproveTool)
	}
}
