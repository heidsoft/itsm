package change

// 回归：change_affected_cis 是 M2M 关联表，且没有 tenant_id（在
// internal/schema/tenant_guard.go 登记为 derived 豁免）。外键只保证
// 「CI 存在」，不保证「CI 属于本租户」——写入时只做 strconv.Atoi + id>0 校验，
// 于是 A 租户的变更单可以把 B 租户的资产塞进自己的影响面：
//   - 跨租户数据泄漏（B 的资产名/关键度/归属通过影响摘要间接可读）
//   - 影响摘要按 tenant_id 过滤，总数对不上，界面上「关联了却看不到」
//   - 排期冲突检查按 CI 交集判定，跨租户 CI 会产生幽灵冲突
//
// 本文件锁住写入侧的租户归属校验（CreateChange 与 UpdateChange 共用）。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type affectedCIFixture struct {
	svc          *Service
	ctx          context.Context
	tenantAID    int
	tenantBID    int
	ownedCIID    int
	foreignCIID  int
	changeUserID int
}

func newAffectedCIFixture(t *testing.T) affectedCIFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:change_affected_ci_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("租户A").SetCode("affected-ci-a").
		SetDomain("affected-ci-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("租户B").SetCode("affected-ci-b").
		SetDomain("affected-ci-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("affected-ci-u").SetEmail("affected-ci@example.com").
		SetName("U").SetPasswordHash("hash").SetRole("it_admin").SetActive(true).
		SetTenantID(tenantA.ID).SaveX(ctx)

	ciTypeA := client.CIType.Create().SetName("服务器").SetTenantID(tenantA.ID).SaveX(ctx)
	ciTypeB := client.CIType.Create().SetName("服务器").SetTenantID(tenantB.ID).SaveX(ctx)

	owned := client.ConfigurationItem.Create().SetName("A的服务器").SetCiType("server").
		SetCiTypeID(ciTypeA.ID).SetStatus("in_service").SetEnvironment("production").
		SetTenantID(tenantA.ID).SaveX(ctx)
	foreign := client.ConfigurationItem.Create().SetName("B的服务器").SetCiType("server").
		SetCiTypeID(ciTypeB.ID).SetStatus("in_service").SetEnvironment("production").
		SetTenantID(tenantB.ID).SaveX(ctx)

	repo := NewEntRepository(client, nil)
	svc := newTestService(repo, client, zaptest.NewLogger(t).Sugar(), nil)

	return affectedCIFixture{
		svc: svc, ctx: ctx,
		tenantAID: tenantA.ID, tenantBID: tenantB.ID,
		ownedCIID: owned.ID, foreignCIID: foreign.ID, changeUserID: userA.ID,
	}
}

func (f affectedCIFixture) newChange(title string) *Change {
	return &Change{
		Title: title, Description: "d", Type: "normal", Status: "draft",
		Priority: "medium", ImpactScope: "medium", RiskLevel: "low",
		CreatedBy: f.changeUserID, TenantID: f.tenantAID,
		ImplementationPlan: "step1", RollbackPlan: "back1",
	}
}

// 核心回归：把别的租户的 CI 塞进受影响列表必须被拒，且不落库。
func TestCreateChange_RejectsForeignTenantAffectedCI(t *testing.T) {
	f := newAffectedCIFixture(t)

	c := f.newChange("跨租户CI变更")
	c.AffectedCIs = []string{fmt.Sprint(f.ownedCIID), fmt.Sprint(f.foreignCIID)}

	_, err := f.svc.CreateChange(f.ctx, c)
	require.Error(t, err, "含跨租户 CI 的变更不得创建成功")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "必须是分类过的 AppError（否则 500）")
	assert.Equal(t, common.ErrCodeValidation, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus, "参数/权限问题必须是 4xx，不是 500")

	// 文案要能定位到具体 ID，否则调用方无法自查
	assert.Contains(t, appErr.Message, fmt.Sprint(f.foreignCIID),
		"错误文案必须点名越界的 CI ID，实际: %s", appErr.Message)
	assert.NotContains(t, appErr.Message, fmt.Sprintf("%d、%d", f.ownedCIID, f.foreignCIID),
		"本租户的合法 CI 不应被点名为越界")

	// 整单不该落库（失败关闭，而不是「过滤掉非法项后创建」）
	count, err := f.svc.entClient.Change.Query().Count(f.ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "校验失败时不应留下半条变更单")
}

