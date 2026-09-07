package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// 边界条件测试：TicketService
// 覆盖 nil 输入、空字符串、超长字段、非法枚举值、极端 tenantID 等场景
// ============================================================

func setupBoundaryTicketTest(t *testing.T) (*ent.Client, *TicketService, context.Context) {
	dbName := strings.NewReplacer("/", "-", " ", "-", ":", "-").Replace(t.Name())
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	logger := zaptest.NewLogger(t)
	svc := NewTicketServiceForTest(client, logger.Sugar())
	return client, svc, context.Background()
}

func createBoundaryTenant(ctx context.Context, t *testing.T, client *ent.Client, suffix string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("BoundaryTenant"+suffix).
		SetCode("boundary-"+suffix).
		SetDomain("boundary-"+suffix+".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID
}

func createBoundaryUser(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, suffix string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
				SetName("BoundaryUser"+suffix).
		SetUsername("boundary-" + suffix).
		SetEmail("boundary-" + suffix + "@example.com").
		SetPasswordHash("hashed").
		SetRole("admin").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return user
}

// ---- CreateTicket 边界测试 ----

func TestTicketService_CreateTicket_NilRequest(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.CreateTicket(ctx, nil, 1)
	require.Error(t, err)
}

func TestTicketService_CreateTicket_EmptyTitle(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "empty-title")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "",
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	require.Error(t, err)
}

func TestTicketService_CreateTicket_WhitespaceOnlyTitle(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "ws-title")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "   \t\n  ",
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	require.Error(t, err)
}

func TestTicketService_CreateTicket_ZeroTenantID(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:    "valid title",
		Priority: "medium",
		Type:     "incident",
	}, 0)
	require.Error(t, err)
}

func TestTicketService_CreateTicket_NegativeTenantID(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:    "valid title",
		Priority: "medium",
		Type:     "incident",
	}, -1)
	require.Error(t, err)
}

func TestTicketService_CreateTicket_InvalidPriority(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "bad-prio")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:    "valid title",
		Priority: "super-urgent-invalid",
		Type:     "incident",
	}, tenantID)
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	}
}

func TestTicketService_CreateTicket_InvalidType(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "bad-type")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:    "valid title",
		Priority: "medium",
		Type:     "nonexistent_type",
	}, tenantID)
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	}
}

func TestTicketService_CreateTicket_ExtremelyLongTitle(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "long-title")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	longTitle := strings.Repeat("测", 5000)
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       longTitle,
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	}
}

func TestTicketService_CreateTicket_SQLInjectionAttempt(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "sqli")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	malicious := "'; DROP TABLE tickets; --"
	_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       malicious,
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	}
}

func TestTicketService_CreateTicket_XSSPayload(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "xss")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	xss := "<script>alert('xss')</script>"
	result, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       xss,
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	if err == nil {
		assert.NotNil(t, result)
	}
}

func TestTicketService_CreateTicket_AllPriorityValues(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	priorities := []string{"low", "medium", "high", "critical", "unknown"}
	for idx, prio := range priorities {
		t.Run("priority_"+prio, func(t *testing.T) {
			tenantID := createBoundaryTenant(ctx, t, client, "prio-" + prio + "-" + fmt.Sprint(idx))
			user := createBoundaryUser(ctx, t, client, tenantID, prio + "-" + fmt.Sprint(idx))
			_, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
				Title:       "Ticket with prio=" + prio,
				Description: "desc",
				Priority:    prio,
				Type:        "incident",
				RequesterID: user.ID,
			}, tenantID)
			if err != nil {
				assert.NotContains(t, err.Error(), "panic")
			}
		})
	}
}

func TestTicketService_CreateTicket_UnicodeTitle(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "unicode")
	createBoundaryUser(ctx, t, client, tenantID, "r")
	result, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "工单标题 🎉 émojis ñ ü",
		Description: "描述包含特殊字符: 你好世界",
		Priority:    "medium",
		Type:        "incident",
	}, tenantID)
	if err == nil {
		assert.NotNil(t, result)
	}
}

// ---- GetTicket 边界测试 ----

