package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/tenant"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("🚀 开始事件管理功能数据库迁移...")

	// 数据库连接
	db, err := sql.Open("postgres", "postgres://dev:123456!@#$%^@localhost/itsm?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 创建Ent客户端
	client := ent.NewClient(ent.Driver(sql.OpenDB("postgres", db)))
	defer client.Close()

	ctx := context.Background()

	// 执行迁移
	if err := migrateIncidentManagement(ctx, client); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✅ 事件管理功能数据库迁移完成！")
}

func migrateIncidentManagement(ctx context.Context, client *ent.Client) error {
	// 1. 创建事件管理相关表
	if err := createIncidentTables(ctx, client); err != nil {
		return fmt.Errorf("failed to create incident tables: %w", err)
	}

	// 2. 创建默认事件规则
	if err := createDefaultIncidentRules(ctx, client); err != nil {
		return fmt.Errorf("failed to create default incident rules: %w", err)
	}

	// 3. 创建示例事件数据
	if err := createSampleIncidents(ctx, client); err != nil {
		return fmt.Errorf("failed to create sample incidents: %w", err)
	}

	return nil
}

func createIncidentTables(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 创建事件管理相关表...")

	// 获取底层数据库连接
	db := client.Driver().(*sql.DB)

	tables := []string{
		// 更新incidents表，添加新字段
		`ALTER TABLE incidents 
		 ADD COLUMN IF NOT EXISTS severity VARCHAR(50) DEFAULT 'medium',
		 ADD COLUMN IF NOT EXISTS category VARCHAR(100),
		 ADD COLUMN IF NOT EXISTS subcategory VARCHAR(100),
		 ADD COLUMN IF NOT EXISTS impact_analysis JSONB,
		 ADD COLUMN IF NOT EXISTS root_cause JSONB,
		 ADD COLUMN IF NOT EXISTS resolution_steps JSONB,
		 ADD COLUMN IF NOT EXISTS detected_at TIMESTAMP DEFAULT NOW(),
		 ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMP,
		 ADD COLUMN IF NOT EXISTS escalation_level INTEGER DEFAULT 0,
		 ADD COLUMN IF NOT EXISTS is_automated BOOLEAN DEFAULT false,
		 ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'manual',
		 ADD COLUMN IF NOT EXISTS metadata JSONB`,

		// 事件活动记录表
		`CREATE TABLE IF NOT EXISTS incident_events (
			id SERIAL PRIMARY KEY,
			incident_id INTEGER NOT NULL,
			event_type VARCHAR(100) NOT NULL,
			event_name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			severity VARCHAR(50) NOT NULL DEFAULT 'medium',
			data JSONB,
			occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),
			user_id INTEGER,
			source VARCHAR(50) NOT NULL DEFAULT 'system',
			metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_incident_events_incident FOREIGN KEY (incident_id) REFERENCES incidents(id),
			CONSTRAINT fk_incident_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 事件告警表
		`CREATE TABLE IF NOT EXISTS incident_alerts (
			id SERIAL PRIMARY KEY,
			incident_id INTEGER NOT NULL,
			alert_type VARCHAR(100) NOT NULL,
			alert_name VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			severity VARCHAR(50) NOT NULL DEFAULT 'medium',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			channels JSONB,
			recipients JSONB,
			triggered_at TIMESTAMP NOT NULL DEFAULT NOW(),
			acknowledged_at TIMESTAMP,
			resolved_at TIMESTAMP,
			acknowledged_by INTEGER,
			metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_incident_alerts_incident FOREIGN KEY (incident_id) REFERENCES incidents(id),
			CONSTRAINT fk_incident_alerts_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 事件指标表
		`CREATE TABLE IF NOT EXISTS incident_metrics (
			id SERIAL PRIMARY KEY,
			incident_id INTEGER NOT NULL,
			metric_type VARCHAR(100) NOT NULL,
			metric_name VARCHAR(255) NOT NULL,
			metric_value DECIMAL(10,2) NOT NULL,
			unit VARCHAR(50),
			measured_at TIMESTAMP NOT NULL DEFAULT NOW(),
			tags JSONB,
			metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_incident_metrics_incident FOREIGN KEY (incident_id) REFERENCES incidents(id),
			CONSTRAINT fk_incident_metrics_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 事件规则表
		`CREATE TABLE IF NOT EXISTS incident_rules (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			rule_type VARCHAR(100) NOT NULL,
			conditions JSONB,
			actions JSONB,
			priority VARCHAR(50) NOT NULL DEFAULT 'medium',
			is_active BOOLEAN NOT NULL DEFAULT true,
			execution_count INTEGER NOT NULL DEFAULT 0,
			last_executed_at TIMESTAMP,
			metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_incident_rules_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 事件规则执行记录表
		`CREATE TABLE IF NOT EXISTS incident_rule_executions (
			id SERIAL PRIMARY KEY,
			rule_id INTEGER NOT NULL,
			incident_id INTEGER,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			result TEXT,
			error_message TEXT,
			started_at TIMESTAMP NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMP,
			execution_time_ms INTEGER,
			input_data JSONB,
			output_data JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_incident_rule_executions_rule FOREIGN KEY (rule_id) REFERENCES incident_rules(id),
			CONSTRAINT fk_incident_rule_executions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_priority ON incidents(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_category ON incidents(category)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_detected_at ON incidents(detected_at)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_escalation_level ON incidents(escalation_level)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_events_incident ON incident_events(incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_events_type ON incident_events(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_events_occurred_at ON incident_events(occurred_at)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_alerts_incident ON incident_alerts(incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_alerts_status ON incident_alerts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_alerts_triggered_at ON incident_alerts(triggered_at)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_metrics_incident ON incident_metrics(incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_metrics_type ON incident_metrics(metric_type)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_metrics_measured_at ON incident_metrics(measured_at)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_rules_active ON incident_rules(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_rules_type ON incident_rules(rule_type)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_rule_executions_rule ON incident_rule_executions(rule_id)`,
		`CREATE INDEX IF NOT EXISTS idx_incident_rule_executions_status ON incident_rule_executions(status)`,
	}

	for _, tableSQL := range tables {
		if _, err := db.ExecContext(ctx, tableSQL); err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
	}

	fmt.Println("✅ 事件管理相关表创建完成")
	return nil
}

func createDefaultIncidentRules(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 创建默认事件规则...")

	// 获取默认租户
	defaultTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	// 默认事件规则
	defaultRules := []struct {
		name        string
		description string
		ruleType    string
		conditions  map[string]interface{}
		actions     []map[string]interface{}
		priority    string
	}{
		{
			name:        "高优先级事件自动升级",
			description: "当事件优先级为high或urgent时，自动升级到下一级别",
			ruleType:    "escalation",
			conditions: map[string]interface{}{
				"priority": []string{"high", "urgent"},
				"status":   "new",
			},
			actions: []map[string]interface{}{
				{
					"type":    "escalate",
					"level":   1,
					"message": "事件优先级较高，已自动升级",
				},
				{
					"type":      "notify",
					"channels":  []string{"email", "sms"},
					"recipients": []string{"manager@company.com"},
				},
			},
			priority: "high",
		},
		{
			name:        "长时间未处理事件告警",
			description: "当事件超过24小时未处理时，发送告警通知",
			ruleType:    "alert",
			conditions: map[string]interface{}{
				"status":      "in_progress",
				"hours_open":  24,
				"assignee_id": nil,
			},
			actions: []map[string]interface{}{
				{
					"type":      "create_alert",
					"severity":  "high",
					"message":   "事件长时间未处理，需要关注",
					"channels":  []string{"email", "slack"},
					"recipients": []string{"team@company.com"},
				},
			},
			priority: "medium",
		},
		{
			name:        "严重事件自动分配",
			description: "当事件严重程度为critical时，自动分配给高级工程师",
			ruleType:    "assignment",
			conditions: map[string]interface{}{
				"severity": "critical",
				"status":   "new",
			},
			actions: []map[string]interface{}{
				{
					"type":        "assign",
					"assignee_id": 1, // 假设ID为1的是高级工程师
					"message":     "严重事件已自动分配给高级工程师",
				},
				{
					"type":      "notify",
					"channels":  []string{"email"},
					"recipients": []string{"senior@company.com"},
				},
			},
			priority: "high",
		},
		{
			name:        "事件解决后自动关闭",
			description: "当事件状态为resolved且超过7天时，自动关闭",
			ruleType:    "auto_close",
			conditions: map[string]interface{}{
				"status":     "resolved",
				"days_since": 7,
			},
			actions: []map[string]interface{}{
				{
					"type":    "close",
					"message": "事件已解决超过7天，自动关闭",
				},
				{
					"type":      "notify",
					"channels":  []string{"email"},
					"recipients": []string{"reporter@company.com"},
				},
			},
			priority: "low",
		},
		{
			name:        "重复事件检测",
			description: "检测相似的事件，避免重复处理",
			ruleType:    "duplicate_detection",
			conditions: map[string]interface{}{
				"similarity_threshold": 0.8,
				"time_window_hours":    24,
			},
			actions: []map[string]interface{}{
				{
					"type":    "link_incidents",
					"message": "检测到相似事件，已关联处理",
				},
				{
					"type":      "notify",
					"channels":  []string{"email"},
					"recipients": []string{"analyst@company.com"},
				},
			},
			priority: "medium",
		},
	}

	for _, ruleData := range defaultRules {
		// 检查是否已存在
		existing, err := client.IncidentRule.Query().
			Where(
				ent.IncidentRule.NameEQ(ruleData.name),
				ent.IncidentRule.TenantIDEQ(defaultTenant.ID),
			).
			First(ctx)
		if err == nil {
			fmt.Printf("事件规则 '%s' 已存在，跳过创建\n", ruleData.name)
			continue
		}

		// 创建事件规则
		_, err = client.IncidentRule.Create().
			SetName(ruleData.name).
			SetDescription(ruleData.description).
			SetRuleType(ruleData.ruleType).
			SetConditions(ruleData.conditions).
			SetActions(ruleData.actions).
			SetPriority(ruleData.priority).
			SetIsActive(true).
			SetExecutionCount(0).
			SetTenantID(defaultTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create incident rule '%s': %w", ruleData.name, err)
		}

		fmt.Printf("✅ 创建事件规则: %s\n", ruleData.name)
	}

	fmt.Println("✅ 默认事件规则创建完成")
	return nil
}

func createSampleIncidents(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 创建示例事件数据...")

	// 获取默认租户
	defaultTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	// 获取测试用户
	testUser, err := client.User.Query().
		Where(
			ent.User.UsernameEQ("testuser"),
			ent.User.TenantIDEQ(defaultTenant.ID),
		).
		First(ctx)
	if err != nil {
		fmt.Println("⚠️ 未找到测试用户，跳过创建示例事件")
		return nil
	}

	// 示例事件数据
	sampleIncidents := []struct {
		title       string
		description string
		priority    string
		severity    string
		category    string
		source      string
	}{
		{
			title:       "服务器CPU使用率过高",
			description: "生产环境Web服务器CPU使用率持续超过90%，影响系统性能",
			priority:    "high",
			severity:    "high",
			category:    "performance",
			source:      "monitoring",
		},
		{
			title:       "数据库连接超时",
			description: "应用程序无法连接到主数据库，出现连接超时错误",
			priority:    "urgent",
			severity:    "critical",
			category:    "connectivity",
			source:      "application",
		},
		{
			title:       "用户登录失败率异常",
			description: "用户登录失败率从正常的5%上升到25%，可能存在安全问题",
			priority:    "medium",
			severity:    "medium",
			category:    "security",
			source:      "analytics",
		},
		{
			title:       "磁盘空间不足告警",
			description: "文件服务器磁盘使用率达到95%，需要清理或扩容",
			priority:    "medium",
			severity:    "medium",
			category:    "storage",
			source:      "monitoring",
		},
		{
			title:       "网络延迟异常",
			description: "内网网络延迟从正常的1ms增加到50ms，影响用户体验",
			priority:    "low",
			severity:    "low",
			category:    "network",
			source:      "monitoring",
		},
	}

	for i, incidentData := range sampleIncidents {
		// 检查是否已存在
		incidentNumber := fmt.Sprintf("INC-%06d", i+1)
		existing, err := client.Incident.Query().
			Where(
				ent.Incident.IncidentNumberEQ(incidentNumber),
				ent.Incident.TenantIDEQ(defaultTenant.ID),
			).
			First(ctx)
		if err == nil {
			fmt.Printf("事件 '%s' 已存在，跳过创建\n", incidentNumber)
			continue
		}

		// 创建事件
		_, err = client.Incident.Create().
			SetTitle(incidentData.title).
			SetDescription(incidentData.description).
			SetStatus("new").
			SetPriority(incidentData.priority).
			SetSeverity(incidentData.severity).
			SetIncidentNumber(incidentNumber).
			SetReporterID(testUser.ID).
			SetCategory(incidentData.category).
			SetSource(incidentData.source).
			SetDetectedAt(time.Now()).
			SetIsAutomated(false).
			SetTenantID(defaultTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create incident '%s': %w", incidentNumber, err)
		}

		fmt.Printf("✅ 创建示例事件: %s\n", incidentNumber)
	}

	fmt.Println("✅ 示例事件数据创建完成")
	return nil
}
