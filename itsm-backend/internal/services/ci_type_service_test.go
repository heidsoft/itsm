package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/enttest"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ciTypeDeleteFixture 搭建「一个类型 + 若干引用它的 CI」的最小场景。
// SQLite 打开了 _fk=1，因此配置项外键会被真实强制——这正是本组测试要覆盖的路径。
//
// 关键：必须挂上生产的读拦截器（database.RegisterSecurityInterceptors），
// 否则 enttest 裸 client 能看到 scrapped 行，测试就锁不住「rawDB 不可省略」这一点——
// 去掉 rawDB 分支后测试仍然全绿，等于没有守住缺陷。
type ciTypeDeleteFixture struct {
	dsn      string
	client   *ent.Client
	ctx      context.Context
	tenantID int
	typeID   int
}

func newCITypeDeleteFixture(t *testing.T) *ciTypeDeleteFixture {
	t.Helper()
	dsn := fmt.Sprintf("file:ci_type_delete_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	// 与生产同口径：ConfigurationItemQuery 自动附加 lifecycle_status != 'scrapped'。
	database.RegisterSecurityInterceptors(client, "off")

	ctx := context.Background()
	tenant := client.Tenant.Create().
		SetName("类型删除口径").SetCode("ci-type-delete").
		SetDomain("ci-type-delete.example.com").SetStatus("active").SaveX(ctx)
	ciType := client.CIType.Create().
		SetName("server").SetDescription("服务器").SetTenantID(tenant.ID).SaveX(ctx)

	return &ciTypeDeleteFixture{
		dsn: dsn, client: client, ctx: ctx, tenantID: tenant.ID, typeID: ciType.ID,
	}
}

func (f *ciTypeDeleteFixture) newService() *CITypeService {
	return NewCITypeService(f.client, zap.NewNop().Sugar())
}

// newServiceWithRawDB 注入真实连接。rawDB 分支与外键兜底是两个独立防线，
// 不注入连接时测试只会走到兜底那条，rawDB 分支等于没被测到。
func (f *ciTypeDeleteFixture) newServiceWithRawDB(t *testing.T) *CITypeService {
	t.Helper()
	db, err := sql.Open("sqlite3", f.dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc := NewCITypeService(f.client, zap.NewNop().Sugar())
	svc.SetRawDB(db)
	return svc
}

// addCI 造一条引用该类型的配置项；scrapped=true 表示已退役（仍物理占用外键）。
func (f *ciTypeDeleteFixture) addCI(t *testing.T, name string, scrapped bool) {
	t.Helper()
	builder := f.client.ConfigurationItem.Create().
		SetName(name).
		SetCiNumber("CI-" + name).
		SetCiTypeID(f.typeID).SetCiType("server").
		SetStatus("active").
		SetTenantID(f.tenantID)
	if scrapped {
		builder.SetLifecycleStatus(common.CILifecycleStatusScrapped).SetStatus("retired")
	} else {
		builder.SetLifecycleStatus(common.CILifecycleStatusOnline)
	}
	_, err := builder.Save(f.ctx)
	require.NoError(t, err)
}

// 退役 CI 仍物理占用 ci_type_id（Required 外键不可置空）：
// 删除类型必须被拦成 409，而不是放行后撞 23503 变成 500。
func TestCITypeService_DeleteCIType_RejectsScrappedReference(t *testing.T) {
	f := newCITypeDeleteFixture(t)
	f.addCI(t, "retired-01", true)

	err := f.newServiceWithRawDB(t).DeleteCIType(f.ctx, f.typeID, f.tenantID)
	require.Error(t, err, "退役 CI 仍占用外键，删除类型必须被拒绝")

	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可识别的业务错误，否则前端只会看到 500")
	require.Equal(t, common.ConflictCode, bizErr.Code, "类型被占用的结果是可预期冲突，不是服务故障")
	require.Contains(t, bizErr.Message, "引用", "文案要点明阻塞原因")
	require.Contains(t, bizErr.Message, "1", "数量应进文案，便于运维判断清理范围")

	// 类型必须仍在
	exist, err := f.client.CIType.Query().Where().Exist(f.ctx)
	require.NoError(t, err)
	require.True(t, exist)
}

// 未注入原生连接时退化为 Ent 查询（看不见退役行），此时必须由外键兜底
// 给出 409——这是第二条独立防线，不能因为修了预检就让它失效。
func TestCITypeService_DeleteCIType_FallsBackToForeignKeyGuard(t *testing.T) {
	f := newCITypeDeleteFixture(t)
	f.addCI(t, "retired-02", true)

	err := f.newService().DeleteCIType(f.ctx, f.typeID, f.tenantID)
	require.Error(t, err, "退役 CI 占用外键，裸 Ent 预检看不到它，仍必须被拦住")

	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr),
		"外键冲突必须归类为业务错误，否则并发写入会把它暴露成 500")
	require.Equal(t, common.ConflictCode, bizErr.Code)

	// 类型必须仍在（兜底拦截不能造成半删）
	count, err := f.client.CIType.Query().Count(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "被拦截的删除不得留下半删状态")
}

