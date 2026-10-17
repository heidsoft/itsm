//go:build integration_rls

// Package integration: 真实 PostgreSQL 上的「黄金路径」集成测试。
//
// 该测试覆盖工单、事件、问题、变更、发布、服务请求 6 个核心域的端到端生命周期，
// 并在每个域中验证 PostgreSQL 行级安全（RLS）策略。
//
// 运行：
//
//	ITSM_TEST_DSN='postgres://postgres:postgres@127.0.0.1:55432/itsm?sslmode=disable' \
//	RLS_SETUP_DSN='postgres://postgres:postgres@127.0.0.1:55432/itsm?sslmode=disable' \
//	  go test -tags integration_rls -v ./integration/... -run TestGoldenPath
//
// 必须：
//   - 数据库表结构已迁移（ent migrate 或 atlas apply）
//   - roles 已建立（itsm_app / itsm_admin / itsm_user）
//   - 目标 schema 是迁移过的同一个库
//   - ITSM_TEST_DSN 连接的角色必须能 `SET ROLE itsm_app`，并且不因 RLS 拒绝写入
//     （owner / superuser / BYPASSRLS 角色）。理由：本文件用该连接写入夹具数据，
//     又用它在 `SET ROLE itsm_app` 后做 RLS 可见性探测。若用受限的 itsm_app
//     连接，`FORCE ROW LEVEL SECURITY` 会直接拒绝写入。
//   - RLS_SETUP_DSN 用于设置/清理临时 RLS 策略的 DDL（需要 owner 权限）。
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"itsm-backend/database/rls"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entticket "itsm-backend/ent/ticket"

	"github.com/stretchr/testify/require"
)

// openGoldenDB 打开一个真实 PostgreSQL 连接（其底层 schema 已迁移）。
// 当 ITSM_TEST_DSN 未设置时，跳过测试，避免污染仅 SQLite 的开发者环境。
func openGoldenDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ITSM_TEST_DSN")
	if dsn == "" {
		t.Skip("ITSM_TEST_DSN not set, skipping real-PostgreSQL golden path")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

// openOwnerDBForGolden 用于在测试期间创建临时策略。
func openOwnerDBForGolden(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("RLS_SETUP_DSN")
	if dsn == "" {
		t.Skip("RLS_SETUP_DSN not set, skipping real-PostgreSQL golden path")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

// seedTenantAndUsers 在黄金路径开始前创建两个互不干扰的租户。
// 返回：tenantA.ID, userA.ID, tenantB.ID, userB.ID。
func seedTenantAndUsers(t *testing.T, client *ent.Client) (int, int, int, int) {
	t.Helper()
	ctx := context.Background()

	// 用时间戳后缀保证可重复运行（沙箱 dev 库中允许多次执行）。
	suffix := fmt.Sprintf("GP_%d", time.Now().UnixNano())

	tenantA, err := client.Tenant.Create().
		SetName("Golden Tenant A " + suffix).
		SetCode("GTA_" + suffix).
		SetDomain("gta-" + suffix + ".test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenantB, err := client.Tenant.Create().
		SetName("Golden Tenant B " + suffix).
		SetCode("GTB_" + suffix).
		SetDomain("gtb-" + suffix + ".test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	userA, err := client.User.Create().
		SetUsername("gp_user_a_" + suffix).
		SetEmail("gp_a_" + suffix + "@test.com").
		SetPasswordHash("x").
		SetName("Golden User A").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	userB, err := client.User.Create().
		SetUsername("gp_user_b_" + suffix).
		SetEmail("gp_b_" + suffix + "@test.com").
		SetPasswordHash("x").
		SetName("Golden User B").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenantA.ID, userA.ID, tenantB.ID, userB.ID
}

// TestGoldenPath_Ticket 工单域的端到端黄金路径：
// 创建 → 查询（同租户可见 / 跨租户不可见）→ 更新状态。
func TestGoldenPath_Ticket(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, tenantB, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create ticket in tenant A", func(t *testing.T) {
		tk, err := client.Ticket.Create().
			SetTitle("VPN 连不上").
			SetDescription("Cannot reach internal VPN").
			SetPriority("high").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TKT-GP-%d", time.Now().UnixNano())).
			SetRequesterID(userA).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, tk.ID)

		// 应用层租户隔离：同租户查询可见，跨租户查询不可见。
		// （数据库级 RLS 由 TestGoldenPath_RLSEnforced 在临时策略下验证。）
		visibleA, err := client.Ticket.Query().
			Where(entticket.TenantID(tenantA)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, visibleA, 1, "tenantA 应能看到自己刚创建的工单")

		visibleB, err := client.Ticket.Query().
			Where(entticket.TenantID(tenantB)).
			All(ctx)
		require.NoError(t, err)
		require.Empty(t, visibleB, "tenantB 不得看到 tenantA 的工单")
	})

	t.Run("update status", func(t *testing.T) {
		tk, err := client.Ticket.Create().
			SetTitle("第二个工单").
			SetPriority("low").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TKT-GP2-%d", time.Now().UnixNano())).
			SetRequesterID(userA).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)

		updated, err := client.Ticket.UpdateOneID(tk.ID).
			SetStatus("in_progress").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "in_progress", updated.Status)
		require.Equal(t, tenantA, updated.TenantID)
	})
}

