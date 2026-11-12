package service

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitemhistory"
	"itsm-backend/ent/enttest"
)

// 回归：CI 历史快照曾按 Ent 列名写入 before/after，并在驼峰转下划线时把连续
// 大写缩写拆坏（ID→i_d、CINumber→c_i_number）。该快照原样出现在
// GET /cis/:id/history 的响应里，等于把持久化列名泄漏成 API 字段，
// 调用方无法映射回 dto.CIResponse；同时回滚逻辑按 after["ci_type_id"].(int)
// 读取，而 JSON 字段反序列化后是 float64，整数字段被静默跳过。

type ciHistoryFixture struct {
	client   *ent.Client
	history  *CIHistoryService
	ctx      context.Context
	tenantID int
	ciTypeID int
}

func newCIHistoryFixture(t *testing.T) ciHistoryFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("CMDB 历史快照").
		SetCode("cmdb-history-snapshot").
		SetDomain("cmdb-history-snapshot.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	return ciHistoryFixture{
		client:   client,
		history:  NewCIHistoryService(client, zaptest.NewLogger(t).Sugar()),
		ctx:      ctx,
		tenantID: tenant.ID,
		ciTypeID: ciType.ID,
	}
}

func (f ciHistoryFixture) createCI(t *testing.T, name, assetTag string) *ent.ConfigurationItem {
	t.Helper()
	ci, err := f.client.ConfigurationItem.Create().
		SetName(name).
		SetCiTypeID(f.ciTypeID).
		SetCiType("Server").
		SetStatus("active").
		SetAssetTag(assetTag).
		SetCloudAccountID("acct-123").
		SetTenantID(f.tenantID).
		Save(f.ctx)
	require.NoError(t, err)
	return ci
}

func (f ciHistoryFixture) histories(t *testing.T, ciID int) []*ent.ConfigurationItemHistory {
	t.Helper()
	rows, err := f.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.TenantIDEQ(f.tenantID),
		).
		Order(ent.Asc(configurationitemhistory.FieldVersion)).
		All(f.ctx)
	require.NoError(t, err)
	return rows
}

func requireCamelCaseKeys(t *testing.T, snapshot map[string]interface{}, context string) {
	t.Helper()
	for key := range snapshot {
		require.NotContains(t, key, "_", "%s 出现持久化列名键 %q: %v", context, key, snapshot)
	}
}

func TestCIHistorySnapshot_WritesCamelCaseContractKeys(t *testing.T) {
	f := newCIHistoryFixture(t)
	ci := f.createCI(t, "web-01", "asset-1")

	require.NoError(t, f.history.RecordCIHistory(f.ctx, ci.ID, f.tenantID, 7, "tester", "create", "", nil, ci))

	rows := f.histories(t, ci.ID)
	require.Len(t, rows, 1)
	after := rows[0].After

	requireCamelCaseKeys(t, after, "after 快照")
	require.Equal(t, "web-01", after["name"])
	// DTO 契约键：type / ciTypeId / assetTag / cloudAccountId
	require.Contains(t, after, "type")
	require.Contains(t, after, "assetTag")
	require.Contains(t, after, "cloudAccountId")
	require.NotContains(t, after, "i_d")
	require.NotContains(t, after, "ci_type_i_d")
	require.NotContains(t, after, "ci_type_id")

	for _, field := range rows[0].ChangedFields {
		require.NotContains(t, field, "_", "changedFields 不得含列名: %v", rows[0].ChangedFields)
	}
	require.Contains(t, rows[0].ChangedFields, "assetTag")
}

func TestCIHistoryRead_NormalizesLegacySnapshotKeys(t *testing.T) {
	f := newCIHistoryFixture(t)
	ci := f.createCI(t, "legacy-01", "legacy-tag")

	// 模拟 2026-10 之前写入的存量快照：列名键 + 被拆坏的缩写。
	_, err := f.client.ConfigurationItemHistory.Create().
		SetCiID(ci.ID).
		SetVersion(1).
		SetOperation("update").
		SetBefore(map[string]interface{}{
			"i_d":         ci.ID,
			"name":        "legacy-old",
			"asset_tag":   "legacy-old-tag",
			"ci_type_i_d": f.ciTypeID,
		}).
		SetAfter(map[string]interface{}{
			"i_d":       ci.ID,
			"name":      "legacy-new",
			"asset_tag": "legacy-new-tag",
			"ci_type":   "Server",
		}).
		SetChangedFields([]string{"asset_tag", "i_d"}).
		SetOperatorID(1).
		SetOperatorName("tester").
		SetTenantID(f.tenantID).
		Save(f.ctx)
	require.NoError(t, err)

	resp, err := f.history.GetCIHistory(f.ctx, ci.ID, f.tenantID, 1, 20)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)

	item := resp.Items[0]
	requireCamelCaseKeys(t, item.Before, "before 响应")
	requireCamelCaseKeys(t, item.After, "after 响应")
	require.Equal(t, "legacy-old-tag", item.Before["assetTag"])
	require.Equal(t, "legacy-new-tag", item.After["assetTag"])
	require.Equal(t, "Server", item.After["type"])
	require.EqualValues(t, f.ciTypeID, item.Before["ciTypeId"])
	require.Equal(t, []string{"assetTag", "id"}, item.ChangedFields)
}

func TestRevertCIVersion_RestoresFieldsFromSnapshot(t *testing.T) {
	f := newCIHistoryFixture(t)
	original := f.createCI(t, "revert-v1", "tag-v1")
	require.NoError(t, f.history.RecordCIHistory(f.ctx, original.ID, f.tenantID, 7, "tester", "create", "", nil, original))

	updated, err := f.client.ConfigurationItem.UpdateOneID(original.ID).
		SetName("revert-v2").
		SetAssetTag("tag-v2").
		Save(f.ctx)
	require.NoError(t, err)
	require.NoError(t, f.history.RecordCIHistory(f.ctx, original.ID, f.tenantID, 7, "tester", "update", "", original, updated))

	reverted, err := f.history.RevertCIVersion(f.ctx, original.ID, f.tenantID, 7, "tester",
		&dto.RevertCIVersionRequest{Version: 1, Remark: "回归测试"})
	require.NoError(t, err)
	require.Equal(t, "revert-v1", reverted.Name)
	require.Equal(t, "tag-v1", reverted.AssetTag)

	stored, err := f.client.ConfigurationItem.Get(f.ctx, original.ID)
	require.NoError(t, err)
	require.Equal(t, "revert-v1", stored.Name)
	require.Equal(t, "tag-v1", stored.AssetTag)
	// 整数字段必须真正还原：旧实现按 after["ci_type_id"].(int) 读取，
	// JSON 反序列化后是 float64，断言类型永远失败、字段被静默跳过。
	require.Equal(t, f.ciTypeID, stored.CiTypeID)
	require.True(t, strings.HasPrefix(stored.Name, "revert-v1"))
}
