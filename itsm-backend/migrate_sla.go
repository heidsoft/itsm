package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/tenant"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("🚀 开始SLA管理功能数据库迁移...")

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
	if err := migrateSLA(ctx, client); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✅ SLA管理功能数据库迁移完成！")
}

func migrateSLA(ctx context.Context, client *ent.Client) error {
	// 1. 创建SLA相关表
	if err := createSLATables(ctx, client); err != nil {
		return fmt.Errorf("failed to create SLA tables: %w", err)
	}

	// 2. 创建默认SLA定义
	if err := createDefaultSLADefinitions(ctx, client); err != nil {
		return fmt.Errorf("failed to create default SLA definitions: %w", err)
	}

	// 3. 更新现有工单的SLA字段
	if err := updateExistingTickets(ctx, client); err != nil {
		return fmt.Errorf("failed to update existing tickets: %w", err)
	}

	return nil
}

func createSLATables(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 创建SLA相关表...")

	// 获取底层数据库连接
	db := client.Driver().(*sql.DB)

	tables := []string{
		// SLA定义表
		`CREATE TABLE IF NOT EXISTS sla_definitions (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			service_type VARCHAR(100),
			priority VARCHAR(50),
			response_time INTEGER NOT NULL DEFAULT 30,
			resolution_time INTEGER NOT NULL DEFAULT 240,
			business_hours JSONB,
			escalation_rules JSONB,
			conditions JSONB,
			is_active BOOLEAN NOT NULL DEFAULT true,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_sla_definitions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// SLA违规记录表
		`CREATE TABLE IF NOT EXISTS sla_violations (
			id SERIAL PRIMARY KEY,
			sla_definition_id INTEGER NOT NULL,
			ticket_id INTEGER NOT NULL,
			violation_type VARCHAR(100) NOT NULL,
			violation_time TIMESTAMP NOT NULL DEFAULT NOW(),
			description TEXT,
			severity VARCHAR(50) NOT NULL DEFAULT 'medium',
			is_resolved BOOLEAN NOT NULL DEFAULT false,
			resolved_at TIMESTAMP,
			resolution_notes TEXT,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_sla_violations_definition FOREIGN KEY (sla_definition_id) REFERENCES sla_definitions(id),
			CONSTRAINT fk_sla_violations_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id),
			CONSTRAINT fk_sla_violations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// SLA指标表
		`CREATE TABLE IF NOT EXISTS sla_metrics (
			id SERIAL PRIMARY KEY,
			sla_definition_id INTEGER NOT NULL,
			metric_type VARCHAR(100) NOT NULL,
			metric_name VARCHAR(255) NOT NULL,
			metric_value DECIMAL(10,2) NOT NULL,
			unit VARCHAR(50),
			measurement_time TIMESTAMP NOT NULL DEFAULT NOW(),
			metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_sla_metrics_definition FOREIGN KEY (sla_definition_id) REFERENCES sla_definitions(id),
			CONSTRAINT fk_sla_metrics_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,

		// 更新tickets表，添加SLA相关字段
		`ALTER TABLE tickets 
		 ADD COLUMN IF NOT EXISTS sla_definition_id INTEGER,
		 ADD COLUMN IF NOT EXISTS sla_response_deadline TIMESTAMP,
		 ADD COLUMN IF NOT EXISTS sla_resolution_deadline TIMESTAMP,
		 ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMP,
		 ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP`,

		// 添加外键约束
		`ALTER TABLE tickets 
		 ADD CONSTRAINT IF NOT EXISTS fk_tickets_sla_definition 
		 FOREIGN KEY (sla_definition_id) REFERENCES sla_definitions(id)`,

		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_sla_definitions_tenant ON sla_definitions(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_definitions_active ON sla_definitions(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_violations_ticket ON sla_violations(ticket_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_violations_definition ON sla_violations(sla_definition_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_violations_time ON sla_violations(violation_time)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_metrics_definition ON sla_metrics(sla_definition_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sla_metrics_time ON sla_metrics(measurement_time)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_sla_deadline ON tickets(sla_response_deadline, sla_resolution_deadline)`,
	}

	for _, tableSQL := range tables {
		if _, err := db.ExecContext(ctx, tableSQL); err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
	}

	fmt.Println("✅ SLA相关表创建完成")
	return nil
}

