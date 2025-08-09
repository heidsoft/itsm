package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/incident"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func setupTestIncidentService(t *testing.T) (*IncidentService, *ent.Client, func()) {
	// 使用内存数据库进行测试
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建schema
	err := client.Schema.Create(context.Background())
	require.NoError(t, err)

	// 创建测试用户
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	_, err = client.User.Create().
		SetUsername("testuser").
		SetPasswordHash(string(passwordHash)).
		SetName("测试用户").
		SetEmail("test@example.com").
		SetActive(true).
		SetTenantID(1).
		Save(context.Background())
	require.NoError(t, err)

	// 创建测试配置项
	_, err = client.ConfigurationItem.Create().
		SetName("测试服务器").
		SetDescription("测试用服务器").
		SetType("server").
		SetStatus("active").
		SetTenantID(1).
		Save(context.Background())
	require.NoError(t, err)

	// 创建服务实例
	logger, _ := zap.NewDevelopment()
	service := NewIncidentService(client, logger.Sugar())

	// 清理函数
	cleanup := func() {
		client.Close()
	}

	return service, client, cleanup
}

func TestCreateIncident(t *testing.T) {
	service, client, cleanup := setupTestIncidentService(t)
	defer cleanup()

	ctx := context.Background()

	// 测试用例1：创建基本事件
	t.Run("创建基本事件", func(t *testing.T) {
		req := &dto.CreateIncidentRequest{
			Title:       "测试事件",
			Description: "这是一个测试事件",
			Priority:    "medium",
		}

		incident, err := service.CreateIncident(ctx, req, 1, 1)
		require.NoError(t, err)
		assert.NotNil(t, incident)
		assert.Equal(t, "测试事件", incident.Title)
		assert.Equal(t, "这是一个测试事件", incident.Description)
		assert.Equal(t, "medium", incident.Priority)
		assert.Equal(t, dto.IncidentStatusNew, incident.Status)
		assert.NotEmpty(t, incident.IncidentNumber)
		assert.Equal(t, 1, incident.ReporterID)
		assert.Equal(t, 1, incident.TenantID)
	})

	// 测试用例2：创建高优先级事件
	t.Run("创建高优先级事件", func(t *testing.T) {
		req := &dto.CreateIncidentRequest{
			Title:       "紧急事件",
			Description: "这是一个紧急事件",
			Priority:    "urgent",
			AssigneeID:  &[]int{1}[0],
		}

		incident, err := service.CreateIncident(ctx, req, 1, 1)
		require.NoError(t, err)
		assert.NotNil(t, incident)
		assert.Equal(t, "urgent", incident.Priority)
		assert.Equal(t, 1, *incident.AssigneeID)
	})

	// 测试用例3：创建关联配置项的事件
	t.Run("创建关联配置项的事件", func(t *testing.T) {
		// 先创建一个配置项
		ci, err := client.ConfigurationItem.Create().
			SetName("数据库服务器").
			SetDescription("生产环境数据库").
			SetType("database").
			SetStatus("active").
			SetTenantID(1).
			Save(ctx)
		require.NoError(t, err)

		req := &dto.CreateIncidentRequest{
			Title:               "数据库连接异常",
			Description:         "数据库连接超时",
			Priority:            "high",
			ConfigurationItemID: &ci.ID,
		}

		incident, err := service.CreateIncident(ctx, req, 1, 1)
		require.NoError(t, err)
		assert.NotNil(t, incident)
		assert.Equal(t, ci.ID, *incident.ConfigurationItemID)
	})

	// 测试用例4：验证事件编号生成
	t.Run("验证事件编号生成", func(t *testing.T) {
		req := &dto.CreateIncidentRequest{
			Title:       "事件1",
			Description: "测试事件编号",
			Priority:    "low",
		}

		incident1, err := service.CreateIncident(ctx, req, 1, 1)
		require.NoError(t, err)

		req.Title = "事件2"
		incident2, err := service.CreateIncident(ctx, req, 1, 1)
		require.NoError(t, err)

		// 验证事件编号不同且格式正确
		assert.NotEqual(t, incident1.IncidentNumber, incident2.IncidentNumber)
		assert.Contains(t, incident1.IncidentNumber, "INC-")
		assert.Contains(t, incident2.IncidentNumber, "INC-")
	})
}

