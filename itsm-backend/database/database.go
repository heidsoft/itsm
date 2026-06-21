package database

// database包：负责数据库连接和配置
// 使用Ent ORM框架与PostgreSQL数据库交互

import (
	"context"      // Go标准库，用于处理上下文（超时、取消等）
	"database/sql" // Go标准库，提供数据库操作的通用接口
	"fmt"          // Go标准库，用于格式化字符串
	"log"
	"time" // Go标准库，用于时间处理

	"itsm-backend/config" // 自定义配置包
	"itsm-backend/ent"    // Ent ORM生成的代码包

	entsql "entgo.io/ent/dialect/sql" // Ent ORM的SQL方言包
	_ "github.com/lib/pq"             // PostgreSQL驱动，下划线表示只导入init函数
)

var rawDB *sql.DB

// GetRawDB returns the underlying *sql.DB for raw SQL operations (e.g., pgvector)
func GetRawDB() *sql.DB { return rawDB }

// InitDB initializes a raw database connection without Ent-specific setup
// Used for migrations and other operations that don't need Ent ORM
func InitDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode, cfg.Password)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// InitDatabase 初始化数据库连接
// 参数：cfg - 数据库配置信息（主机、端口、用户名、密码等）
// 返回值：*ent.Client - Ent ORM客户端，用于数据库操作
//
//	error - 错误信息
func InitDatabase(cfg *config.DatabaseConfig) (*ent.Client, error) {
	// 构建数据库连接字符串（DSN - Data Source Name）
	// PostgreSQL连接字符串格式：host=xxx port=xxx user=xxx dbname=xxx sslmode=xxx password=xxx
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
		cfg.Host,     // 数据库主机地址
		cfg.Port,     // 数据库端口号
		cfg.User,     // 数据库用户名
		cfg.DBName,   // 数据库名称
		cfg.SSLMode,  // SSL模式（disable/require/verify-ca等）
		cfg.Password) // 数据库密码

	// 方法一：使用 sql.Open 创建连接，然后配置连接池
	// sql.Open 不会立即连接数据库，只是验证连接字符串格式
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		// fmt.Errorf 创建格式化的错误信息
		// %w 是Go 1.13+的错误包装动词，保留原始错误信息
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	// 配置数据库连接池
	// 连接池可以重用数据库连接，提高性能
	db.SetMaxOpenConns(25)                 // 最大打开连接数：同时可以使用的最大连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数：连接池中保持的空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生存时间：连接在连接池中的最大生存时间

	// 测试数据库连接
	// context.WithTimeout 创建带超时的上下文，5秒后自动取消
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // defer确保函数退出时取消上下文

	// PingContext 测试数据库连接是否正常
	// 如果连接失败，会返回错误
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := normalizeMarketplaceItemDefaults(ctx, db); err != nil {
		return nil, err
	}

	// Try enable pgvector (ignore permission error), then conditionally create vectors table/index if extension exists
	if _, err := db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		log.Printf("pgvector extension enable failed (non-fatal): %v", err)
	}
	var hasVector bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')`).Scan(&hasVector); err != nil {
		log.Printf("pgvector extension check failed (non-fatal): %v", err)
		hasVector = false
	}
	if hasVector {
		if _, err := db.ExecContext(ctx, `
            CREATE TABLE IF NOT EXISTS vectors (
                id BIGSERIAL PRIMARY KEY,
                tenant_id INT NOT NULL,
                object_type TEXT NOT NULL,
                object_id INT NOT NULL,
                embedding VECTOR(1536) NOT NULL,
                content TEXT,
                source TEXT,
                created_at TIMESTAMPTZ DEFAULT NOW()
            );
        `); err != nil {
			log.Printf("create vectors table failed (non-fatal): %v", err)
		}
		// ensure unique constraint for upsert on (tenant_id, object_type, object_id)
		if _, err := db.ExecContext(ctx, `
            CREATE UNIQUE INDEX IF NOT EXISTS vectors_unique_tenant_obj ON vectors(tenant_id, object_type, object_id);
        `); err != nil {
			log.Printf("create vectors unique index failed (non-fatal): %v", err)
		}
		if _, err := db.ExecContext(ctx, `
            CREATE INDEX IF NOT EXISTS vectors_embedding_idx ON vectors USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
        `); err != nil {
			log.Printf("create vectors index failed (non-fatal): %v", err)
		}
	} else {
		log.Printf("pgvector extension not available. Vector features will be disabled.")
	}

	// Create ai_feedbacks table for AI telemetry
	if _, err := db.ExecContext(ctx, `
            CREATE TABLE IF NOT EXISTS ai_feedbacks (
                id BIGSERIAL PRIMARY KEY,
                created_at TIMESTAMPTZ DEFAULT NOW(),
                tenant_id INT NOT NULL,
                user_id INT NOT NULL,
                request_id TEXT NOT NULL,
                kind TEXT NOT NULL,
                query TEXT,
                item_type TEXT,
                item_id INT,
                useful BOOLEAN NOT NULL,
                score INT,
                notes TEXT
            );
        `); err != nil {
		log.Printf("create ai_feedbacks table failed (non-fatal): %v", err)
	}

	// Create indexes for ai_feedbacks
	if _, err := db.ExecContext(ctx, `
            CREATE INDEX IF NOT EXISTS ai_feedbacks_tenant_idx ON ai_feedbacks(tenant_id);
        `); err != nil {
		log.Printf("create ai_feedbacks tenant index failed (non-fatal): %v", err)
	}
	if _, err := db.ExecContext(ctx, `
            CREATE INDEX IF NOT EXISTS ai_feedbacks_created_idx ON ai_feedbacks(created_at);
        `); err != nil {
		log.Printf("create ai_feedbacks created index failed (non-fatal): %v", err)
	}

	// Compatibility fix: if legacy column "author" exists and is NOT NULL, relax constraint to allow inserts via current schema (author_id)
	var hasAuthorColumn bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='knowledge_articles' AND column_name='author'
    )`).Scan(&hasAuthorColumn); err == nil && hasAuthorColumn {
		// drop NOT NULL to avoid insert failures
		if _, err := db.ExecContext(ctx, `ALTER TABLE knowledge_articles ALTER COLUMN author DROP NOT NULL`); err != nil {
			log.Printf("relax knowledge_articles.author not null failed (non-fatal): %v", err)
		}
	}

	// 创建 Ent Driver 并返回 Client
	// entsql.OpenDB 将标准库的sql.DB包装为Ent的Driver
	drv := entsql.OpenDB("postgres", db)
	// ent.NewClient 创建Ent ORM客户端
	// ent.Driver(drv) 设置数据库驱动
	rawDB = db
	return ent.NewClient(ent.Driver(drv)), nil
}

func normalizeMarketplaceItemDefaults(ctx context.Context, db *sql.DB) error {
	const query = `
		SELECT column_name, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'marketplace_items'
		  AND column_name IN ('rating', 'price')
		  AND column_default = '0.0'
		ORDER BY column_name;
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("check marketplace_items numeric defaults: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName, columnDefault string
		if err := rows.Scan(&columnName, &columnDefault); err != nil {
			return fmt.Errorf("scan marketplace_items numeric defaults: %w", err)
		}
		columns = append(columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate marketplace_items numeric defaults: %w", err)
	}

	for _, column := range columns {
		stmt := fmt.Sprintf(`ALTER TABLE marketplace_items ALTER COLUMN %s SET DEFAULT 0`, column)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("normalize marketplace_items.%s default from 0.0 to 0: %w. Run as the marketplace_items table owner: %s", column, err, stmt)
		}
		log.Printf("normalized marketplace_items.%s default from 0.0 to 0", column)
	}

	return nil
}

// 数据库连接池说明：
// - MaxOpenConns: 控制最大并发连接数，避免数据库过载
// - MaxIdleConns: 保持一定数量的空闲连接，减少连接建立的开销
// - ConnMaxLifetime: 定期关闭旧连接，避免连接泄漏
//
// PostgreSQL特点：
// - ACID事务：原子性、一致性、隔离性、持久性
// - 复杂查询：支持子查询、窗口函数、JSON操作等
// - 扩展性强：支持自定义函数、数据类型、索引等
// - 高性能：MVCC并发控制、WAL日志、查询优化器等
//
// Ent ORM特点：
// - 类型安全：编译时检查SQL查询
// - 代码生成：根据schema自动生成CRUD代码
// - 图查询：支持复杂的关联查询
// - 迁移管理：自动处理数据库schema变更
