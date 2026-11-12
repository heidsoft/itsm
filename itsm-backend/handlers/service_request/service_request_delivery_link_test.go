package service_request

import (
	"context"
	"testing"
	"time"

	"itsm-backend/internal/services"
	"itsm-backend/service/sla"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newDeliveryLinkService(t *testing.T) (*Service, *sla.Store, int, int) {
	t.Helper()
	repo, client, tenantID := srOutboxSetup(t)
	ctx := context.Background()
	requester, err := client.User.Query().First(ctx)
	require.NoError(t, err)

	slaSvc := service.NewTicketSLAService(client, zaptest.NewLogger(t).Sugar())
	store := sla.NewStore(client, sla.NewEngine(nil))
	slaSvc.SetSLAStore(store)

	svc := newTestService(repo, nil, nil, client, zaptest.NewLogger(t).Sugar(), nil)
	svc.SetSLAService(slaSvc)
	return svc, store, tenantID, requester.ID
}

func createProvisioningServiceRequest(t *testing.T, svc *Service, tenantID, requesterID int, status string) *ServiceRequest {
	t.Helper()
	ctx := context.Background()
	created, err := svc.client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(1).
		SetRequesterID(requesterID).
		SetTitle("交付联动测试").
		SetStatus(status).
		Save(ctx)
	require.NoError(t, err)
	return &ServiceRequest{ID: created.ID, TenantID: created.TenantID, RequesterID: created.RequesterID, Status: created.Status}
}

func TestProvisioningLifecycleRecordsSLAFirstResponseAndResolution(t *testing.T) {
	svc, store, tenantID, requesterID := newDeliveryLinkService(t)
	ctx := context.Background()
	req := createProvisioningServiceRequest(t, svc, tenantID, requesterID, SRStatusProvisioning)

	err := store.SaveDeadlines(ctx, tenantID, "service_request", req.ID, 0, nil, nil)
	require.NoError(t, err)

	svc.OnProvisioningStarted(ctx, tenantID, req.ID, requesterID)
	state, err := store.GetState(ctx, tenantID, "service_request", req.ID)
	require.NoError(t, err)
	require.False(t, state.FirstResponseAt.IsZero(), "开始交付即视为首次响应")
	firstResponse := state.FirstResponseAt

	// 重复通知不得把响应时间越改越晚。
	svc.OnProvisioningStarted(ctx, tenantID, req.ID, requesterID)
	state, err = store.GetState(ctx, tenantID, "service_request", req.ID)
	require.NoError(t, err)
	require.True(t, firstResponse.Equal(state.FirstResponseAt), "首次响应只记一次")

	svc.OnProvisioningFinished(ctx, tenantID, req.ID, requesterID, true, "")
	state, err = store.GetState(ctx, tenantID, "service_request", req.ID)
	require.NoError(t, err)
	require.False(t, state.ResolvedAt.IsZero(), "交付成功必须记录解决时间")
	require.Equal(t, "met", state.Status)
}

func TestProvisioningFailureKeepsSLAUnresolved(t *testing.T) {
	svc, store, tenantID, requesterID := newDeliveryLinkService(t)
	ctx := context.Background()
	req := createProvisioningServiceRequest(t, svc, tenantID, requesterID, SRStatusProvisioning)

	err := store.SaveDeadlines(ctx, tenantID, "service_request", req.ID, 0, nil, nil)
	require.NoError(t, err)

	svc.OnProvisioningFinished(ctx, tenantID, req.ID, requesterID, false, "云 API 失败")
	state, err := store.GetState(ctx, tenantID, "service_request", req.ID)
	require.NoError(t, err)
	require.True(t, state.ResolvedAt.IsZero(), "交付失败不是解决，SLA 必须保持未结")
	require.Equal(t, "active", state.Status)
}

func TestUpdateStatusDeliveredResolvesSLA(t *testing.T) {
	svc, store, tenantID, requesterID := newDeliveryLinkService(t)
	ctx := context.Background()
	req := createProvisioningServiceRequest(t, svc, tenantID, requesterID, SRStatusProvisioning)

	err := store.SaveDeadlines(ctx, tenantID, "service_request", req.ID, 0, nil, nil)
	require.NoError(t, err)

	require.NoError(t, svc.UpdateStatus(ctx, req.ID, tenantID, requesterID, RoleAgent, SRStatusDelivered))

	state, err := store.GetState(ctx, tenantID, "service_request", req.ID)
	require.NoError(t, err)
	require.False(t, state.ResolvedAt.IsZero())
	require.WithinDuration(t, time.Now(), state.ResolvedAt, time.Minute)
}
