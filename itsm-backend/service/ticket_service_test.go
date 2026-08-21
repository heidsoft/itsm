package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketcomment"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketService_CreateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试工单分类
	testCategory, err := client.TicketCategory.Create().
		SetName("incident").
		SetCode("incident").
		SetDescription("事件类工单").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_ = testCategory

	tests := []struct {
		name          string
		request       *dto.CreateTicketRequest
		tenantID      int
		expectedError bool
	}{
		{
			name: "成功创建工单",
			request: &dto.CreateTicketRequest{
				Title:       "测试工单",
				Description: "这是一个测试工单的详细描述",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
				FormFields: map[string]interface{}{
					"category": "hardware",
					"urgency":  "normal",
				},
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "标题为空",
			request: &dto.CreateTicketRequest{
				Title:       "",
				Description: "描述",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "描述为空（V2 不做必填校验，会创建成功）",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "无效的优先级",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "描述",
				Priority:    "invalid",
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保 assignee 存在（如果测试用例指定了）
			if tt.request.AssigneeID > 0 {
				// 先尝试查询，如果不存在则创建
				exists, _ := client.User.Query().Where(user.ID(tt.request.AssigneeID)).Exist(ctx)
				if !exists {
					u, err := client.User.Create().
						SetUsername(fmt.Sprintf("assignee_%d", tt.request.AssigneeID)).
						SetEmail(fmt.Sprintf("assignee_%d@example.com", tt.request.AssigneeID)).
						SetName("Assignee").
						SetPasswordHash("hash").
						SetRole("agent").
						SetActive(true).
						SetTenantID(tt.tenantID).
						Save(ctx)
					if err == nil {
						tt.request.AssigneeID = u.ID
					}
				}
			}

			response, err := ticketService.CreateTicket(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.request.Title, response.Title)
				assert.Equal(t, tt.request.Description, response.Description)
				assert.Equal(t, tt.request.Priority, string(response.Priority))
				assert.Equal(t, "new", string(response.Status)) // V2 默认状态为 new
				assert.NotEmpty(t, response.TicketNumber)
				assert.Equal(t, tt.tenantID, response.TenantID)
			}
		})
	}
}

func TestTicketService_CreateTicketTypeMapping(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_ticket_type?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	ctx := context.Background()

	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test-ticket-type").
		SetDomain("ticket-type.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("typeuser").
		SetEmail("type@example.com").
		SetName("Type User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	serviceRequest, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "服务请求工单",
		Description: "申请开通服务请求类型",
		Priority:    "medium",
		Type:        "service_request",
		RequesterID: testUser.ID,
	}, testTenant.ID)
	require.NoError(t, err)
	require.NotNil(t, serviceRequest)
	assert.Equal(t, "service_request", string(serviceRequest.Type))

	defaulted, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "默认类型工单",
		Description: "未传类型时不应写入空字符串",
		Priority:    "medium",
		RequesterID: testUser.ID,
	}, testTenant.ID)
	require.NoError(t, err)
	require.NotNil(t, defaulted)
	assert.Equal(t, "incident", string(defaulted.Type))
}

func TestTicketService_ConfiguredTypePersistenceAndIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:configured_ticket_type?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	typeService := NewTicketTypeService(client, logger)
	tenant, err := client.Tenant.Create().SetName("Tenant A").SetCode("tta").SetDomain("tta.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	other, err := client.Tenant.Create().SetName("Tenant B").SetCode("ttb").SetDomain("ttb.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().SetUsername("requester").SetEmail("requester@tta.test").SetName("Requester").SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	configured, err := typeService.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "pacs", Name: "PACS 故障", DefaultPriority: "high", CustomFields: []dto.CustomFieldDefinition{{ID: "node", Name: "node", Label: "PACS 节点", Type: dto.CustomFieldTypeText, Required: true, Order: 0}}}, tenant.ID, requester.ID)
	require.NoError(t, err)

	created, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "PACS unavailable", Description: "PACS node is unavailable", Priority: "high", RequesterID: requester.ID, TicketTypeID: &configured.ID, FormFields: map[string]interface{}{"node": "pacs-01"}}, tenant.ID)
	require.NoError(t, err)
	stored, err := client.Ticket.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, configured.ID, stored.TicketTypeID)
	assert.Equal(t, "pacs", stored.TicketTypeCodeSnapshot)
	assert.Equal(t, "PACS 故障", stored.TicketTypeNameSnapshot)
	assert.Equal(t, "pacs-01", stored.FormFields["node"])

	_, err = ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Foreign", Description: "foreign tenant type", Priority: "high", RequesterID: requester.ID, TicketTypeID: &configured.ID, FormFields: map[string]interface{}{"node": "x"}}, other.ID)
	assert.Error(t, err)
	_, err = ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Missing", Description: "required field missing", Priority: "high", RequesterID: requester.ID, TicketTypeID: &configured.ID, FormFields: map[string]interface{}{}}, tenant.ID)
	assert.Error(t, err)
	_, err = ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Unknown", Description: "unknown field", Priority: "high", RequesterID: requester.ID, TicketTypeID: &configured.ID, FormFields: map[string]interface{}{"node": "x", "injected": true}}, tenant.ID)
	assert.Error(t, err)
	_, err = typeService.SetStatus(ctx, configured.ID, tenant.ID, requester.ID, dto.TicketTypeStatusInactive)
	require.NoError(t, err)
	_, err = ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Disabled", Description: "disabled type", Priority: "high", RequesterID: requester.ID, TicketTypeID: &configured.ID, FormFields: map[string]interface{}{"node": "x"}}, tenant.ID)
	assert.Error(t, err)

	agent, err := client.User.Create().SetUsername("agent").SetEmail("agent@tta.test").SetName("Agent").SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	sla, err := client.SLADefinition.Create().SetName("Bound SLA").SetResponseTime(15).SetResolutionTime(60).SetIsActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	rule, err := client.TicketAssignmentRule.Create().SetName("Bound rule").SetIsActive(true).SetTenantID(tenant.ID).SetActions(map[string]interface{}{"type": "user", "value": float64(agent.ID)}).Save(ctx)
	require.NoError(t, err)
	bound, err := typeService.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "bound", Name: "Bound", DefaultPriority: "high", SLAEnabled: true, DefaultSLAID: &sla.ID, AutoAssignEnabled: true, AssignmentRuleID: &rule.ID}, tenant.ID, requester.ID)
	require.NoError(t, err)
	ruleService := NewTicketAssignmentRuleService(client, logger)
	ticketService.slaSvc = NewTicketSLAService(client, logger)
	ticketService.assignmentSmartService = NewTicketAssignmentSmartService(client, logger, NewTicketAssignmentService(client, logger), ruleService)
	boundTicket, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Bound execution", Description: "execute bound services", Priority: "high", RequesterID: requester.ID, TicketTypeID: &bound.ID}, tenant.ID)
	require.NoError(t, err)
	boundStored, err := client.Ticket.Get(ctx, boundTicket.ID)
	require.NoError(t, err)
	assert.Equal(t, sla.ID, boundStored.SLADefinitionID)
	assert.Equal(t, agent.ID, boundStored.AssigneeID)
}

func TestTicketService_CreateTicketPersistsAssociations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_create_associations?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "create-associations")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-requester")
	assignee := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-assignee")
	category, err := client.TicketCategory.Create().
		SetName("Hardware").SetCode("create-hardware").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	template, err := client.TicketTemplate.Create().
		SetName("Hardware template").SetCategory("hardware").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	tag, err := client.TicketTag.Create().
		SetName("urgent-device").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	parent, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Parent ticket", Description: "parent", Priority: "medium", RequesterID: requester.ID,
	}, tenant.ID)
	require.NoError(t, err)

	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:          "Child ticket",
		Description:    "child",
		Priority:       "high",
		RequesterID:    requester.ID,
		AssigneeID:     assignee.ID,
		CategoryID:     &category.ID,
		TemplateID:     &template.ID,
		ParentTicketID: &parent.ID,
		TagIDs:         []int{tag.ID, tag.ID},
	}, tenant.ID)
	require.NoError(t, err)

	entity, err := client.Ticket.Query().Where(entTicket.IDEQ(created.ID)).WithTags().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, entity.ParentTicketID)
	assert.Equal(t, template.ID, entity.TemplateID)
	assert.Equal(t, category.ID, entity.CategoryID)
	assert.Equal(t, assignee.ID, entity.AssigneeID)
	require.Len(t, entity.Edges.Tags, 1)
	assert.Equal(t, tag.ID, entity.Edges.Tags[0].ID)
}

