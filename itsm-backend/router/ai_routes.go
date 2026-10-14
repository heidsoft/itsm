package router

import (
	aiHandler "itsm-backend/handlers/ai"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAIRoutes 注册 AI & Analytics 相关路由。
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
		// 审计闭环：采纳/拒绝翻转（handler 自 2026-10 存在但漏注册路由，前端 PATCH 恒 404）
		aiGrp.PATCH("/audit-logs/:id", middleware.RequirePermission("ai", "write"), h.UpdateAuditAccepted)
		aiGrp.POST("/triage", middleware.RequirePermission("ai", "read"), h.Triage)
		// RAG endpoints
		// KnowledgeSearch 用 ShouldBindJSON 从 body 读 {query,limit,type}，GET 不解析 body，因此必须注册为 POST。
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
