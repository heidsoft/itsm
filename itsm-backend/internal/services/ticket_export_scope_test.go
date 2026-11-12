package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 工单导出可见性回归锁。
//
// 缺陷（2026-11-05 白盒审查 E-P0-1）：ExportTickets 只加了 TenantID 谓词，
// 既没有 DataScope 也没有 DeletedAtIsNil。导出是「第二个读取通道」，
// 页面列表把普通角色收窄到本人创建/分配给自己，但同一个人换个导出按钮
// 就能把全租户工单（含他人工单的描述正文）下载走，还能导出已软删的单。
//
// 三条用例分别锁住：行级范围、受理人可见、软删不可见。

func seedExportTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID, requesterID int, number, title string) *ent.Ticket {
	t.Helper()
	return client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle(title).
		SetDescription("正文-" + number).
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		SaveX(ctx)
}

func newExportFixture(t *testing.T) (*ent.Client, context.Context, *TicketService, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:ticket_export_scope_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("导出可见性").SetCode("export-scope").SetDomain("export-scope.example.com").SetStatus("active").SaveX(ctx)
	owner := client.User.Create().SetUsername("export-scope-owner").SetEmail("owner@export-scope.example.com").SetName("Owner").SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)
	viewer := client.User.Create().SetUsername("export-scope-viewer").SetEmail("viewer@export-scope.example.com").SetName("Viewer").SetPasswordHash("h").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)

	svc := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	return client, ctx, svc, tenant.ID, owner.ID, viewer.ID
}

func TestExportTickets_HidesTicketsOutsideDataScope(t *testing.T) {
	client, ctx, svc, tenantID, ownerID, viewerID := newExportFixture(t)

	seedExportTicket(t, ctx, client, tenantID, ownerID, "TKT-SCOPE-OTHER-001", "他人创建的工单")
	seedExportTicket(t, ctx, client, tenantID, viewerID, "TKT-SCOPE-OWN-001", "自己创建的工单")

	data, err := svc.ExportTickets(ctx, tenantID, map[string]interface{}{}, "csv", viewerID, "end_user")
	require.NoError(t, err)
	body := string(data)

	require.Contains(t, body, "TKT-SCOPE-OWN-001", "本人创建的工单必须可见")
	require.NotContains(t, body, "TKT-SCOPE-OTHER-001", "普通角色不得导出他人创建的工单")
	require.NotContains(t, body, "正文-TKT-SCOPE-OTHER-001", "他人工单正文更不得出现在导出里")
}

func TestExportTickets_IncludesAssignedTickets(t *testing.T) {
	client, ctx, svc, tenantID, ownerID, viewerID := newExportFixture(t)

	// 分配给 viewer 的工单：即使不是本人创建，也属于 DataScopeOwnedOrAssigned。
	client.Ticket.Create().
		SetTicketNumber("TKT-SCOPE-ASSIGNED-001").
		SetTitle("分配给我的工单").
		SetDescription("正文").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(ownerID).
		SetAssigneeID(viewerID).
		SetTenantID(tenantID).
		SaveX(ctx)

	data, err := svc.ExportTickets(ctx, tenantID, map[string]interface{}{}, "csv", viewerID, "end_user")
	require.NoError(t, err)
	require.Contains(t, string(data), "TKT-SCOPE-ASSIGNED-001")
}

func TestExportTickets_ExcludesSoftDeletedEvenForAdmin(t *testing.T) {
	client, ctx, svc, tenantID, ownerID, _ := newExportFixture(t)

	live := seedExportTicket(t, ctx, client, tenantID, ownerID, "TKT-SCOPE-LIVE-001", "在库工单")
	goneAt := time.Now().Add(-time.Hour)
	client.Ticket.Create().
		SetTicketNumber("TKT-SCOPE-DELETED-001").
		SetTitle("已软删工单").
		SetDescription("正文").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(ownerID).
		SetTenantID(tenantID).
		SetDeletedAt(goneAt).
		SaveX(ctx)

	data, err := svc.ExportTickets(ctx, tenantID, map[string]interface{}{}, "csv", ownerID, "super_admin")
	require.NoError(t, err)
	body := string(data)

	require.Contains(t, body, live.TicketNumber)
	require.NotContains(t, body, "TKT-SCOPE-DELETED-001", "已软删工单不得出现在导出文件里")
}

// TestExportTickets_BlankViewerYieldsNoRows 锁住身份缺失时的安全收窄：
// viewerID<=0 不能被当成「无过滤」，否则鉴权链出问题时导出会退化成全租户。
func TestExportTickets_BlankViewerYieldsNoRows(t *testing.T) {
	client, ctx, svc, tenantID, ownerID, _ := newExportFixture(t)
	seedExportTicket(t, ctx, client, tenantID, ownerID, "TKT-SCOPE-GUARD-001", "在库工单")

	data, err := svc.ExportTickets(ctx, tenantID, map[string]interface{}{}, "csv", 0, "end_user")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 1, "无有效身份时只应剩表头行，实际输出：%s", string(data))
}