func TestTicketService_CreateTicketRejectsCrossTenantReferences(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_create_cross_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenantA := createTicketAssociationTenant(t, ctx, client, "create-tenant-a")
	tenantB := createTicketAssociationTenant(t, ctx, client, "create-tenant-b")
	userA := createTicketAssociationUser(t, ctx, client, tenantA.ID, "create-user-a")
	userB := createTicketAssociationUser(t, ctx, client, tenantB.ID, "create-user-b")
	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	foreignParent, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Foreign parent", Description: "foreign", Priority: "medium", RequesterID: userB.ID,
	}, tenantB.ID)
	require.NoError(t, err)

	_, err = service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Invalid requester", Description: "invalid", Priority: "medium", RequesterID: userB.ID,
	}, tenantA.ID)
	require.ErrorContains(t, err, "申请人不存在")

	_, err = service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Invalid parent", Description: "invalid", Priority: "medium", RequesterID: userA.ID, ParentTicketID: &foreignParent.ID,
	}, tenantA.ID)
	require.ErrorContains(t, err, "父工单不存在")
}

func TestTicketService_GetTicketStatsCountsNewAsPending(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_ticket_stats?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	ctx := context.Background()

	testTenant, err := client.Tenant.Create().
		SetName("Stats Tenant").
		SetCode("test-ticket-stats").
		SetDomain("ticket-stats.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("statsuser").
		SetEmail("stats@example.com").
		SetName("Stats User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
			Title:       fmt.Sprintf("新工单 %d", i),
			Description: "新建状态应计入待处理统计",
			Priority:    "medium",
			Type:        "incident",
			RequesterID: testUser.ID,
		}, testTenant.ID)
		require.NoError(t, err)
	}

	stats, err := ticketService.GetTicketStats(ctx, testTenant.ID)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 3, stats.Pending)
	assert.Equal(t, 3, stats.Open)
}

func TestTicketService_GetTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建多个测试工单
	tickets := make([]*ent.Ticket, 3)
	for i := 0; i < 3; i++ {
		ticket, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("测试工单 %d", i+1)).
			SetDescription(fmt.Sprintf("测试工单描述 %d", i+1)).
			SetPriority("medium").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TICKET-%d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
		tickets[i] = ticket
	}

	tests := []struct {
		name          string
		request       *dto.ListTicketsRequest
		tenantID      int
		expectedCount int
		expectedError bool
	}{
		{
			name: "获取所有工单",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "分页查询",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 2,
			},
			tenantID:      testTenant.ID,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "按状态筛选",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
				Status:   "open",
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "按优先级筛选",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
				Priority: "medium",
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 阻断8：以 testUser（end_user）身份查询，
			// DataScope=OwnedOrAssigned 应仅返回本人创建/分配的工单。
			// 测试数据中 3 张工单均由 testUser 创建，故总数仍为 3。
			response, err := ticketService.ListTickets(ctx, tt.request, tt.tenantID, testUser.ID, "end_user")

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Tickets, tt.expectedCount)
				assert.Equal(t, 3, response.Total) // 总数始终为3
				assert.Equal(t, tt.request.Page, response.Page)
				assert.Equal(t, tt.request.PageSize, response.PageSize)
			}
		})
	}
}

