package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/middleware"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCMDBAuditOnlineAndTypeGate(t *testing.T) {
	f := newCIHistoryFixture(t)
	s := NewConfigurationItemService(f.client, zaptest.NewLogger(t).Sugar(), f.history, nil)
	ci := f.createCI(t, "draft", "audit-gate")
	ci = ci.Update().SetLifecycleStatus(common.CILifecycleStatusDraft).SaveX(f.ctx)
	_, err := createTestAttrDefinition(f.ctx, f.client, f.tenantID, f.ciTypeID, "hostname", "主机名", "string", true, false, "", "")
	require.NoError(t, err)
	for _, attrs := range []map[string]interface{}{nil, {"hostname": nil}, {"hostname": "  "}} {
		ci.Update().SetAttributes(attrs).SaveX(f.ctx)
		_, err = s.UpdateCI(f.ctx, ci.ID, f.tenantID, &dto.UpdateCIRequest{LifecycleStatus: "online"})
		require.Error(t, err)
		require.Equal(t, "draft", f.client.ConfigurationItem.GetX(f.ctx, ci.ID).LifecycleStatus)
	}
	_, err = s.UpdateCI(f.ctx, ci.ID, f.tenantID, &dto.UpdateCIRequest{Attributes: map[string]interface{}{"hostname": "web01"}})
	require.NoError(t, err)
	_, err = s.UpdateCI(f.ctx, ci.ID, f.tenantID, &dto.UpdateCIRequest{LifecycleStatus: "online"})
	require.NoError(t, err)
	other, err := createTestCIType(f.ctx, f.client, f.tenantID, "Required serial")
	require.NoError(t, err)
	_, err = createTestAttrDefinition(f.ctx, f.client, f.tenantID, other.ID, "serial", "序列号", "string", true, false, "", "")
	require.NoError(t, err)
	_, err = s.UpdateCI(f.ctx, ci.ID, f.tenantID, &dto.UpdateCIRequest{CITypeID: other.ID})
	require.Error(t, err)
	require.Equal(t, f.ciTypeID, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).CiTypeID)
	// Dedicated lifecycle entry must retain the same gate and the version-zero contract.
	ci.Update().SetLifecycleStatus("draft").SetAttributes(map[string]interface{}{}).SaveX(f.ctx)
	_, err = s.UpdateLifecycleStatus(f.ctx, ci.ID, f.tenantID, LifecycleChangeParams{Status: "online"}, 0, "system")
	require.Error(t, err)
	ci.Update().SetAttributes(map[string]interface{}{"hostname": "web01"}).SaveX(f.ctx)
	result, err := s.BatchUpdateLifecycleStatus(f.ctx, []int{ci.ID}, f.tenantID, LifecycleChangeParams{Status: "online"}, 0, "system")
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
}

func TestCMDBAuditBatchCAS(t *testing.T) {
	f := newCIHistoryFixture(t)
	s := NewConfigurationItemService(f.client, zaptest.NewLogger(t).Sugar(), f.history, nil)
	ci := f.createCI(t, "before", "audit-cas")
	injected := false
	f.client.ConfigurationItem.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !injected && m.Op().Is(ent.OpUpdateOne) {
				injected = true
				mutation := m.(*ent.ConfigurationItemMutation)
				_, err := mutation.Client().ConfigurationItem.Update().Where(configurationitem.IDEQ(ci.ID)).AddVersion(1).Save(ctx)
				if err != nil {
					return nil, err
				}
			}
			return next.Mutate(ctx, m)
		})
	})
	result, err := s.BatchUpdateCI(f.ctx, &dto.BatchUpdateCIRequest{IDs: []int{ci.ID}, Updates: &dto.UpdateCIRequest{Name: "after"}}, f.tenantID)
	require.NoError(t, err)
	require.True(t, injected)
	require.Equal(t, []int{ci.ID}, result.FailedIDs)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], "版本")
	require.Zero(t, result.SuccessCount)
	require.Equal(t, "before", f.client.ConfigurationItem.GetX(f.ctx, ci.ID).Name)
}

