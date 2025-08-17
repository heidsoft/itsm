//go:build migrate
// +build migrate

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接参数
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "dev")
	dbPassword := getEnv("DB_PASSWORD", "123456!@#$%^")
	dbName := getEnv("DB_NAME", "itsm")
	sslMode := getEnv("DB_SSLMODE", "disable")

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// 执行迁移
	if err := migrateSLAFields(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully")
}

func migrateSLAFields(db *sql.DB) error {
	// 1. 添加新的JSONB列
	log.Println("Adding new JSONB columns...")
	
	// 添加新的business_hours_json列
	_, err := db.Exec(`
		ALTER TABLE sla_definitions 
		ADD COLUMN IF NOT EXISTS business_hours_json JSONB
	`)
	if err != nil {
		return fmt.Errorf("failed to add business_hours_json column: %v", err)
	}

	// 添加新的字段
	_, err = db.Exec(`
		ALTER TABLE sla_definitions 
		ADD COLUMN IF NOT EXISTS is_active_new BOOLEAN DEFAULT true
	`)
	if err != nil {
		return fmt.Errorf("failed to add is_active_new column: %v", err)
	}

	// 2. 迁移现有数据
	log.Println("Migrating existing data...")
	
	// 将现有的business_hours字符串转换为JSONB
	_, err = db.Exec(`
		UPDATE sla_definitions 
		SET business_hours_json = CASE 
			WHEN business_hours = '' THEN '{}'::jsonb
			ELSE business_hours::jsonb
		END
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate business_hours data: %v", err)
	}

	// 3. 删除旧列并重命名新列
	log.Println("Replacing old columns...")
	
	// 删除旧列
	_, err = db.Exec(`ALTER TABLE sla_definitions DROP COLUMN IF EXISTS business_hours`)
	if err != nil {
		return fmt.Errorf("failed to drop old business_hours column: %v", err)
	}

	// 重命名新列
	_, err = db.Exec(`ALTER TABLE sla_definitions RENAME COLUMN business_hours_json TO business_hours`)
	if err != nil {
		return fmt.Errorf("failed to rename business_hours_json column: %v", err)
	}

	// 删除不需要的列
	_, err = db.Exec(`ALTER TABLE sla_definitions DROP COLUMN IF EXISTS holidays`)
	if err != nil {
		return fmt.Errorf("failed to drop holidays column: %v", err)
	}

	_, err = db.Exec(`ALTER TABLE sla_definitions DROP COLUMN IF EXISTS impact`)
	if err != nil {
		return fmt.Errorf("failed to drop impact column: %v", err)
	}

	_, err = db.Exec(`ALTER TABLE sla_definitions DROP COLUMN IF EXISTS created_by`)
	if err != nil {
		return fmt.Errorf("failed to drop created_by column: %v", err)
	}

	// 4. 更新列约束
	log.Println("Updating column constraints...")
	
	// 设置business_hours为可空
	_, err = db.Exec(`ALTER TABLE sla_definitions ALTER COLUMN business_hours DROP NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to make business_hours nullable: %v", err)
	}

	// 设置service_type和priority为可空
	_, err = db.Exec(`ALTER TABLE sla_definitions ALTER COLUMN service_type DROP NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to make service_type nullable: %v", err)
	}

	_, err = db.Exec(`ALTER TABLE sla_definitions ALTER COLUMN priority DROP NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to make priority nullable: %v", err)
	}

	// 5. 重新创建索引
	log.Println("Recreating indexes...")
	
	// 删除旧索引
	_, err = db.Exec(`DROP INDEX IF EXISTS sladefinition_tenant_id_service_type_priority_impact`)
	if err != nil {
		return fmt.Errorf("failed to drop old index: %v", err)
	}

	// 创建新索引
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS sladefinition_tenant_id_service_type_priority 
		ON sla_definitions (tenant_id, service_type, priority)
	`)
	if err != nil {
		return fmt.Errorf("failed to create new index: %v", err)
	}

	log.Println("SLA table migration completed successfully")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
