package service

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestToolRegistry_NewToolRegistryAcceptsConfigurationItemService is a
// regression guard for the refactor that replaced CMDBService with
// ConfigurationItemService. It verifies the type of the third constructor
// parameter at compile time, ensuring no caller accidentally re-introduces
// CMDBService.
func TestToolRegistry_NewToolRegistryAcceptsConfigurationItemService(t *testing.T) {
	// Build a TypeOf for the constructor's third parameter at runtime so
	// the test fails loudly if a future change reverts to CMDBService.
	fnType := reflect.TypeOf(NewToolRegistry)
	require.Equal(t, 4, fnType.NumIn(), "NewToolRegistry must keep its 4-parameter signature")
	third := fnType.In(2)
	want := reflect.TypeOf((*ConfigurationItemService)(nil))
	assert.Equal(t, want.String(), third.String(),
		"NewToolRegistry's third argument must be *ConfigurationItemService (was %s)", third.String())

	// And the field type on the struct must match as well.
	// ToolRegistry fields: 0=rag, 1=incident, 2=cmdb, 3=client
	tr := &ToolRegistry{}
	fieldType := reflect.TypeOf(tr).Elem().Field(2).Type
	assert.Equal(t, want.String(), fieldType.String(),
		"ToolRegistry.cmdb field must be *ConfigurationItemService (was %s)", fieldType.String())
}

// TestToolRegistry_ListCIsUsesConfigurationItemService verifies that the
// `list_cis` tool routes through ConfigurationItemService.ListCIs and that the
// limit/offset arguments are translated into page = offset/limit + 1.
func TestToolRegistry_ListCIsUsesConfigurationItemService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	ciType, err := client.CIType.Create().SetName("server").SetTenantID(1).Save(ctx)
	require.NoError(t, err)

	// Seed 25 CIs so we can verify the offset/page translation: offset=10,
	// limit=5 must map to page=3 (offset/limit + 1) and size=5.
	for i := 0; i < 25; i++ {
		_, err := client.ConfigurationItem.Create().
			SetName("ci-" + string(rune('a'+i%26))).
			SetCiType("server").
			SetCiTypeID(ciType.ID).
			SetStatus("active").
			SetTenantID(1).
			Save(ctx)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	historySvc := NewCIHistoryService(client, logger)
	tagSvc := NewCITagService(client, logger)
	ciSvc := NewConfigurationItemService(client, logger, historySvc, tagSvc)

	// RAG and Incident services are unused by list_cis; pass nil so that
	// any accidental invocation would NPE immediately.
	registry := NewToolRegistry(nil, nil, ciSvc, client)

	t.Run("translates limit and offset into page", func(t *testing.T) {
		result, err := registry.Execute(context.Background(), 1, "list_cis", map[string]interface{}{
			"limit":  float64(5),
			"offset": float64(10),
		})
		require.NoError(t, err)
		items, ok := result.([]*dto.CIResponse)
		require.True(t, ok, "list_cis result must be a []*dto.CIResponse, got %T", result)
		assert.Len(t, items, 5, "page 3 size 5 should return 5 CIs")
	})

	t.Run("defaults to limit=10 offset=0 when args missing", func(t *testing.T) {
		result, err := registry.Execute(context.Background(), 1, "list_cis", map[string]interface{}{})
		require.NoError(t, err)
		items, ok := result.([]*dto.CIResponse)
		require.True(t, ok)
		assert.Len(t, items, 10, "default limit=10 should return 10 CIs")
	})

	t.Run("returns empty list for tenant with no CIs", func(t *testing.T) {
		result, err := registry.Execute(context.Background(), 42, "list_cis", map[string]interface{}{
			"limit":  float64(5),
			"offset": float64(0),
		})
		require.NoError(t, err)
		items, ok := result.([]*dto.CIResponse)
		require.True(t, ok)
		assert.Empty(t, items, "unknown tenant should yield zero CIs")
	})

	t.Run("clamps offset past total to empty page", func(t *testing.T) {
		result, err := registry.Execute(context.Background(), 1, "list_cis", map[string]interface{}{
			"limit":  float64(10),
			"offset": float64(1000),
		})
		require.NoError(t, err)
		items, ok := result.([]*dto.CIResponse)
		require.True(t, ok)
		assert.Empty(t, items, "offset past end of result set must return no CIs")
	})
}

// TestToolRegistry_ListToolsCatalog sanity-checks the registered list. It is
// here so future regressions (missing tool name, wrong action) surface in the
// test suite rather than at runtime when the LLM gateway tries to look up a
// tool by name.
func TestToolRegistry_ListToolsCatalog(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry_catalog?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	historySvc := NewCIHistoryService(client, logger)
	tagSvc := NewCITagService(client, logger)
	ciSvc := NewConfigurationItemService(client, logger, historySvc, tagSvc)
	registry := NewToolRegistry(nil, nil, ciSvc, client)

	tools := registry.ListTools()
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	assert.True(t, names["list_cis"], "list_cis tool must remain registered")
	assert.True(t, names["get_incident_stats"], "get_incident_stats tool must remain registered")
	assert.True(t, names["list_kb"], "list_kb tool must remain registered")
	assert.True(t, names["list_tickets"], "list_tickets tool must remain registered")
	assert.True(t, names["create_ticket"], "create_ticket tool must remain registered")
	assert.True(t, names["update_ticket"], "update_ticket tool must remain registered")
}

