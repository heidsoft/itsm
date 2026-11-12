package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
)

// release 的关联变更/负责人引用必须同租户，防跨租户引用注入。
func TestCreateRelease_RejectsCrossTenantChangeReference(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant1, err := client.Tenant.Create().
		SetName("R1").SetCode("rel-r1").SetDomain("rel-r1.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().
		SetName("R2").SetCode("rel-r2").SetDomain("rel-r2.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().
		SetUsername("rel-actor").SetEmail("rel-actor@test.com").SetName("发布人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)
	foreignOwner, err := client.User.Create().
		SetUsername("rel-other").SetEmail("rel-other@test.com").SetName("他租户负责人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	foreignChange, err := client.Change.Create().
		SetTitle("他租户变更").SetChangeNumber("CHG-REL-OTHER").SetStatus("draft").
		SetDescription("d").SetCreatedBy(foreignOwner.ID).SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewReleaseService(client, logger)

	foreignID := foreignChange.ID
	_, err = svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title:    "跨租户引用回归",
		Type:     "normal",
		ChangeID: &foreignID,
	}, actor.ID, tenant1.ID)
	require.ErrorContains(t, err, "关联变更不存在或不属于当前租户")

	count, err := client.Release.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestCreateRelease_RejectsCrossTenantOwnerReference(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant1, err := client.Tenant.Create().
		SetName("O1").SetCode("rel-o1").SetDomain("rel-o1.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().
		SetName("O2").SetCode("rel-o2").SetDomain("rel-o2.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().
		SetUsername("own-actor").SetEmail("own-actor@test.com").SetName("发布人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)
	foreignUser, err := client.User.Create().
		SetUsername("own-other").SetEmail("own-other@test.com").SetName("他租户用户").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewReleaseService(client, logger)

	foreignID := foreignUser.ID
	_, err = svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title:   "跨租户负责人回归",
		Type:    "normal",
		OwnerID: &foreignID,
	}, actor.ID, tenant1.ID)
	require.ErrorContains(t, err, "负责人不存在或不属于当前租户")

	// 同租户负责人必须放行，证明谓词不是无差别拒绝。
	ownID := actor.ID
	rel, err := svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title:   "同租户负责人",
		Type:    "normal",
		OwnerID: &ownID,
	}, actor.ID, tenant1.ID)
	require.NoError(t, err)
	require.NotNil(t, rel)
}
