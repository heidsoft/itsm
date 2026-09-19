package service

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/commandbus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 变更域流程路由契约测试（issue #92 P1） ====================
// 锁死 WorkflowStartCommandHandler change 分支的三级解析语义：
// payload 显式指定 > process_bindings 路由（按变更类型/风险等级）> 内置兜底；
// 路由指向租户未部署的流程定义时回退内置并继续，不得卡死变更生命周期。

type fakeWorkflowTrigger struct {
	reqs []*dto.ProcessTriggerRequest
}

func (f *fakeWorkflowTrigger) TriggerProcess(_ context.Context, req *dto.ProcessTriggerRequest) (*dto.ProcessTriggerResponse, error) {
	f.reqs = append(f.reqs, req)
	return &dto.ProcessTriggerResponse{ProcessInstanceID: 1, ProcessDefinitionKey: req.ProcessDefinitionKey}, nil
}

func (f *fakeWorkflowTrigger) TriggerByBusinessType(_ context.Context, _ dto.BusinessType, _ int, _ map[string]interface{}, _ string, _ int) (*dto.ProcessTriggerResponse, error) {
	return nil, nil
}

func (f *fakeWorkflowTrigger) CancelProcess(_ context.Context, _ int, _ string, _ int) error {
	return nil
}

func (f *fakeWorkflowTrigger) SuspendProcess(_ context.Context, _ int, _ string, _ int) error {
	return nil
}
func (f *fakeWorkflowTrigger) ResumeProcess(_ context.Context, _ int, _ int) error { return nil }
func (f *fakeWorkflowTrigger) GetProcessStatus(_ context.Context, _ int, _ int) (*dto.ProcessTriggerResponse, error) {
	return nil, nil
}

func setupChangeWorkflowTest(t *testing.T) (*ent.Client, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:chg_route_%d?mode=memory&cache=shared&_fk=1", nextRouteTestSeq()))
	ctx := context.Background()
	t.Cleanup(func() { _ = client.Close() })
	return client, ctx
}

var routeTestSeqCounter int

func nextRouteTestSeq() int64 {
	routeTestSeqCounter++
	return int64(routeTestSeqCounter)
}

func createRouteTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Route Tenant " + suffix).
		SetCode("route" + suffix).
		SetDomain("route-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
}

func createRouteTestChange(ctx context.Context, client *ent.Client, tenantID int, changeType, riskLevel string) (*ent.Change, error) {
	routeTestSeqCounter++
	return client.Change.Create().
		SetChangeNumber(fmt.Sprintf("CHG-ROUTE-%d", routeTestSeqCounter)).
		SetTitle("路由契约测试变更 " + changeType).
		SetType(changeType).
		SetRiskLevel(riskLevel).
		SetCreatedBy(1).
		SetTenantID(tenantID).
		Save(ctx)
}

func createRouteTestCommand(ctx context.Context, client *ent.Client, tenantID, changeID int, payload map[string]interface{}) (*ent.OperationalCommand, error) {
	routeTestSeqCounter++
	return client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("change").
		SetAggregateID(changeID).
		SetIdempotencyKey(fmt.Sprintf("change:%d:workflow:start:test", changeID)).
		SetPayload(payload).
		SetStatus(commandbus.StatusPending).
		Save(ctx)
}

// createDeployedDefinition 部署一个自定义流程定义（路由命中存在性校验用）。
func createDeployedDefinition(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, key string) {
	t.Helper()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("dep-%s-%d", key, tenantID)).
		SetDeploymentName("deploy " + key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey(key).
		SetName("自定义流程 " + key).
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
}