func TestCMDBAuditHistoryFailureRollsBack(t *testing.T) {
	for _, operation := range []string{"create", "update", "batch-create", "batch-update", "revert"} {
		t.Run(operation, func(t *testing.T) {
			f := newCIHistoryFixture(t)
			s := NewConfigurationItemService(f.client, zaptest.NewLogger(t).Sugar(), f.history, nil)
			ci := f.createCI(t, "before", "audit-history")
			require.NoError(t, f.history.RecordCIHistory(f.ctx, ci.ID, f.tenantID, 0, "system", "create", "", nil, ci))
			f.client.ConfigurationItemHistory.Use(func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(context.Context, ent.Mutation) (ent.Value, error) {
					return nil, errors.New("injected history failure")
				})
			})
			count := f.client.ConfigurationItem.Query().CountX(f.ctx)
			req := dto.CreateCIRequest{Name: "new", CITypeID: f.ciTypeID, Status: "active"}
			var err error
			switch operation {
			case "create":
				_, err = s.CreateCI(f.ctx, &req, f.tenantID)
			case "update":
				_, err = s.UpdateCI(f.ctx, ci.ID, f.tenantID, &dto.UpdateCIRequest{Name: "after"})
			case "batch-create":
				var result *dto.BatchOperationResponse
				result, err = s.BatchCreateCI(f.ctx, &dto.BatchCreateCIRequest{Items: []dto.CreateCIRequest{req}}, f.tenantID)
				require.NoError(t, err)
				require.Equal(t, 1, result.FailedCount)
				require.Zero(t, result.SuccessCount)
			case "batch-update":
				var result *dto.BatchOperationResponse
				result, err = s.BatchUpdateCI(f.ctx, &dto.BatchUpdateCIRequest{IDs: []int{ci.ID}, Updates: &dto.UpdateCIRequest{Name: "after"}}, f.tenantID)
				require.NoError(t, err)
				require.Equal(t, []int{ci.ID}, result.FailedIDs)
				require.Zero(t, result.SuccessCount)
			case "revert":
				_, err = f.history.RevertCIVersion(f.ctx, ci.ID, f.tenantID, 0, "system", &dto.RevertCIVersionRequest{Version: 1})
			}
			if operation == "create" || operation == "update" || operation == "revert" {
				require.ErrorContains(t, err, "injected history failure")
			}
			require.Equal(t, count, f.client.ConfigurationItem.Query().CountX(f.ctx))
			current := f.client.ConfigurationItem.GetX(f.ctx, ci.ID)
			require.Equal(t, ci.Name, current.Name)
			require.Equal(t, ci.Version, current.Version)
			require.Equal(t, 1, f.client.ConfigurationItemHistory.Query().CountX(f.ctx))
		})
	}
}

func TestCMDBAuditRevertInvalidAttributes(t *testing.T) {
	f := newCIHistoryFixture(t)
	ci := f.createCI(t, "old", "audit-revert")
	ci = ci.Update().SetAttributes(map[string]interface{}{"hostname": " "}).SaveX(f.ctx)
	require.NoError(t, f.history.RecordCIHistory(f.ctx, ci.ID, f.tenantID, 0, "system", "create", "", nil, ci))
	_, err := createTestAttrDefinition(f.ctx, f.client, f.tenantID, f.ciTypeID, "hostname", "主机名", "string", true, false, "", "")
	require.NoError(t, err)
	current := ci.Update().SetName("current").SetAttributes(map[string]interface{}{"hostname": "web01"}).SaveX(f.ctx)
	_, err = f.history.RevertCIVersion(f.ctx, ci.ID, f.tenantID, 0, "system", &dto.RevertCIVersionRequest{Version: 1})
	require.ErrorContains(t, err, "回滚属性校验失败")
	require.Equal(t, current.Name, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).Name)
	require.Equal(t, current.Attributes, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).Attributes)
}

func TestCMDBAuditBatchAttributeValidation(t *testing.T) {
	f := newCIHistoryFixture(t)
	s := NewConfigurationItemService(f.client, zaptest.NewLogger(t).Sugar(), f.history, nil)
	ci := f.createCI(t, "before", "audit-batch-attrs")
	_, err := createTestAttrDefinition(f.ctx, f.client, f.tenantID, f.ciTypeID, "hostname", "主机名", "string", true, false, "", "")
	require.NoError(t, err)
	result, err := s.BatchCreateCI(f.ctx, &dto.BatchCreateCIRequest{Items: []dto.CreateCIRequest{{Name: "missing", CITypeID: f.ciTypeID, Status: "active"}}}, f.tenantID)
	require.NoError(t, err)
	require.Equal(t, []int{1}, result.FailedIDs)
	result, err = s.BatchUpdateCI(f.ctx, &dto.BatchUpdateCIRequest{IDs: []int{ci.ID}, Updates: &dto.UpdateCIRequest{Attributes: map[string]interface{}{"hostname": " "}}}, f.tenantID)
	require.NoError(t, err)
	require.Equal(t, []int{ci.ID}, result.FailedIDs)
	result, err = s.BatchUpdateCI(f.ctx, &dto.BatchUpdateCIRequest{IDs: []int{ci.ID}, Updates: &dto.UpdateCIRequest{Attributes: map[string]interface{}{"hostname": "web01"}}}, f.tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	result, err = s.BatchCreateCI(f.ctx, &dto.BatchCreateCIRequest{Items: []dto.CreateCIRequest{{Name: "complete", CITypeID: f.ciTypeID, Status: "active", Attributes: map[string]interface{}{"hostname": "web02"}}}}, f.tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 2, f.client.ConfigurationItemHistory.Query().CountX(f.ctx))
	other, err := createTestCIType(f.ctx, f.client, f.tenantID, "Required serial")
	require.NoError(t, err)
	_, err = createTestAttrDefinition(f.ctx, f.client, f.tenantID, other.ID, "serial", "序列号", "string", true, false, "", "")
	require.NoError(t, err)
	result, err = s.BatchUpdateCI(f.ctx, &dto.BatchUpdateCIRequest{IDs: []int{ci.ID}, Updates: &dto.UpdateCIRequest{CITypeID: other.ID}}, f.tenantID)
	require.NoError(t, err)
	require.Equal(t, []int{ci.ID}, result.FailedIDs)
	require.Equal(t, f.ciTypeID, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).CiTypeID)
}

