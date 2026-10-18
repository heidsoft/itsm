package feishu

// P1 命令纪律回归：数据库事务内禁止连接器/HTTP 外部调用。
// SyncTicketToFeishu 曾以 *ent.Tx 形态被命令 handler 在打开的事务内调用，
// HTTP 外呼横跨整个 DB 事务（锁被远端延迟放大、连接池耗尽、回滚丢弃已成功的
// 映射状态）。修复后拆为「短事务 claim → 无事务外呼 → 短事务落结果」，
// 以下测试锁定该形态：
//   1. 外呼进行中（服务端阻塞），并发 DB 写不被阻塞（证明无事务横跨外呼）。
//   2. create 成功后映射落库（synced + GUID 回填）。
//   3. create 失败后映射落库为 failed + 错误信息（不吞错，状态可观察）。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/feishuticketsync"

	_ "github.com/mattn/go-sqlite3"
)

type feishuTaskServer struct {
	mu        sync.Mutex
	phase     string // "block" | "fail" | "ok"
	release   chan struct{}
	started   chan struct{}
	taskGUIDs []string
}

func (s *feishuTaskServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/auth/v3/tenant_access_token/internal"):
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0, "tenant_access_token": "t-1", "expire": 7200,
		})
		return
	}
	if !strings.Contains(r.URL.Path, "/task/v2/tasks") {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	phase := s.phase
	guid := fmt.Sprintf("g-%d", len(s.taskGUIDs)+1)
	s.taskGUIDs = append(s.taskGUIDs, guid)
	s.mu.Unlock()

	switch phase {
	case "block":
		close(s.started)
		<-s.release
	case "fail":
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 99901, "msg": "stub failure"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0, "msg": "ok", "data": map[string]interface{}{
			"task": map[string]interface{}{"guid": guid},
		},
	})
}

func setupSyncScenery(t *testing.T, phase string) (*Feishu, *ent.Client, *feishuTaskServer, int, int, int) {
	t.Helper()
	state := &feishuTaskServer{
		phase:   phase,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	srv := httptest.NewServer(state)
	t.Cleanup(srv.Close)
	entClient := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:feishu_sync_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { entClient.Close() })

	// tenant/user requester_id 的外键链：按依赖顺序落 fixture。
	fxTenant, err := entClient.Tenant.Create().
		SetName("FX Tenant").
		SetCode(fmt.Sprintf("fx-%d", time.Now().UnixNano())).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create fixture tenant: %v", err)
	}
	u, err := entClient.User.Create().
		SetUsername("fx-requester").
		SetName("Fixture Requester").
		SetEmail("fx-requester@example.com").
		SetPasswordHash("stub").
		SetTenantID(fxTenant.ID).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create fixture user: %v", err)
	}
	tk1, err := entClient.Ticket.Create().
		SetTitle("P1 分相回归工单").
		SetTicketNumber("TK-7").
		SetStatus("open").
		SetPriority("medium").
		SetRequesterID(u.ID).
		SetTenantID(fxTenant.ID).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create fixture ticket: %v", err)
	}
	tk2, err := entClient.Ticket.Create().
		SetTitle("并发写 fixture 工单").
		SetTicketNumber("TK-8").
		SetStatus("open").
		SetPriority("low").
		SetRequesterID(u.ID).
		SetTenantID(fxTenant.ID).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create fixture ticket 2: %v", err)
	}
	f := &Feishu{client: NewClient(srv.URL, "app", "sec", "", "")}
	return f, entClient, state, fxTenant.ID, tk1.ID, tk2.ID
}

func syncFixtureTicket(id, tenantID int) *ent.Ticket {
	return &ent.Ticket{
		ID:          id,
		TenantID:    tenantID,
		Title:       "P1 分相回归工单",
		Description: "desc",
		Status:      "open",
		Priority:    "medium",
		CreatedAt:   time.Now(),
	}
}

// TestSyncTicketToFeishu_NoTxHeldDuringExternalCall 飞书外呼进行中，其他
// DB 写必须不被阻塞——外呼绝不能横跨 DB 事务。
func TestSyncTicketToFeishu_NoTxHeldDuringExternalCall(t *testing.T) {
	f, entClient, state, tn, ticket1, ticket2 := setupSyncScenery(t, "block")

	errCh := make(chan error, 1)
	go func() {
		_, err := f.SyncTicketToFeishu(context.Background(), entClient, syncFixtureTicket(ticket1, tn))
		errCh <- err
	}()

	select {
	case <-state.started:
	case <-time.After(2 * time.Second):
		t.Fatal("feishu outbound call never started")
	}

	// 外呼在途：并发 DB 写（另一工单的 claim 行）必须能立即落库。
	writeDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := entClient.FeishuTicketSync.Create().
			SetTenantID(tn).
			SetTicketID(ticket2).
			SetFeishuTaskID(fmt.Sprintf("pending:%d:%d", tn, ticket2)).
			Save(ctx)
		writeDone <- err
	}()
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("concurrent DB write blocked by in-flight feishu call: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("DB write blocked while feishu HTTP in flight — transaction spans external call")
	}

	close(state.release)
	if err := <-errCh; err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

// TestSyncTicketToFeishu_SuccessPersistsMapping create 成功后映射必须落库
// （synced + GUID 回填）。
func TestSyncTicketToFeishu_SuccessPersistsMapping(t *testing.T) {
	f, entClient, _, tn, ticket1, _ := setupSyncScenery(t, "ok")
	ctx := context.Background()

	rec, err := f.SyncTicketToFeishu(ctx, entClient, syncFixtureTicket(ticket1, tn))
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if rec.GUID == "" {
		t.Fatal("created task must carry guid")
	}
	mapping, err := entClient.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tn), feishuticketsync.TicketID(ticket1)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query mapping: %v", err)
	}
	if mapping.SyncStatus != "synced" || mapping.FeishuTaskGUID != rec.GUID {
		t.Fatalf("unexpected mapping: status=%s guid=%s", mapping.SyncStatus, mapping.FeishuTaskGUID)
	}
}

// TestSyncTicketToFeishu_FailurePersistsFailedState create 失败后必须把
// failed 状态与错误信息落库（可观察），而不是只把错误上抛。
func TestSyncTicketToFeishu_FailurePersistsFailedState(t *testing.T) {
	f, entClient, _, tn, ticket1, _ := setupSyncScenery(t, "fail")
	ctx := context.Background()

	if _, err := f.SyncTicketToFeishu(ctx, entClient, syncFixtureTicket(ticket1, tn)); err == nil {
		t.Fatal("expected create failure to surface as error")
	}
	mapping, err := entClient.FeishuTicketSync.Query().
		Where(feishuticketsync.TenantID(tn), feishuticketsync.TicketID(ticket1)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query mapping: %v", err)
	}
	if mapping.SyncStatus != "failed" {
		t.Fatalf("sync status = %s, want failed", mapping.SyncStatus)
	}
	if mapping.ErrorMessage == "" {
		t.Fatal("error message must be persisted for observability")
	}
}