// 在线 CI 引用同样拦截，且数量进文案。
func TestCITypeService_DeleteCIType_RejectsOnlineReference(t *testing.T) {
	f := newCITypeDeleteFixture(t)
	f.addCI(t, "online-01", false)

	err := f.newService().DeleteCIType(f.ctx, f.typeID, f.tenantID)
	require.Error(t, err)

	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr))
	require.Equal(t, common.ConflictCode, bizErr.Code)
}

// 无任何引用时必须真删掉——防止修复把「一律拒绝」写死。
func TestCITypeService_DeleteCIType_SucceedsWithoutReference(t *testing.T) {
	f := newCITypeDeleteFixture(t)

	err := f.newService().DeleteCIType(f.ctx, f.typeID, f.tenantID)
	require.NoError(t, err, "无引用时应当可删")

	remain, err := f.client.CIType.Query().Count(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 0, remain)
}

// 分类器必须能认出驱动层外键报错，否则并发写入的兜底会降级成 500。
func TestIsForeignKeyViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"postgres 23503", errors.New("pq: update or delete on table \"ci_types\" violates foreign key constraint (23503)"), true},
		{"postgres 文案", errors.New("pq: update or delete on table violates foreign key constraint \"configuration_items_ci_types_cis\""), true},
		{"sqlite", errors.New("FOREIGN KEY constraint failed"), true},
		{"mysql 1451", errors.New("Error 1451: Cannot delete or update a parent row"), true},
		{"唯一约束不算外键", errors.New("UNIQUE constraint failed: ci_types.name"), false},
		{"nil", nil, false},
		{"其他错误", errors.New("connection refused"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isForeignKeyViolation(tc.err))
		})
	}
}

func TestCITypeService_ListCITypes_CanonicalTypesFirst(t *testing.T) {
	f := newCITypeDeleteFixture(t)
	// 先在夹具里已有 server；再造 3 个「后建」的非规范类型，
	// 它们 created_at 更新，旧排序会把它们排到首屏。
	for _, name := range []string{"tmp-probe-a", "tmp-probe-b", "tmp-probe-c"} {
		_, err := f.client.CIType.Create().SetName(name).SetTenantID(f.tenantID).Save(f.ctx)
		require.NoError(t, err)
	}
	// 再造 1 个后建的规范类型（database），它也必须排在非规范类型之前
	_, err := f.client.CIType.Create().SetName("database").SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)

	svc := f.newService()
	resp, err := svc.ListCITypes(f.ctx, f.tenantID, 1, 100, "")
	require.NoError(t, err)

	names := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		names = append(names, item.Name)
	}

	// 规范类型必须整体排在非规范类型之前
	firstNonCanonical := len(names)
	for i, n := range names {
		if n == "tmp-probe-a" || n == "tmp-probe-b" || n == "tmp-probe-c" {
			if i < firstNonCanonical {
				firstNonCanonical = i
			}
		}
	}
	for i, n := range names {
		if n == "server" || n == "database" {
			require.Less(t, i, firstNonCanonical,
				"规范类型 %q 排在第 %d 位，却在非规范类型（第 %d 位）之后：%v", n, i, firstNonCanonical, names)
		}
	}
}

