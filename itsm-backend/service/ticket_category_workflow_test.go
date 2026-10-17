package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 工单分类的 workflow_definition_key 必须是建单选路的真实输入。
// 优先级口径：请求显式 > 工单类型 > 工单分类 > 流程绑定表 > 内置兜底；
// 分类写入的流程 Key 必须能在本租户解析到激活的流程定义，否则保存就该失败，
// 而不是「保存成功但工单仍走默认流程」。

func catFlowTenantAndRequester(t *testing.T, client *ent.Client, ctx context.Context, suffix string) (*ent.Tenant, *ent.User) {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Cat Flow Tenant " + suffix).SetCode("cat-flow-" + suffix).
		SetDomain("cat-flow-" + suffix + ".example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().
		SetUsername("cat_requester_" + suffix).SetEmail("cat-" + suffix + "@example.com").
		SetName("Cat Requester").SetPasswordHash("hash").SetRole("end_user").
		SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	return tenant, requester
}

func deployFlow(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, key string) {
	t.Helper()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("dep-%s-%d", key, tenantID)).
		SetDeploymentName("deploy " + key).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey(key).SetName("流程 " + key).SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
}

func newCategoryWithFlow(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, code, flowKey string) int {
	t.Helper()
	svc := NewTicketCategoryService(client)
	key := flowKey
	category, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name: code, Code: code, TenantID: tenantID, IsActive: true, WorkflowDefinitionKey: &key,
	})
	require.NoError(t, err)
	return category.ID
}

// workflowStartPayloadKey 读取工单创建同事务入箱的 workflow.start 命令里的流程 Key。
func workflowStartPayloadKey(t *testing.T, client *ent.Client, ctx context.Context, tenantID, ticketID int) string {
	t.Helper()
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("ticket"),
		operationalcommand.AggregateIDEQ(ticketID),
	).Only(ctx)
	require.NoError(t, err)
	key, _ := cmd.Payload["workflowDefinitionKey"].(string)
	return key
}

func TestCreateTicket_CategoryWorkflowKeyDrivesRouting(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, requester := catFlowTenantAndRequester(t, client, ctx, "cat-only")
	deployFlow(t, client, ctx, tenant.ID, "cat_default_flow")
	categoryID := newCategoryWithFlow(t, client, ctx, tenant.ID, "cat-only", "cat_default_flow")

	svc := NewTicketServiceForTest(client, logger)
	svc.EnableWorkflowOutbox()

	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "分类默认流程", Description: "只带 categoryId，流程 Key 必须来自分类",
		Priority: "medium", RequesterID: requester.ID, CategoryID: &categoryID,
	}, tenant.ID)
	require.NoError(t, err)

	require.Equal(t, "cat_default_flow", workflowStartPayloadKey(t, client, ctx, tenant.ID, created.ID))
}

// TestCreateTicket_WorkflowKeyPrecedence 锁死三级绑定来源的先后顺序，
// 防止分类绑定覆盖工单类型，或请求显式指定被静默改写。
func TestCreateTicket_WorkflowKeyPrecedence(t *testing.T) {
	tests := []struct {
		slug        string
		name        string
		sendReqKey  bool
		typeHasFlow bool
		wantKey     string
	}{
		{slug: "cat", name: "只有分类_用分类", wantKey: "cat_flow"},
		{slug: "type", name: "类型与分类_用类型", typeHasFlow: true, wantKey: "type_flow"},
		{slug: "req", name: "请求与类型与分类_用请求", sendReqKey: true, typeHasFlow: true, wantKey: "req_flow"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", testDSN())
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, requester := catFlowTenantAndRequester(t, client, ctx, tc.slug)
			for _, key := range []string{"cat_flow", "type_flow", "req_flow"} {
				deployFlow(t, client, ctx, tenant.ID, key)
			}
			categoryID := newCategoryWithFlow(t, client, ctx, tenant.ID, "prec-cat-"+tc.slug, "cat_flow")

			var typeID *int
			typeReq := &dto.CreateTicketTypeRequest{Code: "prec_type_" + tc.slug, Name: "优先级测试类型"}
			if tc.typeHasFlow {
				typeReq.WorkflowDefinitionKey = "type_flow"
			}
			createdType, err := NewTicketTypeService(client, logger).CreateTicketType(ctx, typeReq, tenant.ID, requester.ID)
			require.NoError(t, err)
			typeID = &createdType.ID

			req := &dto.CreateTicketRequest{
				Title: "优先级 " + tc.name, Description: "绑定来源优先级", Priority: "medium",
				RequesterID: requester.ID, TicketTypeID: typeID, CategoryID: &categoryID,
			}
			if tc.sendReqKey {
				req.WorkflowDefinitionKey = "req_flow"
			}

			svc := NewTicketServiceForTest(client, logger)
			svc.EnableWorkflowOutbox()
			created, err := svc.CreateTicket(ctx, req, tenant.ID)
			require.NoError(t, err)

			require.Equal(t, tc.wantKey, workflowStartPayloadKey(t, client, ctx, tenant.ID, created.ID))
		})
	}
}

