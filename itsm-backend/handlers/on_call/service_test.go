package on_call

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

type onCallFixture struct {
	service  *Service
	tenantID int
	groupID  int
	userID   int
	outsider int
}

func newOnCallFixture(t *testing.T) *onCallFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:on_call_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("OnCall").SetCode("oncall").
		SetDomain("oncall.example.com").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("oncall-agent").SetEmail("agent@example.com").
		SetName("Agent").SetPasswordHash("hash").SetRole("agent").SetActive(true).
		SetTenantID(tenant.ID).SaveX(ctx)
	outsider := client.User.Create().SetUsername("oncall-outsider").SetEmail("outsider@example.com").
		SetName("Outsider").SetPasswordHash("hash").SetRole("agent").SetActive(true).
		SetTenantID(tenant.ID).SaveX(ctx)
	group := client.Group.Create().SetName("一线支持").SetTenantID(tenant.ID).
		AddMemberIDs(user.ID).SaveX(ctx)

	return &onCallFixture{
		service: NewService(client), tenantID: tenant.ID, groupID: group.ID,
		userID: user.ID, outsider: outsider.ID,
	}
}

func TestScheduleLifecycleAndShiftGuards(t *testing.T) {
	f := newOnCallFixture(t)
	ctx := context.Background()

	schedule, err := f.service.CreateSchedule(ctx, f.tenantID, ScheduleInput{GroupID: f.groupID, Name: "一线值班"})
	require.NoError(t, err)
	require.Equal(t, "Asia/Shanghai", schedule.Timezone, "时区缺省应可预测")
	require.Equal(t, "active", schedule.Status)

	// 名称为空 → 参数错误
	_, err = f.service.CreateSchedule(ctx, f.tenantID, ScheduleInput{GroupID: f.groupID, Name: "  "})
	require.ErrorIs(t, err, ErrInvalidSchedule)

	// 跨租户不可见
	_, err = f.service.GetSchedule(ctx, f.tenantID+999, schedule.ID)
	require.ErrorIs(t, err, ErrScheduleNotFound)

	updated, err := f.service.UpdateSchedule(ctx, f.tenantID, schedule.ID, ScheduleInput{Name: "一线值班（夜班）"})
	require.NoError(t, err)
	require.Equal(t, "一线值班（夜班）", updated.Name)

	start := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	shift, err := f.service.CreateShift(ctx, f.tenantID, schedule.ID, f.userID, start, start.Add(8*time.Hour))
	require.NoError(t, err)

	// 时间倒置 → 参数错误
	_, err = f.service.CreateShift(ctx, f.tenantID, schedule.ID, f.userID, start, start)
	require.ErrorIs(t, err, ErrInvalidShift)

	// 重叠班次 → 冲突
	_, err = f.service.CreateShift(ctx, f.tenantID, schedule.ID, f.userID, start.Add(time.Hour), start.Add(2*time.Hour))
	require.ErrorIs(t, err, ErrOverlappingShift)

	// 非支持组成员不能值班
	_, err = f.service.CreateShift(ctx, f.tenantID, schedule.ID, f.outsider, start.Add(24*time.Hour), start.Add(32*time.Hour))
	require.Error(t, err)
	require.Contains(t, err.Error(), "active member")

	// 有班次时禁止删除排班表
	require.ErrorIs(t, f.service.DeleteSchedule(ctx, f.tenantID, schedule.ID), ErrScheduleHasShift)

	require.NoError(t, f.service.DeleteShift(ctx, f.tenantID, shift.ID))
	require.NoError(t, f.service.DeleteSchedule(ctx, f.tenantID, schedule.ID))
}

func TestCurrentResolverPicksActiveShift(t *testing.T) {
	f := newOnCallFixture(t)
	ctx := context.Background()

	schedule, err := f.service.CreateSchedule(ctx, f.tenantID, ScheduleInput{GroupID: f.groupID, Name: "轮值"})
	require.NoError(t, err)
	start := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	_, err = f.service.CreateShift(ctx, f.tenantID, schedule.ID, f.userID, start, start.Add(24*time.Hour))
	require.NoError(t, err)

	current, err := f.service.CurrentResolver(ctx, f.tenantID, f.groupID, start.Add(10*time.Hour))
	require.NoError(t, err)
	require.Equal(t, f.userID, current.UserID)
	require.Equal(t, schedule.ID, current.ScheduleID)

	// 班次窗口之外 → 没有值班人
	_, err = f.service.CurrentResolver(ctx, f.tenantID, f.groupID, start.Add(48*time.Hour))
	require.True(t, errors.Is(err, ErrNoOnCall))

	// 排班表停用后不再参与解析
	_, err = f.service.UpdateSchedule(ctx, f.tenantID, schedule.ID, ScheduleInput{Status: "inactive"})
	require.NoError(t, err)
	_, err = f.service.CurrentResolver(ctx, f.tenantID, f.groupID, start.Add(10*time.Hour))
	require.True(t, errors.Is(err, ErrNoOnCall))
}
