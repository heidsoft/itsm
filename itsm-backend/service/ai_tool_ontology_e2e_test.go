package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	entticket "itsm-backend/ent/ticket"
	ticketrepo "itsm-backend/repository/ticket"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupOntologyE2E 搭建 AI 工具层的端到端夹具：一个租户 + 两个 CI（一库一服务器）
// + 完整的 ToolRegistry（ticket/cmdb 服务均已装配），模拟生产装配状态。
// 返回 client, registry, ctx, tenantID, userID, hisDBCIID
func setupOntologyE2E(t *testing.T) (*ent.Client, *ToolRegistry, context.Context, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("hospital").SetCode("hospital").SetDomain("hospital.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	ciType, err := client.CIType.Create().SetName("Database").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// HIS-DB-01：用户报障的目标资产
	hisDB, err := client.ConfigurationItem.Create().
		SetName("HIS-DB-01").
		SetCiTypeID(ciType.ID).SetCiType("database").
		SetStatus("active").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 另一台服务器，用于验证 ci_type 过滤确实生效
	_, err = client.ConfigurationItem.Create().
		SetName("APP-SERVER-02").
		SetCiTypeID(ciType.ID).SetCiType("server").
		SetStatus("active").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("nurse01").SetName("nurse01").
		SetEmail("nurse01@hospital.test").
		SetPasswordHash("x").SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	cmdbSvc := NewConfigurationItemService(client, logger, NewCIHistoryService(client, logger), NewCITagService(client, logger))
	ticketSvc := NewTicketService(&TicketServiceConfig{
		Repository: ticketrepo.NewEntRepository(client, logger),
		Client:     client,
		Logger:     logger,
	})

	reg := NewToolRegistry(nil, nil, cmdbSvc, client)
	reg.SetTicketService(ticketSvc)

	return client, reg, ctx, tenant.ID, user.ID, hisDB.ID
}

// TestE2E_AILocatesCIAndCreatesLinkedTicket 端到端验证 AI 本体链路：
// 用户口语报障 → list_cis 定位 CI → create_ticket 带 ci_id → 工单落库且外键生效。
func TestE2E_AILocatesCIAndCreatesLinkedTicket(t *testing.T) {
	client, reg, ctx, tenantID, userID, _ := setupOntologyE2E(t)
	defer client.Close()

	// 步骤 1：AI 用 list_cis 按名称模糊搜索定位 CI
	found, err := reg.Execute(ctx, tenantID, "list_cis", map[string]interface{}{
		"search": "HIS-DB",
		"limit":  float64(10),
	})
	require.NoError(t, err)
	items, ok := found.([]*dto.CIResponse)
	require.True(t, ok, "list_cis 应返回 CI 列表")
	require.Len(t, items, 1, "search=HIS-DB 应只命中 HIS-DB-01")
	ciID := items[0].ID
	require.Equal(t, "HIS-DB-01", items[0].Name)

	// 步骤 2：验证 ci_type 过滤可区分资产类型（AI 面对同名歧义时的消歧能力）
	servers, err := reg.Execute(ctx, tenantID, "list_cis", map[string]interface{}{
		"ci_type": "server",
		"limit":   float64(10),
	})
	require.NoError(t, err)
	serverItems, ok := servers.([]*dto.CIResponse)
	require.True(t, ok)
	require.Len(t, serverItems, 1)
	require.Equal(t, "APP-SERVER-02", serverItems[0].Name)

	// 步骤 3：AI 带 ci_id 创建工单
	res, err := reg.Execute(ctx, tenantID, "create_ticket", map[string]interface{}{
		"title":        "HIS-DB-01 连接超时",
		"description":  "护士站反馈 HIS 数据库连接超时，无法开医嘱",
		"priority":     "high",
		"ci_id":        float64(ciID),
		"requester_id": float64(userID),
	})
	require.NoError(t, err)

	payload, ok := res.(map[string]interface{})
	require.True(t, ok, "带 ci_id 建单时应返回带 ciLinked 的复合结果")
	require.Equal(t, true, payload["ciLinked"], "工单应已绑定到 CI")
	require.Equal(t, ciID, payload["ciId"])

	ticketResp, ok := payload["ticket"].(*dto.TicketResponse)
	require.True(t, ok)
	require.NotZero(t, ticketResp.ID)

	// 步骤 4：校验真实外键落库（而非 form_fields JSON）
	exists, err := client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.HasTicketsWith(entticket.IDEQ(ticketResp.ID))).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, exists, "tickets.configuration_item_tickets 外键应已写入")

	// 步骤 5：反查影响面 —— get_ci_tickets 应能看到刚才的工单
	tickets, err := reg.Execute(ctx, tenantID, "get_ci_tickets", map[string]interface{}{
		"ci_id": float64(ciID),
	})
	require.NoError(t, err)
	list, ok := tickets.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, list, 1)
	require.Equal(t, ticketResp.ID, list[0]["id"])
	require.Equal(t, "HIS-DB-01 连接超时", list[0]["title"])
}