// TestCreateTicket_CategoryResolvedByNameStillRoutes 前端下拉常只传分类名，
// 按名解析出的分类同样要参与选路。
func TestCreateTicket_CategoryResolvedByNameStillRoutes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, requester := catFlowTenantAndRequester(t, client, ctx, "by-name")
	deployFlow(t, client, ctx, tenant.ID, "named_cat_flow")
	newCategoryWithFlow(t, client, ctx, tenant.ID, "named-cat", "named_cat_flow")

	svc := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	svc.EnableWorkflowOutbox()
	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "按名解析分类", Description: "只传 category 名称", Priority: "medium",
		RequesterID: requester.ID, Type: "incident", Category: "named-cat",
	}, tenant.ID)
	require.NoError(t, err)

	require.Equal(t, "named_cat_flow", workflowStartPayloadKey(t, client, ctx, tenant.ID, created.ID))
}

// TestCreateTicket_UnboundCategoryLeavesRoutingToBindings 分类未绑定流程时，
// 命令 payload 必须保持为空，由消费侧继续走绑定表与内置兜底。
func TestCreateTicket_UnboundCategoryLeavesRoutingToBindings(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, requester := catFlowTenantAndRequester(t, client, ctx, "unbound")
	category, err := client.TicketCategory.Create().
		SetName("unbound-cat").SetCode("unbound-cat").SetTenantID(tenant.ID).
		SetIsActive(true).SetLevel(1).Save(ctx)
	require.NoError(t, err)

	svc := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	svc.EnableWorkflowOutbox()
	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "未绑定分类", Description: "分类没有流程 Key", Priority: "medium",
		RequesterID: requester.ID, CategoryID: &category.ID,
	}, tenant.ID)
	require.NoError(t, err)

	require.Empty(t, workflowStartPayloadKey(t, client, ctx, tenant.ID, created.ID))
}

// TestTicketCategoryWorkflowKeyMustResolve 分类写入必须校验流程定义：
// 未知 Key、他租户 Key 都要拒绝且不落库；空串才是解除绑定。
func TestTicketCategoryWorkflowKeyMustResolve(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenantA, _ := catFlowTenantAndRequester(t, client, ctx, "wf-a")
	tenantB, _ := catFlowTenantAndRequester(t, client, ctx, "wf-b")
	deployFlow(t, client, ctx, tenantB.ID, "tenant_b_flow")

	svc := NewTicketCategoryService(client)
	key := "ghost_flow"
	_, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name: "ghost", Code: "ghost", TenantID: tenantA.ID, IsActive: true, WorkflowDefinitionKey: &key,
	})
	var invalid *common.AppError
	require.True(t, errors.As(err, &invalid), "未知流程 Key 必须返回参数错误，实际: %v", err)
	require.Equal(t, 400, invalid.HTTPStatus)
	require.Equal(t, "工作流不存在、未激活或不属于当前租户", invalid.Message)

	existing, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name: "ok", Code: "ok", TenantID: tenantA.ID, IsActive: true,
	})
	require.NoError(t, err)

	// 跨租户流程 Key 同样拒绝，且原值不被覆盖
	foreign := "tenant_b_flow"
	_, err = svc.UpdateCategory(ctx, existing.ID, &UpdateCategoryRequest{WorkflowDefinitionKey: &foreign}, tenantA.ID)
	require.True(t, errors.As(err, &invalid), "跨租户流程 Key 必须返回参数错误，实际: %v", err)
	row, err := client.TicketCategory.Get(ctx, existing.ID)
	require.NoError(t, err)
	require.Empty(t, row.WorkflowDefinitionKey, "被拒绝的写入不得留下脏数据")

	// 本租户已部署流程可以绑定，再用空串解除
	deployFlow(t, client, ctx, tenantA.ID, "tenant_a_flow")
	tenantKey := "tenant_a_flow"
	updated, err := svc.UpdateCategory(ctx, existing.ID, &UpdateCategoryRequest{WorkflowDefinitionKey: &tenantKey}, tenantA.ID)
	require.NoError(t, err)
	require.Equal(t, "tenant_a_flow", updated.WorkflowDefinitionKey)

	cleared, err := svc.UpdateCategory(ctx, existing.ID, &UpdateCategoryRequest{WorkflowDefinitionKey: strPtr("")}, tenantA.ID)
	require.NoError(t, err)
	require.Empty(t, cleared.WorkflowDefinitionKey)
}
