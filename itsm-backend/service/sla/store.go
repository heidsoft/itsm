package sla

import (
	"context"
	"time"

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
func (s *Store) RecordResolution(ctx context.Context, tenantID int, aggregateType string, aggregateID int, at time.Time) error {
	n, err := s.client.SLAState.Update().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
		).
		SetResolvedAt(at).
		SetStatus("met").
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		zap.S().Warnw("sla_state not found for RecordResolution",
			"tenantID", tenantID, "aggregateType", aggregateType, "aggregateID", aggregateID)
	}
	return nil
}

// Pause 暂停 SLA 计时。
func (s *Store) Pause(ctx context.Context, tenantID int, aggregateType string, aggregateID int, reason string) error {
	_, err := s.client.SLAState.Update().
		Where(
			slastate.TenantID(tenantID),
			slastate.AggregateType(aggregateType),
			slastate.AggregateID(aggregateID),
			slastate.StatusEQ("active"),
		).
		SetStatus("paused").
		SetPausedAt(time.Now()).
		SetPauseReason(reason).
		Save(ctx)
	return err
}

// Resume 恢复 SLA 计时，累加暂停时长。
func (s *Store) Resume(ctx context.Context, tenantID int, aggregateType string, aggregateID int) error {
	state, err := s.GetState(ctx, tenantID, aggregateType, aggregateID)
	if err != nil {
		return err
	}
	if state.Status != "paused" || state.PausedAt.IsZero() {
		return nil
	}

	pauseDuration := int(time.Since(state.PausedAt).Seconds())
	_, err = state.Update().
		SetStatus("active").
		SetPausedDurationSeconds(state.PausedDurationSeconds + pauseDuration).
		ClearPausedAt().
		ClearPauseReason().
		Save(ctx)
	return err
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
