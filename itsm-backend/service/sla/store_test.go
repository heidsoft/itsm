package sla

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/slastate"

	entsql "entgo.io/ent/dialect/sql"

	_ "github.com/mattn/go-sqlite3"
)

type testEnv struct {
	store        *Store
	client       *ent.Client
	definitionID int
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:ent?mode=memory&_fk=1")
	if err != nil {
		t.Fatalf("open sql driver: %v", err)
	}
	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	def, err := client.SLADefinition.Create().
		SetName("test-sla").
		SetTenantID(1).
		SetResponseTime(60).
		SetResolutionTime(480).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create sla definition: %v", err)
	}

	engine := NewEngine(nil).WithNowFunc(func() time.Time {
		return time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	})
	return &testEnv{
		store:        NewStore(client, engine),
		client:       client,
		definitionID: def.ID,
	}
}

func TestStore_ComputeAndSave(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	state, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     100,
		SLADefinitionID: env.definitionID,
		SLAPolicyID:     "pol-1",
		ResponseTime:    60,
		ResolutionTime:  480,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}
	if state.Status != "active" {
		t.Errorf("status: got %q, want %q", state.Status, "active")
	}
	expectedResp := start.Add(60 * time.Minute)
	if !state.ResponseDeadline.Equal(expectedResp) {
		t.Errorf("response deadline: got %v, want %v", state.ResponseDeadline, expectedResp)
	}
	expectedRes := start.Add(480 * time.Minute)
	if !state.ResolutionDeadline.Equal(expectedRes) {
		t.Errorf("resolution deadline: got %v, want %v", state.ResolutionDeadline, expectedRes)
	}
}

func TestStore_ComputeAndSave_Upsert(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	input := CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     100,
		SLADefinitionID: env.definitionID,
		ResponseTime:    60,
		ResolutionTime:  480,
		StartTime:       start,
	}

	_, err := env.store.ComputeAndSave(ctx, input)
	if err != nil {
		t.Fatalf("first ComputeAndSave: %v", err)
	}

	input.SLADefinitionID = env.definitionID
	input.ResponseTime = 30
	updated, err := env.store.ComputeAndSave(ctx, input)
	if err != nil {
		t.Fatalf("second ComputeAndSave: %v", err)
	}
	if updated.SLADefinitionID != env.definitionID {
		t.Errorf("definition ID: got %d, want %d", updated.SLADefinitionID, env.definitionID)
	}
	expectedResp := start.Add(30 * time.Minute)
	if !updated.ResponseDeadline.Equal(expectedResp) {
		t.Errorf("updated response deadline: got %v, want %v", updated.ResponseDeadline, expectedResp)
	}
}

func TestStore_GetState(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "incident",
		AggregateID:     200,
		SLADefinitionID: env.definitionID,
		ResponseTime:    30,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	state, err := env.store.GetState(ctx, 1, "incident", 200)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.AggregateType != "incident" {
		t.Errorf("aggregate type: got %q, want %q", state.AggregateType, "incident")
	}
}

func TestStore_RecordFirstResponse(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     300,
		SLADefinitionID: env.definitionID,
		ResponseTime:    60,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	respAt := start.Add(20 * time.Minute)
	err = env.store.RecordFirstResponse(ctx, 1, "ticket", 300, respAt)
	if err != nil {
		t.Fatalf("RecordFirstResponse: %v", err)
	}

	state, _ := env.store.GetState(ctx, 1, "ticket", 300)
	if !state.FirstResponseAt.Equal(respAt) {
		t.Errorf("first_response_at: got %v, want %v", state.FirstResponseAt, respAt)
	}
}

func TestStore_RecordResolution(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     400,
		SLADefinitionID: env.definitionID,
		ResponseTime:    60,
		ResolutionTime:  480,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	// 响应必须在 11:00 截止前完成，met 才是正确预期；缺响应时按 engine.go:115
	// 应判 breached（见 DeadlineOutcome/late_response 用例）。
	if err := env.store.RecordFirstResponse(ctx, 1, "ticket", 400, start.Add(30*time.Minute)); err != nil {
		t.Fatalf("RecordFirstResponse: %v", err)
	}

	resolvedAt := start.Add(120 * time.Minute)
	err = env.store.RecordResolution(ctx, 1, "ticket", 400, resolvedAt)
	if err != nil {
		t.Fatalf("RecordResolution: %v", err)
	}

	state, _ := env.store.GetState(ctx, 1, "ticket", 400)
	if state.Status != "met" {
		t.Errorf("status: got %q, want %q", state.Status, "met")
	}
	if !state.ResolvedAt.Equal(resolvedAt) {
		t.Errorf("resolved_at: got %v, want %v", state.ResolvedAt, resolvedAt)
	}
}

