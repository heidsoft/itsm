package service

import (
	"testing"
	"time"

	"itsm-backend/database"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TenantUsageStats 回归锁（缺陷 O8）。
//
// 修复前：`/admin/tenants` 的「资源使用」列恒为 0——后端 DTO 无 userCount /
// ticketCount 字段，列表服务也只查 tenants 表、不做任何聚合，前端 `|| 0`
// 把 undefined 显示成「0 用户 0 工单」。
//
// 本测试跑在 enttest 真实仓储（不是 mock），因此覆盖 ent GroupBy + Aggregate
// 的实际 SQL 语义、软删拦截器是否生效，以及「未命中租户不出现在 map」的契约。

// TestTenantUsageStats_CountsPerTenant 锁住核心语义：每个租户拿到自己的计数，
// 不是全库总数、也不是硬编码 0。
func TestTenantUsageStats_CountsPerTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	// 与生产装配同款（两个注册点都要，覆盖是并集）：
	//   - database.InitDatabaseWithRLS → RegisterSecurityInterceptors（租户写护栏）
	//   - internal/bootstrap/app.go   → RegisterSoftDeleteInterceptors（User/Change 软删）
	// 只注册其中一个会让「软删不计入」这条断言锁住测试夹具而非产品行为。
	database.RegisterSecurityInterceptors(client, "off")
	database.RegisterSoftDeleteInterceptors(client)
	ctx := t.Context()
	svc := NewTenantService(client, zaptest.NewLogger(t).Sugar())

	t1, err := client.Tenant.Create().SetName("用量租户A").SetCode("usage-a").Save(ctx)
	require.NoError(t, err)
	t2, err := client.Tenant.Create().SetName("用量租户B").SetCode("usage-b").Save(ctx)
	require.NoError(t, err)

	// 租户 A：3 个用户（第 3 个停用，仍占席位）+ 2 张工单
	for i := 0; i < 3; i++ {
		name := "usage-a-" + string(rune('a'+i))
		u, createErr := client.User.Create().
			SetUsername(name).
			SetEmail(name + "@x.test").
			SetName(name).
			SetPasswordHash("x").
			SetActive(i < 2).
			SetTenantID(t1.ID).Save(ctx)
		require.NoError(t, createErr)
		if i == 0 {
			for j := 0; j < 2; j++ {
				_, ticketErr := client.Ticket.Create().
					SetTitle("用量工单").
					SetTicketNumber("USAGE-A-" + string(rune('1'+j))).
					SetRequesterID(u.ID).
					SetStatus("open").
					SetTenantID(t1.ID).Save(ctx)
				require.NoError(t, ticketErr)
			}
		}
	}

	// 租户 B：1 个用户 0 张工单
	_, err = client.User.Create().
		SetUsername("usage-b-a").
		SetEmail("usage-b-a@x.test").
		SetName("usage-b-a").
		SetPasswordHash("x").
		SetTenantID(t2.ID).Save(ctx)
	require.NoError(t, err)

	stats, err := svc.TenantUsageStats(ctx, []int{t1.ID, t2.ID})
	require.NoError(t, err)

	assert.Equal(t, dto.TenantUsageStat{UserCount: 3, TicketCount: 2}, stats[t1.ID],
		"租户 A：停用用户仍计入席位，只有软删才排除")
	assert.Equal(t, dto.TenantUsageStat{UserCount: 1, TicketCount: 0}, stats[t2.ID],
		"租户 B 的 0 工单是真实读数，不是聚合失败")
}