func createDefaultSLADefinitions(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 创建默认SLA定义...")

	// 获取默认租户
	defaultTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	// 默认SLA定义
	defaultSLAs := []struct {
		name            string
		description     string
		serviceType     string
		priority        string
		responseTime    int
		resolutionTime  int
		businessHours   map[string]interface{}
		escalationRules map[string]interface{}
		conditions      map[string]interface{}
	}{
		{
			name:           "标准服务SLA",
			description:    "标准IT服务的SLA定义",
			serviceType:    "standard",
			priority:       "medium",
			responseTime:   30,
			resolutionTime: 240,
			businessHours: map[string]interface{}{
				"timezone": "Asia/Shanghai",
				"schedule": []map[string]interface{}{
					{
						"day":     "monday",
						"start":   "09:00",
						"end":     "18:00",
						"enabled": true,
					},
					{
						"day":     "tuesday",
						"start":   "09:00",
						"end":     "18:00",
						"enabled": true,
					},
					{
						"day":     "wednesday",
						"start":   "09:00",
						"end":     "18:00",
						"enabled": true,
					},
					{
						"day":     "thursday",
						"start":   "09:00",
						"end":     "18:00",
						"enabled": true,
					},
					{
						"day":     "friday",
						"start":   "09:00",
						"end":     "18:00",
						"enabled": true,
					},
				},
			},
			escalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{
						"level":        1,
						"time_percent": 50,
						"action":       "notify_manager",
						"recipients":   []string{"manager@company.com"},
					},
					{
						"level":        2,
						"time_percent": 80,
						"action":       "escalate_to_director",
						"recipients":   []string{"director@company.com"},
					},
					{
						"level":        3,
						"time_percent": 100,
						"action":       "critical_escalation",
						"recipients":   []string{"cto@company.com"},
					},
				},
			},
			conditions: map[string]interface{}{
				"priority": []string{"low", "medium"},
				"category": []string{"general", "software", "hardware"},
			},
		},
		{
			name:           "高优先级服务SLA",
			description:    "高优先级IT服务的SLA定义",
			serviceType:    "high_priority",
			priority:       "high",
			responseTime:   15,
			resolutionTime: 120,
			businessHours: map[string]interface{}{
				"timezone": "Asia/Shanghai",
				"schedule": []map[string]interface{}{
					{
						"day":     "monday",
						"start":   "08:00",
						"end":     "20:00",
						"enabled": true,
					},
					{
						"day":     "tuesday",
						"start":   "08:00",
						"end":     "20:00",
						"enabled": true,
					},
					{
						"day":     "wednesday",
						"start":   "08:00",
						"end":     "20:00",
						"enabled": true,
					},
					{
						"day":     "thursday",
						"start":   "08:00",
						"end":     "20:00",
						"enabled": true,
					},
					{
						"day":     "friday",
						"start":   "08:00",
						"end":     "20:00",
						"enabled": true,
					},
				},
			},
			escalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{
						"level":        1,
						"time_percent": 30,
						"action":       "notify_senior_manager",
						"recipients":   []string{"senior_manager@company.com"},
					},
					{
						"level":        2,
						"time_percent": 60,
						"action":       "escalate_to_director",
						"recipients":   []string{"director@company.com"},
					},
					{
						"level":        3,
						"time_percent": 90,
						"action":       "critical_escalation",
						"recipients":   []string{"cto@company.com"},
					},
				},
			},
			conditions: map[string]interface{}{
				"priority": []string{"high", "urgent"},
				"category": []string{"critical", "production", "security"},
			},
		},
		{
			name:           "紧急服务SLA",
			description:    "紧急IT服务的SLA定义",
			serviceType:    "urgent",
			priority:       "urgent",
			responseTime:   5,
			resolutionTime: 60,
			businessHours: map[string]interface{}{
				"timezone": "Asia/Shanghai",
				"schedule": []map[string]interface{}{
					{
						"day":     "monday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "tuesday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "wednesday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "thursday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "friday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "saturday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
					{
						"day":     "sunday",
						"start":   "00:00",
						"end":     "23:59",
						"enabled": true,
					},
				},
			},
			escalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{
						"level":        1,
						"time_percent": 20,
						"action":       "immediate_notification",
						"recipients":   []string{"oncall@company.com", "manager@company.com"},
					},
					{
						"level":        2,
						"time_percent": 50,
						"action":       "escalate_to_director",
						"recipients":   []string{"director@company.com", "cto@company.com"},
					},
					{
						"level":        3,
						"time_percent": 80,
						"action":       "critical_escalation",
						"recipients":   []string{"ceo@company.com", "cto@company.com"},
					},
				},
			},
			conditions: map[string]interface{}{
				"priority": []string{"urgent"},
				"category": []string{"critical", "outage", "security_breach"},
			},
		},
	}

	for _, slaData := range defaultSLAs {
		// 检查是否已存在
		existing, err := client.SLADefinition.Query().
			Where(
				schema.SLADefinition.NameEQ(slaData.name),
				schema.SLADefinition.TenantIDEQ(defaultTenant.ID),
			).
			First(ctx)
		if err == nil {
			fmt.Printf("SLA定义 '%s' 已存在，跳过创建\n", slaData.name)
			continue
		}

		// 创建SLA定义
		_, err = client.SLADefinition.Create().
			SetName(slaData.name).
			SetDescription(slaData.description).
			SetServiceType(slaData.serviceType).
			SetPriority(slaData.priority).
			SetResponseTime(slaData.responseTime).
			SetResolutionTime(slaData.resolutionTime).
			SetBusinessHours(slaData.businessHours).
			SetEscalationRules(slaData.escalationRules).
			SetConditions(slaData.conditions).
			SetIsActive(true).
			SetTenantID(defaultTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create SLA definition '%s': %w", slaData.name, err)
		}

		fmt.Printf("✅ 创建SLA定义: %s\n", slaData.name)
	}

	fmt.Println("✅ 默认SLA定义创建完成")
	return nil
}

