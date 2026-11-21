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
		aiGrp.POST("/tickets/:id/analyze",
			// Phase A（2026-11-17）：补 ticket:read 双权限，与 investigate/incident analyze
			// 同口径——持 ai:read 但无 ticket:read 的角色此前可触发任意租户内工单分析。
			middleware.RequirePermission("ai", "read"),
			middleware.RequirePermission("ticket", "read"),
			h.AnalyzeTicket,
		)
		aiGrp.POST("/incidents/:id/analyze",
			middleware.RequirePermission("ai", "read"),
			middleware.RequirePermission("incident", "read"),
			h.AnalyzeIncident,
		)
		// 工单调查编排（triage → KB → CMDB → LLM 综合）：读取工单需要
		// ticket:read，调用 AI 工具链需要 ai:read，双权限仿 incident analyze。
		aiGrp.POST("/tickets/:id/investigate",
			middleware.RequirePermission("ai", "read"),
			middleware.RequirePermission("ticket", "read"),
			h.InvestigateTicket,
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
		// 预演：跑完真实判定链但不执行、不建待审批行（2026-11-21）。
		//
		// 权限用 ai:read：预演零业务副作用，只读权限者应当能判断自己「能不能写」；
		// 若按 ai:write 门控，会出现「无写权限者无法预演、只能真调一次才知道被拒」
		// 的悖论，而真调恰恰要建待审批行。
		//
		// 放在 /tools/execute 之后与静态段并列：Gin 静态段优先于 :id 参数段，
		// 与 GET /tools/invocations 的既有共存关系一致，不会劫持 GET /tools/:id。
		agentGrp.POST("/tools/dry-run", middleware.RequirePermission("ai", "read"), h.DryRunTool)
		agentGrp.GET("/tools/:id", middleware.RequirePermission("ai", "read"), h.GetToolInvocation)
		// 复核既有调用：按原发起主体 + 当前策略重新判定（2026-11-21）。
		//
		// 与 dry-run 的分工：dry-run 问「现在调这个工具会不会被允许」，
		// 本端点问「队列里这条已经排着的调用，现在还该放行吗」。
		// 审批效力取决于批准那一刻的判定，不是排队时的——这是两者不能合并的原因。
		//
		// 权限用 ai:read（只读判定、零副作用、不改审批状态）；真正的审批动作
		// 仍在 POST /tools/:id/approve 上受 ai:approve 门控。
		// 注意路径必须写在 /tools/:id 之后仍可共存：Gin 的静态段与参数段不冲突，
		// 但 :id 会匹配 "invocations"，故使用更长的静态前缀区分。
		agentGrp.POST("/tools/invocations/:id/review", middleware.RequirePermission("ai", "read"), h.ReviewToolInvocation)
		// 审批人待办列表：GET /api/v1/agent/tools/invocations?state=pending
		agentGrp.GET("/tools/invocations", middleware.RequirePermission("ai", "read"), h.ListToolInvocations)
		// 运行轮次证据链：GET /api/v1/agent/runs/:runId/invocations
		//
		// 审计回放的唯一读面：一次 Agent 运行（一轮对话 / 一次调查 / 一次 MCP
		// 会话）的全部工具调用按时间正序取出，含每步的策略版本与审批态。
		// 复用 ai:read 而不新增权限码——它读的是同域同类数据，只是换了
		// 聚合维度；新增码还会撞 dead-codes 基线（只减不增）。
		agentGrp.GET("/runs/:runId/invocations", middleware.RequirePermission("ai", "read"), h.ListRunInvocations)
		// 控制面聚合读面：GET /api/v1/agent/control-plane
		//
		// 一次请求返回工具可见面 / 待批队列 / 策略配置面 / 最近拒绝。
		// 与上面几个读面的关系是「聚合」而非「替代」：它内部复用
		// HasResourcePermission 与 ListToolInvocations 的真源，不新增判定。
		//
		// 权限用 ai:read：纯只读聚合，零副作用、不改任何状态。
		// 放在 /runs/:runId 之后、静态段 control-plane 与 /tools/:id 不冲突
		// （不同前缀），不会劫持任何既有路由。
		agentGrp.GET("/control-plane", middleware.RequirePermission("ai", "read"), h.ControlPlaneOverview)
		// Phase A（2026-11-17）：审批权从 ai:write 拆出为 ai:approve。
		// end_user 此前持 ai:write 即可审批 AI 工具调用，违背职责分立；
		// handler 内还有「审批人须持目标工具 resource:action」二次校验。
		agentGrp.POST("/tools/:id/approve", middleware.RequirePermission("ai", "approve"), h.ApproveTool)
	}
}
