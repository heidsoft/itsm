package database

// database包：负责数据库连接和配置
// 使用Ent ORM框架与PostgreSQL数据库交互

import (
	"context"      // Go标准库，用于处理上下文（超时、取消等）
	"database/sql" // Go标准库，提供数据库操作的通用接口
	"errors"
	"fmt" // Go标准库，用于格式化字符串
	"log"
	"regexp"
	"time" // Go标准库，用于时间处理

	"itsm-backend/config"       // 自定义配置包
	"itsm-backend/database/rls" // RLS driver 装饰器
	"itsm-backend/ent"          // Ent ORM生成的代码包

	entsql "entgo.io/ent/dialect/sql" // Ent ORM的SQL方言包
	_ "github.com/lib/pq"             // PostgreSQL驱动，下划线表示只导入init函数
	"go.uber.org/zap"
)

var rawDB *sql.DB

// EnsureOperationalIndexes 创建运营性索引与序列（幂等）。
//
// 这些对象不属于 ent schema，但承担数据正确性约束：
//   - sla_violation_unique_open_idx：同一 (ticket_id, violation_type) 在未解决状态下
//     只能有一条违规，是 SLA Monitor 跨实例竞态的最后一道防线（服务层把唯一冲突
//     识别为幂等跳过）；部分索引条件允许已解决后再次触发新违规。
//   - sla_violation_ticket_type_idx：按工单查违规的常规索引。
//   - incident_number_seq + incidents_tenant_number_idx：保证 INC-YYYYMM-NNNNNN
//     后缀唯一且单调递增，消除「COUNT+1 并发复用 / 删除回退 / UTC 窗口错乱」。
//
// 必须在 schema 与迁移之后调用（bootstrap.InitializeStorage 末尾）。它同时仍挂在
// 数据库连接路径上做 best-effort，但那里空库必然失败：表还没建出来，四条 DDL 全部
// 报错并被 "non-fatal" 日志吞掉，之后不再重试。2026-10-27 用空库首装与生产库对账，
// 全新安装缺上述三个索引——事件编号与 SLA 违规的防重约束在首装环境形同虚设。
func EnsureOperationalIndexes(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("ensure operational indexes: nil database handle")
	}

	stmts := []struct {
		name string
		sql  string
	}{
		{
			name: "sla_violation_unique_open_idx",
			sql: `
				CREATE UNIQUE INDEX IF NOT EXISTS sla_violation_unique_open_idx
				ON sla_violations (ticket_id, violation_type)
				WHERE is_resolved = false;`,
		},
		{
			name: "sla_violation_ticket_type_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS sla_violation_ticket_type_idx
				ON sla_violations (ticket_id, violation_type);`,
		},
		{
			name: "incident_number_seq",
			sql:  `CREATE SEQUENCE IF NOT EXISTS incident_number_seq;`,
		},
		{
			// 播种为「已存在事件编号的最大数字后缀 + 1」，避免与历史编号碰撞。
			// 表为空时 setval(1, false)：下一次 nextval 返回 1（is_called=false），
			// 而不是被 GREATEST 抬到 2 白白跳过 000001。
			//
			// 历史实现写的是 setval(seq, 0)——序列下界是 1，该调用恒报
			// "value 0 is out of bounds"，被 non-fatal 日志吞掉；也就是说
			// 空库首装从来没有真正播种过序列，恢复带数据的环境后极易撞号。
			name: "incident_number_seq.seed",
			sql: `
				WITH m AS (
					SELECT COALESCE(MAX(CAST(substring(incident_number FROM 12) AS BIGINT)), 0) AS v
					FROM incidents
					WHERE incident_number ~ '^INC-[0-9]{6}-[0-9]{6}$'
				)
				SELECT setval('incident_number_seq', GREATEST(v, 1), v > 0) FROM m;`,
		},
		{
			name: "incidents_tenant_number_idx",
			sql: `
				CREATE UNIQUE INDEX IF NOT EXISTS incidents_tenant_number_idx
				ON incidents (tenant_id, incident_number)
				WHERE incident_number IS NOT NULL;`,
		},
	}

	var errs []error
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s.sql); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.name, err))
		}
	}
	return errors.Join(errs...)
}

// duplicateIndexPairs 是「完全重复索引」的收敛表，每项为 {保留, 删除}。
//
// 判据是列、唯一性、谓词、排序方向四项全同（用 pg_get_indexdef 去掉索引名后
// 逐字比较 + indoption + indisunique），因此删除任一侧都不会改变任何查询的计划
// 可选性。2026-10-27 生产库实测 32 对，冗余侧累计 1861 次扫描 / 368 kB。
//
// 根因是三份「补索引」批次脚本按名字判断缺失：add_missing_indexes.sql /
// add_missing_indexes_batch2.sql / 20261027_prod_schema_guards.sql 看到「没有叫
// idx_ticket_types_tenant 的索引」就补一条，而 ent 早已为同一列建过
// tickettype_tenant_id——名字不同、定义相同。其中 6 对是 ent 自己造成的：字段上
// 写了 .Unique()（生成 <表>_<列>_key），又在 Indexes() 里写了一遍
// index.Fields(<列>).Unique()（生成 <实体>_<列>）。
//
// 收敛方向：保留 ent 声明的那一份。ent 的 auto-migrate 在 ITSM_BOOTSTRAP_ONLY
// 初始化容器里每次部署都跑，删掉它必然被重建，手写索引删掉就是删掉。
//
// 三处例外（表不由 ent 管理，两份都是手写）：change_pi_rs /
// incident_related_incidents / ticket_comments，保留更早的功能迁移那份。
// 第四处方向相反（cloud_resources）：20260829 迁移已显式 DROP ent 那份并改建
// cloud_resources_account_resource_lookup_idx，但 ent schema 仍声明着它，
// 于是每次 bootstrap 又建回来——生产库因此有两份、首装库只有一份。该对保留
// 迁移那份，并已从 ent/schema/cloud_resource.go 移除声明。
//
// 为什么不是一份 migrations/*.sql：无日期脚本（add_missing_indexes*）按字符串
// 排序落在所有 YYYYMMDD_* 之后，任何日期化「删除型」迁移都会被它们随后重建，
// 首装环境因此永远收敛不掉（首装门禁实测：日期化迁移先跑，幸存方都还不存在）。
// 这三份脚本的 checksum 已在生产账本里固化，不能改写内容。所以收敛放在迁移流
// 之后的 InitializeStorage 末尾，与 EnsureOperationalIndexes 同一挂载点。
var duplicateIndexPairs = [][2]string{
	{"idx_change_pi_rs_tenant_id", "idx_change_pi_rs_tenant"},
	{"cirelationship_tenant_id", "idx_ci_relationships_tenant_id"},
	{"cloud_resources_account_resource_lookup_idx", "cloudresource_cloud_account_id_resource_id"},
	{"cloudresource_tenant_id_identity_hash", "cloud_resources_identity_expand_idx"},
	{"cmdb_export_tasks_task_id_key", "cmdbexporttask_task_id"},
	{"cmdb_import_tasks_task_id_key", "cmdbimporttask_task_id"},
	{"configurationitem_tenant_id_source_id", "configuration_items_source_expand_idx"},
	{"discoveryjob_status_lease_expires_at", "discovery_jobs_claim_idx"},
	{"discoveryjob_tenant_id_operation_source_id_idempotency_key", "discovery_jobs_idempotency_expand_idx"},
	{"discoveryresult_tenant_id_job_id_resource_identity", "discovery_results_identity_expand_idx"},
	{"discoverysource_tenant_id_cloud_account_id", "discovery_sources_tenant_cloud_account_idx"},
	{"domainconfig_tenant_id_config_type_config_key", "idx_domain_config_lookup"},
	{"domainconfig_tenant_id_department_id_team_id", "idx_domain_config_scope"},
	{"emailoutboundmessage_tenant_id_status_created_at", "email_outbound_messages_delivery_idx"},
	{"inboundemailmessage_tenant_id_processing_status_received_at", "inbound_email_messages_processing_idx"},
	{"idx_incident_related_incidents_incident_id", "idx_inc_related_inc"},
	{"knowledgearticlereview_article_id_created_at", "idx_knowledge_article_reviews_article_created"},
	{"knowledgearticlereview_tenant_id_article_id", "idx_knowledge_article_reviews_tenant_article"},
	{"knowledgearticle_tenant_id_source_problem_id", "idx_knowledge_articles_source_problem"},
	{"oncallshift_tenant_id_schedule_id_start_at_end_at", "on_call_shifts_lookup_idx"},
	{"processbinding_tenant_id_busin_3b376c1950cea2bb24c7d1dc673951a4", "idx_process_binding_routing"},
	{"process_execution_histories_history_id_key", "processexecutionhistory_history_id"},
	{"process_instances_process_instance_id_key", "processinstance_process_instance_id"},
	{"process_tasks_task_id_key", "processtask_task_id"},
	{"process_variables_variable_id_key", "processvariable_variable_id"},
	{"ticket_comment_ticket_id_idx", "idx_tkt_cmt_ticket"},
	{"ticket_tenant_id_ticket_type_id", "idx_tickets_tenant_ticket_type"},
	{"tickettype_status", "idx_ticket_types_status"},
	{"tickettype_tenant_id", "idx_ticket_types_tenant"},
	{"tickettype_tenant_id_sort_order", "idx_ticket_types_tenant_sort"},
	{"workflowtemplate_tenant_id_domain_status", "idx_workflow_templates_domain"},
	{"workflowtemplate_tenant_id_is_public", "idx_workflow_templates_public"},
}

// safeIndexName 只放行小写标识符，避免把外部字符串拼进 DDL。
var safeIndexName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// EnsureIndexConvergence 删除 duplicateIndexPairs 里的冗余索引（幂等），返回删除条数。
//
// 每一对先断言「保留方存在」再删「冗余方」：保留方缺失说明环境形态与假设不符
// （例如某份建索引的迁移被跳过），此时删冗余方会让该访问路径彻底没有索引。
// 这种情况记成错误而非静默跳过——收敛失败要让部署看得见。
//
// 必须在全部迁移（含无日期脚本）之后调用，见 duplicateIndexPairs 的注释。
func EnsureIndexConvergence(ctx context.Context, db *sql.DB) (int, error) {
	if db == nil {
		return 0, errors.New("ensure index convergence: nil database handle")
	}

	var errs []error
	dropped := 0
	for _, pair := range duplicateIndexPairs {
		keep, redundant := pair[0], pair[1]
		if !safeIndexName.MatchString(keep) || !safeIndexName.MatchString(redundant) {
			errs = append(errs, fmt.Errorf("index convergence: 非法索引名 %q/%q", keep, redundant))
			continue
		}

		var keepExists bool
		if err := db.QueryRowContext(ctx,
			`SELECT to_regclass($1) IS NOT NULL`, keep).Scan(&keepExists); err != nil {
			errs = append(errs, fmt.Errorf("index convergence: 探测 %s: %w", keep, err))
			continue
		}
		if !keepExists {
			errs = append(errs, fmt.Errorf(
				"index convergence: 保留方 %s 不存在，拒绝删除 %s（会让该访问路径失去索引）",
				keep, redundant))
			continue
		}

		res, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+redundant)
		if err != nil {
			errs = append(errs, fmt.Errorf("index convergence: 删除 %s: %w", redundant, err))
			continue
		}
		if n, err := res.RowsAffected(); err == nil && n > 0 {
			dropped++
		}
	}
	return dropped, errors.Join(errs...)
}

// rlsDriver 保存最后一次初始化的 RLS 装饰器实例，供 /internal/rls-stats
// 等诊断端点读取运行时统计。为 nil 表示未启用 RLS 或未初始化。
var rlsDriver *rls.Driver

// GetRawDB returns the underlying *sql.DB for raw SQL operations (e.g., pgvector)
func GetRawDB() *sql.DB { return rawDB }

// GetRLSDriver 返回当前进程使用的 RLS 装饰器（可能为 nil）。
// 用于运维/诊断端点导出 Stats()。
func GetRLSDriver() *rls.Driver { return rlsDriver }

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

	// Schema is owned by the explicit bootstrap migration job. Application
	// startup must remain read-only with respect to database structure.
	//
	// 例外只有一处：EnsureOperationalIndexes 里那几条 CREATE INDEX/SEQUENCE。
	// 它们挂在连接路径上是为了兼容「早于 bootstrap job 就存在」的历史库，
	// 空库时必然失败（表还不存在）——所以真正的创建点在 InitializeStorage
	// 的末尾，这里只做 best-effort，失败仅告警，不阻断连接。
	if err := EnsureOperationalIndexes(ctx, db); err != nil {
		log.Printf("ensure operational indexes skipped at connect time (expected on empty database; bootstrap job creates them after schema): %v", err)
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
	client := ent.NewClient(ent.Driver(drv))
	// 注册全局安全拦截器：软删除读透明 + 写路径租户守卫（fail-closed）
	RegisterSecurityInterceptors(client, "")
	return client, nil
}

// InitDatabaseWithRLS 与 InitDatabase 行为完全一致，但在返回 Ent Client 之前
// 用 RLS 装饰器包裹 SQL Driver。
//
// 三档行为（由 rlsCfg.Mode 决定）：
//   - off      (默认)：装饰器为 no-op，零开销、零风险，行为等同 InitDatabase
//   - shadow   ：审计每次查询，warn 缺少 tenant 的调用点；不改变 SQL 语义
//   - enforce  ：审计计数 + 依赖 middleware / AcquireConn 在 conn 上 SET 变量
//
// 说明：本函数不主动执行 SET LOCAL/SESSION。变量注入由 middleware 与
// 显式 rls.AcquireConn 负责，装饰器只是插入观测点。这么设计的原因：
//  1. 一次 request 可能触发多次 Ent 查询，共享同一 *sql.Conn。若在装饰器
//     内每查询前后 SET+RESET，连接来回换值成本高且易出错。
//  2. SET LOCAL 只在事务内生效；Ent 大多数查询是 autocommit，事务边界由
//     业务逻辑控制，装饰器无法感知。
//  3. 装饰器保持无副作用，因此 shadow 模式（只观测、不改数据库行为）可以安全上线。
func InitDatabaseWithRLS(cfg *config.DatabaseConfig, rlsCfg *config.RLSConfig, logger *zap.SugaredLogger) (*ent.Client, error) {
	// 复用 InitDatabase 完整逻辑（连接池、pgvector、schema 兼容修复等）
	client, err := InitDatabase(cfg)
	if err != nil {
		return nil, err
	}
	if rlsCfg == nil || rlsCfg.Mode == "" || rlsCfg.Mode == string(rls.ModeOff) {
		if logger != nil {
			logger.Infow("rls: driver mode=off (default, pass-through)")
		}
		rlsDriver = nil
		return client, nil
	}

	// 装饰：把底层 entsql driver 换成 RLS 装饰器
	innerDrv := entsql.OpenDB("postgres", rawDB)
	deco := rls.From(innerDrv, rlsCfg.Mode, logger)
	rlsDriver = deco
	if logger != nil {
		logger.Infow(
			"rls: driver installed",
			"mode", string(deco.Mode()),
			"tenant_var", rlsCfg.TenantVarName,
		)
	}
	rlsClient := ent.NewClient(ent.Driver(deco))
	RegisterSecurityInterceptors(rlsClient, rlsCfg.Mode)
	return rlsClient, nil
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
