package service

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupProblemTest(t *testing.T) (*ent.Client, *ProblemService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewProblemService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createProblemTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createProblemTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser" + suffix).
		SetEmail("test" + suffix + "@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== 创建问题测试 ====================

func TestProblemService_CreateProblem_Success(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	req := &dto.CreateProblemRequest{
		Title:       "数据库连接问题",
		Description: "生产环境数据库间歇性连接失败，影响用户登录",
		Priority:    "high",
		Category:    "infrastructure",
		RootCause:   "待分析",
		Impact:      "用户无法正常登录系统",
	}

	response, err := service.CreateProblem(ctx, req, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Title, response.Title)
	assert.Equal(t, req.Description, response.Description)
	assert.Equal(t, "open", response.Status)
	assert.Equal(t, req.Priority, response.Priority)
	assert.Equal(t, testUser.ID, response.CreatedBy)
	assert.NotEmpty(t, response.CreatedAt)
}

func TestProblemService_CreateProblem_WithCategory(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "cat")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "cat")
	require.NoError(t, err)

	req := &dto.CreateProblemRequest{
		Title:       "网络延迟问题",
		Description: "部分用户反馈访问系统时出现明显延迟",
		Priority:    "medium",
		Category:    "network",
	}

	response, err := service.CreateProblem(ctx, req, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "network", response.Category)
}

// ==================== 获取问题测试 ====================

func TestProblemService_GetProblem_Success(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "get")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "get")
	require.NoError(t, err)

	// 创建测试问题
	testProblem, err := client.Problem.Create().
		SetTitle("Test Problem").
		SetDescription("Test description").
		SetPriority("medium").
		SetStatus("open").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	response, err := service.GetProblem(ctx, testProblem.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, testProblem.ID, response.ID)
	assert.Equal(t, testProblem.Title, response.Title)
}

func TestProblemService_GetProblem_NotFound(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	_, err = service.GetProblem(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "问题不存在")
}

func TestProblemService_GetProblem_TenantMismatch(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant1, err := createProblemTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createProblemTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant1.ID, "tenant")
	require.NoError(t, err)

	testProblem, err := client.Problem.Create().
		SetTitle("Test Problem").
		SetDescription("Test description").
		SetPriority("medium").
		SetStatus("open").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// 使用另一个租户ID获取，应该失败
	_, err = service.GetProblem(ctx, testProblem.ID, testTenant2.ID)
	require.Error(t, err)
}

// ==================== 列出问题测试 ====================

func TestProblemService_ListProblems_Pagination(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建多个测试问题
	for i := 0; i < 15; i++ {
		_, err := client.Problem.Create().
			SetTitle(fmt.Sprintf("Test Problem %d", i+1)).
			SetDescription("Test description").
			SetPriority("medium").
			SetStatus("open").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	response, err := service.ListProblems(ctx, &dto.ListProblemsRequest{Page: 1, PageSize: 10}, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Problems, 10)

	// 测试第二页
	response, err = service.ListProblems(ctx, &dto.ListProblemsRequest{Page: 2, PageSize: 10}, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Problems, 5)
}

func TestProblemService_ListProblems_Filters(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "filter")
	require.NoError(t, err)

	// 创建不同状态和优先级的问题
	statuses := []string{"open", "in_progress", "resolved"}
	priorities := []string{"low", "medium", "high"}

	for _, status := range statuses {
		for _, priority := range priorities {
			_, err := client.Problem.Create().
				SetTitle(fmt.Sprintf("Problem %s-%s", status, priority)).
				SetDescription("Test description").
				SetPriority(priority).
				SetStatus(status).
				SetCreatedBy(testUser.ID).
				SetTenantID(testTenant.ID).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	// 测试状态过滤
	response, err := service.ListProblems(ctx, &dto.ListProblemsRequest{Page: 1, PageSize: 10, Status: "open"}, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, response.Total) // 3 种优先级 × 1 个状态

	// 测试优先级过滤
	response, err = service.ListProblems(ctx, &dto.ListProblemsRequest{Page: 1, PageSize: 10, Priority: "high"}, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, response.Total) // 3 个状态 × 1 个优先级
}

// ==================== 更新问题测试 ====================

func TestProblemService_UpdateProblem_Success(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "update")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "update")
	require.NoError(t, err)

	testProblem, err := client.Problem.Create().
		SetTitle("Original Title").
		SetDescription("Original description").
		SetPriority("low").
		SetStatus("open").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	newTitle := "Updated Title"
	newPriority := "high"

	response, err := service.UpdateProblem(ctx, testProblem.ID, &dto.UpdateProblemRequest{
		Title:    &newTitle,
		Priority: &newPriority,
	}, testTenant.ID)

	require.NoError(t, err)
	assert.Equal(t, newTitle, response.Title)
	assert.Equal(t, newPriority, response.Priority)
}

func TestProblemService_UpdateProblem_StatusTransition(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "status")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "status")
	require.NoError(t, err)

	t.Run("open -> in_progress", func(t *testing.T) {
		testProblem, err := client.Problem.Create().
			SetTitle("Status Test").
			SetDescription("Test").
			SetPriority("medium").
			SetStatus("open").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "in_progress"
		response, err := service.UpdateProblem(ctx, testProblem.ID, &dto.UpdateProblemRequest{
			Status: &newStatus,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
	})

	t.Run("in_progress -> resolved", func(t *testing.T) {
		testProblem, err := client.Problem.Create().
			SetTitle("Status Test 2").
			SetDescription("Test").
			SetPriority("medium").
			SetStatus("in_progress").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "resolved"
		response, err := service.UpdateProblem(ctx, testProblem.ID, &dto.UpdateProblemRequest{
			Status: &newStatus,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
	})
}

// ==================== 删除问题测试 ====================

func TestProblemService_DeleteProblem_Success(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "delete")
	require.NoError(t, err)

	testProblem, err := client.Problem.Create().
		SetTitle("To Be Deleted").
		SetDescription("Test description").
		SetPriority("medium").
		SetStatus("open").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	err = service.DeleteProblem(ctx, testProblem.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已软删除
	deletedProblem, err := client.Problem.Get(ctx, testProblem.ID)
	require.NoError(t, err)
	assert.NotNil(t, deletedProblem.DeletedAt)
}

func TestProblemService_DeleteProblem_NotFound(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	err = service.DeleteProblem(ctx, 99999, testTenant.ID)
	require.Error(t, err)
}

// ==================== 问题统计测试 ====================

func TestProblemService_GetProblemStats(t *testing.T) {
	client, service, ctx := setupProblemTest(t)
	defer client.Close()

	testTenant, err := createProblemTestTenant(ctx, client, "stats")
	require.NoError(t, err)

	testUser, err := createProblemTestUser(ctx, client, testTenant.ID, "stats")
	require.NoError(t, err)

	// 创建不同状态的问题
	statuses := []struct {
		status string
		count  int
	}{
		{"open", 3},
		{"in_progress", 2},
		{"resolved", 4},
		{"closed", 1},
	}

	for _, s := range statuses {
		for i := 0; i < s.count; i++ {
			_, err := client.Problem.Create().
				SetTitle(fmt.Sprintf("Stats Test %s %d", s.status, i)).
				SetDescription("Test description").
				SetPriority("medium").
				SetStatus(s.status).
				SetCreatedBy(testUser.ID).
				SetTenantID(testTenant.ID).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	stats, err := service.GetProblemStats(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	totalExpected := 3 + 2 + 4 + 1 // 10
	assert.Equal(t, totalExpected, stats.Total)
}