// TestTenantUsageStats_ExcludesSoftDeleted 锁软删口径：
// 「删除」= deleted_at 非空（与「停用」active=false 语义分离，见 database/softdelete.go）。
func TestTenantUsageStats_ExcludesSoftDeleted(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	// 与生产装配同款（两个注册点都要，覆盖是并集）：
	//   - database.InitDatabaseWithRLS → RegisterSecurityInterceptors（租户写护栏）
	//   - internal/bootstrap/app.go   → RegisterSoftDeleteInterceptors（User/Change 软删）
	// 只注册其中一个会让「软删不计入」这条断言锁住测试夹具而非产品行为。
	database.RegisterSecurityInterceptors(client, "off")
	database.RegisterSoftDeleteInterceptors(client)
	ctx := t.Context()
	svc := NewTenantService(client, zaptest.NewLogger(t).Sugar())

	tnt, err := client.Tenant.Create().SetName("软删租户").SetCode("usage-softdel").Save(ctx)
	require.NoError(t, err)

	// 两个活跃用户
	for i := 0; i < 2; i++ {
		name := "live-" + string(rune('a'+i))
		_, createErr := client.User.Create().
			SetUsername(name).
			SetEmail(name + "@x.test").
			SetName(name).
			SetPasswordHash("x").
			SetTenantID(tnt.ID).Save(ctx)
		require.NoError(t, createErr)
	}
	// 一个已软删用户
	deletedUser, err := client.User.Create().
		SetUsername("gone").
		SetEmail("gone@x.test").
		SetName("gone").
		SetPasswordHash("x").
		SetActive(false).
		SetTenantID(tnt.ID).Save(ctx)
	require.NoError(t, err)
	err = client.User.UpdateOneID(deletedUser.ID).
		SetDeletedAt(time.Now()).
		Exec(ctx)
	require.NoError(t, err)

	// 一张活跃工单 + 一张已软删工单
	requester, err := client.User.Query().Where().First(ctx)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		tk, createErr := client.Ticket.Create().
			SetTitle("软删口径工单").
			SetTicketNumber("USAGE-SD-" + string(rune('1'+i))).
			SetRequesterID(requester.ID).
			SetStatus("open").
			SetTenantID(tnt.ID).Save(ctx)
		require.NoError(t, createErr)
		if i == 1 {
			delErr := client.Ticket.UpdateOneID(tk.ID).SetDeletedAt(time.Now()).Exec(ctx)
			require.NoError(t, delErr)
		}
	}

	stats, err := svc.TenantUsageStats(ctx, []int{tnt.ID})
	require.NoError(t, err)

	assert.Equal(t, 2, stats[tnt.ID].UserCount, "软删用户不得计入")
	assert.Equal(t, 1, stats[tnt.ID].TicketCount, "软删工单不得计入")
}

// TestTenantUsageStats_EmptyInput 空入参不查库、返回空 map（列表页无租户时不打库）。
func TestTenantUsageStats_EmptyInput(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	// 与生产装配同款（两个注册点都要，覆盖是并集）：
	//   - database.InitDatabaseWithRLS → RegisterSecurityInterceptors（租户写护栏）
	//   - internal/bootstrap/app.go   → RegisterSoftDeleteInterceptors（User/Change 软删）
	// 只注册其中一个会让「软删不计入」这条断言锁住测试夹具而非产品行为。
	database.RegisterSecurityInterceptors(client, "off")
	database.RegisterSoftDeleteInterceptors(client)
	ctx := t.Context()
	svc := NewTenantService(client, zaptest.NewLogger(t).Sugar())

	stats, err := svc.TenantUsageStats(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, stats)

	stats, err = svc.TenantUsageStats(ctx, []int{})
	require.NoError(t, err)
	assert.Empty(t, stats)
}

// TestTenantUsageStats_AbsentTenantNotZeroFilled 锁「缺失 ≠ 0」契约：
// 查不到数据的租户不出现在 map 里，调用方据此知道「未聚合」而不是「0 个用户」。
func TestTenantUsageStats_AbsentTenantNotZeroFilled(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	// 与生产装配同款（两个注册点都要，覆盖是并集）：
	//   - database.InitDatabaseWithRLS → RegisterSecurityInterceptors（租户写护栏）
	//   - internal/bootstrap/app.go   → RegisterSoftDeleteInterceptors（User/Change 软删）
	// 只注册其中一个会让「软删不计入」这条断言锁住测试夹具而非产品行为。
	database.RegisterSecurityInterceptors(client, "off")
	database.RegisterSoftDeleteInterceptors(client)
	ctx := t.Context()
	svc := NewTenantService(client, zaptest.NewLogger(t).Sugar())

	stats, err := svc.TenantUsageStats(ctx, []int{9001})
	require.NoError(t, err)
	_, present := stats[9001]
	assert.False(t, present, "无数据的租户必须缺席，不得返回 {0,0} 假装已聚合")
}
