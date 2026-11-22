package sla

import (
	"context"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/slastate"

	"go.uber.org/zap"
)

// Store 持久化 SLA 状态。
// 组合 Engine（纯计算）与 ent.Client（持久化），对外提供面向聚合根的 SLA 生命周期操作。
type Store struct {
	client *ent.Client
	engine *Engine
}

// NewStore 创建 SLA 持久化层。
func NewStore(client *ent.Client, engine *Engine) *Store {
	return &Store{client: client, engine: engine}
}

// CreateInput 创建 SLA 状态的输入。
type CreateInput struct {
	TenantID        int
	AggregateType   string
	AggregateID     int
	SLADefinitionID int
	SLAPolicyID     string
	ResponseTime    int
	ResolutionTime  int
	BusinessHours   map[string]interface{}
	StartTime       time.Time
}

// ComputeAndSave 计算截止时间并写入 sla_states。
// 若同一聚合根已有记录则 upsert（更新截止时间与状态）。
func (s *Store) ComputeAndSave(ctx context.Context, input CreateInput) (*ent.SLAState, error) {
	result := s.engine.ComputeDeadlines(ctx, ComputeInput{
		TenantID:       input.TenantID,
		ResponseTime:   input.ResponseTime,
		ResolutionTime: input.ResolutionTime,
		BusinessHours:  input.BusinessHours,
		StartTime:      input.StartTime,
	})

	existing, _ := s.client.SLAState.Query().
		Where(
			slastate.TenantID(input.TenantID),
			slastate.AggregateType(input.AggregateType),
			slastate.AggregateID(input.AggregateID),
		).
		Only(ctx)

	if existing != nil {
		upd := existing.Update()
		upd.SetSLADefinitionID(input.SLADefinitionID)
		if input.SLAPolicyID != "" {
			upd.SetSLAPolicyID(input.SLAPolicyID)
		}
		if result.ResponseDeadline != nil {
			upd.SetResponseDeadline(*result.ResponseDeadline)
		}
		if result.ResolutionDeadline != nil {
			upd.SetResolutionDeadline(*result.ResolutionDeadline)
		}
		upd.SetStatus("active")
		return upd.Save(ctx)
	}

	cr := s.client.SLAState.Create().
		SetTenantID(input.TenantID).
		SetAggregateType(input.AggregateType).
		SetAggregateID(input.AggregateID).
		SetSLADefinitionID(input.SLADefinitionID).
		SetStatus("active")

	if input.SLAPolicyID != "" {
		cr.SetSLAPolicyID(input.SLAPolicyID)
	}
	if result.ResponseDeadline != nil {
		cr.SetResponseDeadline(*result.ResponseDeadline)
	}
	if result.ResolutionDeadline != nil {
		cr.SetResolutionDeadline(*result.ResolutionDeadline)
	}

	return cr.Save(ctx)
}

// SaveDeadlines 写入已计算好的截止时间（双写路径）。
// 与 ComputeAndSave 不同，调用方已经自行计算好截止时间，此处只做持久化。
func (s *Store) SaveDeadlines(ctx context.Context, tenantID int, aggregateType string, aggregateID int, slaDefinitionID int, responseDeadline, resolutionDeadline *time.Time) error {
	existing, _ := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		Only(ctx)

	if existing != nil {
		upd := existing.Update()
		if slaDefinitionID > 0 {
			upd.SetSLADefinitionID(slaDefinitionID)
		}
		if responseDeadline != nil {
			upd.SetResponseDeadline(*responseDeadline)
		}
		if resolutionDeadline != nil {
			upd.SetResolutionDeadline(*resolutionDeadline)
		}
		upd.SetStatus("active")
		_, err := upd.Save(ctx)
		return err
	}

	cr := s.client.SLAState.Create().
		SetTenantID(tenantID).
		SetAggregateType(aggregateType).
		SetAggregateID(aggregateID).
		SetStatus("active")
	if slaDefinitionID > 0 {
		cr.SetSLADefinitionID(slaDefinitionID)
	}
	if responseDeadline != nil {
		cr.SetResponseDeadline(*responseDeadline)
	}
	if resolutionDeadline != nil {
		cr.SetResolutionDeadline(*resolutionDeadline)
	}
	_, err := cr.Save(ctx)
	return err
}

