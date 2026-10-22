package change

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回滚清单（失败可回退）回归锁。
//
// 背景：链路 C「受控变更」的上线门禁里，「失败可回退」此前只落在一次状态翻转上——
// rolled_back 可以没有任何执行证据。回滚计划是一段 TEXT，提交门禁只校验非空；
// 写了几步、走到第几步、每步成没成，系统一概不知。于是审计里最该被回答的三个
// 问题（回滚到哪了？哪一步没回成？谁确认的？）全部无法回答，PIR 的
// 「影响是否已复原」只能靠人凭记忆填。
//
// 本文件锁的行为：
//  1. 清单存在且还有未确认步骤 → 不许标记 rolled_back（422，且状态不动）；
//  2. 全部步骤已确认（done/skipped/failed 都算确认）→ 放行；
//  3. 没有清单 → 放行（应急回滚不能被"你还没编排清单"挡住）；
//  4. 清单读取失败 → 失败关闭（读不到就不许回滚，回滚是不可逆动作）；
//  5. 实施开始后清单锁定（否则"清空清单再回滚"就能绕过 1）；
//  6. skipped/failed 必须带备注（否则"有意跳过"与"漏做"在审计里无法区分）。

// seedChangeWithStatus 创建一个指定状态的变更（走 mock，不依赖 DB）。
func seedChangeWithStatus(t *testing.T, repo *mockRepository, tenantID, userID int, status string) *Change {
	t.Helper()
	c := createTestChange(repo, tenantID, userID)
	c.Status = status
	return c
}

// seedRollbackSteps 直接向 mock 写入清单（绕过 service 的状态前置校验，
// 用于构造"清单已存在"的场景）。
func seedRollbackSteps(t *testing.T, repo *mockRepository, changeID int, steps ...*RollbackStep) []*RollbackStep {
	t.Helper()
	require.NoError(t, repo.ReplaceRollbackSteps(context.Background(), changeID, 1, steps))
	stored, err := repo.ListRollbackSteps(context.Background(), changeID, 1)
	require.NoError(t, err)
	return stored
}

// TestTransitionStatus_RollbackBlockedByPendingStep 清单还有未确认步骤：
// 回滚必须被拒绝，且变更状态停在原地——不能出现"状态已 rolled_back、
// 清单还挂着第 2 步"的假复原记录。
func TestTransitionStatus_RollbackBlockedByPendingStep(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	steps := seedRollbackSteps(t, repo, c.ID,
		&RollbackStep{Description: "摘除 SLB 流量"},
		&RollbackStep{Description: "回滚应用镜像到上一版本"},
		&RollbackStep{Description: "验证健康检查与关键链路"},
	)
	// 第 1 步已确认，第 2、3 步仍未确认。
	ok, err := repo.ResolveRollbackStep(ctx, steps[0].ID, c.ID, tenantID, actorID, RollbackStepDone, "已摘除", time.Now())
	require.NoError(t, err)
	require.True(t, ok)

	_, err = svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", "数据库连接池耗尽，按计划回滚", "change_manager")
	require.Error(t, err, "清单未走完时不得标记已回滚")
	assert.ErrorIs(t, err, ErrRollbackStepsUnresolved)
	assert.Contains(t, err.Error(), "第2步", "文案必须点出还差哪几步（可行动）")
	assert.Contains(t, err.Error(), "第3步")

	fresh, getErr := repo.Get(ctx, c.ID, tenantID)
	require.NoError(t, getErr)
	assert.Equal(t, "in_progress", fresh.Status, "被门禁拒绝时状态不得推进")
	assert.Empty(t, fresh.OutcomeNote, "被门禁拒绝时不应留下回滚原因（否则看起来像已经回滚过）")
}

// TestTransitionStatus_RollbackAllowedWhenAllStepsResolved done/skipped/failed
// 都算"已确认"——回滚失败也是真实结果，把它卡在 pending 只会逼人谎报完成。
func TestTransitionStatus_RollbackAllowedWhenAllStepsResolved(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	steps := seedRollbackSteps(t, repo, c.ID,
		&RollbackStep{Description: "摘除 SLB 流量"},
		&RollbackStep{Description: "回滚应用镜像到上一版本"},
		&RollbackStep{Description: "清理本次新增的临时配置"},
	)
	resolutions := []struct {
		status string
		note   string
	}{
		{RollbackStepDone, "已摘除，监控确认流量归零"},
		{RollbackStepFailed, "镜像回滚卡在 72%，已手工恢复到上一版本并复核"},
		{RollbackStepSkipped, "本次未新增临时配置，无清理动作"},
	}
	for i, r := range resolutions {
		ok, err := repo.ResolveRollbackStep(ctx, steps[i].ID, c.ID, tenantID, actorID, r.status, r.note, time.Now())
		require.NoError(t, err)
		require.True(t, ok)
	}

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", "变更导致连接池耗尽，已按清单回滚", "change_manager")
	require.NoError(t, err, "全部步骤已确认时应放行")
	assert.Equal(t, "rolled_back", updated.Status)
	assert.Contains(t, updated.OutcomeNote, "连接池耗尽")
}

