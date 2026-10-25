package change

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"
)

// 回归：toDomain 曾对 Ent 的可选字段直接取址（&ec.AssigneeID / &ec.PlannedStartDate …）。
// Ent 的 Optional() 字段生成 Go 值类型，未设置时是零值 —— 取址后 API 下发
// assigneeId:0 与 0001-01-01T00:00:00Z，调用方无法区分「未设置」和「设置为零值」，
// 前端还会把 0 当成合法用户去反查姓名、把公元 1 年当成排期画进日历。

func openMapperTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:change-mapper-%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestToDomain_MapsOptionalZeroValuesToNil(t *testing.T) {
	client := openMapperTestClient(t)

	created, err := client.Change.Create().
		SetTitle("未排期变更").
		SetDescription("desc").
		SetType("normal").
		SetStatus(change.StatusDraft).
		SetPriority("medium").
		SetImpactScope("low").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(1).
		Save(context.Background())
	require.NoError(t, err)

	repo := NewEntRepository(client, nil)
	got, err := repo.Get(context.Background(), created.ID, 1)
	require.NoError(t, err)

	require.Nil(t, got.AssigneeID, "未分配处理人必须是 nil，不是指向 0 的指针")
	require.Nil(t, got.PlannedStartDate)
	require.Nil(t, got.PlannedEndDate)
	require.Nil(t, got.ActualStartDate)
	require.Nil(t, got.ActualEndDate)

	payload, err := json.Marshal(toDTO(got))
	require.NoError(t, err)
	require.NotContains(t, string(payload), "0001-01-01", "零值时间不得下发: %s", payload)

	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &wire))
	for _, key := range []string{"assigneeId", "plannedStartDate", "plannedEndDate", "actualStartDate", "actualEndDate"} {
		require.JSONEq(t, "null", string(wire[key]), "%s 应为 null，实际 %s", key, wire[key])
	}
}

func TestToDomain_PreservesRealOptionalValues(t *testing.T) {
	client := openMapperTestClient(t)

	start := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(4 * time.Hour)

	created, err := client.Change.Create().
		SetTitle("已排期变更").
		SetDescription("desc").
		SetType("normal").
		SetStatus(change.StatusDraft).
		SetPriority("medium").
		SetImpactScope("low").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(1).
		SetAssigneeID(42).
		SetPlannedStartDate(start).
		SetPlannedEndDate(end).
		Save(context.Background())
	require.NoError(t, err)

	repo := NewEntRepository(client, nil)
	got, err := repo.Get(context.Background(), created.ID, 1)
	require.NoError(t, err)

	require.NotNil(t, got.AssigneeID)
	require.Equal(t, 42, *got.AssigneeID)
	require.NotNil(t, got.PlannedStartDate)
	require.True(t, start.Equal(*got.PlannedStartDate))
	require.NotNil(t, got.PlannedEndDate)
	require.True(t, end.Equal(*got.PlannedEndDate))

	payload, err := json.Marshal(toDTO(got))
	require.NoError(t, err)
	var wire struct {
		AssigneeID       *int   `json:"assigneeId"`
		PlannedStartDate string `json:"plannedStartDate"`
	}
	require.NoError(t, json.Unmarshal(payload, &wire))
	require.NotNil(t, wire.AssigneeID)
	require.Equal(t, 42, *wire.AssigneeID)
	require.Contains(t, wire.PlannedStartDate, "2026-11-02")
}