func TestCMDBAuditRevertCAS(t *testing.T) {
	f := newCIHistoryFixture(t)
	ci := f.createCI(t, "before", "audit-revert-cas")
	require.NoError(t, f.history.RecordCIHistory(f.ctx, ci.ID, f.tenantID, 0, "system", "create", "", nil, ci))
	injected := false
	f.client.ConfigurationItem.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !injected && m.Op().Is(ent.OpUpdateOne) {
				injected = true
				mutation := m.(*ent.ConfigurationItemMutation)
				_, err := mutation.Client().ConfigurationItem.Update().Where(configurationitem.IDEQ(ci.ID)).AddVersion(1).Save(ctx)
				if err != nil {
					return nil, err
				}
			}
			return next.Mutate(ctx, m)
		})
	})
	_, err := f.history.RevertCIVersion(f.ctx, ci.ID, f.tenantID, 0, "system", &dto.RevertCIVersionRequest{Version: 1})
	require.True(t, injected)
	require.ErrorContains(t, err, "版本")
	require.Equal(t, ci.Version, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).Version)
	require.Equal(t, 1, f.client.ConfigurationItemHistory.Query().CountX(f.ctx))
}

// Exercise the production HTTP facade with real domain services and persistence.
func TestCMDBAuditUpdateHandlerGate(t *testing.T) {
	f := newCIHistoryFixture(t)
	logger := zaptest.NewLogger(t).Sugar()
	s := NewConfigurationItemService(f.client, logger, f.history, nil)
	ci := f.createCI(t, "draft", "audit-handler")
	ci.Update().SetLifecycleStatus("draft").SaveX(f.ctx)
	_, err := createTestAttrDefinition(f.ctx, f.client, f.tenantID, f.ciTypeID, "hostname", "主机名", "string", true, false, "", "")
	require.NoError(t, err)
	production := cmdb.NewProductionService(logger, nil, nil, s, nil, f.history, nil, nil, nil)
	handler := cmdb.NewHandler(cmdb.NewService(nil, production, logger))
	r := gin.New()
	scope := f.tenantID
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: scope})
		c.Set("user_id", 1)
		c.Next()
	})
	r.PUT("/api/v1/cmdb/cis/:id", handler.UpdateCI)
	send := func(body string) (int, map[string]interface{}) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/cmdb/cis/%d", ci.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var envelope map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
		return w.Code, envelope
	}
	status, envelope := send(`{"lifecycleStatus":"online"}`)
	require.Equal(t, http.StatusBadRequest, status)
	require.EqualValues(t, common.BadRequestCode, envelope["code"])
	require.Equal(t, "draft", f.client.ConfigurationItem.GetX(f.ctx, ci.ID).LifecycleStatus)
	status, envelope = send(`{"lifecycleStatus":"online","attributes":{"hostname":"web01"}}`)
	require.Equal(t, http.StatusOK, status)
	require.EqualValues(t, 0, envelope["code"])
	data := envelope["data"].(map[string]interface{})
	require.Equal(t, "online", data["lifecycleStatus"])
	require.EqualValues(t, f.ciTypeID, data["ciTypeId"])
	require.NotContains(t, data, "tenant_id")
	require.NotContains(t, data, "password")
	tenantB, err := createCMDBTestTenant(f.ctx, f.client, "Tenant B", "audit-b", "audit-b.example.com")
	require.NoError(t, err)
	scope = tenantB.ID
	status, envelope = send(`{"name":"cross-tenant"}`)
	require.Equal(t, http.StatusNotFound, status)
	require.EqualValues(t, common.NotFoundCode, envelope["code"])
	require.Equal(t, ci.Name, f.client.ConfigurationItem.GetX(f.ctx, ci.ID).Name)
}