// TestTransitionStatus_RollbackWithoutChecklistStillAllowed 没有清单时不拦：
// 生产里既有变更都没有清单，且出事时来不及编排清单正是常态。
// 门禁的语义是"清单存在则必须走完"，不是"必须先有清单"。
func TestTransitionStatus_RollbackWithoutChecklistStillAllowed(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "failed")

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", "实施失败，直接回滚到变更前", "change_manager")
	require.NoError(t, err)
	assert.Equal(t, "rolled_back", updated.Status)
}

// TestTransitionStatus_RollbackChecklistReadFailureFailsClosed 读不到清单就不许回滚：
// 此时无法证明"清单已走完"，而回滚是不可逆动作，宁可按失败处理。
func TestTransitionStatus_RollbackChecklistReadFailureFailsClosed(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	repo.rollbackStepsErr = errors.New("db is down")

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", "按计划回滚", "change_manager")
	require.Error(t, err, "读不到清单时不得放行回滚")
	assert.Contains(t, err.Error(), "回滚清单")

	fresh, getErr := repo.Get(ctx, c.ID, tenantID)
	require.NoError(t, getErr)
	assert.Equal(t, "in_progress", fresh.Status)
}

// TestReplaceRollbackSteps_LockedOnceImplementationStarted 实施开始后清单一律锁定：
// 若允许此时整表覆盖，操作人"清空清单 → 点回滚"就绕过了上一条门禁。
func TestReplaceRollbackSteps_LockedOnceImplementationStarted(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	for _, status := range []string{"in_progress", "failed", "rolled_back", "completed", "closed", "cancelled"} {
		c := seedChangeWithStatus(t, repo, tenantID, actorID, status)
		_, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{"摘流量"})
		require.Error(t, err, "状态 '%s' 下不应允许改写回滚清单", status)
		assert.ErrorIs(t, err, ErrRollbackChecklistLocked, "状态 '%s'", status)
	}
}

// TestReplaceRollbackSteps_AllowedBeforeImplementation 计划阶段可编排：
// draft/pending/approved/scheduled 都能改（还没执行，改计划不改历史）。
func TestReplaceRollbackSteps_AllowedBeforeImplementation(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	for _, status := range []string{"draft", "pending", "approved", "scheduled"} {
		c := seedChangeWithStatus(t, repo, tenantID, actorID, status)
		steps, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{"摘除流量", "回滚镜像"})
		require.NoError(t, err, "状态 '%s' 下应允许编排清单", status)
		require.Len(t, steps, 2)
		assert.Equal(t, 1, steps[0].StepNo, "序号由服务端按顺序生成")
		assert.Equal(t, 2, steps[1].StepNo)
		assert.Equal(t, RollbackStepPending, steps[0].Status, "新清单一律 pending")
	}
}

// TestReplaceRollbackSteps_RejectsBlankAndOversizedSteps 空步骤与超长步骤都拒绝：
// 空步骤无法执行，却会让"清单已走完"的门禁失去意义。
func TestReplaceRollbackSteps_RejectsBlankAndOversizedSteps(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201
	c := seedChangeWithStatus(t, repo, tenantID, actorID, "draft")

	_, err := svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{"摘除流量", "   "})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRollbackStepsUnresolved)
	assert.Contains(t, err.Error(), "第 2 步")

	long := make([]rune, rollbackStepMaxLen+1)
	for i := range long {
		long[i] = '步'
	}
	_, err = svc.ReplaceRollbackSteps(ctx, c.ID, tenantID, []string{string(long)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "超过")
}

// TestResolveRollbackStep_RequiresNoteForSkipAndFail 跳过与失败必须写依据：
// 没有理由的"跳过"在审计里与"漏做"无法区分，而漏做正是回滚最危险的形态。
func TestResolveRollbackStep_RequiresNoteForSkipAndFail(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	steps := seedRollbackSteps(t, repo, c.ID, &RollbackStep{Description: "回滚镜像"})

	for _, status := range []string{RollbackStepSkipped, RollbackStepFailed} {
		_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, status, "  ")
		require.Error(t, err, "status=%s 缺备注时必须拒绝", status)
		assert.ErrorIs(t, err, ErrRollbackStepsUnresolved)
		assert.Contains(t, err.Error(), "备注")
	}

	// done 不强制备注：成功本身即结果，强制填反而诱导敷衍。
	_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "")
	require.NoError(t, err)
}