// TestTicketService_ListTickets_DataScope 阻断8 回归测试：
// 验证行级数据权限。end_user 只能看到自己创建或分配给自己的工单，
// admin/manager 可见全租户工单。这是安全关键路径，防止越权读取 HR/薪酬/安全工单。
func TestTicketService_ListTickets_DataScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建租户
	testTenant, err := client.Tenant.Create().
		SetName("DataScope Tenant").
		SetCode("dscope").
		SetDomain("dscope.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建两个 end_user 和一个 admin
	alice, err := client.User.Create().
		SetUsername("alice").
		SetEmail("alice@example.com").
		SetName("Alice").
		SetPasswordHash("hashed").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	bob, err := client.User.Create().
		SetUsername("bob").
		SetEmail("bob@example.com").
		SetName("Bob").
		SetPasswordHash("hashed").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername("admin").
		SetEmail("admin@example.com").
		SetName("Admin").
		SetPasswordHash("hashed").
		SetRole("admin").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// Alice 创建 2 张工单，Bob 创建 1 张工单（含敏感信息）
	_, err = client.Ticket.Create().
		SetTicketNumber("DSCOPE-1").
		SetTitle("Alice 的工单 1").
		SetDescription("desc").
		SetPriority("medium").
		SetStatus("open").
		SetType("incident").
		SetRequesterID(alice.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTicketNumber("DSCOPE-2").
		SetTitle("Alice 的工单 2").
		SetDescription("desc").
		SetPriority("medium").
		SetStatus("open").
		SetType("incident").
		SetRequesterID(alice.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	bobTicket, err := client.Ticket.Create().
		SetTicketNumber("DSCOPE-3").
		SetTitle("Bob 的薪酬工单（敏感）").
		SetDescription("salary info").
		SetPriority("high").
		SetStatus("open").
		SetType("incident").
		SetRequesterID(bob.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 将一张 Bob 的工单分配给 Alice，验证"分配给自己"也可见
	_, err = client.Ticket.Create().
		SetTicketNumber("DSCOPE-4").
		SetTitle("Bob 创建但分配给 Alice").
		SetDescription("assigned to alice").
		SetPriority("medium").
		SetStatus("open").
		SetType("incident").
		SetRequesterID(bob.ID).
		SetAssigneeID(alice.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	req := &dto.ListTicketsRequest{Page: 1, PageSize: 100}

	// 场景1：Alice（end_user）只能看到自己创建的 2 张 + 分配给自己的 1 张 = 3 张，
	// 看不到 Bob 的 DSCOPE-3（薪酬敏感工单）。
	aliceResp, err := ticketService.ListTickets(ctx, req, testTenant.ID, alice.ID, "end_user")
	require.NoError(t, err)
	assert.Equal(t, 3, aliceResp.Total, "Alice 应只看到自己创建或分配给自己的工单")
	for _, tk := range aliceResp.Tickets {
		assert.NotEqual(t, "DSCOPE-3", tk.TicketNumber,
			"Alice 不应看到 Bob 的薪酬敏感工单")
	}

	// 场景2：Bob（end_user）只能看到自己创建的 2 张（DSCOPE-3 + DSCOPE-4）。
	bobResp, err := ticketService.ListTickets(ctx, req, testTenant.ID, bob.ID, "end_user")
	require.NoError(t, err)
	assert.Equal(t, 2, bobResp.Total, "Bob 应只看到自己创建的工单")

	// 场景3：Admin 可见全租户全部 4 张工单。
	adminResp, err := ticketService.ListTickets(ctx, req, testTenant.ID, admin.ID, "admin")
	require.NoError(t, err)
	assert.Equal(t, 4, adminResp.Total, "Admin 应看到全租户所有工单")

	// 场景4：空角色 + 未提供 userID，fail closed 返回空集。
	emptyResp, err := ticketService.ListTickets(ctx, req, testTenant.ID, 0, "")
	require.NoError(t, err)
	assert.Equal(t, 0, emptyResp.Total, "未提供身份时应 fail closed 返回空集")

	// 场景5：Alice 不能通过 RequesterID=bob 过滤绕过 DataScope 查看 Bob 独有的工单。
	// 即使 Alice 传入 requesterID=bob.ID，DataScope 仍会强制收窄到"本人可见"。
	// DSCOPE-4 虽由 Bob 创建但分配给 Alice，Alice 有权看到（assignee 路径）；
	// 但 DSCOPE-3（Bob 独有）对 Alice 不可见。
	bobIDFilter := bob.ID
	aliceBypassResp, err := ticketService.ListTickets(ctx, &dto.ListTicketsRequest{
		Page:        1,
		PageSize:    100,
		RequesterID: &bobIDFilter,
	}, testTenant.ID, alice.ID, "end_user")
	require.NoError(t, err)
	// Alice 只能看到 DSCOPE-4（分配给自己），看不到 DSCOPE-3（Bob 独有）。
	assert.Equal(t, 1, aliceBypassResp.Total,
		"Alice 以 requesterID=bob 过滤时只应看到分配给自己的 DSCOPE-4")
	for _, tk := range aliceBypassResp.Tickets {
		assert.NotEqual(t, "DSCOPE-3", tk.TicketNumber,
			"Alice 不应通过 RequesterID 过滤绕过 DataScope 看到 Bob 的薪酬敏感工单")
	}
	_ = bobTicket // keep linter happy
}

func TestTicketService_GetTicketByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("测试工单").
		SetDescription("测试工单描述").
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功获取工单",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "工单不存在",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name:          "租户不匹配",
			ticketID:      testTicket.ID,
			tenantID:      99999,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := ticketService.GetTicket(ctx, tt.ticketID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, ticket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ticket)
				assert.Equal(t, testTicket.ID, ticket.ID)
				assert.Equal(t, "测试工单", ticket.Title)
				assert.Equal(t, "high", string(ticket.Priority))
			}
		})
	}
}