// TestGoldenPath_Incident 事件域端到端：create → resolve。
func TestGoldenPath_Incident(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, _, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create + resolve incident", func(t *testing.T) {
		inc, err := client.Incident.Create().
			SetTitle("Disk full on app server").
			SetDescription("Production DB server out of disk").
			SetPriority("critical").
			SetStatus("open").
			SetIncidentNumber(fmt.Sprintf("INC-GP-%d", time.Now().UnixNano())).
			SetReporterID(userA).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, inc.ID)

		resolved, err := client.Incident.UpdateOneID(inc.ID).
			SetStatus("resolved").
			SetRootCause(map[string]interface{}{"cause": "log volume full"}).
			SetResolutionSteps([]map[string]interface{}{{"action": "rotate logs and grow volume"}}).
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "resolved", resolved.Status)
		require.NotNil(t, resolved.RootCause)
		require.NotEmpty(t, resolved.ResolutionSteps)
	})
}

// TestGoldenPath_Problem 问题域端到端：create → investigate → close。
func TestGoldenPath_Problem(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, _, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create problem + investigation lifecycle", func(t *testing.T) {
		p, err := client.Problem.Create().
			SetTitle("Recurring database timeouts").
			SetDescription("Multiple incidents with similar fingerprint").
			SetPriority("high").
			SetStatus("open").
			SetCategory("infrastructure").
			SetImpact("medium").
			SetCreatedBy(userA).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, p.ID)

		// 进入调查状态
		investigated, err := client.Problem.UpdateOneID(p.ID).
			SetStatus("investigating").
			SetRootCause("connection pool exhausted under load").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "investigating", investigated.Status)
		require.NotEmpty(t, investigated.RootCause)

		// 关闭问题
		closed, err := client.Problem.UpdateOneID(p.ID).
			SetStatus("resolved").
			SetResolution("raise pool size + add monitoring").
			SetWorkaround("restart app periodically").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "resolved", closed.Status)
		require.NotEmpty(t, closed.Resolution)
		require.NotEmpty(t, closed.Workaround)
	})
}

// TestGoldenPath_Change 变更域端到端：create → submit for approval → approve → implement → close。
// 这是用户特别提到的「变更审批」核心场景。
func TestGoldenPath_Change(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, _, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create + approve + implement change", func(t *testing.T) {
		// 1. 创建变更（draft）
		ch, err := client.Change.Create().
			SetTitle("Upgrade PostgreSQL to 17.5").
			SetDescription("Security patch + perf improvement").
			SetType("normal").
			SetPriority("medium").
			SetStatus("draft").
			SetRiskLevel("medium").
			SetTenantID(tenantA).
			SetCreatedBy(userA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, ch.ID)
		require.Equal(t, "draft", string(ch.Status))

		// 2. 提交审批（change.status 枚举：draft/pending/approved/rejected/
		//    scheduled/in_progress/completed/failed/rolled_back/cancelled/closed）
		submitted, err := client.Change.UpdateOneID(ch.ID).
			SetStatus("pending").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "pending", string(submitted.Status))

		// 3. 审批通过
		approved, err := client.Change.UpdateOneID(ch.ID).
			SetStatus("approved").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "approved", string(approved.Status))

		// 4. 实施完成
		implemented, err := client.Change.UpdateOneID(ch.ID).
			SetStatus("completed").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "completed", string(implemented.Status))

		// 5. 关闭
		closed, err := client.Change.UpdateOneID(ch.ID).
			SetStatus("closed").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "closed", string(closed.Status))
	})

	t.Run("rejected change stays in rejected state", func(t *testing.T) {
		ch, err := client.Change.Create().
			SetTitle("High-risk schema migration").
			SetDescription("Drop a column in production").
			SetType("normal").
			SetPriority("high").
			SetStatus("draft").
			SetRiskLevel("high").
			SetTenantID(tenantA).
			SetCreatedBy(userA).
			Save(ctx)
		require.NoError(t, err)

		rejected, err := client.Change.UpdateOneID(ch.ID).
			SetStatus("rejected").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "rejected", string(rejected.Status))

		// 验证：被拒后再次尝试切到 implemented 应被业务校验拦截
		// (此处只验持久化状态机，service 层校验留给 handler/service 集成测试)
	})
}

