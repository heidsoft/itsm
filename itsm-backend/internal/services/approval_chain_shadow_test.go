package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestShadowComparator_BothPathsMatch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))

	tn, err := client.Tenant.Create().
		SetName("ShadowTN").SetCode("SHDW").SetDomain("shdw.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	mgr, err := client.User.Create().
		SetUsername("shadow-mgr").SetEmail("shadow-mgr@example.com").SetName("Mgr").
		SetPasswordHash("h").SetRole("manager").SetActive(true).SetTenantID(tn.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewApprovalChainService(client, logger)
	_, err = svc.CreateApprovalChain(ctx, &dto.ApprovalChainRequest{
		Name:       "ShadowChain",
		EntityType: "ticket",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "L1", IsRequired: true, ApprovalType: "serial"},
		},
	}, tn.ID)
	require.NoError(t, err)

	evalCtx := ApprovalEvalContext{RequesterID: mgr.ID}
	directPlan, directErr := svc.ResolveApprovalPlan(ctx, tn.ID, "ticket", evalCtx, nil)
	require.NoError(t, directErr)

	shadowResult := svc.ShadowCompare(ctx, tn.ID, "ticket", evalCtx, nil, directPlan, directErr)
	require.NotNil(t, shadowResult)
	assert.NoError(t, shadowResult.Err)
	assert.Equal(t, directPlan.ChainID, shadowResult.ChainID)
	assert.Equal(t, directPlan.Passed, shadowResult.Passed)
	assert.Equal(t, directPlan.Blocked, shadowResult.Blocked)
}

func TestShadowComparator_BothPathsError(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))

	tn, err := client.Tenant.Create().
		SetName("ShadowErrTN").SetCode("SHDE").SetDomain("shde.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	svc := NewApprovalChainService(client, logger)
	evalCtx := ApprovalEvalContext{RequesterID: 999}

	directPlan, directErr := svc.ResolveApprovalPlan(ctx, tn.ID, "ticket", evalCtx, nil)
	require.Error(t, directErr)

	shadowResult := svc.ShadowCompare(ctx, tn.ID, "ticket", evalCtx, nil, directPlan, directErr)
	require.NotNil(t, shadowResult)
	assert.Error(t, shadowResult.Err, "shadow path should also error when no active chain exists")
}
