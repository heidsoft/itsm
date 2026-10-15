package router

import (
	knowledgeHandler "itsm-backend/handlers/knowledge"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupKnowledgeRoutes 注册知识库相关路由。
func SetupKnowledgeRoutes(tenant *gin.RouterGroup, h *knowledgeHandler.Handler) {
	// New route structure: /api/v1/knowledge/articles
	knowledgeGrp := tenant.Group("/knowledge")
	{
		// Articles
		articles := knowledgeGrp.Group("/articles")
		{
			articles.GET("", middleware.RequirePermission("knowledge", "read"), h.ListArticles)
			articles.POST("", middleware.RequirePermission("knowledge", "write"), h.CreateArticle)
			articles.GET("/:id", middleware.RequirePermission("knowledge", "read"), h.GetArticle)
			articles.PUT("/:id", middleware.RequirePermission("knowledge", "write"), h.UpdateArticle)
			articles.DELETE("/:id", middleware.RequirePermission("knowledge", "delete"), h.DeleteArticle)
			articles.POST("/:id/publish", middleware.RequirePermission("knowledge", "write"), h.PublishArticle)
			articles.POST("/:id/unpublish", middleware.RequirePermission("knowledge", "write"), h.UnpublishArticle)
			// 内容复核（L1 时效闭环）：确认内容仍然适用，解除「逾期未复核」过滤
			articles.POST("/:id/review", middleware.RequirePermission("knowledge", "write"), h.MarkArticleReviewed)

			// 评审状态机：submit/approve/reject/withdraw/republish。
			// 读侧（reviews 列表）用 read，其余动作统一 write —— 评审人本身
			// 需要 knowledge:write 才能改文章，用更严的码会把评审权锁死在作者手里。
			articles.GET("/:id/reviews", middleware.RequirePermission("knowledge", "read"), h.ListArticleReviews)
			articles.POST("/:id/review/submit", middleware.RequirePermission("knowledge", "write"), h.SubmitReview)
			articles.POST("/:id/review/approve", middleware.RequirePermission("knowledge", "write"), h.ApproveReview)
			articles.POST("/:id/review/reject", middleware.RequirePermission("knowledge", "write"), h.RejectReview)
			articles.POST("/:id/review/withdraw", middleware.RequirePermission("knowledge", "write"), h.WithdrawReview)
			articles.POST("/:id/review/republish", middleware.RequirePermission("knowledge", "write"), h.RepublishReview)

			// 版本历史：列表 / 恢复 / 对比。
			// 2026-10-14 前这三组端点未注册（service 层早有 RestoreVersion），
			// 前端文章详情的「版本历史」Tab 打开即失败。静态段必须先于 :version
			// 参数段注册，避免路由树把 compare 当成版本号匹配。
			articles.GET("/:id/versions", middleware.RequirePermission("knowledge", "read"), h.ListArticleVersions)
			articles.GET("/:id/versions/compare", middleware.RequirePermission("knowledge", "read"), h.CompareArticleVersions)
			articles.POST("/:id/versions/:version/restore", middleware.RequirePermission("knowledge", "write"), h.RestoreArticleVersion)

			// Comments
			articles.GET("/:id/comments", middleware.RequirePermission("knowledge", "read"), h.GetArticleComments)
			articles.POST("/:id/comments", middleware.RequirePermission("knowledge", "write"), h.AddArticleComment)
		}

		// Categories
		knowledgeGrp.GET("/categories", middleware.RequirePermission("knowledge", "read"), h.GetCategories)

		// 知识分类可见性纳管（L0 权限边界）：控制哪些分类的知识仅授权角色可读
		knowledgeGrp.GET("/categories/restricted", middleware.RequirePermission("knowledge", "write"), h.ListRestrictedCategories)
		knowledgeGrp.POST("/categories/restricted", middleware.RequirePermission("knowledge", "write"), h.SetCategoryRestriction)

		// Search
		knowledgeGrp.POST("/search", middleware.RequirePermission("knowledge", "read"), h.SearchArticles)

		// Recommendations
		knowledgeGrp.GET("/recommendations", middleware.RequirePermission("knowledge", "read"), h.GetRecommendations)
		knowledgeGrp.GET("/recent", middleware.RequirePermission("knowledge", "read"), h.GetRecentArticles)

		// Stats
		knowledgeGrp.GET("/stats", middleware.RequirePermission("knowledge", "read"), h.GetStats)
	}

	// Legacy route for backward compatibility: /api/v1/knowledge-articles/*
	kbGrp := tenant.Group("/knowledge-articles")
	{
		kbGrp.GET("", middleware.RequirePermission("knowledge", "read"), h.ListArticles)
		kbGrp.POST("", middleware.RequirePermission("knowledge", "write"), h.CreateArticle)
		// 静态路由必须在动态路由 /:id 之前注册，否则 Gin 会将 "categories" 当作 :id 参数
		kbGrp.GET("/categories", middleware.RequirePermission("knowledge", "read"), h.GetCategories)
		kbGrp.GET("/:id", middleware.RequirePermission("knowledge", "read"), h.GetArticle)
	}
}
