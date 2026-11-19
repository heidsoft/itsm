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
	"itsm-backend/ent/enttest"

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
