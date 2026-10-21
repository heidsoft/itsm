// Package on_call 提供值班排班（schedule）与班次（shift）的管理能力。
//
// 为什么单独成包：排班数据被多个域消费（邮件受理的兜底派单、SLA 升级、告警路由），
// 但它此前只以 OnCallService 的形式寄生在 handlers/email_intake 里，且没有任何 HTTP
// 端点——管理员无法维护排班，只能直接改库。本包把「排班」提升为一等领域：
//   - schedule：某个支持组的值班表（时区 + 启停）；
//   - shift：具体某人在某段时间值班，同一 schedule 内不允许时间重叠。
//
// 表结构由 migrations/20260824_email_intake_poc.sql 建立，
// 排班重叠由 20260825_email_intake_hardening.sql 的排他约束兜底（应用层先做友好校验）。
package on_call

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/group"
	"itsm-backend/ent/oncallschedule"
	"itsm-backend/ent/oncallshift"
	"itsm-backend/ent/user"
)

var (
	ErrOverlappingShift = errors.New("on-call shift overlaps an existing shift")
	ErrInvalidShift     = errors.New("on-call shift end must be after start")
	ErrShiftNotFound    = errors.New("on-call shift not found")
	ErrScheduleNotFound = errors.New("on-call schedule not found")
	ErrScheduleHasShift = errors.New("on-call schedule still has shifts")
	ErrInvalidSchedule  = errors.New("on-call schedule name and group are required")
	ErrNoOnCall         = errors.New("no on-call resolver found for group")
)

