package migration

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

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

// vendors.code / contracts.contract_number 唯一性作用域回归。
//
// 缺陷：两张表都带 tenant_id 且业务读写按租户过滤（handlers/vendor/service.go），
// 但 ent 字段级 Unique 把约束域声明成全局。MSP 场景下新租户无法使用其他租户
// 已占用的供应商编码或合同编号，与 users.username(20261006)、
// tickets.ticket_number(20261013) 同一缺陷类。
//
// contracts 当前没有 HTTP 生产入口（仅 ent 生成代码引用），行为证明落在
// ent 约束层；vendors 的生产入口证明在 router/vendor_routes_test.go
// TestVendorCodeUniqueIsTenantScoped。
//
// sqlite 测试不依赖外部数据库，CI 必执行；PostgreSQL 测试沿用
// ITSM_INITIALIZATION_TEST_DSN 隔离跑法，无 DSN 时 skip。

const vendorContractUniqueVersion = "20261113_tenant_scope_vendor_contract_unique"

// uniqueIndexesOnColumnFromTable 返回生成的 migrate schema 里携带指定列的
// 唯一索引名及其字段序列。
func uniqueIndexesOnColumnFromTable(table *schemadsl.Table, column string) map[string][]string {
	found := make(map[string][]string)
	for _, idx := range table.Indexes {
		if !idx.Unique {
			continue
		}
		cols := make([]string, 0, len(idx.Columns))
		hasColumn := false
		for _, c := range idx.Columns {
			cols = append(cols, c.Name)
			if c.Name == column {
				hasColumn = true
			}
		}
		if hasColumn {
			found[idx.Name] = cols
		}
	}
	return found
}

func requireColumnNotGloballyUnique(t *testing.T, cols []*schemadsl.Column, name string) {
	t.Helper()
	for _, col := range cols {
		if col.Name == name {
			require.False(t, col.Unique, "%s 列不得再声明全局唯一", name)
			return
		}
	}
	t.Fatalf("generated schema has no %s column", name)
}

// TestVendorContractUniqueScope_GeneratedSchemaIsTenantScoped 锁住生成的
// migrate schema：code / contract_number 只能出现在租户组合唯一索引里，
// 列级不得再带全局唯一。这是自动建表路径的唯一真相。
func TestVendorContractUniqueScope_GeneratedSchemaIsTenantScoped(t *testing.T) {
	require.Equal(t, map[string][]string{
		"vendor_tenant_id_code": {"tenant_id", "code"},
	}, uniqueIndexesOnColumnFromTable(migrate.VendorsTable, "code"),
		"vendors 上涉及 code 的唯一索引必须恰好是租户内复合唯一")
	require.Equal(t, map[string][]string{
		"contract_tenant_id_contract_number": {"tenant_id", "contract_number"},
	}, uniqueIndexesOnColumnFromTable(migrate.ContractsTable, "contract_number"),
		"contracts 上涉及 contract_number 的唯一索引必须恰好是租户内复合唯一")

	requireColumnNotGloballyUnique(t, migrate.VendorsColumns, "code")
	requireColumnNotGloballyUnique(t, migrate.ContractsColumns, "contract_number")
}

// TestVendorContractUniqueScope_SourceMatchesGenerated 防止两套真相：
// ent/schema 源与生成代码必须一致，否则手改生成文件会在下一次 codegen 被静默还原。
func TestVendorContractUniqueScope_SourceMatchesGenerated(t *testing.T) {
	vendorSchema := entschema.Vendor{}
	var vendorFromSource []string
	for _, idx := range vendorSchema.Indexes() {
		d := idx.Descriptor()
		if d.Unique && contains(d.Fields, "code") {
			vendorFromSource = append(vendorFromSource, strings.Join(d.Fields, ","))
		}
	}
	require.Equal(t, []string{"tenant_id,code"}, vendorFromSource,
		"ent/schema/vendor.go 必须把 code 声明为租户内复合唯一")
	for _, f := range vendorSchema.Fields() {
		if d := f.Descriptor(); d.Name == "code" {
			require.False(t, d.Unique, "code 字段级 Unique 必须移除")
		}
	}

	contractSchema := entschema.Contract{}
	var contractFromSource []string
	for _, idx := range contractSchema.Indexes() {
		d := idx.Descriptor()
		if d.Unique && contains(d.Fields, "contract_number") {
			contractFromSource = append(contractFromSource, strings.Join(d.Fields, ","))
		}
	}
	require.Equal(t, []string{"tenant_id,contract_number"}, contractFromSource,
		"ent/schema/contract.go 必须把 contract_number 声明为租户内复合唯一")
	for _, f := range contractSchema.Fields() {
		if d := f.Descriptor(); d.Name == "contract_number" {
			require.False(t, d.Unique, "contract_number 字段级 Unique 必须移除")
		}
	}
}

