package incident

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/incident"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件锁住事件写路径的四个契约，全部为「修复前失败、修复后通过」的回归测试：
//
//  1. NULL 语义保留：resolved_at / closed_at / assignee_id 在 ent 中是 Optional
//     非 Nillable，DB NULL 会读成 Go 零值。toDomain 若直接取地址，就会伪造出
//     「指向 0001-01-01 的非 nil 指针」，随后被写路径当成有效值落库。
//  2. 读-改-写不得污染时间列：一次只改标题的 Update 不能让 NULL 变成 0001-01-01。
//  3. 乐观锁：Update 必须按 id + tenant_id + version 条件更新并自增 version，
//     陈旧快照返回 ErrStaleVersion，而不是静默覆盖。
//  4. 分类字段往返：impact / urgency / version / is_major_incident 必须能从
//     仓储读出，Create 留空时让 schema Default 生效、显式给值时原样保留。

func newEntTestRepo(t *testing.T) (*EntRepository, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:incident_update_integrity?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	return NewEntRepository(client), client
}

// seedRawIncident 绕过仓储直接用 ent 建行，以便精确控制哪些列保持 NULL。
func seedRawIncident(t *testing.T, client *ent.Client, number string, tenantID int) *ent.Incident {
	t.Helper()
	row, err := client.Incident.Create().
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber(number).
		SetReporterID(7).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return row
}