func TestTicketService_UpdateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		request       *dto.UpdateTicketRequest
		tenantID      int
		expectedError bool
	}{
		{
			name:     "成功更新工单",
			ticketID: testTicket.ID,
			request: &dto.UpdateTicketRequest{
				Title:       "更新后的标题",
				Description: "更新后的描述",
				Priority:    "high",
				Status:      "in_progress",
				UserID:      testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:     "部分更新",
			ticketID: testTicket.ID,
			request: &dto.UpdateTicketRequest{
				Priority: "critical",
				UserID:   testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:     "工单不存在",
			ticketID: 99999,
			request: &dto.UpdateTicketRequest{
				Title:  "新标题",
				UserID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedTicket, err := ticketService.UpdateTicket(ctx, tt.ticketID, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)

				if tt.request.Title != "" {
					assert.Equal(t, tt.request.Title, updatedTicket.Title)
				}
				if tt.request.Description != "" {
					assert.Equal(t, tt.request.Description, updatedTicket.Description)
				}
				if tt.request.Priority != "" {
					assert.Equal(t, tt.request.Priority, string(updatedTicket.Priority))
				}
				if tt.request.Status != "" {
					assert.Equal(t, tt.request.Status, string(updatedTicket.Status))
				}
			}
		})
	}
}

func TestTicketService_UpdateTicketPersistsTypeCategoryAndTags(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_update_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "update-contract")
	otherTenant := createTicketAssociationTenant(t, ctx, client, "update-contract-other")
	user := createTicketAssociationUser(t, ctx, client, tenant.ID, "update-contract-user")
	category, err := client.TicketCategory.Create().SetName("Software").SetCode("update-software").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	foreignCategory, err := client.TicketCategory.Create().SetName("Foreign").SetCode("update-foreign").SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)
	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Update contract", Description: "before", Priority: "medium", RequesterID: user.ID,
	}, tenant.ID)
	require.NoError(t, err)

	updated, err := service.UpdateTicket(ctx, created.ID, &dto.UpdateTicketRequest{
		Type: "problem", CategoryID: &category.ID, Tags: []string{"backend", "backend", "customer"}, Version: created.Version,
	}, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "problem", string(updated.Type))
	entity, err := client.Ticket.Query().Where(entTicket.IDEQ(created.ID)).WithTags().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, category.ID, entity.CategoryID)
	require.Len(t, entity.Edges.Tags, 2)

	zero := 0
	cleared, err := service.UpdateTicket(ctx, created.ID, &dto.UpdateTicketRequest{
		CategoryID: &zero, Tags: []string{}, Version: updated.Version,
	}, tenant.ID)
	require.NoError(t, err)
	assert.Nil(t, cleared.CategoryID)
	entity, err = client.Ticket.Query().Where(entTicket.IDEQ(created.ID)).WithTags().Only(ctx)
	require.NoError(t, err)
	assert.Empty(t, entity.Edges.Tags)

	_, err = service.UpdateTicket(ctx, created.ID, &dto.UpdateTicketRequest{
		CategoryID: &foreignCategory.ID, Version: cleared.Version,
	}, tenant.ID)
	require.ErrorContains(t, err, "工单分类不存在")
}

