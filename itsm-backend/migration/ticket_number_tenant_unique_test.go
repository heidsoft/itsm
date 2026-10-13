package migration

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	schemadsl "entgo.io/ent/dialect/sql/schema"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/migrate"
	entschema "itsm-backend/ent/schema"
)

// 工单号唯一性作用域回归。
//
// 缺陷：生成域是租户内的（repository/ticket/ticket_number_repository.go 按
// tenant_id + prefix 取 max+1，existsNumber 也带 tenant_id），而约束域是全局的
// （ent/schema/ticket.go 曾把 ticket_number 声明为字段级 Unique）。两者不一致时，
// MSP 场景下新开通租户当月首单必然算出与既有租户相同的编号，撞全局唯一索引；
// repository/ticket/repository_impl.go:769 的三次重试每次都重新算出同一个号，
// 于是建单永久失败。与 users.username(20261006)、menus(20261011) 同一缺陷类。
//
// 本文件的前两个测试不依赖任何外部数据库，因此在阻断型 backend-ci/test 作业里
// 真正执行；PostgreSQL 那个测试沿用既有 ITSM_INITIALIZATION_TEST_DSN 跑法。

const ticketNumberUniqueVersion = "20261013_tenant_scope_ticket_number_unique"

// uniqueTicketNumberIndexes 返回携带 ticket_number 的唯一索引名及其字段序列。
func uniqueTicketNumberIndexes(table *schemadsl.Table) map[string][]string {
	found := make(map[string][]string)
	for _, idx := range table.Indexes {
		if !idx.Unique {
			continue
		}
		cols := make([]string, 0, len(idx.Columns))
		hasTicketNumber := false
		for _, c := range idx.Columns {
			cols = append(cols, c.Name)
			if c.Name == "ticket_number" {
				hasTicketNumber = true
			}
		}
		if hasTicketNumber {
			found[idx.Name] = cols
		}
	}
	return found
}

// TestTicketNumberUniqueScope_GeneratedSchemaIsTenantScoped 锁住生成的 migrate
// schema：ticket_number 只能出现在 (tenant_id, ticket_number) 复合唯一索引里，
// 列级不得再带全局唯一。这是自动建表路径（bootstrap 中 Schema.Create 先于版本化
// 迁移执行）的唯一真相，手工编辑过的生成文件由此测试兜底。
func TestTicketNumberUniqueScope_GeneratedSchemaIsTenantScoped(t *testing.T) {
	indexes := uniqueTicketNumberIndexes(migrate.TicketsTable)
	require.Equal(t, map[string][]string{
		"ticket_tenant_id_ticket_number": {"tenant_id", "ticket_number"},
	}, indexes, "tickets 上涉及 ticket_number 的唯一索引必须恰好是租户内复合唯一")

	for _, col := range migrate.TicketsColumns {
		if col.Name == "ticket_number" {
			require.False(t, col.Unique, "ticket_number 列不得再声明全局唯一")
			return
		}
	}
	t.Fatal("generated schema has no ticket_number column")
}