func TestStore_RecordResolutionInTx_IdempotentAndPreservesBreach(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID: 1, AggregateType: "ticket", AggregateID: 600,
		SLADefinitionID: env.definitionID, ResponseTime: 60, ResolutionTime: 480, StartTime: start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	// 响应必须在 11:00 截止前完成，met 才是正确预期；缺响应时按 engine.go:115
	// 应判 breached（见 DeadlineOutcome/late_response 用例）。
	if err := env.store.RecordFirstResponse(ctx, 1, "ticket", 600, start.Add(30*time.Minute)); err != nil {
		t.Fatalf("RecordFirstResponse: %v", err)
	}

	first := start.Add(120 * time.Minute)
	tx, err := env.client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := env.store.RecordResolutionInTx(ctx, tx, 1, "ticket", 600, first); err != nil {
		_ = tx.Rollback()
		t.Fatalf("RecordResolutionInTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	state, err := env.store.GetState(ctx, 1, "ticket", 600)
	if err != nil || state.Status != "met" || !state.ResolvedAt.Equal(first) {
		t.Fatalf("first resolution: state=%+v err=%v", state, err)
	}

	// 重开/重复解决不得覆盖首次解决时间戳——SLA 达标按首次解决判定。
	second := first.Add(time.Hour)
	tx, err = env.client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin second tx: %v", err)
	}
	if err := env.store.RecordResolutionInTx(ctx, tx, 1, "ticket", 600, second); err != nil {
		_ = tx.Rollback()
		t.Fatalf("second RecordResolutionInTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("second commit: %v", err)
	}
	state, _ = env.store.GetState(ctx, 1, "ticket", 600)
	if !state.ResolvedAt.Equal(first) {
		t.Errorf("resolved_at overwritten: got %v, want %v", state.ResolvedAt, first)
	}

	// 迟到的解决不抹除已记录的 breached 事实。
	_, err = env.store.ComputeAndSave(ctx, CreateInput{
		TenantID: 1, AggregateType: "ticket", AggregateID: 601,
		SLADefinitionID: env.definitionID, ResponseTime: 60, ResolutionTime: 480, StartTime: start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave breached: %v", err)
	}
	if _, err := env.client.SLAState.Update().
		Where(slastate.TenantID(1), slastate.AggregateType("ticket"), slastate.AggregateID(601)).
		SetStatus("breached").Save(ctx); err != nil {
		t.Fatalf("mark breached: %v", err)
	}
	late := start.Add(600 * time.Minute)
	tx, err = env.client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin breach tx: %v", err)
	}
	if err := env.store.RecordResolutionInTx(ctx, tx, 1, "ticket", 601, late); err != nil {
		_ = tx.Rollback()
		t.Fatalf("breach RecordResolutionInTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("breach commit: %v", err)
	}
	breached, _ := env.store.GetState(ctx, 1, "ticket", 601)
	if breached.Status != "breached" || !breached.ResolvedAt.Equal(late) {
		t.Errorf("breach state: status=%q resolvedAt=%v", breached.Status, breached.ResolvedAt)
	}
}

func TestStore_PauseResume(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     500,
		SLADefinitionID: env.definitionID,
		ResponseTime:    60,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	err = env.store.Pause(ctx, 1, "ticket", 500, "waiting for customer")
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	state, _ := env.store.GetState(ctx, 1, "ticket", 500)
	if state.Status != "paused" {
		t.Errorf("after pause: status got %q, want %q", state.Status, "paused")
	}
	if state.PauseReason != "waiting for customer" {
		t.Errorf("pause reason: got %q, want %q", state.PauseReason, "waiting for customer")
	}

	err = env.store.Resume(ctx, 1, "ticket", 500)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	state, _ = env.store.GetState(ctx, 1, "ticket", 500)
	if state.Status != "active" {
		t.Errorf("after resume: status got %q, want %q", state.Status, "active")
	}
	if !state.PausedAt.IsZero() {
		t.Errorf("paused_at should be cleared after resume")
	}
}

func TestStore_BreachCheck(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	_, err := env.store.ComputeAndSave(ctx, CreateInput{
		TenantID:        1,
		AggregateType:   "ticket",
		AggregateID:     600,
		SLADefinitionID: env.definitionID,
		ResponseTime:    30,
		StartTime:       start,
	})
	if err != nil {
		t.Fatalf("ComputeAndSave: %v", err)
	}

	futureNow := start.Add(60 * time.Minute)
	breached, err := env.store.BreachCheck(ctx, futureNow)
	if err != nil {
		t.Fatalf("BreachCheck: %v", err)
	}
	if breached != 1 {
		t.Errorf("breached count: got %d, want 1", breached)
	}

	state, _ := env.store.GetState(ctx, 1, "ticket", 600)
	if state.Status != "breached" {
		t.Errorf("status: got %q, want %q", state.Status, "breached")
	}
}

func TestStore_SaveDeadlines(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	responseDL := time.Date(2026, 9, 27, 11, 0, 0, 0, time.UTC)
	resolutionDL := time.Date(2026, 9, 27, 18, 0, 0, 0, time.UTC)

	err := env.store.SaveDeadlines(ctx, 1, "ticket", 700, env.definitionID, &responseDL, &resolutionDL)
	if err != nil {
		t.Fatalf("SaveDeadlines: %v", err)
	}

	state, err := env.store.GetState(ctx, 1, "ticket", 700)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Status != "active" {
		t.Errorf("status: got %q, want %q", state.Status, "active")
	}
	if !state.ResponseDeadline.Equal(responseDL) {
		t.Errorf("response_deadline: got %v, want %v", state.ResponseDeadline, responseDL)
	}
	if !state.ResolutionDeadline.Equal(resolutionDL) {
		t.Errorf("resolution_deadline: got %v, want %v", state.ResolutionDeadline, resolutionDL)
	}
	if state.SLADefinitionID != env.definitionID {
		t.Errorf("sla_definition_id: got %d, want %d", state.SLADefinitionID, env.definitionID)
	}

	// Upsert: update deadlines
	newResponseDL := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	err = env.store.SaveDeadlines(ctx, 1, "ticket", 700, env.definitionID, &newResponseDL, &resolutionDL)
	if err != nil {
		t.Fatalf("SaveDeadlines upsert: %v", err)
	}

	state, _ = env.store.GetState(ctx, 1, "ticket", 700)
	if !state.ResponseDeadline.Equal(newResponseDL) {
		t.Errorf("updated response_deadline: got %v, want %v", state.ResponseDeadline, newResponseDL)
	}
}

// 扫描器尚未运行时，解决动作也必须独立守住截止时间不变量。
func TestStore_RecordResolutionInTx_DeadlineOutcome(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		responseDone bool
		minutes      int
		want         string
	}{
		{"late_resolution", "active", true, 600, "breached"},
		{"late_response", "active", false, 120, "breached"},
		{"on_time", "active", false, 30, "met"},
		{"already_breached", "breached", true, 30, "breached"},
		{"paused", "paused", false, 600, "met"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
			state, err := env.store.ComputeAndSave(ctx, CreateInput{TenantID: 1, AggregateType: "ticket", AggregateID: 700, SLADefinitionID: env.definitionID, ResponseTime: 60, ResolutionTime: 480, StartTime: start})
			if err != nil {
				t.Fatal(err)
			}
			update := env.client.SLAState.UpdateOneID(state.ID).SetStatus(tc.status)
			if tc.responseDone {
				update.SetFirstResponseAt(start.Add(10 * time.Minute))
			}
			if tc.status == "paused" {
				update.SetPausedAt(start.Add(20 * time.Minute))
			}
			if _, err := update.Save(ctx); err != nil {
				t.Fatal(err)
			}
			tx, err := env.client.Tx(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			at := start.Add(time.Duration(tc.minutes) * time.Minute)
			if err := env.store.RecordResolutionInTx(ctx, tx, 1, "ticket", 700, at); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			got, err := env.store.GetState(ctx, 1, "ticket", 700)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.want || !got.ResolvedAt.Equal(at) {
				t.Fatalf("state=%+v want status=%s", got, tc.want)
			}
		})
	}
}

// 独立事务解决同样必须守住扫描器尚未运行时的截止时间不变量。
func TestStore_RecordResolution_DeadlineOutcome(t *testing.T) {
	for i, tc := range []struct {
		name, status string
		responseDone bool
		minutes      int
		want         string
	}{
		{"late_resolution", "active", true, 600, "breached"},
		{"late_response", "active", false, 120, "breached"},
		{"on_time", "active", true, 30, "met"},
		{"already_breached", "breached", true, 30, "breached"},
		{"paused", "paused", false, 600, "met"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
			id := 800 + i
			state, err := env.store.ComputeAndSave(ctx, CreateInput{TenantID: 1, AggregateType: "ticket", AggregateID: id, SLADefinitionID: env.definitionID, ResponseTime: 60, ResolutionTime: 480, StartTime: start})
			if err != nil {
				t.Fatal(err)
			}
			update := env.client.SLAState.UpdateOneID(state.ID).SetStatus(tc.status)
			if tc.status == "paused" {
				update.SetPausedAt(start.Add(20 * time.Minute))
			}
			if _, err := update.Save(ctx); err != nil {
				t.Fatal(err)
			}
			if tc.responseDone {
				if err := env.store.RecordFirstResponse(ctx, 1, "ticket", id, start.Add(10*time.Minute)); err != nil {
					t.Fatal(err)
				}
			}
			at := start.Add(time.Duration(tc.minutes) * time.Minute)
			if err := env.store.RecordResolution(ctx, 1, "ticket", id, at); err != nil {
				t.Fatal(err)
			}
			got, err := env.store.GetState(ctx, 1, "ticket", id)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.want || !got.ResolvedAt.Equal(at) {
				t.Fatalf("state=%+v want status=%s", got, tc.want)
			}
		})
	}
}