// TestE2E_CreateTicketWithoutCIIDStillSucceeds 未定位到 CI 时不应阻断建单。
func TestE2E_CreateTicketWithoutCIIDStillSucceeds(t *testing.T) {
	client, reg, ctx, tenantID, userID, _ := setupOntologyE2E(t)
	defer client.Close()

	res, err := reg.Execute(ctx, tenantID, "create_ticket", map[string]interface{}{
		"title":        "打印机卡纸",
		"description":  "三楼打印机卡纸",
		"priority":     "low",
		"requester_id": float64(userID),
	})
	require.NoError(t, err)

	// 无 ci_id 时返回的是纯工单响应，不应强制走复合结果
	ticketResp, ok := res.(*dto.TicketResponse)
	require.True(t, ok, "未传 ci_id 时应直接返回工单响应，而非带 ciLinked 的复合结果")
	require.NotZero(t, ticketResp.ID)
}

// TestE2E_CreateTicketWithInvalidCIIDDegradesGracefully
// ci_id 指向的 CI 不存在时，工单仍应创建成功（有效交付物），仅回传绑定失败提示。
func TestE2E_CreateTicketWithInvalidCIIDDegradesGracefully(t *testing.T) {
	client, reg, ctx, tenantID, userID, _ := setupOntologyE2E(t)
	defer client.Close()

	res, err := reg.Execute(ctx, tenantID, "create_ticket", map[string]interface{}{
		"title":        "未知设备故障",
		"priority":     "medium",
		"ci_id":        float64(999999), // 不存在的 CI
		"requester_id": float64(userID),
	})
	require.NoError(t, err, "CI 绑定失败不应导致整个建单失败")

	payload, ok := res.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, false, payload["ciLinked"])
	require.NotEmpty(t, payload["ciLinkWarn"], "应回传绑定失败原因供 AI 提示用户")
	require.True(t, strings.Contains(payload["ciLinkWarn"].(string), "配置项不存在或无权访问"))
}

// TestE2E_LinkExistingTicketToCI 补挂场景：先建单、后定位到设备。
func TestE2E_LinkExistingTicketToCI(t *testing.T) {
	client, reg, ctx, tenantID, userID, _ := setupOntologyE2E(t)
	defer client.Close()

	// 先建单（未带 ci_id）
	res, err := reg.Execute(ctx, tenantID, "create_ticket", map[string]interface{}{
		"title":        "系统很卡",
		"priority":     "medium",
		"requester_id": float64(userID),
	})
	require.NoError(t, err)
	ticketResp, ok := res.(*dto.TicketResponse)
	require.True(t, ok, "未带 ci_id 时应直接返回工单响应")

	// 后定位到 CI 并补挂
	found, err := reg.Execute(ctx, tenantID, "list_cis", map[string]interface{}{"search": "HIS-DB"})
	require.NoError(t, err)
	ciID := found.([]*dto.CIResponse)[0].ID

	linkRes, err := reg.Execute(ctx, tenantID, "link_ticket_ci", map[string]interface{}{
		"ticket_id": float64(ticketResp.ID),
		"ci_id":     float64(ciID),
	})
	require.NoError(t, err)
	require.Equal(t, true, linkRes.(map[string]interface{})["ciLinked"])

	exists, err := client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.HasTicketsWith(entticket.IDEQ(ticketResp.ID))).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, exists)
}