func TestTicketService_DeleteTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("待删除工单").
		SetDescription("待删除工单描述").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功删除工单",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "工单不存在",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ticketService.DeleteTicket(ctx, tt.ticketID, tt.tenantID, 0, "super_admin")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 对业务查询不可见，但底层记录保留用于审计。
				_, err := ticketService.GetTicket(ctx, tt.ticketID, tt.tenantID)
				assert.Error(t, err)
				raw, err := client.Ticket.Get(ctx, tt.ticketID)
				require.NoError(t, err)
				assert.NotNil(t, raw.DeletedAt)
			}
		})
	}
}

// TestTicketService_DeleteTicket_RowScope 验证 M-7 行级删除归属：
// 非全量角色只能删除自己创建或分配给自己的工单，越权返回 ForbiddenError(403)。
func TestTicketService_DeleteTicket_RowScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Scope Tenant").SetCode("scope").SetDomain("scope.com").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	owner, err := client.User.Create().
		SetUsername("owner").SetEmail("owner@x.com").SetName("Owner").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	assignee, err := client.User.Create().
		SetUsername("assignee").SetEmail("assignee@x.com").SetName("Assignee").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername("admin").SetEmail("admin@x.com").SetName("Admin").
		SetPasswordHash("h").SetRole("admin").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 工单属于 owner，并分配给 assignee
	tk, err := client.Ticket.Create().
		SetTitle("Scope Ticket").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("SCOPE-001").SetRequesterID(owner.ID).SetAssigneeID(assignee.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("无关用户删除->拒绝(403)", func(t *testing.T) {
		stranger, err := client.User.Create().
			SetUsername("stranger").SetEmail("stranger@x.com").SetName("Stranger").
			SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		err = ticketService.DeleteTicket(ctx, tk.ID, tenant.ID, stranger.ID, "end_user")
		require.Error(t, err)
		var appErr *common.AppError
		require.True(t, errors.As(err, &appErr), "应返回 *common.AppError")
		assert.Equal(t, common.ErrCodeForbidden, appErr.Code)
		// 工单仍可见（未被删除）
		_, gerr := ticketService.GetTicket(ctx, tk.ID, tenant.ID)
		assert.NoError(t, gerr)
	})

	t.Run("创建人删除->放行", func(t *testing.T) {
		err := ticketService.DeleteTicket(ctx, tk.ID, tenant.ID, owner.ID, "end_user")
		assert.NoError(t, err)
		_, gerr := ticketService.GetTicket(ctx, tk.ID, tenant.ID)
		assert.Error(t, gerr, "owner 删除后应对业务查询不可见")
	})

	// 重建一个分配给 assignee 的工单，验证处理人维度
	tk2, err := client.Ticket.Create().
		SetTitle("Scope Ticket 2").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("SCOPE-002").SetRequesterID(owner.ID).SetAssigneeID(assignee.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("处理人删除->放行", func(t *testing.T) {
		err := ticketService.DeleteTicket(ctx, tk2.ID, tenant.ID, assignee.ID, "end_user")
		assert.NoError(t, err)
	})

	t.Run("管理员删除他人工单->放行(全量角色)", func(t *testing.T) {
		tk3, err := client.Ticket.Create().
			SetTitle("Scope Ticket 3").SetDescription("d").SetPriority("low").SetStatus("open").
			SetTicketNumber("SCOPE-003").SetRequesterID(owner.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		err = ticketService.DeleteTicket(ctx, tk3.ID, tenant.ID, admin.ID, "admin")
		assert.NoError(t, err)
	})
}

// TestTicketService_BatchDeleteTickets_RowScope 验证批量删除的行级归属：
// 非全量角色若集合中存在任一非本人工单，整体拒绝（fail closed）。
func TestTicketService_BatchDeleteTickets_RowScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("BatchScope Tenant").SetCode("bscope").SetDomain("bscope.com").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	owner, err := client.User.Create().
		SetUsername("bowner").SetEmail("bowner@x.com").SetName("BOwner").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	stranger, err := client.User.Create().
		SetUsername("bstranger").SetEmail("bstranger@x.com").SetName("BStranger").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	ownA, err := client.Ticket.Create().
		SetTitle("BA").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("BSCOPE-A").SetRequesterID(owner.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	ownB, err := client.Ticket.Create().
		SetTitle("BB").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("BSCOPE-B").SetRequesterID(owner.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	other, err := client.Ticket.Create().
		SetTitle("BC").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("BSCOPE-C").SetRequesterID(stranger.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("集合含他人工单->整体拒绝", func(t *testing.T) {
		err := ticketService.BatchDeleteTickets(ctx, []int{ownA.ID, other.ID}, tenant.ID, owner.ID, "end_user")
		require.Error(t, err)
		var appErr *common.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, common.ErrCodeForbidden, appErr.Code)
		// 全部保留
		_, e1 := ticketService.GetTicket(ctx, ownA.ID, tenant.ID)
		_, e2 := ticketService.GetTicket(ctx, other.ID, tenant.ID)
		assert.NoError(t, e1)
		assert.NoError(t, e2)
	})

	t.Run("集合全为本人工单->放行", func(t *testing.T) {
		err := ticketService.BatchDeleteTickets(ctx, []int{ownA.ID, ownB.ID}, tenant.ID, owner.ID, "end_user")
		assert.NoError(t, err)
	})
}

func TestTicketService_DeleteTicket_CascadeTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// Create tenant 1
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("tenant1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// Create tenant 2
	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("tenant2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// Create user for tenant 1
	user1, err := client.User.Create().
		SetUsername("user1").
		SetEmail("user1@tenant1.com").
		SetName("User 1").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create ticket for tenant 1 with a comment
	ticket1, err := client.Ticket.Create().
		SetTitle("Tenant 1 Ticket").
		SetDescription("Test ticket").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-T1-001").
		SetRequesterID(user1.ID).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create a comment for the ticket
	_, err = client.TicketComment.Create().
		SetTicketID(ticket1.ID).
		SetUserID(user1.ID).
		SetContent("Test comment").
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create an attachment for the ticket
	_, err = client.TicketAttachment.Create().
		SetTicketID(ticket1.ID).
		SetFileName("test.txt").
		SetFilePath("/uploads/test.txt").
		SetFileURL("/uploads/test.txt").
		SetFileSize(1024).
		SetFileType("text/plain").
		SetMimeType("text/plain").
		SetUploadedBy(user1.ID).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Tenant 2 tries to delete tenant 1's ticket.
	err = ticketService.DeleteTicket(ctx, ticket1.ID, tenant2.ID, 0, "end_user")
	assert.Error(t, err)

	// Verify ticket still exists (未被删除，跨租户隔离仍然有效)
	_, err = client.Ticket.Get(ctx, ticket1.ID)
	assert.NoError(t, err)

	// Verify cascade comments were NOT deleted
	comments, err := client.TicketComment.Query().Where(ticketcomment.TicketIDEQ(ticket1.ID)).Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, comments, "comment should still exist after failed cross-tenant delete attempt")
}

// scopeTicketIDs 从列表响应中提取工单 ID 集合，便于断言行级可见范围。
func scopeTicketIDs(tickets []*dto.TicketResponse) []int {
	ids := make([]int, 0, len(tickets))
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}
	return ids
}

// TestTicketService_ListTickets_RowScope 验证 H-14 行级列表过滤：
// 非全量角色仅见本人创建/分配工单；全量角色见全租户。
func TestTicketService_ListTickets_RowScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("ListScope").SetCode("lscope").SetDomain("lscope.com").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	alice, err := client.User.Create().
		SetUsername("lalice").SetEmail("lalice@x.com").SetName("Alice").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	bob, err := client.User.Create().
		SetUsername("lbob").SetEmail("lbob@x.com").SetName("Bob").
		SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	admin, err := client.User.Create().
		SetUsername("ladmin").SetEmail("ladmin@x.com").SetName("Admin").
		SetPasswordHash("h").SetRole("admin").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tA, err := client.Ticket.Create().
		SetTitle("A").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("LSCOPE-A").SetRequesterID(alice.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	tB, err := client.Ticket.Create().
		SetTitle("B").SetDescription("d").SetPriority("low").SetStatus("open").
		SetTicketNumber("LSCOPE-B").SetRequesterID(bob.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("alice 仅见自己的工单", func(t *testing.T) {
		resp, err := ticketService.ListTickets(ctx, &dto.ListTicketsRequest{Page: 1, PageSize: 50}, tenant.ID, alice.ID, "end_user")
		require.NoError(t, err)
		ids := scopeTicketIDs(resp.Tickets)
		assert.Contains(t, ids, tA.ID)
		assert.NotContains(t, ids, tB.ID)
	})

	t.Run("admin 见全租户工单", func(t *testing.T) {
		resp, err := ticketService.ListTickets(ctx, &dto.ListTicketsRequest{Page: 1, PageSize: 50}, tenant.ID, admin.ID, "admin")
		require.NoError(t, err)
		ids := scopeTicketIDs(resp.Tickets)
		assert.Contains(t, ids, tA.ID)
		assert.Contains(t, ids, tB.ID)
	})
}

func TestTicketService_SearchTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试工单
	_, err = client.Ticket.Create().
		SetTitle("网络连接问题").
		SetDescription("用户无法连接到网络").
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTitle("打印机故障").
		SetDescription("打印机无法正常工作").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TICKET-002").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		searchTerm    string
		tenantID      int
		expectedCount int
		expectedError bool
	}{
		{
			name:          "搜索网络相关工单",
			searchTerm:    "网络",
			tenantID:      testTenant.ID,
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索打印机相关工单",
			searchTerm:    "打印机",
			tenantID:      testTenant.ID,
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索不存在的内容",
			searchTerm:    "不存在的内容",
			tenantID:      testTenant.ID,
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "空搜索词",
			searchTerm:    "",
			tenantID:      testTenant.ID,
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, err := ticketService.SearchTickets(ctx, tt.searchTerm, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, tickets)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tickets)
				assert.Len(t, tickets, tt.expectedCount)
			}
		})
	}
}

