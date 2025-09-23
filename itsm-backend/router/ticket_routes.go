package router

import (
	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupTicketRoutes 设置工单相关路由
func SetupTicketRoutes(
	auth *gin.RouterGroup,
	ticketController *controller.TicketController,
	ticketTagController *controller.TicketTagController,
	ticketAssignmentController *controller.TicketAssignmentController,
	ticketCategoryController *controller.TicketCategoryController,
	knowledgeIntegrationController *controller.KnowledgeIntegrationController,
	client *ent.Client,
) {
	// 工单管理路由
	tickets := auth.Group("/tickets")
	{
		// 基础CRUD操作
		tickets.GET("", middleware.RequirePermission("ticket", "read"), ticketController.ListTickets)
		tickets.POST("", middleware.RequirePermission("ticket", "create"), ticketController.CreateTicket)
		tickets.GET("/:id", middleware.RequirePermission("ticket", "read"), ticketController.GetTicket)
		tickets.PUT("/:id", middleware.RequirePermission("ticket", "update"), ticketController.UpdateTicket)
		tickets.DELETE("/:id", middleware.RequirePermission("ticket", "delete"), ticketController.DeleteTicket)

		// 统计和分析
		tickets.GET("/stats", middleware.RequirePermission("ticket", "read"), ticketController.GetTicketStats)
		tickets.GET("/analytics", middleware.RequirePermission("ticket", "read"), ticketController.GetTicketAnalytics)

		// 批量操作
		tickets.POST("/batch-delete", middleware.RequirePermission("ticket", "delete"), ticketController.BatchDeleteTickets)
		tickets.POST("/batch-assign", middleware.RequirePermission("ticket", "assign"), ticketController.AssignTickets)

		// 导入导出
		tickets.POST("/export", middleware.RequirePermission("ticket", "export"), ticketController.ExportTickets)
		tickets.POST("/import", middleware.RequirePermission("ticket", "import"), ticketController.ImportTickets)

		// 工单状态操作
		tickets.POST("/:id/assign", middleware.RequirePermission("ticket", "assign"), ticketController.AssignTicket)
		tickets.POST("/:id/escalate", middleware.RequirePermission("ticket", "escalate"), ticketController.EscalateTicket)
		tickets.POST("/:id/resolve", middleware.RequirePermission("ticket", "resolve"), ticketController.ResolveTicket)
		tickets.POST("/:id/close", middleware.RequirePermission("ticket", "close"), ticketController.CloseTicket)

		// 查询和搜索
		tickets.GET("/search", middleware.RequirePermission("ticket", "read"), ticketController.SearchTickets)
		tickets.GET("/overdue", middleware.RequirePermission("ticket", "read"), ticketController.GetOverdueTickets)
		tickets.GET("/assignee/:assignee_id", middleware.RequirePermission("ticket", "read"), ticketController.GetTicketsByAssignee)
		tickets.GET("/:id/activity", middleware.RequirePermission("ticket", "read"), ticketController.GetTicketActivity)

		// 知识库集成路由
		knowledge := tickets.Group("/:id/knowledge")
		knowledge.Use(middleware.RequirePermission("knowledge", "read"))
		{
			knowledge.GET("/recommendations", knowledgeIntegrationController.RecommendSolutions)
			knowledge.POST("/associate", knowledgeIntegrationController.AssociateWithKnowledge)
			knowledge.GET("/ai-recommendations", knowledgeIntegrationController.GetAIRecommendations)
			knowledge.GET("/related-articles", knowledgeIntegrationController.GetRelatedArticles)
			knowledge.GET("/associations", knowledgeIntegrationController.GetKnowledgeAssociations)
		}
	}

	// 工单分类管理
	categories := auth.Group("/ticket-categories")
	categories.Use(middleware.RequirePermission("ticket_category", "read"))
	{
		categories.GET("", ticketCategoryController.ListCategories)
		categories.POST("", middleware.RequirePermission("ticket_category", "create"), ticketCategoryController.CreateCategory)
		categories.GET("/:id", ticketCategoryController.GetCategory)
		categories.PUT("/:id", middleware.RequirePermission("ticket_category", "update"), ticketCategoryController.UpdateCategory)
		categories.DELETE("/:id", middleware.RequirePermission("ticket_category", "delete"), ticketCategoryController.DeleteCategory)
	}

	// 工单标签管理
	tags := auth.Group("/ticket-tags")
	tags.Use(middleware.RequirePermission("ticket_tag", "read"))
	{
		tags.GET("", ticketTagController.ListTags)
		tags.POST("", middleware.RequirePermission("ticket_tag", "create"), ticketTagController.CreateTag)
		tags.GET("/:id", ticketTagController.GetTag)
		tags.PUT("/:id", middleware.RequirePermission("ticket_tag", "update"), ticketTagController.UpdateTag)
		tags.DELETE("/:id", middleware.RequirePermission("ticket_tag", "delete"), ticketTagController.DeleteTag)
	}

	// 工单模板管理
	templates := auth.Group("/ticket-templates")
	templates.Use(middleware.RequirePermission("ticket_template", "read"))
	{
		templates.GET("", ticketController.GetTicketTemplates)
		templates.POST("", middleware.RequirePermission("ticket_template", "create"), ticketController.CreateTicketTemplate)
		templates.PUT("/:id", middleware.RequirePermission("ticket_template", "update"), ticketController.UpdateTicketTemplate)
		templates.DELETE("/:id", middleware.RequirePermission("ticket_template", "delete"), ticketController.DeleteTicketTemplate)
	}
}