// TestGoldenPath_Release 发布域端到端：create → schedule → deploy → close。
func TestGoldenPath_Release(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, _, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create + complete release", func(t *testing.T) {
		rel, err := client.Release.Create().
			SetTitle("2026-Q3 Release 1").
			SetDescription("Quarterly batch of changes").
			SetStatus("scheduled").
			SetReleaseNumber(fmt.Sprintf("REL-GP-%d", time.Now().UnixNano())).
			SetCreatedBy(userA).
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, rel.ID)
		require.Equal(t, "scheduled", string(rel.Status))

		// 进入 deploy 阶段
		deployed, err := client.Release.UpdateOneID(rel.ID).
			SetStatus("in-progress").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "in-progress", string(deployed.Status))

		// 发布完成
		closed, err := client.Release.UpdateOneID(rel.ID).
			SetStatus("completed").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "completed", string(closed.Status))
	})
}

// TestGoldenPath_ServiceRequest 服务请求域端到端：create → fulfill → close。
func TestGoldenPath_ServiceRequest(t *testing.T) {
	openGoldenDB(t)
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()

	tenantA, userA, _, _ := seedTenantAndUsers(t, client)
	ctx := context.Background()

	t.Run("create + fulfill service request", func(t *testing.T) {
		catalog, err := client.ServiceCatalog.Create().
			SetName("Jira 账号服务").
			SetStatus("active").
			SetTenantID(tenantA).
			Save(ctx)
		require.NoError(t, err)

		sr, err := client.ServiceRequest.Create().
			SetTitle("申请 Jira 账号").
			SetReason("Need Jira access for new joiner").
			SetCatalogID(catalog.ID).
			SetStatus("submitted").
			SetTenantID(tenantA).
			SetRequesterID(userA).
			Save(ctx)
		require.NoError(t, err)
		require.NotZero(t, sr.ID)
		require.Equal(t, "submitted", sr.Status)

		// 进入审批/履行
		inProgress, err := client.ServiceRequest.UpdateOneID(sr.ID).
			SetStatus("in_progress").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "in_progress", inProgress.Status)

		// 完成
		closed, err := client.ServiceRequest.UpdateOneID(sr.ID).
			SetStatus("completed").
			Save(ctx)
		require.NoError(t, err)
		require.Equal(t, "completed", closed.Status)
	})
}