// TestEntRepository_ToDomain_PreservesNullSemantics 验证 NULL 列映射为 nil 指针。
//
// 修复前：toDomain 用 ResolvedAt: &e.ResolvedAt / AssigneeID: &e.AssigneeID，
// DB NULL 被读成零值后取地址，得到指向 0001-01-01 与 0 的非 nil 指针，
// 「未解决」与「公元 1 年解决」、「未分配」与「分配给用户 0」不可区分。
func TestEntRepository_ToDomain_PreservesNullSemantics(t *testing.T) {
	repo, client := newEntTestRepo(t)
	ctx := context.Background()

	row := seedRawIncident(t, client, "INC-NULL-0001", 1)

	got, err := repo.Get(ctx, row.ID, 1)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Nil(t, got.ResolvedAt, "resolved_at 为 NULL 时必须映射成 nil，而不是指向零值时间的指针")
	assert.Nil(t, got.ClosedAt, "closed_at 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.EscalatedAt, "escalated_at 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.AssigneeID, "assignee_id 为 NULL/0 时必须映射成 nil（未分配）")
	assert.Nil(t, got.ConfigurationItemID, "configuration_item_id 未设置时必须映射成 nil")
	assert.Nil(t, got.SLADefinitionID, "sla_definition_id 未设置时必须映射成 nil")
	assert.Nil(t, got.SLAResponseDeadline, "sla_response_deadline 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.SLAResolutionDeadline, "sla_resolution_deadline 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.SLAFirstResponseAt, "sla_first_response_at 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.SLAResolvedAt, "sla_resolved_at 为 NULL 时必须映射成 nil")
	assert.Nil(t, got.SLAPausedAt, "sla_paused_at 为 NULL 时必须映射成 nil")
}

// TestEntRepository_Update_DoesNotCorruptNullTimes 是不可逆数据污染的端到端回归。
//
// 修复前：Get 把 NULL 读成零值指针 → 只改标题 → Update 的 `if i.ResolvedAt != nil`
// 判定通过 → SetResolvedAt(0001-01-01) 落库。此后该事件在 MTTR / SLA / 报表中
// 永远被当成「公元 1 年已解决」，且无法从数据本身区分是真实值还是污染值。
func TestEntRepository_Update_DoesNotCorruptNullTimes(t *testing.T) {
	repo, client := newEntTestRepo(t)
	ctx := context.Background()

	row := seedRawIncident(t, client, "INC-CORRUPT-0001", 1)

	current, err := repo.Get(ctx, row.ID, 1)
	require.NoError(t, err)

	current.Title = "只改标题"
	updated, err := repo.Update(ctx, current)
	require.NoError(t, err)
	assert.Equal(t, "只改标题", updated.Title)

	// 直接查 ent 行，避免经过被测的 toDomain 自证清白。
	raw, err := client.Incident.Get(ctx, row.ID)
	require.NoError(t, err)
	assert.Equal(t, "只改标题", raw.Title, "标题必须真的写进去")
	assert.True(t, raw.ResolvedAt.IsZero(),
		"P1 验收：一次只改标题的 Update 不得把 resolved_at 从 NULL 污染成 0001-01-01")
	assert.True(t, raw.ClosedAt.IsZero(),
		"P1 验收：一次只改标题的 Update 不得把 closed_at 从 NULL 污染成 0001-01-01")
	assert.True(t, raw.EscalatedAt.IsZero(),
		"P1 验收：escalated_at 同样不得被污染")

	// 污染值一旦落库就会通过 API 泄漏给前端，这里锁住出口。
	afterGet, err := repo.Get(ctx, row.ID, 1)
	require.NoError(t, err)
	assert.Nil(t, afterGet.ResolvedAt)
	assert.Nil(t, afterGet.ClosedAt)
}

// TestEntRepository_Update_KeepsRealTimestamps 防止上面的修复矫枉过正：
// 真实已解决/已关闭的时间戳必须原样保留，不能被 optionalTime 吞掉。
func TestEntRepository_Update_KeepsRealTimestamps(t *testing.T) {
	repo, client := newEntTestRepo(t)
	ctx := context.Background()

	row := seedRawIncident(t, client, "INC-KEEP-0001", 1)
	resolvedAt := time.Date(2026, 9, 12, 8, 30, 0, 0, time.UTC)
	_, err := client.Incident.UpdateOneID(row.ID).
		SetStatus("resolved").
		SetResolvedAt(resolvedAt).
		Save(ctx)
	require.NoError(t, err)

	current, err := repo.Get(ctx, row.ID, 1)
	require.NoError(t, err)
	require.NotNil(t, current.ResolvedAt, "真实 resolved_at 必须读成非 nil")
	assert.True(t, current.ResolvedAt.Equal(resolvedAt), "读出的时间必须与落库值一致")

	current.Title = "改标题但保留解决时间"
	_, err = repo.Update(ctx, current)
	require.NoError(t, err)

	raw, err := client.Incident.Get(ctx, row.ID)
	require.NoError(t, err)
	assert.True(t, raw.ResolvedAt.Equal(resolvedAt),
		"Update 不得丢失已有的 resolved_at")
}

// TestEntRepository_Update_VersionCAS 验证条件更新与版本自增。
//
// 修复前：Update 是 read-then-unconditional-write，VersionEQ 从未参与条件，
// version 也从不自增；陈旧客户端可以静默覆盖他人写入，且 DTO 的 version
// 字段是死契约。
func TestEntRepository_Update_VersionCAS(t *testing.T) {
	repo, client := newEntTestRepo(t)
	ctx := context.Background()

	row := seedRawIncident(t, client, "INC-CAS-0001", 1)
	require.Equal(t, 1, row.Version, "schema Default(1) 应使新行版本为 1")

	t.Run("命中当前版本：写入成功且 version 自增", func(t *testing.T) {
		current, err := repo.Get(ctx, row.ID, 1)
		require.NoError(t, err)
		require.Equal(t, 1, current.Version)

		current.Title = "第一次更新"
		updated, err := repo.Update(ctx, current)
		require.NoError(t, err)
		assert.Equal(t, 2, updated.Version, "Update 必须自增 version，否则乐观锁形同虚设")

		raw, err := client.Incident.Get(ctx, row.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, raw.Version, "version 自增必须真的落库")
		assert.Equal(t, "第一次更新", raw.Title)
	})

	t.Run("陈旧版本：返回 ErrStaleVersion 且不覆盖数据", func(t *testing.T) {
		stale, err := repo.Get(ctx, row.ID, 1)
		require.NoError(t, err)
		stale.Version = 1 // 模拟客户端持有过期快照
		stale.Title = "陈旧写入不应生效"

		_, err = repo.Update(ctx, stale)
		require.Error(t, err, "陈旧版本必须失败，不得静默覆盖")
		assert.ErrorIs(t, err, ErrStaleVersion,
			"P1 验收：条件更新未命中必须返回 ErrStaleVersion，供 service 映射成 409")

		raw, err := client.Incident.Get(ctx, row.ID)
		require.NoError(t, err)
		assert.Equal(t, "第一次更新", raw.Title, "陈旧写入不得改动数据")
		assert.Equal(t, 2, raw.Version, "失败路径不得自增 version")
	})

	t.Run("跨租户写入 fail closed", func(t *testing.T) {
		other, err := repo.Get(ctx, row.ID, 1)
		require.NoError(t, err)
		other.TenantID = 2 // 越权租户
		other.Title = "跨租户写入不应生效"

		_, err = repo.Update(ctx, other)
		require.Error(t, err, "跨租户 Update 必须失败")

		raw, err := client.Incident.Get(ctx, row.ID)
		require.NoError(t, err)
		assert.Equal(t, "第一次更新", raw.Title, "跨租户写入不得改动数据")
		assert.Equal(t, 1, raw.TenantID, "租户归属不得被改写")
	})
}

// TestEntRepository_Create_ImpactUrgencyGuard 验证 schema 校验与默认值的边界。
//
// ent 对 impact / urgency 的 Validate 拒绝空串，因此留空时必须不写该列，
// 让 Default("medium") 生效；显式给值时必须原样保留（修复前 handler 从不
// 传 impact/urgency，用户填写的影响度/紧急度被静默丢弃）。
func TestEntRepository_Create_ImpactUrgencyGuard(t *testing.T) {
	repo, client := newEntTestRepo(t)
	ctx := context.Background()

	t.Run("留空时让 Default(medium) 生效，不得触发校验失败", func(t *testing.T) {
		created, err := repo.Create(ctx, &Incident{
			Title:          "未填写影响度",
			Description:    "d",
			Status:         "new",
			Priority:       "medium",
			Severity:       "medium",
			Impact:         "",
			Urgency:        "",
			IncidentNumber: "INC-CREATE-DEFAULT",
			ReporterID:     7,
			TenantID:       1,
			DetectedAt:     time.Now(),
		})
		require.NoError(t, err, "impact/urgency 留空不能让 Create 失败")
		assert.Equal(t, "medium", created.Impact)
		assert.Equal(t, "medium", created.Urgency)
	})

	t.Run("显式给值时必须原样落库并读回", func(t *testing.T) {
		created, err := repo.Create(ctx, &Incident{
			Title:           "填写了影响度",
			Description:     "d",
			Status:          "new",
			Priority:        "high",
			Severity:        "high",
			Impact:          "critical",
			Urgency:         "low",
			IncidentNumber:  "INC-CREATE-EXPLICIT",
			ReporterID:      7,
			TenantID:        1,
			IsMajorIncident: true,
			DetectedAt:      time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, "critical", created.Impact, "P2 验收：impact 不得被静默丢弃")
		assert.Equal(t, "low", created.Urgency, "P2 验收：urgency 不得被静默丢弃")

		raw, err := client.Incident.Query().
			Where(incident.IncidentNumberEQ("INC-CREATE-EXPLICIT")).
			Only(ctx)
		require.NoError(t, err)
		assert.Equal(t, "critical", raw.Impact)
		assert.Equal(t, "low", raw.Urgency)

		got, err := repo.Get(ctx, created.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, "critical", got.Impact, "toDomain 必须暴露 impact")
		assert.Equal(t, "low", got.Urgency, "toDomain 必须暴露 urgency")
		assert.Equal(t, 1, got.Version, "toDomain 必须暴露 version，否则前端无法回传乐观锁")
		assert.True(t, got.IsMajorIncident, "toDomain 必须暴露 isMajorIncident")
	})
}
