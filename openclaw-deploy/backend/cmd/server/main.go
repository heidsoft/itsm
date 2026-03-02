package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	godotenv.Load()

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
			deployments.POST("/:id/start", startDeployment)
			deployments.POST("/:id/stop", stopDeployment)
			deployments.GET("/:id/metrics", getMetrics)
		}

		// 监控告警
		api.GET("/monitor/system", getSystemStatus)
		api.GET("/monitor/alerts", getAlerts)

		// 用户管理
		api.GET("/users/me", getCurrentUser)
	}

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

// Mock handlers (待实现真实逻辑)
func getDeployments(c *gin.Context) {
	c.JSON(200, gin.H{
		"data": []gin.H{
			{"id": "1", "instance_name": "prod-001", "status": "running", "metrics": map[string]interface{}{"cpu_usage": 45.5, "memory_usage": 2.1, "qps": 1234}},
			{"id": "2", "instance_name": "prod-002", "status": "running", "metrics": map[string]interface{}{"cpu_usage": 38.2, "memory_usage": 1.8, "qps": 987}},
			{"id": "3", "instance_name": "test-001", "status": "deploying", "metrics": map[string]interface{}{"cpu_usage": 12.0, "memory_usage": 0.5, "qps": 0}},
		},
		"total": 3,
	})
}

func createDeployment(c *gin.Context) {
	c.JSON(201, gin.H{"message": "Deployment created", "id": "4"})
}

func getDeployment(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id"), "instance_name": "prod-001", "status": "running"})
}

func startDeployment(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Deployment started"})
}

func stopDeployment(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Deployment stopped"})
}

func getMetrics(c *gin.Context) {
	c.JSON(200, gin.H{
		"cpu_usage": 45.5,
		"memory_usage": 2.1,
		"disk_usage": 15.3,
		"qps": 1234,
		"response_time": 45,
	})
}

func getSystemStatus(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy", "uptime": "99.9%"})
}

func getAlerts(c *gin.Context) {
	c.JSON(200, gin.H{"alerts": []gin.H{}})
}

func getCurrentUser(c *gin.Context) {
	c.JSON(200, gin.H{"username": "admin", "role": "admin"})
}
