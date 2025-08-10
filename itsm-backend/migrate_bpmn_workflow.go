//go:build migrate
// +build migrate

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"itsm-backend/config"
	"itsm-backend/ent"
	"itsm-backend/ent/migrate"

	_ "github.com/lib/pq"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
		cfg.Database.Password)

	// 连接数据库
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 创建Ent客户端
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open ent client: %v", err)
	}
	defer client.Close()

	// 创建上下文
	ctx := context.Background()

	fmt.Println("🚀 开始BPMN工作流引擎数据迁移...")

	// 运行Ent自动迁移
	if err := client.Schema.Create(ctx, migrate.WithDropIndex(true), migrate.WithDropColumn(true)); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	fmt.Println("✅ Ent schema迁移完成")

	// 创建BPMN工作流相关表
	if err := createBPMNTables(ctx, db); err != nil {
		log.Fatalf("Failed to create BPMN tables: %v", err)
	}

	fmt.Println("✅ BPMN工作流表创建完成")

	// 创建索引
	if err := createBPMNIndexes(ctx, db); err != nil {
		log.Fatalf("Failed to create BPMN indexes: %v", err)
	}

	fmt.Println("✅ BPMN工作流索引创建完成")

	// 插入示例数据
	if err := insertSampleData(ctx, client); err != nil {
		log.Fatalf("Failed to insert sample data: %v", err)
	}

	fmt.Println("✅ 示例数据插入完成")

	fmt.Println("🎉 BPMN工作流引擎数据迁移完成！")
}