// TestResolveRollbackStep_RejectsWhenNotExecuting 只有实施中/失败才能确认步骤：
// 否则可以先在草稿阶段把清单勾满，再靠一次状态翻转身"秒回滚"。
func TestResolveRollbackStep_RejectsWhenNotExecuting(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	for _, status := range []string{"draft", "pending", "approved", "scheduled", "rolled_back", "completed"} {
		c := seedChangeWithStatus(t, repo, tenantID, actorID, status)
		steps := seedRollbackSteps(t, repo, c.ID, &RollbackStep{Description: "回滚镜像"})
		_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "")
		require.Error(t, err, "状态 '%s' 下不应允许确认步骤", status)
		assert.ErrorIs(t, err, ErrRollbackChecklistLocked, "状态 '%s'", status)
	}
}

// TestResolveRollbackStep_DoubleResolveConflicts 重复确认返回 409 而不是静默覆盖：
// 后到的人覆盖前一位操作人的备注与时间，等于篡改执行记录。
func TestResolveRollbackStep_DoubleResolveConflicts(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorA, actorB = 1, 201, 202

	c := seedChangeWithStatus(t, repo, tenantID, actorA, "in_progress")
	steps := seedRollbackSteps(t, repo, c.ID, &RollbackStep{Description: "回滚镜像"})

	_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorA, RollbackStepDone, "已完成")
	require.NoError(t, err)

	_, err = svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorB, RollbackStepFailed, "其实失败了")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRollbackStepAlreadyResolved)
	assert.Contains(t, err.Error(), "已被", "文案应说明已被谁确认（先到者优先）")

	// 先到者的结论必须还在。
	current, err := repo.ListRollbackSteps(ctx, c.ID, tenantID)
	require.NoError(t, err)
	require.Len(t, current, 1)
	assert.Equal(t, RollbackStepDone, current[0].Status)
	assert.Equal(t, "已完成", current[0].Note)
}

// TestResolveRollbackStep_UnknownStepIsNotFound 不存在/跨租户的步骤统一 404：
// 两者返回不同错误会让攻击者能枚举出别的租户有哪些 step id。
func TestResolveRollbackStep_UnknownStepIsNotFound(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, 999999, actorID, RollbackStepDone, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRollbackStepNotFound)
}

// TestGetRollbackSteps_UnknownChangeIsNotFound 父对象不存在时不得返回空清单：
// 空清单会被前端解读为"这张单没有清单、可以直接回滚"，掩盖越权探测的成功信号。
func TestGetRollbackSteps_UnknownChangeIsNotFound(t *testing.T) {
	svc, _ := newTransitionTestService(t)
	_, err := svc.GetRollbackSteps(context.Background(), 999999, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrChangeNotFound)
}

// TestResolveRollbackStep_ResetToPendingClearsResolver 撤销误点：状态回 pending
// 并清空确认人/时间，但保留备注作为线索（备注是操作人写的，不该被系统抹掉）。
func TestResolveRollbackStep_ResetToPendingClearsResolver(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "failed")
	steps := seedRollbackSteps(t, repo, c.ID, &RollbackStep{Description: "回滚镜像"})

	_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepDone, "以为回好了")
	require.NoError(t, err)

	reset, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, RollbackStepPending, "复核发现镜像 tag 未变")
	require.NoError(t, err)
	assert.Equal(t, RollbackStepPending, reset.Status)
	assert.Nil(t, reset.ResolvedBy, "撤销后不应再挂着确认人")
	assert.Nil(t, reset.ResolvedAt)
	assert.Equal(t, "复核发现镜像 tag 未变", reset.Note, "备注保留，作为复核线索")
}

// TestResolveRollbackStep_InvalidStatusRejected 未知状态一律拒绝（失败关闭）。
func TestResolveRollbackStep_InvalidStatusRejected(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 201

	c := seedChangeWithStatus(t, repo, tenantID, actorID, "in_progress")
	steps := seedRollbackSteps(t, repo, c.ID, &RollbackStep{Description: "回滚镜像"})

	_, err := svc.ResolveRollbackStep(ctx, c.ID, tenantID, steps[0].ID, actorID, "ok", "done")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不合法")
}

// 保持测试文件引用 zaptest（与同包其他测试一致，避免 import 摆动）。
var _ = zaptest.NewLogger