// 方言占位符断言：sqlite 单测跑通不代表 Postgres 跑得通。
//
// 曾的缺陷：CASE 表达式硬编码 "?"，sqlite 全绿，真机 Postgres 却报
// `pq: syntax error at or near "THEN" (42601)`，类型列表接口直接 500。
// 这里直接断言**生成的 SQL 文本**，不依赖「能不能跑」。
func TestWriteCanonicalOrderCase_DialectPlaceholders(t *testing.T) {
	pg := entsql.Dialect("postgres").String(func(b *entsql.Builder) {
		writeCanonicalOrderCase(b, citype.Table, citype.FieldName)
	})
	// Postgres 用 $n 占位符，且不得出现裸 "?"
	require.NotContains(t, pg, "WHEN ?", "Postgres 不接受 ? 占位符，会报 42601 语法错误")
	require.Contains(t, pg, "WHEN $1 THEN 0", "首个规范 code 应是 $1 且权重 0")
	require.Contains(t, pg, fmt.Sprintf("WHEN $%d THEN %d", len(canonicalCITypeCodes), len(canonicalCITypeCodes)-1))
	// 🔴 标识符必须换成双引号。反引号在 Postgres 里不是合法标识符分隔符，
	// 真机曾因此在 CASE 处报 `syntax error at or near "THEN" (42601)`。
	require.NotContains(t, pg, "`", "Postgres 不接受反引号标识符：%s", pg)
	require.Contains(t, pg, `CASE "ci_types"."name"`, "Postgres 表列名须用双引号：%s", pg)

	sq := entsql.Dialect("sqlite3").String(func(b *entsql.Builder) {
		writeCanonicalOrderCase(b, citype.Table, citype.FieldName)
	})
	require.Contains(t, sq, "WHEN ? THEN 0", "SQLite 保持 ? 占位符")
	require.NotContains(t, sq, "$1", "SQLite 不认 $n 占位符")
	require.Contains(t, sq, "CASE `ci_types`.`name`", "SQLite 表列名用反引号：%s", sq)
}

// 🔴 占位符与参数列表必须一一对应 —— 这才是真机上翻车的那一层。
//
// 缺陷史：第一版硬编码 "?"（Postgres 42601 语法错）；第二版改回
// OrderExprFunc / Dialect(d).Expr，两者内部都是 `Expr(d.String(f))`，
// 把回调渲染成**字符串**后包进无参 expr，占位符留在 SQL 里而 args 全丢，
// 真机报 `not enough args to execute query: want 9 got 1`。
//
// 所以不能只断言 SQL 文本，必须断言 `Query()` 返回的 args 长度与占位符数目一致。
func TestCiteTypeOrderCanonicalFirst_ArgsMatchPlaceholders(t *testing.T) {
	const preWhereArgs = 1 // 模拟 WHERE tenant_id = $1

	for _, tc := range []struct {
		dialect    string
		firstParam string
		caseHead   string
	}{
		{"postgres", fmt.Sprintf("WHEN $%d THEN 0", preWhereArgs+1), `CASE "ci_types"."name"`},
		{"sqlite3", "WHEN ? THEN 0", "CASE `ci_types`.`name`"},
	} {
		t.Run(tc.dialect, func(t *testing.T) {
			sel := entsql.Select("name").From(entsql.Table("ci_types"))
			sel.SetDialect(tc.dialect)
			// 模拟真实的 WHERE tenant_id = ? 先占用 1 个参数位
			sel.Where(entsql.EQ("tenant_id", 7))

			citeTypeOrderCanonicalFirst()(sel)

			q, args := sel.Query()
			require.NoError(t, sel.Err())
			require.Contains(t, q, tc.firstParam,
				"ORDER BY 的第一个占位符必须接在 WHERE 之后，实际 SQL=%s", q)
			// 标识符引号也必须在真实路径上被锁住：order option 执行时 Selector 的
			// SelectTable 还没灌入方言，selector.C() 会退回反引号，Postgres 直接 42601。
			require.Contains(t, q, tc.caseHead,
				"CASE 的表列名引号不对：%s", q)
			// 1 个 WHERE + 8 个 code
			require.Len(t, args, preWhereArgs+len(canonicalCITypeCodes),
				"占位符与参数数目不匹配说明 args 被丢弃：SQL=%s args=%v", q, args)
			for i, code := range canonicalCITypeCodes {
				require.Equal(t, code, args[preWhereArgs+i],
					"第 %d 个 ORDER BY 参数应是 %q，实际 %v", i+1, code, args[preWhereArgs+i])
			}
		})
	}
}

// 权重单调递增：保证规范类型按白名单顺序排在前面，非规范类型统一落最后。
func TestCanonicalCITypeCodes_WeightsAreContiguous(t *testing.T) {
	require.NotEmpty(t, canonicalCITypeCodes)
	seen := map[string]bool{}
	for _, code := range canonicalCITypeCodes {
		require.False(t, seen[code], "白名单里 %q 重复，CASE 分支权重会错位", code)
		seen[code] = true
	}
	// 白名单长度即「非规范类型」的 ELSE 权重；必须与内置种子类型一一对应，
	// 少一个就会让某个规范类型被当成非规范排在后面。
	require.Len(t, canonicalCITypeCodes, 8)
}