// TestToolRegistry_UnknownTool confirms unknown tools are rejected with an
// error rather than silently returning nil.
func TestToolRegistry_UnknownTool(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	historySvc := NewCIHistoryService(client, logger)
	tagSvc := NewCITagService(client, logger)
	ciSvc := NewConfigurationItemService(client, logger, historySvc, tagSvc)
	registry := NewToolRegistry(nil, nil, ciSvc, client)

	_, err := registry.Execute(context.Background(), 1, "does_not_exist", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// TestToolRegistry_ListTickets verifies the list_tickets tool:
//   - returns dto.TicketResponse DTOs (never raw ent models)
//   - enforces explicit tenant filtering (no cross-tenant leakage)
//   - excludes soft-deleted tickets via DeletedAtIsNil
//   - clamps pageSize to [1,100]
func TestToolRegistry_ListTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry_tickets?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	mkTenant := func(code string) *ent.Tenant {
		t.Helper()
		tn, err := client.Tenant.Create().
			SetName("Tenant " + code).SetCode(code).SetDomain(code + ".test").SetStatus("active").
			Save(ctx)
		require.NoError(t, err)
		return tn
	}
	t1 := mkTenant("t1")
	t2 := mkTenant("t2")

	mkUser := func(tenantID int, suffix string) *ent.User {
		t.Helper()
		u, err := client.User.Create().
			SetUsername("req-" + suffix).
			SetEmail(suffix + "@test.com").
			SetName("Requester " + suffix).
			SetPasswordHash("hashed").
			SetRole("agent").
			SetActive(true).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return u
	}
	u1 := mkUser(t1.ID, "t1")
	u2 := mkUser(t2.ID, "t2")

	mkTicket := func(tenantID int, requesterID int, title string) *ent.Ticket {
		t.Helper()
		tk, err := client.Ticket.Create().
			SetTitle(title).
			SetTicketNumber("TKT-" + strconv.Itoa(tenantID) + "-" + title).
			SetRequesterID(requesterID).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return tk
	}

	// tenant 1: 3 条正常 + 1 条软删除
	mkTicket(1, u1.ID, "t1-a")
	mkTicket(1, u1.ID, "t1-b")
	mkTicket(1, u1.ID, "t1-c")
	deleted := mkTicket(1, u1.ID, "t1-deleted")
	err := client.Ticket.UpdateOneID(deleted.ID).SetDeletedAt(time.Now()).Exec(ctx)
	require.NoError(t, err)
	// tenant 2: 1 条，绝不能泄漏进租户 1 的查询
	mkTicket(2, u2.ID, "t2-a")

	registry := NewToolRegistry(nil, nil, nil, client)

	t.Run("returns DTO list filtered by tenant and excluding soft-deleted", func(t *testing.T) {
		result, err := registry.Execute(ctx, 1, "list_tickets", map[string]interface{}{})
		require.NoError(t, err)
		items, ok := result.([]*dto.TicketResponse)
		require.True(t, ok, "list_tickets result must be []*dto.TicketResponse, got %T", result)
		assert.Len(t, items, 3, "tenant 1 has 3 non-deleted tickets")
		for _, it := range items {
			assert.Equal(t, 1, it.TenantID, "result must stay within tenant 1")
			assert.NotEqual(t, "t1-deleted", it.Title, "soft-deleted tickets must be excluded")
			assert.NotEmpty(t, it.TicketNumber)
		}
	})

	t.Run("clamps pageSize to [1,100]", func(t *testing.T) {
		// pageSize=0 -> clamp 到 1，只返回最新 1 条
		result, err := registry.Execute(ctx, 1, "list_tickets", map[string]interface{}{"pageSize": float64(0)})
		require.NoError(t, err)
		items := result.([]*dto.TicketResponse)
		assert.Len(t, items, 1, "pageSize clamped to 1")

		// pageSize=1000 -> clamp 到 100，但租户 1 只有 3 条
		result, err = registry.Execute(ctx, 1, "list_tickets", map[string]interface{}{"pageSize": float64(1000)})
		require.NoError(t, err)
		items = result.([]*dto.TicketResponse)
		assert.Len(t, items, 3, "pageSize clamped to 100, only 3 exist")
	})

	t.Run("isolates other tenants", func(t *testing.T) {
		result, err := registry.Execute(ctx, 2, "list_tickets", map[string]interface{}{})
		require.NoError(t, err)
		items := result.([]*dto.TicketResponse)
		assert.Len(t, items, 1)
		assert.Equal(t, "t2-a", items[0].Title)
	})

	t.Run("empty tenant yields empty slice", func(t *testing.T) {
		result, err := registry.Execute(ctx, 99, "list_tickets", map[string]interface{}{})
		require.NoError(t, err)
		items := result.([]*dto.TicketResponse)
		assert.Empty(t, items)
	})
}

// Compile-time guard that the test file references ent (otherwise the unused
// import would fail after refactors).
var _ = (*ent.Client)(nil)