// TestGoldenPath_RLSEnforced 端到端验证 RLS 在所有 6 个核心域生效。
// 每个域在 tenantA 创建一行，再在 itsm_app + tenant=tenantB 上下文中查询，
// 期望：全部查询不到。
//
// 这是用户「PostgreSQL + RLS_MODE=enforce 而非 SQLite」要求的关键校验。
func TestGoldenPath_RLSEnforced(t *testing.T) {
	openGoldenDB(t)
	openOwnerDBForGolden(t)

	owner := openOwnerDBForGolden(t)
	defer owner.Close()
	ctx := context.Background()

	// 在测试开始时开启 RLS + policy，覆盖 6 张核心表。
	tables := []string{"tickets", "incidents", "problems", "changes", "releases", "service_requests"}
	for _, tbl := range tables {
		stmts := []string{
			fmt.Sprintf(`ALTER TABLE %s ENABLE ROW LEVEL SECURITY`, tbl),
			fmt.Sprintf(`ALTER TABLE %s FORCE ROW LEVEL SECURITY`, tbl),
			fmt.Sprintf(`DROP POLICY IF EXISTS gp_tenant_isolation ON %s`, tbl),
			fmt.Sprintf(`CREATE POLICY gp_tenant_isolation ON %s
				USING       (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint)
				WITH CHECK  (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint)`, tbl),
		}
		for _, s := range stmts {
			if _, err := owner.ExecContext(ctx, s); err != nil {
				t.Logf("policy setup on %s skipped: %v (table may not exist yet)", tbl, err)
			}
		}
	}

	t.Cleanup(func() {
		for _, tbl := range tables {
			for _, s := range []string{
				fmt.Sprintf(`DROP POLICY IF EXISTS gp_tenant_isolation ON %s`, tbl),
				fmt.Sprintf(`ALTER TABLE %s NO FORCE ROW LEVEL SECURITY`, tbl),
				fmt.Sprintf(`ALTER TABLE %s DISABLE ROW LEVEL SECURITY`, tbl),
			} {
				_, _ = owner.ExecContext(ctx, s)
			}
		}
	})

	// 用 ent 客户端在 tenantA 写入所有 6 类记录。
	// 写入连接必须是 owner / superuser / BYPASSRLS 角色：策略已 FORCE RLS，
	// 受限角色在缺少 app.current_tenant 时会连 INSERT 都被拒。
	client := enttest.Open(t, "postgres", os.Getenv("ITSM_TEST_DSN"))
	defer client.Close()
	tenantA, userA, tenantB, _ := seedTenantAndUsers(t, client)

	suffix := time.Now().UnixNano()
	ticket, err := client.Ticket.Create().
		SetTitle("RLS ticket").
		SetTicketNumber(fmt.Sprintf("TKT-RLS-%d", suffix)).
		SetStatus("open").
		SetPriority("low").
		SetRequesterID(userA).
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	incident, err := client.Incident.Create().
		SetTitle("RLS incident").
		SetIncidentNumber(fmt.Sprintf("INC-RLS-%d", suffix)).
		SetStatus("open").
		SetPriority("low").
		SetReporterID(userA).
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	problem, err := client.Problem.Create().
		SetTitle("RLS problem").
		SetStatus("open").
		SetPriority("low").
		SetCreatedBy(userA).
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	change, err := client.Change.Create().
		SetTitle("RLS change").
		SetType("normal").
		SetStatus("draft").
		SetRiskLevel("low").
		SetTenantID(tenantA).
		SetCreatedBy(userA).
		Save(ctx)
	require.NoError(t, err)

	release, err := client.Release.Create().
		SetTitle("RLS release").
		SetReleaseNumber(fmt.Sprintf("REL-RLS-%d", suffix)).
		SetStatus("scheduled").
		SetCreatedBy(userA).
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	catalog, err := client.ServiceCatalog.Create().
		SetName("RLS catalog").
		SetStatus("active").
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)

	sr, err := client.ServiceRequest.Create().
		SetTitle("RLS service request").
		SetCatalogID(catalog.ID).
		SetStatus("submitted").
		SetTenantID(tenantA).
		SetRequesterID(userA).
		Save(ctx)
	require.NoError(t, err)

	checks := []struct {
		table string
		id    int
	}{
		{"tickets", ticket.ID},
		{"incidents", incident.ID},
		{"problems", problem.ID},
		{"changes", change.ID},
		{"releases", release.ID},
		{"service_requests", sr.ID},
	}

	// 在 itsm_app 角色 + 指定租户上下文下统计每张表的可见行数。
	// 这是真实的生产路径：租户由 rls.WithTenant 注入 ctx，AcquireConn 负责
	// set_config('app.current_tenant', ...)，Set ROLE 让策略真正生效。
	// 缺少租户上下文时 AcquireConn 会 fail closed（ErrNoTenant）。
	dsn := os.Getenv("ITSM_TEST_DSN")
	appDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer appDB.Close()

	countVisible := func(tenantID int) []int {
		tenantCtx := rls.WithTenant(context.Background(), int64(tenantID))
		conn, err := rls.AcquireConn(tenantCtx, appDB)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, rls.ReleaseConn(tenantCtx, conn), "释放连接必须清理会话状态")
		}()

		_, err = conn.ExecContext(tenantCtx, "SET ROLE itsm_app")
		require.NoError(t, err)

		counts := make([]int, 0, len(checks))
		for _, c := range checks {
			var n int
			require.NoError(t, conn.QueryRowContext(tenantCtx,
				fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = $1", c.table), c.id).Scan(&n),
				"table %s query failed", c.table)
			counts = append(counts, n)
		}
		return counts
	}

	// 跨租户：tenantB 上下文下 tenantA 的 6 行全部不可见。
	for i, n := range countVisible(tenantB) {
		require.Zero(t, n, "tenantB must not see %s id=%d", checks[i].table, checks[i].id)
	}

	// 正控制：tenantA 上下文下 6 行全部可见——证明上一步的 0 是租户过滤
	// 的结果，而不是策略把所有人都挡在外面。
	for i, n := range countVisible(tenantA) {
		require.Equal(t, 1, n, "tenantA must see %s id=%d", checks[i].table, checks[i].id)
	}
}
