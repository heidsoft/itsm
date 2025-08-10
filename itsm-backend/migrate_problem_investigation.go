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

	// 创建问题调查相关表
	if err := createProblemInvestigationTables(ctx, rawDB); err != nil {
		log.Fatalf("Failed to create problem investigation tables: %v", err)
	}

	log.Println("Problem investigation tables created successfully")
}

// createProblemInvestigationTables 创建问题调查相关表
func createProblemInvestigationTables(ctx context.Context, db *sql.DB) error {
	// 创建问题调查记录表
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_investigations (
			id SERIAL PRIMARY KEY,
			problem_id INTEGER NOT NULL,
			investigator_id INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
			start_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			estimated_completion_date TIMESTAMP,
			actual_completion_date TIMESTAMP,
			investigation_summary TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
			FOREIGN KEY (investigator_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建问题调查步骤表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_investigation_steps (
			id SERIAL PRIMARY KEY,
			investigation_id INTEGER NOT NULL,
			step_number INTEGER NOT NULL,
			step_title VARCHAR(200) NOT NULL,
			step_description TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			assigned_to INTEGER,
			start_date TIMESTAMP,
			completion_date TIMESTAMP,
			notes TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (investigation_id) REFERENCES problem_investigations(id) ON DELETE CASCADE,
			FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建根本原因分析表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_root_cause_analyses (
			id SERIAL PRIMARY KEY,
			problem_id INTEGER NOT NULL,
			analyst_id INTEGER NOT NULL,
			analysis_method VARCHAR(100) NOT NULL,
			root_cause_description TEXT NOT NULL,
			contributing_factors TEXT,
			evidence TEXT,
			confidence_level VARCHAR(20) NOT NULL DEFAULT 'medium',
			analysis_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			reviewed_by INTEGER,
			review_date TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
			FOREIGN KEY (analyst_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建问题解决方案表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_solutions (
			id SERIAL PRIMARY KEY,
			problem_id INTEGER NOT NULL,
			solution_type VARCHAR(50) NOT NULL,
			solution_description TEXT NOT NULL,
			proposed_by INTEGER NOT NULL,
			proposed_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(20) NOT NULL DEFAULT 'proposed',
			priority VARCHAR(20) NOT NULL DEFAULT 'medium',
			estimated_effort_hours INTEGER,
			estimated_cost DECIMAL(10,2),
			risk_assessment TEXT,
			approval_status VARCHAR(20) NOT NULL DEFAULT 'pending',
			approved_by INTEGER,
			approval_date TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
			FOREIGN KEY (proposed_by) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	// 创建解决方案实施跟踪表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_solution_implementations (
			id SERIAL PRIMARY KEY,
			solution_id INTEGER NOT NULL,
			implementer_id INTEGER NOT NULL,
			implementation_status VARCHAR(20) NOT NULL DEFAULT 'not_started',
			start_date TIMESTAMP,
			completion_date TIMESTAMP,
			actual_effort_hours INTEGER,
			actual_cost DECIMAL(10,2),
			implementation_notes TEXT,
			challenges_encountered TEXT,
			lessons_learned TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (solution_id) REFERENCES problem_solutions(id) ON DELETE CASCADE,
			FOREIGN KEY (implementer_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建问题关联表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_relationships (
			id SERIAL PRIMARY KEY,
			problem_id INTEGER NOT NULL,
			related_type VARCHAR(50) NOT NULL,
			related_id INTEGER NOT NULL,
			relationship_type VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建问题知识库表
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS problem_knowledge_articles (
			id SERIAL PRIMARY KEY,
			problem_id INTEGER NOT NULL,
			article_title VARCHAR(200) NOT NULL,
			article_content TEXT NOT NULL,
			article_type VARCHAR(50) NOT NULL,
			author_id INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'draft',
			published_date TIMESTAMP,
			tags TEXT[],
			view_count INTEGER NOT NULL DEFAULT 0,
			helpful_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_problem_investigations_problem_id ON problem_investigations(problem_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_investigations_status ON problem_investigations(status)",
		"CREATE INDEX IF NOT EXISTS idx_problem_investigation_steps_investigation_id ON problem_investigation_steps(investigation_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_root_cause_analyses_problem_id ON problem_root_cause_analyses(problem_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_solutions_problem_id ON problem_solutions(problem_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_solutions_status ON problem_solutions(status)",
		"CREATE INDEX IF NOT EXISTS idx_problem_solution_implementations_solution_id ON problem_solution_implementations(solution_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_relationships_problem_id ON problem_relationships(problem_id)",
		"CREATE INDEX IF NOT EXISTS idx_problem_knowledge_articles_problem_id ON problem_knowledge_articles(problem_id)",
	}

	for _, index := range indexes {
		if _, err := db.ExecContext(ctx, index); err != nil {
			return err
		}
	}

	return nil
}