// GetState 查询聚合根的 SLA 状态。
func (s *Store) GetState(ctx context.Context, tenantID int, aggregateType string, aggregateID int) (*ent.SLAState, error) {
	return s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		Only(ctx)
}

// BatchGetStates 批量查询聚合根的 SLA 状态。
// 返回 map[aggregateID]*SLAState。
func (s *Store) BatchGetStates(ctx context.Context, tenantID int, aggregateType string, aggregateIDs []int) (map[int]*ent.SLAState, error) {
	if len(aggregateIDs) == 0 {
		return map[int]*ent.SLAState{}, nil
	}
	states, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateIDIn(aggregateIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int]*ent.SLAState, len(states))
	for _, st := range states {
		result[st.AggregateID] = st
	}
	return result, nil
}

// FindOverdueAggregateIDs 返回 resolution_deadline 已过且状态为 active 的聚合根 ID。
func (s *Store) FindOverdueAggregateIDs(ctx context.Context, tenantID int, aggregateType string, now time.Time) ([]int, error) {
	states, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.StatusEQ("active"),
			slastate.ResolutionDeadlineNotNil(),
			slastate.ResolutionDeadlineLT(now),
		).
		Select(slastate.FieldAggregateID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(states))
	for i, st := range states {
		ids[i] = st.AggregateID
	}
	return ids, nil
}

// FindResponseBreachedAggregateIDs 返回 response_deadline 已过且尚未首次响应的聚合根 ID。
func (s *Store) FindResponseBreachedAggregateIDs(ctx context.Context, tenantID int, aggregateType string, now time.Time) ([]int, error) {
	states, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.StatusEQ("active"),
			slastate.ResponseDeadlineNotNil(),
			slastate.ResponseDeadlineLT(now),
			slastate.FirstResponseAtIsNil(),
		).
		Select(slastate.FieldAggregateID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(states))
	for i, st := range states {
		ids[i] = st.AggregateID
	}
	return ids, nil
}

// FindAtRiskAggregateIDs 返回 response_deadline 在 [now, now+within] 窗口内且尚未首次响应的聚合根 ID。
func (s *Store) FindAtRiskAggregateIDs(ctx context.Context, tenantID int, aggregateType string, now time.Time, within time.Duration) ([]int, error) {
	deadline := now.Add(within)
	states, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.StatusEQ("active"),
			slastate.ResponseDeadlineNotNil(),
			slastate.ResponseDeadlineGTE(now),
			slastate.ResponseDeadlineLTE(deadline),
			slastate.FirstResponseAtIsNil(),
		).
		Select(slastate.FieldAggregateID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(states))
	for i, st := range states {
		ids[i] = st.AggregateID
	}
	return ids, nil
}

// CountBreached 返回已 breached 的聚合根数量。
func (s *Store) CountBreached(ctx context.Context, tenantID int, aggregateType string) (int, error) {
	return s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.StatusEQ("breached"),
		).
		Count(ctx)
}

// FindWithSLABound 返回绑定了 SLA（sla_definition_id > 0）的聚合根 ID。
func (s *Store) FindWithSLABound(ctx context.Context, tenantID int, aggregateType string) ([]int, error) {
	states, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.SLADefinitionIDNotNil(),
		).
		Select(slastate.FieldAggregateID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(states))
	for i, st := range states {
		ids[i] = st.AggregateID
	}
	return ids, nil
}