func TestGetIncidents(t *testing.T) {
	service, _, cleanup := setupTestIncidentService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	createTestIncidents(t, service.client)

	// 测试用例1：获取所有事件
	t.Run("获取所有事件", func(t *testing.T) {
		req := &dto.GetIncidentsRequest{
			Page:     1,
			Size:     10,
			TenantID: 1,
		}

		result, err := service.GetIncidents(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Total, 3) // 至少应该有3个事件
		assert.Len(t, result.Incidents, result.Total)
	})

	// 测试用例2：按状态过滤
	t.Run("按状态过滤", func(t *testing.T) {
		req := &dto.GetIncidentsRequest{
			Page:     1,
			Size:     10,
			Status:   dto.IncidentStatusNew,
			TenantID: 1,
		}

		result, err := service.GetIncidents(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证所有返回的事件都是新建状态
		for _, incident := range result.Incidents {
			assert.Equal(t, dto.IncidentStatusNew, incident.Status)
		}
	})

	// 测试用例3：按优先级过滤
	t.Run("按优先级过滤", func(t *testing.T) {
		req := &dto.GetIncidentsRequest{
			Page:     1,
			Size:     10,
			Priority: dto.IncidentPriorityHigh,
			TenantID: 1,
		}

		result, err := service.GetIncidents(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证所有返回的事件都是高优先级
		for _, incident := range result.Incidents {
			assert.Equal(t, dto.IncidentPriorityHigh, incident.Priority)
		}
	})
}

func TestUpdateIncident(t *testing.T) {
	service, _, cleanup := setupTestIncidentService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试事件
	req := &dto.CreateIncidentRequest{
		Title:       "原始事件",
		Description: "原始描述",
		Priority:    "medium",
	}

	incident, err := service.CreateIncident(ctx, req, 1, 1)
	require.NoError(t, err)

	// 测试用例1：更新事件标题和描述
	t.Run("更新事件标题和描述", func(t *testing.T) {
		updateReq := &dto.UpdateIncidentRequest{
			Title:       &[]string{"更新后的事件标题"}[0],
			Description: &[]string{"更新后的描述"}[0],
		}

		updatedIncident, err := service.UpdateIncident(ctx, incident.ID, updateReq, 1)
		require.NoError(t, err)
		assert.Equal(t, "更新后的事件标题", updatedIncident.Title)
		assert.Equal(t, "更新后的描述", updatedIncident.Description)
	})

	// 测试用例2：更新事件状态为已解决
	t.Run("更新事件状态为已解决", func(t *testing.T) {
		status := dto.IncidentStatusResolved
		updateReq := &dto.UpdateIncidentRequest{
			Status: &status,
		}

		updatedIncident, err := service.UpdateIncident(ctx, incident.ID, updateReq, 1)
		require.NoError(t, err)
		assert.Equal(t, dto.IncidentStatusResolved, updatedIncident.Status)
		assert.NotNil(t, updatedIncident.ResolvedAt)
	})

	// 测试用例3：更新事件优先级
	t.Run("更新事件优先级", func(t *testing.T) {
		priority := dto.IncidentPriorityUrgent
		updateReq := &dto.UpdateIncidentRequest{
			Priority: &priority,
		}

		updatedIncident, err := service.UpdateIncident(ctx, incident.ID, updateReq, 1)
		require.NoError(t, err)
		assert.Equal(t, dto.IncidentPriorityUrgent, updatedIncident.Priority)
	})
}

func TestCloseIncident(t *testing.T) {
	service, _, cleanup := setupTestIncidentService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试事件
	req := &dto.CreateIncidentRequest{
		Title:       "待关闭事件",
		Description: "这是一个待关闭的事件",
		Priority:    "medium",
	}

	incident, err := service.CreateIncident(ctx, req, 1, 1)
	require.NoError(t, err)

	// 测试用例1：关闭事件
	t.Run("关闭事件", func(t *testing.T) {
		err := service.CloseIncident(ctx, incident.ID, 1)
		require.NoError(t, err)

		// 验证事件状态已更新
		updatedIncident, err := service.GetIncident(ctx, incident.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, dto.IncidentStatusClosed, updatedIncident.Status)
		assert.NotNil(t, updatedIncident.ClosedAt)
	})

	// 测试用例2：重复关闭事件
	t.Run("重复关闭事件", func(t *testing.T) {
		err := service.CloseIncident(ctx, incident.ID, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "事件已经关闭")
	})
}

func TestGetIncidentStats(t *testing.T) {
	service, client, cleanup := setupTestIncidentService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	createTestIncidents(t, client)

	// 测试获取事件统计
	t.Run("获取事件统计", func(t *testing.T) {
		stats, err := service.GetIncidentStats(ctx, 1)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.TotalIncidents, 3)
		assert.GreaterOrEqual(t, stats.OpenIncidents, 1)
	})
}

func TestIncidentService_GetIncidentStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	zl, _ := zap.NewDevelopment()
	s := NewIncidentService(client, zl.Sugar())
	ctx := context.Background()

	// seed incidents for tenant 1 and 2
	_, _ = client.Incident.Create().SetIncidentNumber("INC-1").SetTitle("t1").SetDescription("d").SetStatus("new").SetPriority("high").SetReporterID(1).SetTenantID(1).Save(ctx)
	_, _ = client.Incident.Create().SetIncidentNumber("INC-2").SetTitle("t1").SetDescription("d").SetStatus("resolved").SetPriority("urgent").SetReporterID(1).SetTenantID(1).Save(ctx)
	_, _ = client.Incident.Create().SetIncidentNumber("INC-3").SetTitle("t2").SetDescription("d").SetStatus("closed").SetPriority("low").SetReporterID(2).SetTenantID(2).Save(ctx)

	stats, err := s.GetIncidentStats(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalIncidents != 2 {
		t.Fatalf("want total=2 got=%d", stats.TotalIncidents)
	}
	if stats.UrgentIncidents != 1 {
		t.Fatalf("want urgent=1 got=%d", stats.UrgentIncidents)
	}

	// ensure filtering by tenant works
	stats2, err := s.GetIncidentStats(ctx, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats2.TotalIncidents != 1 {
		t.Fatalf("want total=1 got=%d", stats2.TotalIncidents)
	}

	// verify resolved counted for tenant 1
	countResolved, _ := client.Incident.Query().Where(incident.StatusEQ("resolved"), incident.TenantID(1)).Count(ctx)
	if countResolved != stats.ResolvedIncidents {
		t.Fatalf("resolved mismatch: query=%d stats=%d", countResolved, stats.ResolvedIncidents)
	}
}

// 辅助函数：创建测试事件数据
func createTestIncidents(t *testing.T, client *ent.Client) {
	ctx := context.Background()

	// 创建多个测试事件
	incidents := []struct {
		title    string
		priority string
		status   string
	}{
		{"低优先级事件", dto.IncidentPriorityLow, dto.IncidentStatusNew},
		{"高优先级事件", dto.IncidentPriorityHigh, dto.IncidentStatusInProgress},
		{"紧急事件", dto.IncidentPriorityUrgent, dto.IncidentStatusResolved},
	}

	for i, incident := range incidents {
		_, err := client.Incident.Create().
			SetIncidentNumber(fmt.Sprintf("INC-2024-%05d", i+1)).
			SetTitle(incident.title).
			SetDescription(fmt.Sprintf("这是第%d个测试事件", i+1)).
			SetStatus(incident.status).
			SetPriority(incident.priority).
			SetReporterID(1).
			SetTenantID(1).
			Save(ctx)
		require.NoError(t, err)
	}
}
