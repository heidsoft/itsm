package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"openclaw-deploy/pkg/aliyun"
	"openclaw-deploy/pkg/database"
)

func main() {
	// 加载环境变量
	godotenv.Load()

	// 初始化数据库
	if err := database.Init(); err != nil {
		log.Printf("⚠️  Database initialization failed: %v", err)
		log.Println("Running in mock mode")
	}

	// 初始化阿里云 SDK
	if err := aliyun.Init(); err != nil {
		log.Printf("⚠️  Aliyun SDK initialization failed: %v", err)
		log.Println("Running in mock mode")
	} else {
		log.Println("✅ Aliyun SDK initialized")
	}

	// 创建 Gin 路由
	r := gin.Default()

	// 跨域中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
	})

	// API 路由
	api := r.Group("/api/v1")
	{
		// 部署管理
		deployments := api.Group("/deployments")
		{
			deployments.GET("", getDeployments)
			deployments.POST("", createDeployment)
			deployments.GET("/:id", getDeployment)
			deployments.PUT("/:id", updateDeployment)
			deployments.DELETE("/:id", deleteDeployment)
			deployments.POST("/:id/start", startDeployment)
			deployments.POST("/:id/stop", stopDeployment)
			deployments.GET("/:id/metrics", getMetrics)
			deployments.GET("/:id/logs", getLogs)
		}

		// 监控告警
		monitor := api.Group("/monitor")
		{
			monitor.GET("/system", getSystemStatus)
			monitor.GET("/alerts", getAlerts)
			monitor.POST("/alerts/:id/ack", acknowledgeAlert)
		}

		// 用户管理
		users := api.Group("/users")
		{
			users.GET("/me", getCurrentUser)
			users.GET("", getUsers)
			users.PUT("/:id", updateUser)
		}

		// 域名管理
		domains := api.Group("/domains")
		{
			domains.GET("", getDomains)
			domains.POST("", createDomain)
			domains.DELETE("/:id", deleteDomain)
		}

		// 账号管理
		accounts := api.Group("/accounts")
		{
			accounts.GET("", getAccounts)
			accounts.POST("", createAccount)
			accounts.PUT("/:id/reset-password", resetPassword)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Starting OpenClaw Deploy API on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// Mock handlers (实际项目中应该实现真实逻辑)
func getDeployments(c *gin.Context) {
	// TODO: 从数据库获取
	c.JSON(200, gin.H{
		"data": []gin.H{
			{"id": "1", "instance_name": "prod-001", "user_id": "u001", "plan": "pro", "status": "running", "domain": "user1.openclaw.cn", "metrics": map[string]interface{}{"cpu_usage": 45.5, "memory_usage": 2.1, "qps": 1234, "response_time": 45}},
			{"id": "2", "instance_name": "prod-002", "user_id": "u002", "plan": "pro", "status": "running", "domain": "user2.openclaw.cn", "metrics": map[string]interface{}{"cpu_usage": 38.2, "memory_usage": 1.8, "qps": 987, "response_time": 52}},
			{"id": "3", "instance_name": "test-001", "user_id": "u003", "plan": "community", "status": "deploying", "domain": "user3.openclaw.cn", "metrics": map[string]interface{}{"cpu_usage": 12.0, "memory_usage": 0.5, "qps": 0, "response_time": 0}},
		},
		"total": 3,
		"page": 1,
		"page_size": 20,
	})
}

func createDeployment(c *gin.Context) {
	// TODO: 调用阿里云 API 创建 ECS
	c.JSON(201, gin.H{"message": "Deployment created", "id": "4", "status": "deploying"})
}

func getDeployment(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id"), "instance_name": "prod-001", "status": "running"})
}

func updateDeployment(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Deployment updated"})
}

func deleteDeployment(c *gin.Context) {
	// TODO: 调用阿里云 API 删除 ECS
	c.JSON(200, gin.H{"message": "Deployment deleted"})
}

func startDeployment(c *gin.Context) {
	// TODO: 调用阿里云 API 启动 ECS
	c.JSON(200, gin.H{"message": "Deployment started"})
}

func stopDeployment(c *gin.Context) {
	// TODO: 调用阿里云 API 停止 ECS
	c.JSON(200, gin.H{"message": "Deployment stopped"})
}

func getMetrics(c *gin.Context) {
	c.JSON(200, gin.H{
		"cpu_usage": 45.5,
		"memory_usage": 2.1,
		"disk_usage": 15.3,
		"network_in": 1024,
		"network_out": 512,
		"qps": 1234,
		"response_time": 45,
	})
}

func getLogs(c *gin.Context) {
	c.JSON(200, gin.H{"logs": []string{}})
}

func getSystemStatus(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy", "uptime": "99.9%"})
}

func getAlerts(c *gin.Context) {
	c.JSON(200, gin.H{"alerts": []gin.H{}})
}

func acknowledgeAlert(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Alert acknowledged"})
}

func getCurrentUser(c *gin.Context) {
	c.JSON(200, gin.H{"username": "admin", "role": "admin", "email": "admin@openclaw.cn"})
}

func getUsers(c *gin.Context) {
	c.JSON(200, gin.H{"users": []gin.H{}})
}

func updateUser(c *gin.Context) {
	c.JSON(200, gin.H{"message": "User updated"})
}

func getDomains(c *gin.Context) {
	c.JSON(200, gin.H{"domains": []gin.H{}})
}

func createDomain(c *gin.Context) {
	c.JSON(201, gin.H{"message": "Domain created"})
}

func deleteDomain(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Domain deleted"})
}

func getAccounts(c *gin.Context) {
	c.JSON(200, gin.H{"accounts": []gin.H{}})
}

func createAccount(c *gin.Context) {
	c.JSON(201, gin.H{"message": "Account created"})
}

func resetPassword(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Password reset", "password": "random_password"})
}