// RecordFirstResponse 记录首次响应时间。
func (s *Store) RecordFirstResponse(ctx context.Context, tenantID int, aggregateType string, aggregateID int, at time.Time) error {
	n, err := s.client.SLAState.Update().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		SetFirstResponseAt(at).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		zap.S().Warnw("sla_state not found for RecordFirstResponse",
			"tenantID", tenantID, "aggregateType", aggregateType, "aggregateID", aggregateID)
	}
	return nil
}

// RecordResolution 记录解决时间。
// 与 RecordResolutionInTx 共用 resolutionStatus，按权威截止时间判定，
// 与扫描器 BreachCheck 口径一致，迟到解决不抹除违规事实。
func (s *Store) RecordResolution(ctx context.Context, tenantID int, aggregateType string, aggregateID int, at time.Time) error {
	existing, err := s.client.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			zap.S().Warnw("sla_state not found for RecordResolution",
				"tenantID", tenantID, "aggregateType", aggregateType, "aggregateID", aggregateID)
			return nil
		}
		return err
	}

	upd := existing.Update().SetResolvedAt(at)
	upd.SetStatus(resolutionStatus(existing, at))
	if _, err := upd.Save(ctx); err != nil {
		return err
	}
	return nil
}

// RecordResolutionInTx 在调用方事务内记录解决时间，保证「实体已 resolved」与
// 「SLA 权威状态已收口」同生共死。
//
// 幂等：已 resolved 的只保留首次时间戳；并发重复调用由 ResolvedAtIsNil 条件更新
// 收敛，affected=0 视为对方已写入。已 breached 的状态保持 breached，迟到解决
// 不抹除违规事实。
func (s *Store) RecordResolutionInTx(ctx context.Context, tx *ent.Tx, tenantID int, aggregateType string, aggregateID int, at time.Time) error {
	state, err := tx.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 未绑定 SLA 的聚合根是合法状态；SLA 绑定失败不应回滚业务解决。
			zap.S().Warnw("sla_state not found for RecordResolutionInTx",
				"tenantID", tenantID, "aggregateType", aggregateType, "aggregateID", aggregateID)
			return nil
		}
		return err
	}
	if !state.ResolvedAt.IsZero() {
		return nil
	}

	upd := tx.SLAState.Update().
		Where(
			slastate.ID(state.ID),
			slastate.ResolvedAtIsNil(),
		).
		SetResolvedAt(at)
	upd.SetStatus(resolutionStatus(state, at))
	_, err = upd.Save(ctx)
	return err
}

// resolutionStatus 是两条解决路径的唯一判定源，与扫描器 BreachCheck 共用 SLAStatus。
// 不能依赖扫描器先标记 breached，否则扫描间隙内解决会永久漏报。
func resolutionStatus(state *ent.SLAState, at time.Time) string {
	if state.Status == "breached" {
		return "breached"
	}
	// 暂停期间不消耗 SLA 时间；用暂停时刻判定，仍保留暂停前已发生的违规。
	effectiveAt := at
	if state.Status == "paused" && !state.PausedAt.IsZero() && at.After(state.PausedAt) {
		effectiveAt = state.PausedAt
	}
	if SLAStatus(effectiveAt, timePtrOrNil(state.ResponseDeadline), timePtrOrNil(state.ResolutionDeadline), state.FirstResponseAt, effectiveAt) == "breached" {
		return "breached"
	}
	return "met"
}

// Pause 暂停 SLA 计时（独立事务便捷入口）。
func (s *Store) Pause(ctx context.Context, tenantID int, aggregateType string, aggregateID int, reason string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.PauseInTx(ctx, tx, tenantID, aggregateType, aggregateID, reason, time.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

// PauseInTx 在调用方事务内暂停 SLA 计时，使 sla_states 与领域实体（ticket/incident）
// 的内嵌 SLA 字段同事务提交，杜绝「实体已暂停、权威表仍 active」的双写漂移。
// 条件更新 active→paused 并检查 affected，竞态/非 active 一律 conflict，不得静默成功。
func (s *Store) PauseInTx(ctx context.Context, tx *ent.Tx, tenantID int, aggregateType string, aggregateID int, reason string, now time.Time) error {
	affected, err := tx.SLAState.Update().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
			slastate.StatusEQ("active"),
		).
		SetStatus("paused").
		SetPausedAt(now).
		SetPauseReason(reason).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return common.NewBusinessError(common.ConflictCode, "SLA 状态非 active，无法暂停（不存在、已暂停或已违规）", "")
	}
	return nil
}