func TestWorkflowStart_ChangeRouting(t *testing.T) {
	tests := []struct {
		name         string
		changeType   string
		riskLevel    string
		setupBinding bool
		bindingType  string
		bindingKey   string
		deployTarget bool
		payloadKey   string
		wantKey      string
	}{
		{
			name:       "无路由绑定_普通变更_回退内置normal",
			changeType: "normal",
			wantKey:    "change_normal_flow",
		},
		{
			name:       "无路由绑定_标准变更_回退内置normal",
			changeType: "standard",
			wantKey:    "change_normal_flow",
		},
		{
			name:       "无路由绑定_紧急变更_回退内置emergency",
			changeType: "emergency",
			wantKey:    "change_emergency_flow",
		},
		{
			name:         "路由命中standard_部署存在_取自定义key",
			changeType:   "standard",
			setupBinding: true,
			bindingType:  "standard",
			bindingKey:   "custom_change_flow",
			deployTarget: true,
			wantKey:      "custom_change_flow",
		},
		{
			name:         "路由命中但定义未部署_回退内置并告警",
			changeType:   "normal",
			setupBinding: true,
			bindingType:  "normal",
			bindingKey:   "ghost_change_flow",
			deployTarget: false,
			wantKey:      "change_normal_flow",
		},
		{
			name:         "payload显式指定_优先于路由与内置",
			changeType:   "normal",
			payloadKey:   "payload_change_flow",
			deployTarget: true,
			wantKey:      "payload_change_flow",
		},
		{
			name:         "payload显式指定但定义未部署_回退内置",
			changeType:   "normal",
			payloadKey:   "ghost_payload_flow",
			deployTarget: false,
			wantKey:      "change_normal_flow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, ctx := setupChangeWorkflowTest(t)
			tenant, err := createRouteTestTenant(ctx, client, fmt.Sprint(routeTestSeqCounter))
			require.NoError(t, err)

			if tc.deployTarget {
				key := tc.bindingKey
				if key == "" {
					key = tc.payloadKey
				}
				createDeployedDefinition(t, ctx, client, tenant.ID, key)
			}
			if tc.setupBinding {
				_, err := client.ProcessBinding.Create().
					SetBusinessType("change").
					SetBusinessSubType(tc.bindingType).
					SetProcessDefinitionKey(tc.bindingKey).
					SetPriority(10).
					SetIsActive(true).
					SetTenantID(tenant.ID).
					Save(ctx)
				require.NoError(t, err)
			}

			ch, err := createRouteTestChange(ctx, client, tenant.ID, tc.changeType, "high")
			require.NoError(t, err)

			payload := map[string]interface{}{"businessType": "change", "businessId": ch.ID}
			if tc.payloadKey != "" {
				payload["workflowDefinitionKey"] = tc.payloadKey
			}
			cmd, err := createRouteTestCommand(ctx, client, tenant.ID, ch.ID, payload)
			require.NoError(t, err)

			trigger := &fakeWorkflowTrigger{}
			resolver := NewProcessResolver(client, nil)
			handler := NewWorkflowStartCommandHandler(client, trigger, resolver, zaptest.NewLogger(t).Sugar())

			require.NoError(t, handler.Handle(ctx, cmd))
			require.Len(t, trigger.reqs, 1)
			assert.Equal(t, tc.wantKey, trigger.reqs[0].ProcessDefinitionKey)
			assert.Equal(t, dto.BusinessTypeChange, trigger.reqs[0].BusinessType)
		})
	}
}

// TestWorkflowStart_ChangeRouting_TenantIsolation 锁死租户隔离：其他租户的路由绑定不得影响本租户。
func TestWorkflowStart_ChangeRouting_TenantIsolation(t *testing.T) {
	client, ctx := setupChangeWorkflowTest(t)

	tenantA, err := createRouteTestTenant(ctx, client, "a")
	require.NoError(t, err)
	tenantB, err := createRouteTestTenant(ctx, client, "b")
	require.NoError(t, err)

	// 租户 B 配置了 standard → 自定义流程；租户 A 不配置。
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("dep-tenantb-custom").
		SetDeploymentName("tenantB custom").
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("tenantb_custom_flow").
		SetName("租户B自定义流程").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessBinding.Create().
		SetBusinessType("change").
		SetBusinessSubType("standard").
		SetProcessDefinitionKey("tenantb_custom_flow").
		SetPriority(10).
		SetIsActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	// 租户 A 的 standard 变更必须回退内置，而不是命中租户 B 的绑定。
	chA, err := createRouteTestChange(ctx, client, tenantA.ID, "standard", "medium")
	require.NoError(t, err)
	cmdA, err := createRouteTestCommand(ctx, client, tenantA.ID, chA.ID, map[string]interface{}{})
	require.NoError(t, err)

	trigger := &fakeWorkflowTrigger{}
	resolver := NewProcessResolver(client, nil)
	handler := NewWorkflowStartCommandHandler(client, trigger, resolver, zaptest.NewLogger(t).Sugar())

	require.NoError(t, handler.Handle(ctx, cmdA))
	require.Len(t, trigger.reqs, 1)
	assert.Equal(t, "change_normal_flow", trigger.reqs[0].ProcessDefinitionKey)
}
