package router

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(jwtSecret string) *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RecoveryMiddleware())

	// 公共路由（无需认证）
	public := r.Group("/api/v1")
	{
		public.POST("/login", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "message": "登录功能开发中"})
		})
		public.POST("/refresh-token", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "message": "刷新Token功能开发中"})
		})
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
		public.GET("/version", func(c *gin.Context) {
			c.JSON(200, gin.H{"version": "1.0.0"})
		})
	}

	// 认证路由（需要JWT）
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthMiddleware(jwtSecret))
	{
		// 工单管理路由 - 暂时使用占位符
		tickets := auth.Group("/tickets")
		{
			tickets.GET("", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "工单列表功能开发中"})
			})
			tickets.POST("", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "创建工单功能开发中"})
			})
			tickets.GET("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "获取工单详情功能开发中"})
			})
			tickets.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "更新工单功能开发中"})
			})
			tickets.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "删除工单功能开发中"})
			})
		}
	}

	return r
}
