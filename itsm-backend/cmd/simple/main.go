package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 初始化数据库
	client, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer client.Close()

	// 运行数据库迁移
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// 初始化路由
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "itsm-cmdb",
			"version": "1.0.0",
		})
	})

	// 基础API
	api := r.Group("/api/v1")
	{
		cmdb := api.Group("/cmdb")
		{
			// CI类型
			cmdb.GET("/classes", func(c *gin.Context) {
				classes := []map[string]interface{}{
					{"name": "cmdb_ci_server", "label": "服务器", "icon": "server", "color": "#28a745"},
					{"name": "cmdb_ci_ip_switch", "label": "交换机", "icon": "network", "color": "#17a2b8"},
					{"name": "cmdb_ci_database", "label": "数据库", "icon": "database", "color": "#fd7e14"},
				}
				c.JSON(200, gin.H{"success": true, "data": classes})
			})

			// CI列表
			cmdb.GET("/cis", func(c *gin.Context) {
				cis, err := client.ConfigurationItem.Query().All(context.Background())
				if err != nil {
					c.JSON(500, gin.H{"success": false, "message": err.Error()})
					return
				}
				c.JSON(200, gin.H{"success": true, "data": cis})
			})

			// 创建CI
			cmdb.POST("/cis", func(c *gin.Context) {
				var req map[string]interface{}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"success": false, "message": err.Error()})
					return
				}

				name, _ := req["name"].(string)
				ciType, _ := req["ci_type"].(string)
				if ciType == "" {
					ciType = "server"
				}

				if name == "" {
					c.JSON(400, gin.H{"success": false, "message": "name is required"})
					return
				}

				ci, err := client.ConfigurationItem.Create().
					SetName(name).
					SetCiType(ciType).
					SetStatus("operational").
					SetEnvironment("production").
					SetCriticality("medium").
					SetTenantID(1).
					Save(context.Background())

				if err != nil {
					c.JSON(500, gin.H{"success": false, "message": err.Error()})
					return
				}

				c.JSON(200, gin.H{"success": true, "data": ci})
			})
		}
	}

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 CMDB服务启动成功！\n")
	fmt.Printf("🌐 API地址: http://localhost:%s\n", port)
	fmt.Printf("🔍 健康检查: http://localhost:%s/health\n", port)
	fmt.Printf("📋 CI类型: http://localhost:%s/api/v1/cmdb/classes\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDatabase() (*ent.Client, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "sqlite:///tmp/itsm_cmdb.db?_fk=1"
	}

	var driver, dsn string
	if strings.HasPrefix(dbURL, "postgres://") {
		driver = "postgres"
		dsn = dbURL
	} else {
		driver = "sqlite3"
		dsn = "/tmp/itsm_cmdb.db?_fk=1&cache=shared&mode=rwc"
	}

	client, err := ent.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return client, nil
}