// createBPMNTables 创建BPMN工作流相关表
func createBPMNTables(ctx context.Context, db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS process_definitions (
			id SERIAL PRIMARY KEY,
			key VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
			category VARCHAR(100) NOT NULL DEFAULT 'default',
			bpmn_xml BYTEA NOT NULL,
			process_variables JSONB,
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_latest BOOLEAN NOT NULL DEFAULT true,
			deployment_id INTEGER NOT NULL,
			deployment_name VARCHAR(255),
			deployed_at TIMESTAMP NOT NULL DEFAULT NOW(),
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS process_instances (
			id SERIAL PRIMARY KEY,
			process_instance_id VARCHAR(255) NOT NULL UNIQUE,
			business_key VARCHAR(255),
			process_definition_key VARCHAR(255) NOT NULL,
			process_definition_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'running',
			current_activity_id VARCHAR(255),
			current_activity_name VARCHAR(255),
			variables JSONB,
			start_time TIMESTAMP NOT NULL DEFAULT NOW(),
			end_time TIMESTAMP,
			suspended_time TIMESTAMP,
			suspended_reason TEXT,
			tenant_id INTEGER NOT NULL,
			initiator VARCHAR(255),
			parent_process_instance_id VARCHAR(255),
			root_process_instance_id VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS process_tasks (
			id SERIAL PRIMARY KEY,
			task_id VARCHAR(255) NOT NULL UNIQUE,
			process_instance_id VARCHAR(255) NOT NULL,
			process_definition_key VARCHAR(255) NOT NULL,
			task_definition_key VARCHAR(255) NOT NULL,
			task_name VARCHAR(255) NOT NULL,
			task_type VARCHAR(50) NOT NULL DEFAULT 'user_task',
			assignee VARCHAR(255),
			candidate_users TEXT,
			candidate_groups TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'created',
			priority VARCHAR(20) NOT NULL DEFAULT 'normal',
			due_date TIMESTAMP,
			created_time TIMESTAMP NOT NULL DEFAULT NOW(),
			assigned_time TIMESTAMP,
			started_time TIMESTAMP,
			completed_time TIMESTAMP,
			form_key VARCHAR(255),
			task_variables JSONB,
			description TEXT,
			parent_task_id VARCHAR(255),
			root_task_id VARCHAR(255),
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS process_variables (
			id SERIAL PRIMARY KEY,
			variable_id VARCHAR(255) NOT NULL UNIQUE,
			process_instance_id VARCHAR(255) NOT NULL,
			task_id VARCHAR(255),
			variable_name VARCHAR(255) NOT NULL,
			variable_type VARCHAR(50) NOT NULL DEFAULT 'string',
			variable_value TEXT,
			scope VARCHAR(20) NOT NULL DEFAULT 'process',
			is_transient BOOLEAN NOT NULL DEFAULT false,
			serialization_format VARCHAR(20) NOT NULL DEFAULT 'json',
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS process_deployments (
			id SERIAL PRIMARY KEY,
			deployment_id VARCHAR(255) NOT NULL UNIQUE,
			deployment_name VARCHAR(255) NOT NULL,
			deployment_source TEXT,
			deployment_time TIMESTAMP NOT NULL DEFAULT NOW(),
			deployed_by VARCHAR(255),
			deployment_comment TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			deployment_category VARCHAR(100) NOT NULL DEFAULT 'default',
			deployment_metadata JSONB,
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS process_execution_history (
			id SERIAL PRIMARY KEY,
			history_id VARCHAR(255) NOT NULL UNIQUE,
			process_instance_id VARCHAR(255) NOT NULL,
			process_definition_key VARCHAR(255) NOT NULL,
			activity_id VARCHAR(255),
			activity_name VARCHAR(255),
			activity_type VARCHAR(50) NOT NULL,
			event_type VARCHAR(50) NOT NULL,
			event_detail TEXT,
			variables JSONB,
			user_id VARCHAR(255),
			user_name VARCHAR(255),
			timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
			comment TEXT,
			error_message TEXT,
			error_code VARCHAR(100),
			tenant_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
	}

	for i, tableSQL := range tables {
		if _, err := db.ExecContext(ctx, tableSQL); err != nil {
			return fmt.Errorf("failed to create table %d: %v", err)
		}
		fmt.Printf("✅ 表 %d 创建成功\n", i+1)
	}

	return nil
}

// createBPMNIndexes 创建BPMN工作流相关索引
func createBPMNIndexes(ctx context.Context, db *sql.DB) error {
	indexes := []string{
		// ProcessDefinition 索引
		`CREATE INDEX IF NOT EXISTS idx_process_definitions_key_version ON process_definitions(key, version)`,
		`CREATE INDEX IF NOT EXISTS idx_process_definitions_tenant_key ON process_definitions(tenant_id, key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_definitions_deployment_id ON process_definitions(deployment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_definitions_is_active ON process_definitions(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_process_definitions_is_latest ON process_definitions(is_latest)`,

		// ProcessInstance 索引
		`CREATE INDEX IF NOT EXISTS idx_process_instances_process_instance_id ON process_instances(process_instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_business_key ON process_instances(business_key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_process_definition_key ON process_instances(process_definition_key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_process_definition_id ON process_instances(process_definition_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_status ON process_instances(status)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_tenant_id ON process_instances(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_initiator ON process_instances(initiator)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_start_time ON process_instances(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_parent_process_instance_id ON process_instances(parent_process_instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_instances_root_process_instance_id ON process_instances(root_process_instance_id)`,

		// ProcessTask 索引
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_task_id ON process_tasks(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_process_instance_id ON process_tasks(process_instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_process_definition_key ON process_tasks(process_definition_key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_task_definition_key ON process_tasks(task_definition_key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_assignee ON process_tasks(assignee)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_status ON process_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_priority ON process_tasks(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_due_date ON process_tasks(due_date)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_tenant_id ON process_tasks(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_created_time ON process_tasks(created_time)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_parent_task_id ON process_tasks(parent_task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_tasks_root_task_id ON process_tasks(root_task_id)`,

		// ProcessVariable 索引
		`CREATE INDEX IF NOT EXISTS idx_process_variables_variable_id ON process_variables(variable_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_process_instance_id ON process_variables(process_instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_task_id ON process_variables(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_variable_name ON process_variables(variable_name)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_scope ON process_variables(scope)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_tenant_id ON process_variables(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_variables_created_at ON process_variables(created_at)`,

		// ProcessDeployment 索引
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_deployment_id ON process_deployments(deployment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_deployment_name ON process_deployments(deployment_name)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_deployment_time ON process_deployments(deployment_time)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_deployed_by ON process_deployments(deployed_by)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_is_active ON process_deployments(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_deployment_category ON process_deployments(deployment_category)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_tenant_id ON process_deployments(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_deployments_created_at ON process_deployments(created_at)`,

		// ProcessExecutionHistory 索引
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_history_id ON process_execution_history(history_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_process_instance_id ON process_execution_history(process_instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_process_definition_key ON process_execution_history(process_definition_key)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_activity_id ON process_execution_history(activity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_activity_type ON process_execution_history(activity_type)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_event_type ON process_execution_history(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_user_id ON process_execution_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_timestamp ON process_execution_history(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_tenant_id ON process_execution_history(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_process_execution_history_created_at ON process_execution_history(created_at)`,
	}

	for i, indexSQL := range indexes {
		if _, err := db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index %d: %v", err)
		}
		fmt.Printf("✅ 索引 %d 创建成功\n", i+1)
	}

	return nil
}

// insertSampleData 插入示例数据
func insertSampleData(ctx context.Context, client *ent.Client) error {
	// 创建示例租户
	_, err := client.Tenant.Create().
		SetName("示例租户").
		SetCode("sample-tenant").
		SetType("standard").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample tenant: %v", err)
	}

	fmt.Println("✅ 示例租户创建成功")

	// 创建示例流程部署
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("sample-deployment-001").
		SetDeploymentName("示例流程部署").
		SetDeploymentSource("manual").
		SetDeployedBy("admin").
		SetDeploymentComment("用于测试的示例流程部署").
		SetIsActive(true).
		SetDeploymentCategory("sample").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample deployment: %v", err)
	}

	fmt.Println("✅ 示例流程部署创建成功")

	// 创建示例流程定义
	processDef, err := client.ProcessDefinition.Create().
		SetKey("sample-process").
		SetName("示例流程").
		SetDescription("这是一个用于测试的示例BPMN流程").
		SetVersion("1.0.0").
		SetCategory("sample").
		SetBpmnXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="sample-process" name="示例流程">
    <bpmn:startEvent id="start-event" name="开始"/>
    <bpmn:userTask id="user-task" name="用户任务"/>
    <bpmn:endEvent id="end-event" name="结束"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start-event" targetRef="user-task"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="user-task" targetRef="end-event"/>
  </bpmn:process>
</bpmn:definitions>`)).
		SetProcessVariables(map[string]interface{}{
			"initiator": "string",
			"priority":  "string",
		}).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(deployment.DeploymentID).
		SetDeploymentName(deployment.DeploymentName).
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample process definition: %v", err)
	}

	fmt.Println("✅ 示例流程定义创建成功")

	// 创建示例流程实例
	processInstance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("sample-instance-001").
		SetBusinessKey("SAMPLE-001").
		SetProcessDefinitionKey(processDef.Key).
		SetProcessDefinitionID(processDef.ID).
		SetStatus("running").
		SetCurrentActivityID("user-task").
		SetCurrentActivityName("用户任务").
		SetVariables(map[string]interface{}{
			"initiator": "user123",
			"priority":  "normal",
		}).
		SetTenantID(tenant.ID).
		SetInitiator("user123").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample process instance: %v", err)
	}

	fmt.Println("✅ 示例流程实例创建成功")

	// 创建示例流程任务
	_, err = client.ProcessTask.Create().
		SetTaskID("sample-task-001").
		SetProcessInstanceID(processInstance.ProcessInstanceID).
		SetProcessDefinitionKey(processDef.Key).
		SetTaskDefinitionKey("user-task").
		SetTaskName("用户任务").
		SetTaskType("user_task").
		SetAssignee("user456").
		SetStatus("assigned").
		SetPriority("normal").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample process task: %v", err)
	}

	fmt.Println("✅ 示例流程任务创建成功")

	// 创建示例流程变量
	_, err = client.ProcessVariable.Create().
		SetVariableID("sample-variable-001").
		SetProcessInstanceID(processInstance.ProcessInstanceID).
		SetVariableName("task_comment").
		SetVariableType("string").
		SetVariableValue("这是一个示例任务").
		SetScope("task").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample process variable: %v", err)
	}

	fmt.Println("✅ 示例流程变量创建成功")

	// 创建示例执行历史
	_, err = client.ProcessExecutionHistory.Create().
		SetHistoryID("sample-history-001").
		SetProcessInstanceID(processInstance.ProcessInstanceID).
		SetProcessDefinitionKey(processDef.Key).
		SetActivityID("start-event").
		SetActivityName("开始").
		SetActivityType("start_event").
		SetEventType("start").
		SetEventDetail("流程实例启动").
		SetVariables(map[string]interface{}{
			"initiator": "user123",
		}).
		SetUserID("user123").
		SetUserName("用户123").
		SetTenantID(tenant.ID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sample execution history: %v", err)
	}

	fmt.Println("✅ 示例执行历史创建成功")

	return nil
}
