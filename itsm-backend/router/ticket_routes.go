package router

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupTicketRoutes 注册工单主路由（含关联子域：视图/评论/附件/审批/流转/自动化/分配/评分/依赖/通知，以及顶层分析别名与 AI 工单总结）。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// 签名沿用 SetupCMDBRoutes 先例直接接收 *RouterConfig，避免十余个 handler 平铺传参。
func SetupTicketRoutes(tenant *gin.RouterGroup, config *RouterConfig) {
	tickets := tenant.Group("/tickets")
	{
		tickets.GET("", middleware.RequirePermission("ticket", "read"), config.TicketHandler.ListTickets)
		tickets.POST("", middleware.RequirePermission("ticket", "create"), config.TicketHandler.CreateTicket)

		if config.TicketViewHandler != nil {
			tickets.GET("/views", middleware.RequirePermission("view", "read"), config.TicketViewHandler.ListTicketViews)
			tickets.POST("/views", middleware.RequirePermission("view", "create"), config.TicketViewHandler.CreateTicketView)
			tickets.GET("/views/:id", middleware.RequirePermission("view", "read"), config.TicketViewHandler.GetTicketView)
			tickets.PUT("/views/:id", middleware.RequirePermission("view", "update"), config.TicketViewHandler.UpdateTicketView)
			tickets.DELETE("/views/:id", middleware.RequirePermission("view", "delete"), config.TicketViewHandler.DeleteTicketView)
		}

		tickets.GET("/search", middleware.RequirePermission("ticket", "read"), config.TicketHandler.SearchTickets)
		tickets.GET("/stats", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetTicketStats)
		tickets.GET("/overdue", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetOverdueTickets)
		tickets.POST("/export", middleware.RequirePermission("ticket", "export"), config.TicketHandler.ExportTickets)
		tickets.POST("/batch-delete", middleware.RequirePermission("ticket", "delete"), config.TicketHandler.BatchDeleteTickets)
		if config.TicketWorkflowHandler != nil {
			tickets.GET("/cc/my", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.ListMyCCRecords)
		}

		// 工单模板
		tickets.GET("/templates", middleware.RequirePermission("template", "read"), config.TicketHandler.GetTicketTemplates)
		tickets.GET("/templates/categories", middleware.RequirePermission("template", "read"), config.TicketHandler.GetTicketTemplateCategories)
		tickets.POST("/templates", middleware.RequirePermission("template", "create"), config.TicketHandler.CreateTicketTemplate)
		tickets.GET("/templates/:id", middleware.RequirePermission("template", "read"), config.TicketHandler.GetTicketTemplate)
		tickets.PUT("/templates/:id", middleware.RequirePermission("template", "update"), config.TicketHandler.UpdateTicketTemplate)
		tickets.PATCH("/templates/:id/status", middleware.RequirePermission("template", "update"), config.TicketHandler.UpdateTicketTemplateStatus)
		tickets.POST("/templates/:id/copy", middleware.RequirePermission("template", "create"), config.TicketHandler.CopyTicketTemplate)
		tickets.DELETE("/templates/:id", middleware.RequirePermission("template", "delete"), config.TicketHandler.DeleteTicketTemplate)

		tickets.POST("/:id/escalate", middleware.RequirePermission("ticket", "escalate"), config.TicketHandler.EscalateTicket)
		tickets.GET("/:id/history", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetTicketActivity)
		tickets.PUT("/:id/sla/pause", middleware.RequirePermission("ticket", "update"), config.TicketHandler.PauseSLA)
		tickets.PUT("/:id/sla/resume", middleware.RequirePermission("ticket", "update"), config.TicketHandler.ResumeSLA)
		tickets.GET("/types", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{"types": []gin.H{
				{"id": 1, "name": "故障工单", "code": "incident"},
				{"id": 2, "name": "服务请求", "code": "service_request"},
				{"id": 3, "name": "变更工单", "code": "change"},
				{"id": 4, "name": "问题工单", "code": "problem"},
				{"id": 5, "name": "综合工单", "code": "ticket"},
				{"id": 6, "name": "持续改进", "code": "improvement"},
			}, "total": 6})
		})
		tickets.GET("/:id", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetTicket)
		tickets.PUT("/:id", middleware.RequirePermission("ticket", "update"), config.TicketHandler.UpdateTicket)
		// REST 契约：部分更新使用 PATCH（与 PUT 共用同一 handler，DTO 指针字段区分未传/传零值）
		tickets.PATCH("/:id", middleware.RequirePermission("ticket", "update"), config.TicketHandler.UpdateTicket)
		tickets.PUT("/:id/status", middleware.RequirePermission("ticket", "update"), config.TicketHandler.UpdateTicketStatus)
		tickets.DELETE("/:id", middleware.RequirePermission("ticket", "delete"), config.TicketHandler.DeleteTicket)
		tickets.POST("/:id/assign", middleware.RequirePermission("ticket", "assign"), config.TicketHandler.AssignTicket)
		tickets.POST("/:id/resolve", middleware.RequirePermission("ticket", "update"), config.TicketHandler.ResolveTicket)
		tickets.POST("/:id/close", middleware.RequirePermission("ticket", "update"), config.TicketHandler.CloseTicket)

		// 工单SLA信息
		tickets.GET("/:id/sla", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetTicketSLAInfo)

		// 子任务管理
		tickets.GET("/:id/subtasks", middleware.RequirePermission("ticket", "read"), config.TicketHandler.GetSubtasks)
		tickets.POST("/:id/subtasks", middleware.RequirePermission("ticket", "create"), config.TicketHandler.CreateSubtask)
		tickets.PATCH("/:id/subtasks/:subtask_id", middleware.RequirePermission("ticket", "update"), config.TicketHandler.UpdateSubtask)
		tickets.DELETE("/:id/subtasks/:subtask_id", middleware.RequirePermission("ticket", "delete"), config.TicketHandler.DeleteSubtask)

		// 工单关联（已接入 TicketAssociationService）
		if config.TicketAssociationService != nil {
			tickets.GET("/:id/relations", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				// URL :id 必须通过 c.Param 读取；c.GetInt 只能拿到 context store 里的
				// tenant_id/user_id/role 等键值，URL 参数不在该 store 中。
				ticketID, err := strconv.Atoi(c.Param("id"))
				if err != nil || ticketID == 0 {
					common.Fail(c, common.ParamErrorCode, "invalid ticket id")
					return
				}
				deps, err := config.TicketAssociationService.GetTicketDependencies(c.Request.Context(), ticketID)
				if err != nil {
					common.Fail(c, common.InternalErrorCode, err.Error())
					return
				}
				common.Success(c, gin.H{
					"parentChain":    deps.ParentChain,
					"childrenTree":   deps.ChildrenTree,
					"relatedTickets": deps.RelatedTickets,
				})
			})
			tickets.GET("/:id/relations/stats", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				ticketID, err := strconv.Atoi(c.Param("id"))
				if err != nil || ticketID == 0 {
					common.Fail(c, common.ParamErrorCode, "invalid ticket id")
					return
				}
				deps, err := config.TicketAssociationService.GetTicketDependencies(c.Request.Context(), ticketID)
				if err != nil {
					common.Fail(c, common.InternalErrorCode, err.Error())
					return
				}
				common.Success(c, gin.H{
					"totalRelations":  len(deps.RelatedTickets),
					"relationsByType": gin.H{},
					"inboundCount":    len(deps.RelatedTickets),
					"outboundCount":   len(deps.RelatedTickets),
					"parentCount":     len(deps.ParentChain),
					"childrenCount":   len(deps.ChildrenTree),
					"blockedByCount":  0,
					"blockingCount":   0,
					"relatedCount":    len(deps.RelatedTickets),
					"duplicateCount":  0,
				})
			})
			// 工单→配置项反向查询：复用 TicketAssociationService.GetConfigurationItems
			// （AI 本体链路落地后，工单详情页可展示其影响的配置项）
			tickets.GET("/:id/configuration-items", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				ticketID, err := strconv.Atoi(c.Param("id"))
				if err != nil || ticketID == 0 {
					common.Fail(c, common.ParamErrorCode, "invalid ticket id")
					return
				}
				items, err := config.TicketAssociationService.GetConfigurationItems(c.Request.Context(), ticketID)
				if err != nil {
					common.Fail(c, common.InternalErrorCode, err.Error())
					return
				}
				common.Success(c, items)
			})
		} else {
			// 兼容：服务未初始化时返回空
			tickets.GET("/:id/relations", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				common.Success(c, []gin.H{})
			})
			tickets.GET("/:id/relations/stats", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				common.Success(c, gin.H{
					"totalRelations":  0,
					"relationsByType": gin.H{},
					"inboundCount":    0,
					"outboundCount":   0,
					"parentCount":     0,
					"childrenCount":   0,
					"blockedByCount":  0,
					"blockingCount":   0,
					"relatedCount":    0,
					"duplicateCount":  0,
				})
			})
		}

		// 评论
		if config.TicketCommentHandler != nil {
			tickets.GET("/:id/comments", middleware.RequirePermission("ticket", "read"), config.TicketCommentHandler.ListTicketComments)
			tickets.POST("/:id/comments", middleware.RequirePermission("ticket", "create"), config.TicketCommentHandler.CreateTicketComment)
			tickets.PUT("/:id/comments/:comment_id", middleware.RequirePermission("ticket", "update"), config.TicketCommentHandler.UpdateTicketComment)
			tickets.DELETE("/:id/comments/:comment_id", middleware.RequirePermission("ticket", "delete"), config.TicketCommentHandler.DeleteTicketComment)
		}

		// 附件
		if config.TicketAttachmentHandler != nil {
			tickets.GET("/:id/attachments", middleware.RequirePermission("ticket", "read"), config.TicketAttachmentHandler.ListTicketAttachments)
			tickets.POST("/:id/attachments", middleware.RequirePermission("ticket", "create"), config.TicketAttachmentHandler.UploadAttachment)
			tickets.DELETE("/:id/attachments/:attachment_id", middleware.RequirePermission("ticket", "delete"), config.TicketAttachmentHandler.DeleteAttachment)
		}

		// 审批流程管理
		if config.ApprovalHandler != nil {
			tickets.POST("/approval/submit", middleware.RequirePermission("approval", "write"), config.ApprovalHandler.SubmitApproval)
			tickets.GET("/approval/records", middleware.RequirePermission("approval", "read"), config.ApprovalHandler.GetApprovalRecords)

			// 审批工作流CRUD
			approvalWorkflows := tenant.Group("/approval-workflows")
			{
				approvalWorkflows.GET("", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.ListWorkflows)
				approvalWorkflows.POST("", middleware.RequirePermission("approval_workflow", "create"), config.ApprovalHandler.CreateWorkflow)
				approvalWorkflows.GET("/:id", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.GetWorkflow)
				approvalWorkflows.PUT("/:id", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalHandler.UpdateWorkflow)
				approvalWorkflows.POST("/:id/migrate-to-bpmn", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalHandler.MigrateWorkflowToBPMN)
				approvalWorkflows.PATCH("/:id", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalHandler.PatchWorkflow)
				approvalWorkflows.DELETE("/:id", middleware.RequirePermission("approval_workflow", "delete"), config.ApprovalHandler.DeleteWorkflow)
			}
			// 兼容旧路径 /approvals
			approvals := tenant.Group("/approvals")
			{
				approvals.GET("", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.ListWorkflows)
				approvals.POST("", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalHandler.CreateWorkflow)
				approvals.GET("/:id", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.GetWorkflow)
				approvals.PUT("/:id", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalHandler.UpdateWorkflow)
				approvals.PATCH("/:id", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalHandler.PatchWorkflow)
				approvals.DELETE("/:id", middleware.RequirePermission("approval_workflow", "delete"), config.ApprovalHandler.DeleteWorkflow)
				approvals.GET("/records", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.GetApprovalRecords)
				approvals.POST("/submit", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalHandler.SubmitApproval)
				// 兼容旧路径：/approval-records 和 /my-approvals
				tenant.GET("/approval-records", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.GetApprovalRecords)
				tenant.GET("/my-approvals", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalHandler.GetApprovalRecords)

			}

		}

		// 工单流转工作流
		if config.TicketWorkflowHandler != nil {
			tickets.POST("/workflow/accept", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.AcceptTicket)
			tickets.POST("/workflow/reject", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.RejectTicket)
			tickets.POST("/workflow/withdraw", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.WithdrawTicket)
			tickets.POST("/workflow/forward", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.ForwardTicket)
			tickets.POST("/workflow/cc", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.CCTicket)
			tickets.POST("/workflow/approve", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.ApproveTicket)
			tickets.POST("/workflow/resolve", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.ResolveTicket)
			tickets.POST("/workflow/close", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.CloseTicket)
			tickets.POST("/workflow/reopen", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowHandler.ReopenTicket)
			tickets.GET("/:id/cc", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.ListTicketCCRecords)
			tickets.GET("/:id/workflow/state", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.GetTicketWorkflowState)
			// 工单详情体验增强：V2 聚合 BPMN 真实节点状态（当前/下一/历史）。
			tickets.GET("/:id/workflow/state-v2", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.GetTicketWorkflowStateV2)
			tickets.GET("/:id/workflow-history", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.GetTicketWorkflowHistory)
			tickets.GET("/:id/workflow_records", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowHandler.GetTicketWorkflowHistory)
		}

		// 工单自动化规则
		if config.TicketAutomationRuleHandler != nil {
			tickets.GET("/automation-rules", middleware.RequirePermission("automation_rule", "read"), config.TicketAutomationRuleHandler.ListAutomationRules)
			tickets.POST("/automation-rules", middleware.RequirePermission("automation_rule", "create"), config.TicketAutomationRuleHandler.CreateAutomationRule)
			tickets.GET("/automation-rules/:id", middleware.RequirePermission("automation_rule", "read"), config.TicketAutomationRuleHandler.GetAutomationRule)
			tickets.PUT("/automation-rules/:id", middleware.RequirePermission("automation_rule", "update"), config.TicketAutomationRuleHandler.UpdateAutomationRule)
			tickets.DELETE("/automation-rules/:id", middleware.RequirePermission("automation_rule", "delete"), config.TicketAutomationRuleHandler.DeleteAutomationRule)
			tickets.POST("/automation-rules/:id/test", middleware.RequirePermission("automation_rule", "update"), config.TicketAutomationRuleHandler.TestAutomationRule)
		}

		// 工单分配规则
		if config.TicketAssignmentSmartHandler != nil {
			tickets.POST("/:id/auto-assign", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartHandler.AutoAssign)
			tickets.GET("/assign-recommendations/:id", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartHandler.GetAssignRecommendations)
			tickets.GET("/assignment-rules", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartHandler.ListAssignmentRules)
			tickets.POST("/assignment-rules", middleware.RequirePermission("assignment_rule", "create"), config.TicketAssignmentSmartHandler.CreateAssignmentRule)
			tickets.POST("/assignment-rules/test", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartHandler.TestAssignmentRule)
			tickets.GET("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartHandler.GetAssignmentRule)
			tickets.PUT("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartHandler.UpdateAssignmentRule)
			tickets.DELETE("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "delete"), config.TicketAssignmentSmartHandler.DeleteAssignmentRule)
		}

		// 工单评分
		if config.TicketRatingHandler != nil {
			tickets.POST("/:id/rating", middleware.RequirePermission("ticket", "update"), config.TicketRatingHandler.SubmitRating)
			tickets.GET("/:id/rating", middleware.RequirePermission("ticket", "read"), config.TicketRatingHandler.GetRating)
			tickets.GET("/rating-stats", middleware.RequirePermission("ticket", "read"), config.TicketRatingHandler.GetRatingStats)
		}

		if config.TicketTagHandler != nil {
			tickets.POST("/:id/tags", middleware.RequirePermission("ticket_tag", "create"), config.TicketTagHandler.AssignTagsToTicket)
			tickets.DELETE("/:id/tags", middleware.RequirePermission("ticket_tag", "delete"), config.TicketTagHandler.RemoveTagsFromTicket)
		}

		// 工单分析与预测 (Alias for frontend compatibility)
		if config.AIHandler != nil {
			tickets.POST("/analytics/deep", middleware.RequirePermission("report", "read"), config.AIHandler.GetDeepAnalytics)
			tickets.POST("/prediction/trend", middleware.RequirePermission("report", "read"), config.AIHandler.GetTrendPrediction)
		}

		if config.AnalyticsHandler != nil {
			tickets.POST("/analytics/export", middleware.RequirePermission("report", "export"), config.AnalyticsHandler.ExportAnalytics)
			// B8: GET /api/v1/analytics/tickets - 工单分析概览
			// 挂在 tenant 顶层 group，路径 = /api/v1/analytics/tickets
		}

		if config.PredictionHandler != nil {
			tickets.POST("/prediction/export", middleware.RequirePermission("report", "read"), config.PredictionHandler.ExportPredictionReport)
		}
	}

	if config.TicketDependencyHandler != nil {
		tickets.GET("/:id/dependencies", middleware.RequirePermission("ticket", "read"), config.TicketDependencyHandler.AnalyzeDependencyImpact)
	}

	// Ticket Notifications
	if config.TicketNotificationHandler != nil {
		tickets.GET("/:id/notifications", middleware.RequirePermission("notification", "read"), config.TicketNotificationHandler.ListTicketNotifications)
		tickets.POST("/:id/notifications", middleware.RequirePermission("notification", "create"), config.TicketNotificationHandler.SendTicketNotification)
	}

	if config.AnalyticsHandler != nil {
		tenant.GET("/analytics/tickets", middleware.RequirePermission("report", "read"), config.AnalyticsHandler.GetTicketAnalytics)
	}

	aiGroup := tenant.Group("/ai")
	{
		// 权限与 /ai/tickets/:id/analyze 对齐（ai.read）：两者同属 AI 能力面，
		// 避免同组端点一个走 ticket.read 一个走 ai.read 的对称性漂移。
		aiGroup.GET("/tickets/:id/summary", middleware.RequirePermission("ai", "read"), config.AIHandler.SummarizeTicket)
	}
}