// 本租户的 CI 必须照常放行。
func TestCreateChange_AcceptsOwnTenantAffectedCI(t *testing.T) {
	f := newAffectedCIFixture(t)

	c := f.newChange("本租户CI变更")
	c.AffectedCIs = []string{fmt.Sprint(f.ownedCIID)}

	created, err := f.svc.CreateChange(f.ctx, c)
	require.NoError(t, err)
	require.NotNil(t, created)

	// 领域对象不 eager-load 关系边，落库证据要直接查关联表
	assertAffectedCIs(t, f, created.ID, f.ownedCIID)
}

// 非数字项（标准变更模板里以名称书写）本就无法建关系，必须继续跳过而不是报错。
func TestCreateChange_SkipsNonNumericAffectedCIs(t *testing.T) {
	f := newAffectedCIFixture(t)

	c := f.newChange("含名称项")
	c.AffectedCIs = []string{"服务器", fmt.Sprint(f.ownedCIID)}

	created, err := f.svc.CreateChange(f.ctx, c)
	require.NoError(t, err, "名称项是历史合法输入，不应因本次校验而变红")
	assertAffectedCIs(t, f, created.ID, f.ownedCIID)
}

// assertAffectedCIs 断言变更单的关联表恰好只含 wantCIID（其余项已被跳过）。
func assertAffectedCIs(t *testing.T, f affectedCIFixture, changeID int, wantCIID int) {
	t.Helper()
	rel, err := f.svc.entClient.Change.Get(f.ctx, changeID)
	require.NoError(t, err)
	affected, err := rel.QueryAffectedCis().All(f.ctx)
	require.NoError(t, err)
	require.Len(t, affected, 1, "关联表应只剩 1 条（名称项被跳过）")
	assert.Equal(t, wantCIID, affected[0].ID)
}

// 空列表 / 全名称项：不触发查询，直接放行（纯文档型变更合法）。
func TestCreateChange_NoNumericAffectedCIsPasses(t *testing.T) {
	f := newAffectedCIFixture(t)

	c := f.newChange("无CI变更")
	c.AffectedCIs = nil
	if _, err := f.svc.CreateChange(f.ctx, c); err != nil {
		t.Fatalf("不关联 CI 的变更必须放行，实际: %v", err)
	}

	c2 := f.newChange("只有名称项")
	c2.AffectedCIs = []string{"核心交换机"}
	if _, err := f.svc.CreateChange(f.ctx, c2); err != nil {
		t.Fatalf("纯名称项不得触发归属校验，实际: %v", err)
	}
}

// 更新路径同一条契约：创建时合法，更新时也不能换成别的租户 CI。
func TestUpdateChange_RejectsForeignTenantAffectedCI(t *testing.T) {
	f := newAffectedCIFixture(t)

	c := f.newChange("更新换CI")
	c.AffectedCIs = []string{fmt.Sprint(f.ownedCIID)}
	created, err := f.svc.CreateChange(f.ctx, c)
	require.NoError(t, err)

	// 模拟 handler 的字段合并：把受影响 CI 换成别的租户的
	updated := *created
	updated.AffectedCIs = []string{fmt.Sprint(f.foreignCIID)}

	_, err = f.svc.UpdateChange(f.ctx, &updated, f.changeUserID, "it_admin")
	require.Error(t, err, "更新侧同样不得接受跨租户 CI")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, common.ErrCodeValidation, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)

	// 关联表里不能出现别的租户的 CI
	rel, err := f.svc.entClient.Change.Get(f.ctx, created.ID)
	require.NoError(t, err)
	affected, err := rel.QueryAffectedCis().All(f.ctx)
	require.NoError(t, err)
	for _, ci := range affected {
		assert.Equal(t, f.tenantAID, ci.TenantID, "关联表里不得出现别的租户的 CI: %s", ci.Name)
	}
}