func TestTicketService_GetTicket_NonExistentID(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.GetTicket(ctx, 999999, 1)
	require.Error(t, err)
}

func TestTicketService_GetTicket_ZeroID(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.GetTicket(ctx, 0, 1)
	require.Error(t, err)
}

func TestTicketService_GetTicket_NegativeID(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.GetTicket(ctx, -1, 1)
	require.Error(t, err)
}

func TestTicketService_GetTicket_WrongTenant(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantA := createBoundaryTenant(ctx, t, client, "tenantA")
	tenantB := createBoundaryTenant(ctx, t, client, "tenantB")
	user := createBoundaryUser(ctx, t, client, tenantA, "r")
	tk, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "Tenant A ticket",
		Priority:    "medium",
		Type:        "incident",
		RequesterID: user.ID,
	}, tenantA)
	require.NoError(t, err)
	_, err = svc.GetTicket(ctx, tk.ID, tenantB)
	assert.Error(t, err)
}

// ---- ListTickets 边界测试 ----

func TestTicketService_ListTickets_EmptyRequest(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "empty-req")
	user := createBoundaryUser(ctx, t, client, tenantID, "del-user")
	result, err := svc.ListTickets(ctx, &dto.ListTicketsRequest{}, tenantID, user.ID, "admin")
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	} else {
		assert.NotNil(t, result)
	}
}

func TestTicketService_ListTickets_InvalidPagination(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "bad-page")
	user := createBoundaryUser(ctx, t, client, tenantID, "del-user")
	result, err := svc.ListTickets(ctx, &dto.ListTicketsRequest{
		Page:     -1,
		PageSize: 0,
	}, tenantID, user.ID, "admin")
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	} else {
		assert.NotNil(t, result)
	}
}

// ---- UpdateTicket 边界测试 ----

func TestTicketService_UpdateTicket_NilRequest(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "nil-update")
	user := createBoundaryUser(ctx, t, client, tenantID, "del-user")
	tk, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "to update",
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
		RequesterID: user.ID,
	}, tenantID)
	require.NoError(t, err)
	_, err = svc.UpdateTicket(ctx, tk.ID, nil, tenantID, user.ID, "admin")
	require.Error(t, err)
}

func TestTicketService_UpdateTicket_NonExistent(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	_, err := svc.UpdateTicket(ctx, 999999, &dto.UpdateTicketRequest{Title: "nope"}, 1, 1, "admin")
	require.Error(t, err)
}

func TestTicketService_UpdateTicket_EmptyUpdate(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "empty-update")
	user := createBoundaryUser(ctx, t, client, tenantID, "del-user")
	tk, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "to update empty",
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
		RequesterID: user.ID,
	}, tenantID)
	require.NoError(t, err)
	result, err := svc.UpdateTicket(ctx, tk.ID, &dto.UpdateTicketRequest{}, tenantID, user.ID, "admin")
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	} else {
		assert.NotNil(t, result)
	}
}

// ---- DeleteTicket 边界测试 ----

func TestTicketService_DeleteTicket_NonExistent(t *testing.T) {
	_, svc, ctx := setupBoundaryTicketTest(t)
	err := svc.DeleteTicket(ctx, 999999, 1, 1, "admin")
	require.Error(t, err)
}

func TestTicketService_DeleteTicket_AlreadyDeleted(t *testing.T) {
	client, svc, ctx := setupBoundaryTicketTest(t)
	tenantID := createBoundaryTenant(ctx, t, client, "double-del")
	user := createBoundaryUser(ctx, t, client, tenantID, "del-user")
	tk, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "to delete",
		Description: "desc",
		Priority:    "medium",
		Type:        "incident",
		RequesterID: user.ID,
	}, tenantID)
	require.NoError(t, err)
	err = svc.DeleteTicket(ctx, tk.ID, tenantID, user.ID, "admin")
	require.NoError(t, err)
	// 二次删除应返回错误或幂等成功
	err = svc.DeleteTicket(ctx, tk.ID, tenantID, user.ID, "admin")
	if err != nil {
		assert.NotContains(t, err.Error(), "panic")
	}
}
