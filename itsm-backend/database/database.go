package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"itsm-backend/config"
	"itsm-backend/ent"
)

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.DatabaseConfig) (*ent.Client, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode, cfg.Password)

	// 方法一：使用 sql.Open 创建连接，然后配置连接池
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生存时间

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 创建 Ent Driver 并返回 Client
	drv := entsql.OpenDB("postgres", db)
	return ent.NewClient(ent.Driver(drv)), nil
}
