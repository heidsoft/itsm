package msp

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"itsm-backend/common"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
)

func TestAllocationDelegation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		role     string
		status   int
	}{
		{"foreign", "b", "tenant_admin", 403}, {"own", "a", "tenant_admin", 200},
		{"unowned", "", "tenant_admin", 403}, {"platform_foreign", "b", "super_admin", 403},
		{"platform_unowned", "", "sysadmin", 403}, {"platform_valid", "a", "sysadmin", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:delegation_"+tc.name+"?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			a := seedHistoryTenant(t, client, "A", "a", "msp_provider")
			b := seedHistoryTenant(t, client, "B", "b", "msp_provider")
			customer := seedHistoryTenant(t, client, "C", "c", "customer")
			if tc.provider != "" {
				id := a.ID
				if tc.provider == "b" {
					id = b.ID
				}
				require.NoError(t, client.Tenant.UpdateOneID(customer.ID).SetMspProviderID(id).Exec(ctx))
			}
			actor := seedHistoryUser(t, client, "actor", a.ID)
			operatorID := actor.ID
			if tc.name == "platform_valid" {
				// 平台管理员不必与目标服务商同租户，但目标员工仍须匹配客户 provider。
				operatorID = seedHistoryUser(t, client, "platform", b.ID).ID
			}
			logger := zaptest.NewLogger(t).Sugar()
			h := NewHandler(NewMSPAllocationService(client, logger), nil, logger)
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("user_id", operatorID); c.Set("role", tc.role); c.Next() })
			r.POST("/api/v1/msp/allocations", h.CreateAllocation)
			status, body := doAllocationRequest(t, r, http.MethodPost, "/api/v1/msp/allocations", fmt.Sprintf(`{"mspUserId":%d,"customerTenantId":%d,"role":"primary"}`, actor.ID, customer.ID))
			require.Equal(t, tc.status, status, "%v", body)
			if tc.status == 403 {
				require.EqualValues(t, common.ForbiddenCode, body["code"])
				count, err := client.MSPAllocation.Query().Count(ctx)
				require.NoError(t, err)
				require.Zero(t, count)
			} else {
				require.EqualValues(t, 0, body["code"])
				data := body["data"].(map[string]interface{})
				require.EqualValues(t, customer.ID, data["customerTenantId"])
				require.NotContains(t, data, "passwordHash")
			}
			// A 无法解除 B 员工的分配；分配历史与列表也不能暴露 B 员工。
			agentB := seedHistoryUser(t, client, "agent_b", b.ID)
			seedHistoryAllocation(t, client, agentB.ID, customer.ID, time.Now(), nil)
			require.ErrorIs(t, h.mspAllocationService.Deactivate(ctx, actor.ID, agentB.ID, customer.ID, "tenant_admin"), ErrMSPAllocationForbidden)
			rows, _, err := h.mspAllocationService.ListHistory(ctx, MSPAllocationHistoryFilter{MSPTenantID: a.ID})
			require.NoError(t, err)
			for _, row := range rows {
				require.NotEqual(t, agentB.ID, row.MSPUserID)
			}
			own, err := h.mspAllocationService.ListByMSPUser(ctx, actor.ID)
			require.NoError(t, err)
			for _, row := range own {
				require.Equal(t, actor.ID, row.MSPUserID)
			}
		})
	}
}

func TestCustomerDirectoryUsesValidatedScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:directory_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	a := seedHistoryTenant(t, client, "A", "a", "msp_provider")
	b := seedHistoryTenant(t, client, "B", "b", "msp_provider")
	actor := seedHistoryUser(t, client, "actor", a.ID)
	require.NoError(t, client.User.UpdateOneID(actor.ID).SetMspRole("provider_admin").Exec(ctx))
	var allowedID int
	for _, name := range []string{"active", "suspended", "expired", "foreign"} {
		customer := seedHistoryTenant(t, client, name, name, "customer")
		providerID := a.ID
		if name == "foreign" {
			providerID = b.ID
		}
		update := client.Tenant.UpdateOneID(customer.ID).SetMspProviderID(providerID)
		if name == "suspended" {
			update.SetStatus("suspended")
		}
		if name == "expired" {
			update.SetExpiresAt(time.Now().Add(-time.Hour))
		}
		require.NoError(t, update.Exec(ctx))
		seedHistoryAllocation(t, client, actor.ID, customer.ID, time.Now(), nil)
		if name == "active" {
			allowedID = customer.ID
		}
	}
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(NewMSPAllocationService(client, logger), nil, logger)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", actor.ID); c.Next() }, middleware.MSPMiddleware(client))
	r.GET("/api/v1/msp/customers", h.GetAllCustomers)
	status, body := doAllocationRequest(t, r, "GET", "/api/v1/msp/customers", "")
	require.Equal(t, 200, status, "%v", body)
	require.EqualValues(t, 0, body["code"])
	data := body["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 1)
	require.EqualValues(t, allowedID, items[0].(map[string]interface{})["id"])
}
