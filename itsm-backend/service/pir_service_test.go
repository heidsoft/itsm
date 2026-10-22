package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// PIR 状态门禁
//
// PIR 是「实施后回顾」：只有真正实施过的变更才有可回顾的事实。
// 此前 CreatePIR 只校验变更存在与"是否已有 PIR"，草稿/待审批的变更也能建 PIR，
// 于是 PIR 列表里混着还没开始的单，"实施成功率 / 目标是否达成"这类指标直接失真。
// ============================================================

// mkPIRChange 建一个指定状态的变更（PIR 门禁只关心 status/tenant）。
func mkPIRChange(t *testing.T, client *ent.Client, tenantID int, status change.Status, title string) int {
	t.Helper()
	c, err := client.Change.Create().
		SetChangeNumber("CHG-PIR-" + title).
		SetTitle(title).
		SetType("normal").
		SetStatus(status).
		SetPriority("medium").
		SetImpactScope("low").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return c.ID
}

// TestCreatePIR_RejectsUnimplementedChange 草稿/待审批/已批准但未实施完成的变更不能建 PIR。
func TestCreatePIR_RejectsUnimplementedChange(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewChangePIRService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()
	const tenantID = 1

	for _, status := range []change.Status{
		change.StatusDraft,
		change.StatusPending,
		change.StatusApproved,
		change.StatusScheduled,
		change.StatusInProgress,
	} {
		changeID := mkPIRChange(t, client, tenantID, status, string(status))
		_, err := svc.CreatePIR(ctx, &dto.CreateChangePIRRequest{
			ChangeID:           changeID,
			OverallResult:      "successful",
			ObjectivesAchieved: true,
		}, 7, tenantID)

		require.Error(t, err, "状态 %s 不应允许创建 PIR", status)
		assert.ErrorIs(t, err, ErrPIRRequiresImplementedChange)
		assert.Contains(t, err.Error(), "尚未实施完成", "错误必须说明原因，便于使用者判断该等什么")
	}
}

// TestCreatePIR_AllowsImplementedChange completed / rolled_back / failed 三种"已发生实施事实"
// 的状态都必须能建 PIR——包括失败与回滚，那正是最需要复盘的两类结果。
func TestCreatePIR_AllowsImplementedChange(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewChangePIRService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()
	const tenantID = 1

	for _, status := range []change.Status{
		change.StatusCompleted,
		change.StatusRolledBack,
		change.StatusFailed,
	} {
		changeID := mkPIRChange(t, client, tenantID, status, string(status))
		pir, err := svc.CreatePIR(ctx, &dto.CreateChangePIRRequest{
			ChangeID:           changeID,
			OverallResult:      "successful",
			ObjectivesAchieved: true,
		}, 7, tenantID)

		require.NoError(t, err, "状态 %s 应允许创建 PIR", status)
		require.NotNil(t, pir)
		assert.Equal(t, changeID, pir.ChangeID)
	}
}
