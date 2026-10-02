package ticket

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/common/datascope"
	ticketrepo "itsm-backend/repository/ticket"

	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustCreateUser 建一个最小可用用户：ticket.requester_id 是指向 users 的必需外键。
func mustCreateUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, username string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@hdscope.local").
		SetName(username).
		SetPasswordHash("hashed").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u
}

// seedScopeTicket 直接写库，绕过 handler，让断言只落在行级谓词上。
func seedScopeTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, number string, requesterID int, assigneeID *int) {
	t.Helper()
	builder := client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle(number).
		SetDescription("desc").
		SetPriority("medium").
		SetStatus("new").
		SetType("incident").
		SetRequesterID(requesterID).
		SetTenantID(tenantID)
	if assigneeID != nil {
		builder.SetAssigneeID(*assigneeID)
	}
	_, err := builder.Save(ctx)
	require.NoError(t, err)
}

// TestEntRepository_List_DataScopeMapping 复现并锁死 2026-10-02 实测的越权读取：
// GET /api/v1/tickets 对 technician 返回租户全量 45 条（本人只应看到 23 条）。
// 根因是 handlers/ticket/repository_impl.go 用 ticket.DataScope(dataScope) 做数值转换：
// datascope 枚举含 DataScopeDepartment(1) 使 OwnedOrAssigned=2，而仓储层只有两档
// （OwnedOrAssigned=1），转换后的 2 让仓储谓词永不命中、行级权限静默失效。
// 同一场景在 service/ticket_service_test.go 里一直是绿的，因为它走旧 TicketService
// 路径并直接使用仓储层枚举，没有覆盖这条真正接在路由上的适配边界。
func TestEntRepository_List_DataScopeMapping(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_datascope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("DataScope Tenant").
		SetCode("hdscope").
		SetDomain("hdscope.local").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	alice := mustCreateUser(t, ctx, client, tenant.ID, "alice")
	bob := mustCreateUser(t, ctx, client, tenant.ID, "bob")

	seedScopeTicket(t, ctx, client, tenant.ID, "DS-1", alice.ID, nil)     // alice 创建
	seedScopeTicket(t, ctx, client, tenant.ID, "DS-2", bob.ID, nil)       // bob 独有（薪酬敏感）
	seedScopeTicket(t, ctx, client, tenant.ID, "DS-3", bob.ID, &alice.ID) // bob 创建但分配给 alice

	repo := NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()))

	listNumbers := func(ds datascope.DataScope, userID int) []string {
		t.Helper()
		tickets, total, err := repo.List(ctx, tenant.ID, 1, 100, nil, ds, userID)
		require.NoError(t, err)
		got := make([]string, 0, len(tickets))
		for _, tk := range tickets {
			got = append(got, tk.TicketNumber)
		}
		require.Len(t, got, total, "信封 total 与 items 必须一致")
		return got
	}

	assert.ElementsMatch(t, []string{"DS-1", "DS-3"}, listNumbers(datascope.DataScopeOwnedOrAssigned, alice.ID),
		"非管理角色只能看到本人创建或受理的工单")
	assert.ElementsMatch(t, []string{"DS-2", "DS-3"}, listNumbers(datascope.DataScopeOwnedOrAssigned, bob.ID),
		"bob 可见本人创建的两张，DS-1 属 alice 不可见")
	assert.ElementsMatch(t, []string{"DS-1", "DS-2", "DS-3"}, listNumbers(datascope.DataScopeAll, alice.ID),
		"管理角色可见全租户")

	// Department 档在仓储层没有对应实现，必须按最窄档处理而不是放宽。
	assert.ElementsMatch(t, []string{"DS-1", "DS-3"}, listNumbers(datascope.DataScopeDepartment, alice.ID),
		"仓储层不支持的档位必须 fail closed 收窄")

	// 身份缺失时 fail closed：返回空集，不能回落到全量。
	assert.Empty(t, listNumbers(datascope.DataScopeOwnedOrAssigned, 0),
		"缺少 userID 时行级权限必须返回空集")
}
