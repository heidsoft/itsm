package main

import (
	"context"
	"fmt"
	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/service"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 初始化日志
	logger := initLogger()
	defer logger.Sync()
	sugar := logger.Sugar()

	// 初始化数据库
	client, err := initDatabase()
	if err != nil {
		sugar.Fatalf("Failed to initialize database: %v", err)
	}
	defer client.Close()

	// 运行数据库迁移
	if err := runMigrations(context.Background(), client); err != nil {
		sugar.Fatalf("Failed to run migrations: %v", err)
	}

	// 初始化服务
	cmdbService := service.NewCMDBService(client)

	// 初始化控制器
	cmdbController := controller.NewCMDBController(cmdbService)

	// 初始化路由
	r := gin.Default()
	
	// 设置CORS
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

	// 设置路由
	api := r.Group("/api/v1")
	{
		cmdb := api.Group("/configuration-items")
		{
			cmdb.GET("", cmdbController.ListCIs)
			cmdb.POST("", cmdbController.CreateCI)
			cmdb.GET("/:id", cmdbController.GetCI)
			cmdb.GET("/:id/topology", cmdbController.GetCITopology)
			cmdb.POST("/relationships", cmdbController.CreateRelationship)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "itsm-cmdb",
			"version": "1.0.0",
		})
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sugar.Infow("Starting CMDB server", "port", port)
	if err := r.Run(":" + port); err != nil {
		sugar.Fatalf("Failed to start server: %v", err)
	}
}

// initLogger 初始化日志
func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

// initDatabase 初始化数据库
func initDatabase() (*ent.Client, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "sqlite:///tmp/itsm_cmdb.db"
	}

	var driver, dsn string
	if strings.HasPrefix(dbURL, "postgres://") {
		driver = "postgres"
		dsn = dbURL
	} else if strings.HasPrefix(dbURL, "sqlite://") {
		driver = "sqlite3"
		dsn = strings.TrimPrefix(dbURL, "sqlite://")
	} else {
		driver = "sqlite3"
		dsn = "/tmp/itsm_cmdb.db"
	}

	client, err := ent.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return client, nil
}

// runMigrations 运行数据库迁移
func runMigrations(ctx context.Context, client *ent.Client) error {
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// 初始化基础数据
	if err := initializeBaseData(ctx, client); err != nil {
		return fmt.Errorf("failed to initialize base data: %w", err)
	}

	return nil
}

// initializeBaseData 初始化基础数据
func initializeBaseData(ctx context.Context, client *ent.Client) error {
	// 创建默认CI类型
	ciClasses := []struct {
		Name  string
		Label string
		Icon  string
		Color string
	}{
		{"cmdb_ci", "配置项", "default", "#007bff"},
		{"cmdb_ci_server", "服务器", "server", "#28a745"},
		{"cmdb_ci_ip_switch", "交换机", "network", "#17a2b8"},
		{"cmdb_ci_ip_router", "路由器", "router", "#ffc107"},
		{"cmdb_ci_ip_firewall", "防火墙", "shield", "#dc3545"},
		{"cmdb_ci_vm_instance", "虚拟机实例", "cloud", "#6f42c1"},
		{"cmdb_ci_database", "数据库", "database", "#fd7e14"},
		{"cmdb_ci_kubernetes", "Kubernetes资源", "kubernetes", "#20c997"},
	}

	for _, class := range ciClasses {
		count, err := client.CIType.Query().
			Where(citype.Name(class.Name), citype.TenantID(1)).
			Count(ctx)
		if err != nil {
			count = 0
		}

		if count == 0 {
			_, err := client.CIType.Create().
				SetName(class.Name).
				SetDescription(class.Label).
				SetIcon(class.Icon).
				SetColor(class.Color).
				SetAttributeSchema("").
				SetTenantID(1).
				Save(ctx)
			if err != nil {
				log.Printf("Warning: failed to create CI type %s: %v", class.Name, err)
			}
		}
	}

	return nil
}
