package service

import (
	"context"
	"sync/atomic"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/configurationitemhistory"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/hook"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

func setupCIHistoryFixture(t *testing.T, dbName string) (*ent.Client, *CIHistoryService, context.Context, *ent.Tenant, *ent.ConfigurationItem) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+dbName+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := createCMDBTestTenant(ctx, client, "History "+dbName, "hist-"+dbName, dbName+".example.com")
	if err != nil {
		t.Fatal(err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "服务器")
	if err != nil {
		t.Fatal(err)
	}
	ci, err := createTestCI(ctx, client, tenant.ID, ciType.ID, "web-01")
	if err != nil {
		t.Fatal(err)
	}
	return client, NewCIHistoryService(client, logger), ctx, tenant, ci
}

// TestRecordCIHistoryOperatorID 无法解析操作者时历史必须写成功：
// operator_id=0 使用系统操作者约定值，负数收敛为 SystemOperatorID
func TestRecordCIHistoryOperatorID(t *testing.T) {
	client, svc, ctx, tenant, ci := setupCIHistoryFixture(t, "ci_history_operator")

	if err := svc.RecordCIHistory(ctx, ci.ID, tenant.ID, 0, "", "create", "", nil, ci); err != nil {
		t.Fatalf("operator_id=0 should be recorded: %v", err)
	}
	if err := svc.RecordCIHistory(ctx, ci.ID, tenant.ID, -1, "", "update", "", ci, ci); err != nil {
		t.Fatalf("negative operator_id should fall back to system operator: %v", err)
	}

	histories, err := client.ConfigurationItemHistory.Query().
		Where(configurationitemhistory.CiIDEQ(ci.ID)).
		Order(ent.Asc(configurationitemhistory.FieldVersion)).
		All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(histories) != 2 {
		t.Fatalf("len(histories) = %d, want 2", len(histories))
	}
	for _, h := range histories {
		if h.OperatorID != SystemOperatorID {
			t.Fatalf("operator_id = %d, want %d", h.OperatorID, SystemOperatorID)
		}
	}
	if histories[0].Version != 1 || histories[1].Version != 2 {
		t.Fatalf("versions = %d,%d, want 1,2", histories[0].Version, histories[1].Version)
	}
}

// TestRecordCIHistoryVersionConflictRetry 版本竞态：MAX+1 撞 (ci_id,version)
// 唯一索引时应重取版本号重试，而非直接失败。
// 通过 ent hook 在首次写入前抢占同一版本号，确定性复现并发竞态。
func TestRecordCIHistoryVersionConflictRetry(t *testing.T) {
	client, svc, ctx, tenant, ci := setupCIHistoryFixture(t, "ci_history_race")

	var injected atomic.Bool
	client.ConfigurationItemHistory.Use(func(next ent.Mutator) ent.Mutator {
		return hook.ConfigurationItemHistoryFunc(func(ctx context.Context, m *ent.ConfigurationItemHistoryMutation) (ent.Value, error) {
			if m.Op().Is(ent.OpCreate) && injected.CompareAndSwap(false, true) {
				if version, ok := m.Version(); ok {
					// 模拟并发写入者抢先占用同一版本号（内层 Create 因 injected 已置位直接放行）
					if _, err := client.ConfigurationItemHistory.Create().
						SetCiID(ci.ID).
						SetVersion(version).
						SetOperation("update").
						SetOperatorID(SystemOperatorID).
						SetTenantID(tenant.ID).
						Save(ctx); err != nil {
						t.Fatalf("failed to inject conflicting history: %v", err)
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	if err := svc.RecordCIHistory(ctx, ci.ID, tenant.ID, 1, "admin", "create", "", nil, ci); err != nil {
		t.Fatalf("version conflict should be retried: %v", err)
	}

	histories, err := client.ConfigurationItemHistory.Query().
		Where(configurationitemhistory.CiIDEQ(ci.ID)).
		Order(ent.Asc(configurationitemhistory.FieldVersion)).
		All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(histories) != 2 {
		t.Fatalf("len(histories) = %d, want 2 (injected + retried)", len(histories))
	}
	if histories[0].Version != 1 || histories[1].Version != 2 {
		t.Fatalf("versions = %d,%d, want 1,2", histories[0].Version, histories[1].Version)
	}
	if histories[1].Operation != "create" || histories[1].OperatorID != 1 {
		t.Fatalf("retried record = %+v, want operation=create operator_id=1", histories[1])
	}
}