func createVendorForUniqueScope(ctx context.Context, client *ent.Client, tenantID int, code string) error {
	_, err := client.Vendor.Create().
		SetName("vendor " + code).SetCode(code).SetVendorType("IT服务").
		SetContactName("联系人").SetContactEmail("contact@example.com").SetContactPhone("13800000000").
		SetAddress("地址").SetWebsite("https://example.com").
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

func createContractForUniqueScope(ctx context.Context, client *ent.Client, tenantID int, number string) error {
	now := time.Now()
	_, err := client.Contract.Create().
		SetContractNumber(number).SetTitle("contract " + number).SetContractType("服务协议").
		SetStartDate(now).SetEndDate(now.Add(24 * time.Hour)).SetDescription("").
		SetTenantID(tenantID).
		Save(ctx)
	return err
}

// TestVendorContractUniqueScope_TwoTenantsMayShareCodeAndNumber 是行为级证明，
// 跑在 sqlite 上因此 CI 必执行：跨租户同编码/同合同号必须放行（修复前失败），
// 同租户重复必须仍被拒绝（否则等于把约束整个删掉）。
func TestVendorContractUniqueScope_TwoTenantsMayShareCodeAndNumber(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3",
		"file:vendor_contract_unique?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	tenantA := client.Tenant.Create().SetName("VC A").SetCode("vc-a").SetDomain("vc-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("VC B").SetCode("vc-b").SetDomain("vc-b.example.com").SetStatus("active").SaveX(ctx)

	const sharedCode = "hw-001"
	require.NoError(t, createVendorForUniqueScope(ctx, client, tenantA.ID, sharedCode))
	require.NoError(t, createVendorForUniqueScope(ctx, client, tenantB.ID, sharedCode),
		"租户 B 使用同编码，修复前会被全局唯一约束拒绝")
	err := createVendorForUniqueScope(ctx, client, tenantB.ID, sharedCode)
	require.Error(t, err, "同租户内重编码必须仍被拒绝")
	require.True(t, ent.IsConstraintError(err), "expected a constraint error, got %T: %v", err, err)

	const sharedNumber = "HT-2026-001"
	require.NoError(t, createContractForUniqueScope(ctx, client, tenantA.ID, sharedNumber))
	require.NoError(t, createContractForUniqueScope(ctx, client, tenantB.ID, sharedNumber),
		"租户 B 使用同合同号，修复前会被全局唯一约束拒绝")
	err = createContractForUniqueScope(ctx, client, tenantB.ID, sharedNumber)
	require.Error(t, err, "同租户内重合同号必须仍被拒绝")
	require.True(t, ent.IsConstraintError(err), "expected a constraint error, got %T: %v", err, err)
}

// TestVendorContractUniqueMigration_DiscoveredWithRollback 不依赖数据库，CI 必执行：
// 「目录即真相」自发现必须捡到这份迁移，且 *_down.sql 正确成对。文件名写错时
// 下面所有 PostgreSQL 测试只会静默 skip，唯有这条能拦住。
func TestVendorContractUniqueMigration_DiscoveredWithRollback(t *testing.T) {
	migs, err := FilesystemMigrations("../migrations")
	require.NoError(t, err)

	var found *Migration
	for i := range migs {
		if migs[i].Version == vendorContractUniqueVersion {
			found = &migs[i]
			break
		}
	}
	require.NotNil(t, found, "migration %s must be discovered on disk", vendorContractUniqueVersion)
	require.NotEmpty(t, found.SQLContent, "up script must carry SQL")
	require.NotEmpty(t, found.RollbackSQL, "down script must be paired via the _down.sql suffix")

	require.Contains(t, found.SQLContent, "RAISE EXCEPTION", "存量重复必须中止迁移，而不是静默跳过")
	require.Contains(t, found.SQLContent, "vendor_tenant_id_code")
	require.Contains(t, found.SQLContent, "contract_tenant_id_contract_number")
	require.Contains(t, found.RollbackSQL, "vendors_code_key", "rollback must restore the legacy global key")
	require.Contains(t, found.RollbackSQL, "contracts_contract_number_key")
}

// legacyVendorsDDL / legacyContractsDDL 复刻修复前的实测生产形状：ent 字段级
// Unique 在 PostgreSQL 上物化为独立唯一索引 vendors_code_key /
// contracts_contract_number_key（pg_constraint 无对应行，不是列约束）。
const legacyVendorsDDL = `
CREATE TABLE vendors (
	id serial PRIMARY KEY,
	name text NOT NULL,
	code text NOT NULL,
	tenant_id integer NOT NULL
);
CREATE UNIQUE INDEX vendors_code_key ON vendors (code)`

const legacyContractsDDL = `
CREATE TABLE contracts (
	id serial PRIMARY KEY,
	contract_number text NOT NULL,
	tenant_id integer NOT NULL
);
CREATE UNIQUE INDEX contracts_contract_number_key ON contracts (contract_number)`

func uniqueIndexesOnColumnInPG(t *testing.T, ctx context.Context, db *sql.DB, schema, table, column string) []string {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
		SELECT i.indexname
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_index x ON x.indexrelid = c.oid
		WHERE i.schemaname = $1 AND i.tablename = $2 AND x.indisunique
		  AND EXISTS (
			SELECT 1 FROM unnest(x.indkey) AS k(attnum)
			JOIN pg_attribute a ON a.attrelid = x.indrelid AND a.attnum = k.attnum
			WHERE a.attname = $3)
		ORDER BY i.indexname`, schema, table, column)
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

func insertVendorPG(t *testing.T, ctx context.Context, db *sql.DB, tenantID int, code string) error {
	t.Helper()
	_, err := db.ExecContext(ctx,
		`INSERT INTO vendors (name, code, tenant_id) VALUES ('v', $1, $2)`, code, tenantID)
	return err
}

func insertContractPG(t *testing.T, ctx context.Context, db *sql.DB, tenantID int, number string) error {
	t.Helper()
	_, err := db.ExecContext(ctx,
		`INSERT INTO contracts (contract_number, tenant_id) VALUES ($1, $2)`, number, tenantID)
	return err
}

// TestVendorContractUniqueMigration_ReplacesGlobalKeyWithTenantScope 跑迁移文件
// 的真实字节：修复前第二个租户用不了同编码/同合同号，修复后跨租户放行、
// 同租户仍拒绝。
func TestVendorContractUniqueMigration_ReplacesGlobalKeyWithTenantScope(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	_, err := db.ExecContext(ctx, legacyVendorsDDL)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, legacyContractsDDL)
	require.NoError(t, err)

	require.NoError(t, insertVendorPG(t, ctx, db, 1, "hw-001"))
	require.NoError(t, insertContractPG(t, ctx, db, 1, "HT-2026-001"))

	requireUniqueViolation(t, insertVendorPG(t, ctx, db, 2, "hw-001"),
		"legacy global key must block a second tenant's vendor code")
	requireUniqueViolation(t, insertContractPG(t, ctx, db, 2, "HT-2026-001"),
		"legacy global key must block a second tenant's contract number")
	require.Equal(t, []string{"vendors_code_key"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "vendors", "code"))
	require.Equal(t, []string{"contracts_contract_number_key"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "contracts", "contract_number"))

	_, err = db.ExecContext(ctx, migrationSQL(t, vendorContractUniqueVersion))
	require.NoError(t, err, "migration must apply cleanly to the legacy shape")

	require.NoError(t, insertVendorPG(t, ctx, db, 2, "hw-001"),
		"each tenant must own its own vendor code space")
	require.NoError(t, insertContractPG(t, ctx, db, 2, "HT-2026-001"),
		"each tenant must own its own contract number space")
	requireUniqueViolation(t, insertVendorPG(t, ctx, db, 2, "hw-001"),
		"per-tenant vendor code uniqueness must survive the relaxation")
	requireUniqueViolation(t, insertContractPG(t, ctx, db, 2, "HT-2026-001"),
		"per-tenant contract number uniqueness must survive the relaxation")
	require.Equal(t, []string{"vendor_tenant_id_code"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "vendors", "code"))
	require.Equal(t, []string{"contract_tenant_id_contract_number"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "contracts", "contract_number"))
}

// TestVendorContractUniqueMigration_RefusesExistingDuplicates 锁住「先检测再建索引」：
// 存量同租户重复必须让迁移中止，且不得留下半个索引。两张表各自独立证明。
func TestVendorContractUniqueMigration_RefusesExistingDuplicates(t *testing.T) {
	t.Run("vendors", func(t *testing.T) {
		ctx, db := newUserUniqueTestSchema(t)
		schema := schemaOfTestDB(t, ctx, db)

		_, err := db.ExecContext(ctx, legacyVendorsDDL)
		require.NoError(t, err)
		// 同租户重复只可能在全局唯一键消失后存在，手工 dropping 以到达守卫必须拦截的状态。
		_, err = db.ExecContext(ctx, `DROP INDEX vendors_code_key`)
		require.NoError(t, err)
		require.NoError(t, insertVendorPG(t, ctx, db, 1, "hw-001"))
		require.NoError(t, insertVendorPG(t, ctx, db, 1, "hw-001"))

		_, err = db.ExecContext(ctx, migrationSQL(t, vendorContractUniqueVersion))
		require.Error(t, err, "duplicate (tenant_id, code) rows must abort the migration")
		require.Contains(t, err.Error(), "deduplicate")
		require.NotContains(t, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "vendors", "code"), "vendor_tenant_id_code",
			"the failing migration must leave no half-applied index set")
	})

	t.Run("contracts", func(t *testing.T) {
		ctx, db := newUserUniqueTestSchema(t)
		schema := schemaOfTestDB(t, ctx, db)

		_, err := db.ExecContext(ctx, legacyContractsDDL)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `DROP INDEX contracts_contract_number_key`)
		require.NoError(t, err)
		require.NoError(t, insertContractPG(t, ctx, db, 1, "HT-2026-001"))
		require.NoError(t, insertContractPG(t, ctx, db, 1, "HT-2026-001"))

		_, err = db.ExecContext(ctx, migrationSQL(t, vendorContractUniqueVersion))
		require.Error(t, err, "duplicate (tenant_id, contract_number) rows must abort the migration")
		require.Contains(t, err.Error(), "deduplicate")
		require.NotContains(t, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "contracts", "contract_number"), "contract_tenant_id_contract_number",
			"the failing migration must leave no half-applied index set")
	})
}

// TestVendorContractUniqueMigration_MatchesGeneratedEntSchema 防止两套真相：
// 自动建表与版本化迁移必须收敛到同一个索引名，否则每次升级都会多建或漏建索引。
func TestVendorContractUniqueMigration_MatchesGeneratedEntSchema(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Schema.Create(ctx))

	require.Equal(t, []string{"vendor_tenant_id_code"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "vendors", "code"),
		"ent auto-migrate must install the tenant-scoped vendor index")
	require.Equal(t, []string{"contract_tenant_id_contract_number"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "contracts", "contract_number"),
		"ent auto-migrate must install the tenant-scoped contract index")

	script := migrationSQL(t, vendorContractUniqueVersion)
	for round := 1; round <= 2; round++ {
		_, err := db.ExecContext(ctx, script)
		require.NoErrorf(t, err, "migration must be idempotent on an ent-managed database (round %d)", round)
	}
	require.Equal(t, []string{"vendor_tenant_id_code"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "vendors", "code"))
	require.Equal(t, []string{"contract_tenant_id_contract_number"}, uniqueIndexesOnColumnInPG(t, ctx, db, schema, "contracts", "contract_number"))
}
