//go:build migrate
// +build migrate

package main

import (
	"context"
	"database/sql"
	"itsm-backend/config"
	"itsm-backend/database"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	// 获取原始数据库连接
	rawDB := database.GetRawDB()
	if rawDB == nil {
		log.Fatalf("Failed to get raw database connection")
	}

	ctx := context.Background()

	// 创建变更审批相关表
	if err := createChangeApprovalTables(ctx, rawDB); err != nil {
		log.Fatalf("Failed to create change approval tables: %v", err)
	}

	log.Println("Change approval tables created successfully")
}

// createChangeApprovalTables 创建变更审批相关表
func createChangeApprovalTables(ctx context.Context, db *sql.DB) error {
	// 创建变更审批记录表
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_approvals (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			approver_id INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			comment TEXT,
			approved_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE,
			FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建变更审批链表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_approval_chains (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			level INTEGER NOT NULL,
			approver_id INTEGER NOT NULL,
			role VARCHAR(50) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			is_required BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE,
			FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建变更风险评估表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_risk_assessments (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			risk_level VARCHAR(20) NOT NULL,
			risk_description TEXT NOT NULL,
			impact_analysis TEXT NOT NULL,
			mitigation_measures TEXT NOT NULL,
			contingency_plan TEXT,
			risk_owner VARCHAR(100) NOT NULL,
			risk_review_date TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建变更实施计划表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_implementation_plans (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			phase VARCHAR(100) NOT NULL,
			description TEXT NOT NULL,
			tasks JSONB NOT NULL,
			responsible VARCHAR(100) NOT NULL,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			prerequisites JSONB,
			dependencies JSONB,
			success_criteria TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建变更回滚计划表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_rollback_plans (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			trigger_conditions JSONB NOT NULL,
			rollback_steps JSONB NOT NULL,
			responsible VARCHAR(100) NOT NULL,
			estimated_time VARCHAR(50) NOT NULL,
			communication_plan TEXT,
			test_plan TEXT,
			approval_required BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建变更回滚执行记录表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS change_rollback_executions (
			id SERIAL PRIMARY KEY,
			change_id INTEGER NOT NULL,
			rollback_plan_id INTEGER NOT NULL,
			trigger_reason TEXT NOT NULL,
			initiated_by INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'initiated',
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			result TEXT,
			comments TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE,
			FOREIGN KEY (rollback_plan_id) REFERENCES change_rollback_plans(id) ON DELETE CASCADE,
			FOREIGN KEY (initiated_by) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_change_approvals_change_id ON change_approvals(change_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_approvals_approver_id ON change_approvals(approver_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_approval_chains_change_id ON change_approval_chains(change_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_approval_chains_level ON change_approval_chains(level)",
		"CREATE INDEX IF NOT EXISTS idx_change_risk_assessments_change_id ON change_risk_assessments(change_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_implementation_plans_change_id ON change_implementation_plans(change_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_rollback_plans_change_id ON change_rollback_plans(change_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_rollback_executions_change_id ON change_rollback_executions(change_id)",
	}

	for _, index := range indexes {
		if _, err := db.ExecContext(ctx, index); err != nil {
			return err
		}
	}

	return nil
}