func TestTicketService_GetMSPCustomerReports_AllocationAware(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// Setup: Create MSP tenant and multiple customer tenants
	mspTenant, _ := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)

	allocatedTenant, _ := client.Tenant.Create().
		SetName("AllocatedCustomer").
		SetCode("alloc_cust").
		SetType("customer").
		Save(ctx)

	unallocatedTenant, _ := client.Tenant.Create().
		SetName("UnallocatedCustomer").
		SetCode("unalloc_cust").
		SetType("customer").
		Save(ctx)

	// Create MSP user
	mspUser, _ := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)

	// Create allocation ONLY to allocatedTenant
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(allocatedTenant.ID).
		SetRole("provider_agent").
		Save(ctx)

	// Test: V2 GetMSPCustomerReports 按 mspTenantID 维度聚合统计
	dateFrom, _ := time.Parse("2006-01-02", "2024-01-01")
	dateTo, _ := time.Parse("2006-01-02", "2024-12-31")
	reports, err := ticketService.GetMSPCustomerReports(ctx, mspTenant.ID, dateFrom, dateTo)
	assert.NoError(t, err)
	assert.NotNil(t, reports)
	// V2 返回的 reports 至少包含 status_summary 等字段
	if len(reports) > 0 {
		assert.Contains(t, reports[0], "status_summary")
		assert.Contains(t, reports[0], "total_tickets")
	}

	// Test: 验证未分配租户场景下 V2 仅返回 msp 租户维度统计，不会报错
	reports, err = ticketService.GetMSPCustomerReports(ctx, unallocatedTenant.ID, dateFrom, dateTo)
	assert.NoError(t, err)
	assert.NotNil(t, reports)
}

// 基准测试
func BenchmarkTicketService_CreateTicket(b *testing.B) {
	client := enttest.Open(b, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, _ := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(b, err)

	request := &dto.CreateTicketRequest{
		Title:       "基准测试工单",
		Description: "这是一个基准测试工单",
		Priority:    "medium",
		Category:    "incident",
		RequesterID: testUser.ID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ticketService.CreateTicket(ctx, request, testTenant.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