func updateExistingTickets(ctx context.Context, client *ent.Client) error {
	fmt.Println("📋 更新现有工单的SLA字段...")

	// 获取默认租户
	defaultTenant, err := client.Tenant.Query().
		Where(tenant.CodeEQ("default")).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	// 获取标准SLA定义
	standardSLA, err := client.SLADefinition.Query().
		Where(
			schema.SLADefinition.NameEQ("标准服务SLA"),
			schema.SLADefinition.TenantIDEQ(defaultTenant.ID),
		).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to get standard SLA definition: %w", err)
	}

	// 更新现有工单，设置默认SLA
	tickets, err := client.Ticket.Query().
		Where(schema.Ticket.TenantIDEQ(defaultTenant.ID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tickets: %w", err)
	}

	updatedCount := 0
	for _, ticket := range tickets {
		// 计算SLA截止时间
		responseDeadline := ticket.CreatedAt.Add(time.Duration(standardSLA.ResponseTime) * time.Minute)
		resolutionDeadline := ticket.CreatedAt.Add(time.Duration(standardSLA.ResolutionTime) * time.Minute)

		_, err := client.Ticket.UpdateOneID(ticket.ID).
			SetSLADefinitionID(standardSLA.ID).
			SetSLAResponseDeadline(responseDeadline).
			SetSLAResolutionDeadline(resolutionDeadline).
			Save(ctx)
		if err != nil {
			fmt.Printf("⚠️ 更新工单 %d 失败: %v\n", ticket.ID, err)
			continue
		}

		updatedCount++
	}

	fmt.Printf("✅ 更新了 %d 个工单的SLA字段\n", updatedCount)
	return nil
}
