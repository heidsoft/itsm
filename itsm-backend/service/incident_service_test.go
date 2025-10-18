package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"
	"itsm-backend/ent/incidentrule"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 测试辅助函数
func setupTestDB(t *testing.T) *ent.Client {
	// 这里应该设置测试数据库连接
	// 简化实现，实际应该使用测试数据库
	return nil
}

func createTestTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("测试租户").
		SetCode("test").
		SetDescription("测试租户描述").
		SetIsActive(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

func createTestUser(ctx context.Context, client *ent.Client, tenantID int) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetFullName("测试用户").
		SetIsActive(true).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
}

// 事件服务测试
func TestIncidentService_CreateIncident(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试租户和用户
	testTenant, err := createTestTenant(ctx, client)
	if err != nil {
		t.Fatalf("创建测试租户失败: %v", err)
	}

	testUser, err := createTestUser(ctx, client, testTenant.ID)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 测试创建事件
		req := &dto.CreateIncidentRequest{
			Title:       "测试事件",
			Description: "这是一个测试事件",
		Priority:    "high",
		Severity:    "medium",
		Category:    "performance",
		Source:      "manual",
	}

	response, err := service.CreateIncident(ctx, req, testTenant.ID)
	if err != nil {
		t.Fatalf("创建事件失败: %v", err)
	}

	// 验证响应
	if response.Title != req.Title {
		t.Errorf("期望标题 %s，实际 %s", req.Title, response.Title)
	}
	if response.Priority != req.Priority {
		t.Errorf("期望优先级 %s，实际 %s", req.Priority, response.Priority)
	}
	if response.Severity != req.Severity {
		t.Errorf("期望严重程度 %s，实际 %s", req.Severity, response.Severity)
	}
	if response.Status != "new" {
		t.Errorf("期望状态 new，实际 %s", response.Status)
	}
	if response.TenantID != testTenant.ID {
		t.Errorf("期望租户ID %d，实际 %d", testTenant.ID, response.TenantID)
	}

	// 验证事件编号格式
	if response.IncidentNumber == "" {
		t.Error("事件编号不应为空")
	}

	// 清理测试数据
	client.Incident.DeleteOneID(response.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func TestIncidentService_GetIncident(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	// 测试获取事件
	response, err := service.GetIncident(ctx, testIncident.ID, testTenant.ID)
	if err != nil {
		t.Fatalf("获取事件失败: %v", err)
	}

	// 验证响应
	if response.ID != testIncident.ID {
		t.Errorf("期望ID %d，实际 %d", testIncident.ID, response.ID)
	}
	if response.Title != testIncident.Title {
		t.Errorf("期望标题 %s，实际 %s", testIncident.Title, response.Title)
	}

	// 测试不存在的ID
	_, err = service.GetIncident(ctx, 99999, testTenant.ID)
	if err == nil {
		t.Error("期望获取不存在的事件时返回错误")
	}

	// 清理测试数据
	client.Incident.DeleteOneID(testIncident.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func TestIncidentService_ListIncidents(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	// 创建多个测试事件
	incidents := make([]*ent.Incident, 3)
	for i := 0; i < 3; i++ {
		incident, err := client.Incident.Create().
			SetTitle(fmt.Sprintf("测试事件 %d", i+1)).
			SetDescription("测试描述").
			SetStatus("new").
			SetPriority("medium").
			SetSeverity("low").
			SetIncidentNumber(fmt.Sprintf("INC-202401-%06d", i+1)).
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建测试事件失败: %v", err)
		}
		incidents[i] = incident
	}

	// 测试获取事件列表
	responses, total, err := service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{})
	if err != nil {
		t.Fatalf("获取事件列表失败: %v", err)
	}

	// 验证响应
	if total != 3 {
		t.Errorf("期望总数 3，实际 %d", total)
	}
	if len(responses) != 3 {
		t.Errorf("期望返回 3 个事件，实际 %d", len(responses))
	}

	// 测试状态筛选
	responses, total, err = service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"status": "new",
	})
	if err != nil {
		t.Fatalf("获取筛选后的事件列表失败: %v", err)
	}
	if total != 3 {
		t.Errorf("期望筛选后总数 3，实际 %d", total)
	}

	// 测试优先级筛选
	responses, total, err = service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"priority": "high",
	})
	if err != nil {
		t.Fatalf("获取筛选后的事件列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("期望筛选后总数 0，实际 %d", total)
	}

	// 清理测试数据
	for _, incident := range incidents {
		client.Incident.DeleteOneID(incident.ID).Exec(ctx)
	}
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func TestIncidentService_UpdateIncident(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("low").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	// 测试更新事件
	newTitle := "更新后的标题"
	newPriority := "high"
	req := &dto.UpdateIncidentRequest{
		Title:    &newTitle,
		Priority: &newPriority,
		Status:   stringPtr("in_progress"),
	}

	response, err := service.UpdateIncident(ctx, testIncident.ID, req, testTenant.ID)
	if err != nil {
		t.Fatalf("更新事件失败: %v", err)
	}

	// 验证响应
	if response.Title != newTitle {
		t.Errorf("期望标题 %s，实际 %s", newTitle, response.Title)
	}
	if response.Priority != newPriority {
		t.Errorf("期望优先级 %s，实际 %s", newPriority, response.Priority)
	}
	if response.Status != "in_progress" {
		t.Errorf("期望状态 in_progress，实际 %s", response.Status)
	}

	// 清理测试数据
	client.Incident.DeleteOneID(testIncident.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func TestIncidentService_DeleteIncident(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("low").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	// 测试删除事件
	err = service.DeleteIncident(ctx, testIncident.ID, testTenant.ID)
	if err != nil {
		t.Fatalf("删除事件失败: %v", err)
	}

	// 验证事件已被删除
	_, err = service.GetIncident(ctx, testIncident.ID, testTenant.ID)
	if err == nil {
		t.Error("期望删除后获取事件时返回错误")
	}

	// 清理测试数据
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

// 事件规则引擎测试
func TestIncidentRuleEngine_ExecuteRule(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewIncidentRuleEngine(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	// 创建测试规则
	testRule, err := client.IncidentRule.Create().
		SetName("测试规则").
		SetDescription("测试规则描述").
		SetRuleType("escalation").
		SetConditions(map[string]interface{}{
			"priority": []string{"high", "urgent"},
		}).
		SetActions([]map[string]interface{}{
			{
				"type":    "escalate",
				"level":   1,
				"reason":  "优先级较高",
			},
		}).
		SetPriority("high").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试规则失败: %v", err)
	}

	// 测试执行规则
	err = engine.ExecuteRule(ctx, testRule, testIncident, testTenant.ID)
	if err != nil {
		t.Fatalf("执行规则失败: %v", err)
	}

	// 验证规则执行记录
	executions, err := client.IncidentRuleExecution.Query().
		Where(incidentruleexecution.RuleIDEQ(testRule.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("获取规则执行记录失败: %v", err)
	}
	if len(executions) == 0 {
		t.Error("期望有规则执行记录")
	}

	// 验证规则执行次数更新
	updatedRule, err := client.IncidentRule.Get(ctx, testRule.ID)
	if err != nil {
		t.Fatalf("获取更新后的规则失败: %v", err)
	}
	if updatedRule.ExecutionCount != 1 {
		t.Errorf("期望执行次数 1，实际 %d", updatedRule.ExecutionCount)
	}

	// 清理测试数据
	client.IncidentRuleExecution.Delete().Exec(ctx)
	client.IncidentRule.DeleteOneID(testRule.ID).Exec(ctx)
	client.Incident.DeleteOneID(testIncident.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

// 事件告警服务测试
func TestIncidentAlertingService_CreateIncidentAlert(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentAlertingService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	// 测试创建告警
	req := &dto.CreateIncidentAlertRequest{
		IncidentID: testIncident.ID,
		AlertType:  "escalation",
		AlertName:  "测试告警",
		Message:    "这是一个测试告警",
		Severity:   "high",
		Channels:   []string{"email", "slack"},
		Recipients: []string{"test@example.com"},
	}

	response, err := service.CreateIncidentAlert(ctx, req, testTenant.ID)
	if err != nil {
		t.Fatalf("创建告警失败: %v", err)
	}

	// 验证响应
	if response.IncidentID != testIncident.ID {
		t.Errorf("期望事件ID %d，实际 %d", testIncident.ID, response.IncidentID)
	}
	if response.AlertType != req.AlertType {
		t.Errorf("期望告警类型 %s，实际 %s", req.AlertType, response.AlertType)
	}
	if response.AlertName != req.AlertName {
		t.Errorf("期望告警名称 %s，实际 %s", req.AlertName, response.AlertName)
	}
	if response.Message != req.Message {
		t.Errorf("期望告警消息 %s，实际 %s", req.Message, response.Message)
	}
	if response.Severity != req.Severity {
		t.Errorf("期望严重程度 %s，实际 %s", req.Severity, response.Severity)
	}
	if response.Status != "active" {
		t.Errorf("期望状态 active，实际 %s", response.Status)
	}

	// 验证告警记录
	alert, err := client.IncidentAlert.Get(ctx, response.ID)
	if err != nil {
		t.Fatalf("获取告警记录失败: %v", err)
	}
	if alert.IncidentID != testIncident.ID {
		t.Errorf("期望事件ID %d，实际 %d", testIncident.ID, alert.IncidentID)
	}

	// 清理测试数据
	client.IncidentAlert.DeleteOneID(response.ID).Exec(ctx)
	client.Incident.DeleteOneID(testIncident.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func TestIncidentAlertingService_AcknowledgeAlert(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	if client == nil {
		t.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentAlertingService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	testIncident, err := client.Incident.Create().
		SetTitle("测试事件").
		SetDescription("测试描述").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试事件失败: %v", err)
	}

	testAlert, err := client.IncidentAlert.Create().
		SetIncidentID(testIncident.ID).
		SetAlertType("escalation").
		SetAlertName("测试告警").
		SetMessage("测试告警消息").
		SetSeverity("high").
		SetStatus("active").
		SetChannels([]string{"email"}).
		SetRecipients([]string{"test@example.com"}).
		SetTriggeredAt(time.Now()).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试告警失败: %v", err)
	}

	// 测试确认告警
	err = service.AcknowledgeAlert(ctx, testAlert.ID, testUser.ID, testTenant.ID)
	if err != nil {
		t.Fatalf("确认告警失败: %v", err)
	}

	// 验证告警状态更新
	updatedAlert, err := client.IncidentAlert.Get(ctx, testAlert.ID)
	if err != nil {
		t.Fatalf("获取更新后的告警失败: %v", err)
	}
	if updatedAlert.Status != "acknowledged" {
		t.Errorf("期望状态 acknowledged，实际 %s", updatedAlert.Status)
	}
	if updatedAlert.AcknowledgedBy == nil || *updatedAlert.AcknowledgedBy != testUser.ID {
		t.Errorf("期望确认人ID %d，实际 %v", testUser.ID, updatedAlert.AcknowledgedBy)
	}
	if updatedAlert.AcknowledgedAt == nil {
		t.Error("期望确认时间不为空")
	}

	// 清理测试数据
	client.IncidentAlert.DeleteOneID(testAlert.ID).Exec(ctx)
	client.Incident.DeleteOneID(testIncident.ID).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

// 基准测试
func BenchmarkIncidentService_CreateIncident(b *testing.B) {
	ctx := context.Background()
	client := setupTestDB(&testing.T{})
	if client == nil {
		b.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(b).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &dto.CreateIncidentRequest{
			Title:       fmt.Sprintf("基准测试事件 %d", i),
			Description: "基准测试描述",
			Priority:    "medium",
			Severity:    "low",
			Category:    "performance",
			Source:      "manual",
		}

		response, err := service.CreateIncident(ctx, req, testTenant.ID)
		if err != nil {
			b.Fatalf("创建事件失败: %v", err)
		}

		// 清理
		client.Incident.DeleteOneID(response.ID).Exec(ctx)
	}

	// 清理测试数据
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}

func BenchmarkIncidentService_ListIncidents(b *testing.B) {
	ctx := context.Background()
	client := setupTestDB(&testing.T{})
	if client == nil {
		b.Skip("测试数据库未配置")
	}

	logger := zaptest.NewLogger(b).Sugar()
	service := NewIncidentService(client, logger)

	// 创建测试数据
	testTenant, _ := createTestTenant(ctx, client)
	testUser, _ := createTestUser(ctx, client, testTenant.ID)

	// 创建测试事件
	for i := 0; i < 100; i++ {
		_, err := client.Incident.Create().
			SetTitle(fmt.Sprintf("基准测试事件 %d", i)).
			SetDescription("基准测试描述").
			SetStatus("new").
			SetPriority("medium").
			SetSeverity("low").
			SetIncidentNumber(fmt.Sprintf("INC-202401-%06d", i)).
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			b.Fatalf("创建测试事件失败: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{})
		if err != nil {
			b.Fatalf("获取事件列表失败: %v", err)
		}
	}

	// 清理测试数据
	client.Incident.Delete().Where(incident.TenantIDEQ(testTenant.ID)).Exec(ctx)
	client.User.DeleteOneID(testUser.ID).Exec(ctx)
	client.Tenant.DeleteOneID(testTenant.ID).Exec(ctx)
}