// Resume 恢复 SLA 计时（独立事务便捷入口）。
func (s *Store) Resume(ctx context.Context, tenantID int, aggregateType string, aggregateID int) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, _, err := s.ResumeInTx(ctx, tx, tenantID, aggregateType, aggregateID, time.Now()); err != nil {
		return err
	}
	return tx.Commit()
}

// ResumeInTx 在调用方事务内恢复 SLA 计时，并把暂停时长并入权威 deadline。
// 返回延长后的 response/resolution deadline，供调用方用同一值双写领域实体，
// 保证 sla_states 与内嵌字段口径一致。
//
// 此前 Resume 只累加 paused_duration_seconds 而不延长 deadline，监控扫描以
// sla_states.deadline 为权威，恢复后立刻再次判违规→回收→再违规循环。
func (s *Store) ResumeInTx(ctx context.Context, tx *ent.Tx, tenantID int, aggregateType string, aggregateID int, now time.Time) (time.Time, time.Time, error) {
	state, err := tx.SLAState.Query().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return time.Time{}, time.Time{}, common.NewBusinessError(common.ConflictCode, "SLA 状态不存在，无法恢复", "")
		}
		return time.Time{}, time.Time{}, err
	}
	if state.Status != "paused" || state.PausedAt.IsZero() {
		return time.Time{}, time.Time{}, common.NewBusinessError(common.ConflictCode, "SLA 未处于暂停状态，无法恢复", "")
	}

	pausedDuration := now.Sub(state.PausedAt)
	var extendedResp, extendedRes time.Time
	upd := tx.SLAState.Update().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
			slastate.StatusEQ("paused"),
		).
		SetStatus("active").
		AddPausedDurationSeconds(int(pausedDuration.Seconds())).
		ClearPausedAt().
		ClearPauseReason()
	if !state.ResponseDeadline.IsZero() {
		extendedResp = state.ResponseDeadline.Add(pausedDuration)
		upd.SetResponseDeadline(extendedResp)
	}
	if !state.ResolutionDeadline.IsZero() {
		extendedRes = state.ResolutionDeadline.Add(pausedDuration)
		upd.SetResolutionDeadline(extendedRes)
	}
	affected, err := upd.Save(ctx)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if affected == 0 {
		return time.Time{}, time.Time{}, common.NewBusinessError(common.ConflictCode, "SLA 状态并发变更，恢复失败", "")
	}
	return extendedResp, extendedRes, nil
}

// BreachCheck 检查并标记超时违规。
// 返回被标记为 breached 的记录数。
func (s *Store) BreachCheck(ctx context.Context, now time.Time) (int, error) {
	activeStates, err := s.client.SLAState.Query().
		Where(
			slastate.StatusEQ("active"),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}

	breached := 0
	for _, st := range activeStates {
		status := SLAStatus(now,
			timePtrOrNil(st.ResponseDeadline),
			timePtrOrNil(st.ResolutionDeadline),
			st.FirstResponseAt,
			st.ResolvedAt,
		)
		if status == "breached" {
			_, err := st.Update().SetStatus("breached").Save(ctx)
			if err != nil {
				zap.S().Errorw("failed to mark sla_state breached",
					"id", st.ID, "error", err)
				continue
			}
			breached++
		}
	}
	return breached, nil
}

// timePtrOrNil 将 time.Time 转为 *time.Time；零值返回 nil。
func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