// CurrentOnCall 是「此刻谁值班」的解析结果。
type CurrentOnCall struct {
	ScheduleID int       `json:"scheduleId"`
	ShiftID    int       `json:"shiftId"`
	GroupID    int       `json:"groupId"`
	UserID     int       `json:"userId"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
}

// Service 是值班排班的领域服务。
type Service struct{ client *ent.Client }

func NewService(client *ent.Client) *Service { return &Service{client: client} }

// ─── Schedule ───────────────────────────────────────────────────────

type ScheduleInput struct {
	GroupID  int
	Name     string
	Timezone string
	Status   string
}

func (s *Service) CreateSchedule(ctx context.Context, tenantID int, input ScheduleInput) (*ent.OnCallSchedule, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || input.GroupID <= 0 {
		return nil, ErrInvalidSchedule
	}
	exists, err := s.client.Group.Query().
		Where(group.IDEQ(input.GroupID), group.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("validate on-call group: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("on-call group %d not found in tenant", input.GroupID)
	}
	return s.client.OnCallSchedule.Create().
		SetTenantID(tenantID).
		SetGroupID(input.GroupID).
		SetName(name).
		SetTimezone(defaultTimezone(input.Timezone)).
		SetStatus(defaultStatus(input.Status)).
		Save(ctx)
}

func (s *Service) ListSchedules(ctx context.Context, tenantID, groupID int) ([]*ent.OnCallSchedule, error) {
	query := s.client.OnCallSchedule.Query().Where(oncallschedule.TenantIDEQ(tenantID))
	if groupID > 0 {
		query.Where(oncallschedule.GroupIDEQ(groupID))
	}
	return query.Order(ent.Asc(oncallschedule.FieldName)).All(ctx)
}

func (s *Service) GetSchedule(ctx context.Context, tenantID, scheduleID int) (*ent.OnCallSchedule, error) {
	schedule, err := s.client.OnCallSchedule.Query().
		Where(oncallschedule.IDEQ(scheduleID), oncallschedule.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return schedule, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, tenantID, scheduleID int, input ScheduleInput) (*ent.OnCallSchedule, error) {
	update := s.client.OnCallSchedule.UpdateOneID(scheduleID).Where(oncallschedule.TenantIDEQ(tenantID))
	if name := strings.TrimSpace(input.Name); name != "" {
		update.SetName(name)
	}
	if input.Timezone != "" {
		update.SetTimezone(strings.TrimSpace(input.Timezone))
	}
	if input.Status != "" {
		update.SetStatus(strings.TrimSpace(input.Status))
	}
	schedule, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return schedule, nil
}

// DeleteSchedule 删除排班表。仍有班次时拒绝删除——排班是派单依据，
// 静默级联删除会让历史派单失去可解释性。
func (s *Service) DeleteSchedule(ctx context.Context, tenantID, scheduleID int) error {
	if _, err := s.GetSchedule(ctx, tenantID, scheduleID); err != nil {
		return err
	}
	hasShift, err := s.client.OnCallShift.Query().
		Where(oncallshift.TenantIDEQ(tenantID), oncallshift.ScheduleIDEQ(scheduleID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check on-call shifts: %w", err)
	}
	if hasShift {
		return ErrScheduleHasShift
	}
	_, err = s.client.OnCallSchedule.Delete().
		Where(oncallschedule.IDEQ(scheduleID), oncallschedule.TenantIDEQ(tenantID)).
		Exec(ctx)
	return err
}

// ─── Shift ──────────────────────────────────────────────────────────

func (s *Service) CreateShift(ctx context.Context, tenantID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidShift
	}
	if err := s.validateShiftAssignment(ctx, tenantID, scheduleID, userID); err != nil {
		return nil, err
	}
	overlap, err := s.overlaps(ctx, tenantID, scheduleID, 0, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, ErrOverlappingShift
	}
	return s.client.OnCallShift.Create().
		SetTenantID(tenantID).SetScheduleID(scheduleID).SetUserID(userID).
		SetStartAt(startAt).SetEndAt(endAt).Save(ctx)
}

func (s *Service) ListShifts(ctx context.Context, tenantID, scheduleID int, from, to *time.Time) ([]*ent.OnCallShift, error) {
	query := s.client.OnCallShift.Query().Where(oncallshift.TenantIDEQ(tenantID))
	if scheduleID > 0 {
		query.Where(oncallshift.ScheduleIDEQ(scheduleID))
	}
	if from != nil {
		query.Where(oncallshift.EndAtGT(*from))
	}
	if to != nil {
		query.Where(oncallshift.StartAtLT(*to))
	}
	return query.Order(ent.Asc(oncallshift.FieldStartAt)).All(ctx)
}

func (s *Service) UpdateShift(ctx context.Context, tenantID, shiftID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidShift
	}
	if _, err := s.client.OnCallShift.Query().
		Where(oncallshift.IDEQ(shiftID), oncallshift.TenantIDEQ(tenantID)).
		Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	if err := s.validateShiftAssignment(ctx, tenantID, scheduleID, userID); err != nil {
		return nil, err
	}
	overlap, err := s.overlaps(ctx, tenantID, scheduleID, shiftID, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, ErrOverlappingShift
	}
	return s.client.OnCallShift.UpdateOneID(shiftID).
		SetScheduleID(scheduleID).SetUserID(userID).SetStartAt(startAt).SetEndAt(endAt).
		Save(ctx)
}

func (s *Service) DeleteShift(ctx context.Context, tenantID, shiftID int) error {
	deleted, err := s.client.OnCallShift.Delete().
		Where(oncallshift.IDEQ(shiftID), oncallshift.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrShiftNotFound
	}
	return nil
}

// CurrentResolver 返回指定支持组在当前时刻的值班人。
// 同一时刻命中多个班次时取最早开始的那个（交接班重叠时的确定性规则）。
func (s *Service) CurrentResolver(ctx context.Context, tenantID, groupID int, at time.Time) (*CurrentOnCall, error) {
	shifts, err := s.client.OnCallShift.Query().Where(
		oncallshift.TenantIDEQ(tenantID), oncallshift.StartAtLTE(at), oncallshift.EndAtGT(at),
		oncallshift.HasScheduleWith(
			oncallschedule.TenantIDEQ(tenantID),
			oncallschedule.GroupIDEQ(groupID),
			oncallschedule.StatusEQ("active"),
		),
		oncallshift.HasUserWith(user.TenantIDEQ(tenantID), user.ActiveEQ(true)),
	).WithSchedule().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query current on-call resolver: %w", err)
	}
	if len(shifts) == 0 {
		return nil, ErrNoOnCall
	}
	sort.Slice(shifts, func(i, j int) bool { return shifts[i].StartAt.Before(shifts[j].StartAt) })
	shift := shifts[0]
	return &CurrentOnCall{
		ScheduleID: shift.ScheduleID, ShiftID: shift.ID, GroupID: groupID,
		UserID: shift.UserID, StartAt: shift.StartAt, EndAt: shift.EndAt,
	}, nil
}

func (s *Service) overlaps(ctx context.Context, tenantID, scheduleID, excludeShiftID int, startAt, endAt time.Time) (bool, error) {
	query := s.client.OnCallShift.Query().Where(
		oncallshift.TenantIDEQ(tenantID), oncallshift.ScheduleIDEQ(scheduleID),
		oncallshift.StartAtLT(endAt), oncallshift.EndAtGT(startAt),
	)
	if excludeShiftID > 0 {
		query.Where(oncallshift.IDNEQ(excludeShiftID))
	}
	overlap, err := query.Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("check overlapping on-call shifts: %w", err)
	}
	return overlap, nil
}

// validateShiftAssignment 要求：排班表属于本租户且启用；值班人是该支持组的在职成员。
// 后者防止把班次派给一个看不到该组工单的人——那等于没有值班。
func (s *Service) validateShiftAssignment(ctx context.Context, tenantID, scheduleID, userID int) error {
	schedule, err := s.client.OnCallSchedule.Query().Where(
		oncallschedule.IDEQ(scheduleID), oncallschedule.TenantIDEQ(tenantID), oncallschedule.StatusEQ("active"),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("load on-call schedule: %w", err)
	}
	member, err := s.client.Group.Query().Where(
		group.IDEQ(schedule.GroupID), group.TenantIDEQ(tenantID),
		group.HasMembersWith(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("validate on-call member: %w", err)
	}
	if !member {
		return errors.New("on-call user must be an active member of the support group")
	}
	return nil
}

func defaultTimezone(value string) string {
	if timezone := strings.TrimSpace(value); timezone != "" {
		return timezone
	}
	return "Asia/Shanghai"
}

func defaultStatus(value string) string {
	if status := strings.TrimSpace(value); status != "" {
		return status
	}
	return "active"
}
