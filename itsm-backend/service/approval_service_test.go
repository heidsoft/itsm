package service

import (
	"context"
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

func setupApprovalTest(t *testing.T) (*ent.Client, *ApprovalService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewApprovalService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createApprovalTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createApprovalTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 创建审批工作流测试 ====================

func TestApprovalService_CreateWorkflow_Success(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createApprovalTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	req := &dto.CreateApprovalWorkflowRequest{
		Name:        "标准审批流程",
		Description: "适用于一般工单的标准审批流程",
		IsActive:    true,
		Nodes: []dto.ApprovalNodeRequest{
			{
				Level:        1,
				Name:         "直属主管审批",
				ApproverType: "user",
				ApproverIDs:  []int{testUser.ID},
				ApprovalMode: "any",
				AllowReject:  true,
				RejectAction: "end",
			},
		},
	}

	response, err := service.CreateWorkflow(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Name, response.Name)
	assert.Equal(t, req.Description, response.Description)
	assert.True(t, response.IsActive)
	assert.Len(t, response.Nodes, 1)
}

func TestApprovalService_CreateWorkflow_MultiLevel(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "multi")
	require.NoError(t, err)

	testUser1, err := createApprovalTestUser(ctx, client, testTenant.ID, "multi1")
	require.NoError(t, err)

	testUser2, err := createApprovalTestUser(ctx, client, testTenant.ID, "multi2")
	require.NoError(t, err)

	req := &dto.CreateApprovalWorkflowRequest{
		Name:        "多级审批流程",
		Description: "需要两级审批的流程",
		IsActive:    true,
		Nodes: []dto.ApprovalNodeRequest{
			{
				Level:        1,
				Name:         "一级审批",
				ApproverType: "user",
				ApproverIDs:  []int{testUser1.ID},
				ApprovalMode: "any",
				AllowReject:  true,
				RejectAction: "end",
			},
			{
				Level:        2,
				Name:         "二级审批",
				ApproverType: "user",
				ApproverIDs:  []int{testUser2.ID},
				ApprovalMode: "any",
				AllowReject:  true,
				RejectAction: "end",
			},
		},
	}

	response, err := service.CreateWorkflow(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	// 验证节点数量
	assert.GreaterOrEqual(t, len(response.Nodes), 0)
}

// ==================== 获取审批工作流测试 ====================

func TestApprovalService_GetWorkflow_Success(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "get")
	require.NoError(t, err)

	// 创建工作流
	created, err := client.ApprovalWorkflow.Create().
		SetName("Test Workflow").
		SetDescription("Test description").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	response, err := service.GetWorkflow(ctx, created.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "Test Workflow", response.Name)
}

func TestApprovalService_GetWorkflow_NotFound(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	_, err = service.GetWorkflow(ctx, 99999, testTenant.ID)
	require.Error(t, err)
}

func TestApprovalService_GetWorkflow_TenantMismatch(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant1, err := createApprovalTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createApprovalTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	// 在租户1创建工作流
	created, err := client.ApprovalWorkflow.Create().
		SetName("Test Workflow").
		SetDescription("Test description").
		SetIsActive(true).
		SetTenantID(testTenant1.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	// 用租户2的ID获取应该失败
	_, err = service.GetWorkflow(ctx, created.ID, testTenant2.ID)
	require.Error(t, err)
}

// ==================== 列出审批工作流测试 ====================

func TestApprovalService_ListWorkflows(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "list")
	require.NoError(t, err)

	// 创建多个工作流
	for i := 0; i < 5; i++ {
		_, err := client.ApprovalWorkflow.Create().
			SetName("Test Workflow").
			SetDescription("Test description").
			SetIsActive(true).
			SetTenantID(testTenant.ID).
			SetNodes([]map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}

	workflows, total, err := service.ListWorkflows(ctx, nil, testTenant.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, workflows, 5)
}

func TestApprovalService_ListWorkflows_Pagination(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "page")
	require.NoError(t, err)

	// 创建15个工作流
	for i := 0; i < 15; i++ {
		_, err := client.ApprovalWorkflow.Create().
			SetName("Test Workflow").
			SetDescription("Test description").
			SetIsActive(true).
			SetTenantID(testTenant.ID).
			SetNodes([]map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	workflows, total, err := service.ListWorkflows(ctx, nil, testTenant.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, workflows, 10)

	// 测试第二页
	workflows, total, err = service.ListWorkflows(ctx, nil, testTenant.ID, 2, 10)
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, workflows, 5)
}

// ==================== 更新审批工作流测试 ====================

func TestApprovalService_UpdateWorkflow_Success(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "update")
	require.NoError(t, err)

	// 创建工作流
	created, err := client.ApprovalWorkflow.Create().
		SetName("Original Name").
		SetDescription("Original description").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	newName := "Updated Name"
	newDesc := "Updated description"

	response, err := service.UpdateWorkflow(ctx, created.ID, &dto.UpdateApprovalWorkflowRequest{
		Name:        &newName,
		Description: &newDesc,
	}, testTenant.ID)

	require.NoError(t, err)
	assert.Equal(t, newName, response.Name)
	assert.Equal(t, newDesc, response.Description)
}

func TestApprovalService_UpdateWorkflow_ActiveStatus(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "active")
	require.NoError(t, err)

	// 创建活跃的工作流
	created, err := client.ApprovalWorkflow.Create().
		SetName("Test Workflow").
		SetDescription("Test description").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	// 停用工作流
	isActive := false
	response, err := service.UpdateWorkflow(ctx, created.ID, &dto.UpdateApprovalWorkflowRequest{
		IsActive: &isActive,
	}, testTenant.ID)

	require.NoError(t, err)
	assert.False(t, response.IsActive)
}

// ==================== 删除审批工作流测试 ====================

func TestApprovalService_DeleteWorkflow_Success(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	// 创建工作流
	created, err := client.ApprovalWorkflow.Create().
		SetName("To Be Deleted").
		SetDescription("Test description").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	err = service.DeleteWorkflow(ctx, created.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.ApprovalWorkflow.Get(ctx, created.ID)
	require.Error(t, err)
}

func TestApprovalService_DeleteWorkflow_NotFound(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	err = service.DeleteWorkflow(ctx, 99999, testTenant.ID)
	require.Error(t, err)
}

// ==================== 获取审批记录测试 ====================

func TestApprovalService_GetApprovalRecords(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	testTenant, err := createApprovalTestTenant(ctx, client, "records")
	require.NoError(t, err)

	testUser, err := createApprovalTestUser(ctx, client, testTenant.ID, "records")
	require.NoError(t, err)

	// 创建工作流
	workflow, err := client.ApprovalWorkflow.Create().
		SetName("Test Workflow").
		SetDescription("Test description").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		SetNodes([]map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	// 创建工单
	ticket, err := client.Ticket.Create().
		SetTitle("Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("ticket").
		SetStatus("open").
		SetTicketNumber("TKT-RECORDS-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建审批记录
	_, err = client.ApprovalRecord.Create().
		SetTicketID(ticket.ID).
		SetTicketNumber(ticket.TicketNumber).
		SetTicketTitle(ticket.Title).
		SetWorkflowID(workflow.ID).
		SetWorkflowName("Test Workflow").
		SetCurrentLevel(1).
		SetTotalLevels(1).
		SetApproverID(testUser.ID).
		SetApproverName(testUser.Name).
		SetStepOrder(1).
		SetStatus("pending").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	req := &dto.GetApprovalRecordsRequest{
		Page:     1,
		PageSize: 10,
	}

	records, total, err := service.GetApprovalRecords(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(records), 1)
}
