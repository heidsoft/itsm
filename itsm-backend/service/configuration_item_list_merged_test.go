package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestListCIsMergedFields P1-1 合并验证：
// ListCIRequest 新增 SortBy/SortOrder/TagIDs/DateFrom/DateTo/WithRelations 字段。
// 这些字段必须被 ListCIs 正确处理（不 panic、不被静默忽略）。
func TestListCIsMergedFields(t *testing.T) {
	dsn := testDSN()
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("merged-tenant").SetCode("merged").SetDomain("merged.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	tag, err := client.CITag.Create().SetKey("env").SetValue("prod").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 3 条样本 CI（不同 name/criticality/createdAt）
	now := time.Now()
	ciA := mustCreateCI(t, client, tenant.ID, ciType.ID, []int{tag.ID}, "ci-alpha", "critical", now.Add(-3*time.Hour))
	ciB := mustCreateCI(t, client, tenant.ID, ciType.ID, []int{tag.ID}, "ci-bravo", "low", now.Add(-2*time.Hour))
	ciC := mustCreateCI(t, client, tenant.ID, ciType.ID, []int{tag.ID}, "ci-charlie", "medium", now.Add(-1*time.Hour))

	_ = ciA
	_ = ciB
	_ = ciC

	svc := NewConfigurationItemService(client, logger, nil, nil)

	t.Run("SortBy+SortOrder=name asc", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			SortBy: "name", SortOrder: "asc",
		})
		require.NoError(t, err)
		require.Len(t, resp.Items, 3)
		require.Equal(t, "ci-alpha", resp.Items[0].Name)
		require.Equal(t, "ci-bravo", resp.Items[1].Name)
		require.Equal(t, "ci-charlie", resp.Items[2].Name)
	})

	t.Run("SortBy+SortOrder=criticality desc", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			SortBy: "criticality", SortOrder: "desc",
		})
		require.NoError(t, err)
		require.Len(t, resp.Items, 3)
		// SQLite ORDER BY criticality DESC 按字符串字典序倒序：m(medium) > l(low) > c(critical)
		require.Equal(t, "medium", resp.Items[0].Criticality)
		require.Equal(t, "low", resp.Items[1].Criticality)
		require.Equal(t, "critical", resp.Items[2].Criticality)
	})

	t.Run("DateFrom+DateTo 范围过滤", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			DateFrom: ptrTimeOrNil(now.Add(-2*time.Hour - 30*time.Minute)),
			DateTo:   ptrTimeOrNil(now.Add(-30*time.Minute)),
		})
		require.NoError(t, err)
		// 应只命中 ci-bravo（now-2h 在范围内）和 ci-charlie（now-1h 在范围内）；ci-alpha (now-3h) 被排除
		require.Len(t, resp.Items, 2)
		names := []string{resp.Items[0].Name, resp.Items[1].Name}
		require.Contains(t, names, "ci-bravo")
		require.Contains(t, names, "ci-charlie")
	})

	t.Run("TagIDs 过滤", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			TagIDs: []int{tag.ID},
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Items), 3) // 3 条样本都打了这个标签
	})

	t.Run("WithRelations=true 不 panic", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			WithRelations: true,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Items), 3)
	})

	t.Run("Search 关键词宽模糊命中 name", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			Search: "alpha",
		})
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)
		require.Equal(t, "ci-alpha", resp.Items[0].Name)
	})

	t.Run("Status 半匹配（合并后改为 Contains 语义）", func(t *testing.T) {
		resp, err := svc.ListCIs(ctx, tenant.ID, &dto.ListCIRequest{
			Page: 1, Size: 10,
			Status: "act", // 半匹配
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Items), 3)
	})
}

func mustCreateCI(t *testing.T, client *ent.Client, tenantID, ciTypeID int, tagIDs []int, name, criticality string, createdAt time.Time) *ent.ConfigurationItem {
	t.Helper()
	ctx := context.Background()
	ci, err := client.ConfigurationItem.Create().
		SetName(name).
		SetCiTypeID(ciTypeID).
		SetTenantID(tenantID).
		SetStatus("active").
		SetEnvironment("dev").
		SetCriticality(criticality).
		SetAssetTag("tag-" + name).
		SetSerialNumber(name + "-sn").
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err)
	for _, tid := range tagIDs {
		err := client.ConfigurationItem.UpdateOneID(ci.ID).AddTagIDs(tid).Exec(ctx)
		require.NoError(t, err)
	}
	return ci
}

func ptrTimeOrNil(t time.Time) *time.Time {
	return &t
}