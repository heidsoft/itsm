//go:build migrate
// +build migrate

package main

import (
	"context"
	"database/sql"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 连接数据库
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("failed connecting to database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 1. 首先创建一个默认租户
	defaultTenant, err := createDefaultTenant(ctx, client)
	if err != nil {
		log.Fatalf("failed to create default tenant: %v", err)
	}

	fmt.Printf("Created default tenant with ID: %d\n", defaultTenant.ID)

	// 2. 更新现有数据，将它们分配给默认租户
	err = migrateExistingData(ctx, client, defaultTenant.ID)
	if err != nil {
		log.Fatalf("failed to migrate existing data: %v", err)
	}

	fmt.Println("Data migration completed successfully!")

	// 3. 现在可以安全地运行 schema 迁移
	err = client.Schema.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}

	fmt.Println("Schema migration completed successfully!")
}

func createDefaultTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
	// 检查是否已经存在默认租户
	existingTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err == nil {
		// 默认租户已存在
		return existingTenant, nil
	}

	// 创建默认租户
	return client.Tenant.Create().
		SetName("默认租户").
		SetCode("default").
		SetDomain("localhost").
		SetStatus(tenant.StatusActive).
		SetType(tenant.TypeEnterprise).
		SetSettings(map[string]interface{}{
			"max_users":   1000,
			"max_tickets": 10000,
		}).
		SetQuota(map[string]interface{}{
			"storage":   "10GB",
			"bandwidth": "100GB",
		}).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func migrateExistingData(ctx context.Context, client *ent.Client, tenantID int) error {
	// 获取底层的 sql.DB 连接
	db := client.Driver().(*sql.DB)

	tables := []string{
		"users",
		"tickets",
		"service_catalogs",
		"service_requests",
		"subscriptions",
	}

	for _, table := range tables {
		// 检查表是否存在 tenant_id 列
		var exists bool
		err := db.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = '%s' AND column_name = 'tenant_id'
			)`, table)).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check column existence for table %s: %w", table, err)
		}

		if !exists {
			fmt.Printf("Table %s does not have tenant_id column, skipping...\n", table)
			continue
		}

		// 更新现有数据
		result, err := db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET tenant_id = $1 WHERE tenant_id IS NULL
		`, table), tenantID)
		if err != nil {
			return fmt.Errorf("failed to update table %s: %w", table, err)
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Updated %d rows in table %s\n", rowsAffected, table)
	}

	return nil
}
