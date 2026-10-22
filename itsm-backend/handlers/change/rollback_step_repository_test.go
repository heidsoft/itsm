package change

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回滚清单仓储的 SQL 级回归。
//
// 为什么要单独跑一遍真 SQL（而不是只靠 mockRepository）：
//   - 清单是「一行一步」的原生 SQL 子表，SQL 本身写错（占位符、CASE WHEN、
//     条件更新的 OR 分支、租户谓词）在 mock 上完全看不出来；
//   - 「撤销误点」依赖 `OR $1 = 'pending'` 这个分支，它一旦退化成
//     `AND status='pending'`，撤销在物理上就不可能发生，而 mock 会继续绿。
//
// SQLite 只用于验证语句语义与租户边界；生产差异（BIGSERIAL/TIMESTAMPTZ）
// 在 migrations/20261022_change_rollback_steps.sql 里，由 PG 执行。
func setupRollbackStepTest(t *testing.T) (*Service, *ent.Client, *sql.DB, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	// 与迁移文件等价的 SQLite DDL（BIGSERIAL→INTEGER PRIMARY KEY AUTOINCREMENT、
	// TIMESTAMPTZ→DATETIME）。
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE change_rollback_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			change_id INTEGER NOT NULL,
			tenant_id INTEGER NOT NULL,
			step_no INTEGER NOT NULL,
			description TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			note TEXT,
			resolved_by INTEGER,
			resolved_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE UNIQUE INDEX change_rollback_steps_change_step_uidx ON change_rollback_steps (change_id, step_no);
	`)
	require.NoError(t, err)

	tenant, err := client.Tenant.Create().
		SetName("RollbackStepTenant").
		SetCode("RBS" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain("rbs.test").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	actor, err := client.User.Create().
		SetUsername("rbs-actor-" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetEmail("rbs@example.com").
		SetName("RBS Actor").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(context.Background())
	require.NoError(t, err)

	repo := NewEntRepository(client, db)
	svc := NewService(repo, client, zaptest.NewLogger(t).Sugar(), nil)
	return svc, client, db, tenant.ID, actor.ID
}

func createRollbackTestChange(t *testing.T, client *ent.Client, tenantID, actorID int, status string) *ent.Change {
	t.Helper()
	created, err := client.Change.Create().
		SetChangeNumber(fmt.Sprintf("CHG-RBS-%d-%d", tenantID, time.Now().UnixNano())).
		SetTitle("回滚清单测试变更").
		SetDescription("desc").
		SetType("normal").
		SetStatus(change.Status(status)).
		SetPriority("medium").
		SetImpactScope("low").
		SetRiskLevel("high").
		SetCreatedBy(actorID).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return created
}

// TestRollbackStepRepository_SQLRoundTrip 真 SQL 往返：编排 → 读取（有序）→
// 逐项确认 → 再次读取。覆盖 CASE WHEN 与条件更新分支。
func TestRollbackStepRepository_SQLRoundTrip(t *testing.T) {
	svc, client, _, tenantID, actorID := setupRollbackStepTest(t)
	ctx := context.Background()
	c := createRollbackTestChange(t, client, tenantID, actorID, "scheduled")

	steps, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{
		"摘除 SLB 流量",
		"回滚应用镜像到上一版本",
	})
	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, []int{1, 2}, []int{steps[0].StepNo, steps[1].StepNo})
	assert.True(t, allPending(steps))

	// scheduled 允许编排、不允许确认（确认只在 in_progress/failed）。
	_, err = svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "")
	require.Error(t, err, "scheduled 状态下不应允许确认回滚结果")

	started, err := client.Change.UpdateOneID(c.ID).SetStatus(change.StatusInProgress).Save(ctx)
	require.NoError(t, err)
	require.Equal(t, change.StatusInProgress, started.Status)

	first, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "已摘除，流量归零")
	require.NoError(t, err)
	assert.Equal(t, RollbackStepDone, first.Status)
	require.NotNil(t, first.ResolvedBy, "确认人必须落库（审计要答「谁确认的」）")
	assert.Equal(t, actorID, *first.ResolvedBy)
	require.NotNil(t, first.ResolvedAt)
	assert.Equal(t, "已摘除，流量归零", first.Note)

	// 重复确认被条件更新挡住，不能覆盖前一位的备注。
	_, err = svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepFailed, "其实失败了")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRollbackStepAlreadyResolved)

	// 撤销误点：status=pending 走 OR 分支，清空确认人/时间但保留备注。
	reset, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepPending, "复核发现镜像 tag 未变")
	require.NoError(t, err)
	assert.Equal(t, RollbackStepPending, reset.Status)
	assert.Nil(t, reset.ResolvedBy)
	assert.Nil(t, reset.ResolvedAt)
	assert.Equal(t, "复核发现镜像 tag 未变", reset.Note)
}

// TestRollbackStepRepository_TenantIsolation 跨租户不可见、不可改：
// 子表没有 RLS，租户边界只能靠仓储的 (tenant_id, change_id) 双谓词守住。
func TestRollbackStepRepository_TenantIsolation(t *testing.T) {
	svc, client, _, tenantID, actorID := setupRollbackStepTest(t)
	ctx := context.Background()
	c := createRollbackTestChange(t, client, tenantID, actorID, "scheduled")
	steps, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{"摘流量"})
	require.NoError(t, err)

	const otherTenant = 999
	leaked, err := svc.repo.ListRollbackSteps(ctx, c.ID, otherTenant)
	require.NoError(t, err)
	assert.Empty(t, leaked, "别的租户读同一 change_id 必须看不到任何步骤")

	ok, err := svc.repo.ResolveRollbackStep(ctx, steps[0].ID, c.ID, otherTenant, actorID, RollbackStepDone, "越权", time.Now())
	require.NoError(t, err)
	assert.False(t, ok, "别的租户不得改写本租户的步骤")

	stillPending, err := svc.repo.ListRollbackSteps(ctx, c.ID, tenantID)
	require.NoError(t, err)
	require.Len(t, stillPending, 1)
	assert.Equal(t, RollbackStepPending, stillPending[0].Status, "越权写入不得生效")
}

// TestValidateRollbackChecklist_RealRows 门禁读真表：
// 有 pending → 拦；全部确认（含 failed）→ 放行；清单为空 → 放行。
func TestValidateRollbackChecklist_RealRows(t *testing.T) {
	svc, client, _, tenantID, actorID := setupRollbackStepTest(t)
	ctx := context.Background()

	// 1) 无清单：放行（应急回滚不能被「你还没编排清单」挡住）
	empty := createRollbackTestChange(t, client, tenantID, actorID, "in_progress")
	require.NoError(t, svc.validateRollbackChecklist(ctx, empty.ID, tenantID))

	// 2) 有清单且全部 pending：拦，并点出未确认的步骤号
	c := createRollbackTestChange(t, client, tenantID, actorID, "scheduled")
	steps, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{"摘流量", "回滚镜像"})
	require.NoError(t, err)
	err = svc.validateRollbackChecklist(ctx, c.ID, tenantID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRollbackStepsUnresolved)
	assert.Contains(t, err.Error(), "第1步")
	assert.Contains(t, err.Error(), "第2步")

	// 3) 全部确认（一步 done、一步 failed）：放行——失败也是真实结果
	_, err = client.Change.UpdateOneID(c.ID).SetStatus(change.StatusInProgress).Save(ctx)
	require.NoError(t, err)
	_, err = svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "已完成")
	require.NoError(t, err)
	_, err = svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[1].ID, actorID, RollbackStepFailed, "回滚卡住，已人工恢复并复核")
	require.NoError(t, err)
	require.NoError(t, svc.validateRollbackChecklist(ctx, c.ID, tenantID))
}

func allPending(steps []*RollbackStep) bool {
	for _, s := range steps {
		if s.Status != RollbackStepPending {
			return false
		}
	}
	return true
}