// TestTicketNumberUniqueScope_SourceMatchesGenerated 防止两套真相：ent/schema 源
// 与生成代码必须一致，否则手改生成文件会在下一次 codegen 被静默还原。
func TestTicketNumberUniqueScope_SourceMatchesGenerated(t *testing.T) {
	ticketSchema := entschema.Ticket{}

	var fromSource []string
	for _, idx := range ticketSchema.Indexes() {
		d := idx.Descriptor()
		if !d.Unique || !contains(d.Fields, "ticket_number") {
			continue
		}
		fromSource = append(fromSource, strings.Join(d.Fields, ","))
	}
	require.Equal(t, []string{"tenant_id,ticket_number"}, fromSource,
		"ent/schema/ticket.go 必须把 ticket_number 声明为租户内复合唯一")

	for _, f := range ticketSchema.Fields() {
		d := f.Descriptor()
		if d.Name == "ticket_number" {
			require.False(t, d.Unique, "ticket_number 字段级 Unique 必须移除")
			return
		}
	}
	t.Fatal("ent/schema/ticket.go has no ticket_number field")
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// TestTicketNumberUniqueScope_TwoTenantsMayShareNumber 是行为级证明，跑在 sqlite
// 上因此 CI 必执行：跨租户同号必须放行（修复前失败），同租户重号必须仍被拒绝
// （否则等于把约束整个删掉）。
func TestTicketNumberUniqueScope_TwoTenantsMayShareNumber(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3",
		"file:ticket_number_unique?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	tenantA := createTicketTenant(t, ctx, client, "tenant-a", "a.example.com")
	tenantB := createTicketTenant(t, ctx, client, "tenant-b", "b.example.com")
	userA := createTicketUser(t, ctx, client, tenantA, "user-a")
	userB := createTicketUser(t, ctx, client, tenantB, "user-b")

	const shared = "TKT-202610-000001"
	require.NoError(t, createTicket(t, ctx, client, tenantA, userA, shared),
		"租户 A 的当月首单")

	require.NoError(t, createTicket(t, ctx, client, tenantB, userB, shared),
		"租户 B 的当月首单算出同一个号，修复前会被全局唯一索引拒绝")

	err := createTicket(t, ctx, client, tenantB, userB, shared)
	require.Error(t, err, "同租户内重号必须仍被拒绝")
	require.True(t, ent.IsConstraintError(err),
		"expected a constraint error, got %T: %v", err, err)
}

func createTicketTenant(t *testing.T, ctx context.Context, client *ent.Client, code, domain string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(code).SetCode(code).SetDomain(domain).SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID
}

func createTicketUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, name string) int {
	t.Helper()
	u, err := client.User.Create().
		SetName(name).SetUsername(name).SetEmail(name + "@example.com").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u.ID
}

func createTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID, requesterID int, number string) error {
	t.Helper()
	_, err := client.Ticket.Create().
		SetTitle("ticket " + number).
		SetType("incident").
		SetStatus("open").
		SetPriority("medium").
		SetTicketNumber(number).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

// legacyTicketsDDL 复刻修复前的形状：ticket_number 带字段级全局唯一，PostgreSQL
// 把它命名为 tickets_ticket_number_key，ent 自动建表另外生成 ticket_ticket_number。
const legacyTicketsDDL = `
CREATE TABLE tickets (
	id serial PRIMARY KEY,
	title text NOT NULL,
	ticket_number text NOT NULL UNIQUE,
	tenant_id integer NOT NULL,
	requester_id integer NOT NULL
);
CREATE UNIQUE INDEX ticket_ticket_number ON tickets (ticket_number)`

func uniqueTicketIndexesInPG(t *testing.T, ctx context.Context, db *sql.DB, schema string) []string {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
		SELECT i.indexname
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_index x ON x.indexrelid = c.oid
		WHERE i.schemaname = $1 AND i.tablename = 'tickets' AND x.indisunique
		  AND EXISTS (
			SELECT 1 FROM unnest(x.indkey) AS k(attnum)
			JOIN pg_attribute a ON a.attrelid = x.indrelid AND a.attnum = k.attnum
			WHERE a.attname = 'ticket_number')
		ORDER BY i.indexname`, schema)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func insertTicketPG(t *testing.T, ctx context.Context, db *sql.DB, tenantID int, number string) error {
	t.Helper()
	_, err := db.ExecContext(ctx,
		`INSERT INTO tickets (title, ticket_number, tenant_id, requester_id) VALUES ('t', $1, $2, 1)`,
		number, tenantID)
	return err
}

// TestTicketNumberUniqueMigration_DiscoveredWithRollback 不依赖数据库，因此 CI 必执行：
// 「目录即真相」自发现必须捡到这份迁移，且 *_down.sql 正确成对。文件名写错时
// 下面所有 PostgreSQL 测试只会静默 skip，唯有这条能拦住。
func TestTicketNumberUniqueMigration_DiscoveredWithRollback(t *testing.T) {
	migs, err := FilesystemMigrations("../migrations")
	require.NoError(t, err)

	var found *Migration
	for i := range migs {
		if migs[i].Version == ticketNumberUniqueVersion {
			found = &migs[i]
			break
		}
	}
	require.NotNil(t, found, "migration %s must be discovered on disk", ticketNumberUniqueVersion)
	require.NotEmpty(t, found.SQLContent, "up script must carry SQL")
	require.NotEmpty(t, found.RollbackSQL, "down script must be paired via the _down.sql suffix")

	require.Contains(t, found.SQLContent, "RAISE EXCEPTION",
		"存量重复必须中止迁移，而不是静默跳过")
	require.Contains(t, found.SQLContent, "ticket_tenant_id_ticket_number")
	require.Contains(t, found.RollbackSQL, "tickets_ticket_number_key",
		"rollback must restore the legacy global key")
}

// TestTicketNumberUniqueMigration_ReplacesGlobalKeyWithTenantScope 跑迁移文件的真实
// 字节：修复前第二个租户建不出同号工单，修复后跨租户放行、同租户仍拒绝。
func TestTicketNumberUniqueMigration_ReplacesGlobalKeyWithTenantScope(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	_, err := db.ExecContext(ctx, legacyTicketsDDL)
	require.NoError(t, err)
	require.NoError(t, insertTicketPG(t, ctx, db, 1, "TKT-202610-000001"))

	requireUniqueViolation(t, insertTicketPG(t, ctx, db, 2, "TKT-202610-000001"),
		"legacy global key must block a second tenant's first ticket of the month")
	require.Equal(t, []string{"ticket_ticket_number"}, uniqueTicketIndexesInPG(t, ctx, db, schema))

	_, err = db.ExecContext(ctx, migrationSQL(t, ticketNumberUniqueVersion))
	require.NoError(t, err, "migration must apply cleanly to the legacy shape")

	require.NoError(t, insertTicketPG(t, ctx, db, 2, "TKT-202610-000001"),
		"each tenant must own its own numbering space")
	requireUniqueViolation(t, insertTicketPG(t, ctx, db, 2, "TKT-202610-000001"),
		"per-tenant uniqueness must survive the relaxation")
	require.Equal(t, []string{"ticket_tenant_id_ticket_number"}, uniqueTicketIndexesInPG(t, ctx, db, schema))
}

// TestTicketNumberUniqueMigration_RefusesExistingDuplicates 锁住「先检测再建索引」：
// 存量同租户重号必须让迁移中止，且不得留下半个索引。
func TestTicketNumberUniqueMigration_RefusesExistingDuplicates(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	_, err := db.ExecContext(ctx, legacyTicketsDDL)
	require.NoError(t, err)
	// 同租户重号只可能在全局唯一键消失后存在，手工 dropping 以到达守卫必须拦截的状态。
	_, err = db.ExecContext(ctx, `ALTER TABLE tickets DROP CONSTRAINT tickets_ticket_number_key`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DROP INDEX ticket_ticket_number`)
	require.NoError(t, err)
	require.NoError(t, insertTicketPG(t, ctx, db, 1, "TKT-202610-000001"))
	require.NoError(t, insertTicketPG(t, ctx, db, 1, "TKT-202610-000001"))

	_, err = db.ExecContext(ctx, migrationSQL(t, ticketNumberUniqueVersion))
	require.Error(t, err, "duplicate (tenant_id, ticket_number) rows must abort the migration")
	require.Contains(t, err.Error(), "deduplicate")
	require.NotContains(t, uniqueTicketIndexesInPG(t, ctx, db, schema), "ticket_tenant_id_ticket_number",
		"the failing migration must leave no half-applied index set")
}

// TestTicketNumberUniqueMigration_MatchesGeneratedEntSchema 防止两套真相：自动建表
// 与版本化迁移必须收敛到同一个索引名，否则每次升级都会多建或漏建索引。
func TestTicketNumberUniqueMigration_MatchesGeneratedEntSchema(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Schema.Create(ctx))

	require.Equal(t, []string{"ticket_tenant_id_ticket_number"}, uniqueTicketIndexesInPG(t, ctx, db, schema),
		"ent auto-migrate must install the tenant-scoped index")

	script := migrationSQL(t, ticketNumberUniqueVersion)
	for round := 1; round <= 2; round++ {
		_, err := db.ExecContext(ctx, script)
		require.NoErrorf(t, err, "migration must be idempotent on an ent-managed database (round %d)", round)
	}
	require.Equal(t, []string{"ticket_tenant_id_ticket_number"}, uniqueTicketIndexesInPG(t, ctx, db, schema))
}
